// Command multiwindow shows several windows. The first window has a button that
// opens a new window and one that opens a dialog; every window is the same page
// with a counter of its own, so any window can open more. A dialog is modal to the
// window that opened it, and closes with it. It uses only package material,
// imported as m.
//
// Every window is also a drop target: drag a file from a file manager, or some text,
// onto "Drop here" to see its path; "Drag me" can be dragged out of the window, into
// a text editor or into another window of the app. (Drags between applications need
// a pointer button held down, so they cannot be tried on a headless compositor.)
//
// Run it with: go run ./examples/multiwindow
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>. In debug
// builds: GLUTTER_DEV_OPEN=window,dialog opens a window, a dialog, or both when the
// first window is shown.
package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(m.NewWindowManager(NewMultiWindowApp()), m.RunAppOpts{
		AppID:  "dev.glutter.multiwindow",
		Window: m.WindowConfig{Title: "Main window", Size: m.Size{Width: 420, Height: 520}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// MultiWindowApp is the app of the first window: a Material app whose home page is
// the page every window shows.
type MultiWindowApp struct{ m.StatelessWidget }

// NewMultiWindowApp creates the app.
func NewMultiWindowApp() *MultiWindowApp { return &MultiWindowApp{} }

func (a *MultiWindowApp) Build(m.BuildContext) m.Widget {
	return windowApp(NewWindowPage(true))
}

// windowApp is the Material app that every window has, around its page.
func windowApp(home m.Widget) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Multiwindow",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       home,
	})
}

// WindowPage is the content of every window: the title of the window, a counter,
// and the buttons that open a window and a dialog. The first window's page also
// opens what GLUTTER_DEV_OPEN names.
type WindowPage struct {
	m.StatefulWidget
	first bool
}

// NewWindowPage creates the page; first is true for the page of the first window.
func NewWindowPage(first bool, opts ...m.WidgetOpts) *WindowPage {
	return &WindowPage{StatefulWidget: m.NewStatefulWidgetBase(opts...), first: first}
}

func (p *WindowPage) CreateState() m.State { return &windowPageState{} }

type windowPageState struct {
	m.StateOf[*WindowPage]
	count   int
	windows int // the windows opened from this page, to number them

	// dropped is what was dropped on the page last: the paths of files, or text.
	dropped []string
}

func (s *windowPageState) InitState() {
	if !s.Widget().first {
		return
	}
	// The windows are made once the page is on screen: a window cannot be made while
	// the tree is being built.
	what := devOpen(os.Getenv)
	if len(what) == 0 {
		return
	}
	m.WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) {
		for _, kind := range what {
			switch kind {
			case "window":
				s.openWindow()
			case "dialog":
				s.openDialog()
			}
		}
	})
}

// closing removes a window from the registry once the window is gone.
type closing struct {
	m.RegularWindowControllerDelegateBase
	done func()
}

func (c closing) OnWindowDestroyed() { c.done() }

// closingDialog is closing for a dialog.
type closingDialog struct {
	m.DialogWindowControllerDelegateBase
	done func()
}

func (c closingDialog) OnWindowDestroyed() { c.done() }

// openWindow opens a window with a page of its own.
func (s *windowPageState) openWindow() {
	bc := s.Context()
	registry := m.WindowRegistryOf(bc)
	s.windows++
	var entry *m.WindowEntry
	title := "Window " + strconv.Itoa(s.windows)
	controller, err := m.NewRegularWindowController(bc, m.Size{Width: 420, Height: 520}, m.RegularWindowControllerOpts{
		Title:    title,
		Delegate: closing{done: func() { registry.Unregister(entry) }},
	})
	if err != nil {
		log.Print(err)
		return
	}
	entry = m.NewWindowEntry(controller, func(m.BuildContext) m.Widget { return windowApp(NewWindowPage(false)) })
	registry.Register(entry)
}

// openDialog opens a dialog that is modal to the window of this page.
func (s *windowPageState) openDialog() {
	bc := s.Context()
	registry := m.WindowRegistryOf(bc)
	var entry *m.WindowEntry
	controller, err := m.NewDialogWindowController(bc, m.Size{Width: 320, Height: 180}, m.DialogWindowControllerOpts{
		Parent:   m.WindowScopeOf(bc),
		Title:    "Dialog",
		Delegate: closingDialog{done: func() { registry.Unregister(entry) }},
	})
	if err != nil {
		log.Print(err)
		return
	}
	entry = m.NewWindowEntry(controller, func(m.BuildContext) m.Widget {
		return windowApp(NewDialogPage(controller))
	})
	registry.Register(entry)
}

func (s *windowPageState) Build(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText(m.WindowScopeTitleOf(bc))}),
		Body: m.NewCenter(
			m.NewColumn([]m.Widget{
				m.NewText("This window's counter:"),
				m.NewText(strconv.Itoa(s.count), m.TextOpts{Style: theme.TextTheme.HeadlineMedium}),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
				m.NewFilledButton(func() { s.SetState(func() { s.count++ }) }, m.NewText("Add one")),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(8.0)}),
				m.NewFilledButtonTonal(s.openWindow, m.NewText("New window")),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(8.0)}),
				m.NewOutlinedButton(s.openDialog, m.NewText("Open dialog")),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(20.0)}),
				s.dragChip(bc),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
				s.dropZone(bc),
			}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
		),
	})
}

// chip is a small rounded label.
func chip(text string, color ui.Color) m.Widget {
	return m.NewMaterial(m.NewPadding(m.NewEdgeInsetsSymmetric(14, 8), m.NewText(text)),
		m.MaterialOpts{Color: m.Some(color), BorderRadius: m.NewBorderRadiusCircular(16)})
}

// dragChip is "Drag me": it can be dragged inside the window, onto the drop zone of
// any window of the app, and out of the window as text.
func (s *windowPageState) dragChip(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	title := m.WindowScopeTitleOf(bc)
	if title == "" {
		title = "this window"
	}
	return m.NewDraggable[string](
		chip("Dragging "+title, scheme.TertiaryContainer()),
		chip("Drag me", scheme.SecondaryContainer()),
		m.DraggableOpts[string]{
			Data:   m.Some(title),
			Export: func(text string) *m.DragExport { return &m.DragExport{Text: "Dragged from " + text} },
		})
}

// dropZone shows what was dropped on it: the paths of the files that a file manager
// dragged here, or text.
func (s *windowPageState) dropZone(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	return m.NewDragTarget(func(bc m.BuildContext, candidates []m.Opt[any], _ []any) m.Widget {
		color := scheme.SurfaceContainerHighest()
		if len(candidates) > 0 {
			color = scheme.PrimaryContainer() // a drag is over the zone and would be accepted
		}
		lines := s.dropped
		if len(lines) == 0 {
			lines = []string{"Drop here: a file or some text"}
		}
		children := make([]m.Widget, len(lines))
		for i, line := range lines {
			children[i] = m.NewText(line, m.TextOpts{MaxLines: m.Some(1), Overflow: m.TextOverflowEllipsis})
		}
		return m.NewContainer(m.NewColumn(children), m.ContainerOpts{
			Width: m.Some(360.0), Padding: m.NewEdgeInsetsAll(12), Color: m.Some(color),
		})
	}, m.DragTargetOpts[any]{OnAcceptWithDetails: func(d m.DragTargetDetails[any]) {
		s.SetState(func() { s.dropped = droppedLines(d.Data) })
	}})
}

// droppedLines is what to show for the data of a drop.
func droppedLines(data any) []string {
	switch v := data.(type) {
	case *m.ExternalDragData:
		if files := v.Files(); len(files) > 0 {
			return files
		}
		if v.Text != "" {
			return []string{v.Text}
		}
		return []string{"(nothing readable was dropped)"}
	case string:
		return []string{"From this app: " + v}
	}
	return nil
}

// DialogPage is the content of a dialog: a message and a button that closes it.
type DialogPage struct {
	m.StatelessWidget
	controller *m.DialogWindowController
}

// NewDialogPage creates the page of the dialog of the controller.
func NewDialogPage(controller *m.DialogWindowController, opts ...m.WidgetOpts) *DialogPage {
	return &DialogPage{StatelessWidget: m.NewStatelessWidgetBase(opts...), controller: controller}
}

func (p *DialogPage) Build(bc m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewCenter(
			m.NewColumn([]m.Widget{
				m.NewText("The window behind is blocked until this closes."),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
				m.NewFilledButton(p.controller.Destroy, m.NewText("Close")),
			}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
		),
	})
}

// devOpen is what GLUTTER_DEV_OPEN asks the first window to open, "window" and
// "dialog" in the order given; it is empty in release and profile builds.
func devOpen(getenv func(string) string) []string {
	if !foundation.DebugMode {
		return nil
	}
	var out []string
	for _, part := range strings.Split(getenv("GLUTTER_DEV_OPEN"), ",") {
		if part = strings.TrimSpace(part); part == "window" || part == "dialog" {
			out = append(out, part)
		}
	}
	return out
}
