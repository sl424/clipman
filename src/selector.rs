use crate::Multi;
use failure::Error;
use std::collections::HashMap;
use std::collections::HashSet;
use std::io::Write;
use std::process::{Command, Stdio};

pub fn select(
    data: &[HashSet<Multi>],
    max: usize,
    prompt: &str,
    tool: String,
    toolargs: Option<String>,
) -> Result<Option<HashSet<Multi>>, Error> {
    if data.is_empty() {
        return Err(format_err!("Empty data"));
    };

    if &tool != "wofi" {
        return Err(format_err!("Unsupported tool"));
    };

    let mut args = vec!["-p", prompt, "--cache-file", "/dev/null", "--dmenu"];
    match &toolargs {
        Some(a) => {
            for arg in a.split(' ') {
                args.push(arg);
            }
        }
        None => {}
    };

    let (prepro, guide) = process(&data)?;

    let output: String = {
        let mut child = Command::new("wofi")
            .args(&args)
            .stdin(Stdio::piped())
            .stderr(Stdio::piped())
            .stdout(Stdio::piped())
            .spawn()?;

        child
            .stdin
            .as_mut()
            .ok_or(format_err!("Child process stdin has not been captured!"))?
            .write_all(prepro.join("\n").as_bytes())?;

        let output = child.wait_with_output()?;

        if output.status.success() {
            String::from_utf8(output.stdout)?
        } else {
            let err = String::from_utf8(output.stderr)?;
            return Err(format_err!("External command failed:\n {}", err));
        }
    };

    if output.is_empty() {
        return Ok(None);
    };

    Ok(Some(guide[&output.trim().to_string()].clone()))
}

fn process(
    data: &[HashSet<Multi>],
) -> Result<(Vec<String>, HashMap<String, HashSet<Multi>>), Error> {
    let mut guide = HashMap::new();
    let mut escaped = Vec::new();
    for el in data.iter().rev() {
        // fixme: better way to find label
        let original =
            String::from_utf8_lossy(&el.iter().next().ok_or(format_err!("no item"))?.source)
                .to_string();
        let repr = original
            .replace("\\n", "\\\\n")
            .replace("\n", "\\n")
            .replace("\\t", "\\\\t")
            .replace("\t", "\\\\t");

        let repr = if repr.len() > 200 {
            repr[..200].to_string()
        } else {
            repr
        };

        guide.insert(repr.clone(), el.clone());
        escaped.push(repr);
    }

    Ok((escaped, guide))
}
