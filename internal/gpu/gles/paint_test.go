package gles

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

func twoColors() []ui.Color { return []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)} }

func TestPaintSpecsAreSharedAndMerged(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 20, 20)
	g := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 100}, twoColors())
	shaderPaint := func(sh ui.Shader) *ui.Paint {
		p := ui.NewPaint()
		p.SetShader(sh)
		return p
	}
	b := mustBuild(t, func(c *ui.Canvas) {
		c.DrawRect(rect, shaderPaint(g))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 30}), shaderPaint(g))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 60}), shaderPaint(ui.NewGradientRadial(ui.Offset{Dx: 5}, 20, twoColors())))
	})
	if len(b.frame.calls) != 2 || b.frame.calls[0].count != 2 || b.frame.calls[0].spec == b.frame.calls[1].spec {
		t.Fatalf("calls = %d, first has %d instances", len(b.frame.calls), b.frame.calls[0].count)
	}
	sp := b.frame.calls[0].spec
	if sp.Shader != gpu.ShaderLinear || sp.RampN != 2 || sp.P0 != [4]float32{0, 0, 100, 0} || sp.P1[0] != 0.0001 || !sp.Dither {
		t.Errorf("linear spec = %+v", sp)
	}
	// A shader paint's color is its alpha in every component, for the shader to scale.
	if got := b.frame.data[pipeShape][12:16]; got[0] != 1 || got[3] != 1 {
		t.Errorf("instance color = %v", got)
	}
	if b.frame.calls[1].spec.Shader != gpu.ShaderRadial {
		t.Errorf("radial spec = %+v", b.frame.calls[1].spec)
	}
}

func TestGradientSpecs(t *testing.T) {
	cols := twoColors()
	tests := []struct {
		name   string
		shader ui.Shader
		check  func(t *testing.T, sp *paintSpec)
	}{
		{"sweep", ui.NewGradientSweep(ui.Offset{Dx: 3, Dy: 4}, cols, ui.GradientSweepOpts{StartAngle: 1, EndAngle: 3, TileMode: ui.TileModeMirror}), func(t *testing.T, sp *paintSpec) {
			if sp.Shader != gpu.ShaderSweep || sp.P0 != [4]float32{3, 4, 1, 2} || sp.Tile[0] != 2 {
				t.Errorf("spec = %+v", sp)
			}
		}},
		{"conical", ui.NewGradientRadial(ui.Offset{Dx: 50, Dy: 50}, 20, cols, ui.GradientRadialOpts{Focal: &ui.Offset{Dx: 40, Dy: 45}, FocalRadius: 3, TileMode: ui.TileModeRepeated}), func(t *testing.T, sp *paintSpec) {
			if sp.Shader != gpu.ShaderConical || sp.P0 != [4]float32{40, 45, 3, 0} || sp.P1 != [4]float32{50, 50, 20, 0} || sp.Tile[0] != 1 {
				t.Errorf("spec = %+v", sp)
			}
		}},
		{"stops that do not reach the ends", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, append(cols, ui.NewColor(0xFF00FF00)), ui.GradientLinearOpts{ColorStops: []float64{0.2, 0.5, 0.6}}), func(t *testing.T, sp *paintSpec) {
			want := []float32{0, 0.2, 0.5, 0.6, 1}
			if int(sp.RampN) != len(want) {
				t.Fatalf("ramp has %d stops", sp.RampN)
			}
			for i, p := range want {
				if sp.RampPos[i] != p {
					t.Errorf("stop %d at %v, want %v", i, sp.RampPos[i], p)
				}
			}
		}},
		{"degenerate line is the last color", ui.NewGradientLinear(ui.Offset{Dx: 5}, ui.Offset{Dx: 5}, cols), func(t *testing.T, sp *paintSpec) {
			if sp.Shader != gpu.ShaderConst || sp.ConstCol != [4]float32{0, 0, 1, 1} || sp.RampN != 0 {
				t.Errorf("spec = %+v", sp)
			}
		}},
		{"degenerate decal line is nothing", ui.NewGradientLinear(ui.Offset{Dx: 5}, ui.Offset{Dx: 5}, cols, ui.GradientLinearOpts{TileMode: ui.TileModeDecal}), func(t *testing.T, sp *paintSpec) {
			if sp.Shader != gpu.ShaderConst || sp.ConstCol != [4]float32{} {
				t.Errorf("spec = %+v", sp)
			}
		}},
		{"a matrix that cannot be undone", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 5}, cols, ui.GradientLinearOpts{Matrix4: ui.NewMatrix4Diagonal3Values(0, 1, 1)}), func(t *testing.T, sp *paintSpec) {
			if sp.Shader != gpu.ShaderConst || sp.ConstCol != [4]float32{} {
				t.Errorf("spec = %+v", sp)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ui.NewPaint()
			p.SetShader(tt.shader)
			b := mustBuild(t, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), p) })
			if len(b.frame.calls) != 1 {
				t.Fatalf("%d calls", len(b.frame.calls))
			}
			tt.check(t, b.frame.calls[0].spec)
		})
	}
	// The matrix of the shader and the canvas's transform go into the map from pixels.
	p := ui.NewPaint()
	p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, cols, ui.GradientLinearOpts{Matrix4: ui.NewMatrix4Diagonal3Values(2, 2, 1)}))
	b := mustBuild(t, func(c *ui.Canvas) {
		c.Translate(10, 20)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), p)
	})
	if xf := b.frame.calls[0].spec.XF; xf != [6]float32{0.5, 0, 0, 0.5, -5, -10} {
		t.Errorf("xf = %v", xf)
	}
}

func TestColorFilterSpecs(t *testing.T) {
	var m [20]float64
	for i := range m {
		m[i] = float64(i)
	}
	tests := []struct {
		name   string
		filter *ui.ColorFilter
		invert bool
		check  func(t *testing.T, sp *paintSpec)
	}{
		{"mode", ui.NewColorFilterMode(ui.NewColor(0x80FF0000), ui.BlendModeMultiply), false, func(t *testing.T, sp *paintSpec) {
			a := float32(0x80) / 255
			if sp.Filter != gpu.FilterMode || sp.FilterBlend != 24 || sp.FilterCol != [4]float32{a, 0, 0, a} {
				t.Errorf("spec = %+v", sp)
			}
		}},
		{"matrix by columns", ui.NewColorFilterMatrix(m), false, func(t *testing.T, sp *paintSpec) {
			// Column 1 holds the weights of green in the four outputs: m[1], m[6], m[11], m[16].
			if sp.Filter != gpu.FilterMatrix || sp.FilterMat[4] != 1 || sp.FilterMat[5] != 6 || sp.FilterMat[6] != 11 || sp.FilterMat[7] != 16 || sp.FilterMat[16] != 4 {
				t.Errorf("matrix = %v", sp.FilterMat)
			}
		}},
		{"gamma to sRGB", ui.NewColorFilterLinearToSrgbGamma(), false, func(t *testing.T, sp *paintSpec) {
			if sp.Filter != gpu.FilterToSrgb {
				t.Errorf("filter = %d", sp.Filter)
			}
		}},
		{"gamma from sRGB", ui.NewColorFilterSrgbToLinearGamma(), false, func(t *testing.T, sp *paintSpec) {
			if sp.Filter != gpu.FilterFromSrgb {
				t.Errorf("filter = %d", sp.Filter)
			}
		}},
		{"inverted colors", nil, true, func(t *testing.T, sp *paintSpec) {
			if !sp.Invert || sp.Filter != gpu.FilterNone {
				t.Errorf("spec = %+v", sp)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := solidPaint(0xFF204060)
			p.SetColorFilter(tt.filter)
			p.SetInvertColors(tt.invert)
			b := mustBuild(t, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), p) })
			tt.check(t, b.frame.calls[0].spec)
			if b.frame.calls[0].spec.ReadBlend != 0 {
				t.Error("a color filter needs no copy of the destination")
			}
		})
	}
}

func TestImageShaderSpec(t *testing.T) {
	res, _, _ := newTestResources(t)
	img := testImage(16, 8)
	p := ui.NewPaint()
	p.SetShader(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeMirror, ui.NewMatrix4Diagonal3Values(0.25, 0.25, 1), ui.ImageShaderOpts{FilterQuality: ui.FilterQualityHigh}))
	b := frameWith(t, res, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), p) })
	sp := b.frame.calls[0].spec
	// Four image pixels per surface pixel: the level two mip, with bicubic filtering.
	if sp.Shader != gpu.ShaderImage || sp.Tile != [2]int32{1, 2} || sp.Image == nil || sp.ImgInfo != [4]float32{3, 2, 0, 0} || sp.XF != [6]float32{4, 0, 0, 4, 0, 0} {
		t.Errorf("spec = %+v", sp)
	}
	// Without the caches an image shader is unsupported.
	_, err := build(t, false, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), p) })
	if err == nil {
		t.Error("an image shader without caches was drawn")
	}
}

func TestBlendsThatReadTheDestination(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 20, 20)
	mode := func(m ui.BlendMode) *ui.Paint {
		p := solidPaint(0xFF2060A0)
		p.SetBlendMode(m)
		return p
	}
	b := mustBuild(t, func(c *ui.Canvas) {
		c.DrawRect(rect, mode(ui.BlendModeSrcOver))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 30}), mode(ui.BlendModeSrcOver))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 60}), mode(ui.BlendModeMultiply))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 65}), mode(ui.BlendModeMultiply))
		c.DrawRect(rect.Shift(ui.Offset{Dx: 90}), mode(ui.BlendModePlus))
	})
	// The fixed-function modes merge; the others are one call each, so that a draw
	// sees what the one before it left, and each copies only the pixels it covers.
	if len(b.frame.calls) != 4 || b.frame.calls[0].count != 2 {
		t.Fatalf("%d calls, first of %d", len(b.frame.calls), b.frame.calls[0].count)
	}
	for _, i := range []int{1, 2} {
		c := b.frame.calls[i]
		if c.spec.ReadBlend != int32(ui.BlendModeMultiply-ui.BlendModeClear)+1 || c.blend != replaceBlend || c.seq == 0 {
			t.Errorf("call %d = %+v spec %+v", i, c.callKey, c.spec)
		}
		x := 70 + (i-1)*5
		if c.scissor != [4]int{x - 1, 9, 22, 22} {
			t.Errorf("call %d scissor = %v", i, c.scissor)
		}
	}
	if b.frame.calls[3].blend != blendFuncs[ui.BlendModePlus] || b.frame.calls[3].spec.ReadBlend != 0 {
		t.Errorf("plus = %+v", b.frame.calls[3].callKey)
	}
	// A draw that leaves no pixel in the clip has no call.
	b = mustBuild(t, func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 5, 5))
		c.DrawRect(rect, mode(ui.BlendModeMultiply))
	})
	if len(b.frame.calls) != 0 {
		t.Errorf("%d calls for a draw outside the clip", len(b.frame.calls))
	}
}

func TestLayersBecomeBeginAndComposite(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 60, 40)
	faded := solidPaint(0x80FFFFFF)
	b := mustBuild(t, func(c *ui.Canvas) {
		c.SaveLayer(&ui.Rect{Left: 20, Top: 5, Right: 50, Bottom: 80}, faded)
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.Restore()
	})
	ops := []callOp{}
	for _, c := range b.frame.calls {
		ops = append(ops, c.op)
	}
	if len(ops) != 3 || ops[0] != opBeginLayer || ops[1] != opDraw || ops[2] != opComposite {
		t.Fatalf("ops = %v", ops)
	}
	begin, draw, comp := b.frame.calls[0], b.frame.calls[1], b.frame.calls[2]
	want := image.Rect(20, 5, 50, 80)
	if begin.rect != want || comp.rect != want {
		t.Errorf("layer rects %v and %v, want %v", begin.rect, comp.rect, want)
	}
	// What is drawn in the layer is limited to it.
	if draw.scissor != [4]int{20, 5, 30, 75} {
		t.Errorf("scissor in the layer = %v", draw.scissor)
	}
	// The composite is one instance of the layer pipeline with the layer's alpha.
	in := b.frame.data[pipeLayer]
	if len(in) != 12 || in[0] != 20 || in[1] != 5 || in[2] != 50 || in[3] != 80 || in[4] < 0.49 || in[4] > 0.51 || in[7] != in[4] {
		t.Errorf("composite instance = %v", in)
	}
	if comp.pipe != pipeLayer || comp.count != 1 || comp.blend != blendFuncs[ui.BlendModeSrcOver] || comp.scissor != [4]int{20, 5, 30, 75} {
		t.Errorf("composite = %+v", comp)
	}
}

func TestLayerParameters(t *testing.T) {
	filtered := ui.NewPaint()
	filtered.SetColorFilter(ui.NewColorFilterLinearToSrgbGamma())
	filtered.SetInvertColors(true)
	blended := ui.NewPaint()
	blended.SetBlendMode(ui.BlendModeHardLight)
	rect := ui.NewRectFromLTWH(10, 10, 60, 40)
	b := mustBuild(t, func(c *ui.Canvas) {
		c.SaveLayer(nil, filtered)
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.Restore()
		c.SaveLayer(nil, blended)
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.Restore()
	})
	var composites []drawCall
	for _, c := range b.frame.calls {
		if c.op == opComposite {
			composites = append(composites, c)
		}
	}
	if len(composites) != 2 {
		t.Fatalf("%d composites", len(composites))
	}
	if sp := composites[0].spec; sp.Filter != gpu.FilterToSrgb || !sp.Invert || sp.ReadBlend != 0 {
		t.Errorf("filtered layer spec = %+v", sp)
	}
	if c := composites[1]; c.spec.ReadBlend != int32(ui.BlendModeHardLight-ui.BlendModeClear)+1 || c.blend != replaceBlend || c.seq == 0 {
		t.Errorf("blended layer = %+v", c.callKey)
	}
}

func TestLayersCullAndNest(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 60, 40)
	// A layer with nothing to show has no framebuffer, and what it holds counts as drawn.
	b := mustBuild(t, func(c *ui.Canvas) {
		c.Save()
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 0, 0))
		c.SaveLayer(nil, solidPaint(0x80FFFFFF))
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.Restore()
		c.Restore()
		off := ui.NewPaint()
		off.SetBlendMode(ui.BlendModeDst)
		c.SaveLayer(nil, off)
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.SaveLayer(&ui.Rect{Left: 300, Top: 300, Right: 400, Bottom: 400}, nil)
		c.Restore()
		c.Restore()
	})
	if len(b.frame.calls) != 0 || b.stats.Commands != 2 || b.stats.Drawn != 2 || b.stats.Skipped != 0 {
		t.Errorf("%d calls, stats %+v", len(b.frame.calls), b.stats)
	}
	if len(b.layers) != 0 || b.culled != 0 || b.skipping != 0 {
		t.Errorf("layers %d, culled %d, skipping %d left", len(b.layers), b.culled, b.skipping)
	}
	// A layer inside a layer is limited to it, and composites into it.
	b = mustBuild(t, func(c *ui.Canvas) {
		c.SaveLayer(&ui.Rect{Left: 10, Top: 10, Right: 90, Bottom: 90}, nil)
		c.SaveLayer(&ui.Rect{Left: 50, Top: 0, Right: 150, Bottom: 50}, solidPaint(0x80FFFFFF))
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.Restore()
		c.Restore()
	})
	var ops []callOp
	for _, c := range b.frame.calls {
		ops = append(ops, c.op)
	}
	wantOps := []callOp{opBeginLayer, opBeginLayer, opDraw, opComposite, opComposite}
	if len(ops) != len(wantOps) {
		t.Fatalf("ops = %v", ops)
	}
	for i := range ops {
		if ops[i] != wantOps[i] {
			t.Fatalf("ops = %v, want %v", ops, wantOps)
		}
	}
	if inner := b.frame.calls[1].rect; inner != image.Rect(50, 10, 90, 50) {
		t.Errorf("inner layer = %v", inner)
	}
	// The inner one composites into the outer, which is what its scissor is limited to.
	if got := b.frame.calls[3].scissor; got != [4]int{50, 10, 40, 40} {
		t.Errorf("inner composite scissor = %v", got)
	}
	if got := b.frame.calls[4].rect; got != image.Rect(10, 10, 90, 90) {
		t.Errorf("outer layer = %v", got)
	}
}

func TestUnsupportedGroupsAreSkippedWhole(t *testing.T) {
	feature := unknownBlendFeature
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.SaveLayer(nil, unknownBlendPaint())
	c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solidPaint(0xFF000000))
	c.Restore()
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	list, err := gpu.Flatten(sb.Build(), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100)})
	if err != nil {
		t.Fatal(err)
	}
	strict := &frameBuilder{w: 100, h: 100}
	var got error
	for _, c := range list.Commands {
		if got = strict.command(c); got != nil {
			break
		}
	}
	if appErr, ok := errors.FromError(got); !ok || appErr.Tag != tagGLNotImplemented || appErr.Metadata["feature"] != feature {
		t.Fatalf("error = %v, want GLNotImplemented for %q", got, feature)
	}
	skipping := &frameBuilder{w: 100, h: 100, skip: true}
	for _, c := range list.Commands {
		if err := skipping.command(c); err != nil {
			t.Fatal(err)
		}
	}
	if len(skipping.stats.Unsupported) != 1 || skipping.stats.Unsupported[0] != feature || skipping.stats.Skipped != 1 || len(skipping.frame.calls) != 0 {
		t.Errorf("stats = %+v", skipping.stats)
	}
}

func TestSceneLayersOfOpacityFiltersAndClips(t *testing.T) {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 40, 40), solidPaint(0xFF000000))
	pic := rec.EndRecording()
	sb := ui.NewSceneBuilder()
	sb.PushOpacity(51, ui.PushOpacityOpts{Offset: ui.Offset{Dx: 10, Dy: 10}})
	sb.AddPicture(ui.Offset{}, pic)
	sb.Pop()
	sb.PushColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeScreen))
	sb.AddPicture(ui.Offset{Dx: 60}, pic)
	sb.Pop()
	sb.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(5.5, 5.5, 50, 50), 10, 10), ui.PushClipRRectOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
	sb.AddPicture(ui.Offset{}, pic)
	sb.Pop()
	list, err := gpu.Flatten(sb.Build(), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100)})
	if err != nil {
		t.Fatal(err)
	}
	res, _, _ := newTestResources(t)
	res.beginFrame()
	b := &frameBuilder{w: 100, h: 100, res: res}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			t.Fatal(err)
		}
	}
	var composites []drawCall
	for _, c := range b.frame.calls {
		if c.op == opComposite {
			composites = append(composites, c)
		}
	}
	if len(composites) != 3 {
		t.Fatalf("%d composites", len(composites))
	}
	if a := b.frame.data[pipeLayer][4]; math.Abs(float64(a)-0.2) > 0.001 {
		t.Errorf("opacity layer alpha = %v, want 0.2", a)
	}
	if sp := composites[1].spec; sp.Filter != gpu.FilterMode || sp.FilterBlend != int32(ui.BlendModeScreen-ui.BlendModeClear) {
		t.Errorf("color filter layer spec = %+v", sp)
	}
	// The clip layer is composited through the mask of its shape, and what is drawn
	// inside it is not clipped by the shape.
	if composites[2].clip == nil {
		t.Error("the clip layer has no mask")
	}
	inside := b.frame.calls[len(b.frame.calls)-2]
	if inside.op != opDraw || inside.clip != nil {
		t.Errorf("the draw in the clip layer = %+v", inside.callKey)
	}
}

func TestRendererDrawsLayersAndReadsTheDestination(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	multiply := solidPaint(0xFF2060A0)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	grad := ui.NewPaint()
	grad.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 50}, twoColors()))
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), grad)
	c.SaveLayer(&ui.Rect{Left: 10, Top: 10, Right: 40, Bottom: 40}, solidPaint(0x80FFFFFF))
	c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), solidPaint(0xFFFF0000))
	c.DrawRect(ui.NewRectFromLTWH(20, 20, 30, 30), multiply)
	c.Restore()
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	scene := sb.Build()
	gl.resetCalls()
	if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	// One framebuffer for the layer, cleared when it starts; the gradient, the red
	// rectangle, the multiplied one and the composite are four draws, and the
	// multiplied one has a copy of what is under it.
	if gl.calls["genFramebuffer"] != 1 || gl.calls["drawArraysInstanced"] != 4 || gl.calls["copyTexSubImage2D"] != 1 {
		t.Errorf("framebuffers %d, draws %d, copies %d", gl.calls["genFramebuffer"], gl.calls["drawArraysInstanced"], gl.calls["copyTexSubImage2D"])
	}
	if gl.clears != 2 { // the surface, and the layer
		t.Errorf("%d clears", gl.clears)
	}
	st := r.LastFrame()
	if st.Commands != 3 || st.Drawn != 3 || st.Batches != 4 || st.Instances != 4 {
		t.Errorf("stats = %+v", st)
	}
	// The window's framebuffer is bound again, and the layer's went back to the pool.
	last := ""
	for _, call := range gl.log {
		if len(call) > 15 && call[:15] == "bindFramebuffer" {
			last = call
		}
	}
	if last != "bindFramebuffer36160 0" {
		t.Errorf("last framebuffer bound: %s", last)
	}
	if ds := r.ctx.dev.Stats(); ds.Framebuffers != 1 || ds.FreeFramebuffers != 1 {
		t.Errorf("device stats = %+v", ds)
	}
	// The next frame reuses it.
	gl.resetCalls()
	if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	if gl.calls["genFramebuffer"] != 0 {
		t.Error("the layer's framebuffer was made again")
	}
}
