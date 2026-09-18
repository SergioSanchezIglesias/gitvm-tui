package gitvm

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	actionMenu screen = iota
	switchSelection
	deleteSelection
	activationConfirmation
	deleteConfirmation
	createForm
)

type activationResult struct {
	id, message string
	err         error
}
type creationResult struct {
	profile Profile
	err     error
}
type deletionResult struct {
	id  string
	err error
}

type Model struct {
	state                 screen
	fields                [4]string
	field, action, cursor int
	create                func(Profile) error
	delete                func(string) error
	profiles              []Profile
	current               string
	busy                  bool
	status                string
	failed                bool
	activate              func(Profile) (string, error)
}

func NewModel(profiles []Profile, current string, activate func(Profile) (string, error)) Model {
	return Model{profiles: profiles, current: current, activate: activate}
}

// WithCreator enables profile creation using the supplied storage boundary.
func (m Model) WithCreator(create func(Profile) error) Model { m.create = create; return m }

// WithDeleter enables deletion; the model guards the active profile.
func (m Model) WithDeleter(delete func(string) error) Model { m.delete = delete; return m }
func (m Model) Init() tea.Cmd                               { return nil }
func (m *Model) feedback(text string, failed bool)          { m.status = text; m.failed = failed }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case creationResult:
		m.busy = false
		if msg.err != nil {
			m.feedback("Creation failed: "+msg.err.Error(), true)
		} else {
			m.profiles = append(m.profiles, msg.profile)
			m.cursor = len(m.profiles) - 1
			m.state = actionMenu
			m.fields = [4]string{}
			m.feedback("Created "+msg.profile.ID+". Select it to activate.", false)
		}
	case activationResult:
		m.busy = false
		m.state = switchSelection
		if msg.err != nil {
			m.feedback("Activation failed: "+msg.err.Error(), true)
		} else {
			m.current = msg.id
			m.feedback(msg.message, false)
		}
	case deletionResult:
		m.busy = false
		m.state = deleteSelection
		if msg.err != nil {
			m.feedback("Deletion failed: "+msg.err.Error(), true)
		} else {
			remaining := make([]Profile, 0, len(m.profiles))
			for _, p := range m.profiles {
				if p.ID != msg.id {
					remaining = append(remaining, p)
				}
			}
			m.profiles = remaining
			if m.cursor >= len(m.profiles) {
				m.cursor = max(0, len(m.profiles)-1)
			}
			m.feedback("Deleted "+msg.id+".", false)
		}
	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		if m.state == createForm {
			return m.updateForm(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			switch m.state {
			case actionMenu:
				return m, tea.Quit
			case activationConfirmation:
				m.state = switchSelection
				m.feedback("Activation cancelled.", false)
			case deleteConfirmation:
				m.state = deleteSelection
				m.feedback("Deletion cancelled.", false)
			default:
				m.state = actionMenu
			}
		case "n":
			if m.state == actionMenu {
				m.beginCreate()
			}
		case "up", "k":
			if m.state == actionMenu {
				m.action = max(0, m.action-1)
			} else if m.state == switchSelection || m.state == deleteSelection {
				m.cursor = max(0, m.cursor-1)
			}
		case "down", "j":
			if m.state == actionMenu {
				m.action = min(2, m.action+1)
			} else if (m.state == switchSelection || m.state == deleteSelection) && len(m.profiles) > 0 {
				m.cursor = min(len(m.profiles)-1, m.cursor+1)
			}
		case "enter":
			if m.state == actionMenu {
				switch m.action {
				case 0:
					m.state = switchSelection
				case 1:
					m.beginCreate()
				case 2:
					m.state = deleteSelection
					m.cursor = 0
				}
				return m, nil
			}
			if len(m.profiles) == 0 {
				return m, nil
			}
			p := m.profiles[m.cursor]
			switch m.state {
			case switchSelection:
				if m.activate == nil {
					m.feedback("Activation unavailable.", true)
				} else {
					m.state = activationConfirmation
				}
			case deleteSelection, deleteConfirmation:
				if p.ID == m.current {
					m.feedback("Cannot delete active profile. Switch to another profile first.", true)
					return m, nil
				}
				if m.delete == nil {
					m.feedback("Deletion unavailable.", true)
					return m, nil
				}
				if m.state == deleteSelection {
					m.state = deleteConfirmation
					return m, nil
				}
				m.busy = true
				m.feedback("Deleting...", false)
				return m, func() tea.Msg { return deletionResult{p.ID, m.delete(p.ID)} }
			case activationConfirmation:
				m.busy = true
				m.feedback("Activating...", false)
				return m, func() tea.Msg { s, err := m.activate(p); return activationResult{p.ID, s, err} }
			}
		}
	}
	return m, nil
}
func (m *Model) beginCreate() {
	if m.create == nil {
		m.feedback("Creation unavailable.", true)
		return
	}
	m.state = createForm
	m.fields = [4]string{}
	m.field = 0
	m.feedback("", false)
}
func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.state = actionMenu
		m.fields = [4]string{}
		m.feedback("Creation cancelled.", false)
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
			m.feedback(err.Error(), true)
			return m, nil
		}
		m.busy = true
		m.feedback("Creating...", false)
		return m, func() tea.Msg { return creationResult{p, m.create(p)} }
	}
	return m, nil
}

var (
	focusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	frameStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

func focused(text string, focus bool) string {
	if focus {
		return focusStyle.Render("> " + text)
	}
	return "  " + text
}
func (m Model) View() string {
	var out strings.Builder
	if m.state == actionMenu {
		out.WriteString("   GitVM\n<--o   o-->\n    \\ /\n     o\nGit profile manager\n\n")
	}
	titles := []string{"Profiles", "Switch profile", "Delete profile", "Switch profile", "Delete profile", "Create profile"}
	out.WriteString(focusStyle.Render("GitVM — "+titles[m.state]) + "\n\n")
	hints := "↑/↓ or j/k: navigate • Enter: select • Esc/q/Ctrl+C: back"
	switch m.state {
	case actionMenu:
		for i, label := range []string{"Switch profile", "Create profile", "Delete profile"} {
			line := focused(label, i == m.action)
			if i == 2 {
				line += dangerStyle.Render(" [destructive]")
			}
			out.WriteString(line + "\n")
		}
		if len(m.profiles) == 0 {
			out.WriteString("\nNo valid profiles found. Choose Create profile.\n")
		}
		hints = "↑/↓ or j/k: navigate • Enter: select • n: new profile • Esc/q/Ctrl+C: quit"
	case createForm:
		for i, label := range []string{"Profile ID", "Author name", "Email", "SSH alias (optional)"} {
			out.WriteString(focused(label+": "+m.fields[i], i == m.field) + "\n")
		}
		hints = "Tab/Shift+Tab or ↑/↓: fields • Enter: next/save on alias\nBackspace: erase • Esc/Ctrl+C: cancel"
	default:
		if len(m.profiles) == 0 {
			out.WriteString("No valid profiles found. Esc: back to actions.\n")
		}
		for i, p := range m.profiles {
			line := focused(p.ID, i == m.cursor)
			if p.ID == m.current {
				line += activeStyle.Render(" [active]")
			}
			fmt.Fprintf(&out, "%s\n    %s | %s | alias: %s\n", line, p.Name, p.Email, p.Alias)
		}
		if m.state == deleteSelection {
			out.WriteString("\n" + dangerStyle.Render("Delete profile [destructive] — active profiles are protected.") + "\n")
		}
		if (m.state == activationConfirmation || m.state == deleteConfirmation) && len(m.profiles) > 0 {
			prompt := "Confirm activation of " + m.profiles[m.cursor].ID + "?"
			if m.state == deleteConfirmation {
				prompt = dangerStyle.Render("Confirm deletion of " + m.profiles[m.cursor].ID + "? This cannot be undone.")
			}
			out.WriteString("\n" + prompt + "\n")
			hints = "Enter: confirm • Esc/q/Ctrl+C: cancel"
		}
	}
	if m.status != "" {
		style := activeStyle
		label := "Status: "
		if m.failed {
			style = dangerStyle
			label = "Error: "
		}
		out.WriteString("\n" + style.Render(label+m.status) + "\n")
	}
	if m.busy {
		hints = "Working… Please wait."
	}
	out.WriteString("\n" + hints)
	return frameStyle.Render(out.String()) + "\n"
}
