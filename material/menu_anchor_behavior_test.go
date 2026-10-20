// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// anchorTester pumps a MenuAnchor with a button as its child, aligned in the body
// of a Scaffold.
type anchorTester struct {
	*gluttertest.Tester
	controller *widgets.MenuController
	opened     int
	closed     int
	statuses   []animation.AnimationStatus
	pressed    []string
}

func (a *anchorTester) item(name string, o MenuItemButtonOpts) widgets.Widget {
	o.OnPressed = func() { a.pressed = append(a.pressed, name) }
	o.Child = widgets.NewText(name)
	return NewMenuItemButton(o)
}

func pumpAnchor(t *testing.T, size ui.Size, align painting.AlignmentGeometry, opts func(a *anchorTester) MenuAnchorOpts, children func(a *anchorTester) []widgets.Widget) *anchorTester {
	t.Helper()
	a := &anchorTester{controller: widgets.NewMenuController()}
	a.Tester = gluttertest.New(t, gluttertest.Opts{Size: size})
	var o MenuAnchorOpts
	if opts != nil {
		o = opts(a)
	}
	o.Controller = a.controller
	o.OnOpen = func() { a.opened++ }
	o.OnClose = func() { a.closed++ }
	o.OnAnimationStatusChanged = func(s animation.AnimationStatus) { a.statuses = append(a.statuses, s) }
	if o.Child == nil {
		o.Child = NewTextButton(func() { a.controller.Open() }, widgets.NewText("Anchor"))
	}
	a.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewAlign(NewMenuAnchor(children(a), o), widgets.AlignOpts{Alignment: align})})}))
	return a
}

func twoItems(a *anchorTester) []widgets.Widget {
	return []widgets.Widget{a.item("One", MenuItemButtonOpts{}), a.item("Two", MenuItemButtonOpts{})}
}

func TestMenuAnchorController(t *testing.T) {
	a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, twoItems)
	if a.controller.IsOpen() {
		t.Fatal("the menu starts open")
	}
	a.Tap(gluttertest.ByText("Anchor"))
	a.PumpAndSettle()
	if !a.controller.IsOpen() || !a.Exists(gluttertest.ByText("One")) || a.opened != 1 {
		t.Fatalf("after the tap: open %v, One shown %v, onOpen %d", a.controller.IsOpen(), a.Exists(gluttertest.ByText("One")), a.opened)
	}
	// The menu opens below the anchor, aligned with its start (the default
	// alignment is the top end of the menu in the anchor, flipped to fit).
	anchor := a.TopLeft(gluttertest.ByType[*MenuAnchor]())
	first := a.TopLeft(gluttertest.ByText("One"))
	if first.Dy < anchor.Dy {
		t.Errorf("the menu is above the anchor: %v vs %v", first, anchor)
	}
	a.controller.Close()
	a.PumpAndSettle()
	if a.controller.IsOpen() || a.Exists(gluttertest.ByText("One")) || a.closed != 1 {
		t.Errorf("after Close: open %v, One shown %v, onClose %d", a.controller.IsOpen(), a.Exists(gluttertest.ByText("One")), a.closed)
	}
}

func TestMenuAnchorOutsideTap(t *testing.T) {
	tests := []struct {
		name            string
		consume         bool
		wantOtherTapped int
	}{
		{"an outside tap closes the menu and reaches the widget below", false, 1},
		{"consumeOutsideTap eats the tap", true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			otherTapped := 0
			a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentTopLeft, func(a *anchorTester) MenuAnchorOpts {
				return MenuAnchorOpts{ConsumeOutsideTap: tc.consume}
			}, twoItems)
			_ = otherTapped
			a.controller.Open()
			a.PumpAndSettle()
			a.TapAt(ui.Offset{Dx: 700, Dy: 500})
			a.PumpAndSettle()
			if a.controller.IsOpen() {
				t.Error("the menu is still open after an outside tap")
			}
		})
	}
}

func TestMenuAnchorOpenAtPosition(t *testing.T) {
	a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentTopLeft, func(a *anchorTester) MenuAnchorOpts {
		return MenuAnchorOpts{Child: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(200.0)})}
	}, twoItems)
	anchor := a.TopLeft(gluttertest.ByType[*MenuAnchor]())
	a.controller.Open(ui.Offset{Dx: 40, Dy: 30})
	a.PumpAndSettle()
	menu := a.TopLeft(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("One"), gluttertest.ByType[*Material]())))
	_ = menu
	got := a.TopLeft(gluttertest.ByText("One"))
	// The item sits inside the padding of the panel and the button: the panel
	// starts at the position.
	if got.Dx < anchor.Dx+40 || got.Dy < anchor.Dy+30 || got.Dx > anchor.Dx+40+40 || got.Dy > anchor.Dy+30+40 {
		t.Errorf("the first item is at %v, want near %v", got, ui.Offset{Dx: anchor.Dx + 40, Dy: anchor.Dy + 30})
	}
}

func TestMenuAnchorStaysInTheView(t *testing.T) {
	size := ui.Size{Width: 400, Height: 300}
	tests := []struct {
		name  string
		align painting.AlignmentGeometry
	}{
		{"bottom right", painting.AlignmentBottomRight},
		{"bottom left", painting.AlignmentBottomLeft},
		{"top right", painting.AlignmentTopRight},
		{"top left", painting.AlignmentTopLeft},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := pumpAnchor(t, size, tc.align, nil, twoItems)
			a.controller.Open()
			a.PumpAndSettle()
			for _, text := range []string{"One", "Two"} {
				tl, br := a.TopLeft(gluttertest.ByText(text)), a.BottomRight(gluttertest.ByText(text))
				if tl.Dx < 0 || tl.Dy < 0 || br.Dx > size.Width || br.Dy > size.Height {
					t.Errorf("%s is out of the view: %v to %v", text, tl, br)
				}
			}
		})
	}
}

func TestMenuAnchorAnimated(t *testing.T) {
	a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, func(*anchorTester) MenuAnchorOpts {
		return MenuAnchorOpts{Animated: true}
	}, twoItems)
	a.controller.Open()
	a.Pump()
	a.Pump(100 * time.Millisecond)
	if !a.Exists(gluttertest.ByText("One")) {
		t.Fatal("the menu is not shown while it opens")
	}
	a.PumpAndSettle()
	a.controller.Close()
	a.Pump()
	a.Pump(50 * time.Millisecond)
	if !a.Exists(gluttertest.ByText("One")) {
		t.Error("the menu left at once instead of animating the closing")
	}
	a.PumpAndSettle()
	if a.Exists(gluttertest.ByText("One")) {
		t.Error("the menu did not close after the animation")
	}
	want := []animation.AnimationStatus{animation.AnimationStatusForward, animation.AnimationStatusCompleted, animation.AnimationStatusReverse, animation.AnimationStatusDismissed}
	if len(a.statuses) != len(want) {
		t.Fatalf("statuses = %v, want %v", a.statuses, want)
	}
	for i := range want {
		if a.statuses[i] != want[i] {
			t.Errorf("status %d = %v, want %v", i, a.statuses[i], want[i])
		}
	}
}

func TestMenuItemButton(t *testing.T) {
	tests := []struct {
		name  string
		opts  MenuItemButtonOpts
		tap   bool
		check func(t *testing.T, a *anchorTester)
	}{
		{"press closes the menu and calls onPressed", MenuItemButtonOpts{}, true, func(t *testing.T, a *anchorTester) {
			if a.controller.IsOpen() || len(a.pressed) != 1 {
				t.Errorf("open %v, pressed %v", a.controller.IsOpen(), a.pressed)
			}
		}},
		{"NoCloseOnActivate keeps the menu open", MenuItemButtonOpts{NoCloseOnActivate: true}, true, func(t *testing.T, a *anchorTester) {
			if !a.controller.IsOpen() || len(a.pressed) != 1 {
				t.Errorf("open %v, pressed %v", a.controller.IsOpen(), a.pressed)
			}
		}},
		{"shortcut label", MenuItemButtonOpts{Shortcut: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyK, widgets.SingleActivatorOpts{Control: true})}, false, func(t *testing.T, a *anchorTester) {
			if !a.Exists(gluttertest.ByText("Ctrl+K")) {
				t.Error("the shortcut is not shown")
			}
		}},
		{"leading and trailing icons", MenuItemButtonOpts{LeadingIcon: NewIcon(Icons.Add()), TrailingIcon: NewIcon(Icons.Send())}, false, func(t *testing.T, a *anchorTester) {
			item := gluttertest.Descendant(gluttertest.ByType[*MenuItemButton](), gluttertest.ByType[*widgets.Icon]())
			if got := a.Count(item); got != 2 {
				t.Errorf("icons = %d, want 2", got)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, func(a *anchorTester) []widgets.Widget {
				return []widgets.Widget{a.item("Item", tc.opts)}
			})
			a.controller.Open()
			a.PumpAndSettle()
			if tc.tap {
				a.Tap(gluttertest.ByText("Item"))
				a.PumpAndSettle()
				a.FlushPosted()
				a.PumpAndSettle()
			}
			tc.check(t, a)
		})
	}
}

func TestMenuItemButtonDisabled(t *testing.T) {
	a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, func(a *anchorTester) []widgets.Widget {
		return []widgets.Widget{NewMenuItemButton(MenuItemButtonOpts{Child: widgets.NewText("Off")})}
	})
	a.controller.Open()
	a.PumpAndSettle()
	a.Tap(gluttertest.ByText("Off"))
	a.PumpAndSettle()
	if !a.controller.IsOpen() {
		t.Error("a disabled item closed the menu")
	}
}

func TestMenuItemButtonHoverFocus(t *testing.T) {
	tests := []struct {
		name      string
		noRequest bool
		wantFocus bool
	}{
		{"hover takes the focus", false, true},
		{"NoRequestFocusOnHover keeps it away", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := widgets.NewFocusNode()
			hovered := []bool{}
			a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, func(a *anchorTester) []widgets.Widget {
				return []widgets.Widget{NewMenuItemButton(MenuItemButtonOpts{FocusNode: node, NoRequestFocusOnHover: tc.noRequest,
					OnPressed: func() {}, OnHover: func(h bool) { hovered = append(hovered, h) }, Child: widgets.NewText("Item")})}
			})
			a.controller.Open()
			a.PumpAndSettle()
			a.Hover(a.Center(gluttertest.ByText("Item")))
			a.PumpAndSettle()
			if node.HasPrimaryFocus() != tc.wantFocus {
				t.Errorf("focused = %v, want %v", node.HasPrimaryFocus(), tc.wantFocus)
			}
			a.Hover(ui.Offset{Dx: 5, Dy: 5})
			a.PumpAndSettle()
			if len(hovered) != 2 || !hovered[0] || hovered[1] {
				t.Errorf("onHover calls = %v, want [true false]", hovered)
			}
		})
	}
}

func TestCheckboxMenuButtonValues(t *testing.T) {
	tests := []struct {
		name     string
		tristate bool
		from     foundation.Opt[bool]
		want     foundation.Opt[bool]
	}{
		{"false becomes true", false, foundation.Some(false), foundation.Some(true)},
		{"true becomes false", false, foundation.Some(true), foundation.Some(false)},
		{"true becomes none when tristate", true, foundation.Some(true), foundation.None[bool]()},
		{"none becomes false", true, foundation.None[bool](), foundation.Some(false)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []foundation.Opt[bool]
			a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, func(a *anchorTester) []widgets.Widget {
				return []widgets.Widget{NewCheckboxMenuButton(tc.from, func(v foundation.Opt[bool]) { got = append(got, v) }, widgets.NewText("Check"),
					CheckboxMenuButtonOpts{Tristate: tc.tristate})}
			})
			a.controller.Open()
			a.PumpAndSettle()
			a.Tap(gluttertest.ByText("Check"))
			a.PumpAndSettle()
			a.FlushPosted()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("onChanged = %v, want [%v]", got, tc.want)
			}
		})
	}
	t.Run("disabled without onChanged", func(t *testing.T) {
		if NewCheckboxMenuButton(foundation.Some(true), nil, widgets.NewText("x")).Enabled() {
			t.Error("a checkbox menu button without onChanged is enabled")
		}
	})
}

func TestRadioMenuButtonValues(t *testing.T) {
	tests := []struct {
		name       string
		toggleable bool
		group      foundation.Opt[string]
		want       foundation.Opt[string]
	}{
		{"selects the value", false, foundation.Some("b"), foundation.Some("a")},
		{"selected stays selected", false, foundation.Some("a"), foundation.Some("a")},
		{"toggleable selected becomes none", true, foundation.Some("a"), foundation.None[string]()},
		{"toggleable other selects", true, foundation.None[string](), foundation.Some("a")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []foundation.Opt[string]
			a := pumpAnchor(t, ui.Size{Width: 800, Height: 600}, painting.AlignmentCenter, nil, func(a *anchorTester) []widgets.Widget {
				return []widgets.Widget{NewRadioMenuButton("a", tc.group, func(v foundation.Opt[string]) { got = append(got, v) }, widgets.NewText("Radio"),
					RadioMenuButtonOpts{Toggleable: tc.toggleable})}
			})
			a.controller.Open()
			a.PumpAndSettle()
			a.Tap(gluttertest.ByText("Radio"))
			a.PumpAndSettle()
			a.FlushPosted()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("onChanged = %v, want [%v]", got, tc.want)
			}
		})
	}
}

func TestSubmenuButtonKeyboard(t *testing.T) {
	m := pumpMenuBar(t, fileMenuBar)
	key := func(k services.LogicalKeyboardKey) {
		m.SendKeyEvent(k)
		m.PumpAndSettle()
	}
	// Focus the File button, and open its menu with Down.
	m.Tap(gluttertest.ByText("File"))
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("New")) {
		t.Fatal("the File menu did not open")
	}
	key(services.LogicalKeyboardKeyArrowDown)
	key(services.LogicalKeyboardKeyArrowDown)
	key(services.LogicalKeyboardKeyArrowDown)
	// Right on the Recent submenu button opens it and focuses the first item.
	key(services.LogicalKeyboardKeyArrowRight)
	if !m.Exists(gluttertest.ByText("One")) {
		t.Fatal("Right did not open the submenu")
	}
	// Left, in a submenu of a menu bar's menu, focuses the button that opened the
	// submenu, and leaves the submenu open.
	key(services.LogicalKeyboardKeyArrowLeft)
	if !m.Exists(gluttertest.ByText("One")) {
		t.Error("Left closed the submenu of a menu bar's menu")
	}
	// Escape closes everything.
	key(services.LogicalKeyboardKeyEscape)
	if m.Exists(gluttertest.ByText("New")) || m.Exists(gluttertest.ByText("One")) {
		t.Error("Escape did not close the menu")
	}
}

func TestSubmenuButtonBarArrows(t *testing.T) {
	m := pumpMenuBar(t, fileMenuBar)
	m.Tap(gluttertest.ByText("File"))
	m.PumpAndSettle()
	m.SendKeyEvent(services.LogicalKeyboardKeyArrowRight)
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("Copy")) || m.Exists(gluttertest.ByText("Open")) {
		t.Errorf("Right in the bar should move to the Edit menu: Copy shown %v, Open shown %v",
			m.Exists(gluttertest.ByText("Copy")), m.Exists(gluttertest.ByText("Open")))
	}
	m.SendKeyEvent(services.LogicalKeyboardKeyArrowLeft)
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("Open")) {
		t.Error("Left in the bar should move back to the File menu")
	}
}

func TestSubmenuButtonHoverOpenDelay(t *testing.T) {
	m := pumpMenuBar(t, func(m *menuTester) widgets.Widget {
		return NewMenuBar([]widgets.Widget{NewSubmenuButton([]widgets.Widget{
			NewSubmenuButton([]widgets.Widget{m.item("Deep")}, widgets.NewText("Nested"), SubmenuButtonOpts{HoverOpenDelay: 300 * time.Millisecond}),
		}, widgets.NewText("Menu"))})
	})
	m.Tap(gluttertest.ByText("Menu"))
	m.PumpAndSettle()
	m.Hover(m.Center(gluttertest.ByText("Nested")))
	m.Pump(100 * time.Millisecond)
	if m.Exists(gluttertest.ByText("Deep")) {
		t.Fatal("the submenu opened before the hover delay")
	}
	m.Pump(300 * time.Millisecond)
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByText("Deep")) {
		t.Error("the submenu did not open after the hover delay")
	}
}

func TestMenuBarNeedsAnOverlay(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewMenuBar(nil)))
	if len(ts.TakeErrors()) == 0 {
		t.Error("a MenuBar without an Overlay did not report an error")
	}
}
