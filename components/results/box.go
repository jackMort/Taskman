package results

import (
	"fmt"

	"speedtestui/app"
	"speedtestui/components/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var subHeader = lipgloss.NewStyle().Foreground(config.COLOR_LIGHTER).Bold(true).Padding(1, 0)

var taskStyle = lipgloss.NewStyle().Padding(0, 1)
var completedStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(config.COLOR_GRAY)
var taskContainerStyle = lipgloss.NewStyle().Padding(0, 2)

type rowKind int

const (
	rowHeader rowKind = iota
	rowItem
)

type row struct {
	kind  rowKind
	label string
	id    int // only for rowItem
}

// ----- model -----

type model struct {
	store   *app.Store
	rows    []row
	cursor  int // index in rows (can land on headers; movement skips them)
	width   int
	height  int
	err     error
	loading bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case app.TaskFormResultMsg:
		if msg.Result {
			// add new task
			if strings.TrimSpace(msg.Title) != "" {
				if _, err := m.store.Add(msg.Title, "", nil); err != nil {
					m.err = err
				}
				m.rebuildRows()
				// move cursor to new item
				m.cursor = m.findRowByID(m.store.NextID - 1)
				if m.cursor == -1 {
					m.cursor = m.nextSelectable(-1, +1)
				}
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = m.nextSelectable(m.cursor, -1)
		case "down", "j":
			m.cursor = m.nextSelectable(m.cursor, +1)
		case " ", "enter":
			// toggle completion on selected row
			if m.rows[m.cursor].kind == rowItem {
				id := m.rows[m.cursor].id
				if _, err := m.store.ToggleCompleted(id); err != nil {
					m.err = err
				}
				m.rebuildRows()
				// after rebuild, attempt to keep cursor on same id
				m.cursor = m.findRowByID(id)
				if m.cursor == -1 {
					m.cursor = m.nextSelectable(-1, +1)
				}
			}
		case "d":
			if m.rows[m.cursor].kind == rowItem {
				id := m.rows[m.cursor].id
				if err := m.store.Delete(id); err != nil {
					m.err = err
				}
				// move cursor up first so it feels natural
				m.cursor = m.nextSelectable(m.cursor, -1)
				m.rebuildRows()
			}
		}
	}
	return m, nil
}

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Padding(1, 1).Foreground(config.COLOR_LIGHTER)
	itemStyle     = lipgloss.NewStyle()
	selectedStyle = lipgloss.NewStyle().Background(config.COLOR_HIGHLIGHT).Foreground(config.COLOR_FOREGROUND).Bold(true)
	doneStyle     = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	notesStyle    = lipgloss.NewStyle().Italic(true).Foreground(config.COLOR_LIGHTER)
	cursorGlyph   = "› " // looks nice; change to "> " if you prefer
	indent        = "  "
)

func (m model) View() string {
	var b strings.Builder
	for i, r := range m.rows {
		switch r.kind {
		case rowHeader:
			b.WriteString(headerStyle.Render(r.label) + "\n")
		case rowItem:
			line := r.label
			// style completed items
			// (Detect completion by presence in Completed section: cheap check)
			// If previous header was "Completed", it's done:
			done := m.isInCompletedSection(i)
			if done {
				line = doneStyle.Render(line)
			} else {
				line = itemStyle.Render(line)
			}
			prefix := indent
			if i == m.cursor {
				line = selectedStyle.Copy().Width(m.width - 2).Render(line)
				prefix = cursorGlyph
			}
			b.WriteString(prefix + line + "\n")
		}
	}
	if m.err != nil {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+m.err.Error()) + "\n")
	}
	return baseStyle.Width(m.width - 2).Height(m.height - 2).Render(
		b.String(),
	)
}
func (m *model) rebuildRows() {
	todos := []*app.Task{}
	dones := []*app.Task{}
	for _, t := range m.store.List() {
		if t.IsCompleted() {
			dones = append(dones, t)
		} else {
			todos = append(todos, t)
		}
	}
	var rows []row
	rows = append(rows, row{kind: rowHeader, label: fmt.Sprintf(" TODO (%d)", len(todos))})
	for _, t := range todos {
		rows = append(rows, row{kind: rowItem, id: t.ID, label: taskLine(t)})
	}
	rows = append(rows, row{kind: rowHeader, label: fmt.Sprintf(" Completed (%d)", len(dones))})
	for _, t := range dones {
		rows = append(rows, row{kind: rowItem, id: t.ID, label: taskLine(t)})
	}
	m.rows = rows
	// clamp cursor
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		m.cursor = m.nextSelectable(-1, +1)
	}
	// if landed on header, move to next selectable
	if len(m.rows) > 0 && m.rows[m.cursor].kind == rowHeader {
		m.cursor = m.nextSelectable(m.cursor, +1)
	}
}
func taskLine(t *app.Task) string {
	var parts []string
	completed := t.IsCompleted() && t.CompletedAt != nil
	parts = append(parts, titleStyle.Render(t.Title))
	if t.Due != nil && !t.IsCompleted() {
		parts = append(parts, "• due "+t.Due.Format("2006-01-02"))
	}
	if completed {
		parts = append(parts, "• done "+t.CompletedAt.Format("2006-01-02 15:04"))
	}
	if strings.TrimSpace(t.Notes) != "" {
		parts = append(parts, "— "+notesStyle.Render(t.Notes))
	}
	return strings.Join(parts, " ")
}
func (m *model) nextSelectable(start, dir int) int {
	i := start + dir
	for i >= 0 && i < len(m.rows) {
		if m.rows[i].kind == rowItem {
			return i
		}
		i += dir
	}
	// if none in that direction, wrap once
	if dir > 0 {
		// wrap to first item
		for j := 0; j < len(m.rows); j++ {
			if m.rows[j].kind == rowItem {
				return j
			}
		}
	} else {
		for j := len(m.rows) - 1; j >= 0; j-- {
			if m.rows[j].kind == rowItem {
				return j
			}
		}
	}
	return start
}

func (m model) isInCompletedSection(i int) bool {
	// find the closest header above this row
	for j := i; j >= 0; j-- {
		if m.rows[j].kind == rowHeader {
			return strings.Contains(m.rows[j].label, "Completed")
		}
	}
	return false
}

func (m model) findRowByID(id int) int {
	for i, r := range m.rows {
		if r.kind == rowItem && r.id == id {
			return i
		}
	}
	return -1
}

func New() *model {
	store, err := app.Load("todo-tasks.json")
	if err != nil {
		panic(err)
	}
	println("Loaded store with", len(store.List()), "tasks")
	m := &model{
		store: store,
	}
	m.rebuildRows()
	// place cursor on first selectable item
	m.cursor = m.nextSelectable(-1, +1)
	return m

}
