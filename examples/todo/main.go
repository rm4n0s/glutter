// Command todo is a to-do list in a table. A field and an Add button at the top add a
// task; each row of the DataTable has a checkbox that marks the task done, its title, and an
// Edit and a Delete button. Edit opens a dialog with the title in a text field, and Delete
// removes the row. It uses only package material, imported as m.
//
// Run it with: go run ./examples/todo
//
// On Windows: go run ./cmd/glutter-windows run ./examples/todo (under Wine on Linux, headless with
// -headless; GLUTTER_BACKEND is win32 there). Nothing of this example is missing on Windows.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>.
package main

import (
	"context"
	"log"
	"math"
	"strconv"
	"strings"

	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(NewTodoApp(), m.RunAppOpts{
		AppID:  "dev.glutter.todo",
		Window: m.WindowConfig{Title: "To-do", Size: m.Size{Width: 560, Height: 520}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Task is one row of the table.
type Task struct {
	ID    int
	Title string
	Done  bool
}

// TodoApp is the Material app around the page.
type TodoApp struct{ m.StatelessWidget }

// NewTodoApp creates the app.
func NewTodoApp() *TodoApp { return &TodoApp{} }

// Build builds the app.
func (a *TodoApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "To-do",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Indigo.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewTodoPage(),
	})
}

// TodoPage holds the list of tasks.
type TodoPage struct{ m.StatefulWidget }

// NewTodoPage creates the page, with three tasks to start with.
func NewTodoPage(opts ...m.WidgetOpts) *TodoPage {
	return &TodoPage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *TodoPage) CreateState() m.State { return &todoPageState{} }

type todoPageState struct {
	m.StateOf[*TodoPage]
	tasks  []*Task
	nextID int
	field  *m.TextEditingController
}

func (s *todoPageState) InitState() {
	s.field = m.NewTextEditingController()
	for _, title := range []string{"Buy milk", "Write the report", "Call the bank"} {
		s.tasks = append(s.tasks, &Task{ID: s.nextID, Title: title})
		s.nextID++
	}
}

func (s *todoPageState) Dispose() { s.field.Dispose() }

// add appends a task with the text of the field, unless it is blank.
func (s *todoPageState) add() {
	title := strings.TrimSpace(s.field.Text())
	if title == "" {
		return
	}
	s.SetState(func() {
		s.tasks = append(s.tasks, &Task{ID: s.nextID, Title: title})
		s.nextID++
		s.field.SetText("")
	})
}

// remove deletes the task.
func (s *todoPageState) remove(t *Task) {
	s.SetState(func() {
		for i, other := range s.tasks {
			if other == t {
				s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
				return
			}
		}
	})
}

// edit opens the dialog for the task and renames it with the answer: a blank or cancelled
// answer leaves the title as it is. The dialog's result comes from a channel, which a
// goroutine of the state waits on.
func (s *todoPageState) edit(t *Task) {
	result := m.ShowDialog[string](s.Context(), func(bc m.BuildContext) m.Widget { return NewEditDialog(t.Title) })
	s.Go(func(ctx context.Context) {
		select {
		case title, ok := <-result:
			if !ok {
				return
			}
			title = strings.TrimSpace(title)
			if title == "" {
				return
			}
			s.SetState(func() { t.Title = title })
		case <-ctx.Done():
		}
	})
}

func (s *todoPageState) Build(bc m.BuildContext) m.Widget {
	columns := []m.DataColumn{
		m.NewDataColumn(m.NewText("Done")),
		m.NewDataColumn(m.NewText("Task")),
		m.NewDataColumn(m.NewText("Actions")),
	}
	rows := make([]m.DataRow, len(s.tasks))
	for i, t := range s.tasks {
		rows[i] = m.NewDataRow([]m.DataCell{
			m.NewDataCell(m.NewCheckbox(m.Some(t.Done), func(v m.Opt[bool]) {
				s.SetState(func() { t.Done = v.OrElse(false) })
			})),
			m.NewDataCell(m.NewText(t.Title, m.TextOpts{Style: titleStyle(bc, t)})),
			m.NewDataCell(m.NewRow([]m.Widget{
				m.NewIconButton(func() { s.edit(t) }, m.NewIcon(m.Icons.Edit()), m.IconButtonOpts{Tooltip: "Edit"}),
				m.NewIconButton(func() { s.remove(t) }, m.NewIcon(m.Icons.Delete()), m.IconButtonOpts{Tooltip: "Delete"}),
			})),
		}, m.DataRowOpts{Key: m.NewValueKey(t.ID)})
	}
	remaining := 0
	for _, t := range s.tasks {
		if !t.Done {
			remaining++
		}
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("To-do: " + strconv.Itoa(remaining) + " left")}),
		Body: m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
			m.NewRow([]m.Widget{
				m.NewExpanded(m.NewTextField(m.TextFieldOpts{
					Controller:  s.field,
					Decoration:  &m.InputDecoration{LabelText: m.Some("New task")},
					OnSubmitted: func(string) { s.add() },
				})),
				m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
				m.NewFilledButton(s.add, m.NewText("Add")),
			}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
			m.NewExpanded(m.NewSingleChildScrollView(
				m.NewSizedBox(m.NewDataTable(columns, rows), m.SizedBoxOpts{Width: m.Some(math.Inf(1))}))),
		})),
	})
}

// titleStyle crosses out the title of a task that is done.
func titleStyle(bc m.BuildContext, t *Task) *m.TextStyle {
	style := m.ThemeOf(bc).TextTheme.BodyLarge
	if !t.Done {
		return style
	}
	return style.CopyWith(m.TextStyleCopy{Decoration: m.Some(m.TextDecorationLineThrough)})
}

// EditDialog asks for the new title of a task and pops with it.
type EditDialog struct {
	m.StatefulWidget
	title string
}

// NewEditDialog creates the dialog, with the current title in the field.
func NewEditDialog(title string, opts ...m.WidgetOpts) *EditDialog {
	return &EditDialog{StatefulWidget: m.NewStatefulWidgetBase(opts...), title: title}
}

// Title is the title that the field starts with.
func (d *EditDialog) Title() string { return d.title }

// CreateState creates the state.
func (d *EditDialog) CreateState() m.State { return &editDialogState{} }

type editDialogState struct {
	m.StateOf[*EditDialog]
	field *m.TextEditingController
}

func (s *editDialogState) InitState() { s.field = m.NewTextEditingController(s.Widget().Title()) }

func (s *editDialogState) Dispose() { s.field.Dispose() }

func (s *editDialogState) Build(bc m.BuildContext) m.Widget {
	save := func() { m.NavigatorPop(bc, s.field.Text()) }
	return m.NewAlertDialog(m.AlertDialogOpts{
		Title: m.NewText("Edit task"),
		Content: m.NewTextField(m.TextFieldOpts{
			Controller:  s.field,
			Autofocus:   true,
			OnSubmitted: func(string) { save() },
		}),
		Actions: []m.Widget{
			m.NewTextButton(func() { m.NavigatorPop(bc) }, m.NewText("Cancel")),
			m.NewTextButton(save, m.NewText("Save")),
		},
	})
}
