package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/raw_menu_anchor_test.dart.

// menuOf is Menu of the Dart tests: a RawMenuAnchor whose overlay is a panel of
// the children below the anchor.
func menuOf(controller *MenuController, panel []Widget, child Widget, opts RawMenuAnchorOpts) *RawMenuAnchor {
	if controller == nil {
		controller = NewMenuController()
	}
	return NewRawMenuAnchor(controller, func(bc BuildContext, info RawMenuOverlayInfo) Widget {
		return NewPositioned(NewTapRegion(NewColumn(panel, FlexOpts{MainAxisSize: rendering.MainAxisSizeMin}), TapRegionOpts{GroupID: info.TapRegionGroupID}),
			PositionedOpts{Left: foundation.Some(info.AnchorRect.Right), Top: foundation.Some(info.AnchorRect.Top)})
	}, child, opts)
}

// anchorButton is AnchorButton: a label that toggles the menu of its anchor.
func anchorButton(label string, focus *FocusNode) Widget {
	return NewBuilder(func(bc BuildContext) Widget {
		return NewGestureDetector(NewFocus(ltrText(label), FocusOpts{FocusNode: focus}), GestureDetectorOpts{OnTap: func() {
			c := MenuControllerMaybeOf(bc)
			if c.IsOpen() {
				c.Close()
			} else {
				c.Open()
			}
		}})
	})
}

func menuApp(ts *gluttertest.Tester, home Widget) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewAlign(home, AlignOpts{Alignment: painting.AlignmentTopLeft})}))
}

func showing(ts *gluttertest.Tester, text string) bool { return ts.Exists(gluttertest.ByText(text)) }

func TestRawMenuAnchorIsOpenWhenTheOverlayIsShown(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	menuApp(ts, menuOf(controller, []Widget{ltrText("a")}, anchorButton("anchor", nil), RawMenuAnchorOpts{}))
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if !controller.IsOpen() || !showing(ts, "a") {
		t.Fatalf("after tap: open=%v a=%v", controller.IsOpen(), showing(ts, "a"))
	}
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if controller.IsOpen() || showing(ts, "a") {
		t.Fatalf("after second tap: open=%v a=%v", controller.IsOpen(), showing(ts, "a"))
	}
}

func TestRawMenuAnchorOpenAndCloseToggleTheOverlay(t *testing.T) {
	ts := gluttertest.New(t)
	controller, nested := NewMenuController(), NewMenuController()
	menuApp(ts, menuOf(controller, []Widget{
		ltrText("a"),
		menuOf(nested, []Widget{ltrText("b.a")}, anchorButton("b", nil), RawMenuAnchorOpts{}),
	}, anchorButton("anchor", nil), RawMenuAnchorOpts{}))
	if controller.IsOpen() || showing(ts, "a") {
		t.Fatal("the menu starts open")
	}
	controller.Open()
	ts.Pump()
	if !controller.IsOpen() || !showing(ts, "a") || !showing(ts, "b") || showing(ts, "b.a") {
		t.Fatal("Open did not show the first level only")
	}
	nested.Open()
	ts.Pump()
	if !nested.IsOpen() || !showing(ts, "b.a") {
		t.Fatal("the nested menu did not open")
	}
	controller.Close()
	ts.Pump()
	if controller.IsOpen() || nested.IsOpen() || showing(ts, "a") || showing(ts, "b.a") {
		t.Fatal("Close left menus open")
	}
}

func TestRawMenuAnchorCloseChildrenClosesTheSubmenusOnly(t *testing.T) {
	ts := gluttertest.New(t)
	controller, nested := NewMenuController(), NewMenuController()
	menuApp(ts, menuOf(controller, []Widget{
		ltrText("a"),
		menuOf(nested, []Widget{ltrText("b.a")}, anchorButton("b", nil), RawMenuAnchorOpts{}),
	}, anchorButton("anchor", nil), RawMenuAnchorOpts{}))
	controller.Open()
	ts.Pump()
	ts.Tap(gluttertest.ByText("b"))
	ts.Pump()
	if !showing(ts, "b.a") {
		t.Fatal("the submenu did not open")
	}
	controller.CloseChildren()
	ts.Pump()
	if !controller.IsOpen() || !showing(ts, "b") || showing(ts, "b.a") {
		t.Fatalf("CloseChildren: open=%v b=%v b.a=%v", controller.IsOpen(), showing(ts, "b"), showing(ts, "b.a"))
	}
}

func TestRawMenuAnchorCanOnlyHaveOneOpenChildAnchor(t *testing.T) {
	ts := gluttertest.New(t)
	menuApp(ts, NewRawMenuAnchor(NewMenuController(), func(BuildContext, RawMenuOverlayInfo) Widget {
		return NewColumn([]Widget{
			menuOf(nil, []Widget{ltrText("a.a")}, anchorButton("a", nil), RawMenuAnchorOpts{}),
			menuOf(nil, []Widget{ltrText("b.a")}, anchorButton("b", nil), RawMenuAnchorOpts{}),
		})
	}, anchorButton("anchor", nil)))
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if !showing(ts, "a") || !showing(ts, "b") || showing(ts, "a.a") || showing(ts, "b.a") {
		t.Fatal("unexpected state after opening the root")
	}
	ts.Tap(gluttertest.ByText("a"))
	ts.Pump()
	if !showing(ts, "a.a") || showing(ts, "b.a") {
		t.Fatal("the first submenu did not open alone")
	}
	ts.Tap(gluttertest.ByText("b"))
	ts.Pump()
	if showing(ts, "a.a") || !showing(ts, "b.a") {
		t.Fatal("opening the second submenu did not close the first")
	}
}

func TestRawMenuAnchorGroupCloseAndCloseChildrenCloseTheChildren(t *testing.T) {
	for _, name := range []string{"Close", "CloseChildren"} {
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller, nested := NewMenuController(), NewMenuController()
			menuApp(ts, NewRawMenuAnchorGroup(controller, NewColumn([]Widget{
				menuOf(nested, []Widget{ltrText("b.a")}, anchorButton("b", nil), RawMenuAnchorOpts{}),
			})))
			nested.Open()
			ts.Pump()
			if !controller.IsOpen() || !nested.IsOpen() || !showing(ts, "b.a") {
				t.Fatal("the group is not open with its child")
			}
			if name == "Close" {
				controller.Close()
			} else {
				controller.CloseChildren()
			}
			ts.Pump()
			if controller.IsOpen() || nested.IsOpen() || !showing(ts, "b") || showing(ts, "b.a") {
				t.Fatal("the children did not close")
			}
		})
	}
}

func TestRawMenuAnchorGroupOpenDoesNothing(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	menuApp(ts, NewRawMenuAnchorGroup(controller, ltrText("child")))
	controller.Open()
	ts.Pump()
	if controller.IsOpen() {
		t.Fatal("a group opened")
	}
}

func TestRawMenuAnchorOnOpenAndOnClose(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	opened, closed := 0, 0
	menuApp(ts, menuOf(controller, nil, anchorButton("anchor", nil), RawMenuAnchorOpts{
		OnOpen: func() { opened++ }, OnClose: func() { closed++ },
	}))
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if opened != 1 || closed != 0 {
		t.Fatalf("after open: opened=%d closed=%d", opened, closed)
	}
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if opened != 1 || closed != 1 {
		t.Fatalf("after close: opened=%d closed=%d", opened, closed)
	}
	controller.Open()
	ts.Pump()
	controller.Close()
	ts.Pump()
	if opened != 2 || closed != 2 {
		t.Fatalf("after reopen: opened=%d closed=%d", opened, closed)
	}
}

func TestRawMenuAnchorOnOpenRequestedAndOnCloseRequested(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	var position foundation.Opt[ui.Offset]
	openRequests, closeRequests := 0, 0
	var show, hide foundation.VoidCallback
	menuApp(ts, menuOf(controller, []Widget{ltrText("a")}, anchorButton("anchor", nil), RawMenuAnchorOpts{
		OnOpenRequested: func(p foundation.Opt[ui.Offset], showOverlay foundation.VoidCallback) {
			openRequests++
			position, show = p, showOverlay
		},
		OnCloseRequested: func(hideOverlay foundation.VoidCallback) { closeRequests++; hide = hideOverlay },
	}))
	controller.Open(ui.Offset{Dx: 5, Dy: 6})
	ts.Pump()
	if openRequests != 1 || position.OrElse(ui.Offset{}) != (ui.Offset{Dx: 5, Dy: 6}) || controller.IsOpen() {
		t.Fatalf("request: n=%d position=%v open=%v", openRequests, position, controller.IsOpen())
	}
	show()
	ts.Pump()
	if !controller.IsOpen() || !showing(ts, "a") {
		t.Fatal("showOverlay did not show the menu")
	}
	controller.Close()
	ts.Pump()
	if closeRequests != 1 || !controller.IsOpen() {
		t.Fatalf("close request: n=%d open=%v", closeRequests, controller.IsOpen())
	}
	hide()
	ts.Pump()
	if controller.IsOpen() || showing(ts, "a") {
		t.Fatal("hideOverlay did not hide the menu")
	}
}

func TestRawMenuAnchorOutsideTapTriggersOnCloseRequested(t *testing.T) {
	ts := gluttertest.New(t)
	closeRequests := 0
	controller := NewMenuController()
	menuApp(ts, NewRawMenuAnchor(controller, func(bc BuildContext, info RawMenuOverlayInfo) Widget {
		return NewPositioned(NewTapRegion(ltrText("a"), TapRegionOpts{
			GroupID: info.TapRegionGroupID,
			OnTapOutside: func(*gestures.PointerDownEvent) {
				MenuControllerMaybeOf(bc).Close()
			},
		}), PositionedOpts{Left: foundation.Some(info.AnchorRect.Left), Top: foundation.Some(info.AnchorRect.Bottom)})
	}, anchorButton("anchor", nil), RawMenuAnchorOpts{
		OnCloseRequested: func(hide foundation.VoidCallback) { closeRequests++; hide() },
	}))
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	if !showing(ts, "a") || closeRequests != 0 {
		t.Fatal("the menu is not open")
	}
	ts.TapAt(ui.Offset{})
	ts.Pump()
	if closeRequests != 1 || showing(ts, "a") {
		t.Fatalf("outside tap: requests=%d a=%v", closeRequests, showing(ts, "a"))
	}
}

func TestRawMenuAnchorDismissMenuActionClosesMenus(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	focus := NewFocusNode()
	t.Cleanup(focus.Dispose)
	menuApp(ts, menuOf(nil, []Widget{
		ltrText("a"),
		menuOf(nil, []Widget{
			ltrText("b.a"),
			menuOf(controller, []Widget{ltrText("b.b.a")}, anchorButton("b.b", focus), RawMenuAnchorOpts{}),
		}, anchorButton("b", nil), RawMenuAnchorOpts{}),
	}, anchorButton("anchor", nil), RawMenuAnchorOpts{}))
	ts.Tap(gluttertest.ByText("anchor"))
	ts.Pump()
	ts.Tap(gluttertest.ByText("b"))
	ts.Pump()
	ts.Tap(gluttertest.ByText("b.b"))
	ts.Pump()
	if !controller.IsOpen() {
		t.Fatal("the deepest menu is not open")
	}
	focus.RequestFocus()
	ts.Pump()
	NewActionDispatcher().InvokeAction(NewDismissMenuAction(controller), &DismissIntent{}, focus.Context())
	ts.Pump()
	if showing(ts, "a") {
		t.Fatal("the menus did not close")
	}
}

func TestRawMenuAnchorEscapeClosesMenus(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewMenuController()
	focus := NewFocusNode()
	t.Cleanup(focus.Dispose)
	menuApp(ts, menuOf(controller, []Widget{ltrText("a")}, anchorButton("anchor", focus), RawMenuAnchorOpts{ChildFocusNode: focus}))
	controller.Open()
	focus.RequestFocus()
	ts.Pump()
	if !showing(ts, "a") {
		t.Fatal("the menu did not open")
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	ts.Pump()
	if showing(ts, "a") || controller.IsOpen() {
		t.Fatal("Escape did not close the menu")
	}
}

func TestMenuControllerDetached(t *testing.T) {
	t.Run("close does nothing and isOpen is false", func(t *testing.T) {
		c := NewMenuController()
		c.Close()
		if c.IsOpen() {
			t.Fatal("a detached controller is open")
		}
	})
	t.Run("detached on update", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		c := NewMenuController()
		menuApp(ts, menuOf(c, nil, NewSizedBox(nil), RawMenuAnchorOpts{}))
		c.CloseChildren()
		menuApp(ts, menuOf(nil, nil, NewSizedBox(nil), RawMenuAnchorOpts{}))
		requireTag(t, "MenuControllerNotAttached", c.CloseChildren)
	})
	t.Run("detached on dispose", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		c := NewMenuController()
		menuApp(ts, NewRawMenuAnchorGroup(c, NewSizedBox(nil)))
		c.CloseChildren()
		menuApp(ts, NewSizedBox(nil))
		requireTag(t, "MenuControllerNotAttached", c.CloseChildren)
		requireTag(t, "MenuControllerNotAttached", func() { c.Open() })
	})
}

func TestMenuControllerMultipleAnchorsPanics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	c := NewMenuController()
	// The inner anchor attaches the controller of the outer one last, so the
	// controller found above it is the inner anchor itself.
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: menuOf(c, nil, menuOf(c, nil, NewSizedBox(nil), RawMenuAnchorOpts{}), RawMenuAnchorOpts{})}))
	errs := ts.TakeErrors()
	if len(errs) == 0 {
		t.Fatal("two anchors with one controller did not report an error")
	}
	_ = requireErrorTagFromPanic(t, errs[0], "MenuControllerMultipleAnchors")
}
