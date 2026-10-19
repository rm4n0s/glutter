package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

var (
	rectClipperRect = ui.Rect{}
	pathClipperRect = ui.NewRectFromLTWH(50, 50, 100, 100)
)

// rectClipper is _TestRectClipper.
type rectClipper struct {
	rendering.CustomClipperBase[ui.Rect]
}

func (rectClipper) GetClip(ui.Size) ui.Rect                            { return rectClipperRect }
func (rectClipper) GetApproximateClipRect(ui.Size) ui.Rect             { return rectClipperRect }
func (rectClipper) ShouldReclip(rendering.CustomClipper[ui.Rect]) bool { return true }

// rrectClipper is _TestRRectClipper.
type rrectClipper struct {
	rendering.CustomClipperBase[ui.RRect]
}

func (rrectClipper) GetClip(ui.Size) ui.RRect                            { return ui.RRectZero }
func (rrectClipper) GetApproximateClipRect(ui.Size) ui.Rect              { return ui.RRectZero.OuterRect() }
func (rrectClipper) ShouldReclip(rendering.CustomClipper[ui.RRect]) bool { return true }

// pathClipper is _TestPathClipper.
type pathClipper struct {
	rendering.CustomClipperBase[*ui.Path]
}

func (pathClipper) GetClip(ui.Size) *ui.Path {
	p := ui.NewPath()
	p.AddRect(pathClipperRect)
	return p
}
func (pathClipper) ShouldReclip(rendering.CustomClipper[*ui.Path]) bool { return false }

func TestRenderFittedBoxEmptySize(t *testing.T) {
	h := newHarness(t)
	fitted := rendering.NewRenderFittedBox(rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{
		Painter: newCallbackPainter(func() {})}))
	layout(t, h, fitted, layoutOpts{phase: phaseComposite})
	transform := ui.NewMatrix4Identity()
	fitted.ApplyPaintTransform(fitted.Child(), transform)
	zero := ui.NewMatrix4Identity()
	zero.SetZero()
	if !transform.Equals(zero) {
		t.Errorf("transform of an empty fitted box = %v, want zero", transform)
	}
	if fitted.HitTestChildren(rendering.NewBoxHitTestResult(), ui.OffsetZero) {
		t.Error("an empty fitted box was hit")
	}
}

func TestRenderFittedBoxDoesNotPaintEmpty(t *testing.T) {
	tests := []struct {
		name        string
		preferred   ui.Size
		constraints foundation.Opt[rendering.BoxConstraints]
		want        bool
	}{
		{"both sizes nonempty", ui.Size{Width: 1, Height: 1}, foundation.None[rendering.BoxConstraints](), true},
		{"empty child", ui.Size{}, foundation.None[rendering.BoxConstraints](), false},
		{"empty box", ui.Size{Width: 1, Height: 1}, foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{})), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			painted := false
			fitted := rendering.NewRenderFittedBox(rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{
				PreferredSize: tc.preferred, Painter: newCallbackPainter(func() { painted = true })}))
			layout(t, newHarness(t), fitted, layoutOpts{constraints: tc.constraints, phase: phasePaint})
			if painted != tc.want {
				t.Errorf("painted = %v, want %v", painted, tc.want)
			}
		})
	}
}

func TestRenderPhysicalModelCompositing(t *testing.T) {
	h := newHarness(t)
	root := rendering.NewRenderPhysicalModel(ui.NewColor(0xffff00ff), nil)
	layout(t, h, root, layoutOpts{phase: phaseComposite})
	for _, elevation := range []float64{0, 1, 0} {
		root.SetElevation(elevation)
		h.Pump()
		if root.NeedsCompositing() {
			t.Errorf("a physical model with elevation %v needs compositing", elevation)
		}
	}
}

func TestRenderPhysicalShape(t *testing.T) {
	newRoot := func() *rendering.RenderPhysicalShape {
		return rendering.NewRenderPhysicalShape(rendering.NewShapeBorderClipper(painting.NewCircleBorder()), ui.NewColor(0xffff00ff), nil)
	}
	t.Run("shape change triggers repaint", func(t *testing.T) {
		skipUnlessDebug(t)
		h := newHarness(t)
		root := newRoot()
		layout(t, h, root, layoutOpts{phase: phaseComposite})
		if root.DebugNeedsPaint() {
			t.Fatal("needs paint after the frame")
		}
		root.SetClipper(rendering.NewShapeBorderClipper(painting.NewCircleBorder()))
		if root.DebugNeedsPaint() {
			t.Error("the same shape marked the box for repaint")
		}
		root.SetClipper(rendering.NewShapeBorderClipper(painting.NewStadiumBorder()))
		if !root.DebugNeedsPaint() {
			t.Error("a different shape did not mark the box for repaint")
		}
	})
	t.Run("compositing", func(t *testing.T) {
		h := newHarness(t)
		root := newRoot()
		layout(t, h, root, layoutOpts{phase: phaseComposite})
		for _, elevation := range []float64{0, 1, 0} {
			root.SetElevation(elevation)
			h.Pump()
			if root.NeedsCompositing() {
				t.Errorf("elevation %v: needs compositing", elevation)
			}
		}
	})
}

func pixelOf(t *testing.T, img *ui.Image, x, y int) uint32 {
	t.Helper()
	p := img.Pixels()
	i := (y*img.Width() + x) * 4
	return uint32(p[i])<<24 | uint32(p[i+1])<<16 | uint32(p[i+2])<<8 | uint32(p[i+3])
}

func TestRenderRepaintBoundaryImages(t *testing.T) {
	opts := ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}}
	empty := func(ratio float64, sync bool) *ui.Image {
		h := newHarness(t)
		boundary := rendering.NewRenderRepaintBoundary(nil)
		layout(t, h, boundary, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 200})), phaseComposite))
		var img *ui.Image
		var err error
		if sync {
			img, err = boundary.ToImageSync(ratio, opts)
		} else {
			img, err = boundary.ToImage(ratio, opts)
		}
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	for _, sync := range []bool{false, true} {
		for _, tc := range []struct {
			ratio float64
			w, h  int
		}{{1, 100, 200}, {2, 200, 400}} {
			img := empty(tc.ratio, sync)
			if img.Width() != tc.w || img.Height() != tc.h {
				t.Errorf("sync=%v ratio=%v: image is %dx%d, want %dx%d", sync, tc.ratio, img.Width(), img.Height(), tc.w, tc.h)
			}
		}
	}

	// Two children, one of them in an opacity layer.
	h := newHarness(t)
	black := rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xff000000))}),
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 20, Height: 20}), nil))
	white := rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xffffffff))}),
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}), nil))
	positioned := rendering.NewRenderPositionedBox(white, rendering.RenderPositionedBoxOpts{
		WidthFactor: foundation.Some(2.0), HeightFactor: foundation.Some(2.0), Alignment: painting.AlignmentTopRight})
	boundary := rendering.NewRenderRepaintBoundary(newTStack(
		rendering.NewRenderOpacity(black, rendering.RenderOpacityOpts{Opacity: foundation.Some(0.5)}), positioned))
	layout(t, h, boundary, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 20, Height: 20})), phaseComposite))
	for _, sync := range []bool{false, true} {
		var img *ui.Image
		var err error
		if sync {
			img, err = boundary.ToImageSync(1, opts)
		} else {
			img, err = boundary.ToImage(1, opts)
		}
		if err != nil {
			t.Fatal(err)
		}
		if img.Width() != 20 || img.Height() != 20 {
			t.Fatalf("image is %dx%d, want 20x20", img.Width(), img.Height())
		}
		if got := pixelOf(t, img, 0, 0); got != 0x00000080 {
			t.Errorf("sync=%v: pixel (0,0) = %08x, want 00000080", sync, got)
		}
		if got := pixelOf(t, img, 19, 0); got != 0xffffffff {
			t.Errorf("sync=%v: pixel (19,0) = %08x, want ffffffff", sync, got)
		}
	}
	layer := boundary.Layer().(rendering.OffsetLayerLike).AsOffsetLayer()
	img, err := layer.ToImage(ui.NewRectFromLTWH(-10, -10, 30, 30), 1, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []struct {
		x, y int
		want uint32
	}{{0, 0, 0}, {10, 10, 0x00000080}, {29, 0, 0}, {29, 10, 0xffffffff}} {
		if got := pixelOf(t, img, p.x, p.y); got != p.want {
			t.Errorf("shifted: pixel (%d,%d) = %08x, want %08x", p.x, p.y, got, p.want)
		}
	}
	img, err = layer.ToImage(ui.NewRectFromLTWH(-10, -10, 30, 30), 2, opts)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width() != 60 || img.Height() != 60 {
		t.Errorf("image at ratio 2 is %dx%d, want 60x60", img.Width(), img.Height())
	}
	for _, p := range []struct {
		x, y int
		want uint32
	}{{0, 0, 0}, {20, 20, 0x00000080}, {59, 0, 0}, {59, 20, 0xffffffff}} {
		if got := pixelOf(t, img, p.x, p.y); got != p.want {
			t.Errorf("ratio 2: pixel (%d,%d) = %08x, want %08x", p.x, p.y, got, p.want)
		}
	}
}

func TestRenderOpacityCompositing(t *testing.T) {
	tests := []struct {
		name    string
		opacity float64
		want    bool
	}{{"transparent", 0, false}, {"opaque", 1, true}, {"partially opaque", 0.1, true}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := rendering.NewRenderOpacity(newSizedBox(1, 1), rendering.RenderOpacityOpts{Opacity: foundation.Some(tc.opacity)})
			layout(t, newHarness(t), o, layoutOpts{phase: phaseComposite})
			if o.NeedsCompositing() != tc.want {
				t.Errorf("needs compositing = %v, want %v", o.NeedsCompositing(), tc.want)
			}
		})
	}
}

func TestRenderObjectsReuseTheirLayer(t *testing.T) {
	sized := func() rendering.Box { return newSizedBox(1, 1) }
	boundary := func() rendering.Box { return rendering.NewRenderRepaintBoundary(sized()) }
	t.Run("RenderOpacity", func(t *testing.T) {
		testLayerReuse[*rendering.OpacityLayer](t, rendering.NewRenderOpacity(boundary(), rendering.RenderOpacityOpts{Opacity: foundation.Some(0.5)}))
	})
	t.Run("RenderShaderMask", func(t *testing.T) {
		testLayerReuse[*rendering.ShaderMaskLayer](t, rendering.NewRenderShaderMask(func(r ui.Rect) ui.Shader {
			return ui.NewGradientRadial(r.Center(), r.ShortestSide()/2, []ui.Color{ui.NewColor(0xff000000), ui.NewColor(0xffffffff)})
		}, sized()))
	})
	t.Run("RenderBackdropFilter", func(t *testing.T) {
		testLayerReuse[*rendering.BackdropFilterLayer](t, rendering.NewRenderBackdropFilter(ui.NewImageFilterBlur(), sized()))
	})
	t.Run("RenderClipRect", func(t *testing.T) {
		testLayerReuse[*rendering.ClipRectLayer](t, rendering.NewRenderClipRect(boundary(), rendering.RenderClipRectOpts{Clipper: rectClipper{}}))
	})
	t.Run("RenderClipRRect", func(t *testing.T) {
		testLayerReuse[*rendering.ClipRRectLayer](t, rendering.NewRenderClipRRect(boundary(), rendering.RenderClipRRectOpts{Clipper: rrectClipper{}}))
	})
	t.Run("RenderClipOval", func(t *testing.T) {
		testLayerReuse[*rendering.ClipPathLayer](t, rendering.NewRenderClipOval(boundary(), rendering.RenderClipRectOpts{Clipper: rectClipper{}}))
	})
	t.Run("RenderClipPath", func(t *testing.T) {
		testLayerReuse[*rendering.ClipPathLayer](t, rendering.NewRenderClipPath(boundary(), rendering.RenderClipPathOpts{Clipper: pathClipper{}}))
	})
	t.Run("RenderPhysicalModel", func(t *testing.T) {
		testLayerReuse[*rendering.ClipRRectLayer](t, rendering.NewRenderPhysicalModel(ui.NewColor(0xff000000), boundary(),
			rendering.RenderPhysicalModelOpts{ClipBehavior: ui.ClipHardEdge}))
	})
	t.Run("RenderPhysicalShape", func(t *testing.T) {
		testLayerReuse[*rendering.ClipPathLayer](t, rendering.NewRenderPhysicalShape(pathClipper{}, ui.NewColor(0xff000000), boundary(),
			rendering.RenderPhysicalModelOpts{ClipBehavior: ui.ClipHardEdge}))
	})
	t.Run("RenderTransform", func(t *testing.T) {
		testLayerReuse[*rendering.TransformLayer](t, rendering.NewRenderTransform(ui.NewMatrix4RotationX(0.1), boundary()))
	})
	t.Run("RenderFittedBox clip", func(t *testing.T) {
		testLayerReuse[*rendering.ClipRectLayer](t, rendering.NewRenderFittedBox(rendering.NewRenderRepaintBoundary(newSizedBox(100, 200)),
			rendering.RenderFittedBoxOpts{Fit: painting.BoxFitCover, ClipBehavior: ui.ClipHardEdge}))
	})
	t.Run("RenderFittedBox transform", func(t *testing.T) {
		testLayerReuse[*rendering.TransformLayer](t, rendering.NewRenderFittedBox(rendering.NewRenderRepaintBoundary(newSizedBox(1, 1)),
			rendering.RenderFittedBoxOpts{Fit: painting.BoxFitFill}))
	})
}

func TestRenderFittedBoxRespectsClipBehavior(t *testing.T) {
	viewport := rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}
	tests := []struct {
		clip ui.Clip
		want ui.Clip
	}{
		{ui.ClipUnspecified, ui.ClipNone},
		{ui.ClipNone, ui.ClipNone},
		{ui.ClipHardEdge, ui.ClipHardEdge},
		{ui.ClipAntiAlias, ui.ClipAntiAlias},
		{ui.ClipAntiAliasWithSaveLayer, ui.ClipAntiAliasWithSaveLayer},
	}
	for _, tc := range tests {
		t.Run(tc.clip.String(), func(t *testing.T) {
			box := rendering.NewRenderFittedBox(rendering.NewRenderRepaintBoundary(newSizedBox(200, 200)),
				rendering.RenderFittedBoxOpts{Fit: painting.BoxFitNone, ClipBehavior: tc.clip})
			layout(t, newHarness(t), box, atPhase(withConstraints(viewport), phaseComposite))
			if box.ClipBehavior() != tc.want {
				t.Errorf("clip behavior = %v, want %v", box.ClipBehavior(), tc.want)
			}
			clip, isClip := box.Layer().(*rendering.ClipRectLayer)
			if (tc.want != ui.ClipNone) != isClip {
				t.Fatalf("layer is %T for clip %v", box.Layer(), tc.want)
			}
			if isClip && clip.ClipBehavior() != tc.want {
				t.Errorf("layer clip behavior = %v, want %v", clip.ClipBehavior(), tc.want)
			}
		})
	}
}

func TestRenderMouseRegionChangesPropertiesWhenDetached(t *testing.T) {
	r := rendering.NewRenderMouseRegion(nil)
	r.SetOpaque(false)
	r.SetOnEnter(func(*gestures.PointerEnterEvent) {})
	r.SetOnExit(func(*gestures.PointerExitEvent) {})
	r.SetOnHover(func(*gestures.PointerHoverEvent) {})
}

func TestRenderFollowerLayerHitTest(t *testing.T) {
	tests := []struct {
		name             string
		leader           bool
		showWhenUnlinked bool
		want             bool
	}{
		{"no leader, shown when unlinked", false, true, true},
		{"no leader, hidden when unlinked", false, false, false},
		{"leader, shown when unlinked", true, true, true},
		{"leader, hidden when unlinked", true, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			link := rendering.NewLayerLink()
			if tc.leader {
				rendering.NewLeaderLayer(link).Attach(new(int))
			}
			follower := rendering.NewRenderFollowerLayer(link, newSizedBox(1, 1),
				rendering.RenderFollowerLayerOpts{ShowWhenUnlinked: foundation.Some(tc.showWhenUnlinked)})
			layout(t, newHarness(t), follower, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 200})))
			if got := follower.HitTest(rendering.NewBoxHitTestResult(), ui.OffsetZero); got != tc.want {
				t.Errorf("hit test = %v, want %v", got, tc.want)
			}
		})
	}
}

// condRepaintBoundary is ConditionalRepaintBoundary of proxy_box_test.dart.
type condRepaintBoundary struct {
	rendering.RenderProxyBox
	boundary   bool
	factory    func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike
	paintCount int
}

func newCondRepaintBoundary(boundary bool, child rendering.Box) *condRepaintBoundary {
	c := &condRepaintBoundary{boundary: boundary}
	rendering.InitRenderProxyBox(&c.RenderProxyBox, c, child)
	return c
}

func (c *condRepaintBoundary) IsRepaintBoundary() bool { return c.boundary }
func (c *condRepaintBoundary) UpdateCompositedLayer(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
	if c.factory != nil {
		return c.factory(old)
	}
	return c.RenderProxyBox.UpdateCompositedLayer(old)
}
func (c *condRepaintBoundary) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	c.paintCount++
	c.RenderProxyBox.Paint(ctx, offset)
}

// testOffsetLayerA is TestOffsetLayerA.
type testOffsetLayerA struct{ rendering.OffsetLayer }

func newTestOffsetLayerA() *testOffsetLayerA {
	l := &testOffsetLayerA{}
	rendering.InitOffsetLayer(&l.OffsetLayer, l)
	return l
}

func TestRenderObjectCanBecomeRepaintBoundary(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	child := newCondRepaintBoundary(false, nil)
	box := newCondRepaintBoundary(false, child)
	layout(t, h, box, layoutOpts{phase: phaseComposite})
	if child.paintCount != 1 || box.paintCount != 1 {
		t.Fatalf("paint counts %d, %d; want 1, 1", child.paintCount, box.paintCount)
	}
	box.boundary = true
	box.MarkNeedsCompositingBitsUpdate()
	box.MarkNeedsCompositedLayerUpdate()
	h.Pump()
	// The first time the render object becomes a repaint boundary it repaints
	// from the parent, to allow the layer to be created.
	if child.paintCount != 2 || box.paintCount != 2 {
		t.Errorf("paint counts %d, %d; want 2, 2", child.paintCount, box.paintCount)
	}
	if _, ok := box.DebugLayer().(rendering.OffsetLayerLike); !ok || box.DebugLayer() == nil {
		t.Errorf("layer is %T, want an offset layer", box.DebugLayer())
	}
	box.MarkNeedsCompositedLayerUpdate()
	if box.DebugNeedsPaint() || !box.DebugNeedsCompositedLayerUpdate() {
		t.Error("a layer update should not need paint")
	}
	h.Pump()
	// The second time the layer exists and paint is skipped.
	if child.paintCount != 2 || box.paintCount != 2 {
		t.Errorf("paint counts %d, %d; want 2, 2", child.paintCount, box.paintCount)
	}
	box.boundary = false
	box.MarkNeedsCompositingBitsUpdate()
	h.Pump()
	// It must repaint to remove the layer.
	if child.paintCount != 3 || box.paintCount != 3 || box.DebugLayer() != nil {
		t.Errorf("paint counts %d, %d; layer %v; want 3, 3, nil", child.paintCount, box.paintCount, box.DebugLayer())
	}
	box.MarkNeedsCompositedLayerUpdate()
	if !box.DebugNeedsPaint() || !box.DebugNeedsCompositedLayerUpdate() {
		t.Error("a layer update of a non-boundary should be a paint")
	}
}

func TestRenderObjectBoundaryLayerReplacementPanics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		setup func(child *condRepaintBoundary)
		mark  func(child *condRepaintBoundary)
		tag   string
	}{
		{"replaced during layer property update",
			func(c *condRepaintBoundary) {
				c.factory = func(rendering.OffsetLayerLike) rendering.OffsetLayerLike { return newTestOffsetLayerA() }
			},
			func(c *condRepaintBoundary) { c.MarkNeedsCompositedLayerUpdate() }, "UpdateCompositedLayerNewInstance"},
		{"replaced during painting",
			func(c *condRepaintBoundary) {
				c.factory = func(rendering.OffsetLayerLike) rendering.OffsetLayerLike { return newTestOffsetLayerA() }
			},
			func(c *condRepaintBoundary) { c.MarkNeedsPaint() }, "UpdateCompositedLayerNewInstance"},
		{"updates its own offset",
			func(c *condRepaintBoundary) {
				c.factory = func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
					if old == nil {
						old = newTestOffsetLayerA()
					}
					old.AsOffsetLayer().SetOffset(ui.Offset{Dx: 2133, Dy: 4422})
					return old
				}
			},
			func(c *condRepaintBoundary) { c.MarkNeedsPaint() }, "UpdateCompositedLayerOffsetChanged"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			child := newCondRepaintBoundary(true, nil)
			box := newCondRepaintBoundary(false, child)
			// The first frame creates the layer, so the factory must accept it.
			first := true
			orig := tc.setup
			orig(child)
			f := child.factory
			child.factory = func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
				if first {
					return child.RenderProxyBox.UpdateCompositedLayer(old)
				}
				return f(old)
			}
			layout(t, h, box, layoutOpts{phase: phaseComposite})
			first = false
			tc.mark(child)
			requireTag(t, tc.tag, func() { h.Pump() })
		})
	}
}

func TestRenderObjectStopsBeingRepaintBoundaryWhileMarked(t *testing.T) {
	tests := []struct {
		name string
		mark func(child *condRepaintBoundary)
	}{
		{"paint, boundary off, bits", func(c *condRepaintBoundary) {
			c.MarkNeedsPaint()
			c.boundary = false
			c.MarkNeedsCompositingBitsUpdate()
		}},
		{"boundary off, bits, paint", func(c *condRepaintBoundary) {
			c.boundary = false
			c.MarkNeedsCompositingBitsUpdate()
			c.MarkNeedsPaint()
		}},
		{"boundary off, layer update, bits", func(c *condRepaintBoundary) {
			c.boundary = false
			c.MarkNeedsCompositedLayerUpdate()
			c.MarkNeedsCompositingBitsUpdate()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			child := newCondRepaintBoundary(true, nil)
			box := newCondRepaintBoundary(false, child)
			child.factory = func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
				if old != nil {
					return old
				}
				return newTestOffsetLayerA()
			}
			layout(t, h, box, layoutOpts{phase: phaseComposite})
			tc.mark(child)
			h.Pump()
		})
	}
}

func TestPaintsChild(t *testing.T) {
	t.Run("Offstage", func(t *testing.T) {
		box := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(20.0)}), nil)
		parent := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(20.0)}), nil)
		offstage := rendering.NewRenderOffstage(box, rendering.RenderOffstageOpts{Offstage: foundation.Some(false)})
		parent.SetChild(offstage)
		if !offstage.PaintsChild(box) {
			t.Error("an onstage box does not paint its child")
		}
		offstage.SetOffstage(true)
		if offstage.PaintsChild(box) {
			t.Error("an offstage box paints its child")
		}
	})
	t.Run("Opacity", func(t *testing.T) {
		box := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(20.0)}), nil)
		opacity := rendering.NewRenderOpacity(box)
		if !opacity.PaintsChild(box) {
			t.Error("an opaque box does not paint its child")
		}
		opacity.SetOpacity(0)
		if opacity.PaintsChild(box) {
			t.Error("a transparent box paints its child")
		}
	})
}

func TestRenderClipDescribeApproximatePaintClip(t *testing.T) {
	child := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(
		rendering.TightForOpts{Width: foundation.Some(200.0), Height: foundation.Some(200.0)}), nil)
	clip := rendering.NewRenderClipRect(child, rendering.RenderClipRectOpts{ClipBehavior: ui.ClipNone})
	layout(t, newHarness(t), clip)
	if got := clip.DescribeApproximatePaintClip(child); got.IsSome() {
		t.Errorf("clip none: %v, want none", got)
	}
	for _, behavior := range []ui.Clip{ui.ClipHardEdge, ui.ClipAntiAlias, ui.ClipAntiAliasWithSaveLayer} {
		clip.SetClipBehavior(behavior)
		if got := clip.DescribeApproximatePaintClip(child); got != foundation.Some(ui.OffsetZero.And(clip.Size())) {
			t.Errorf("%v: %v, want the box", behavior, got)
		}
	}
}

func TestRenderClipDebugPaintSize(t *testing.T) {
	skipUnlessDebug(t)
	newChild := func() rendering.Box {
		return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(
			rendering.TightForOpts{Width: foundation.Some(200.0), Height: foundation.Some(200.0)}), nil)
	}
	tests := []struct {
		name   string
		make   func(clip ui.Clip) rendering.Box
		opName string
	}{
		{"RenderClipPath", func(c ui.Clip) rendering.Box {
			return rendering.NewRenderClipPath(newChild(), rendering.RenderClipPathOpts{ClipBehavior: c})
		}, "OpDrawPath"},
		{"RenderClipRect", func(c ui.Clip) rendering.Box {
			return rendering.NewRenderClipRect(newChild(), rendering.RenderClipRectOpts{ClipBehavior: c})
		}, "OpDrawRect"},
		{"RenderClipRRect", func(c ui.Clip) rendering.Box {
			return rendering.NewRenderClipRRect(newChild(), rendering.RenderClipRRectOpts{ClipBehavior: c})
		}, "OpDrawRRect"},
		{"RenderClipOval", func(c ui.Clip) rendering.Box {
			return rendering.NewRenderClipOval(newChild(), rendering.RenderClipRectOpts{ClipBehavior: c})
		}, "OpDrawPath"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Debug paint draws the clip when the clip behavior is not none. The
			// size outline of the box is a DrawRect too, so count the difference.
			with := countOf(debugPaintOps(t, tc.make(ui.ClipHardEdge)), tc.opName)
			without := countOf(debugPaintOps(t, tc.make(ui.ClipNone)), tc.opName)
			if with != without+1 {
				t.Errorf("%s drawn %d times with a clip and %d without, want one more", tc.opName, with, without)
			}
		})
	}
}

func TestRenderProxyBoxDryLayoutWithoutChild(t *testing.T) {
	box := rendering.NewRenderProxyBox(nil)
	if got := box.ComputeDryLayout(rendering.BoxConstraints{MinHeight: 8, MaxWidth: 100, MaxHeight: 100}); got != (ui.Size{Width: 0, Height: 8}) {
		t.Errorf("dry layout = %v, want 0x8", got)
	}
}
