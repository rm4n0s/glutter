// Command multiwindow shows several windows. The first window has a button that
// opens a new window and one that opens a dialog; every window is the same page
// with a counter of its own, so any window can open more. A dialog is modal to the
// window that opened it, and closes with it. It uses only package material,
// imported as m.
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
)

func main() {
	err := m.RunApp(m.NewWindowManager(NewMultiWindowApp()), m.RunAppOpts{
		AppID:  "dev.glutter.multiwindow",
		Window: m.WindowConfig{Title: "Main window", Size: m.Size{Width: 420, Height: 360}},
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
	controller, err := m.NewRegularWindowController(bc, m.Size{Width: 420, Height: 360}, m.RegularWindowControllerOpts{
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
			}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
		),
	})
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
