package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/material/magnifier_test.dart.
// flutter_test runs on Android, so the tests that do not name a platform run on
// it. The CupertinoTextMagnifier of iOS is not ported (SPEC §12.10): the case of
// iOS checks that nothing is built there.

var (
	magnifierReasonableTextField = ui.NewRectFromLTRB(50, 100, 200, 100)
	magnifierBasicOffset         = ui.Offset{
		Dx: MagnifierDefaultMagnifierSize.Width / 2,
		Dy: MagnifierStandardVerticalFocalPointShift + MagnifierDefaultMagnifierSize.Height,
	}
)

// magnifierFixture is the magnifier controller of a test and the context it shows
// the magnifier in (the Placeholder of the app).
type magnifierFixture struct {
	t          *testing.T
	ts         *gluttertest.Tester
	controller *widgets.MagnifierController
	bc         widgets.BuildContext
}

// newMagnifierFixture pumps a MaterialApp with the home and gives the context of
// the home.
func newMagnifierFixture(t *testing.T, home widgets.Widget) *magnifierFixture {
	t.Helper()
	withPlatform(t, foundation.TargetPlatformAndroid)
	f := &magnifierFixture{t: t, ts: gluttertest.New(t), controller: widgets.NewMagnifierController()}
	t.Cleanup(f.controller.RemoveFromOverlay)
	f.pumpApp(home)
	return f
}

func (f *magnifierFixture) pumpApp(home widgets.Widget) {
	f.ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		f.bc = bc
		return home
	})}))
}

func (f *magnifierFixture) show(info *foundation.ValueNotifier[widgets.MagnifierInfo]) {
	f.t.Helper()
	shown := f.controller.Show(f.bc, func(widgets.BuildContext) widgets.Widget { return NewTextMagnifier(info) })
	f.ts.PumpAndSettle()
	// Verify that the magnifier is shown.
	if completed, ended := shown.Completed(); !completed || !ended {
		f.t.Fatalf("the magnifier is not fully shown: completed %v, ended %v", completed, ended)
	}
}

// position is the position of the magnifier: of its Positioned, or of its
// AnimatedPositioned.
func (f *magnifierFixture) position(animated bool) ui.Offset {
	f.t.Helper()
	if animated {
		a := f.ts.Widgets(gluttertest.ByType[*widgets.AnimatedPositioned]())[0].(*widgets.AnimatedPositioned)
		return ui.Offset{Dx: a.Left().OrElse(0), Dy: a.Top().OrElse(0)}
	}
	p := f.ts.Widgets(gluttertest.ByType[*widgets.Positioned]())[0].(*widgets.Positioned)
	return ui.Offset{Dx: p.Left().OrElse(0), Dy: p.Top().OrElse(0)}
}

func (f *magnifierFixture) isAnimated() bool {
	f.t.Helper()
	a := f.ts.Widgets(gluttertest.ByType[*widgets.AnimatedPositioned]())[0].(*widgets.AnimatedPositioned)
	return a.Duration() != 0
}

func (f *magnifierFixture) additionalFocalPoint() ui.Offset {
	f.t.Helper()
	return f.ts.Widgets(gluttertest.ByType[*Magnifier]())[0].(*Magnifier).AdditionalFocalPointOffset()
}

func magnifierInfoOf(t *testing.T, info widgets.MagnifierInfo) *foundation.ValueNotifier[widgets.MagnifierInfo] {
	t.Helper()
	n := foundation.NewValueNotifier(info)
	t.Cleanup(n.Dispose)
	return n
}

func sameLineInfo(line, field, caret ui.Rect, gesture ui.Offset) widgets.MagnifierInfo {
	return widgets.MagnifierInfo{CurrentLineBoundaries: line, FieldBounds: field, CaretRect: caret, GlobalGesturePosition: gesture}
}

func TestAdaptiveMagnifierControllerBuilder(t *testing.T) {
	tests := []struct {
		name     string
		platform foundation.TargetPlatform
		// want is true when the builder builds a TextMagnifier.
		want bool
	}{
		{"should return a TextEditingMagnifier on Android", foundation.TargetPlatformAndroid, true},
		// Flutter builds a CupertinoTextMagnifier on iOS, which is not ported.
		{"builds nothing on iOS (CupertinoTextMagnifier is not ported)", foundation.TargetPlatformIOS, false},
		{"should return null on Fuchsia", foundation.TargetPlatformFuchsia, false},
		{"should return null on Linux", foundation.TargetPlatformLinux, false},
		{"should return null on macOS", foundation.TargetPlatformMacOS, false},
		{"should return null on Windows", foundation.TargetPlatformWindows, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withPlatform(t, tt.platform)
			ts := gluttertest.New(t)
			var bc widgets.BuildContext
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(b widgets.BuildContext) widgets.Widget {
				bc = b
				return widgets.NewPlaceholder(nil)
			})}))

			magnifierPositioner := magnifierInfoOf(t, widgets.MagnifierInfoEmpty)
			builtWidget := TextMagnifierAdaptiveMagnifierConfiguration().MagnifierBuilder()(bc, widgets.NewMagnifierController(), magnifierPositioner)

			if _, isTextMagnifier := builtWidget.(*TextMagnifier); isTextMagnifier != tt.want {
				t.Errorf("built %T, want a TextMagnifier: %v", builtWidget, tt.want)
			}
			if !tt.want && builtWidget != nil {
				t.Errorf("built %T, want nothing", builtWidget)
			}
		})
	}

	t.Run("only iOS shows the handles in the magnifier, and the value is the same each time", func(t *testing.T) {
		for _, tt := range []struct {
			platform foundation.TargetPlatform
			handles  bool
		}{
			{foundation.TargetPlatformIOS, true}, {foundation.TargetPlatformAndroid, false}, {foundation.TargetPlatformLinux, false},
		} {
			withPlatform(t, tt.platform)
			first, second := TextMagnifierAdaptiveMagnifierConfiguration(), TextMagnifierAdaptiveMagnifierConfiguration()
			if first != second {
				t.Errorf("%v: two reads give two configurations", tt.platform)
			}
			if first.ShouldDisplayHandlesInMagnifier() != tt.handles {
				t.Errorf("%v: ShouldDisplayHandlesInMagnifier = %v, want %v", tt.platform, first.ShouldDisplayHandlesInMagnifier(), tt.handles)
			}
		}
	})
}

func TestMagnifierPosition(t *testing.T) {
	t.Run("should be at gesture position if does not violate any positioning rules", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		textField := foundation.NewUniqueKey()

		f.ts.PumpWidget(widgets.NewColoredBox(ui.NewColorFromARGB(255, 0, 255, 179),
			NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				f.bc = bc
				return widgets.NewCenter(widgets.NewContainer(widgets.NewPlaceholder(nil), widgets.ContainerOpts{
					Key: textField, Width: foundation.Some(10.0), Height: foundation.Some(10.0), Color: foundation.Some(ui.NewColor(0xFFF44336)), // Colors.red
				}))
			})})))

		// Magnifier should be positioned directly over the red square.
		tapPointRenderBox := f.ts.RenderObject(gluttertest.ByKey(textField)).(rendering.Box)
		fakeTextFieldRect := tapPointRenderBox.LocalToGlobal(ui.Offset{}, nil).And(tapPointRenderBox.Size())

		magnifierInfo := magnifierInfoOf(t, sameLineInfo(fakeTextFieldRect, fakeTextFieldRect, fakeTextFieldRect, fakeTextFieldRect.Center()))
		f.show(magnifierInfo)

		// Should show two red crossed-out squares: the original in the center,
		// and one in the magnifier, in the upper half of the image, surrounded
		// by a faint offset rounded rectangle shadow.
		f.ts.MatchesGolden("magnifier.position.default")
	})

	t.Run("should never move outside the right bounds of the editing line", func(t *testing.T) {
		const gestureOutsideLine = 100.0
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(
			magnifierReasonableTextField,
			// Inflate these two to make sure we're bounding on the
			// current line boundaries, not anything else.
			magnifierReasonableTextField.Inflate(gestureOutsideLine),
			magnifierReasonableTextField.Inflate(gestureOutsideLine),
			// The tap position is far out of the right side of the app.
			ui.Offset{Dx: magnifierReasonableTextField.Right + gestureOutsideLine})))

		// Should be less than the right edge, since we have padding.
		if got := f.position(false).Dx; got > magnifierReasonableTextField.Right {
			t.Errorf("position.dx = %v, want at most %v", got, magnifierReasonableTextField.Right)
		}
	})

	t.Run("should never move outside the left bounds of the editing line", func(t *testing.T) {
		const gestureOutsideLine = 100.0
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(
			magnifierReasonableTextField,
			magnifierReasonableTextField.Inflate(gestureOutsideLine),
			magnifierReasonableTextField.Inflate(gestureOutsideLine),
			// The tap position is far out of the left side of the app.
			ui.Offset{Dx: magnifierReasonableTextField.Left - gestureOutsideLine})))

		if got := f.position(false).Dx + magnifierBasicOffset.Dx; got < magnifierReasonableTextField.Left {
			t.Errorf("position.dx + the offset = %v, want at least %v", got, magnifierReasonableTextField.Left)
		}
	})

	t.Run("should position vertically at the center of the line", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(magnifierReasonableTextField, magnifierReasonableTextField, magnifierReasonableTextField,
			magnifierReasonableTextField.Center())))

		if got, want := f.position(false).Dy, magnifierReasonableTextField.Center().Dy-magnifierBasicOffset.Dy; got != want {
			t.Errorf("position.dy = %v, want %v", got, want)
		}
	})

	t.Run("should reposition vertically if mashed against the ceiling", func(t *testing.T) {
		topOfScreenTextFieldRect := ui.NewRectFromPoints(ui.Offset{}, ui.Offset{Dx: 200})
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(topOfScreenTextFieldRect, topOfScreenTextFieldRect, topOfScreenTextFieldRect,
			topOfScreenTextFieldRect.TopCenter())))

		if got := f.position(false).Dy; got < 0 {
			t.Errorf("position.dy = %v, want at least 0", got)
		}
	})
}

func TestMagnifierFocalPoint(t *testing.T) {
	t.Run("should shift focal point so that the lens sees nothing out of bounds", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(magnifierReasonableTextField, magnifierReasonableTextField, magnifierReasonableTextField,
			// Gesture on the far right of the magnifier.
			magnifierReasonableTextField.TopLeft())))

		if got := f.additionalFocalPoint().Dx; got >= magnifierReasonableTextField.Left {
			t.Errorf("additional focal point dx = %v, want less than %v", got, magnifierReasonableTextField.Left)
		}
	})

	t.Run("focal point should shift if mashed against the top to always point to text", func(t *testing.T) {
		topOfScreenTextFieldRect := ui.NewRectFromPoints(ui.Offset{}, ui.Offset{Dx: 200})
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(topOfScreenTextFieldRect, topOfScreenTextFieldRect, topOfScreenTextFieldRect,
			topOfScreenTextFieldRect.TopCenter())))

		if got := f.additionalFocalPoint().Dy; got >= 0 {
			t.Errorf("additional focal point dy = %v, want less than 0", got)
		}
	})

	t.Run("a field narrower than the lens points to its center", func(t *testing.T) {
		narrow := ui.NewRectFromLTRB(100, 200, 120, 220)
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, sameLineInfo(narrow, narrow, narrow, narrow.Center())))

		// The magnifier is centered on the gesture, and points at the center of the
		// field: it does not shift.
		if got := f.additionalFocalPoint().Dx; got != 0 {
			t.Errorf("additional focal point dx = %v, want 0", got)
		}
	})
}

func TestMagnifierAnimationState(t *testing.T) {
	const verticalShift = 200.0
	info := func(shift float64) widgets.MagnifierInfo {
		line := magnifierReasonableTextField.Shift(ui.Offset{Dy: shift})
		return widgets.MagnifierInfo{
			CurrentLineBoundaries: line, CaretRect: line,
			FieldBounds:           ui.NewRectFromPoints(magnifierReasonableTextField.TopLeft(), magnifierReasonableTextField.BottomRight().Add(ui.Offset{Dy: shift})),
			GlobalGesturePosition: magnifierReasonableTextField.Center().Add(ui.Offset{Dy: shift}),
		}
	}

	t.Run("should not be animated on the initial state", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		f.show(magnifierInfoOf(t, info(0)))

		if f.isAnimated() {
			t.Error("the magnifier is animated")
		}
	})

	t.Run("should not be animated on horizontal shifts", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		positioner := magnifierInfoOf(t, info(0))
		f.show(positioner)

		// New position has a horizontal shift.
		next := info(0)
		next.GlobalGesturePosition = magnifierReasonableTextField.Center().Add(ui.Offset{Dx: 200})
		positioner.SetValue(next)
		f.ts.PumpAndSettle()

		if f.isAnimated() {
			t.Error("the magnifier is animated")
		}
	})

	t.Run("should be animated on vertical shifts", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		positioner := magnifierInfoOf(t, info(0))
		f.show(positioner)

		// New position has a vertical shift.
		positioner.SetValue(info(verticalShift))

		f.ts.Pump()
		if !f.isAnimated() {
			t.Error("the magnifier is not animated")
		}
	})

	t.Run("should stop being animated when timer is up", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		positioner := magnifierInfoOf(t, info(0))
		f.show(positioner)

		// New position has a vertical shift.
		positioner.SetValue(info(verticalShift))

		f.ts.Pump()
		if !f.isAnimated() {
			t.Error("the magnifier is not animated")
		}
		f.ts.Pump(TextMagnifierJumpBetweenLinesAnimationDuration + 2*time.Second)
		if f.isAnimated() {
			t.Error("the magnifier is still animated")
		}
	})

	t.Run("the position moves to the new line", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		positioner := magnifierInfoOf(t, info(0))
		f.show(positioner)
		before := f.position(true)

		positioner.SetValue(info(verticalShift))
		f.ts.PumpAndSettle()

		if got := f.position(true).Dy - before.Dy; got != verticalShift {
			t.Errorf("the magnifier moved by %v, want %v", got, verticalShift)
		}
	})

	t.Run("a new notifier replaces the old one as what the position follows", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		first, second := magnifierInfoOf(t, info(0)), magnifierInfoOf(t, info(0))
		current := first
		f.controller.Show(f.bc, func(widgets.BuildContext) widgets.Widget { return NewTextMagnifier(current) })
		f.ts.PumpAndSettle()
		before := f.position(true)

		// The overlay entry builds a TextMagnifier of another notifier: the state listens to it.
		current = second
		f.controller.OverlayEntry().MarkNeedsBuild()
		f.ts.PumpAndSettle()
		first.SetValue(info(50))
		f.ts.PumpAndSettle()
		if got := f.position(true); got != before {
			t.Errorf("the magnifier follows the old notifier: it is at %v, want %v", got, before)
		}
		second.SetValue(info(verticalShift))
		f.ts.PumpAndSettle()
		if got := f.position(true).Dy - before.Dy; got != verticalShift {
			t.Errorf("the magnifier moved by %v, want %v", got, verticalShift)
		}
	})
}

func TestMagnifier(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		m := NewMagnifier()
		if m.Size() != MagnifierDefaultMagnifierSize || m.ClipBehavior() != ui.ClipHardEdge || m.FilmColor() != ui.NewColorFromARGB(8, 158, 158, 158) ||
			m.AdditionalFocalPointOffset() != (ui.Offset{}) || len(m.Shadows()) != 1 || m.BorderRadius() != painting.BorderRadiusGeometry(painting.NewBorderRadiusCircular(40)) {
			t.Errorf("defaults = %+v", m)
		}
		want := painting.NewBoxShadow(painting.BoxShadowOpts{BlurRadius: 1.5, Offset: ui.Offset{Dy: 2}, SpreadRadius: 0.75, Color: foundation.Some(ui.NewColorFromARGB(25, 0, 0, 0))})
		if m.Shadows()[0] != want {
			t.Errorf("shadow = %+v, want %+v", m.Shadows()[0], want)
		}
		if got := NewMagnifier(MagnifierOpts{Shadows: []painting.BoxShadow{}}).Shadows(); got == nil || len(got) != 0 {
			t.Errorf("an empty slice of shadows = %v, want no shadow", got)
		}
	})

	t.Run("builds a RawMagnifier in the Material style", func(t *testing.T) {
		withPlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewCenter(NewMagnifier(MagnifierOpts{AdditionalFocalPointOffset: ui.Offset{Dx: 3, Dy: 4}}))}))

		raw := ts.Widgets(gluttertest.ByType[*widgets.RawMagnifier]())
		if len(raw) != 1 {
			t.Fatalf("RawMagnifiers = %d, want 1", len(raw))
		}
		r := raw[0].(*widgets.RawMagnifier)
		wantFocal := ui.Offset{Dx: 3, Dy: 4 + MagnifierStandardVerticalFocalPointShift + MagnifierDefaultMagnifierSize.Height/2}
		if r.MagnificationScale() != 1.25 || r.FocalPointOffset() != wantFocal || r.Size() != MagnifierDefaultMagnifierSize || r.ClipBehavior() != ui.ClipHardEdge {
			t.Errorf("RawMagnifier: scale %v, focal %v (want %v), size %v, clip %v", r.MagnificationScale(), r.FocalPointOffset(), wantFocal, r.Size(), r.ClipBehavior())
		}
		if got := ts.Size(gluttertest.ByType[*widgets.RawMagnifier]()); got != MagnifierDefaultMagnifierSize {
			t.Errorf("size = %v, want %v", got, MagnifierDefaultMagnifierSize)
		}
	})
}

func TestTextMagnifierPositionNeedsTheFirstBuild(t *testing.T) {
	skipUnlessDebug(t)
	// A state that was never given its position: Build is asserted to come after
	// the position was worked out.
	state := (&TextMagnifier{}).CreateState()
	v := func() (v any) {
		defer func() { v = recover() }()
		state.Build(nil)
		return nil
	}()
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Build.TextMagnifierPositionNotSet") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestSelectionAreaMagnifierConfiguration(t *testing.T) {
	custom := widgets.NewTextMagnifierConfiguration()
	tests := []struct {
		name     string
		platform foundation.TargetPlatform
		config   *widgets.TextMagnifierConfiguration
		want     func() *widgets.TextMagnifierConfiguration
	}{
		{"the adaptive one by default", foundation.TargetPlatformAndroid, nil, TextMagnifierAdaptiveMagnifierConfiguration},
		{"the one it is given", foundation.TargetPlatformAndroid, custom, func() *widgets.TextMagnifierConfiguration { return custom }},
		{"disabled", foundation.TargetPlatformAndroid, widgets.TextMagnifierConfigurationDisabled, func() *widgets.TextMagnifierConfiguration { return widgets.TextMagnifierConfigurationDisabled }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withPlatform(t, tt.platform)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewSelectionArea(widgets.NewText("selectable"), SelectionAreaOpts{MagnifierConfiguration: tt.config})}))
			region := gluttertest.WidgetAs[*widgets.SelectableRegion](ts, gluttertest.ByType[*widgets.SelectableRegion]())
			if got := region.MagnifierConfiguration(); got != tt.want() {
				t.Errorf("configuration = %p, want %p", got, tt.want())
			}
		})
	}
}
