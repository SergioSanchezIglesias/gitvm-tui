package gitvm

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type activationResult struct {
	id, message string
	err         error
}
type Model struct {
	profiles         []Profile
	current          string
	cursor           int
	confirming, busy bool
	status           string
	activate         func(Profile) (string, error)
}

func NewModel(profiles []Profile, current string, activate func(Profile) (string, error)) Model {
	return Model{profiles: profiles, current: current, activate: activate}
}
func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case activationResult:
		m.busy = false
		m.confirming = false
		if msg.err != nil {
			m.status = "Activation failed: " + msg.err.Error()
		} else {
			m.current = msg.id
			m.status = msg.message
		}
		return m, nil
	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.confirming {
				m.confirming = false
				m.status = "Activation cancelled."
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if !m.confirming && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if !m.confirming && m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.profiles) == 0 {
				return m, nil
			}
			if !m.confirming {
				m.confirming = true
				m.status = ""
				return m, nil
			}
			p := m.profiles[m.cursor]
			m.busy = true
			m.status = "Activating..."
			return m, func() tea.Msg { s, err := m.activate(p); return activationResult{p.ID, s, err} }
		}
	}
	return m, nil
}
func (m Model) View() string {
	var out strings.Builder
	out.WriteString("GitVM — Select a profile\n\n")
	if len(m.profiles) == 0 {
		out.WriteString("No valid profiles found. Add one with gitvm add.\n")
	}
	for i, p := range m.profiles {
		pointer := "  "
		if i == m.cursor {
			pointer = "> "
		}
		active := ""
		if p.ID == m.current {
			active = " [active]"
		}
		fmt.Fprintf(&out, "%s%s%s\n    %s | %s | alias: %s\n", pointer, p.ID, active, p.Name, p.Email, p.Alias)
	}
	if m.confirming && !m.busy {
		fmt.Fprintf(&out, "\nConfirm activation of %s? Enter to confirm; Esc/q to cancel.\n", m.profiles[m.cursor].ID)
	}
	if m.status != "" {
		out.WriteString("\n" + m.status + "\n")
	}
	out.WriteString("\n↑/↓ or j/k: navigate • Enter: select • Esc/q: quit\n")
	return out.String()
}
