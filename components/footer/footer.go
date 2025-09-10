package footer

import (
	"taskman/components/config"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	container    = lipgloss.NewStyle()
	versionStyle = lipgloss.NewStyle().Foreground(config.COLOR_HIGHLIGHT)
	nameStyle    = lipgloss.NewStyle().Foreground(config.COLOR_HIGHLIGHT).Underline(true)
)

// model represents the properties of the UI.
type model struct {
	height int
	width  int
	help   help.Model
}

// New creates a new instance of the UI.
func New() model {
	return model{
		help: help.New(),
	}
}

// Init intializes the UI.
func (m model) Init() tea.Cmd { return nil }

// Update handles all UI interactions.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	return m, nil
}

// View returns a string representation of the UI.
func (m model) View() string {

	helpView := m.help.View(config.Keys)

	statusWidth := lipgloss.Width(helpView) + 1

	return container.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			" "+helpView,
			lipgloss.PlaceHorizontal(
				m.width-statusWidth-1,
				lipgloss.Right,
				nameStyle.Render("TASKMAN")+

					versionStyle.Render(" v."+config.GetVersion()),
			),
		),
	)
}
