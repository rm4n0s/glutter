// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// menuTester pumps a MenuBar with File and Edit menus in a Material app, and
// records the items that were pressed.
type menuTester struct {
	*gluttertest.Tester
	t       testing.TB
	pressed []string
}

func (m *menuTester) item(name string) widgets.Widget {
	return NewMenuItemButton(MenuItemButtonOpts{OnPressed: func() { m.pressed = append(m.pressed, name) }, Child: widgets.NewText(name)})
}

func pumpMenuBar(t *testing.T, bar func(m *menuTester) widgets.Widget) *menuTester {
	t.Helper()
	m := &menuTester{t: t}
	m.Tester = gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	m.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: bar(m)})}))
	return m
}

func fileMenuBar(m *menuTester) widgets.Widget {
	return NewMenuBar([]widgets.Widget{
		NewSubmenuButton([]widgets.Widget{
			m.item("New"),
			m.item("Open"),
			NewSubmenuButton([]widgets.Widget{m.item("One"), m.item("Two")}, widgets.NewText("Recent")),
		}, widgets.NewText("File")),
		NewSubmenuButton([]widgets.Widget{m.item("Copy"), m.item("Paste")}, widgets.NewText("Edit")),
	})
}

func TestMenuBarOpensAndSelects(t *testing.T) {
	m := pumpMenuBar(t, fileMenuBar)
	if m.Exists(gluttertest.ByText("New")) {
		t.Fatal("the menu is open before it is pressed")
	}
	m.Tap(gluttertest.ByText("File"))
	m.PumpAndSettle()
	for _, text := range []string{"New", "Open", "Recent"} {
		if !m.Exists(gluttertest.ByText(text)) {
			t.Fatalf("%q is not in the open menu", text)
		}
	}
	m.Tap(gluttertest.ByText("Open"))
	m.PumpAndSettle()
	if m.Exists(gluttertest.ByText("New")) {
		t.Error("the menu is still open after an item was selected")
	}
	if len(m.pressed) != 1 || m.pressed[0] != "Open" {
		t.Errorf("pressed = %v, want [Open]", m.pressed)
	}
}

func TestMenuBarSubmenuAndHoverSwitch(t *testing.T) {
	m := pumpMenuBar(t, fileMenuBar)
	m.Tap(gluttertest.ByText("File"))
	m.PumpAndSettle()
	m.Hover(m.Center(gluttertest.ByText("Recent")))
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("One")) {
		t.Fatal("the submenu does not open on hover")
	}
	// Hovering over a sibling of the menu bar opens it, and closes this one.
	m.Hover(m.Center(gluttertest.ByText("Edit")))
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("Copy")) || m.Exists(gluttertest.ByText("Open")) {
		t.Errorf("hovering over Edit: Copy shown %v, Open shown %v", m.Exists(gluttertest.ByText("Copy")), m.Exists(gluttertest.ByText("Open")))
	}
}

func TestMenuBarKeyboardNavigation(t *testing.T) {
	m := pumpMenuBar(t, fileMenuBar)
	m.Tap(gluttertest.ByText("File"))
	m.PumpAndSettle()
	m.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	m.PumpAndSettle()
	m.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	m.PumpAndSettle()
	m.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	m.PumpAndSettle()
	m.FlushPosted()
	m.PumpAndSettle()
	if len(m.pressed) != 1 {
		t.Fatalf("pressed = %v, want one item", m.pressed)
	}
	t.Logf("pressed %v", m.pressed)
	_ = time.Second
	_ = foundation.None[int]
}

// barColor is the color of the Material of the menu bar.
func (m *menuTester) barColor() ui.Color {
	mat := gluttertest.WidgetAs[*Material](m.Tester, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*MenuBar](), gluttertest.ByType[*Material]())))
	return mat.Color().Value()
}

// panelColorOf is the color of the Material of the menu panel that has the text:
// the closest Material above the text that has a color.
func (m *menuTester) panelColorOf(text string) ui.Color {
	var best *Material
	for _, w := range m.Widgets(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*Material]())) {
		if mat := w.(*Material); mat.Color().IsSome() && mat.Type() == MaterialTypeCanvas && mat.Shape() != nil {
			best = mat
			break
		}
	}
	if best == nil {
		m.t.Fatalf("no menu panel above %q", text)
	}
	return best.Color().Value()
}

// textColorOf is the color of the default text style at the text.
func (m *menuTester) textColorOf(text string) ui.Color {
	return widgets.DefaultTextStyleOf(m.Element(gluttertest.ByText(text))).Style().Color.Value()
}
