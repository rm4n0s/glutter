package gles

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

func blurFilter(sx, sy float64, tm ui.TileMode) ui.ImageFilter {
	return ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: sx, SigmaY: sy, TileMode: tm})
}

// callOps lists the operations of the frame's draw calls, with the pipeline of draws.
func callOps(f *frame) []string {
	var out []string
	for _, c := range f.calls {
		switch c.op {
		case opBeginLayer:
			out = append(out, "begin")
		case opComposite:
			out = append(out, "composite")
		default:
			out = append(out, programNames[c.pipe])
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBlurPasses(t *testing.T) {
	tests := []struct {
		name   string
		sx, sy float64
		tile   ui.TileMode
		want   []passKind
		// sizes are the output sizes of the passes.
		sizes [][2]int
	}{
		{"none", 0, 0.001, ui.TileModeDecal, nil, nil},
		{"small", 2, 3, ui.TileModeClamp, []passKind{passBlur, passBlur}, [][2]int{{100, 60}, {100, 60}}},
		{"x only", 3, 0, ui.TileModeClamp, []passKind{passBlur}, [][2]int{{100, 60}}},
		{"y only", 0, 3, ui.TileModeClamp, []passKind{passBlur}, [][2]int{{100, 60}}},
		{"large x is made on a narrower picture", 20, 2, ui.TileModeMirror,
			[]passKind{passDown, passBlur, passBlur, passUp}, [][2]int{{25, 60}, {25, 60}, {25, 60}, {100, 60}}},
		{"large both", 40, 40, ui.TileModeDecal,
			[]passKind{passDown, passBlur, passBlur, passUp}, [][2]int{{13, 8}, {13, 8}, {13, 8}, {100, 60}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blurPasses(tt.sx, tt.sy, tt.tile, 100, 60)
			if len(got) != len(tt.want) {
				t.Fatalf("passes = %d, want %d", len(got), len(tt.want))
			}
			for i, p := range got {
				if p.kind != tt.want[i] || [2]int{p.outW, p.outH} != tt.sizes[i] {
					t.Errorf("pass %d = kind %d size %dx%d, want kind %d size %v", i, p.kind, p.outW, p.outH, tt.want[i], tt.sizes[i])
				}
				if p.kind == passBlur {
					// The kernel is 3 sigma wide on each side and sums to 1.
					var sum float64
					for i := -int(p.radius); i <= int(p.radius); i++ {
						sum += math.Exp(-float64(i*i)/(2*float64(p.sigma)*float64(p.sigma))) * float64(p.norm)
					}
					if math.Abs(sum-1) > 1e-5 || p.radius != int32(math.Ceil(3*float64(p.sigma))) || p.sigma > maxLowSigma {
						t.Errorf("kernel of pass %d: sigma %v radius %d sums to %v", i, p.sigma, p.radius, sum)
					}
					if p.tile != gpu.TileIndex(tt.tile) {
						t.Errorf("tile = %d", p.tile)
					}
				}
			}
		})
	}
	// The second axis of a blur goes along y.
	if p := blurPasses(2, 2, ui.TileModeClamp, 10, 10); p[0].axis != 0 || p[1].axis != 1 {
		t.Errorf("axes = %d, %d", p[0].axis, p[1].axis)
	}
}

func TestFilterPassesOfEveryFilter(t *testing.T) {
	ctm := affine2D{A: 2, D: 2}
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(0.5)
	singular := ui.NewMatrix4Diagonal3Values(0, 1, 1)
	tests := []struct {
		name   string
		filter ui.ImageFilter
		want   []passKind
	}{
		{"blur", blurFilter(2, 3, ui.TileModeClamp), []passKind{passBlur, passBlur}},
		{"dilate", ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 3, RadiusY: 0}), []passKind{passMorph}},
		{"erode", ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 2}), []passKind{passMorph, passMorph}},
		{"matrix", ui.NewImageFilterMatrix(rot), []passKind{passMatrix}},
		{"compose", ui.NewImageFilterCompose(blurFilter(1, 1, ui.TileModeClamp), ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 1})),
			[]passKind{passMorph, passMorph, passBlur, passBlur}},
		{"color", ui.NewColorFilterSaturation(0.5), []passKind{passColor}},
		{"singular matrix", ui.NewImageFilterMatrix(singular), []passKind{passMatrix}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := filterPasses(tt.filter, ctm, ui.TileModeDecal, 40, 30)
			if !ok || len(got) != len(tt.want) {
				t.Fatalf("passes = %d (ok %v), want %d", len(got), ok, len(tt.want))
			}
			for i, p := range got {
				if p.kind != tt.want[i] {
					t.Errorf("pass %d kind = %d, want %d", i, p.kind, tt.want[i])
				}
			}
		})
	}
	if _, ok := filterPasses(nil, ctm, ui.TileModeDecal, 40, 30); ok {
		t.Error("a nil filter was accepted")
	}
	// The radii are in pixels: the transform's scale applies, and a blur without a
	// tile mode gets the mode it is given.
	got, _ := filterPasses(blurFilter(2, 2, ui.TileModeUnspecified), ctm, ui.TileModeMirror, 40, 30)
	if got[0].sigma != 4 || got[0].tile != gpu.TileIndex(ui.TileModeMirror) {
		t.Errorf("blur pass = sigma %v tile %d", got[0].sigma, got[0].tile)
	}
	if got, _ := filterPasses(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 1.4, RadiusY: 0}), ctm, ui.TileModeDecal, 40, 30); got[0].radius != 3 {
		t.Errorf("dilate radius = %d, want 1.4 * 2 rounded", got[0].radius)
	}
	// A matrix filter that cannot be inverted draws nothing; a nearest one is nearest.
	if got, _ := filterPasses(ui.NewImageFilterMatrix(singular), ctm, ui.TileModeDecal, 4, 4); got[0].quality != 2 {
		t.Errorf("singular quality = %d", got[0].quality)
	}
	if got, _ := filterPasses(ui.NewImageFilterMatrix(rot, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}), ctm, ui.TileModeDecal, 4, 4); got[0].quality != 1 {
		t.Errorf("nearest quality = %d", got[0].quality)
	}
}

func TestMaskFilterOnARectIsOneShadowInstance(t *testing.T) {
	p := solidPaint(0xFF204080)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleOuter, 4))
	b := mustBuild(t, func(c *ui.Canvas) {
		c.Translate(10, 5)
		c.Scale(2, 1)
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 20, 30, 20), 6, 4), p)
	})
	if b.frame.items(pipeShape) != 1 || len(b.frame.calls) != 1 || b.stats.Drawn != 1 {
		t.Fatalf("instances %d, calls %v", b.frame.items(pipeShape), callOps(&b.frame))
	}
	in := b.frame.data[pipeShape]
	// The rectangle is in pixels: center, half size, then the corners' radii, which
	// the scale has stretched; the deviation grows with the transform's scale.
	want := map[int]float32{20: 10 + 2*25, 21: 5 + 30, 22: 30, 23: 10, 24: 12, 25: 4, 44: float32(4 * math.Sqrt2), 45: 2}
	for i, v := range want {
		if math.Abs(float64(in[i]-v)) > 1e-4 {
			t.Errorf("instance[%d] = %v, want %v", i, in[i], v)
		}
	}
	if in[10] != kindShadow {
		t.Errorf("kind = %v", in[10])
	}
	// The quad reaches 3 sigma beyond the shape.
	if in[0] > 10+2*10-3*float32(4*math.Sqrt2) || in[3] < 5+40+3*float32(4*math.Sqrt2) {
		t.Errorf("quad = %v", in[:4])
	}
}

func TestShadowCornersFollowFlips(t *testing.T) {
	p := solidPaint(0xFF000000)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 2))
	rr := ui.NewRRectFromRectAndCorners(ui.NewRectFromLTWH(20, 20, 40, 30), ui.Corners{TopLeft: ui.Radius{X: 8, Y: 6}, BottomRight: ui.Radius{X: 3, Y: 2}})
	tests := []struct {
		name         string
		sx, sy       float64
		topLeftIndex int // which corner (clockwise from the top left) the top left radius lands in
		botRightAt   int
	}{
		{"as drawn", 1, 1, 0, 2},
		{"flipped in x", -1, 1, 1, 3},
		{"flipped in y", 1, -1, 3, 1},
		{"flipped in both", -1, -1, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := mustBuild(t, func(c *ui.Canvas) {
				c.Translate(100, 80)
				c.Scale(tt.sx, tt.sy)
				c.DrawRRect(rr, p)
			})
			in := b.frame.data[pipeShape]
			if b.frame.items(pipeShape) != 1 || in[10] != kindShadow {
				t.Fatalf("calls %v", callOps(&b.frame))
			}
			if in[24+2*tt.topLeftIndex] != 8 || in[25+2*tt.topLeftIndex] != 6 || in[24+2*tt.botRightAt] != 3 || in[25+2*tt.botRightAt] != 2 {
				t.Errorf("radii = %v", in[24:32])
			}
		})
	}
}

func TestMaskFilterOnOtherShapesUsesABlurredLayer(t *testing.T) {
	res, _, _ := newTestResources(t)
	blurred := func(style ui.BlurStyle, sigma float64) *ui.Paint {
		p := solidPaint(0xFF204080)
		p.SetMaskFilter(ui.NewMaskFilterBlur(style, sigma))
		return p
	}
	tests := []struct {
		name  string
		inner string // the pipeline that draws the shape into the layer
		draw  func(c *ui.Canvas)
		style int32
		sigma float64
	}{
		{"path", "mask", func(c *ui.Canvas) { c.DrawPath(wave(), blurred(ui.BlurStyleSolid, 3)) }, 1, 3},
		{"stroked rrect", "shape", func(c *ui.Canvas) {
			p := blurred(ui.BlurStyleNormal, 2)
			p.SetStyle(ui.PaintingStyleStroke)
			p.SetStrokeWidth(4)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(30, 30, 40, 30), 5, 5), p)
		}, 0, 2},
		{"rotated rrect", "shape", func(c *ui.Canvas) {
			c.Rotate(0.4)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(30, 30, 40, 30), 5, 5), blurred(ui.BlurStyleInner, 2))
		}, 3, 2},
		{"scaled arc", "mask", func(c *ui.Canvas) {
			c.Scale(2, 2)
			c.DrawArc(ui.NewRectFromLTWH(10, 10, 30, 30), 0, 3, true, blurred(ui.BlurStyleOuter, 1.5))
		}, 2, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res.beginFrame()
			b := frameWith(t, res, tt.draw)
			if got := callOps(&b.frame); !equalStrings(got, []string{"begin", tt.inner, "composite"}) {
				t.Fatalf("calls = %v", got)
			}
			begin, comp := b.frame.calls[0], b.frame.calls[2]
			if comp.eff == nil || comp.eff.mask != tt.style || len(comp.eff.passes) != 2 || comp.eff.passes[0].kind != passBlur {
				t.Fatalf("effect = %+v", comp.eff)
			}
			if got := float64(comp.eff.passes[0].sigma); math.Abs(got-tt.sigma) > 1e-5 {
				t.Errorf("sigma = %v, want %v", got, tt.sigma)
			}
			// The layer is the shape plus 3 sigma on each side, and the composite goes
			// no further than the clip.
			margin := int(math.Ceil(3 * tt.sigma))
			if begin.rect.Dx() <= 2*margin || begin.rect.Empty() {
				t.Errorf("layer = %v", begin.rect)
			}
			if got := b.frame.data[pipeLayer][:4]; got[0] < 0 || got[2] > 200 || got[1] < 0 || got[3] > 100 {
				t.Errorf("composite quad = %v", got)
			}
			if b.stats.Drawn != 1 {
				t.Errorf("stats = %+v", b.stats)
			}
		})
	}
	// A shape that is entirely beyond the blur's reach of the surface draws nothing.
	res.beginFrame()
	far := frameWith(t, res, func(c *ui.Canvas) { c.DrawPath(wave().Shift(ui.Offset{Dx: 1000}), blurred(ui.BlurStyleNormal, 3)) })
	if len(far.frame.calls) != 0 || far.stats.Drawn != 1 {
		t.Errorf("calls %v, stats %+v", callOps(&far.frame), far.stats)
	}
	// A shape beyond the surface still blurs onto it.
	res.beginFrame()
	near := frameWith(t, res, func(c *ui.Canvas) { c.DrawPath(wave().Shift(ui.Offset{Dx: -60}), blurred(ui.BlurStyleNormal, 8)) })
	if len(near.frame.calls) == 0 {
		t.Errorf("a shape just off the surface is not drawn")
	}
}

func TestMaskFilterWithNoBlurDrawsTheShape(t *testing.T) {
	p := solidPaint(0xFF204080)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleSolid, 0))
	b := mustBuild(t, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(10, 10, 30, 20), p) })
	if b.frame.items(pipeShape) != 1 || len(b.frame.calls) != 1 || b.frame.data[pipeShape][10] == kindShadow {
		t.Errorf("calls %v", callOps(&b.frame))
	}
	// A mask filter does not apply to an image.
	res, _, _ := newTestResources(t)
	q := ui.NewPaint()
	q.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 5))
	img := testImage(8, 8)
	ib := frameWith(t, res, func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, q) })
	if got := callOps(&ib.frame); !equalStrings(got, []string{"image"}) {
		t.Errorf("calls = %v", got)
	}
}

func TestPaintImageFilterDrawsIntoAFilteredLayer(t *testing.T) {
	p := solidPaint(0xC0204080)
	p.SetImageFilter(blurFilter(3, 3, ui.TileModeUnspecified))
	p.SetBlendMode(ui.BlendModeMultiply)
	b := mustBuild(t, func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(5, 5, 80, 60))
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 30, 20), p)
	})
	if got := callOps(&b.frame); !equalStrings(got, []string{"begin", "shape", "composite"}) {
		t.Fatalf("calls = %v", got)
	}
	begin, draw, comp := b.frame.calls[0], b.frame.calls[1], b.frame.calls[2]
	if begin.rect != image.Rect(5, 5, 85, 65) {
		t.Errorf("layer = %v", begin.rect)
	}
	// The shape is drawn alone, opaque in the layer's blend, and the layer is
	// filtered, then blended where something was drawn.
	if draw.spec.ReadBlend != 0 || draw.blend != blendFuncs[ui.BlendModeSrcOver] {
		t.Errorf("inner draw blend %+v spec %+v", draw.blend, draw.spec)
	}
	if comp.eff == nil || !comp.eff.drawnOnly || comp.eff.mask != -1 || len(comp.eff.passes) != 2 || comp.spec.ReadBlend == 0 {
		t.Errorf("composite = %+v spec %+v", comp.eff, comp.spec)
	}
	if b.stats.Drawn != 1 || b.stats.Commands != 1 {
		t.Errorf("stats = %+v", b.stats)
	}
	// With the blend mode dst there is nothing to draw.
	p.SetBlendMode(ui.BlendModeDst)
	if got := mustBuild(t, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(10, 10, 30, 20), p) }); len(got.frame.calls) != 0 || got.stats.Drawn != 1 {
		t.Errorf("calls %v", callOps(&got.frame))
	}
}

func TestSaveLayerImageFilter(t *testing.T) {
	f := ui.NewPaint()
	f.SetImageFilter(blurFilter(2, 1, ui.TileModeUnspecified))
	b := mustBuild(t, func(c *ui.Canvas) {
		c.Scale(3, 3)
		c.SaveLayer(&ui.Rect{Left: 5, Top: 5, Right: 20, Bottom: 15}, f)
		c.DrawRect(ui.NewRectFromLTWH(6, 6, 5, 5), solidPaint(0xFF000000))
		c.Restore()
	})
	comp := b.frame.calls[len(b.frame.calls)-1]
	if comp.eff == nil || len(comp.eff.passes) != 2 || comp.eff.passes[0].sigma != 6 || comp.eff.passes[1].sigma != 3 {
		t.Fatalf("effect = %+v", comp.eff)
	}
	// A blur with no tile mode clamps for a save layer.
	if comp.eff.passes[0].tile != gpu.TileIndex(ui.TileModeClamp) {
		t.Errorf("tile = %d", comp.eff.passes[0].tile)
	}
}

func TestSceneImageFilterAndBackdropLayers(t *testing.T) {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 40, 40), solidPaint(0xFF000000))
	pic := rec.EndRecording()
	sb := ui.NewSceneBuilder()
	sb.PushOffset(10, 10)
	sb.PushImageFilter(blurFilter(2, 2, ui.TileModeUnspecified), ui.PushImageFilterOpts{Offset: ui.Offset{Dx: 3, Dy: 0}})
	sb.AddPicture(ui.Offset{}, pic)
	sb.Pop()
	sb.PushClipRect(ui.NewRectFromLTWH(20, 20, 60, 40))
	sb.PushBackdropFilter(blurFilter(4, 4, ui.TileModeUnspecified), ui.PushBackdropFilterOpts{BlendMode: ui.BlendModeMultiply})
	sb.AddPicture(ui.Offset{}, pic)
	sb.Pop()
	sb.Pop()
	sb.Pop()
	list, err := gpu.Flatten(sb.Build(), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 200, 100)})
	if err != nil {
		t.Fatal(err)
	}
	b := &frameBuilder{w: 200, h: 100}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			t.Fatal(err)
		}
	}
	if got := callOps(&b.frame); !equalStrings(got, []string{"begin", "shape", "composite", "begin", "shape", "composite"}) {
		t.Fatalf("calls = %v", got)
	}
	// A blur with no tile mode is transparent beyond the edges for an image filter
	// layer, and mirrors for a backdrop, whose layer starts as the filtered backdrop.
	imageComp := b.frame.calls[2]
	if imageComp.eff == nil || imageComp.eff.passes[0].tile != gpu.TileIndex(ui.TileModeDecal) {
		t.Errorf("image filter effect = %+v", imageComp.eff)
	}
	begin, comp := b.frame.calls[3], b.frame.calls[5]
	if begin.eff == nil || !begin.eff.backdrop || len(begin.eff.passes) != 2 || begin.eff.passes[0].tile != gpu.TileIndex(ui.TileModeMirror) {
		t.Fatalf("backdrop begin = %+v", begin.eff)
	}
	if begin.rect != image.Rect(30, 30, 90, 70) || begin.eff.passes[0].outW != 60 {
		t.Errorf("backdrop layer = %v, passes %+v", begin.rect, begin.eff.passes)
	}
	// The layer is composited with the backdrop's blend mode and no filter.
	if comp.eff != nil || comp.spec.ReadBlend == 0 {
		t.Errorf("backdrop composite = %+v spec %+v", comp.eff, comp.spec)
	}
}

func TestShaderMaskPaintsTheShaderIntoTheLayer(t *testing.T) {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 40, 40), solidPaint(0xFF000000))
	sb := ui.NewSceneBuilder()
	g := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 30}, twoColors())
	sb.PushOffset(5, 5)
	sb.PushShaderMask(g, ui.NewRectFromLTWH(10, 10, 30, 20), ui.BlendModeDstIn)
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	sb.Pop()
	sb.Pop()
	list, err := gpu.Flatten(sb.Build(), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 200, 100)})
	if err != nil {
		t.Fatal(err)
	}
	b := &frameBuilder{w: 200, h: 100}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			t.Fatal(err)
		}
	}
	if got := callOps(&b.frame); !equalStrings(got, []string{"begin", "shape", "shape", "composite"}) {
		t.Fatalf("calls = %v", got)
	}
	mask := b.frame.calls[2]
	// DstIn is blended in the shader, which reads the layer's pixels.
	if mask.spec.Shader != gpu.ShaderLinear || mask.blend != replaceBlend || mask.spec.ReadBlend != int32(ui.BlendModeDstIn-ui.BlendModeClear)+1 {
		t.Errorf("mask draw: shader %d blend %+v read %d", mask.spec.Shader, mask.blend, mask.spec.ReadBlend)
	}
	// The gradient starts at the mask rectangle's corner: the shader's space is the
	// layer's translated by it, so the pixel that is the corner maps to the origin.
	if got := mask.spec.XF; got[4] != -15 || got[5] != -15 {
		t.Errorf("shader transform = %v, want a translation by -15, -15", got)
	}
}

func TestShadowIsMadeOfBlurredLayers(t *testing.T) {
	res, _, _ := newTestResources(t)
	path := wave()
	tests := []struct {
		name        string
		elevation   float64
		transparent bool
		want        []string
	}{
		{"opaque", 6, false, []string{"begin", "begin", "mask", "composite", "begin", "mask", "composite", "mask", "composite"}},
		{"transparent occluder", 6, true, []string{"begin", "begin", "mask", "composite", "begin", "mask", "composite", "composite"}},
		{"no elevation", 0, false, nil},
		{"above the light", 700, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res.beginFrame()
			b := frameWith(t, res, func(c *ui.Canvas) {
				c.DrawShadow(path.Shift(ui.Offset{Dx: 40, Dy: 20}), ui.NewColor(0xFF000000), tt.elevation, tt.transparent)
			})
			if got := callOps(&b.frame); !equalStrings(got, tt.want) {
				t.Fatalf("calls = %v, want %v", got, tt.want)
			}
			if b.stats.Drawn != 1 || b.stats.Commands != 1 {
				t.Errorf("stats = %+v", b.stats)
			}
			if tt.want == nil {
				return
			}
			// The ambient shadow is blurred less than the spot shadow, which is lighter.
			amb, spot := b.frame.calls[3], b.frame.calls[6]
			if amb.eff.passes[0].sigma >= spot.eff.passes[0].sigma {
				t.Errorf("ambient sigma %v, spot sigma %v", amb.eff.passes[0].sigma, spot.eff.passes[0].sigma)
			}
			ambAlpha, spotAlpha := b.frame.data[pipeLayer][4+3], b.frame.data[pipeLayer][12+4+3]
			if math.Abs(float64(ambAlpha)-shadowAmbientAlpha) > 0.003 || math.Abs(float64(spotAlpha)-shadowSpotAlpha) > 0.003 {
				t.Errorf("alphas %v, %v", ambAlpha, spotAlpha)
			}
			// The cut-out removes what the shape covers from the layer.
			if !tt.transparent && b.frame.calls[7].blend != blendFuncs[ui.BlendModeDstOut] {
				t.Errorf("cut blend = %+v", b.frame.calls[7].blend)
			}
		})
	}
}

func TestRendererRunsFilterPassesAndCopiesTheBackdrop(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), solidPaint(0xFFFF0000))
	pic := rec.EndRecording()
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, pic)
	sb.PushClipRect(ui.NewRectFromLTWH(10, 10, 40, 40))
	sb.PushImageFilter(blurFilter(2, 2, ui.TileModeDecal))
	sb.AddPicture(ui.Offset{}, pic)
	sb.Pop()
	sb.PushBackdropFilter(blurFilter(30, 4, ui.TileModeMirror))
	sb.Pop()
	sb.Pop()
	scene := sb.Build()
	gl.resetCalls()
	if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	// The blurred layer: two passes. The backdrop: a copy of what is under it, then
	// down, two blurs and up (the wide blur is made on a smaller picture). The shape
	// and the two composites are instanced draws.
	if gl.calls["drawArrays"] != 2+4 || gl.calls["drawArraysInstanced"] != 4 || gl.calls["copyTexSubImage2D"] != 1 {
		t.Errorf("filter draws %d, instanced draws %d, copies %d", gl.calls["drawArrays"], gl.calls["drawArraysInstanced"], gl.calls["copyTexSubImage2D"])
	}
	st := r.LastFrame()
	if st.Skipped != 0 || st.Drawn != 2 {
		t.Errorf("stats = %+v", st)
	}
	// Everything went back to the pool.
	if ds := r.ctx.dev.Stats(); ds.Framebuffers != ds.FreeFramebuffers || ds.Framebuffers == 0 {
		t.Errorf("device stats = %+v", ds)
	}
	last := ""
	for _, call := range gl.log {
		if len(call) > 15 && call[:15] == "bindFramebuffer" {
			last = call
		}
	}
	if last != "bindFramebuffer36160 0" {
		t.Errorf("last framebuffer bound: %s", last)
	}
}
