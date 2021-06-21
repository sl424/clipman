package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
)

func store(text string, history []string, histfile string, max int, persist bool) error {
	if text == "" {
		return nil
	}

	l := len(history)
	if l > 0 {
		// this avoids entering an endless loop,
		// see https://github.com/bugaevc/wl-clipboard/issues/65
		last := history[l-1]
		if text == last {
			return nil
		}

		// drop oldest items that exceed max list size
		// if max = 0, we allow infinite history; NOTE: users should NOT rely on this behaviour as we might change it without notice
		if max != 0 && l >= max {
			// usually just one item, but more if we suddenly reduce our --max-items
			history = history[l-max+1:]
		}

		// remove duplicates
		history = filter(history, text)
	}

	history = append(history, text)

	// dump history to file so that other apps can query it
	if err := write(history, histfile); err != nil {
		return fmt.Errorf("error writing history: %s", err)
	}

	// make the copy buffer available to all applications,
	// even when the source has disappeared
	// text/plain filetype only
	if persist {
		var exitError *exec.ExitError
		bin, err := exec.LookPath("wl-paste")
		if err != nil {
			smartLog(fmt.Sprintf("couldn't find wl-paste: %v\n", err), "low", *alert)
		}
		cmd := exec.Cmd{Path: bin, Args: []string{bin, "-t", "text/plain"}}
		if err := cmd.Run(); err != nil {
			if errors.As(err, &exitError) {
				return nil
			}
		}
		serveTxt(text)
	}

	return nil
}

// filter removes all occurrences of text
func filter(slice []string, text string) []string {
	var filtered []string
	for _, s := range slice {
		if s != text {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// write dumps history to json file
func write(history []string, histfile string) error {
	b, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(histfile, b, 0600)
}
