// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/: container_test.dart,
// visibility_test.dart, placeholder_test.dart, opacity_test.dart,
// transform_test.dart, clip_test.dart, custom_paint_test.dart,
// decorated_box (container_test.dart), colored_box_test.dart,
// repaint_boundary (basic_test.dart), custom_multi_child_layout_test.dart,
// custom_single_child_layout_test.dart and ignore/absorb pointer tests.

func pixel(t *testing.T, ts *gluttertest.Tester, x, y int) color.RGBA {
	t.Helper()
	return ts.Screenshot().RGBAAt(x, y)
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

// unpainted is a pixel that was not painted: the view has no background.
var unpainted = color.RGBA{}

func TestContainerSizing(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	tests := []struct {
		name string
		w    *Container
		want ui.Size
	}{
		{"no child expands", NewContainer(nil), ui.Size{Width: 800, Height: 600}},
		{"width and height", NewContainer(nil, ContainerOpts{Width: foundation.Some(100.0), Height: foundation.Some(50.0)}), ui.Size{Width: 100, Height: 50}},
		{"child sizes it", NewContainer(newBox(30, 20)), ui.Size{Width: 30, Height: 20}},
		{"padding", NewContainer(newBox(30, 20), ContainerOpts{Padding: painting.NewEdgeInsetsAll(5)}), ui.Size{Width: 40, Height: 30}},
		{"margin", NewContainer(newBox(30, 20), ContainerOpts{Margin: painting.NewEdgeInsetsAll(5)}), ui.Size{Width: 40, Height: 30}},
		{"padding and margin", NewContainer(newBox(30, 20), ContainerOpts{Padding: painting.NewEdgeInsetsAll(5), Margin: painting.NewEdgeInsetsAll(1)}), ui.Size{Width: 42, Height: 32}},
		{"color", NewContainer(newBox(30, 20), ContainerOpts{Color: foundation.Some(red)}), ui.Size{Width: 30, Height: 20}},
		{"decoration padding adds to the padding", NewContainer(newBox(30, 20), ContainerOpts{
			Padding: painting.NewEdgeInsetsAll(5), Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.NewBorderAll(painting.BorderSideOpts{Width: foundation.Some(2.0)})})}),
			ui.Size{Width: 44, Height: 34}},
		{"constraints", NewContainer(newBox(30, 20), ContainerOpts{Constraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 60, Height: 70}))}), ui.Size{Width: 60, Height: 70}},
		{"width merges into the constraints", NewContainer(newBox(30, 20), ContainerOpts{
			Constraints: foundation.Some(rendering.BoxConstraints{MinHeight: 10, MaxHeight: 10, MaxWidth: 500}), Width: foundation.Some(40.0)}), ui.Size{Width: 40, Height: 10}},
		{"alignment expands", NewContainer(newBox(30, 20), ContainerOpts{Alignment: painting.AlignmentCenter}), ui.Size{Width: 800, Height: 600}},
		{"transform keeps the layout size", NewContainer(newBox(30, 20), ContainerOpts{Transform: ui.NewMatrix4Diagonal3Values(2, 2, 1)}), ui.Size{Width: 30, Height: 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			if got := ts.Size(gluttertest.ByWidget(tt.w)); got != tt.want {
				t.Errorf("size = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerBuildsOnlyWhatItNeeds(t *testing.T) {
	deco := painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFF00FF00))})
	tests := []struct {
		name string
		w    *Container
		has  map[string]bool
	}{
		{"unpainted", NewContainer(newBox(1, 1)), map[string]bool{"Padding": false, "ColoredBox": false, "DecoratedBox": false, "ConstrainedBox": false, "Align": false, "ClipPath": false, "Transform": false}},
		{"color", NewContainer(newBox(1, 1), ContainerOpts{Color: foundation.Some(ui.NewColor(0xFFFF0000))}), map[string]bool{"ColoredBox": true, "DecoratedBox": false}},
		{"decoration", NewContainer(newBox(1, 1), ContainerOpts{Decoration: deco}), map[string]bool{"DecoratedBox": true, "ClipPath": false}},
		{"clip", NewContainer(newBox(1, 1), ContainerOpts{Decoration: deco, ClipBehavior: ui.ClipAntiAlias}), map[string]bool{"ClipPath": true, "DecoratedBox": true}},
		{"margin", NewContainer(newBox(1, 1), ContainerOpts{Margin: painting.NewEdgeInsetsAll(1)}), map[string]bool{"Padding": true}},
		{"alignment", NewContainer(newBox(1, 1), ContainerOpts{Alignment: painting.AlignmentCenter}), map[string]bool{"Align": true}},
		{"width", NewContainer(newBox(1, 1), ContainerOpts{Width: foundation.Some(5.0)}), map[string]bool{"ConstrainedBox": true}},
		{"transform", NewContainer(newBox(1, 1), ContainerOpts{Transform: ui.NewMatrix4Identity()}), map[string]bool{"Transform": true}},
		{"foreground decoration", NewContainer(newBox(1, 1), ContainerOpts{ForegroundDecoration: deco}), map[string]bool{"DecoratedBox": true}},
	}
	finders := map[string]gluttertest.Finder{
		"Padding": gluttertest.ByType[*Padding](), "ColoredBox": gluttertest.ByType[*ColoredBox](), "DecoratedBox": gluttertest.ByType[*DecoratedBox](),
		"ConstrainedBox": gluttertest.ByType[*ConstrainedBox](), "Align": gluttertest.ByType[*Align](), "ClipPath": gluttertest.ByType[*ClipPath](),
		"Transform": gluttertest.ByType[*Transform](),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, tt.w))
			for name, want := range tt.has {
				// The Align that the test wraps the widget in is not the container's.
				n := ts.Count(finders[name])
				if name == "Align" {
					continue
				}
				if (n > 0) != want {
					t.Errorf("%s: found %d, want present = %v", name, n, want)
				}
			}
		})
	}
}

func TestContainerMisuse(t *testing.T) {
	skipUnlessDebug(t)
	c := foundation.Some(ui.NewColor(0xFFFF0000))
	deco := painting.NewBoxDecoration()
	tests := []struct {
		name string
		do   func()
		tag  string
	}{
		{"color and decoration", func() { NewContainer(nil, ContainerOpts{Color: c, Decoration: deco}) }, "ContainerColorAndDecoration"},
		{"clip without decoration", func() { NewContainer(nil, ContainerOpts{ClipBehavior: ui.ClipHardEdge}) }, "ContainerClipWithoutDecoration"},
		{"negative padding", func() { NewContainer(nil, ContainerOpts{Padding: painting.NewEdgeInsetsAll(-1)}) }, "ContainerNegativePadding"},
		{"negative margin", func() { NewContainer(nil, ContainerOpts{Margin: painting.NewEdgeInsetsAll(-1)}) }, "ContainerNegativePadding"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.do) })
	}
}

func TestContainerClipsToTheDecoration(t *testing.T) {
	ts := gluttertest.New(t)
	deco := painting.NewBoxDecoration(painting.BoxDecorationOpts{Shape: painting.BoxShapeCircle, Color: foundation.Some(ui.NewColor(0xFF0000FF))})
	pumpLoose(ts, NewContainer(NewColoredBox(ui.NewColor(0xFFFF0000), nil), ContainerOpts{
		Width: foundation.Some(100.0), Height: foundation.Some(100.0), Decoration: deco, ClipBehavior: ui.ClipAntiAlias}), ui.TextDirectionLTR)
	if got := pixel(t, ts, 50, 50); got != rgb(255, 0, 0) {
		t.Errorf("center = %v, want the red child", got)
	}
	if got := pixel(t, ts, 2, 2); got != unpainted {
		t.Errorf("corner = %v, want the white of the view: the circle clips the child and the decoration", got)
	}
}

func TestColoredBoxPaints(t *testing.T) {
	ts := gluttertest.New(t)
	pumpLoose(ts, NewSizedBox(NewColoredBox(ui.NewColor(0xFF336699), nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}), ui.TextDirectionLTR)
	if got := pixel(t, ts, 5, 5); got != rgb(0x33, 0x66, 0x99) {
		t.Errorf("inside = %v", got)
	}
	if got := pixel(t, ts, 30, 5); got != unpainted {
		t.Errorf("outside = %v", got)
	}
	// The color changes with the widget.
	pumpLoose(ts, NewSizedBox(NewColoredBox(ui.NewColor(0xFF000000), nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}), ui.TextDirectionLTR)
	if got := pixel(t, ts, 5, 5); got != rgb(0, 0, 0) {
		t.Errorf("after the update = %v", got)
	}
}

func TestOpacityPaints(t *testing.T) {
	tests := []struct {
		name    string
		opacity float64
		want    color.RGBA
	}{
		{"opaque", 1, rgb(255, 0, 0)},
		{"half", 0.5, color.RGBA{128, 0, 0, 128}},
		{"invisible", 0, unpainted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewOpacity(tt.opacity, NewSizedBox(NewColoredBox(ui.NewColor(0xFFFF0000), nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)})), ui.TextDirectionLTR)
			got := pixel(t, ts, 5, 5)
			if d := int(got.R) - int(tt.want.R); d < -2 || d > 2 || int(got.A) < int(tt.want.A)-2 || int(got.A) > int(tt.want.A)+2 {
				t.Errorf("pixel = %v, want about %v", got, tt.want)
			}
		})
	}
}

func TestTransformConstructors(t *testing.T) {
	tests := []struct {
		name  string
		w     *Transform
		point ui.Offset // where the point (10, 0) of the child goes, the child being 20x10 at the origin
		want  ui.Offset
	}{
		{"translate", NewTransformTranslate(ui.Offset{Dx: 5, Dy: 7}, newBox(20, 10)), ui.Offset{Dx: 10}, ui.Offset{Dx: 15, Dy: 7}},
		{"scale", NewTransformScale(newBox(20, 10), TransformScaleOpts{Scale: foundation.Some(2.0)}), ui.Offset{Dx: 10}, ui.Offset{Dx: 10, Dy: -5}},
		{"scale x", NewTransformScale(newBox(20, 10), TransformScaleOpts{ScaleX: foundation.Some(3.0)}), ui.Offset{Dx: 10}, ui.Offset{Dx: 10, Dy: 0}},
		{"flip x", NewTransformFlip(newBox(20, 10), TransformFlipOpts{FlipX: true}), ui.Offset{Dx: 0}, ui.Offset{Dx: 20}},
		{"rotate a quarter turn", NewTransformRotate(1.5707963267948966, newBox(20, 10)), ui.Offset{Dx: 10}, ui.Offset{Dx: 15, Dy: 5}},
		{"matrix", NewTransform(ui.NewMatrix4TranslationValues(1, 2, 0), newBox(20, 10)), ui.Offset{}, ui.Offset{Dx: 1, Dy: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			box := ts.RenderObject(anyBox).(rendering.Box)
			got := box.LocalToGlobal(tt.point, nil)
			if d := got.Sub(tt.want); d.Distance() > 1e-3 {
				t.Errorf("point %v goes to %v, want %v", tt.point, got, tt.want)
			}
		})
	}
}

func TestTransformMisuse(t *testing.T) {
	skipUnlessDebug(t)
	f := foundation.Some(2.0)
	tests := []struct {
		name string
		do   func()
	}{
		{"scale and scaleX", func() { NewTransformScale(nil, TransformScaleOpts{Scale: f, ScaleX: f}) }},
		{"no scale", func() { NewTransformScale(nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, "TransformScaleInvalid", tt.do) })
	}
}

func TestTransformHitTests(t *testing.T) {
	tests := []struct {
		name string
		hit  foundation.Opt[bool]
		want int
	}{
		{"default transforms the hit test", foundation.None[bool](), 1},
		{"not transformed", foundation.Some(false), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			downs := 0
			l := &pointerProbe{onDown: func() { downs++ }}
			pumpLoose(ts, NewTransform(ui.NewMatrix4TranslationValues(300, 0, 0), newPointerProbe(l, newBox(50, 50)), TransformOpts{TransformHitTests: tt.hit}), ui.TextDirectionLTR)
			ts.TapAt(ui.Offset{Dx: 320, Dy: 10})
			if downs != tt.want {
				t.Errorf("downs = %d, want %d", downs, tt.want)
			}
		})
	}
}

// pointerProbe counts the pointer downs that reach its box.
type pointerProbe struct{ onDown func() }

func newPointerProbe(p *pointerProbe, child Widget) Widget {
	return &probeWidget{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child), probe: p}
}

type probeWidget struct {
	SingleChildRenderObjectWidget
	probe *pointerProbe
}

func (w *probeWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := rendering.NewRenderPointerListener(nil, rendering.RenderPointerListenerOpts{Behavior: rendering.HitTestBehaviorOpaque})
	r.OnPointerDown = func(*gestures.PointerDownEvent) { w.probe.onDown() }
	return r
}

func TestIgnoreAndAbsorbPointer(t *testing.T) {
	tests := []struct {
		name string
		wrap func(child Widget) Widget
		want int
	}{
		{"ignoring", func(c Widget) Widget { return NewIgnorePointer(c) }, 0},
		{"not ignoring", func(c Widget) Widget { return NewIgnorePointer(c, IgnorePointerOpts{Ignoring: foundation.Some(false)}) }, 1},
		{"absorbing", func(c Widget) Widget { return NewAbsorbPointer(c) }, 0},
		{"not absorbing", func(c Widget) Widget {
			return NewAbsorbPointer(c, AbsorbPointerOpts{Absorbing: foundation.Some(false)})
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			downs := 0
			pumpLoose(ts, tt.wrap(newPointerProbe(&pointerProbe{onDown: func() { downs++ }}, newBox(50, 50))), ui.TextDirectionLTR)
			ts.TapAt(ui.Offset{Dx: 10, Dy: 10})
			if downs != tt.want {
				t.Errorf("downs = %d, want %d", downs, tt.want)
			}
		})
	}
}

func TestVisibility(t *testing.T) {
	hidden := foundation.Some(false)
	tests := []struct {
		name     string
		mk       func(child Widget) *Visibility
		wantSize ui.Size
		wantBox  bool // whether the child is in the tree
	}{
		{"visible", func(c Widget) *Visibility { return NewVisibility(c) }, ui.Size{Width: 30, Height: 20}, true},
		{"hidden shows the replacement", func(c Widget) *Visibility { return NewVisibility(c, VisibilityOpts{Visible: hidden}) }, ui.Size{}, false},
		{"hidden with a replacement", func(c Widget) *Visibility {
			return NewVisibility(c, VisibilityOpts{Visible: hidden, Replacement: newBox(5, 5)})
		}, ui.Size{Width: 5, Height: 5}, false},
		{"hidden with the state kept", func(c Widget) *Visibility {
			return NewVisibility(c, VisibilityOpts{Visible: hidden, MaintainState: true})
		}, ui.Size{}, true},
		{"hidden with the size kept", func(c Widget) *Visibility {
			return NewVisibility(c, VisibilityOpts{Visible: hidden, MaintainState: true, MaintainAnimation: true, MaintainSize: true})
		}, ui.Size{Width: 30, Height: 20}, true},
		{"maintain", func(c Widget) *Visibility { return NewVisibilityMaintain(c, VisibilityOpts{Visible: hidden}) }, ui.Size{Width: 30, Height: 20}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			child := newBox(30, 20)
			w := tt.mk(child)
			pumpLoose(ts, w, ui.TextDirectionLTR)
			if got := ts.Size(gluttertest.ByWidget(w)); got != tt.wantSize {
				t.Errorf("size = %v, want %v", got, tt.wantSize)
			}
			if got := ts.Exists(gluttertest.ByWidget(child, gluttertest.FindOpts{IncludeOffstage: true})); got != tt.wantBox {
				t.Errorf("child in the tree = %v, want %v", got, tt.wantBox)
			}
		})
	}
}

func TestVisibilityPaintsOnlyWhenVisible(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	tests := []struct {
		name    string
		visible bool
		want    color.RGBA
	}{
		{"visible", true, rgb(255, 0, 0)},
		{"hidden", false, unpainted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewVisibilityMaintain(NewSizedBox(NewColoredBox(red, nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}),
				VisibilityOpts{Visible: foundation.Some(tt.visible)}), ui.TextDirectionLTR)
			if got := pixel(t, ts, 5, 5); got != tt.want {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVisibilityMaintainInteractivity(t *testing.T) {
	tests := []struct {
		name string
		opts VisibilityOpts
		want int
	}{
		{"hidden ignores pointers", VisibilityOpts{MaintainState: true, MaintainAnimation: true, MaintainSize: true}, 0},
		{"interactive when asked", VisibilityOpts{MaintainState: true, MaintainAnimation: true, MaintainSize: true, MaintainInteractivity: true}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			downs := 0
			tt.opts.Visible = foundation.Some(false)
			pumpLoose(ts, NewVisibility(newPointerProbe(&pointerProbe{onDown: func() { downs++ }}, newBox(50, 50)), tt.opts), ui.TextDirectionLTR)
			ts.TapAt(ui.Offset{Dx: 10, Dy: 10})
			if downs != tt.want {
				t.Errorf("downs = %d, want %d", downs, tt.want)
			}
		})
	}
}

// counterBox is a stateful widget that remembers how many times it was built.
type counterBox struct {
	StatefulWidget
	built *int
}

func (w *counterBox) CreateState() State { return &counterBoxState{} }

type counterBoxState struct{ StateOf[*counterBox] }

func (s *counterBoxState) Build(BuildContext) Widget { *s.Widget().built++; return newBox(5, 5) }

func TestVisibilityKeepsTheState(t *testing.T) {
	ts := gluttertest.New(t)
	built := 0
	mk := func(visible bool, opts VisibilityOpts) Widget {
		opts.Visible = foundation.Some(visible)
		return NewVisibility(&counterBox{NewStatefulWidgetBase(), &built}, opts)
	}
	pumpLoose(ts, mk(true, VisibilityOpts{MaintainState: true}), ui.TextDirectionLTR)
	state := gluttertest.StateOf[*counterBoxState](ts, gluttertest.ByType[*counterBox](gluttertest.FindOpts{IncludeOffstage: true}))
	pumpLoose(ts, mk(false, VisibilityOpts{MaintainState: true}), ui.TextDirectionLTR)
	pumpLoose(ts, mk(true, VisibilityOpts{MaintainState: true}), ui.TextDirectionLTR)
	if again := gluttertest.StateOf[*counterBoxState](ts, gluttertest.ByType[*counterBox](gluttertest.FindOpts{IncludeOffstage: true})); again != state {
		t.Errorf("the state was replaced")
	}
	// Without MaintainState the state is lost.
	pumpLoose(ts, mk(false, VisibilityOpts{}), ui.TextDirectionLTR)
	pumpLoose(ts, mk(true, VisibilityOpts{}), ui.TextDirectionLTR)
	if again := gluttertest.StateOf[*counterBoxState](ts, gluttertest.ByType[*counterBox]()); again == state {
		t.Errorf("the state was kept although it was not maintained")
	}
}

func TestVisibilityOf(t *testing.T) {
	ts := gluttertest.New(t)
	var seen []bool
	probe := func() Widget {
		return builderOf(func(bc BuildContext) Widget { seen = append(seen, VisibilityOf(bc)); return newBox(1, 1) })
	}
	hidden := VisibilityOpts{Visible: foundation.Some(false), MaintainState: true}
	shown := VisibilityOpts{MaintainState: true}
	pumpLoose(ts, NewVisibility(NewVisibility(probe(), shown), shown), ui.TextDirectionLTR)
	pumpLoose(ts, NewVisibility(NewVisibility(probe(), shown), hidden), ui.TextDirectionLTR)
	pumpLoose(ts, NewVisibility(NewVisibility(probe(), hidden), shown), ui.TextDirectionLTR)
	pumpLoose(ts, probe(), ui.TextDirectionLTR)
	want := []bool{true, false, false, true}
	if len(seen) < len(want) {
		t.Fatalf("seen = %v", seen)
	}
	for i, w := range want {
		if seen[len(seen)-len(want)+i] != w {
			t.Errorf("seen = %v, want the last four to be %v", seen, want)
			break
		}
	}
}

func TestVisibilityMisuse(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts VisibilityOpts
	}{
		{"animation without state", VisibilityOpts{MaintainAnimation: true}},
		{"size without animation", VisibilityOpts{MaintainState: true, MaintainSize: true}},
		{"semantics without size", VisibilityOpts{MaintainSemantics: true}},
		{"interactivity without size", VisibilityOpts{MaintainInteractivity: true}},
		{"focusability without state", VisibilityOpts{MaintainFocusability: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, "VisibilityMaintainInvalid", func() { NewVisibility(nil, tt.opts) }) })
	}
}

func TestPlaceholder(t *testing.T) {
	tests := []struct {
		name string
		w    Widget
		want ui.Size
	}{
		{"fills a bounded space", NewPlaceholder(nil), ui.Size{Width: 800, Height: 600}},
		{"fallback size when unbounded", NewUnconstrainedBox(NewPlaceholder(nil)), ui.Size{Width: 400, Height: 400}},
		{"custom fallback", NewUnconstrainedBox(NewPlaceholder(nil, PlaceholderOpts{FallbackWidth: foundation.Some(30.0), FallbackHeight: foundation.Some(20.0)})), ui.Size{Width: 30, Height: 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			if got := ts.Size(gluttertest.ByType[*Placeholder]()); got != tt.want {
				t.Errorf("size = %v, want %v", got, tt.want)
			}
			// A fallback larger than the box overflows unpainted, but the unconstrained box may.
			ts.TakeErrors()
		})
	}
}

// recordingPainter records what the custom paint asks it to do.
type recordingPainter struct {
	rendering.CustomPainterBase
	paints  []ui.Size
	repaint bool
	hit     foundation.Opt[bool]
}

func (p *recordingPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.paints = append(p.paints, size)
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColor(0xFF0000FF))
	canvas.DrawRect(ui.OffsetZero.And(size), paint)
}
func (p *recordingPainter) ShouldRepaint(rendering.CustomPainter) bool { return p.repaint }
func (p *recordingPainter) HitTest(ui.Offset) foundation.Opt[bool]     { return p.hit }

func TestCustomPaint(t *testing.T) {
	ts := gluttertest.New(t)
	p := &recordingPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil)}
	pumpLoose(ts, NewCustomPaint(nil, CustomPaintOpts{Painter: p, Size: ui.Size{Width: 40, Height: 30}}), ui.TextDirectionLTR)
	if len(p.paints) != 1 || p.paints[0] != (ui.Size{Width: 40, Height: 30}) {
		t.Errorf("paints = %v", p.paints)
	}
	if got := pixel(t, ts, 10, 10); got != rgb(0, 0, 255) {
		t.Errorf("pixel = %v", got)
	}
	// A new painter that says it need not repaint does not paint again.
	q := &recordingPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil)}
	pumpLoose(ts, NewCustomPaint(nil, CustomPaintOpts{Painter: q, Size: ui.Size{Width: 40, Height: 30}}), ui.TextDirectionLTR)
	if len(q.paints) != 0 {
		t.Errorf("the new painter painted although ShouldRepaint said no: %v", q.paints)
	}
	q.repaint = true
	pumpLoose(ts, NewCustomPaint(nil, CustomPaintOpts{Painter: &recordingPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), repaint: true}, Size: ui.Size{Width: 40, Height: 30}}), ui.TextDirectionLTR)
	// Unmounting clears the painters of the render object.
	ro := ts.RenderObject(gluttertest.ByType[*CustomPaint]()).(*rendering.RenderCustomPaint)
	pumpLoose(ts, newBox(1, 1), ui.TextDirectionLTR)
	if ro.Painter() != nil || ro.ForegroundPainter() != nil {
		t.Errorf("the painters were not cleared when the widget left the tree")
	}
}

func TestRepaintBoundaryWrap(t *testing.T) {
	keyed := newBox(1, 1, WidgetOpts{Key: foundation.NewKey("k")})
	plain := newBox(1, 1)
	wrapped := RepaintBoundaryWrapAll([]Widget{keyed, plain})
	if len(wrapped) != 2 {
		t.Fatalf("wrapped = %v", wrapped)
	}
	if k := wrapped[0].Key(); k != foundation.Key(foundation.NewValueKey[foundation.Key](keyed.Key())) {
		t.Errorf("key of the keyed child = %v", k)
	}
	if k := wrapped[1].Key(); k != foundation.Key(foundation.NewValueKey(1)) {
		t.Errorf("key of the plain child = %v", k)
	}
	ts := gluttertest.New(t)
	pumpLoose(ts, NewRepaintBoundary(newBox(1, 1)), ui.TextDirectionLTR)
	if _, ok := ts.RenderObject(gluttertest.ByType[*RepaintBoundary]()).(*rendering.RenderRepaintBoundary); !ok {
		t.Errorf("no repaint boundary render object")
	}
}

func TestKeyedSubtree(t *testing.T) {
	ts := gluttertest.New(t)
	pumpLoose(ts, NewKeyedSubtree(newBox(7, 8)), ui.TextDirectionLTR)
	if got := ts.Size(anyBox); got != (ui.Size{Width: 7, Height: 8}) {
		t.Errorf("size = %v", got)
	}
	wrapped := KeyedSubtreeWrapList([]Widget{newBox(1, 1), newBox(1, 1, WidgetOpts{Key: foundation.NewKey("k")})}, 5)
	if wrapped[0].Key() != foundation.Key(foundation.NewValueKey(5)) || wrapped[1].Key() == nil {
		t.Errorf("keys = %v, %v", wrapped[0].Key(), wrapped[1].Key())
	}
}

// fixedLayout places the child of a CustomSingleChildLayout.
type fixedLayout struct {
	rendering.SingleChildLayoutDelegateBase
	size, child ui.Size
	at          ui.Offset
	relayout    bool
}

func (d *fixedLayout) GetSize(c rendering.BoxConstraints) ui.Size { return c.Constrain(d.size) }
func (d *fixedLayout) GetConstraintsForChild(rendering.BoxConstraints) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsTight(d.child)
}
func (d *fixedLayout) GetPositionForChild(_, _ ui.Size) ui.Offset              { return d.at }
func (d *fixedLayout) ShouldRelayout(rendering.SingleChildLayoutDelegate) bool { return d.relayout }

func TestCustomSingleChildLayout(t *testing.T) {
	ts := gluttertest.New(t)
	d := &fixedLayout{size: ui.Size{Width: 100, Height: 80}, child: ui.Size{Width: 10, Height: 10}, at: ui.Offset{Dx: 30, Dy: 40}}
	pumpLoose(ts, NewCustomSingleChildLayout(d, newBox(1, 1)), ui.TextDirectionLTR)
	if got := ts.Size(gluttertest.ByType[*CustomSingleChildLayout]()); got != d.size {
		t.Errorf("size = %v", got)
	}
	if got := ts.TopLeft(anyBox); got != (ui.Offset{Dx: 30, Dy: 40}) {
		t.Errorf("child at %v", got)
	}
	// A new delegate that says to relayout moves the child.
	d2 := &fixedLayout{size: d.size, child: d.child, at: ui.Offset{Dx: 1, Dy: 2}, relayout: true}
	pumpLoose(ts, NewCustomSingleChildLayout(d2, newBox(1, 1)), ui.TextDirectionLTR)
	if got := ts.TopLeft(anyBox); got != (ui.Offset{Dx: 1, Dy: 2}) {
		t.Errorf("child at %v after the delegate changed", got)
	}
}

// rowLayout puts the child "b" after the child "a".
type rowLayout struct {
	rendering.MultiChildLayoutDelegateBase
}

func (d *rowLayout) PerformLayout(size ui.Size) {
	a := d.LayoutChild("a", rendering.NewBoxConstraintsLoose(size))
	d.PositionChild("a", ui.Offset{Dx: 5, Dy: 5})
	d.LayoutChild("b", rendering.NewBoxConstraintsLoose(size))
	d.PositionChild("b", ui.Offset{Dx: 5 + a.Width, Dy: 5})
}
func (d *rowLayout) ShouldRelayout(rendering.MultiChildLayoutDelegate) bool { return false }

func TestCustomMultiChildLayout(t *testing.T) {
	ts := gluttertest.New(t)
	d := &rowLayout{rendering.NewMultiChildLayoutDelegateBase(nil)}
	pumpLoose(ts, NewSizedBox(NewCustomMultiChildLayout(d, []Widget{NewLayoutId("b", newBox(20, 10)), NewLayoutId("a", newBox(30, 10))}),
		SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}), ui.TextDirectionLTR)
	if got := rectsOf(ts); !rectsEqual(got, []ui.Rect{rect(35, 5, 20, 10), rect(5, 5, 30, 10)}) {
		t.Errorf("boxes = %v", got)
	}
	if id := NewLayoutId("x", nil); id.ID() != "x" || id.Key() != foundation.Key(foundation.NewValueKey[any]("x")) {
		t.Errorf("layout id = %v key %v", id.ID(), id.Key())
	}
}

// counterScope is an inherited widget that changes on every update.
type counterScope struct{ InheritedWidget }

func newCounterScope(child Widget) *counterScope { return &counterScope{NewInheritedWidgetBase(child)} }

func (w *counterScope) UpdateShouldNotify(Widget) bool { return true }

// TestDeactivatedDependentsAreNotNotified is the regression test of a dependent
// that left the tree but stayed in the list of its inherited element: the next
// update of the element would mark the unmounted dependent dirty.
func TestDeactivatedDependentsAreNotNotified(t *testing.T) {
	ts := gluttertest.New(t)
	builds := 0
	probe := func() Widget {
		return builderOf(func(bc BuildContext) Widget {
			_, _ = DependOn[*counterScope](bc)
			builds++
			return newBox(1, 1)
		})
	}
	pumpLoose(ts, newCounterScope(NewOffstage(probe())), ui.TextDirectionLTR)
	// The probe leaves the tree: the scope keeps its place.
	pumpLoose(ts, newCounterScope(newBox(2, 2)), ui.TextDirectionLTR)
	// The scope updates again and must not touch the unmounted probe.
	pumpLoose(ts, newCounterScope(newBox(3, 3)), ui.TextDirectionLTR)
	if builds != 1 {
		t.Errorf("builds = %d, want 1", builds)
	}
}

type pathClipper struct {
	rendering.CustomClipperBase[*ui.Path]
}

func (pathClipper) GetClip(size ui.Size) *ui.Path {
	p := ui.NewPath()
	p.AddRect(ui.OffsetZero.And(size))
	return p
}
func (pathClipper) ShouldReclip(rendering.CustomClipper[*ui.Path]) bool { return false }

func TestClipAndPaintWidgetsConfigureTheirRenderObjects(t *testing.T) {
	clipper := pathClipper{rendering.NewCustomClipperBase[*ui.Path](nil)}
	black := ui.NewColor(0xFF000000)
	tests := []struct {
		name  string
		w     Widget
		check func(t *testing.T, ro rendering.RenderObject)
	}{
		{"clip rect defaults to hard edge", NewClipRect(newBox(1, 1)), func(t *testing.T, ro rendering.RenderObject) {
			if b := ro.(*rendering.RenderClipRect).ClipBehavior(); b != ui.ClipHardEdge {
				t.Errorf("clip behavior = %v", b)
			}
		}},
		{"clip rrect defaults to anti alias", NewClipRRect(newBox(1, 1), ClipRRectOpts{BorderRadius: painting.NewBorderRadiusCircular(3)}), func(t *testing.T, ro rendering.RenderObject) {
			if b := ro.(*rendering.RenderClipRRect).ClipBehavior(); b != ui.ClipAntiAlias {
				t.Errorf("clip behavior = %v", b)
			}
		}},
		{"clip oval", NewClipOval(newBox(1, 1), ClipOvalOpts{ClipBehavior: ui.ClipHardEdge}), func(t *testing.T, ro rendering.RenderObject) {
			if b := ro.(*rendering.RenderClipOval).ClipBehavior(); b != ui.ClipHardEdge {
				t.Errorf("clip behavior = %v", b)
			}
		}},
		{"clip path with a clipper", NewClipPath(newBox(1, 1), ClipPathOpts{Clipper: clipper}), func(t *testing.T, ro rendering.RenderObject) {
			if ro.(*rendering.RenderClipPath).Clipper() == nil {
				t.Errorf("no clipper")
			}
		}},
		{"clip path with a shape", NewClipPathShape(painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{}), newBox(1, 1)), func(t *testing.T, ro rendering.RenderObject) {
			if _, ok := ro.(*rendering.RenderClipPath).Clipper().(*rendering.ShapeBorderClipper); !ok {
				t.Errorf("the clipper is not a ShapeBorderClipper")
			}
		}},
		{"physical model", NewPhysicalModel(black, newBox(1, 1), PhysicalModelOpts{Elevation: 4}), func(t *testing.T, ro rendering.RenderObject) {
			r := ro.(*rendering.RenderPhysicalModel)
			if r.Elevation() != 4 || r.Color() != black {
				t.Errorf("elevation %v color %v", r.Elevation(), r.Color())
			}
		}},
		{"physical shape", NewPhysicalShape(clipper, black, newBox(1, 1), PhysicalShapeOpts{Elevation: 2}), func(t *testing.T, ro rendering.RenderObject) {
			if r := ro.(*rendering.RenderPhysicalShape); r.Elevation() != 2 || r.Clipper() == nil {
				t.Errorf("elevation %v", r.Elevation())
			}
		}},
		{"decorated box in front", NewDecoratedBox(painting.NewBoxDecoration(), newBox(1, 1), DecoratedBoxOpts{Position: rendering.DecorationPositionForeground}), func(t *testing.T, ro rendering.RenderObject) {
			if p := ro.(*rendering.RenderDecoratedBox).Position(); p != rendering.DecorationPositionForeground {
				t.Errorf("position = %v", p)
			}
		}},
		{"shader mask", NewShaderMask(func(r ui.Rect) ui.Shader {
			return ui.NewGradientLinear(r.TopLeft(), r.BottomRight(), []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)})
		}, newBox(1, 1)), func(t *testing.T, ro rendering.RenderObject) {
			if b := ro.(*rendering.RenderShaderMask).BlendMode(); b != ui.BlendModeModulate {
				t.Errorf("blend mode = %v", b)
			}
		}},
		{"backdrop filter", NewBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}), newBox(1, 1)), func(t *testing.T, ro rendering.RenderObject) {
			r := ro.(*rendering.RenderBackdropFilter)
			if !r.Enabled() || r.BlendMode() != ui.BlendModeSrcOver {
				t.Errorf("enabled %v blend mode %v", r.Enabled(), r.BlendMode())
			}
		}},
		{"opacity keeps the semantics when asked", NewOpacity(0, newBox(1, 1), OpacityOpts{AlwaysIncludeSemantics: true}), func(t *testing.T, ro rendering.RenderObject) {
			if !ro.(*rendering.RenderOpacity).AlwaysIncludeSemantics() {
				t.Errorf("semantics not kept")
			}
		}},
		{"ignore baseline", NewIgnoreBaseline(newBox(1, 1)), func(t *testing.T, ro rendering.RenderObject) {
			if _, ok := ro.(*rendering.RenderIgnoreBaseline); !ok {
				t.Errorf("%T", ro)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			tt.check(t, ts.RenderObject(gluttertest.ByWidget(tt.w)))
		})
	}
}

func TestCreateLocalImageConfiguration(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{DevicePixelRatio: 2})
	var got painting.ImageConfiguration
	pumpLoose(ts, builderOf(func(bc BuildContext) Widget {
		got = CreateLocalImageConfiguration(bc, foundation.Some(ui.Size{Width: 3, Height: 4}))
		return newBox(1, 1)
	}), ui.TextDirectionRTL)
	if r, _ := got.DevicePixelRatio.Get(); r != 2 {
		t.Errorf("pixel ratio = %v", got.DevicePixelRatio)
	}
	if got.TextDirection != ui.TextDirectionRTL || got.Binding != ts.Binding.Painting() || got.Bundle != ts.Binding.Painting().AssetBundle() {
		t.Errorf("configuration = %+v", got)
	}
	if p, _ := got.Platform.Get(); p != foundation.DefaultTargetPlatform() {
		t.Errorf("platform = %v", got.Platform)
	}
}
