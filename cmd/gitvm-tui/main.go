package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"gitvm/internal/gitvm"
)

const usage = `Usage:
  gitvm-tui
  gitvm-tui --user <profile-id>
  gitvm-tui -u <profile-id>
  gitvm-tui --list | -l
  gitvm-tui --help | -h

Options:
  --user, -u <profile-id>  Activate an existing profile by its exact stored ID.
  --list, -l              List profile IDs, names and emails; * marks the saved active ID.
  --help, -h              Show this help and exit without changes.

List and help must be used alone and never change state.
An empty list prints "No profiles found." and exits successfully.

Without arguments, opens the interactive TUI. Profile activation requires no
terminal or confirmation. It changes global Git name/email defaults and the
current-profile marker; SSH is updated only when existing safety rules permit.
Repository-local Git settings and GitHub CLI authentication are not changed.
Activation is not a transaction: a later failure may leave earlier changes applied.
Exit status: 0 on success or help; 1 on invalid input or runtime failure.

Example: gitvm-tui --user personal
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.UserHomeDir, gitvm.RunGit, func(model gitvm.Model) error {
		_, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
		return err
	}))
}

// run keeps process, Git, and terminal boundaries injectable without changing
// the profile validation or activation shared with the TUI.
func run(args []string, stdout, stderr io.Writer, userHome func() (string, error), runGit func(...string) error, openTUI func(gitvm.Model) error) int {
	argumentError := func(message string) int {
		fmt.Fprintln(stderr, "Error: "+message)
		fmt.Fprintln(stderr, "Run gitvm-tui --help for usage.")
		return 1
	}
	var id string
	list := false
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h":
			if len(args) != 1 {
				return argumentError(args[0] + " must be used alone")
			}
			fmt.Fprint(stdout, usage)
			return 0
		case "--list", "-l":
			if len(args) != 1 {
				return argumentError(args[0] + " must be used alone")
			}
			list = true
		case "--user", "-u":
			if len(args) < 2 {
				return argumentError(args[0] + " requires a profile ID")
			}
			if len(args) > 2 {
				return argumentError(fmt.Sprintf("unexpected argument %q", args[2]))
			}
			id = args[1]
			// Reuse the stored profile's validation rather than allowing path-like IDs.
			if err := (gitvm.Profile{ID: id, Name: "Validation", Email: "validation@example.com"}).Validate(); err != nil {
				return argumentError(fmt.Sprintf("invalid profile ID %q", id))
			}
		default:
			return argumentError(fmt.Sprintf("unknown argument %q", args[0]))
		}
	}
	home, err := userHome()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	profiles, current, err := gitvm.Load(home)
	if err != nil {
		fmt.Fprintln(stderr, "Load profiles:", err)
		return 1
	}
	if list {
		if len(profiles) == 0 {
			fmt.Fprintln(stdout, "No profiles found.")
		}
		for _, profile := range profiles {
			marker := " "
			if profile.ID == current {
				marker = "*"
			}
			fmt.Fprintf(stdout, "%s %s\t%s\t%s\n", marker, profile.ID, profile.Name, profile.Email)
		}
		return 0
	}
	if len(args) > 0 {
		for _, profile := range profiles {
			if profile.ID != id {
				continue
			}
			message, err := gitvm.Activate(home, profile, runGit)
			if err != nil {
				fmt.Fprintln(stderr, "Activation failed:", err)
				return 1
			}
			fmt.Fprintln(stdout, message)
			return 0
		}
		return argumentError(fmt.Sprintf("profile %q not found; create or check its ID in the TUI", id))
	}
	model := gitvm.NewModel(profiles, current, func(p gitvm.Profile) (string, error) { return gitvm.Activate(home, p, runGit) }).WithCreator(func(p gitvm.Profile) error { return gitvm.Create(home, p) }).WithDeleter(func(id string) error { return gitvm.Delete(home, id) })
	if err := openTUI(model); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
