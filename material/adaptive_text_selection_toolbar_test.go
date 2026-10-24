// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests port material/adaptive_text_selection_toolbar_test.dart. Dart's
// TargetPlatformVariant.all() runs a test once for each platform; here every
// platform is a subtest.
//
// Deviation: there is no Cupertino library, so iOS and macOS get the toolbar
// and the buttons of the desktop (SPEC §12.10); where the Dart test expects a
// Cupertino toolbar, these tests expect the desktop one.

var allTargetPlatforms = []foundation.TargetPlatform{
	foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia,
	foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows,
}

// forAllPlatforms runs the test for each target platform.
func forAllPlatforms(t *testing.T, test func(t *testing.T, platform foundation.TargetPlatform)) {
	t.Helper()
	for _, platform := range allTargetPlatforms {
		t.Run(platform.Name(), func(t *testing.T) {
			withPlatform(t, platform)
			test(t, platform)
		})
	}
}

// usesMaterialToolbar is whether the platform has the Material toolbar.
func usesMaterialToolbar(platform foundation.TargetPlatform) bool {
	return platform == foundation.TargetPlatformAndroid
}

// usesMaterialButtons is whether the buttons are TextSelectionToolbarTextButtons.
func usesMaterialButtons(platform foundation.TargetPlatform) bool {
	return platform == foundation.TargetPlatformAndroid || platform == foundation.TargetPlatformFuchsia
}

func expectCount(t *testing.T, ts *gluttertest.Tester, f gluttertest.Finder, want int, what string) {
	t.Helper()
	if got := ts.Count(f); got != want {
		t.Errorf("%s = %d, want %d", what, got, want)
	}
}

func TestAdaptiveTextSelectionToolbarBuildsTheRightToolbarOnEachPlatform(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		const buttonText = "Click me"
		ts := newToolbarTester(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewAdaptiveTextSelectionToolbarButtonItems([]widgets.ContextMenuButtonItem{{Label: foundation.Some(buttonText), OnPressed: func() {}}},
				widgets.TextSelectionToolbarAnchors{}))})}))

		expectCount(t, ts, gluttertest.ByText(buttonText), 1, "buttons")
		material, desktop := 0, 1
		if usesMaterialToolbar(platform) {
			material, desktop = 1, 0
		}
		expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbar](), material, "TextSelectionToolbars")
		expectCount(t, ts, gluttertest.ByType[*DesktopTextSelectionToolbar](), desktop, "DesktopTextSelectionToolbars")
	})
}

func TestAdaptiveTextSelectionToolbarCanBuildChildrenDirectly(t *testing.T) {
	key := foundation.NewKey("child")
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		NewAdaptiveTextSelectionToolbar(widgets.TextSelectionToolbarAnchors{}, []widgets.Widget{widgets.NewContainer(nil, widgets.ContainerOpts{Key: key})}))})}))
	expectCount(t, ts, gluttertest.ByKey(key), 1, "children")
}

func TestAdaptiveTextSelectionToolbarCanBuildFromEditableTextState(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		key := foundation.NewKey("toolbar")
		controller := widgets.NewTextEditingController()
		focusNode := widgets.NewFocusNode()
		t.Cleanup(func() { focusNode.Dispose(); controller.Dispose() })
		ts := newToolbarTester(t)
		// Fill the clipboard so that the Paste option is available in the text
		// selection menu.
		ts.SetClipboardText("Clipboard data")
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewSizedBox(
			widgets.NewEditableText(controller, focusNode, painting.NewTextStyle(), ui.NewColor(0xff00ffff), ui.NewColor(0xff00ffff), widgets.EditableTextOpts{
				SelectionControls: MaterialTextSelectionHandleControls,
				ContextMenuBuilder: func(_ widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
					return NewAdaptiveTextSelectionToolbarEditableText(state, widgets.WidgetOpts{Key: key})
				},
			}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))})}))
		ts.Pump() // Wait for autofocus to take effect.

		expectCount(t, ts, gluttertest.ByKey(key), 0, "toolbars at first")

		// Long-press to bring up the context menu.
		editable := gluttertest.ByType[*widgets.EditableText]()
		ts.LongPress(editable)
		gluttertest.StateOf[*widgets.EditableTextState](ts, editable).ShowToolbar()
		ts.PumpAndSettle()

		expectCount(t, ts, gluttertest.ByKey(key), 1, "toolbars")
		for _, label := range []string{"Copy", "Cut", "Select all"} {
			expectCount(t, ts, gluttertest.ByText(label), 0, label)
		}
		expectCount(t, ts, gluttertest.ByText("Paste"), 1, "Paste")

		if usesMaterialButtons(platform) {
			expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbarTextButton](), 1, "TextSelectionToolbarTextButtons")
		} else {
			expectCount(t, ts, gluttertest.ByType[*DesktopTextSelectionToolbarButton](), 1, "DesktopTextSelectionToolbarButtons")
		}
	})
}

func TestAdaptiveTextSelectionToolbarCanBuildForEditableTextFromRawParameters(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		key := foundation.NewKey("toolbar")
		// Flutter's test view is 800 wide and its text is Ahem; the text of the
		// tester is Go Mono, which is narrower, so the view is 500 wide to have the
		// same two buttons in the overflow menu (SPEC §12.10).
		ts := newToolbarTester(t, gluttertest.Opts{Size: ui.Size{Width: 500, Height: 600}})
		noop := func() {}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewAdaptiveTextSelectionToolbarEditable(widgets.ClipboardStatusPasteable, widgets.EditableButtonItemsCallbacks{
				OnCopy: noop, OnCut: noop, OnPaste: noop, OnSelectAll: noop, OnLiveTextInput: noop, OnLookUp: noop, OnSearchWeb: noop, OnShare: noop,
			}, widgets.TextSelectionToolbarAnchors{}, widgets.WidgetOpts{Key: key}))})}))

		expectCount(t, ts, gluttertest.ByKey(key), 1, "toolbars")

		switch {
		case platform == foundation.TargetPlatformAndroid:
			expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbarTextButton](), 6, "TextSelectionToolbarTextButtons")
			for _, label := range []string{"Cut", "Copy", "Paste", "Share", "Select all", "Look Up"} {
				expectCount(t, ts, gluttertest.ByText(label), 1, label)
			}
			// Material overflow buttons are not TextSelectionToolbarTextButtons.
			expectCount(t, ts, gluttertest.ByKey(widgets.StandardComponentTypeMoreButton.Key()), 1, "next buttons")

			ts.TapAt(ts.Center(gluttertest.ByKey(widgets.StandardComponentTypeMoreButton.Key())))
			ts.PumpAndSettle()

			expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbarTextButton](), 2, "TextSelectionToolbarTextButtons in the overflow menu")
			expectCount(t, ts, gluttertest.ByText("Search Web"), 1, "Search Web")
			expectCount(t, ts, gluttertest.ByText("Scan text"), 1, "Scan text")
			expectCount(t, ts, gluttertest.ByKey(widgets.StandardComponentTypeBackButton.Key()), 1, "back buttons")
		case usesMaterialButtons(platform): // Fuchsia: the buttons of Material in the toolbar of the desktop.
			expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbarTextButton](), 8, "TextSelectionToolbarTextButtons")
			for _, label := range []string{"Cut", "Copy", "Paste", "Select all", "Look Up", "Search Web", "Share"} {
				expectCount(t, ts, gluttertest.ByText(label), 1, label)
			}
		default: // Linux and Windows, and iOS and macOS (see the deviation above).
			expectCount(t, ts, gluttertest.ByType[*DesktopTextSelectionToolbarButton](), 8, "DesktopTextSelectionToolbarButtons")
			for _, label := range []string{"Cut", "Copy", "Paste", "Select all", "Look Up", "Search Web", "Share", "Scan text"} {
				expectCount(t, ts, gluttertest.ByText(label), 1, label)
			}
		}
	})
}

func TestAdaptiveTextSelectionToolbarBuildsAnEmptyToolbarWhenThereAreNoButtons(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		ts := newToolbarTester(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewCenter(
			NewAdaptiveTextSelectionToolbar(widgets.TextSelectionToolbarAnchors{}, nil))}))
		if got := ts.Size(gluttertest.ByType[*AdaptiveTextSelectionToolbar]()); got != ui.SizeZero {
			t.Errorf("size = %v, want zero", got)
		}
		if err := ts.TakeError(); err != nil {
			t.Errorf("exception: %v", err)
		}
	})
}

func TestAdaptiveTextSelectionToolbarGetEditableTextButtonItemsPerPlatform(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		// Fill the clipboard so that the Paste option is available in the text
		// selection menu.
		var buttonTypes map[widgets.ContextMenuButtonType]bool
		controller := widgets.NewTextEditingController()
		focusNode := widgets.NewFocusNode()
		t.Cleanup(func() { focusNode.Dispose(); controller.Dispose() })
		ts := newToolbarTester(t)
		ts.SetClipboardText("Clipboard data")
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			widgets.NewEditableText(controller, focusNode, painting.NewTextStyle(), Colors.Red.Shade500(), Colors.Grey.Shade500(), widgets.EditableTextOpts{
				SelectionControls: MaterialTextSelectionHandleControls,
				ContextMenuBuilder: func(_ widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
					buttonTypes = map[widgets.ContextMenuButtonType]bool{}
					for _, item := range state.ContextMenuButtonItems() {
						buttonTypes[item.Type] = true
					}
					return widgets.NewSizedBoxShrink(nil)
				},
			}))})}))

		state := gluttertest.StateOf[*widgets.EditableTextState](ts, gluttertest.ByType[*widgets.EditableText]())

		// With no text in the field.
		ts.TapAt(ts.TopLeft(gluttertest.ByType[*widgets.EditableText]()).Add(ui.Offset{Dx: 1, Dy: 1}))
		ts.Pump()
		state.ShowToolbar()
		ts.Pump()

		for _, tc := range []struct {
			t    widgets.ContextMenuButtonType
			want bool
		}{{widgets.ContextMenuButtonTypeCut, false}, {widgets.ContextMenuButtonTypeCopy, false},
			{widgets.ContextMenuButtonTypePaste, true}, {widgets.ContextMenuButtonTypeSelectAll, false}} {
			if buttonTypes[tc.t] != tc.want {
				t.Errorf("no text: %s present = %v, want %v", tc.t.Name(), buttonTypes[tc.t], tc.want)
			}
		}

		// With text but no selection.
		const text = "lorem ipsum"
		controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text)))}))
		ts.Pump()

		for _, tc := range []struct {
			t    widgets.ContextMenuButtonType
			want bool
		}{{widgets.ContextMenuButtonTypeCut, false}, {widgets.ContextMenuButtonTypeCopy, false}, {widgets.ContextMenuButtonTypePaste, true},
			{widgets.ContextMenuButtonTypeSelectAll, platform != foundation.TargetPlatformMacOS}} {
			if buttonTypes[tc.t] != tc.want {
				t.Errorf("text, no selection: %s present = %v, want %v", tc.t.Name(), buttonTypes[tc.t], tc.want)
			}
		}

		// With text and selection.
		controller.SetSelection(services.NewTextSelection(0, len("lorem")))
		ts.Pump()

		for _, tc := range []struct {
			t    widgets.ContextMenuButtonType
			want bool
		}{{widgets.ContextMenuButtonTypeCut, true}, {widgets.ContextMenuButtonTypeCopy, true}, {widgets.ContextMenuButtonTypePaste, true},
			{widgets.ContextMenuButtonTypeSelectAll, platform != foundation.TargetPlatformIOS && platform != foundation.TargetPlatformMacOS}} {
			if buttonTypes[tc.t] != tc.want {
				t.Errorf("selection: %s present = %v, want %v", tc.t.Name(), buttonTypes[tc.t], tc.want)
			}
		}
	})
}

func TestAdaptiveTextSelectionToolbarGetAdaptiveButtonsPerPlatform(t *testing.T) {
	forAllPlatforms(t, func(t *testing.T, platform foundation.TargetPlatform) {
		const buttonText = "Click me"
		ts := newToolbarTester(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			buttonItems := []widgets.ContextMenuButtonItem{{Label: foundation.Some(buttonText), OnPressed: func() {}}}
			return widgets.NewListView(AdaptiveTextSelectionToolbarGetAdaptiveButtons(bc, buttonItems))
		}))})}))

		expectCount(t, ts, gluttertest.ByText(buttonText), 1, "buttons")
		material, desktop := 0, 1
		if usesMaterialButtons(platform) {
			material, desktop = 1, 0
		}
		expectCount(t, ts, gluttertest.ByType[*TextSelectionToolbarTextButton](), material, "TextSelectionToolbarTextButtons")
		expectCount(t, ts, gluttertest.ByType[*DesktopTextSelectionToolbarButton](), desktop, "DesktopTextSelectionToolbarButtons")
	})
}

func TestAdaptiveTextSelectionToolbarAndroidAnchors(t *testing.T) {
	// The Material toolbar goes above the primary anchor, and below the secondary
	// one when there is no room above; with no secondary anchor it uses the
	// primary one for both.
	tests := []struct {
		name    string
		anchors widgets.TextSelectionToolbarAnchors
		wantTop float64
	}{
		{"above the primary anchor", widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 400, Dy: 300}, SecondaryAnchor: foundation.Some(ui.Offset{Dx: 400, Dy: 320})},
			300 - 8 - 44},
		{"below the secondary anchor", widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 400, Dy: 40}, SecondaryAnchor: foundation.Some(ui.Offset{Dx: 400, Dy: 60})},
			60 + TextSelectionToolbarContentDistanceBelow},
		{"below the primary anchor without a secondary one", widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 400, Dy: 40}},
			40 + TextSelectionToolbarContentDistanceBelow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withPlatform(t, foundation.TargetPlatformAndroid)
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewAdaptiveTextSelectionToolbarButtonItems(
				[]widgets.ContextMenuButtonItem{{Label: foundation.Some("One"), OnPressed: func() {}}, {Label: foundation.Some("Two"), OnPressed: func() {}}}, tc.anchors)})}))
			if got := ts.TopLeft(findToolbar()).Dy; got != tc.wantTop {
				t.Errorf("top = %v, want %v", got, tc.wantTop)
			}
		})
	}
}

func TestAdaptiveTextSelectionToolbarButtonPositionsAndTaps(t *testing.T) {
	withPlatform(t, foundation.TargetPlatformAndroid)
	var pressed []string
	press := func(s string) func() { return func() { pressed = append(pressed, s) } }
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewAdaptiveTextSelectionToolbarButtonItems(
		[]widgets.ContextMenuButtonItem{
			{Label: foundation.Some("One"), OnPressed: press("one")}, {Label: foundation.Some("Two"), OnPressed: press("two")}, {Label: foundation.Some("Three"), OnPressed: press("three")},
		}, widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 400, Dy: 300}})})}))
	buttons := ts.Widgets(gluttertest.ByType[*TextSelectionToolbarTextButton]())
	if len(buttons) != 3 {
		t.Fatalf("buttons = %d, want 3", len(buttons))
	}
	for i, w := range buttons {
		want := TextSelectionToolbarTextButtonGetPadding(i, 3)
		if got := w.(*TextSelectionToolbarTextButton).Padding(); got != want {
			t.Errorf("padding of button %d = %v, want %v", i, got, want)
		}
		if got := w.(*TextSelectionToolbarTextButton).Alignment(); got != painting.AlignmentGeometry(painting.AlignmentDirectionalCenterStart) {
			t.Errorf("alignment of button %d = %v", i, got)
		}
	}
	ts.Tap(gluttertest.ByText("Two"))
	ts.Pump()
	if len(pressed) != 1 || pressed[0] != "two" {
		t.Errorf("pressed = %q, want [two]", pressed)
	}
}
