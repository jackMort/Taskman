package calendar

import (
	"time"

	"taskman/app"
	"taskman/components/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ethanefung/bubble-datepicker"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var subHeader = lipgloss.NewStyle().Foreground(config.COLOR_LIGHTER).Bold(true).Padding(1, 0)

var taskStyle = lipgloss.NewStyle().Padding(0, 1)
var completedStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(config.COLOR_GRAY)
var taskContainerStyle = lipgloss.NewStyle().Padding(0, 2)

type model struct {
	day    time.Time
	dp     datepicker.Model
	width  int
	height int
	err    error
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case app.DaySelectedMsg:
		m.day = msg.Day
		m.dp.SetTime(m.day)

	}
	return m, nil
}

func (m model) View() string {
	return baseStyle.Width(m.width - 2).Height(m.height - 2).Render(
		m.dp.View(),
	)
}

func New() *model {
	dpView := datepicker.New(time.Now())
	dpView.SelectDate()

	m := &model{
		dp: dpView,
	}
	return m
}
