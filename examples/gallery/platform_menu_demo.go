package main

import (
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

// PlatformMenuBarDemo puts menus in the menu bar of the system, where the platform has one (macOS: the menu
// bar at the top of the screen). It draws a line that says what was chosen last. Elsewhere it draws the
// same line and the menus are not shown.
type PlatformMenuBarDemo struct{ m.StatefulWidget }

// NewPlatformMenuBarDemo creates the demo.
func NewPlatformMenuBarDemo() *PlatformMenuBarDemo { return &PlatformMenuBarDemo{} }

// CreateState creates the state.
func (w *PlatformMenuBarDemo) CreateState() m.State { return &platformMenuBarDemoState{} }

type platformMenuBarDemoState struct {
	m.StateOf[*PlatformMenuBarDemo]
	status string
	opened int
}

func (s *platformMenuBarDemoState) InitState() { s.status = "none yet" }

func (s *platformMenuBarDemoState) act(name string) func() {
	return func() { s.SetState(func() { s.status = name }) }
}

func (s *platformMenuBarDemoState) menus() []m.PlatformMenuItem {
	cmd := m.SingleActivatorOpts{Meta: true}
	item := func(label string, sc m.MenuSerializableShortcut) m.PlatformMenuItem {
		return m.NewPlatformMenuItem(label, m.PlatformMenuItemOpts{OnSelected: s.act(label), Shortcut: sc})
	}
	provided := func(t m.PlatformProvidedMenuItemType) m.PlatformMenuItem { return m.NewPlatformProvidedMenuItem(t) }
	var appMenu []m.PlatformMenuItem
	if m.PlatformProvidedMenuItemHasMenu(m.PlatformProvidedMenuItemTypeAbout) {
		appMenu = []m.PlatformMenuItem{
			m.NewPlatformMenuItemGroup([]m.PlatformMenuItem{provided(m.PlatformProvidedMenuItemTypeAbout)}),
			m.NewPlatformMenuItemGroup([]m.PlatformMenuItem{provided(m.PlatformProvidedMenuItemTypeServicesSubmenu)}),
			m.NewPlatformMenuItemGroup([]m.PlatformMenuItem{
				provided(m.PlatformProvidedMenuItemTypeHide),
				provided(m.PlatformProvidedMenuItemTypeHideOtherApplications),
				provided(m.PlatformProvidedMenuItemTypeShowAllApplications),
			}),
			m.NewPlatformMenuItemGroup([]m.PlatformMenuItem{provided(m.PlatformProvidedMenuItemTypeQuit)}),
		}
	}
	return []m.PlatformMenuItem{
		m.NewPlatformMenu("APP_NAME", appMenu),
		m.NewPlatformMenu("File", []m.PlatformMenuItem{
			item("New", shortcut(services.LogicalKeyboardKeyKeyN, cmd)),
			item("Open…", shortcut(services.LogicalKeyboardKeyKeyO, cmd)),
			m.NewPlatformMenuItemGroup([]m.PlatformMenuItem{
				item("Save", shortcut(services.LogicalKeyboardKeyKeyS, cmd)),
				item("Save As…", shortcut(services.LogicalKeyboardKeyKeyS, m.SingleActivatorOpts{Meta: true, Shift: true})),
			}),
		}, m.PlatformMenuOpts{OnOpen: func() { s.SetState(func() { s.opened++ }) }}),
		m.NewPlatformMenu("View", []m.PlatformMenuItem{
			m.NewPlatformMenuItem("Reset zoom", m.PlatformMenuItemOpts{OnSelected: s.act("Reset zoom"), Shortcut: shortcut(services.LogicalKeyboardKeyDigit0, cmd)}),
			m.NewPlatformMenuItem("Disabled item"),
		}),
	}
}

func (s *platformMenuBarDemoState) Build(bc m.BuildContext) m.Widget {
	line := m.NewText("Last chosen: "+s.status, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodyMedium})
	const hint = "On macOS the menus are in the menu bar at the top of the screen."
	return m.NewPlatformMenuBar(s.menus(), m.PlatformMenuBarOpts{Child: m.NewColumn([]m.Widget{
		m.NewText(hint, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodySmall}), line,
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})})
}
