package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/magnifier_test.dart. The golden of the position
// of the magnifier over a square is TestCupertinoMagnifierGolden (magnifier_golden_test.go).

var magnifierReasonableTextField = ui.NewRectFromLTRB(0, 100, 200, 200)

type magnifierFixture struct {
	t          *testing.T
	ts         *gluttertest.Tester
	controller *widgets.MagnifierController
	bc         widgets.BuildContext
}

func newMagnifierFixture(t *testing.T, home widgets.Widget, theme ...*cupertino.CupertinoThemeData) *magnifierFixture {
	t.Helper()
	f := &magnifierFixture{t: t, ts: gluttertest.New(t), controller: widgets.NewMagnifierController()}
	t.Cleanup(f.controller.RemoveFromOverlay)
	opts := cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		f.bc = bc
		return home
	})}
	if len(theme) > 0 {
		opts.Theme = theme[0]
	}
	f.ts.PumpWidget(testApp(opts))
	return f
}

func (f *magnifierFixture) show(info *foundation.ValueNotifier[widgets.MagnifierInfo]) {
	f.t.Helper()
	shown := f.controller.Show(f.bc, func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoTextMagnifier(f.controller, info)
	})
	f.ts.PumpAndSettle()
	if completed, ended := shown.Completed(); !completed || !ended {
		f.t.Fatalf("the magnifier is not fully shown: completed %v, ended %v", completed, ended)
	}
}

func (f *magnifierFixture) position() ui.Offset {
	f.t.Helper()
	a := f.ts.Widgets(gluttertest.ByType[*widgets.AnimatedPositioned]())[0].(*widgets.AnimatedPositioned)
	return ui.Offset{Dx: a.Left().OrElse(0), Dy: a.Top().OrElse(0)}
}

func info(line ui.Rect, gesture ui.Offset) *foundation.ValueNotifier[widgets.MagnifierInfo] {
	return foundation.NewValueNotifier(widgets.MagnifierInfo{CurrentLineBoundaries: line, FieldBounds: line, CaretRect: line, GlobalGesturePosition: gesture})
}

func TestCupertinoTextMagnifier(t *testing.T) {
	basicOffset := ui.Offset{Dx: cupertino.CupertinoMagnifierDefaultSize.Width / 2,
		Dy: cupertino.CupertinoMagnifierDefaultSize.Height - cupertino.CupertinoMagnifierAboveFocalPoint}
	t.Run("the border color inherits from the CupertinoTheme", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil),
			testTheme(cupertino.CupertinoThemeDataOpts{PrimaryColor: cupertino.CupertinoColors.ActiveGreen}))
		n := info(magnifierReasonableTextField, magnifierReasonableTextField.Center())
		defer n.Dispose()
		f.show(n)
		got := f.ts.Widget(gluttertest.ByType[*cupertino.CupertinoMagnifier]()).(*cupertino.CupertinoMagnifier)
		if c := got.BorderSide().Color; c != cupertino.CupertinoColors.ActiveGreen.Color {
			t.Errorf("border color %v, want the primary color of the theme", c)
		}
	})
	t.Run("the magnifier is never outside the screen padding", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		width := f.ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		n := info(magnifierReasonableTextField, ui.Offset{Dx: width + 100})
		defer n.Dispose()
		f.show(n)
		if x := f.position().Dx; x >= width {
			t.Errorf("the magnifier is at %v, the screen is %v wide", x, width)
		}
	})
	t.Run("the magnifier has some vertical drag", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		width := f.ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		below := magnifierReasonableTextField.Center().Dy + 30
		n := info(magnifierReasonableTextField, ui.Offset{Dx: width / 2, Dy: below})
		defer n.Dispose()
		f.show(n)
		y := f.position().Dy + basicOffset.Dy
		if y <= magnifierReasonableTextField.Center().Dy || y >= below {
			t.Errorf("the magnifier is at %v, want between %v and %v", y, magnifierReasonableTextField.Center().Dy, below)
		}
	})
	t.Run("the magnifier hides when the gesture is far below the line and shows again when it moves back", func(t *testing.T) {
		f := newMagnifierFixture(t, widgets.NewPlaceholder(nil))
		width := f.ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		at := ui.Offset{Dx: width / 2, Dy: magnifierReasonableTextField.Top}
		n := info(magnifierReasonableTextField, at)
		defer n.Dispose()
		f.show(n)
		moved := n.Value()
		moved.GlobalGesturePosition = at.Add(ui.Offset{Dy: 100})
		n.SetValue(moved)
		f.ts.PumpAndSettle()
		if f.controller.Shown() || f.controller.OverlayEntry() == nil {
			t.Fatalf("shown %v, entry %v", f.controller.Shown(), f.controller.OverlayEntry())
		}
		moved.GlobalGesturePosition = at
		n.SetValue(moved)
		f.ts.PumpAndSettle()
		if !f.controller.Shown() || f.controller.OverlayEntry() == nil {
			t.Errorf("shown %v, entry %v", f.controller.Shown(), f.controller.OverlayEntry())
		}
	})
}

func TestCupertinoMagnifierScale(t *testing.T) {
	scaleOf := func(ts *gluttertest.Tester) float64 {
		return ts.Widget(gluttertest.ByType[*widgets.RawMagnifier]()).(*widgets.RawMagnifier).MagnificationScale()
	}
	t.Run("the scale must be greater than zero", func(t *testing.T) {
		skipUnlessDebug(t)
		for _, scale := range []float64{0, -1} {
			err := recoverError(func() {
				cupertino.NewCupertinoMagnifier(cupertino.CupertinoMagnifierOpts{MagnificationScale: foundation.Some(scale)})
			})
			checkErrorTag(t, err, "CupertinoMagnifierScaleInvalid")
		}
	})
	t.Run("the scale defaults to 1 and is passed to the RawMagnifier", func(t *testing.T) {
		for _, tc := range []struct {
			opts cupertino.CupertinoMagnifierOpts
			want float64
		}{{cupertino.CupertinoMagnifierOpts{}, 1}, {cupertino.CupertinoMagnifierOpts{MagnificationScale: foundation.Some(2.0)}, 2}} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(cupertino.NewCupertinoMagnifier(tc.opts))}))
			if got := scaleOf(ts); got != tc.want {
				t.Errorf("scale %v, want %v", got, tc.want)
			}
		}
	})
	t.Run("a magnifier of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
			cupertino.NewCupertinoMagnifier(cupertino.CupertinoMagnifierOpts{MagnificationScale: foundation.Some(2.0)})))}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoMagnifier]()); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
}
