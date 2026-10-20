package widgets_test

import (
	"fmt"
	"github.com/rm4n0s/glutter/ui"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/context_menu_controller_test.dart.
// The clipboard fill of its setUp is not needed: the paste button is not part of
// what these tests look at.

// pumpMenuHome pumps an app and returns the context of its home.
func pumpMenuHome(ts *gluttertest.Tester) *BuildContext {
	var ctx BuildContext
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewBuilder(func(bc BuildContext) Widget {
		ctx = bc
		return NewSizedBoxShrink(nil)
	})}))
	return &ctx
}

func keyed(key foundation.Key) WidgetBuilder {
	return func(BuildContext) Widget { return NewSizedBox(nil, SizedBoxOpts{Key: key}) }
}

func TestContextMenuControllerHidesAndShowsOnlyASingleMenu(t *testing.T) {
	ts := gluttertest.New(t)
	key1, key2 := NewGlobalKey[State](), NewGlobalKey[State]()
	ctx := pumpMenuHome(ts)
	if ts.Exists(gluttertest.ByKey(key1)) || ts.Exists(gluttertest.ByKey(key2)) {
		t.Fatal("menus before showing")
	}
	controller1 := NewContextMenuController()
	ts.Pump()
	controller1.Show(*ctx, keyed(key1))
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(key1)) || ts.Exists(gluttertest.ByKey(key2)) {
		t.Fatal("first menu not shown")
	}

	// Showing the same thing again does nothing and is not an error.
	controller1.Show(*ctx, keyed(key1))
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(key1)) || ts.Exists(gluttertest.ByKey(key2)) {
		t.Fatal("first menu changed after showing again")
	}

	// Showing a new menu hides the first.
	controller2 := NewContextMenuController()
	controller2.Show(*ctx, keyed(key2))
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(key1)) || !ts.Exists(gluttertest.ByKey(key2)) {
		t.Fatal("second menu did not replace the first")
	}
	controller2.Remove()
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(key1)) || ts.Exists(gluttertest.ByKey(key2)) {
		t.Fatal("menus after removing")
	}
}

func TestContextMenuControllerAMenuCanBeHiddenAndThenReshown(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	ctx := pumpMenuHome(ts)
	controller := NewContextMenuController()
	t.Cleanup(controller.Remove)
	// Instantiating the controller does not show it.
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(key1)) {
		t.Fatal("menu shown by the constructor")
	}
	for i := 0; i < 2; i++ {
		controller.Show(*ctx, keyed(key1))
		ts.Pump()
		if !ts.Exists(gluttertest.ByKey(key1)) {
			t.Fatalf("round %d: menu not shown", i)
		}
		if i == 0 {
			controller.Remove()
			ts.Pump()
			if ts.Exists(gluttertest.ByKey(key1)) {
				t.Fatal("menu not removed")
			}
		}
	}
}

func TestContextMenuControllerMarkNeedsBuildCausesTheBuilderToUpdate(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := pumpMenuHome(ts)
	buildCount := 0
	controller := NewContextMenuController()
	controller.Show(*ctx, func(BuildContext) Widget { buildCount++; return NewSizedBoxShrink(nil) })
	if buildCount != 0 {
		t.Fatalf("buildCount = %d before the frame", buildCount)
	}
	ts.Pump()
	if buildCount != 1 {
		t.Fatalf("buildCount = %d after the first frame", buildCount)
	}
	controller.MarkNeedsBuild()
	if buildCount != 1 {
		t.Fatalf("buildCount = %d before the second frame", buildCount)
	}
	ts.Pump()
	if buildCount != 2 {
		t.Fatalf("buildCount = %d after the second frame", buildCount)
	}
	controller.Remove()
}

func TestContextMenuControllerMarkNeedsBuildWhenNotShown(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewContextMenuController()
	requireTag(t, "ContextMenuNotShown", controller.MarkNeedsBuild)
}

func TestContextMenuControllerShowingWhenABuiltInMenuIsShownHidesTheBuiltInMenu(t *testing.T) {
	ts := gluttertest.New(t)
	builtInKey, directKey := NewGlobalKey[State](), NewGlobalKey[State]()
	var ctx BuildContext
	controller := NewTextEditingController("")
	focus := NewFocusNode()
	t.Cleanup(func() { focus.Dispose(); controller.Dispose() })
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewBuilder(func(bc BuildContext) Widget {
		ctx = bc
		return NewEditableText(controller, focus, plainEditableStyle(), ui.NewColor(0xFFFF0000), ui.NewColor(0xFFAAAAAA), EditableTextOpts{
			SelectionControls: testTextSelectionControls{},
			ContextMenuBuilder: func(BuildContext, *EditableTextState) Widget {
				return NewSizedBox(nil, SizedBoxOpts{Key: builtInKey})
			},
		})
	})}))
	state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
	ts.TapAt(textOffsetToPosition(ts, 0))
	ts.Pump()
	if !state.TryShowToolbar() {
		t.Fatal("TryShowToolbar returned false")
	}
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(builtInKey)) || ts.Exists(gluttertest.ByKey(directKey)) {
		t.Fatal("built-in menu not shown alone")
	}

	direct := NewContextMenuController()
	direct.Show(ctx, keyed(directKey))
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(builtInKey)) || !ts.Exists(gluttertest.ByKey(directKey)) || !direct.IsShown() {
		t.Fatal("direct menu did not replace the built-in menu")
	}

	// And showing the built-in menu hides the directly shown menu.
	if !state.TryShowToolbar() {
		t.Fatal("TryShowToolbar returned false the second time")
	}
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(builtInKey)) || ts.Exists(gluttertest.ByKey(directKey)) || direct.IsShown() {
		t.Fatal("built-in menu did not replace the direct menu")
	}

	// Calling Remove on the hidden controller does not hide the built-in menu.
	direct.Remove()
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(builtInKey)) || direct.IsShown() {
		t.Fatal("removing the hidden controller changed the menus")
	}

	state.HideToolbar(true)
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(builtInKey)) || ts.Exists(gluttertest.ByKey(directKey)) {
		t.Fatal("menus after HideToolbar")
	}
}

// statefulMenu is _StatefulMenu: it keeps the value it was first given.
type statefulMenu struct {
	StatefulWidget
	value int
}

func (w *statefulMenu) CreateState() State { return &statefulMenuState{} }

type statefulMenuState struct {
	StateOf[*statefulMenu]
	initialValue int
}

func (s *statefulMenuState) InitState() { s.initialValue = s.Widget().value }

func (s *statefulMenuState) Build(BuildContext) Widget {
	return NewText(fmt.Sprintf("Initial: %d, Current: %d", s.initialValue, s.Widget().value), TextOpts{TextDirection: ui.TextDirectionLTR})
}

func newStatefulMenu(value int) *statefulMenu {
	return &statefulMenu{NewStatefulWidgetBase(), value}
}

func TestContextMenuControllerShowUpdatesInPlace(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewContextMenuController()
	ctx := pumpMenuHome(ts)
	t.Cleanup(func() { ContextMenuControllerRemoveAny(*ctx) })

	controller.Show(*ctx, func(BuildContext) Widget { return newStatefulMenu(1) })
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("Initial: 1, Current: 1")) {
		t.Fatal("first value not shown")
	}
	controller.Show(*ctx, func(BuildContext) Widget { return newStatefulMenu(2) })
	ts.Pump()
	// If it updates in place, the state is preserved, so the initial value is
	// still 1. If it recreated the entry, the state would be lost.
	if !ts.Exists(gluttertest.ByText("Initial: 1, Current: 2")) {
		t.Fatal("the menu was not updated in place")
	}
	controller.Remove()
	ts.Pump()
}

func TestContextMenuControllerShowAfterRemoveCreatesANewOverlayEntry(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewContextMenuController()
	ctx := pumpMenuHome(ts)
	t.Cleanup(func() { ContextMenuControllerRemoveAny(*ctx) })

	controller.Show(*ctx, func(BuildContext) Widget { return newStatefulMenu(1) })
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("Initial: 1, Current: 1")) {
		t.Fatal("first value not shown")
	}
	controller.Remove()
	ts.Pump()
	if ts.Exists(gluttertest.ByText("Initial: 1, Current: 1")) {
		t.Fatal("menu still shown after Remove")
	}
	controller.Show(*ctx, func(BuildContext) Widget { return newStatefulMenu(2) })
	ts.Pump()
	// The state is fresh, so the initial value is 2.
	if !ts.Exists(gluttertest.ByText("Initial: 2, Current: 2")) {
		t.Fatal("the menu did not get a fresh state")
	}
	controller.Remove()
	ts.Pump()
}

func TestContextMenuControllerOnRemoveIsCalled(t *testing.T) {
	ts := gluttertest.New(t)
	removed := 0
	controller := NewContextMenuController(ContextMenuControllerOpts{OnRemove: func() { removed++ }})
	ctx := pumpMenuHome(ts)
	controller.Show(*ctx, keyed(NewGlobalKey[State]()))
	ts.Pump()
	controller.Remove()
	controller.Remove()
	if removed != 1 {
		t.Errorf("OnRemove called %d times, want 1", removed)
	}
}
