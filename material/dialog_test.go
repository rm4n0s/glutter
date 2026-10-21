// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// pumpDialogApp pumps an app whose body has a button that opens the dialog with
// show, and returns the channel that the dialog's result arrives on.
func pumpDialogApp(t *testing.T, theme *ThemeData, show func(bc widgets.BuildContext) <-chan any) (*gluttertest.Tester, *<-chan any) {
	t.Helper()
	result := new(<-chan any)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			*result = show(bc)
		}}))
	})})}))
	ts.PumpAndSettle()
	return ts, result
}

func showAlert(bc widgets.BuildContext, o ShowDialogOpts, d AlertDialogOpts) <-chan any {
	return ShowDialog[any](bc, func(bc widgets.BuildContext) widgets.Widget { return NewAlertDialog(d) }, o)
}

func alertOpts() AlertDialogOpts {
	return AlertDialogOpts{
		Title:   widgets.NewText("TITLE"),
		Content: widgets.NewText("CONTENT"),
		Actions: []widgets.Widget{
			NewTextButton(func() {}, widgets.NewText("CANCEL")),
			NewTextButton(func() {}, widgets.NewText("OK")),
		},
	}
}

func TestAlertDialogLayout(t *testing.T) {
	tests := []struct {
		name     string
		material bool
		radius   float64
		elev     float64
		actionsR float64 // distance of the last action's right edge from the dialog's
	}{
		{"material 3", true, 28, 6, 24},
		{"material 2", false, 4, 24, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, _ := pumpDialogApp(t, standardTheme(tt.material), func(bc widgets.BuildContext) <-chan any { return showAlert(bc, ShowDialogOpts{}, alertOpts()) })
			ts.Tap(gluttertest.ByText("OPEN"))
			ts.PumpAndSettle()
			dialog := dialogSurface()
			size := ts.Size(dialog)
			if size.Width < 280 {
				t.Errorf("dialog width %v is less than the minimum 280", size.Width)
			}
			topLeft := ts.TopLeft(dialog)
			if math.Abs(topLeft.Dx+size.Width/2-400) > 0.01 || math.Abs(topLeft.Dy+size.Height/2-300) > 0.01 {
				t.Errorf("dialog at %v, size %v is not centered", topLeft, size)
			}
			mat := ts.Widget(dialogSurface()).(*Material)
			if got := mat.Elevation(); got != tt.elev {
				t.Errorf("elevation = %v, want %v", got, tt.elev)
			}
			wantShape := painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(tt.radius))})
			if !foundation.ValuesEqual(mat.Shape(), painting.ShapeBorder(wantShape)) {
				t.Errorf("shape = %v, want %v", mat.Shape(), wantShape)
			}
			title := ts.TopLeft(gluttertest.ByText("TITLE"))
			if got := title.Dx - topLeft.Dx; got != 24 {
				t.Errorf("title is %v from the left edge, want 24", got)
			}
			if got := title.Dy - topLeft.Dy; got != 24 {
				t.Errorf("title is %v from the top edge, want 24", got)
			}
			ok := ts.BottomRight(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByText("OK")))
			if got := topLeft.Dx + size.Width - ok.Dx; got < tt.actionsR {
				t.Errorf("the last action ends %v from the right edge, want at least %v", got, tt.actionsR)
			}
		})
	}
}

func TestShowDialogResultAndBarrier(t *testing.T) {
	tests := []struct {
		name      string
		opts      ShowDialogOpts
		theme     *ThemeData
		act       func(ts *gluttertest.Tester)
		wantShown bool
		want      any
	}{
		{"pop with a value", ShowDialogOpts{}, standardTheme(true), func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("OK")) }, false, "ok"},
		{"tap the barrier", ShowDialogOpts{}, standardTheme(true), func(ts *gluttertest.Tester) { ts.TapAt(ui.Offset{Dx: 5, Dy: 5}) }, false, nil},
		{"barrier not dismissible", ShowDialogOpts{BarrierDismissible: foundation.Some(false)}, standardTheme(true), func(ts *gluttertest.Tester) { ts.TapAt(ui.Offset{Dx: 5, Dy: 5}) }, true, nil},
		{"tap inside the dialog", ShowDialogOpts{}, standardTheme(true), func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("CONTENT")) }, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, result := pumpDialogApp(t, tt.theme, func(bc widgets.BuildContext) <-chan any {
				return ShowDialog[any](bc, func(bc widgets.BuildContext) widgets.Widget {
					return NewAlertDialog(AlertDialogOpts{Content: widgets.NewText("CONTENT"), Actions: []widgets.Widget{
						NewTextButton(func() { widgets.NavigatorPop(bc, "ok") }, widgets.NewText("OK")),
					}})
				}, tt.opts)
			})
			ts.Tap(gluttertest.ByText("OPEN"))
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText("CONTENT")) {
				t.Fatal("the dialog is not shown")
			}
			tt.act(ts)
			ts.PumpAndSettle()
			if shown := ts.Exists(gluttertest.ByText("CONTENT")); shown != tt.wantShown {
				t.Errorf("dialog shown = %v, want %v", shown, tt.wantShown)
			}
			if tt.wantShown {
				return
			}
			select {
			case v := <-*result:
				if v != tt.want {
					t.Errorf("popped with %v, want %v", v, tt.want)
				}
			case <-time.After(time.Second):
				// The channel is filled by a goroutine of its own (convertChan).
				t.Error("the route did not complete")
			}
		})
	}
}

func TestShowDialogTypedResult(t *testing.T) {
	var result <-chan int
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			result = ShowDialog[int](bc, func(bc widgets.BuildContext) widgets.Widget {
				return NewSimpleDialog([]widgets.Widget{
					NewSimpleDialogOption(func() { widgets.NavigatorPop(bc, 1) }, widgets.NewText("ONE")),
					NewSimpleDialogOption(func() { widgets.NavigatorPop(bc, "two") }, widgets.NewText("TWO")),
				}, SimpleDialogOpts{Title: widgets.NewText("PICK")})
			})
		}}))
	})})}))
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	for _, text := range []string{"PICK", "ONE", "TWO"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Fatalf("%s is not shown", text)
		}
	}
	// The width is a multiple of 56, the options are 48 high and 24 from the edges.
	size := ts.Size(dialogSurface())
	if math.Mod(size.Width, 56) != 0 || size.Width < 280 {
		t.Errorf("simple dialog width %v is not a multiple of 56 of at least 280", size.Width)
	}
	if got := ts.Size(gluttertest.First(gluttertest.ByType[*SimpleDialogOption]())).Height; got < 36 {
		t.Errorf("option height = %v", got)
	}
	ts.Tap(gluttertest.ByText("ONE"))
	ts.PumpAndSettle()
	if v := <-result; v != 1 {
		t.Errorf("popped with %v, want 1", v)
	}
}

func TestShowDialogTypedResultOfAnotherTypeIsZero(t *testing.T) {
	var result <-chan int
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			result = ShowDialog[int](bc, func(bc widgets.BuildContext) widgets.Widget {
				return NewSimpleDialog([]widgets.Widget{NewSimpleDialogOption(func() { widgets.NavigatorPop(bc, "two") }, widgets.NewText("TWO"))})
			})
		}}))
	})})}))
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("TWO"))
	ts.PumpAndSettle()
	if v, ok := <-result; !ok || v != 0 {
		t.Errorf("popped with %v (open: %v), want 0", v, ok)
	}
}

// TestShowAdaptiveDialogBarrier opens the dialog without a tap: the fonts of the
// Apple and Windows typographies are not in the test environment, so a text is
// not hit on those platforms.
func TestShowAdaptiveDialogBarrier(t *testing.T) {
	tests := []struct {
		name      string
		platform  foundation.TargetPlatform
		wantShown bool
	}{
		{"android", foundation.TargetPlatformAndroid, false},
		{"linux", foundation.TargetPlatformLinux, false},
		{"iOS", foundation.TargetPlatformIOS, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(tt.platform)})
			var home widgets.BuildContext
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				home = bc
				return widgets.NewText("HOME")
			})}))
			ts.PumpAndSettle()
			ShowAdaptiveDialog[any](home, func(bc widgets.BuildContext) widgets.Widget {
				return NewAlertDialogAdaptive(AlertDialogOpts{Content: widgets.NewText("CONTENT")})
			})
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText("CONTENT")) {
				t.Fatalf("the dialog is not shown; errors: %v", ts.Errors())
			}
			ts.TapAt(ui.Offset{Dx: 5, Dy: 5})
			ts.PumpAndSettle()
			if shown := ts.Exists(gluttertest.ByText("CONTENT")); shown != tt.wantShown {
				t.Errorf("shown after a tap on the barrier = %v, want %v", shown, tt.wantShown)
			}
		})
	}
}

func TestDialogThemePrecedence(t *testing.T) {
	green, blue, red := ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0xFFFF0000)
	tests := []struct {
		name   string
		theme  DialogThemeData
		dialog DialogOpts
		wantBG ui.Color
		wantEl float64
	}{
		{"defaults", DialogThemeData{}, DialogOpts{}, standardColorScheme().SurfaceContainerHigh(), 6},
		{"theme", DialogThemeData{BackgroundColor: foundation.Some(green), Elevation: foundation.Some(2.0)}, DialogOpts{}, green, 2},
		{"widget over theme", DialogThemeData{BackgroundColor: foundation.Some(green), Elevation: foundation.Some(2.0)},
			DialogOpts{BackgroundColor: foundation.Some(blue), Elevation: foundation.Some(9.0)}, blue, 9},
		{"theme from the widget tree", DialogThemeData{BackgroundColor: foundation.Some(red)}, DialogOpts{}, red, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), NewDialogTheme(tt.theme, NewDialog(widgets.NewText("D"), tt.dialog)))
			mat := ts.Widget(dialogSurface()).(*Material)
			if got := mat.Color().Value(); got != tt.wantBG {
				t.Errorf("color = %v, want %v", got, tt.wantBG)
			}
			if got := mat.Elevation(); got != tt.wantEl {
				t.Errorf("elevation = %v, want %v", got, tt.wantEl)
			}
		})
	}
}

// dialogSurface finds the Material of the first dialog.
func dialogSurface() gluttertest.Finder {
	return gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*Dialog](), gluttertest.ByType[*Material]()))
}

func standardColorScheme() *ColorScheme { return standardTheme(true).ColorScheme }

func TestDialogFullscreen(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewDialogFullscreen(widgets.NewText("FULL"))}))
	if got := ts.Size(gluttertest.ByType[*Material]()); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("fullscreen dialog = %v, want 800x600", got)
	}
}

func TestAlertDialogScrollableAndActionsOverflow(t *testing.T) {
	long := make([]widgets.Widget, 40)
	for i := range long {
		long[i] = widgets.NewText("line")
	}
	tests := []struct {
		name string
		o    AlertDialogOpts
		fits bool
	}{
		{"scrollable content fits the screen", AlertDialogOpts{Title: widgets.NewText("T"), Content: widgets.NewColumn(long), Scrollable: true, Actions: []widgets.Widget{widgets.NewText("A")}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, _ := pumpDialogApp(t, standardTheme(true), func(bc widgets.BuildContext) <-chan any { return showAlert(bc, ShowDialogOpts{}, tt.o) })
			ts.Tap(gluttertest.ByText("OPEN"))
			ts.PumpAndSettle()
			if h := ts.Size(dialogSurface()).Height; h > 600-48 {
				t.Errorf("dialog height %v does not fit the screen", h)
			}
			if !ts.Exists(gluttertest.ByType[*widgets.SingleChildScrollView]()) {
				t.Error("no scroll view")
			}
		})
	}
}

func TestAlertDialogSemanticsRoles(t *testing.T) {
	ts, _ := pumpDialogApp(t, standardTheme(true), func(bc widgets.BuildContext) <-chan any { return showAlert(bc, ShowDialogOpts{}, alertOpts()) })
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	var roles []ui.SemanticsRole
	for _, w := range ts.Widgets(gluttertest.ByType[*widgets.Semantics]()) {
		if r, ok := w.(*widgets.Semantics).Properties().Role.Get(); ok {
			roles = append(roles, r)
		}
	}
	found := false
	for _, r := range roles {
		if r == ui.SemanticsRoleAlertDialog {
			found = true
		}
	}
	if !found {
		t.Errorf("no alert dialog role among %v", roles)
	}
}

func TestDialogThemeData(t *testing.T) {
	a := DialogThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(0.0), ClipBehavior: ui.ClipNone}
	b := DialogThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(10.0), ClipBehavior: ui.ClipHardEdge,
		InsetPadding: &painting.EdgeInsets{Left: 10, Right: 10}}
	tests := []struct {
		name string
		t    float64
		want DialogThemeData
	}{
		{"start", 0, DialogThemeData{BackgroundColor: a.BackgroundColor, Elevation: a.Elevation, ClipBehavior: ui.ClipNone, InsetPadding: &painting.EdgeInsets{Left: 10, Right: 10}}},
		{"half", 0.5, DialogThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFF808080)),
			Elevation: foundation.Some(5.0), ClipBehavior: ui.ClipHardEdge, InsetPadding: &painting.EdgeInsets{Left: 10, Right: 10}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DialogThemeDataLerp(a, b, tt.t)
			if got.Elevation != tt.want.Elevation || got.ClipBehavior != tt.want.ClipBehavior {
				t.Errorf("lerp = %v, want %v", got, tt.want)
			}
		})
	}
	if !a.Equal(a.CopyWith(DialogThemeDataCopy{})) {
		t.Error("an empty copy is not equal")
	}
	if a.Equal(a.CopyWith(DialogThemeDataCopy{Elevation: foundation.Some(1.0)})) {
		t.Error("a copy with another elevation is equal")
	}
	if got := NewThemeData().DialogTheme; !got.Equal(DialogThemeData{}) {
		t.Errorf("the default theme has %v", got)
	}
}

func TestDialogErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("negative elevation", func(t *testing.T) {
		err := recoverError(func() { NewDialog(nil, DialogOpts{Elevation: foundation.Some(-1.0)}) })
		wantRoute(t, err, "DialogElevationNegative")
	})
	t.Run("inactive context", func(t *testing.T) {
		var captured widgets.BuildContext
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			captured = bc
			return widgets.NewText("HOME")
		})}))
		ts.PumpWidget(widgets.NewSizedBox(nil))
		err := recoverError(func() {
			ShowDialog[any](captured, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("X") })
		})
		wantRoute(t, err, "ShowDialogContextInactive")
	})
}
