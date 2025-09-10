package main

import (
	"speedtestui/app"
	"speedtestui/components/config"
	"speedtestui/components/form"
	"speedtestui/components/popup"
	"speedtestui/utils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	boxer "github.com/treilik/bubbleboxer"
)

var windows = []string{"collections", "url", "request", "results"}

var (
	testStyle2 = lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.NormalBorder()).
			BorderForeground(config.COLOR_SUBTLE).
			PaddingLeft(1)

	testStyleFocused = lipgloss.NewStyle().
				Bold(true).
				Border(lipgloss.NormalBorder()).
				BorderForeground(config.COLOR_HIGHLIGHT).
				PaddingLeft(1)

	listHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(config.COLOR_SUBTLE).
			Render
)

func stripErr(n boxer.Node, _ error) boxer.Node {
	return n
}

func main() {
	rootCmd.Flags().StringP("url", "u", "", "Url")
	rootCmd.Flags().StringP("data", "d", "", "Data")
	rootCmd.Flags().StringP("data-raw", "", "", "Data Raw")
	rootCmd.Flags().StringP("request", "X", "GET", "HTTP method")
	rootCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP header")

	rootCmd.Execute()
}

type Model struct {
	tui    boxer.Boxer
	popup  tea.Model
	width  int
	height int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) GetFadedView() string {
	return lipgloss.NewStyle().Foreground(config.COLOR_SUBTLE).Render(utils.RemoveANSI(m.View()))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Handle popup messages
	case popup.ChoiceResultMsg:
		// If the user chose "yes", quit the program.
		if msg.Result {
			return m, tea.Quit
		}
		m.popup = nil

	case app.TaskFormResultMsg:
		m.popup = nil

	case tea.KeyMsg:
		{
			switch msg.String() {
			case "ctrl+c":
				if m.SizeIsTooSmall() {
					return m, tea.Quit
				}

				width := 100
				m.popup = popup.NewChoice(m.GetFadedView(), width, "Are you sure, you want to quit?", false)
				return m, m.popup.Init()

			case "a":
				if m.popup == nil {
					f := form.NewTaskForm(m.GetFadedView(), m.width-4, m.tui.LayoutTree.GetWidth())
					m.popup = f

					return m, m.popup.Init()
				}

			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tui.UpdateSize(msg)

		return m, nil
	}

	// If there is a popup, we only update that.
	if m.popup != nil {
		m.popup, cmd = m.popup.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		for key, element := range m.tui.ModelMap {
			m.tui.ModelMap[key], cmd = element.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) SizeIsTooSmall() bool {
	return m.width < 40 || m.height < 30
}

func (m Model) View() string {
	if m.SizeIsTooSmall() {
		return config.FullscreenStyle.
			Width(m.width - 2).
			Height(m.height - 2).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					config.BoxHeader.Render("SpeedtesTUI "+version),
					"Please resize the window to at least 140x30"),
			)
	}

	if m.popup != nil {
		return m.popup.View()
	}
	return zone.Scan(m.tui.View())
}
