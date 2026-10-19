package widgets_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/transitions_test.dart. The
// SliverFadeTransition test waits for the slivers (step 12), and the 'toString
// control test' for DebugFillProperties (step 17).

// newVsyncController makes a controller on the scheduler of the tester, with a
// duration (TestVSync).
func newVsyncController(ts *gluttertest.Tester, opts ...animation.AnimationControllerOpts) *animation.AnimationController {
	var o animation.AnimationControllerOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return animation.NewAnimationController(vsyncOf{ts.Binding.Scheduler()}, o)
}

// atFullValue is a controller at 1 with a duration of 2 seconds, as the
// zero-area tests use.
func atFullValue(ts *gluttertest.Tester) *animation.AnimationController {
	return newVsyncController(ts, animation.AnimationControllerOpts{Value: foundation.Some(1.0), Duration: foundation.Some(2 * time.Second)})
}

func closeEnough(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func colorsClose(a, b ui.Color) bool {
	return closeEnough(float64(a.A), float64(b.A), 0.01) && closeEnough(float64(a.R), float64(b.R), 0.01) &&
		closeEnough(float64(a.G), float64(b.G), 0.01) && closeEnough(float64(a.B), float64(b.B), 0.01)
}

func matricesClose(a, b *ui.Matrix4) bool {
	for i, v := range a.Storage() {
		if !closeEnough(float64(v), float64(b.Storage()[i]), 1e-6) {
			return false
		}
	}
	return true
}

// transposedFromRows is Matrix4.fromList(rows)..transpose().
func transposedFromRows(rows [16]float32) *ui.Matrix4 {
	m := ui.NewMatrix4FromList(rows)
	m.Transpose()
	return m
}

func zeroView(ts *gluttertest.Tester) { ts.SetViewSize(ui.Size{}) }

func TestDecoratedBoxTransition(t *testing.T) {
	tween := NewDecorationTween(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(ui.NewColor(0xFFFFFFFF)),
			Border:       painting.NewBorderAll(painting.BorderSideOpts{Width: foundation.Some(4.0)}),
			BorderRadius: painting.BorderRadiusZero,
			BoxShadow: []painting.BoxShadow{painting.NewBoxShadow(painting.BoxShadowOpts{
				Color: foundation.Some(ui.NewColor(0x66000000)), BlurRadius: 10, SpreadRadius: 4})},
		}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(ui.NewColor(0xFF000000)),
			Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFF202020))}),
			BorderRadius: painting.NewBorderRadiusCircular(10),
		}))
	boxOf := func(ts *gluttertest.Tester) *painting.BoxDecoration {
		return ts.RenderObject(gluttertest.ByType[*DecoratedBox]()).(*rendering.RenderDecoratedBox).Decoration().(*painting.BoxDecoration)
	}
	text := func() Widget { return NewText("Doesn't matter", TextOpts{TextDirection: ui.TextDirectionLTR}) }

	t.Run("decoration", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewDecoratedBoxTransition(tween.Animate(controller), text()))
		d := boxOf(ts)
		if c, _ := d.Color.Get(); !colorsClose(c, ui.NewColor(0xFFFFFFFF)) {
			t.Errorf("color = %v", c)
		}
		if s := d.BoxShadow[0]; s.BlurRadius != 10 || s.SpreadRadius != 4 || s.Color != ui.NewColor(0x66000000) {
			t.Errorf("shadow = %+v", s)
		}
		controller.SetValue(0.5)
		ts.Pump()
		d = boxOf(ts)
		if c, _ := d.Color.Get(); !colorsClose(c, ui.NewColor(0xFF7F7F7F)) {
			t.Errorf("color at 0.5 = %v", c)
		}
		border, ok := d.Border.(painting.Border)
		if !ok {
			t.Fatalf("border = %T, want Border", d.Border)
		}
		if border.Left.Width != 2.5 || border.Left.Style != painting.BorderStyleSolid || border.Left.Color != ui.NewColor(0xFF101010) {
			t.Errorf("border.left = %+v", border.Left)
		}
		if d.BorderRadius != painting.BorderRadiusGeometry(painting.NewBorderRadiusCircular(5)) {
			t.Errorf("borderRadius = %v", d.BorderRadius)
		}
		if d.Shape != painting.BoxShapeRectangle {
			t.Errorf("shape = %v", d.Shape)
		}
		if s := d.BoxShadow[0]; s.BlurRadius != 5 || s.SpreadRadius != 2 || s.Color != ui.NewColor(0x66000000) {
			t.Errorf("shadow at 0.5 = %+v", s)
		}
		controller.SetValue(1)
		ts.Pump()
		d = boxOf(ts)
		if c, _ := d.Color.Get(); c != ui.NewColor(0xFF000000) {
			t.Errorf("color at 1 = %v", c)
		}
		if d.BoxShadow != nil {
			t.Errorf("shadow at 1 = %v, want nil", d.BoxShadow)
		}
	})

	t.Run("animations work with curves", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		curved := animation.NewCurvedAnimation(controller, animation.Curves.EaseOut)
		defer curved.Dispose()
		ts.PumpWidget(NewDecoratedBoxTransition(tween.Animate(curved), text(), DecoratedBoxTransitionOpts{Position: rendering.DecorationPositionForeground}))
		d := boxOf(ts)
		if c, _ := d.Color.Get(); !colorsClose(c, ui.NewColor(0xFFFFFFFF)) {
			t.Errorf("color = %v", c)
		}
		controller.SetValue(0.5)
		ts.Pump()
		d = boxOf(ts)
		if c, _ := d.Color.Get(); !colorsClose(c, ui.NewColor(0xFF505050)) {
			t.Errorf("color at 0.5 = %v", c)
		}
		border := d.Border.(painting.Border)
		if !closeEnough(border.Left.Width, 1.9, 0.1) || !colorsClose(border.Left.Color, ui.NewColor(0xFF151515)) {
			t.Errorf("border.left = %+v", border.Left)
		}
		if tl := d.BorderRadius.Resolve(ui.TextDirectionLTR).TopLeft; !closeEnough(tl.X, 6.8, 0.1) {
			t.Errorf("topLeft.x = %v", tl.X)
		}
		if s := d.BoxShadow[0]; !closeEnough(s.BlurRadius, 3.1, 0.1) || !closeEnough(s.SpreadRadius, 1.2, 0.1) {
			t.Errorf("shadow = %+v", s)
		}
	})

	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewDecoratedBoxTransition(
			NewDecorationTween(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFF000000))}),
				painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFFFFFF))})).Animate(controller),
			NewPlaceholder(nil)))))
		if size := ts.Size(gluttertest.ByType[*DecoratedBoxTransition]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

func alignmentTween() *animation.Tween[painting.AlignmentGeometry] {
	return rendering.NewAlignmentGeometryTween(painting.AlignmentCenterLeft, painting.AlignmentBottomRight)
}

func TestAlignTransition(t *testing.T) {
	t.Run("animates", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewAlignTransition(alignmentTween().Animate(controller), NewText("Ready", TextOpts{TextDirection: ui.TextDirectionLTR})))
		box := ts.RenderObject(gluttertest.ByType[*Align]()).(*rendering.RenderPositionedBox)
		if got := box.Alignment(); !got.Equal(painting.AlignmentCenterLeft) {
			t.Errorf("alignment = %v", got)
		}
		controller.SetValue(0.5)
		ts.Pump()
		if got := box.Alignment(); !got.Equal(painting.Alignment{X: 0, Y: 0.5}) {
			t.Errorf("alignment at 0.5 = %v", got)
		}
	})
	t.Run("keeps width and height factors", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewAlignTransition(alignmentTween().Animate(controller), NewText("Ready", TextOpts{TextDirection: ui.TextDirectionLTR}),
			AlignTransitionOpts{WidthFactor: foundation.Some(0.3), HeightFactor: foundation.Some(0.4)}))
		a := ts.Widget(gluttertest.ByType[*Align]()).(*Align)
		if a.WidthFactor() != foundation.Some(0.3) || a.HeightFactor() != foundation.Some(0.4) {
			t.Errorf("factors = %v, %v", a.WidthFactor(), a.HeightFactor())
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		curved := animation.NewCurvedAnimation(controller, animation.Curves.FastOutSlowIn)
		defer curved.Dispose()
		tween := animation.NewTweenFunc[painting.AlignmentGeometry](painting.AlignmentBottomCenter, painting.AlignmentBottomRight, painting.AlignmentGeometryLerp)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewAlignTransition(tween.Animate(curved), NewPlaceholder(nil)))))
		if size := ts.Size(gluttertest.ByType[*AlignTransition]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

func TestRelativePositionedTransitionAnimates(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newVsyncController(ts)
	defer controller.Dispose()
	rect := animation.NewRectTween(foundation.Some(ui.NewRectFromLTWH(0, 0, 30, 40)), foundation.Some(ui.NewRectFromLTWH(100, 200, 100, 200))).Animate(controller)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewStack([]Widget{
		NewRelativePositionedTransition(rect, ui.Size{Width: 200, Height: 300}, NewPlaceholder(nil)),
	}, StackOpts{Alignment: painting.AlignmentCenterLeft})))
	positioned := func() ui.Rect {
		p := ts.Widget(gluttertest.ByType[*Positioned]()).(*Positioned)
		return ui.NewRectFromLTRB(p.Left().Value(), p.Top().Value(), p.Right().OrElse(0), p.Bottom().OrElse(0))
	}
	if got := positioned(); got != ui.NewRectFromLTRB(0, 0, 170, 260) {
		t.Errorf("rect = %v", got)
	}
	if got := ts.Size(gluttertest.ByType[*Placeholder]()); got != (ui.Size{Width: 630, Height: 340}) {
		t.Errorf("size = %v", got)
	}
	controller.SetValue(0.5)
	ts.Pump()
	// Deviation: the Dart test reads the Positioned it found before the pump, so
	// it sees the old values; this one reads the new widget (SPEC section 12.10).
	if got := positioned(); got != ui.NewRectFromLTRB(50, 100, 85, 80) {
		t.Errorf("rect at 0.5 = %v", got)
	}
	if got := ts.Size(gluttertest.ByType[*Placeholder]()); got != (ui.Size{Width: 665, Height: 420}) {
		t.Errorf("size at 0.5 = %v", got)
	}
}

func TestSizeTransition(t *testing.T) {
	factors := func(ts *gluttertest.Tester) (width, height foundation.Opt[float64]) {
		box := ts.RenderObject(gluttertest.ByType[*Align]()).(*rendering.RenderPositionedBox)
		return box.WidthFactor(), box.HeightFactor()
	}
	tests := []struct {
		name  string
		axis  painting.Axis
		fixed float64
		steps []struct{ value, want float64 }
	}{
		{"clamps negative size factors - vertical axis", painting.AxisVertical, 2, []struct{ value, want float64 }{{-1, 0}, {0, 0}, {0.75, 0.5}, {1, 1}}},
		{"clamps negative size factors - horizontal axis", painting.AxisHorizontal, 1, []struct{ value, want float64 }{{-1, 0}, {0, 0}, {0.75, 0.5}, {1, 1}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := newVsyncController(ts)
			defer controller.Dispose()
			anim := animation.NewTween(-1.0, 1.0).Animate(controller)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewSizeTransition(anim, NewText("Ready"),
				SizeTransitionOpts{Axis: tc.axis, FixedCrossAxisSizeFactor: foundation.Some(tc.fixed)})))
			check := func(want float64) {
				t.Helper()
				w, h := factors(ts)
				moving, fixed := h, w
				if tc.axis == painting.AxisHorizontal {
					moving, fixed = w, h
				}
				if moving != foundation.Some(want) || fixed != foundation.Some(tc.fixed) {
					t.Errorf("moving %v, fixed %v; want %v, %v", moving, fixed, want, tc.fixed)
				}
			}
			check(0)
			for _, s := range tc.steps[1:] {
				controller.SetValue(s.value)
				ts.Pump()
				check(s.want)
			}
		})
	}

	t.Run("with fixedCrossAxisSizeFactor sizes the cross axis from its children", func(t *testing.T) {
		for _, axis := range []painting.Axis{painting.AxisVertical, painting.AxisHorizontal} {
			ts := gluttertest.New(t)
			controller := newVsyncController(ts)
			anim := animation.NewTween(0.0, 1.0).Animate(controller)
			key := foundation.NewKey("key")
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBox(
				NewSizeTransition(anim, NewSizedBoxSquare(foundation.Some(100.0), nil), SizeTransitionOpts{Axis: axis, FixedCrossAxisSizeFactor: foundation.Some(1.0)}),
				SizedBoxOpts{Key: key}))))
			for _, v := range []float64{0, 0.5, 1, 0.5, 0} {
				controller.SetValue(v)
				ts.Pump()
				want := ui.Size{Width: 100, Height: 100 * v}
				if axis == painting.AxisHorizontal {
					want = ui.Size{Width: 100 * v, Height: 100}
				}
				if got := ts.Size(gluttertest.ByKey(key)); got != want {
					t.Errorf("axis %v at %v: size = %v, want %v", axis, v, got, want)
				}
			}
			controller.Dispose()
		}
	})

	t.Run("maintains chosen alignment during animation", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		anim := animation.NewTween(0.0, 1.0).Animate(controller)
		ts.PumpWidget(NewSizeTransition(anim, NewSizedBoxShrink(nil), SizeTransitionOpts{Alignment: painting.AlignmentTopLeft}))
		box := ts.RenderObject(gluttertest.ByType[*Align]()).(*rendering.RenderPositionedBox)
		for _, v := range []float64{0, 1} {
			controller.SetValue(v)
			ts.Pump()
			if got := box.Alignment(); got != painting.AlignmentGeometry(painting.AlignmentTopLeft) {
				t.Errorf("alignment at %v = %v", v, got)
			}
		}
	})

	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		curved := animation.NewCurvedAnimation(controller, animation.Curves.FastOutSlowIn)
		defer curved.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(
			NewSizeTransition(curved, NewPlaceholder(nil), SizeTransitionOpts{Axis: painting.AxisHorizontal, Alignment: painting.AlignmentTopLeft}))))
		if size := ts.Size(gluttertest.ByType[*SizeTransition]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})

	t.Run("both alignments panic", func(t *testing.T) {
		skipUnlessDebug(t)
		requireTag(t, "SizeTransitionInvalid", func() {
			NewSizeTransition(animation.AlwaysComplete, nil, SizeTransitionOpts{AxisAlignment: foundation.Some(0.5), Alignment: painting.AlignmentTopLeft})
		})
		requireTag(t, "SizeTransitionInvalid", func() {
			NewSizeTransition(animation.AlwaysComplete, nil, SizeTransitionOpts{FixedCrossAxisSizeFactor: foundation.Some(-1.0)})
		})
	})
}

func TestMatrixTransition(t *testing.T) {
	text := func(s string) Widget { return NewText(s, TextOpts{TextDirection: ui.TextDirectionLTR}) }
	transformOf := func(ts *gluttertest.Tester) *ui.Matrix4 {
		return ts.Widget(gluttertest.ByType[*Transform]()).(*Transform).Transform()
	}

	t.Run("animates", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewMatrixTransition(controller, func(v float64) *ui.Matrix4 {
			return ui.NewMatrix4TranslationValues(float32(v), float32(v), float32(v))
		}, text("Matrix"), MatrixTransitionOpts{Alignment: painting.AlignmentTopRight}))
		if got := transformOf(ts); !got.Equals(ui.NewMatrix4RotationZ(0)) {
			t.Errorf("transform = %v", got)
		}
		for _, v := range []float32{0.5, 0.75} {
			controller.SetValue(float64(v))
			ts.Pump()
			want := transposedFromRows([16]float32{1, 0, 0, v, 0, 1, 0, v, 0, 0, 1, v, 0, 0, 0, 1})
			if got := transformOf(ts); !got.Equals(want) {
				t.Errorf("transform at %v = %v, want %v", v, got, want)
			}
		}
	})

	t.Run("maintains chosen alignment during animation", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewMatrixTransition(controller, func(float64) *ui.Matrix4 { return ui.NewMatrix4Identity() }, text("Matrix"),
			MatrixTransitionOpts{Alignment: painting.AlignmentTopRight}))
		for _, v := range []float64{0, 0.5} {
			controller.SetValue(v)
			ts.Pump()
			if got := ts.Widget(gluttertest.ByType[*MatrixTransition]()).(*MatrixTransition).Alignment(); got != painting.AlignmentGeometry(painting.AlignmentTopRight) {
				t.Errorf("alignment = %v", got)
			}
		}
	})

	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		curved := animation.NewCurvedAnimation(controller, animation.Curves.Linear)
		defer curved.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewMatrixTransition(curved,
			func(float64) *ui.Matrix4 { return ui.NewMatrix4Identity() }, NewPlaceholder(nil)))))
		if size := ts.Size(gluttertest.ByType[*MatrixTransition]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

func TestRotationTransition(t *testing.T) {
	text := func() Widget { return NewText("Rotation", TextOpts{TextDirection: ui.TextDirectionLTR}) }
	t.Run("animates", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewRotationTransition(controller, text(), RotationTransitionOpts{Alignment: painting.AlignmentTopRight}))
		transformOf := func() *ui.Matrix4 { return ts.Widget(gluttertest.ByType[*Transform]()).(*Transform).Transform() }
		if got := transformOf(); !got.Equals(ui.NewMatrix4RotationZ(0)) {
			t.Errorf("transform = %v", got)
		}
		controller.SetValue(0.5)
		ts.Pump()
		if got, want := transformOf(), transposedFromRows([16]float32{-1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}); !matricesClose(got, want) {
			t.Errorf("transform at 0.5 = %v, want %v", got, want)
		}
		controller.SetValue(0.75)
		ts.Pump()
		if got, want := transformOf(), transposedFromRows([16]float32{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}); !matricesClose(got, want) {
			t.Errorf("transform at 0.75 = %v, want %v", got, want)
		}
	})
	t.Run("maintains chosen alignment during animation", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewRotationTransition(controller, text(), RotationTransitionOpts{Alignment: painting.AlignmentTopRight}))
		for _, v := range []float64{0, 0.5} {
			controller.SetValue(v)
			ts.Pump()
			if got := ts.Widget(gluttertest.ByType[*RotationTransition]()).(*RotationTransition).Alignment(); got != painting.AlignmentGeometry(painting.AlignmentTopRight) {
				t.Errorf("alignment = %v", got)
			}
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		curved := animation.NewCurvedAnimation(controller, animation.Curves.ElasticOut)
		defer curved.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewRotationTransition(curved, NewPlaceholder(nil)))))
		if size := ts.Size(gluttertest.ByType[*RotationTransition]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

func TestFadeTransitionAnimates(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newVsyncController(ts)
	defer controller.Dispose()
	anim := animation.NewTween(0.0, 1.0).Animate(controller)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewFadeTransition(anim, NewText("Fade In"))))
	opacity := func() float64 {
		w := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Fade In"), gluttertest.ByType[*FadeTransition]()))).(*FadeTransition)
		return w.Opacity().Value()
	}
	for _, v := range []float64{0, 0.25, 0.5, 0.75, 1} {
		controller.SetValue(v)
		ts.Pump()
		if got := opacity(); got != v {
			t.Errorf("opacity = %v, want %v", got, v)
		}
	}
}

// imageFilterLayers are the image filter layers of the layer tree.
func imageFilterLayers(ts *gluttertest.Tester) []*rendering.ImageFilterLayer {
	var out []*rendering.ImageFilterLayer
	for _, view := range ts.Binding.Renderer().RenderViews() {
		root := view.DebugLayer()
		if root == nil {
			continue
		}
		all := append([]rendering.Layer{root}, root.AsContainerLayer().DepthFirstIterateChildren()...)
		for _, l := range all {
			if f, ok := l.(*rendering.ImageFilterLayer); ok {
				out = append(out, f)
			}
		}
	}
	return out
}

func TestTransitionsUseAnImageFilterWithAFilterQuality(t *testing.T) {
	skipUnlessDebug(t) // the layers are found through DebugLayer
	tests := []struct {
		name string
		make func(animation.Animation[float64]) Widget
	}{
		{"MatrixTransition", func(a animation.Animation[float64]) Widget {
			return NewMatrixTransition(a, func(float64) *ui.Matrix4 { return ui.NewMatrix4Identity() }, NewText("Matrix Transition"),
				MatrixTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityNone)})
		}},
		{"ScaleTransition", func(a animation.Animation[float64]) Widget {
			return NewScaleTransition(a, NewText("Scale Transition"), ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityNone)})
		}},
		{"RotationTransition", func(a animation.Animation[float64]) Widget {
			return NewRotationTransition(a, NewText("Scale Transition"), RotationTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityNone)})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := newVsyncController(ts)
			defer controller.Dispose()
			anim := animation.NewTween(0.0, 1.0).Animate(controller)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, tc.make(anim)))
			if n := len(imageFilterLayers(ts)); n != 0 {
				t.Errorf("at the start: %d image filter layers, want 0", n)
			}
			for _, v := range []float64{0.25, 0.5, 0.75} {
				controller.SetValue(v)
				ts.Pump()
				layers := imageFilterLayers(ts)
				if len(layers) == 0 {
					t.Fatalf("at %v: no image filter layer", v)
				}
				if !strings.HasPrefix(layers[0].ImageFilter().String(), "ImageFilter.matrix(") {
					t.Errorf("at %v: filter = %v", v, layers[0].ImageFilter())
				}
			}
			controller.SetValue(1)
			ts.Pump()
			if n := len(imageFilterLayers(ts)); n != 0 {
				t.Errorf("at the end: %d image filter layers, want 0", n)
			}
		})
	}
}

func TestScaleTransitionDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	zeroView(ts)
	controller := atFullValue(ts)
	defer controller.Dispose()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewScaleTransition(controller, NewPlaceholder(nil)))))
	if size := ts.Size(gluttertest.ByType[*ScaleTransition]()); size != (ui.Size{}) {
		t.Errorf("size = %v", size)
	}
}

// redrawCounter is Flutter's RedrawCounter: it counts its builds.
type redrawCounter struct {
	StatefulWidget
	child Widget
}

func newRedrawCounter(key foundation.Key, child Widget) *redrawCounter {
	return &redrawCounter{NewStatefulWidgetBase(WidgetOpts{Key: key}), child}
}

func (w *redrawCounter) CreateState() State { return &redrawCounterState{} }

type redrawCounterState struct {
	StateOf[*redrawCounter]
	redraws int
}

func (s *redrawCounterState) Build(BuildContext) Widget {
	s.redraws++
	return NewSizedBox(s.Widget().child)
}

func TestBuilders(t *testing.T) {
	builders := []struct {
		name string
		make func(n *foundation.ChangeNotifier, builder TransitionBuilder, child Widget) Widget
	}{
		{"AnimatedBuilder", func(n *foundation.ChangeNotifier, b TransitionBuilder, c Widget) Widget {
			return NewAnimatedBuilder(n, b, c)
		}},
		{"ListenableBuilder", func(n *foundation.ChangeNotifier, b TransitionBuilder, c Widget) Widget {
			return NewListenableBuilder(n, b, c)
		}},
	}
	for _, b := range builders {
		t.Run(b.name+" rebuilds when changed", func(t *testing.T) {
			ts := gluttertest.New(t)
			redrawKey := NewGlobalKey[*redrawCounterState]()
			notifier := foundation.NewChangeNotifier()
			defer notifier.Dispose()
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, b.make(notifier, func(_ BuildContext, child Widget) Widget {
				return newRedrawCounter(redrawKey, child)
			}, nil)))
			redraws := func() int { s, _ := redrawKey.CurrentState(); return s.redraws }
			if redraws() != 1 {
				t.Errorf("redraws = %d, want 1", redraws())
			}
			ts.Pump()
			if redraws() != 1 {
				t.Errorf("redraws = %d, want 1", redraws())
			}
			notifier.NotifyListeners()
			ts.Pump()
			if redraws() != 2 {
				t.Errorf("redraws = %d, want 2", redraws())
			}
			ts.Pump()
			ts.Pump()
			if redraws() != 2 {
				t.Errorf("redraws = %d, want 2", redraws())
			}
		})
		t.Run(b.name+" doesn't rebuild the child", func(t *testing.T) {
			ts := gluttertest.New(t)
			redrawKey := NewGlobalKey[*redrawCounterState]()
			childKey := NewGlobalKey[*redrawCounterState]()
			notifier := foundation.NewChangeNotifier()
			defer notifier.Dispose()
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, b.make(notifier, func(_ BuildContext, child Widget) Widget {
				return newRedrawCounter(redrawKey, child)
			}, newRedrawCounter(childKey, nil))))
			redraws := func(k *GlobalKey[*redrawCounterState]) int { s, _ := k.CurrentState(); return s.redraws }
			if redraws(redrawKey) != 1 || redraws(childKey) != 1 {
				t.Errorf("redraws = %d, %d, want 1, 1", redraws(redrawKey), redraws(childKey))
			}
			notifier.NotifyListeners()
			ts.Pump()
			ts.Pump()
			if redraws(redrawKey) != 2 || redraws(childKey) != 1 {
				t.Errorf("redraws = %d, %d, want 2, 1", redraws(redrawKey), redraws(childKey))
			}
		})
	}

	t.Run("AnimatedBuilder does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		controller := atFullValue(ts)
		defer controller.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewAnimatedBuilder(controller,
			func(BuildContext, Widget) Widget { return NewText("X") }, NewPlaceholder(nil)))))
		if size := ts.Size(gluttertest.ByType[*AnimatedBuilder]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
	t.Run("ListenableBuilder does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		zeroView(ts)
		notifier := foundation.NewChangeNotifier()
		defer notifier.Dispose()
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewListenableBuilder(notifier,
			func(BuildContext, Widget) Widget { return NewPlaceholder(nil) }, nil))))
		if size := ts.Size(gluttertest.ByType[*ListenableBuilder]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

func TestDefaultTextStyleTransitionDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	zeroView(ts)
	controller := atFullValue(ts)
	defer controller.Dispose()
	tween := NewTextStyleTween(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(20.0)}), painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(30.0)}))
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewDefaultTextStyleTransition(tween.Animate(controller), NewPlaceholder(nil)))))
	if size := ts.Size(gluttertest.ByType[*DefaultTextStyleTransition]()); size != (ui.Size{}) {
		t.Errorf("size = %v", size)
	}
}

func TestPositionedTransitionPositionsItsChild(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newVsyncController(ts)
	defer controller.Dispose()
	tween := NewRelativeRectTween(rendering.NewRelativeRectFromLTRB(0, 0, 100, 100), rendering.NewRelativeRectFromLTRB(50, 50, 0, 0))
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewStack([]Widget{
		NewPositionedTransition(tween.Animate(controller), NewPlaceholder(nil)),
	})))
	controller.SetValue(0.5)
	ts.Pump()
	p := ts.Widget(gluttertest.ByType[*Positioned]()).(*Positioned)
	if p.Left() != foundation.Some(25.0) || p.Top() != foundation.Some(25.0) || p.Right() != foundation.Some(50.0) || p.Bottom() != foundation.Some(50.0) {
		t.Errorf("positioned = %v %v %v %v", p.Left(), p.Top(), p.Right(), p.Bottom())
	}
}
