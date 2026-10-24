package main

import (
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func buildMenusPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewWrap(children)
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "MenuBar", NewMenuBarDemo()), section(bc, "MenuAnchor", NewMenuAnchorDemo())),
		top(section(bc, "PopupMenuButton", NewPopupMenuDemo()), section(bc, "DropdownButton", NewDropdownDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(200.0)}),
	})
}

// devOpenMenuDelay is how long GLUTTER_DEV_OPEN_MENU waits before it opens a menu.
const devOpenMenuDelay = 400 * time.Millisecond

// devOpenMenu is the menus that GLUTTER_DEV_OPEN_MENU asks to open, in a debug
// build: "file" opens the File menu, "file/recent" opens its submenu too.
func devOpenMenu(getenv func(string) string) string {
	if !foundation.DebugMode {
		return ""
	}
	return getenv("GLUTTER_DEV_OPEN_MENU")
}

// MenuBarDemo is a menu bar with File, Edit and View menus, with shortcuts that
// work, a checkbox and a radio group, and a submenu.
type MenuBarDemo struct{ m.StatefulWidget }

// NewMenuBarDemo creates the demo.
func NewMenuBarDemo() *MenuBarDemo { return &MenuBarDemo{} }

// CreateState creates the state.
func (w *MenuBarDemo) CreateState() m.State { return &menuBarDemoState{} }

type menuBarDemoState struct {
	m.StateOf[*MenuBarDemo]
	status         string
	showToolbar    bool
	size           m.Opt[string]
	fileController *m.MenuController
	recentMenu     *m.MenuController
}

// InitState opens the menus that GLUTTER_DEV_OPEN_MENU asks for.
func (s *menuBarDemoState) InitState() {
	s.status = "none yet"
	s.showToolbar = true
	s.size = m.Some("Medium")
	s.fileController = m.NewMenuController()
	s.recentMenu = m.NewMenuController()
	open := devOpenMenu(os.Getenv)
	if open == "" {
		return
	}
	// A menu closes when the size of the window changes (as in Flutter), and a
	// window gets its size after the first frame: the menus open a little later.
	timers := m.WidgetsBindingOf(s.Context()).Gestures()
	timers.NewTimer(devOpenMenuDelay, func() {
		if !s.Mounted() {
			return
		}
		s.fileController.Open()
		if open != "file/recent" {
			return
		}
		timers.NewTimer(devOpenMenuDelay, func() {
			if s.Mounted() {
				s.recentMenu.Open()
			}
		})
	})
}

func (s *menuBarDemoState) act(name string) func() {
	return func() { s.SetState(func() { s.status = name }) }
}

func shortcut(key services.LogicalKeyboardKey, o m.SingleActivatorOpts) m.SingleActivator {
	return m.NewSingleActivator(key, o)
}

func (s *menuBarDemoState) item(label string, sc m.MenuSerializableShortcut) m.Widget {
	return m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: s.act(label), Shortcut: sc, Child: m.NewText(label)})
}

func (s *menuBarDemoState) Build(bc m.BuildContext) m.Widget {
	ctrl := m.SingleActivatorOpts{Control: true}
	ctrlShift := m.SingleActivatorOpts{Control: true, Shift: true}
	newDoc := shortcut(services.LogicalKeyboardKeyKeyN, ctrl)
	open := shortcut(services.LogicalKeyboardKeyKeyO, ctrl)
	save := shortcut(services.LogicalKeyboardKeyKeyS, ctrl)
	undo := shortcut(services.LogicalKeyboardKeyKeyZ, ctrl)
	redo := shortcut(services.LogicalKeyboardKeyKeyZ, ctrlShift)
	file := m.NewSubmenuButton([]m.Widget{
		s.item("New", newDoc),
		s.item("Open", open),
		m.NewSubmenuButton([]m.Widget{
			s.item("notes.txt", nil), s.item("report.pdf", nil), s.item("photo.png", nil),
		}, m.NewText("Open recent"), m.SubmenuButtonOpts{Controller: s.recentMenu}),
		s.item("Save", save),
		m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: s.act("Quit"), Shortcut: shortcut(services.LogicalKeyboardKeyKeyQ, ctrl),
			LeadingIcon: m.NewIcon(m.Icons.ExitToApp()), Child: m.NewText("Quit")}),
	}, m.NewMenuAcceleratorLabel("&File"), m.SubmenuButtonOpts{Controller: s.fileController})
	edit := m.NewSubmenuButton([]m.Widget{
		s.item("Undo", undo), s.item("Redo", redo),
		s.item("Cut", shortcut(services.LogicalKeyboardKeyKeyX, ctrl)),
		s.item("Copy", shortcut(services.LogicalKeyboardKeyKeyC, ctrl)),
		s.item("Paste", shortcut(services.LogicalKeyboardKeyKeyV, ctrl)),
	}, m.NewMenuAcceleratorLabel("&Edit"))
	var sizes []m.Widget
	for _, name := range []string{"Small", "Medium", "Large"} {
		name := name
		sizes = append(sizes, m.NewRadioMenuButton(name, s.size, func(v m.Opt[string]) { s.SetState(func() { s.size = v; s.status = "Size " + name }) }, m.NewText(name),
			m.RadioMenuButtonOpts{NoCloseOnActivate: true}))
	}
	view := m.NewSubmenuButton(append([]m.Widget{
		m.NewCheckboxMenuButton(m.Some(s.showToolbar), func(v m.Opt[bool]) { s.SetState(func() { s.showToolbar = v.Value(); s.status = "Toolbar" }) },
			m.NewText("Show toolbar"), m.CheckboxMenuButtonOpts{NoCloseOnActivate: true}),
	}, sizes...), m.NewMenuAcceleratorLabel("&View"))
	bar := m.NewMenuBar([]m.Widget{file, edit, view})

	bindings := []m.CallbackShortcutEntry{
		{Activator: newDoc, Callback: s.act("New")}, {Activator: open, Callback: s.act("Open")}, {Activator: save, Callback: s.act("Save")},
		{Activator: undo, Callback: s.act("Undo")}, {Activator: redo, Callback: s.act("Redo")},
	}
	return m.NewCallbackShortcuts(bindings, m.NewFocus(m.NewColumn([]m.Widget{
		bar,
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewText("Last action: " + s.status),
		m.NewText("Hold Alt to see the accelerators."),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.FocusOpts{Autofocus: true}))
}

// MenuAnchorDemo is a region with a context menu that opens where the secondary
// button is pressed.
type MenuAnchorDemo struct{ m.StatefulWidget }

// NewMenuAnchorDemo creates the demo.
func NewMenuAnchorDemo() *MenuAnchorDemo { return &MenuAnchorDemo{} }

// CreateState creates the state.
func (w *MenuAnchorDemo) CreateState() m.State { return &menuAnchorDemoState{} }

type menuAnchorDemoState struct {
	m.StateOf[*MenuAnchorDemo]
	controller *m.MenuController
	last       string
}

func (s *menuAnchorDemoState) InitState() {
	s.controller = m.NewMenuController()
	s.last = "nothing"
}

func (s *menuAnchorDemoState) Build(bc m.BuildContext) m.Widget {
	pick := func(label string) func() { return func() { s.SetState(func() { s.last = label }) } }
	items := []m.Widget{
		m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: pick("Cut"), LeadingIcon: m.NewIcon(m.Icons.ContentCut()), Child: m.NewText("Cut")}),
		m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: pick("Copy"), LeadingIcon: m.NewIcon(m.Icons.ContentCopy()), Child: m.NewText("Copy")}),
		m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: pick("Paste"), LeadingIcon: m.NewIcon(m.Icons.ContentPaste()), Child: m.NewText("Paste")}),
		m.NewMenuItemButton(m.MenuItemButtonOpts{Child: m.NewText("Disabled")}),
	}
	region := m.NewMenuAnchor(items, m.MenuAnchorOpts{
		Controller: s.controller, ConsumeOutsideTap: true, Animated: true,
		Builder: func(bc m.BuildContext, c *m.MenuController, child m.Widget) m.Widget {
			return m.NewGestureDetector(child, m.GestureDetectorOpts{
				OnSecondaryTapDown: func(d gestures.TapDownDetails) { c.Open(d.LocalPosition) },
				OnTap:              func() { c.Close() },
			})
		},
		Child: m.NewContainer(m.NewCenter(m.NewText("Right-click here")), m.ContainerOpts{
			Width: m.Some(columnWidth), Height: m.Some(120.0),
			Decoration: m.NewBoxDecoration(m.BoxDecorationOpts{
				Color:        m.Some(m.ThemeOf(bc).ColorScheme.SurfaceContainerHighest()),
				BorderRadius: m.NewBorderRadiusCircular(12)}),
		}),
	})
	return m.NewColumn([]m.Widget{region, m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText("Chose: "+s.last))},
		m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// PopupMenuDemo has a popup menu on an icon button and one on a text, with a
// checked item, a divider and a disabled item.
type PopupMenuDemo struct{ m.StatefulWidget }

// NewPopupMenuDemo creates the demo.
func NewPopupMenuDemo() *PopupMenuDemo { return &PopupMenuDemo{} }

// CreateState creates the state.
func (w *PopupMenuDemo) CreateState() m.State { return &popupMenuDemoState{} }

type popupMenuDemoState struct {
	m.StateOf[*PopupMenuDemo]
	selected string
	sortBy   string
	bold     bool
}

func (s *popupMenuDemoState) InitState() {
	s.selected = "nothing"
	s.sortBy = "Name"
}

func (s *popupMenuDemoState) Build(bc m.BuildContext) m.Widget {
	entries := func(bc m.BuildContext) []m.PopupMenuEntry {
		return []m.PopupMenuEntry{
			m.NewPopupMenuItem(m.NewText("Copy"), m.PopupMenuItemOpts[string]{Value: m.Some("Copy")}),
			m.NewPopupMenuItem(m.NewText("Paste"), m.PopupMenuItemOpts[string]{Value: m.Some("Paste")}),
			m.NewPopupMenuDivider(),
			m.NewCheckedPopupMenuItem(m.NewText("Bold"), m.CheckedPopupMenuItemOpts[string]{PopupMenuItemOpts: m.PopupMenuItemOpts[string]{Value: m.Some("Bold")}, Checked: s.bold}),
			m.NewPopupMenuItem(m.NewText("Delete"), m.PopupMenuItemOpts[string]{Value: m.Some("Delete"), Disabled: true}),
		}
	}
	sortEntries := func(bc m.BuildContext) []m.PopupMenuEntry {
		var out []m.PopupMenuEntry
		for _, v := range []string{"Name", "Date", "Size"} {
			out = append(out, m.NewPopupMenuItem(m.NewText(v), m.PopupMenuItemOpts[string]{Value: m.Some(v)}))
		}
		return out
	}
	return m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{
			m.NewPopupMenuButton(entries, m.PopupMenuButtonOpts[string]{
				OnSelected: func(v string) {
					s.SetState(func() {
						s.selected = v
						if v == "Bold" {
							s.bold = !s.bold
						}
					})
				},
				OnCanceled: func() { s.SetState(func() { s.selected = "canceled" }) },
			}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(24.0)}),
			m.NewPopupMenuButton(sortEntries, m.PopupMenuButtonOpts[string]{
				InitialValue: m.Some(s.sortBy), Position: m.PopupMenuPositionUnder, Tooltip: "Sort by",
				OnSelected: func(v string) { s.SetState(func() { s.sortBy = v }) },
				Child: m.NewPadding(m.NewEdgeInsetsSymmetric(8, 12), m.NewRow([]m.Widget{
					m.NewText("Sort by: " + s.sortBy), m.NewIcon(m.Icons.ArrowDropDown()),
				}, m.RowOpts{MainAxisSize: m.MainAxisSizeMin})),
			}),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentCenter}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText("Selected: "+s.selected)),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// DropdownDemo has dropdown buttons: with a value, with a hint, and disabled.
type DropdownDemo struct{ m.StatefulWidget }

// NewDropdownDemo creates the demo.
func NewDropdownDemo() *DropdownDemo { return &DropdownDemo{} }

// CreateState creates the state.
func (w *DropdownDemo) CreateState() m.State { return &dropdownDemoState{} }

type dropdownDemoState struct {
	m.StateOf[*DropdownDemo]
	fruit m.Opt[string]
	size  m.Opt[int]
}

func (s *dropdownDemoState) InitState() { s.fruit = m.Some("Banana") }

func (s *dropdownDemoState) Build(bc m.BuildContext) m.Widget {
	var fruits []*m.DropdownMenuItem[string]
	for _, f := range []string{"Apple", "Banana", "Cherry", "Date"} {
		fruits = append(fruits, m.NewDropdownMenuItem(m.NewText(f), m.DropdownMenuItemOpts[string]{Value: m.Some(f), Disabled: f == "Date"}))
	}
	var sizes []*m.DropdownMenuItem[int]
	for _, n := range []int{10, 20, 30} {
		sizes = append(sizes, m.NewDropdownMenuItem(m.NewText(itoa(n)+" px"), m.DropdownMenuItemOpts[int]{Value: m.Some(n)}))
	}
	gap := m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)})
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewDropdownButton(fruits, func(v m.Opt[string]) { s.SetState(func() { s.fruit = v }) },
			m.DropdownButtonOpts[string]{Value: s.fruit, IsExpanded: true}), m.SizedBoxOpts{Width: m.Some(220.0)}),
		gap,
		m.NewSizedBox(m.NewDropdownButton(sizes, func(v m.Opt[int]) { s.SetState(func() { s.size = v }) },
			m.DropdownButtonOpts[int]{Value: s.size, Hint: m.NewText("Choose a size"), IsExpanded: true}), m.SizedBoxOpts{Width: m.Some(220.0)}),
		gap,
		m.NewSizedBox(m.NewDropdownButton(fruits, nil,
			m.DropdownButtonOpts[string]{Value: s.fruit, DisabledHint: m.NewText("Disabled"), IsExpanded: true}), m.SizedBoxOpts{Width: m.Some(220.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
