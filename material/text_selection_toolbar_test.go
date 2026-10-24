// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"fmt"
	"testing"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests port material/text_selection_toolbar_test.dart.

const toolbarTestContentDistance = 8.0

// toolbarTestBox is the TestBox of the Dart test: a SizedBox of its own type, so
// that find.byType(TestBox) finds only these.
//
// Flutter: material/text_selection_toolbar_test.dart TestBox
type toolbarTestBox struct {
	widgets.StatelessWidget
	child widgets.Widget
}

const (
	toolbarTestBoxHeight = 44.0
	toolbarTestBoxWidth  = 100.0
)

func newToolbarTestBox(opts ...widgets.WidgetOpts) *toolbarTestBox {
	return &toolbarTestBox{StatelessWidget: widgets.NewStatelessWidgetBase(opts...)}
}

func (w *toolbarTestBox) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewSizedBox(w.child, widgets.SizedBoxOpts{Width: foundation.Some(toolbarTestBoxWidth), Height: foundation.Some(toolbarTestBoxHeight)})
}

// newToolbarTester is a tester whose default font is Go Mono: the engine of a
// test has Roboto and Material Icons only and no font for a style that names no
// family, so the label of a toolbar button would have no width. Flutter's tests
// render with Ahem, whose glyphs are 1 em wide; Go Mono's are 0.6 em wide (SPEC
// §12.10). A font loaded under "sans-serif" is the default font.
//
// Unless the test chose a platform, the platform is Android, the default of
// flutter_test: the density of the buttons (48 for the standard density, 40 for
// the compact one of the desktop) changes how many fit in the toolbar.
func newToolbarTester(t *testing.T, opts ...gluttertest.Opts) *gluttertest.Tester {
	t.Helper()
	if foundation.DebugDefaultTargetPlatformOverride().IsNone() {
		withPlatform(t, foundation.TargetPlatformAndroid)
	}
	var o gluttertest.Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Fonts = []gluttertest.Font{{Family: "sans-serif", Data: gomono.TTF}}
	return gluttertest.New(t, o)
}

func toolbarTestBoxes(n int) []widgets.Widget {
	children := make([]widgets.Widget, n)
	for i := range children {
		children[i] = newToolbarTestBox()
	}
	return children
}

// findPrivate finds a widget of a private type by the name of its type, below
// the MaterialApp.
func findPrivate(typeName string) gluttertest.Finder {
	return gluttertest.Descendant(gluttertest.ByType[*MaterialApp](), gluttertest.ByPredicate("type "+typeName, func(w widgets.Widget) bool {
		return foundation.RuntimeTypeName(w) == typeName
	}))
}

// findToolbar finds the visible part of the toolbar for use in measurements:
// finding TextSelectionToolbar would not give the position as the user sees it.
func findToolbar() gluttertest.Finder { return findPrivate("textSelectionToolbarOverflowable") }

func findOverflowButton() gluttertest.Finder {
	return findPrivate("textSelectionToolbarOverflowButton")
}

func findToolbarContainer() gluttertest.Finder {
	return gluttertest.ByPredicate("type textSelectionToolbarContainer", func(w widgets.Widget) bool {
		return foundation.RuntimeTypeName(w) == "textSelectionToolbarContainer"
	})
}

func toolbarTestRect(ts *gluttertest.Tester, f gluttertest.Finder) ui.Rect {
	topLeft := ts.TopLeft(f)
	size := ts.Size(f)
	return ui.NewRectFromLTWH(topLeft.Dx, topLeft.Dy, size.Width, size.Height)
}

// pumpToolbarApp pumps a MaterialApp with the toolbar built by a StatefulBuilder
// and returns the setState function of it.
func pumpToolbarApp(ts *gluttertest.Tester, textDirection ui.TextDirection, build func() widgets.Widget) widgets.StateSetter {
	var setState widgets.StateSetter
	builder := widgets.Widget(widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setter widgets.StateSetter) widgets.Widget {
		setState = setter
		return build()
	}))
	if textDirection == ui.TextDirectionRTL {
		builder = widgets.NewDirectionality(ui.TextDirectionRTL, builder)
	}
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: builder})}))
	return setState
}

func TestTextSelectionToolbarPutsChildrenInAnOverflowMenuIfTheyOverflow(t *testing.T) {
	for _, dir := range []struct {
		name string
		ui.TextDirection
	}{{"puts children in an overflow menu if they overflow", ui.TextDirectionLTR},
		{"puts children in an overflow menu if they overflow in RTL", ui.TextDirectionRTL}} {
		t.Run(dir.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			children := toolbarTestBoxes(7)
			setState := pumpToolbarApp(ts, dir.TextDirection, func() widgets.Widget {
				return NewTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, children)
			})

			// All children fit on the screen, so they are all rendered.
			if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != len(children) {
				t.Fatalf("boxes = %d, want %d", got, len(children))
			}
			if ts.Exists(findOverflowButton()) {
				t.Fatal("there is an overflow button")
			}

			// Adding one more child makes the children overflow.
			setState(func() { children = append(children, newToolbarTestBox()) })
			ts.PumpAndSettle()
			if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != len(children)-1 {
				t.Fatalf("boxes = %d, want %d", got, len(children)-1)
			}
			if got := ts.Count(findOverflowButton()); got != 1 {
				t.Fatalf("overflow buttons = %d, want 1", got)
			}

			// Tap the overflow button to show the overflow menu.
			ts.Tap(findOverflowButton())
			ts.PumpAndSettle()
			if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != 1 {
				t.Fatalf("boxes in the overflow menu = %d, want 1", got)
			}
			if got := ts.Count(findOverflowButton()); got != 1 {
				t.Fatalf("overflow buttons = %d, want 1", got)
			}

			// Tap the overflow button again to hide the overflow menu.
			ts.Tap(findOverflowButton())
			ts.PumpAndSettle()
			if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != len(children)-1 {
				t.Fatalf("boxes = %d, want %d", got, len(children)-1)
			}
			if got := ts.Count(findOverflowButton()); got != 1 {
				t.Fatalf("overflow buttons = %d, want 1", got)
			}
		})
	}
}

func TestTextSelectionToolbarPositionsItselfAtAnchorAboveIfItFits(t *testing.T) {
	ts := newToolbarTester(t)
	const height = 44.0
	const anchorBelowY = 500.0
	anchorAboveY := 0.0

	setState := pumpToolbarApp(ts, ui.TextDirectionLTR, func() widgets.Widget {
		return NewTextSelectionToolbar(ui.Offset{Dx: 50, Dy: anchorAboveY}, ui.Offset{Dx: 50, Dy: anchorBelowY}, []widgets.Widget{
			widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(Colors.Red.Shade500()), Width: foundation.Some(50.0), Height: foundation.Some(height)}),
			widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(Colors.Green.Shade500()), Width: foundation.Some(50.0), Height: foundation.Some(height)}),
			widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(Colors.Blue.Shade500()), Width: foundation.Some(50.0), Height: foundation.Some(height)}),
		})
	})

	tests := []struct {
		name    string
		anchorY float64
		wantY   float64
	}{
		// When the toolbar doesn't fit above aboveAnchor, it positions itself
		// below belowAnchor.
		{"does not fit", 0, anchorBelowY + TextSelectionToolbarContentDistanceBelow},
		// Even when it barely doesn't fit.
		{"barely does not fit", 60, anchorBelowY + TextSelectionToolbarContentDistanceBelow},
		// When it does fit above aboveAnchor, it positions itself there.
		{"fits", 70, 70 - height - toolbarTestContentDistance},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setState(func() { anchorAboveY = tc.anchorY })
			ts.Pump()
			if got := ts.TopLeft(findToolbar()).Dy; got != tc.wantY {
				t.Errorf("toolbar y = %v, want %v", got, tc.wantY)
			}
		})
	}
}

// TestTextSelectionToolbarCustomToolbar ports 'can create and use a custom
// toolbar'. The Dart test overrides the deprecated buildToolbar of
// MaterialTextSelectionControls on a SelectableText; here the custom toolbar is
// the contextMenuBuilder of a SelectionArea, which is how a toolbar is
// customized since buildToolbar was deprecated.
func TestTextSelectionToolbarCustomToolbar(t *testing.T) {
	withPlatform(t, foundation.TargetPlatformAndroid)
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		NewSelectionArea(widgets.NewText("Select me custom menu"), SelectionAreaOpts{
			ContextMenuBuilder: func(_ widgets.BuildContext, state *widgets.SelectableRegionState) widgets.Widget {
				anchors := state.ContextMenuAnchors()
				return NewTextSelectionToolbar(anchors.PrimaryAnchor.Sub(ui.Offset{Dy: toolbarTestContentDistance}),
					anchors.SecondaryAnchor.OrElse(anchors.PrimaryAnchor), []widgets.Widget{
						NewTextSelectionToolbarTextButton(TextSelectionToolbarTextButtonGetPadding(0, 1), widgets.NewText("Custom button"),
							TextSelectionToolbarTextButtonOpts{OnPressed: func() {}}),
					})
			},
		}))})}))

	// The selection menu is not initially shown.
	if ts.Exists(gluttertest.ByText("Custom button")) {
		t.Fatal("the menu is shown at first")
	}

	// Long press on "custom" to select it.
	ts.LongPressAt(ts.TopLeft(gluttertest.ByType[*widgets.Text]()).Add(ui.Offset{Dx: 70, Dy: 7}))
	ts.Pump()

	// The custom selection menu is shown.
	if !ts.Exists(gluttertest.ByText("Custom button")) {
		t.Error("the custom menu is not shown")
	}
	for _, label := range []string{"Cut", "Copy", "Paste", "Select all"} {
		if ts.Exists(gluttertest.ByText(label)) {
			t.Errorf("the default %q button is shown", label)
		}
	}
}

func TestTextSelectionToolbarBackgroundColor(t *testing.T) {
	customBackgroundColor := Colors.Red.Shade500()
	schemes := []struct {
		name   string
		scheme func() *ColorScheme
		want   ui.Color
	}{
		{"light", func() *ColorScheme { return NewThemeData().ColorScheme }, ui.NewColor(0xffffffff)},
		{"dark", func() *ColorScheme { return NewThemeDataDark().ColorScheme }, ui.NewColor(0xff424242)},
	}
	for _, tc := range schemes {
		pump := func(t *testing.T, scheme *ColorScheme) *gluttertest.Tester {
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{ColorScheme: scheme}), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				NewTextSelectionToolbar(ui.OffsetZero, ui.OffsetZero, []widgets.Widget{
					NewTextSelectionToolbarTextButton(TextSelectionToolbarTextButtonGetPadding(0, 1), widgets.NewText("Custom button"),
						TextSelectionToolbarTextButtonOpts{OnPressed: func() {}}),
				}))})}))
			return ts
		}
		containerMaterial := gluttertest.First(gluttertest.Descendant(findToolbarContainer(), gluttertest.ByType[*Material]()))

		t.Run("default background color "+tc.name, func(t *testing.T) {
			ts := pump(t, tc.scheme())
			if !ts.Exists(containerMaterial) {
				t.Fatal("no Material in the toolbar container")
			}
			// The default colors are hardcoded and don't take the default value of
			// the theme's surface color.
			if got := ts.Widget(containerMaterial).(*Material).Color(); got != foundation.Some(tc.want) {
				t.Errorf("color = %v, want %v", got, tc.want)
			}
		})
		t.Run("custom background color "+tc.name, func(t *testing.T) {
			ts := pump(t, tc.scheme().CopyWith(ColorSchemeCopy{Surface: foundation.Some(customBackgroundColor)}))
			if !ts.Exists(containerMaterial) {
				t.Fatal("no Material in the toolbar container")
			}
			if got := ts.Widget(containerMaterial).(*Material).Color(); got != foundation.Some(customBackgroundColor) {
				t.Errorf("color = %v, want %v", got, customBackgroundColor)
			}
		})
	}
}

func TestTextSelectionToolbarOverflowedMenuExpandsChildrenHorizontally(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/144089.
	ts := newToolbarTester(t)
	children := toolbarTestBoxes(7)
	setState := pumpToolbarApp(ts, ui.TextDirectionLTR, func() widgets.Widget {
		return NewTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, children)
	})

	// All children fit on the screen, so they are all rendered.
	if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != len(children) {
		t.Fatalf("boxes = %d, want %d", got, len(children))
	}
	if ts.Exists(findOverflowButton()) {
		t.Fatal("there is an overflow button")
	}

	const (
		short  = "Short"
		medium = "Medium length"
		long   = "Long label in the overflow menu"
	)

	// Adding several children makes the menu overflow.
	setState(func() {
		// Flutter's test uses Ahem, whose glyphs are 14 wide: "Short" does not fit
		// after seven boxes. The texts are in Go Mono (8.4 wide), the default font
		// of the tester, to have it the same way.
		style := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "sans-serif", FontSize: foundation.Some(14.0)})
		children = append(children, widgets.NewText(short, widgets.TextOpts{Style: style}),
			widgets.NewText(medium, widgets.TextOpts{Style: style}), widgets.NewText(long, widgets.TextOpts{Style: style}))
	})
	ts.PumpAndSettle()
	if got := ts.Count(findOverflowButton()); got != 1 {
		t.Fatalf("overflow buttons = %d, want 1", got)
	}

	// Tap the overflow button to show the overflow menu.
	ts.Tap(findOverflowButton())
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByType[*toolbarTestBox]()) {
		t.Error("there are boxes in the overflow menu")
	}
	if got := ts.Count(gluttertest.ByType[*widgets.Text]()); got != 3 {
		t.Errorf("texts = %d, want 3", got)
	}
	if got := ts.Count(findOverflowButton()); got != 1 {
		t.Errorf("overflow buttons = %d, want 1", got)
	}
	if !ts.Exists(findToolbarContainer()) {
		t.Fatal("no toolbar container")
	}

	// Buttons have their width set to the container width.
	overflowMenuWidth := ts.Size(findToolbarContainer()).Width
	for _, label := range []string{long, medium, short} {
		if got := ts.Size(gluttertest.ByText(label)).Width; got != overflowMenuWidth {
			t.Errorf("width of %q = %v, want %v", label, got, overflowMenuWidth)
		}
	}
}

func TestTextSelectionToolbarItemsAreOrderedRightToLeftInRTL(t *testing.T) {
	const itemCount = 3
	ts := newToolbarTester(t)
	children := make([]widgets.Widget, itemCount)
	for i := range children {
		box := newToolbarTestBox(widgets.WidgetOpts{Key: foundation.NewValueKey(fmt.Sprintf("item_%d", i))})
		box.child = widgets.NewText(fmt.Sprint(i))
		children[i] = box
	}
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewDirectionality(ui.TextDirectionRTL,
		NewTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, children))})}))

	// Verify all items are visible.
	if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != itemCount {
		t.Fatalf("boxes = %d, want %d", got, itemCount)
	}

	// Find all text widgets by their content and get their positions.
	textRects := make([]ui.Rect, itemCount)
	for i := range textRects {
		textRects[i] = toolbarTestRect(ts, gluttertest.ByText(fmt.Sprint(i)))
	}

	// In RTL, items should be in reverse order (2, 1, 0). So item 2 should be
	// leftmost, then 1, then 0: each item is to the left of the previous one.
	for i := 0; i < itemCount-1; i++ {
		current, next := textRects[i], textRects[i+1]
		if next.Right > current.Left {
			t.Errorf("In RTL, item %d should be to the left of item %d: %v, %v", i+1, i, next, current)
		}
	}
}

func TestTextSelectionToolbarNeedsAChild(t *testing.T) {
	skipUnlessDebug(t)
	wantRoute(t, recoverError(func() { NewTextSelectionToolbar(ui.Offset{}, ui.Offset{}, nil) }), "TextSelectionToolbarNoChildren")
}

func TestTextSelectionToolbarLayout(t *testing.T) {
	tests := []struct {
		name       string
		size       ui.Size
		padding    ui.ViewPadding
		anchors    [2]ui.Offset
		wantAbove  bool
		wantTopLtr float64
	}{
		{"above the anchor", ui.Size{Width: 800, Height: 600}, ui.ViewPadding{}, [2]ui.Offset{{Dx: 400, Dy: 300}, {Dx: 400, Dy: 320}}, true, 300 - 8 - 44},
		{"below when there is no room above", ui.Size{Width: 800, Height: 600}, ui.ViewPadding{}, [2]ui.Offset{{Dx: 400, Dy: 40}, {Dx: 400, Dy: 60}},
			false, 60 + TextSelectionToolbarContentDistanceBelow},
		{"below when the status bar takes the room", ui.Size{Width: 800, Height: 600}, ui.ViewPadding{Top: 60}, [2]ui.Offset{{Dx: 400, Dy: 100}, {Dx: 400, Dy: 120}},
			false, 120 + TextSelectionToolbarContentDistanceBelow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t, gluttertest.Opts{Size: tc.size, ViewPadding: tc.padding})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextSelectionToolbar(tc.anchors[0], tc.anchors[1], toolbarTestBoxes(2))})}))
			got := toolbarTestRect(ts, findToolbar())
			if got.Top != tc.wantTopLtr {
				t.Errorf("top = %v, want %v", got.Top, tc.wantTopLtr)
			}
			if wantLeft := 400 - 100.0; got.Left != wantLeft {
				t.Errorf("left = %v, want %v (centered on the anchor)", got.Left, wantLeft)
			}
			if got.Height() != 44 {
				t.Errorf("height = %v, want 44", got.Height())
			}
		})
	}
}

func TestTextSelectionToolbarOverflowMenuOpensAboveOrBelow(t *testing.T) {
	// When the toolbar is above its anchor, the overflow button goes to the bottom
	// of the open menu, and the menu grows upwards from the anchor; below its
	// anchor the button is the first item.
	tests := []struct {
		name       string
		anchors    [2]ui.Offset
		buttonLast bool
	}{
		{"above", [2]ui.Offset{{Dx: 400, Dy: 300}, {Dx: 400, Dy: 320}}, true},
		{"below", [2]ui.Offset{{Dx: 400, Dy: 40}, {Dx: 400, Dy: 60}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextSelectionToolbar(tc.anchors[0], tc.anchors[1], toolbarTestBoxes(9))})}))
			ts.Tap(findOverflowButton())
			ts.PumpAndSettle()
			boxes := ts.Widgets(gluttertest.ByType[*toolbarTestBox]())
			if len(boxes) != 2 {
				t.Fatalf("boxes in the overflow menu = %d, want 2", len(boxes))
			}
			button := toolbarTestRect(ts, findOverflowButton())
			box := toolbarTestRect(ts, gluttertest.First(gluttertest.ByType[*toolbarTestBox]()))
			if tc.buttonLast && !(button.Top > box.Top) {
				t.Errorf("the overflow button (%v) should be below the items (%v)", button, box)
			}
			if !tc.buttonLast && !(button.Top < box.Top) {
				t.Errorf("the overflow button (%v) should be above the items (%v)", button, box)
			}
			// Each item of the open menu fills its width.
			if got := ts.Size(gluttertest.First(gluttertest.ByType[*toolbarTestBox]())).Width; got != toolbarTestBoxWidth {
				t.Errorf("item width = %v", got)
			}
		})
	}
}

func TestTextSelectionToolbarOverflowButtonTooltips(t *testing.T) {
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextSelectionToolbar(ui.Offset{Dx: 400, Dy: 300}, ui.Offset{Dx: 400, Dy: 320}, toolbarTestBoxes(9))})}))
	tests := []struct {
		state   string
		tooltip string
		icon    widgets.IconData
		key     foundation.Key
	}{
		{"closed", "More", Icons.MoreVert(), widgets.StandardComponentTypeMoreButton.Key()},
		{"open", "Back", Icons.ArrowBack(), widgets.StandardComponentTypeBackButton.Key()},
	}
	for _, tc := range tests {
		if !ts.Exists(gluttertest.ByTooltip(tc.tooltip)) {
			t.Errorf("%s: no button with the tooltip %q", tc.state, tc.tooltip)
		}
		if !ts.Exists(gluttertest.ByIcon(tc.icon)) {
			t.Errorf("%s: no icon %v", tc.state, tc.icon)
		}
		if got := ts.Widget(findOverflowButton()).Key(); got != tc.key {
			t.Errorf("%s: key = %v, want %v", tc.state, got, tc.key)
		}
		ts.Tap(findOverflowButton())
		ts.PumpAndSettle()
	}
}

func TestTextSelectionToolbarOnstageChildren(t *testing.T) {
	// The children that are not painted are not on stage: the finders skip them
	// (the debugVisitOnstageChildren of the element), unless they ask for the
	// offstage ones.
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextSelectionToolbar(ui.Offset{Dx: 400, Dy: 300}, ui.Offset{Dx: 400, Dy: 320}, toolbarTestBoxes(9))})}))
	// 48 for the overflow button and 7 boxes of 100 fit in the 784 available.
	if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != 7 {
		t.Errorf("on stage = %d, want 7", got)
	}
	if got := ts.Count(gluttertest.ByType[*toolbarTestBox](gluttertest.FindOpts{IncludeOffstage: true})); got != 9 {
		t.Errorf("all = %d, want 9", got)
	}
	// An item that is not painted is not hit either.
	ts.TapAt(ts.Center(findToolbar()))
	ts.Pump()
	if got := ts.Count(gluttertest.ByType[*toolbarTestBox]()); got != 7 {
		t.Errorf("on stage after the tap = %d, want 7", got)
	}
}
