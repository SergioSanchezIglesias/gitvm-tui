package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"gitvm/internal/gitvm"
	"os"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	profiles, current, err := gitvm.Load(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Load profiles:", err)
		os.Exit(1)
	}
	model := gitvm.NewModel(profiles, current, func(p gitvm.Profile) (string, error) { return gitvm.Activate(home, p, gitvm.RunGit) }).WithCreator(func(p gitvm.Profile) error { return gitvm.Create(home, p) }).WithDeleter(func(id string) error { return gitvm.Delete(home, id) })
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
