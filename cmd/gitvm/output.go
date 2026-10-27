package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// Color is limited to terminal output; redirected output stays byte-compatible.
func colorOutput(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && useColor(isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()), os.Getenv("TERM"), os.Getenv("NO_COLOR"))
}

func useColor(terminal bool, term, noColor string) bool {
	return terminal && term != "" && term != "dumb" && noColor == ""
}

func paint(code, text string) string {
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func renderHelp(color bool) string {
	if !color {
		return usage
	}
	lines := strings.Split(usage, "\n")
	for i, line := range lines {
		switch {
		case line == "Usage:" || line == "Options:":
			lines[i] = paint("1;36", line)
		case strings.HasPrefix(line, "  gitvm"):
			lines[i] = "  " + paint("36", strings.TrimPrefix(line, "  "))
		case strings.HasPrefix(line, "  --"):
			// Keep descriptions plain and preserve the exact original spacing.
			option, description, _ := strings.Cut(strings.TrimPrefix(line, "  "), "  ")
			lines[i] = "  " + paint("33", option) + "  " + description
		case strings.HasPrefix(line, "Example:"):
			lines[i] = paint("36", line)
		}
	}
	return strings.Join(lines, "\n")
}

func renderProfile(marker, id, name, email string, color bool) string {
	label := marker + " " + id
	if color {
		if marker == "*" {
			label = paint("1;32", label)
		} else {
			label = paint("36", label)
		}
		name = paint("1", name)
		email = paint("33", email)
	}
	return fmt.Sprintf("%s\t%s\t%s\n", label, name, email)
}
