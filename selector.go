package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
	"image"
	_ "image/gif"
	_ "image/png"
	_ "image/jpeg"
	//_ "code.google.com/p/vp8-go/webp"
)

var pngStr = "\x89\x50\x4e\x47"
var jpgStr = "\xff\xd8\xff\xe0"
var svgStr = "<svg"

func selector(data []string, max int, tool, prompt, toolArgs string, null bool) (string, error) {
	dir, dirErr := ioutil.TempDir("", "clipman.*.dir")
	if dirErr != nil {
		return "", fmt.Errorf("selector: %w", dirErr)
	}
	fmt.Println("[!] create tmp dir: ", dir)
	defer os.RemoveAll(dir)

	if len(data) == 0 {
		return "", errors.New("nothing to show: no data available")
	}

	// output to stdout and return
	if tool == "STDOUT" {
		escaped, _ := preprocessData(data, 0, !null, dir)
		sep := "\n"
		if null {
			sep = "\000"
		}
		os.Stdout.WriteString(strings.Join(escaped, sep))
		return "", nil
	}

	var (
		args []string
		err  error
	)

	switch tool {
	case "dmenu":
		args = []string{"dmenu", "-b",
			"-fn",
			"-misc-dejavu sans mono-medium-r-normal--17-120-100-100-m-0-iso8859-16",
			"-l",
			strconv.Itoa(max)}
	case "bemenu":
		args = []string{"bemenu", "--prompt", prompt, "--list", strconv.Itoa(max)}
	case "rofi":
		args = []string{"rofi", "-p", prompt, "-dmenu",
			"-lines",
			strconv.Itoa(max)}
	case "wofi":
		args = []string{"wofi", "-p", prompt, "--cache-file", "/dev/null", "--dmenu"}
	case "CUSTOM":
		if len(toolArgs) == 0 {
			return "", fmt.Errorf("missing tool args for CUSTOM tool")
		}
		args, err = shellquote.Split(toolArgs)
		if err != nil {
			return "", fmt.Errorf("selector: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported tool: %s", tool)
	}

	if tool == "CUSTOM" {
		tool = args[0]
	} else if len(toolArgs) > 0 {
		targs, err := shellquote.Split(toolArgs)
		if err != nil {
			return "", fmt.Errorf("selector: %w", err)
		}
		args = append(args, targs...)
	}

	bin, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("%s is not installed", tool)
	}

	processed, guide := preprocessData(data, 1000, !null, dir)
	sep := "\n"
	if null {
		sep = "\000"
	}

	cmd := exec.Cmd{Path: bin, Args: args, Stdin: strings.NewReader(strings.Join(processed, sep))}
	cmd.Stderr = os.Stderr // let stderr pass to console
	b, err := cmd.Output()
	if err != nil {
		if err.Error() == "exit status 1" || err.Error() == "exit status 130" {
			// dmenu/rofi exits with 1 when no selection done
			// fzf exits with 1 when no match, 130 when no selection done
			return "", nil
		}
		return "", err
	}

	// we received no selection; wofi doesn't error in this case
	if len(b) == 0 || len(b) == 1 && b[0] == '\n' {
		return "", nil
	}

	// drop newline added by proper unix tools
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	sel, ok := guide[strings.Split(string(b), ":")[0]]
	if !ok {
		return "", errors.New("couldn't recover original string")
	}

	return sel, nil
}

// preprocessData:
// - reverses the data
// - optionally escapes \n, \r and \t (it would break some external selectors)
// - optionally it cuts items longer than maxChars bytes (dmenu doesn't allow more than ~1200)
// A guide is created to allow restoring the selected item.
func preprocessData(data []string, maxChars int, escape bool, tmpdir string) ([]string, map[string]string) {
	var escaped []string
	escaped = append(escaped, "") // append first line as empty
	guide := make(map[string]string)

	count := 0
	for i := len(data) - 1; i >= 0; i-- { // reverse slice
		countStr := strconv.Itoa(count)
		original := data[i]
		repr := countStr + ": " + original

		//decode data as image, write to temporary directory
		reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(original))
		_, format, err := image.DecodeConfig(reader)
		if err != nil {
			smartLog(err.Error(), "low", *alert)
		} 
		if format != "" {
			b, _ := base64.StdEncoding.DecodeString(data[i])
			original = string(b)
			tmpfile, err := ioutil.TempFile(tmpdir, "preview-")
			if err != nil {
				smartLog(err.Error(), "low", *alert)
			}
			if _, err := tmpfile.Write(b); err != nil {
				smartLog(err.Error(), "low", *alert)
			}
			if err := tmpfile.Close(); err != nil {
				smartLog(err.Error(), "low", *alert)
			}
			repr = countStr + ": :" + "img:" + tmpfile.Name()
		}
		fmt.Printf("length: %d\n", len(data[i]) )

		// escape newlines
		if escape {
			repr = strings.ReplaceAll(repr, "\\n", "\\\\n") // preserve literal \n
			repr = strings.ReplaceAll(repr, "\n", "\\n")
			repr = strings.ReplaceAll(repr, "\\t", "\\\\t")
			repr = strings.ReplaceAll(repr, "\t", "\\t")
			repr = strings.ReplaceAll(repr, "\\r", "\\\\r")
			repr = strings.ReplaceAll(repr, "\r", "\\r")
		}
		// optionally cut to maxChars
		if maxChars > 0 && len(repr) > maxChars {
			repr = repr[:maxChars]
		}

		guide[countStr] = original
		escaped = append(escaped, repr)
		count++
	}

	return escaped, guide
}
