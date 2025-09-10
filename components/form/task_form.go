package form

import (
	"taskman/app"
	"taskman/components/config"
	"taskman/components/overlay"
	"taskman/utils"

	"github.com/charmbracelet/bubbles/textarea"
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
	TEXTAREA_IDX
	CLOSE_IDX
	SAVE_IDX
)

type TaskForm struct {
	titleInput textinput.Model
	notesInput textarea.Model
	focused    int
	save       bool
	errors     []string
	bgRaw      string
	width      int
	startRow   int
	startCol   int
}

func NewTaskForm(bgRaw string, width int, vWidth int) TaskForm {

	titleInput := textinput.New()
	titleInput.Placeholder = "Title"
	titleInput.Prompt = "󱞩 "

	textArea := textarea.New()
	textArea.ShowLineNumbers = false

	return TaskForm{
		bgRaw:      bgRaw,
		startRow:   3,
		width:      width,
		startCol:   vWidth - width - 4,
		titleInput: titleInput,
		notesInput: textArea,
		focused:    TITLE_IDX,
	}
}

func (c TaskForm) Title() string {
	return c.titleInput.Value()
}

func (c TaskForm) Notes() string {
	return c.notesInput.Value()
}

// Init initializes the popup.
func (c TaskForm) Init() tea.Cmd {
	return textinput.Blink
}

// nextInput focuses the next input field
func (c *TaskForm) nextInput() {
	c.focused = (c.focused + 1) % 4
}

// prevInput focuses the previous input field
func (c *TaskForm) prevInput() {
	c.focused--
	// Wrap around
	if c.focused < 0 {
		c.focused = 4
	}
}

// Update handles messages.
func (c TaskForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, 2)

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

		c.titleInput.Blur()
		c.notesInput.Blur()
		if c.focused == TITLE_IDX {
			c.titleInput.Focus()
			c.titleInput, cmds[0] = c.titleInput.Update(msg)
		} else if c.focused == TEXTAREA_IDX {
			c.notesInput.Focus()
			c.notesInput, cmds[0] = c.notesInput.Update(msg)
		}
	}

	c.Validate()

	return c, tea.Batch(cmds...)
}

func (c *TaskForm) Validate() {
	c.errors = make([]string, 0)
	if c.Title() == "" {
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
		config.LabelStyle.Width(30).Render("Title:"),
		config.InputStyle.Render(c.titleInput.View()),
		" ",
		config.LabelStyle.Width(30).Render("Notes:"),
		config.InputStyle.Render(c.notesInput.View()),
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
			Notes:  c.Notes(),
		}
	}
}
