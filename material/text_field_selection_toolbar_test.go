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
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests port the cases of material/text_field_test.dart that show the
// toolbar and the handles of a TextField, which flutter_test runs on Android:
// 'Tap shows handles but not toolbar' and its neighbors, 'Tapping selection
// handles toggles the toolbar', 'Check the toolbar appears below the TextField
// when there is not enough space above the TextField to show it', 'the toolbar
// adjusts its position above/below when bottom inset changes', 'Toolbar appears
// in the right places in multiline inputs', 'Selection toolbar fades in', 'The
// selection menu displays in an Overlay without error' and the group 'context
// menu'. The toolbar is the Material one: the platform is Android.

// androidFieldApp pumps a TextField with the text in a Material app on Android.
func androidFieldApp(t *testing.T, text string, field func(*widgets.TextEditingController) widgets.Widget, opts ...gluttertest.Opts) (*gluttertest.Tester, *widgets.TextEditingController) {
	t.Helper()
	controller := widgets.NewTextEditingController(text)
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t, opts...)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(field(controller))}))
	return ts, controller
}

func plainField(controller *widgets.TextEditingController) widgets.Widget {
	return NewTextField(TextFieldOpts{Controller: controller})
}

func editableState(ts *gluttertest.Tester) *widgets.EditableTextState {
	return gluttertest.StateOf[*widgets.EditableTextState](ts, gluttertest.ByType[*widgets.EditableText]())
}

func editableOf(ts *gluttertest.Tester) *rendering.RenderEditable {
	e := ts.Element(gluttertest.ByElementPredicate("a RenderEditable", func(e widgets.Element) bool {
		_, ok := e.RenderObject().(*rendering.RenderEditable)
		return ok
	}))
	return e.RenderObject().(*rendering.RenderEditable)
}

// fieldTextOffsetToPosition is the global position of the caret at the offset.
func fieldTextOffsetToPosition(ts *gluttertest.Tester, offset int) ui.Offset {
	r := editableOf(ts)
	return r.LocalToGlobal(r.GetLocalRectForCaret(ui.NewTextPosition(offset)).Center(), nil)
}

// fieldEndpoints are the global endpoints of the selection.
func fieldEndpoints(ts *gluttertest.Tester, controller *widgets.TextEditingController) []rendering.TextSelectionPoint {
	r := editableOf(ts)
	points := r.GetEndpointsForSelection(controller.Selection())
	for i := range points {
		points[i].Point = r.LocalToGlobal(points[i].Point, nil)
	}
	return points
}

func TestTextFieldAndroidHandlesAndToolbar(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		gesture     func(ts *gluttertest.Tester)
		wantHandles bool
		wantToolbar bool
	}{
		{"Tap shows handles but not toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			ts.Tap(findField())
			ts.Pump()
		}, true, false},
		{"Tap in empty text field does not show handles nor toolbar", "", func(ts *gluttertest.Tester) {
			ts.Tap(findField())
			ts.Pump()
		}, false, false},
		{"Long press shows handles and toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			ts.LongPress(findField())
			ts.Pump()
		}, true, true},
		{"Long press in empty text field shows handles and toolbar", "", func(ts *gluttertest.Tester) {
			ts.LongPress(findField())
			ts.Pump()
		}, true, true},
		{"Double tap shows handles and toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			ts.Tap(findField())
			ts.Pump(50 * time.Millisecond)
			ts.Tap(findField())
			ts.Pump()
		}, true, true},
		{"Double tap in empty text field shows toolbar but not handles", "", func(ts *gluttertest.Tester) {
			ts.Tap(findField())
			ts.Pump(50 * time.Millisecond)
			ts.Tap(findField())
			ts.Pump()
		}, false, true},
		{"Mouse tap does not show handles nor toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			g := ts.StartGesture(ts.Center(findField()), gluttertest.GestureOpts{Pointer: 7, Kind: ui.PointerDeviceKindMouse})
			ts.Pump()
			g.Up()
			ts.Pump()
		}, false, false},
		{"Mouse long press does not show handles nor toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			g := ts.StartGesture(ts.Center(findField()), gluttertest.GestureOpts{Pointer: 7, Kind: ui.PointerDeviceKindMouse})
			ts.Pump(2 * time.Second)
			g.Up()
			ts.Pump()
		}, false, false},
		{"Mouse double tap does not show handles nor toolbar", "abc def ghi", func(ts *gluttertest.Tester) {
			at := ts.Center(findField())
			g := ts.StartGesture(at, gluttertest.GestureOpts{Pointer: 7, Kind: ui.PointerDeviceKindMouse})
			ts.Pump(50 * time.Millisecond)
			g.Up()
			ts.Pump()
			g.Down(at)
			ts.Pump()
			g.Up()
			ts.Pump()
		}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, _ := androidFieldApp(t, tc.text, plainField)
			tc.gesture(ts)
			overlay := editableState(ts).SelectionOverlay()
			if overlay == nil {
				t.Fatal("no selection overlay")
			}
			if got := overlay.HandlesAreVisible(); got != tc.wantHandles {
				t.Errorf("handlesAreVisible = %v, want %v", got, tc.wantHandles)
			}
			if got := overlay.ToolbarIsVisible(); got != tc.wantToolbar {
				t.Errorf("toolbarIsVisible = %v, want %v", got, tc.wantToolbar)
			}
			if tc.wantToolbar && tc.text != "" {
				// The toolbar of Android is the Material one. (An empty field with an
				// empty clipboard has no button to show.)
				if got := ts.Count(gluttertest.ByType[*TextSelectionToolbar]()); got != 1 {
					t.Errorf("TextSelectionToolbars = %d, want 1", got)
				}
			}
		})
	}
}

func TestTextFieldAndroidTappingSelectionHandlesTogglesTheToolbar(t *testing.T) {
	ts, controller := androidFieldApp(t, "abc def ghi", plainField)

	// Tap to position the cursor and show the selection handles.
	ts.TapAt(fieldTextOffsetToPosition(ts, 5), gluttertest.GestureOpts{Pointer: 7}) // Index of 'e'.
	ts.PumpAndSettle()

	overlay := editableState(ts).SelectionOverlay()
	if overlay.ToolbarIsVisible() || !overlay.HandlesAreVisible() {
		t.Fatalf("toolbar %v, handles %v; want no toolbar and handles", overlay.ToolbarIsVisible(), overlay.HandlesAreVisible())
	}

	endpoints := fieldEndpoints(ts, controller)
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %d, want 1", len(endpoints))
	}

	// Tap the handle to show the toolbar.
	handlePos := endpoints[0].Point.Add(ui.Offset{Dy: 1})
	ts.TapAt(handlePos, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if !overlay.ToolbarIsVisible() {
		t.Error("the toolbar is not visible after tapping the handle")
	}

	// Tap the handle again to hide the toolbar.
	ts.TapAt(handlePos, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if overlay.ToolbarIsVisible() {
		t.Error("the toolbar is visible after tapping the handle again")
	}
}

// skipPastScrollingAnimation waits for the scrolling of the text to end.
func skipPastScrollingAnimation(ts *gluttertest.Tester) {
	ts.Pump()
	ts.Pump(200 * time.Millisecond)
}

// showSelectionMenuAt shows the selection menu at the given index into the text
// by tapping to place the cursor and then tapping on the handle.
func showSelectionMenuAt(t *testing.T, ts *gluttertest.Tester, controller *widgets.TextEditingController, index int) {
	t.Helper()
	ts.TapAt(ts.Center(gluttertest.ByType[*widgets.EditableText]()))
	ts.Pump()
	ts.Pump(200 * time.Millisecond) // skip past the frame where the opacity is zero
	if ts.Exists(gluttertest.ByText("Select all")) {
		t.Fatal("the menu is shown before the tap on the handle")
	}
	// Deviation: the handle that the tap above shows is 48 wide and tall where it
	// can be hit, which in the text of the tester covers the position of the next
	// tap, so the handle would take that tap. The test hides it.
	editableState(ts).HideToolbar(true)
	ts.Pump()

	// Tap the selection handle to bring up the "paste / select all" menu for the
	// last line of text.
	ts.TapAt(fieldTextOffsetToPosition(ts, index))
	ts.Pump()
	ts.Pump(200 * time.Millisecond) // skip past the frame where the opacity is zero
	endpoints := fieldEndpoints(ts, controller)
	// Tapping on the part of the handle's GestureDetector where it overlaps with
	// the text itself does not show the menu, so add a small vertical offset to
	// tap below the text.
	ts.TapAt(endpoints[0].Point.Add(ui.Offset{Dx: 1, Dy: 13}))
	ts.Pump()
	ts.Pump(200 * time.Millisecond) // skip past the frame where the opacity is zero
}

func TestTextFieldAndroidToolbarAppearsBelowTheFieldWhenThereIsNotEnoughSpaceAbove(t *testing.T) {
	// This is a regression test for https://github.com/flutter/flutter/issues/29808
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t)
	pump := func(padding float64) {
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewPadding(painting.NewEdgeInsetsAll(padding),
			NewTextField(TextFieldOpts{Controller: controller}))})}))
	}
	const testValue = "abc def ghi"

	pump(30)
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)
	showSelectionMenuAt(t, ts, controller, 5) // the index of 'e'

	// Verify the selection toolbar position is below the text.
	toolbarTopLeft := ts.TopLeft(gluttertest.ByText("Select all"))
	textFieldTopLeft := ts.TopLeft(findField())
	if !(textFieldTopLeft.Dy < toolbarTopLeft.Dy) {
		t.Errorf("the toolbar (%v) is not below the field (%v)", toolbarTopLeft, textFieldTopLeft)
	}

	pump(150)
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)
	showSelectionMenuAt(t, ts, controller, 5)

	// Verify the selection toolbar position.
	toolbarTopLeft = ts.TopLeft(gluttertest.ByText("Select all"))
	textFieldTopLeft = ts.TopLeft(findField())
	if !(toolbarTopLeft.Dy < textFieldTopLeft.Dy) {
		t.Errorf("the toolbar (%v) is not above the field (%v)", toolbarTopLeft, textFieldTopLeft)
	}
}

func TestTextFieldAndroidToolbarAdjustsItsPositionWhenBottomInsetChanges(t *testing.T) {
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewPadding(
		painting.NewEdgeInsetsSymmetric(0, 48),
		widgets.NewColumn([]widgets.Widget{
			widgets.NewIntrinsicHeight(NewTextField(TextFieldOpts{Controller: controller, Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited)})),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(325.0)}),
		}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})))})}))

	const testValue = "abc def ghi"
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)

	showSelectionMenuAt(t, ts, controller, 5)

	// Verify the selection toolbar position is above the text.
	if got := ts.Count(gluttertest.ByText("Select all")); got != 1 {
		t.Fatalf("Select all buttons = %d, want 1", got)
	}
	toolbarTopLeft := ts.TopLeft(gluttertest.ByText("Select all"))
	textFieldTopLeft := ts.TopLeft(findField())
	if !(toolbarTopLeft.Dy < textFieldTopLeft.Dy) {
		t.Errorf("the toolbar (%v) is not above the field (%v)", toolbarTopLeft, textFieldTopLeft)
	}

	// Add a viewInset tall enough to push the field to the top, where there is no
	// room to display the toolbar above. This is similar to when the keyboard is
	// shown. (Dart's 500 is in physical pixels, and flutter_test's pixel ratio is
	// 3; the one of gluttertest is 1.)
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.ViewInsets = ui.ViewPadding{Bottom: 500.0 / 3} })
	ts.PumpAndSettle()

	// Verify the selection toolbar position is below the text.
	toolbarTopLeft = ts.TopLeft(gluttertest.ByText("Select all"))
	textFieldTopLeft = ts.TopLeft(findField())
	if !(toolbarTopLeft.Dy > textFieldTopLeft.Dy) {
		t.Errorf("with the inset the toolbar (%v) is not below the field (%v)", toolbarTopLeft, textFieldTopLeft)
	}

	// Remove the viewInset, as if the keyboard were hidden.
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.ViewInsets = ui.ViewPadding{} })
	ts.PumpAndSettle()

	// Verify the selection toolbar position is above the text.
	toolbarTopLeft = ts.TopLeft(gluttertest.ByText("Select all"))
	textFieldTopLeft = ts.TopLeft(findField())
	if !(toolbarTopLeft.Dy < textFieldTopLeft.Dy) {
		t.Errorf("without the inset the toolbar (%v) is not above the field (%v)", toolbarTopLeft, textFieldTopLeft)
	}
}

func TestTextFieldAndroidToolbarAppearsInTheRightPlacesInMultilineInputs(t *testing.T) {
	// This is a regression test for https://github.com/flutter/flutter/issues/36749
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)}),
		Home: NewScaffold(ScaffoldOpts{Body: widgets.NewPadding(painting.NewEdgeInsetsAll(30),
			NewTextField(TextFieldOpts{Controller: controller, MinLines: foundation.Some(6), MaxLines: foundation.Some(6)}))})}))

	if ts.Exists(gluttertest.ByText("Select all")) {
		t.Fatal("the menu is shown at first")
	}
	const testValue = "abc\ndef\nghi\njkl\nmno\npqr"
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)

	// Show the selection menu on the first line and verify the selection toolbar
	// position is below the first line.
	showSelectionMenuAt(t, ts, controller, 2) // the index of 'c'
	if got := ts.Count(gluttertest.ByText("Select all")); got != 1 {
		t.Fatalf("Select all buttons = %d, want 1", got)
	}
	firstLineToolbarTopLeft := ts.TopLeft(gluttertest.ByText("Select all"))
	firstLineTopLeft := fieldTextOffsetToPosition(ts, 0) // the index of 'a'
	if !(firstLineTopLeft.Dy < firstLineToolbarTopLeft.Dy) {
		t.Errorf("first line: the toolbar (%v) is not below the line (%v)", firstLineToolbarTopLeft, firstLineTopLeft)
	}

	// Show the selection menu on the second to last line and verify the selection
	// toolbar position is above that line and above the first line's toolbar.
	showSelectionMenuAt(t, ts, controller, 18) // the index of 'o'
	if got := ts.Count(gluttertest.ByText("Select all")); got != 1 {
		t.Fatalf("Select all buttons = %d, want 1", got)
	}
	penultimateLineToolbarTopLeft := ts.TopLeft(gluttertest.ByText("Select all"))
	penultimateLineTopLeft := fieldTextOffsetToPosition(ts, 20) // the index of 'p'
	if !(penultimateLineToolbarTopLeft.Dy < penultimateLineTopLeft.Dy) {
		t.Errorf("penultimate line: the toolbar (%v) is not above the line (%v)", penultimateLineToolbarTopLeft, penultimateLineTopLeft)
	}
	if !(penultimateLineToolbarTopLeft.Dy < firstLineToolbarTopLeft.Dy) {
		t.Errorf("penultimate line: the toolbar (%v) is not above the toolbar of the first line (%v)", penultimateLineToolbarTopLeft, firstLineToolbarTopLeft)
	}

	// Show the selection menu on the last line and verify the selection toolbar
	// position is above that line and below the position of the second to last
	// line's toolbar.
	showSelectionMenuAt(t, ts, controller, 22) // the index of 'r'
	if got := ts.Count(gluttertest.ByText("Select all")); got != 1 {
		t.Fatalf("Select all buttons = %d, want 1", got)
	}
	lastLineToolbarTopLeft := ts.TopLeft(gluttertest.ByText("Select all"))
	lastLineTopLeft := fieldTextOffsetToPosition(ts, 20) // the index of 'p'
	if !(lastLineToolbarTopLeft.Dy < lastLineTopLeft.Dy) {
		t.Errorf("last line: the toolbar (%v) is not above the line (%v)", lastLineToolbarTopLeft, lastLineTopLeft)
	}
	if !(lastLineToolbarTopLeft.Dy > penultimateLineToolbarTopLeft.Dy) {
		t.Errorf("last line: the toolbar (%v) is not below the toolbar of the penultimate line (%v)", lastLineToolbarTopLeft, penultimateLineToolbarTopLeft)
	}
}

func TestTextFieldAndroidSelectionToolbarFadesIn(t *testing.T) {
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(widgets.NewCenter(NewTextField(TextFieldOpts{Controller: controller})))}))

	const testValue = "abc def ghi"
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)

	// Tap the selection handle to bring up the "paste / select all" menu.
	ts.TapAt(fieldTextOffsetToPosition(ts, 5))
	ts.Pump()
	// Allow time for the handle to appear and for a double tap to time out.
	ts.Pump(600 * time.Millisecond)
	endpoints := fieldEndpoints(ts, controller)
	ts.TapAt(endpoints[0].Point.Add(ui.Offset{Dx: 1, Dy: 1}))
	// Pump an extra frame to allow the selection menu to read the clipboard.
	ts.Pump()
	ts.Pump()

	// Toolbar should fade in. Starting at 0% opacity.
	if got := ts.Count(gluttertest.ByText("Select all")); got != 1 {
		t.Fatalf("Select all buttons = %d, want 1", got)
	}
	target := ts.Element(gluttertest.ByText("Select all"))
	opacity, ok := widgets.FindAncestorWidgetOfExactType[*widgets.FadeTransition](target)
	if !ok {
		t.Fatal("no FadeTransition above the button")
	}
	if got := opacity.Opacity().Value(); got != 0.0 {
		t.Errorf("opacity = %v, want 0", got)
	}

	// Still fading in.
	ts.Pump(50 * time.Millisecond)
	opacity2, _ := widgets.FindAncestorWidgetOfExactType[*widgets.FadeTransition](target)
	if opacity != opacity2 {
		t.Error("the FadeTransition was replaced")
	}
	if got := opacity.Opacity().Value(); !(got > 0.0 && got < 1.0) {
		t.Errorf("opacity = %v, want it between 0 and 1", got)
	}
}

func TestTextFieldAndroidSelectionMenuDisplaysInAnOverlayWithoutError(t *testing.T) {
	// This is a regression test for https://github.com/flutter/flutter/issues/43787
	controller := widgets.NewTextEditingController("This is a test that shows some odd behavior with Text Selection!")
	t.Cleanup(controller.Dispose)
	ts := newToolbarTester(t)
	entry := widgets.NewOverlayEntry(func(widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(NewTextField(TextFieldOpts{Controller: controller}))
	})
	t.Cleanup(func() { entry.Remove(); entry.Dispose() })
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewColoredBox(Colors.Grey.Shade500(),
		widgets.NewCenter(widgets.NewContainer(widgets.NewOverlay(widgets.OverlayOpts{InitialEntries: []*widgets.OverlayEntry{entry}}),
			widgets.ContainerOpts{Color: foundation.Some(Colors.Red.Shade500()), Width: foundation.Some(300.0), Height: foundation.Some(600.0)})))})}))

	showSelectionMenuAt(t, ts, controller, 20) // the index of 'test'
	ts.PumpAndSettle()
	if err := ts.TakeError(); err != nil {
		t.Errorf("exception: %v", err)
	}
	if !ts.Exists(gluttertest.ByType[*TextSelectionToolbar]()) {
		t.Error("the toolbar is not shown")
	}
}

func TestTextFieldAndroidContextMenu(t *testing.T) {
	longPressShowToolbar := func(ts *gluttertest.Tester) {
		ts.Pump() // Wait for autofocus to take effect.
		ts.LongPress(gluttertest.ByType[*widgets.EditableText]())
		editableState(ts).ShowToolbar()
		ts.PumpAndSettle()
	}

	t.Run("builds AdaptiveTextSelectionToolbar by default", func(t *testing.T) {
		ts, _ := androidFieldApp(t, "", plainField)
		ts.Pump()
		if ts.Exists(gluttertest.ByType[*AdaptiveTextSelectionToolbar]()) {
			t.Fatal("the toolbar is shown at first")
		}
		longPressShowToolbar(ts)
		if got := ts.Count(gluttertest.ByType[*AdaptiveTextSelectionToolbar]()); got != 1 {
			t.Errorf("AdaptiveTextSelectionToolbars = %d, want 1", got)
		}
	})

	t.Run("contextMenuBuilder is used in place of the default text selection toolbar", func(t *testing.T) {
		key := foundation.NewKey("placeholder")
		ts, _ := androidFieldApp(t, "", func(c *widgets.TextEditingController) widgets.Widget {
			return NewTextField(TextFieldOpts{Controller: c, ContextMenuBuilder: func(widgets.BuildContext, *widgets.EditableTextState) widgets.Widget {
				return widgets.NewPlaceholder(nil, widgets.PlaceholderOpts{Key: key})
			}})
		})
		ts.Pump()
		if ts.Exists(gluttertest.ByKey(key)) {
			t.Fatal("the menu is shown at first")
		}
		longPressShowToolbar(ts)
		if got := ts.Count(gluttertest.ByKey(key)); got != 1 {
			t.Errorf("custom menus = %d, want 1", got)
		}
		if ts.Exists(gluttertest.ByType[*AdaptiveTextSelectionToolbar]()) {
			t.Error("the default toolbar is shown too")
		}
	})

	t.Run("contextMenuBuilder changes from default to null", func(t *testing.T) {
		key := widgets.NewGlobalKey[widgets.State]()
		controller := widgets.NewTextEditingController()
		t.Cleanup(controller.Dispose)
		ts := newToolbarTester(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(NewTextField(TextFieldOpts{Key: key, Controller: controller}))}))
		longPressShowToolbar(ts)
		if got := ts.Count(gluttertest.ByType[*AdaptiveTextSelectionToolbar]()); got != 1 {
			t.Fatalf("AdaptiveTextSelectionToolbars = %d, want 1", got)
		}

		// Set contextMenuBuilder to null.
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(NewTextField(TextFieldOpts{Key: key, Controller: controller, NoContextMenu: true}))}))

		// Trigger build one more time...
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(widgets.NewPadding(painting.EdgeInsetsZero,
			NewTextField(TextFieldOpts{Key: key, Controller: controller, NoContextMenu: true})))}))
		if err := ts.TakeError(); err != nil {
			t.Errorf("exception: %v", err)
		}
	})
}

// fieldHandleDragApp pumps a TextField of the given width with the text and a
// scroll controller, as the handle drag tests of Flutter do.
func fieldHandleDragApp(t *testing.T, text string, width float64, scrollController *widgets.ScrollControllerBase) (*gluttertest.Tester, *widgets.TextEditingController) {
	t.Helper()
	return androidFieldApp(t, text, func(c *widgets.TextEditingController) widgets.Widget {
		return widgets.NewAlign(widgets.NewSizedBox(NewTextField(TextFieldOpts{Controller: c, DragStartBehavior: gestures.DragStartBehaviorDown, ScrollController: scrollController}),
			widgets.SizedBoxOpts{Width: foundation.Some(width)}), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
	})
}

// dragHandle drags from the handle position to the new one.
func dragHandle(ts *gluttertest.Tester, from, to ui.Offset) {
	g := ts.StartGesture(from, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(to)
	ts.Pump()
	g.Up()
	ts.Pump()
}

// Flutter's text is Ahem, so 55 characters are wider than the field of 800; the
// Roboto text of the tester is narrower, and the field is 300 wide to have the
// same text longer than the field (SPEC §12.10).
const fieldLongTestValue = "aaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestTextFieldAndroidCanDragTheLeftHandleWhileTheRightHandleRemainsOffScreen(t *testing.T) {
	scrollController := widgets.NewScrollController()
	t.Cleanup(scrollController.Dispose)
	ts, controller := fieldHandleDragApp(t, fieldLongTestValue, 300, scrollController)

	// Double tap 'b' to show handles.
	bPos := fieldTextOffsetToPosition(ts, 28)
	ts.TapAt(bPos)
	ts.Pump(150 * time.Millisecond) // kDoubleTapTimeout ~/ 2
	ts.TapAt(bPos)
	ts.PumpAndSettle()

	selection := controller.Selection()
	if selection.BaseOffset != 28 || selection.ExtentOffset != len(fieldLongTestValue) {
		t.Fatalf("selection = %v, want 28 to %d", selection, len(fieldLongTestValue))
	}

	// Move to the left edge.
	scrollController.JumpTo(0)
	ts.PumpAndSettle()

	endpoints := fieldEndpoints(ts, controller)
	if len(endpoints) != 2 {
		t.Fatalf("endpoints = %d, want 2", len(endpoints))
	}

	// Left handle should appear between textfield's left and right position.
	fieldLeft := ts.TopLeft(findField())
	fieldRight := ts.BottomRight(findField())
	if !(endpoints[0].Point.Dx-fieldLeft.Dx > 0) || !(fieldRight.Dx-endpoints[0].Point.Dx > 0) {
		t.Errorf("the left handle (%v) is not in the field (%v to %v)", endpoints[0].Point, fieldLeft, fieldRight)
	}
	// Right handle should remain off-screen.
	if !(endpoints[1].Point.Dx-fieldRight.Dx > 0) {
		t.Errorf("the right handle (%v) is not off-screen (field right %v)", endpoints[1].Point, fieldRight)
	}

	// Drag the left handle to the right by 25 offset.
	const toOffset = 25
	beforeScrollOffset := scrollController.Offset()
	dragHandle(ts, endpoints[0].Point.Add(ui.Offset{Dx: -1, Dy: 1}), fieldTextOffsetToPosition(ts, toOffset))

	// On Android the base handle stays the base.
	selection = controller.Selection()
	if selection.BaseOffset != toOffset || selection.ExtentOffset != len(fieldLongTestValue) {
		t.Errorf("selection = %v, want %d to %d", selection, toOffset, len(fieldLongTestValue))
	}

	// The scroll area of text field should not move.
	if got := scrollController.Offset(); got != beforeScrollOffset {
		t.Errorf("scroll offset = %v, want %v", got, beforeScrollOffset)
	}
}

func TestTextFieldAndroidCanDragTheRightHandleWhileTheLeftHandleRemainsOffScreen(t *testing.T) {
	scrollController := widgets.NewScrollController()
	t.Cleanup(scrollController.Dispose)
	ts, controller := fieldHandleDragApp(t, fieldLongTestValue, 300, scrollController)

	// Double tap 'a' to show handles.
	aPos := fieldTextOffsetToPosition(ts, 0)
	ts.TapAt(aPos)
	ts.Pump(150 * time.Millisecond) // kDoubleTapTimeout ~/ 2
	ts.TapAt(aPos)
	ts.PumpAndSettle()

	selection := controller.Selection()
	if selection.BaseOffset != 0 || selection.ExtentOffset != 27 {
		t.Fatalf("selection = %v, want 0 to 27", selection)
	}

	// Move to the right edge.
	scrollController.JumpTo(800)
	ts.PumpAndSettle()

	endpoints := fieldEndpoints(ts, controller)
	if len(endpoints) != 2 {
		t.Fatalf("endpoints = %d, want 2", len(endpoints))
	}

	// Right handle should appear between textfield's left and right position.
	fieldLeft := ts.TopLeft(findField())
	fieldRight := ts.BottomRight(findField())
	if !(endpoints[1].Point.Dx-fieldLeft.Dx > 0) || !(fieldRight.Dx-endpoints[1].Point.Dx > 0) {
		t.Errorf("the right handle (%v) is not in the field (%v to %v)", endpoints[1].Point, fieldLeft, fieldRight)
	}
	// Left handle should remain off-screen.
	if !(endpoints[0].Point.Dx < 0) {
		t.Errorf("the left handle (%v) is not off-screen", endpoints[0].Point)
	}

	// Drag the right handle to the left by 50 offset.
	const toOffset = 50
	beforeScrollOffset := scrollController.Offset()
	dragHandle(ts, endpoints[1].Point.Add(ui.Offset{Dx: 1, Dy: 1}), fieldTextOffsetToPosition(ts, toOffset))

	selection = controller.Selection()
	if selection.BaseOffset != 0 || selection.ExtentOffset != toOffset {
		t.Errorf("selection = %v, want 0 to %d", selection, toOffset)
	}

	// The scroll area of text field should not move.
	if got := scrollController.Offset(); got != beforeScrollOffset {
		t.Errorf("scroll offset = %v, want %v", got, beforeScrollOffset)
	}
}

func TestTextFieldAndroidCannotDragOneHandlePastTheOther(t *testing.T) {
	ts, controller := fieldHandleDragApp(t, "", 800, nil)
	const testValue = "abc def ghi"
	ts.EnterText(findField(), testValue)
	skipPastScrollingAnimation(ts)

	// Long press the 'e' to select 'def'.
	ePos := fieldTextOffsetToPosition(ts, 5) // Position before 'e'.
	g := ts.StartGesture(ePos, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump(2 * time.Second)
	g.Up()
	ts.Pump()
	ts.Pump(200 * time.Millisecond) // skip past the frame where the opacity is zero

	selection := controller.Selection()
	if selection.BaseOffset != 4 || selection.ExtentOffset != 7 {
		t.Fatalf("selection = %v, want 4 to 7", selection)
	}
	endpoints := fieldEndpoints(ts, controller)
	if len(endpoints) != 2 {
		t.Fatalf("endpoints = %d, want 2", len(endpoints))
	}

	// Drag the right handle until there's only 1 char selected. We use a small
	// offset because the endpoint is on the very corner of the handle.
	handlePos := endpoints[1].Point.Add(ui.Offset{Dx: 4})
	g = ts.StartGesture(handlePos, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(fieldTextOffsetToPosition(ts, 5)) // Position before 'e'.
	ts.Pump()
	if sel := controller.Selection(); sel.BaseOffset != 4 || sel.ExtentOffset != 5 {
		t.Fatalf("selection = %v, want 4 to 5", sel)
	}

	g.MoveTo(fieldTextOffsetToPosition(ts, 2)) // Position before 'c'.
	ts.Pump()
	g.Up()
	ts.Pump()

	// The selection doesn't move beyond the left handle. There's always at least 1
	// char selected.
	if sel := controller.Selection(); sel.BaseOffset != 4 || sel.ExtentOffset != 5 {
		t.Errorf("selection = %v, want 4 to 5", sel)
	}
}
