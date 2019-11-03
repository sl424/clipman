#[macro_use]
extern crate failure;

use dirs;
use failure::Error;
use serde::{Deserialize, Serialize};
use std::cmp::{Eq, PartialEq};
use std::collections::HashSet;
use std::fs::{remove_dir_all, File};
use std::io::BufReader;
use std::path::PathBuf;
use structopt::StructOpt;
use wl_clipboard_rs::copy::{
    self, ClipboardType, MimeSource, MimeType, Options, Seat, ServeRequests, Source,
};

mod actions;
mod selector;

#[derive(StructOpt)]
#[structopt(
    version = "2.0.0",
    author = "yory8",
    about = "A clipboard manager for Wayland"
)]
struct Opts {
    #[structopt(subcommand)]
    subcmd: SubCommand,
}

#[derive(StructOpt)]
enum SubCommand {
    /// Record clipboard events
    #[structopt(name = "watch")]
    Watch {
        /// Max number of items to store
        #[structopt(short = "m", long = "max-items", default_value = "15")]
        max: usize,
    },
    /// Pick an item from clipboard history
    #[structopt(name = "pick")]
    Pick {
        /// How many items to display
        #[structopt(short = "m", long = "max-items", default_value = "15")]
        max: usize,
        /// Which selector to use. Supported: wofi
        #[structopt(short = "t", long = "tool", default_value = "wofi")]
        tool: String,
        /// Extra arguments to pass to the --tool
        #[structopt(short = "T", long = "tool-args")]
        toolargs: Option<String>,
    },
    /// Remove item(s) from history
    #[structopt(name = "clear")]
    Clear {
        /// How many items to display
        #[structopt(short = "m", long = "max-items", default_value = "15")]
        max: usize,
        /// Which selector to use. Supported: wofi
        #[structopt(short = "t", long = "tool", default_value = "wofi")]
        tool: String,
        /// Extra arguments to pass to the --tool
        #[structopt(short = "T", long = "tool-args")]
        toolargs: Option<String>,
        /// Remove all items
        #[structopt(short = "a", long = "all")]
        all: bool,
    },
    /// Export history index to stdout
    #[structopt(name = "export")]
    Export {
        /// Format
        #[structopt(short = "f", long = "format", default_value = "json")]
        format: String,
    },
}

fn main() -> Result<(), Error> {
    let opts: Opts = Opts::from_args();

    // fixme: create if missing
    let history_dir = {
        let mut path = PathBuf::new();
        path.push(dirs::data_local_dir().ok_or(format_err!("no data dir"))?);
        path.push("clipman");
        path
    };
    let history_idx = {
        let mut path = history_dir.clone();
        path.push("index.cbor");
        path
    };
    let idx: Vec<HashSet<Multi>> = {
        let file = File::open(&history_idx)?;
        let reader = BufReader::new(file);
        serde_cbor::from_reader(reader).unwrap_or(Vec::new())
    };

    match opts.subcmd {
        SubCommand::Watch { max } => actions::watch(idx, &history_dir, &history_idx, max)?,
        SubCommand::Pick {
            max,
            tool,
            toolargs,
        } => actions::pick(idx, &history_idx, max, tool, toolargs)?,
        SubCommand::Clear {
            max,
            tool,
            toolargs,
            all,
        } => actions::clear(idx, &history_dir, &history_idx, max, tool, toolargs, all)?,
        SubCommand::Export { format } => actions::export(idx, format)?,
    }

    Ok(())
}

fn serve(idx: &HashSet<Multi>) -> Result<(), Error> {
    let mut sources = Vec::new();
    for m in idx {
        sources.push(MimeSource {
            source: Source::Bytes(&m.source),
            mime_type: MimeType::Specific(m.mime.clone()),
        });
    }

    let mut opts = Options::new();
    opts.serve_requests(ServeRequests::Unlimited);
    opts.copy_multi(sources)?;

    Ok(())
}

fn wipe(hist_dir: &PathBuf) -> Result<(), Error> {
    copy::clear(ClipboardType::Regular, Seat::All)?;
    remove_dir_all(hist_dir)?;
    Ok(())
}

#[derive(Clone, Deserialize, Debug, Eq, Hash, PartialEq, Serialize)]
pub struct Multi {
    source: Vec<u8>,
    mime: String,
}
