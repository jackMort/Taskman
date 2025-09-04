package form

import (
	"speedtestui/app"
	"speedtestui/components/config"
	"speedtestui/components/overlay"
	"speedtestui/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	general = lipgloss.NewStyle().
		UnsetAlign().
		Padding(0, 1, 0, 1).
		Foreground(config.COLOR_FOREGROUND).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(config.COLOR_HIGHLIGHT)
)

const (
	TITLE_IDX = iota
	CLOSE_IDX
	SAVE_IDX
)

type TaskForm struct {
	inputs   []textinput.Model
	focused  int
	save     bool
	errors   []string
	bgRaw    string
	width    int
	startRow int
	startCol int
}

func NewTaskForm(bgRaw string, width int, vWidth int) TaskForm {
	var inputs []textinput.Model = make([]textinput.Model, 1)

	inputs[TITLE_IDX] = textinput.New()
	inputs[TITLE_IDX].Placeholder = "Title"
	inputs[TITLE_IDX].Prompt = "󱞩 "
	// inputs[COLLECTION_IDX].ShowSuggestions = true
	// inputs[COLLECTION_IDX].KeyMap.AcceptSuggestion = key.NewBinding(
	// 	key.WithKeys("enter"),
	// )

	// if app.GetInstance().SelectedCollection != nil {
	// 	inputs[COLLECTION_IDX].SetValue(app.GetInstance().SelectedCollection.Name)
	// }

	// collections := app.GetInstance().Collections
	// suggestions := make([]string, len(collections))
	// for i, c := range collections {
	// 	suggestions[i] = c.Name
	// }
	// inputs[COLLECTION_IDX].SetSuggestions(suggestions)

	return TaskForm{
		bgRaw:    bgRaw,
		startRow: 3,
		width:    width,
		startCol: vWidth - width - 4,
		inputs:   inputs,
		focused:  TITLE_IDX,
	}
}
func (c TaskForm) Title() string {
	return c.inputs[TITLE_IDX].Value()
}

func (c TaskForm) SetTitle(title string) {
	c.inputs[TITLE_IDX].SetValue(title)
}

// Init initializes the popup.
func (c TaskForm) Init() tea.Cmd {
	return textinput.Blink
}

// nextInput focuses the next input field
func (c *TaskForm) nextInput() {
	c.focused = (c.focused + 1) % (len(c.inputs) + 2)
}

// prevInput focuses the previous input field
func (c *TaskForm) prevInput() {
	c.focused--
	// Wrap around
	if c.focused < 0 {
		c.focused = len(c.inputs) + 1
	}
}

// Update handles messages.
func (c TaskForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(c.inputs))

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if c.focused == CLOSE_IDX {
				return c, c.makeChoice()
			} else if c.focused == SAVE_IDX {
				c.save = true
				return c, c.makeChoice()
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			return c, c.makeChoice()
		case tea.KeyShiftTab, tea.KeyCtrlK:
			c.prevInput()
		case tea.KeyTab, tea.KeyCtrlJ:
			c.nextInput()
		}
		for i := range c.inputs {
			c.inputs[i].Blur()
		}
		if c.focused < len(c.inputs) {
			c.inputs[c.focused].Focus()
		}
	}

	for i := range c.inputs {
		c.inputs[i], cmds[i] = c.inputs[i].Update(msg)
	}

	c.Validate()

	return c, tea.Batch(cmds...)
}

func (c *TaskForm) Validate() {
	c.errors = make([]string, 0)
	if c.inputs[TITLE_IDX].Value() == "" {
		c.errors = append(c.errors, "Title is required")
	}
}

// View renders the popup.
func (c TaskForm) View() string {
	okButtonStyle := config.ButtonStyle
	cancelButtonStyle := config.ButtonStyle
	if c.focused == SAVE_IDX {
		okButtonStyle = config.ActiveButtonStyle
	} else if c.focused == CLOSE_IDX {
		cancelButtonStyle = config.ActiveButtonStyle
	}

	okButton := zone.Mark("add_to_collection_save", okButtonStyle.Render("Save"))
	cancelButton := zone.Mark("add_to_collection_cancel", cancelButtonStyle.Render("Cancel"))

	buttons := lipgloss.PlaceHorizontal(
		c.width,
		lipgloss.Right,
		lipgloss.JoinHorizontal(lipgloss.Right, cancelButton, " ", okButton),
	)

	header := config.BoxHeader.Width(30).Render("Create Task")

	inputs := lipgloss.JoinVertical(
		lipgloss.Left,
		config.LabelStyle.Width(30).Render("Task Title:"),
		config.InputStyle.Render(c.inputs[TITLE_IDX].View()),
		" ",
		utils.RenderErrors(c.errors),
		buttons,
	)

	ui := lipgloss.JoinVertical(lipgloss.Left, header, " ", inputs)

	content := general.Render(ui)
	return overlay.PlaceCenter(content, c.bgRaw)
}

func (c TaskForm) makeChoice() tea.Cmd {
	return func() tea.Msg {
		return app.TaskFormResultMsg{
			Result: c.save,
			Title:  c.Title(),
		}
	}
}
