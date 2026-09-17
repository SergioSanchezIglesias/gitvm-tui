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
type creationResult struct {
	profile Profile
	err     error
}

type Model struct {
	creating         bool
	fields           [4]string
	field            int
	create           func(Profile) error
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

// WithCreator enables profile creation using the supplied storage boundary.
func (m Model) WithCreator(create func(Profile) error) Model {
	m.create = create
	return m
}
func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case creationResult:
		m.busy = false
		if msg.err != nil {
			m.status = "Creation failed: " + msg.err.Error()
		} else {
			m.profiles = append(m.profiles, msg.profile)
			m.cursor = len(m.profiles) - 1
			m.creating = false
			m.fields = [4]string{}
			m.status = "Created " + msg.profile.ID + ". Select it to activate."
		}
		return m, nil
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
		if m.creating {
			return m.updateForm(msg)
		}
		switch msg.String() {
		case "n":
			if !m.confirming && m.create != nil {
				m.creating = true
				m.fields = [4]string{}
				m.field = 0
				m.status = ""
			}
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
func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.creating = false
		m.fields = [4]string{}
		m.status = "Creation cancelled."
	case tea.KeyTab, tea.KeyDown:
		m.field = (m.field + 1) % len(m.fields)
	case tea.KeyShiftTab, tea.KeyUp:
		m.field = (m.field + len(m.fields) - 1) % len(m.fields)
	case tea.KeyBackspace:
		runes := []rune(m.fields[m.field])
		if len(runes) > 0 {
			m.fields[m.field] = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		m.fields[m.field] += " "
	case tea.KeyRunes:
		m.fields[m.field] += string(msg.Runes)
	case tea.KeyEnter:
		if m.field < len(m.fields)-1 {
			m.field++
			break
		}
		p := Profile{m.fields[0], m.fields[1], m.fields[2], m.fields[3]}
		if err := p.Validate(); err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.busy = true
		m.status = "Creating..."
		return m, func() tea.Msg { return creationResult{p, m.create(p)} }
	}
	return m, nil
}

func (m Model) View() string {
	var out strings.Builder
	if m.creating {
		out.WriteString("GitVM — Create profile\n\n")
		for i, label := range []string{"Profile ID", "Author name", "Email", "SSH alias (optional)"} {
			pointer := "  "
			if i == m.field {
				pointer = "> "
			}
			fmt.Fprintf(&out, "%s%s: %s\n", pointer, label, m.fields[i])
		}
		out.WriteString("\n" + m.status + "\n\nTab/Shift+Tab or ↑/↓: fields • Enter: next/save on alias • Backspace: erase • Esc/Ctrl+C: cancel\n")
		return out.String()
	}
	out.WriteString("GitVM — Select a profile\n\n")
	if len(m.profiles) == 0 {
		out.WriteString("No valid profiles found. Press n to create one.\n")
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
	out.WriteString("\n↑/↓ or j/k: navigate • n: new profile • Enter: select • Esc/q: quit\n")
	return out.String()
}
