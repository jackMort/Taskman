package results

import (
	"fmt"
	"sort"
	"time"

	"strings"
	"taskman/app"
	"taskman/components/config"
	"taskman/components/popup"

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
	day           time.Time
	store         *app.Store
	rows          []row
	cursor        int // index in rows (can land on headers; movement skips them)
	width         int
	height        int
	err           error
	loading       bool
	pendingDelete *int // ID of task pending deletion, nil if no pending delete
	popup         tea.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) getFadedView() string {
	return lipgloss.NewStyle().Foreground(config.COLOR_SUBTLE).Render(m.View())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle ChoiceResultMsg from popup first, regardless of popup state
	if _, ok := msg.(popup.ChoiceResultMsg); ok {
		// This is a result from the popup, handle it in the results model
	} else if m.popup != nil {
		// If there's a popup and it's not a ChoiceResultMsg, let the popup handle it
		var cmd tea.Cmd
		m.popup, cmd = m.popup.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildRows()

	case app.DaySelectedMsg:
		m.day = msg.Day
		m.cursor = 0
		m.rebuildRows()

	case app.TaskFormResultMsg:
		if msg.Result {
			// add new task
			if strings.TrimSpace(msg.Title) != "" {
				if _, err := m.store.Add(msg.Title, msg.Notes, nil, m.day); err != nil {
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

	case popup.ChoiceResultMsg:
		if msg.ID == "delete" && m.pendingDelete != nil {
			if msg.Result {
				// User confirmed deletion
				if err := m.store.Delete(*m.pendingDelete); err != nil {
					m.err = err
				}
				// move cursor up first so it feels natural
				m.cursor = m.nextSelectable(m.cursor, -1)
				m.rebuildRows()
			}
			m.pendingDelete = nil
		}
		m.popup = nil

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
				m.pendingDelete = &id
				// Get task title for better confirmation message
				taskTitle := "this task"
				if task, err := m.store.Get(id); err == nil {
					taskTitle = fmt.Sprintf("\"%s\"", task.Title)
				}
				question := fmt.Sprintf("Are you sure you want to delete %s?", taskTitle)
				m.popup = popup.NewChoice("delete", m.getFadedView(), m.width, question, false)
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
	overdueStyle  = lipgloss.NewStyle().Foreground(config.COLOR_ERROR).Bold(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	notesStyle    = lipgloss.NewStyle().Italic(true).Foreground(config.COLOR_LIGHTER)
	dateStyle     = lipgloss.NewStyle().Foreground(config.COLOR_LIGHTER)
	emptyStyle    = lipgloss.NewStyle().Foreground(config.COLOR_LIGHTER).Italic(true)
	cursorGlyph   = " › " // looks nice; change to "> " if you prefer
	indent        = "   "
)

func (m model) View() string {
	if m.popup != nil {
		return m.popup.View()
	}

	var b strings.Builder

	isToday := m.day.IsZero() || (m.day.Year() == time.Now().Year() && m.day.YearDay() == time.Now().YearDay())
	if isToday {
		b.WriteString(config.TopHeaderStyle.Render("TODAY'S TASKS"))
	} else {
		b.WriteString(config.TopHeaderStyle.Render(m.day.Format("Monday, January 2") + " TASKS"))
	}

	for i, r := range m.rows {
		switch r.kind {
		case rowHeader:
			b.WriteString(headerStyle.Render(r.label) + "\n")
			// if no items in this section, show a hint
			if i+1 >= len(m.rows) || m.rows[i+1].kind != rowItem {
				b.WriteString(emptyStyle.Render("  (no items, press 'a' to add)") + "\n")
			}

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
	overdue := []*app.Task{}

	// Check if we're viewing today's tasks
	isToday := m.day.IsZero() || (m.day.Year() == time.Now().Year() && m.day.YearDay() == time.Now().YearDay())

	if isToday {
		// Get all tasks to find overdue ones and completed overdue tasks
		allTasks := m.store.List()
		today := time.Now().Truncate(24 * time.Hour)
		for _, t := range allTasks {
			if !t.IsCompleted() {
				// Incomplete overdue tasks
				taskDate := t.Date.Truncate(24 * time.Hour)
				if taskDate.Before(today) {
					overdue = append(overdue, t)
				}
			} else if t.CompletedAt != nil {
				// Completed tasks from past dates that were completed today
				completedDate := t.CompletedAt.Truncate(24 * time.Hour)
				taskDate := t.Date.Truncate(24 * time.Hour)
				if completedDate.Equal(today) && taskDate.Before(today) {
					dones = append(dones, t)
				}
			}
		}

		// Sort overdue tasks by due date (nil due goes last), then by ID
		sort.SliceStable(overdue, func(i, j int) bool {
			a, b := overdue[i], overdue[j]
			if a.Due == nil && b.Due != nil {
				return false
			}
			if a.Due != nil && b.Due == nil {
				return true
			}
			if a.Due != nil && b.Due != nil && !a.Due.Equal(*b.Due) {
				return a.Due.Before(*b.Due)
			}
			return a.ID < b.ID
		})
	}

	// Get tasks for the current day
	for _, t := range m.store.ListByDate(m.day) {
		if t.IsCompleted() {
			dones = append(dones, t)
		} else {
			todos = append(todos, t)
		}
	}

	// Sort completed tasks by completion time (newest first), then by ID
	sort.SliceStable(dones, func(i, j int) bool {
		a, b := dones[i], dones[j]
		if a.CompletedAt != nil && b.CompletedAt != nil && !a.CompletedAt.Equal(*b.CompletedAt) {
			return b.CompletedAt.Before(*a.CompletedAt) // newest first
		}
		return a.ID < b.ID
	})

	var rows []row

	// Add overdue section if viewing today and there are overdue tasks
	if isToday && len(overdue) > 0 {
		rows = append(rows, row{kind: rowHeader, label: fmt.Sprintf(" OVERDUE (%d)", len(overdue))})
		for _, t := range overdue {
			rows = append(rows, row{kind: rowItem, id: t.ID, label: m.taskLineWithOverdue(t, true)})
		}
	}

	rows = append(rows, row{kind: rowHeader, label: fmt.Sprintf(" TODO (%d)", len(todos))})
	for _, t := range todos {
		rows = append(rows, row{kind: rowItem, id: t.ID, label: m.taskLine(t)})
	}
	rows = append(rows, row{kind: rowHeader, label: fmt.Sprintf(" Completed (%d)", len(dones))})
	for _, t := range dones {
		rows = append(rows, row{kind: rowItem, id: t.ID, label: m.taskLine(t)})
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
func (m *model) taskLine(t *app.Task) string {
	return m.taskLineWithOverdue(t, false)
}

func (m *model) taskLineWithOverdue(t *app.Task, isOverdue bool) string {
	completed := t.IsCompleted() && t.CompletedAt != nil
	title := t.Title
	var date string

	if isOverdue {
		// For overdue tasks, show due date and days overdue
		if t.Due != nil {
			daysOverdue := int(time.Since(*t.Due).Hours() / 24)
			if daysOverdue == 0 {
				date = fmt.Sprintf("Due today (%s)", t.Due.Format("2006-01-02"))
			} else if daysOverdue == 1 {
				date = fmt.Sprintf("1 day overdue (%s)", t.Due.Format("2006-01-02"))
			} else {
				date = fmt.Sprintf("%d days overdue (%s)", daysOverdue, t.Due.Format("2006-01-02"))
			}
		} else {
			// If no due date, show the task date as overdue
			daysOverdue := int(time.Since(t.Date).Hours() / 24)
			if daysOverdue == 1 {
				date = fmt.Sprintf("1 day overdue (%s)", t.Date.Format("2006-01-02"))
			} else {
				date = fmt.Sprintf("%d days overdue (%s)", daysOverdue, t.Date.Format("2006-01-02"))
			}
		}
		title = overdueStyle.Render(t.Title)
	} else if completed {
		date = t.CompletedAt.Format("2006-01-02 15:05")
		title = titleStyle.Render(t.Title)
	} else {
		date = dateStyle.Render(t.CreatedAt.Format("2006-01-02 15:04"))
		title = titleStyle.Render(t.Title)
	}

	padding := m.width - lipgloss.Width(title) - lipgloss.Width(date) - 3 - len(indent)
	if padding < 1 {
		padding = 1
	}
	return title + lipgloss.NewStyle().PaddingLeft(padding).Render(date)
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
	m := &model{
		store: store,
	}
	m.rebuildRows()
	// place cursor on first selectable item
	m.cursor = m.nextSelectable(-1, +1)
	return m

}
