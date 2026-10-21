package explicit

import (
	"fmt"
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The frame is what a draw list becomes before any draw call is recorded: the data of each
// pipeline's instances or vertices and the draw calls that use them, in order. Building it
// first lets the renderer fail on an unsupported feature before it draws anything, and upload
// each pipeline's data once. The textures that the frame needs (coverage tiles, clip masks,
// images) are made, and their pixels queued with the device, while it is built.

// FrameStats says what the last frame did.
//
// Flutter: no counterpart
type FrameStats struct {
	// Commands is the number of drawing commands in the scene.
	Commands int
	// Drawn is the number of commands drawn.
	Drawn int
	// Skipped is the number of commands left out because the renderer cannot draw them
	// yet; only a renderer with SkipUnsupported skips.
	Skipped int
	// Instances is the number of instanced primitives sent to the GPU (shapes, coverage
	// tiles and image rectangles), Vertices the number of vertices, and Batches the number
	// of draw calls that drew them.
	Instances, Vertices, Batches int
	// TilesRasterized, ClipMasksBuilt and ImagesUploaded count the work the frame did that an
	// earlier frame's cache could have saved: coverage tiles rasterized, clip masks made and
	// images uploaded.
	TilesRasterized, ClipMasksBuilt, ImagesUploaded int
	// Unsupported lists the features that caused skipped commands, once each.
	Unsupported []string
}

// pipeKind is one of the shader pipelines that draw a kind of primitive.
type pipeKind int

const (
	pipeShape    pipeKind = iota // analytic shapes: instanced quads (shaders/shape.wgsl)
	pipeMask                     // coverage atlas tiles: instanced quads (shaders/mask.wgsl)
	pipeImage                    // image rectangles: instanced quads (shaders/image.wgsl)
	pipeVertices                 // triangles from vertices (shaders/vertices.wgsl)
	pipeLayer                    // a layer composited into its parent (shaders/layer.wgsl)
	numPipes
)

// itemFloats is the size of an item of each pipeline in floats: an instance, or for the
// vertices pipeline one vertex.
var itemFloats = [numPipes]int{pipeShape: gpu.ShapeFloats, pipeMask: 16, pipeImage: gpu.ImageFloats, pipeVertices: gpu.VertexFloats, pipeLayer: 12}

// vertexParams are the uniforms of a draw of the vertices pipeline.
type vertexParams struct {
	paint  [4]float32 // premultiplied
	clip   [4]float32 // left, top, right, bottom in pixels
	mode   int32      // blend index of the paint with the vertex colors
	colors bool
	// atTexture says that the shader is evaluated at the texture coordinates.
	atTexture bool
}

// callOp is what a draw call does.
type callOp int

const (
	opDraw       callOp = iota // draws items of its pipeline
	opBeginLayer               // starts drawing into a new layer, see drawCall.rect
	opComposite                // ends the innermost layer and draws it into its parent, as one item of pipeLayer
)

// callKey is what two draws must share to be one draw call.
type callKey struct {
	op      callOp
	pipe    pipeKind
	blend   gpu.FixedBlend
	scissor image.Rectangle // in pixels of the surface
	// clip is the coverage mask of the clip's shapes, or nil.
	clip *clipMask
	// page is the atlas page of a coverage tile, and img the image of an image rectangle.
	page int
	img  *imageEntry
	vert vertexParams
	// spec is the paint's shader, color filter and blend; never nil for a draw.
	spec *gpu.PaintSpec
	// seq is not 0 for a call that must not be merged with any other: one that reads the pixels
	// under it.
	seq int
}

// drawCall is a run of items drawn with one state.
type drawCall struct {
	callKey
	first, count int // the items, in the pipeline's data
	// rect is the pixels of the surface that a layer covers, for opBeginLayer and opComposite.
	rect image.Rectangle
}

// replaceBlend writes the shader's output as it is: the blend of a paint whose shader blends
// with the destination.
var replaceBlend = gpu.FixedBlend{Src: gpu.BlendFactorOne, Dst: gpu.BlendFactorZero}

// frameBuilder turns the commands of a draw list into instances. It calls the device only to
// make and fill the textures that the commands need.
type frameBuilder struct {
	w, h int
	skip bool // unsupported commands are skipped and counted instead of failing
	res  *resources

	data  [numPipes][]float32
	calls []drawCall
	stats FrameStats
	// skipping is the depth of the unsupported group being skipped; 0 when none.
	skipping int
	// reasons are the unsupported features seen, to count each once.
	reasons map[string]bool
	// specs interns the paint specs of the frame.
	specs map[gpu.PaintSpec]*gpu.PaintSpec
	// seq numbers the draw calls that read the pixels under them, which are never merged.
	seq int
	// layers are the groups being drawn, innermost last; culled is the depth of a group with
	// nothing to draw, whose commands are dropped.
	layers []*layerRec
	culled int
}

// intern returns the one PaintSpec that equals sp.
func (b *frameBuilder) intern(sp gpu.PaintSpec) *gpu.PaintSpec {
	if p, ok := b.specs[sp]; ok {
		return p
	}
	if b.specs == nil {
		b.specs = map[gpu.PaintSpec]*gpu.PaintSpec{}
	}
	p := &sp
	b.specs[sp] = p
	return p
}

// target is the rectangle of the surface that is being drawn into: the whole surface, or the
// innermost layer.
func (b *frameBuilder) target() image.Rectangle {
	if n := len(b.layers); n > 0 {
		return b.layers[n-1].rect
	}
	return image.Rect(0, 0, b.w, b.h)
}

// items returns the number of items of the pipeline.
func (b *frameBuilder) items(p pipeKind) int { return len(b.data[p]) / itemFloats[p] }

// unsupported is called for a feature the renderer cannot draw. In strict mode it returns
// GPUNotImplemented; when skipping it records the feature and returns nil and the caller
// leaves the command out.
func (b *frameBuilder) unsupported(what string) error {
	if !b.skip {
		return errors.New("GPUNotImplemented", "the GPU renderer cannot draw this yet", "feature", what)
	}
	if !b.reasons[what] {
		if b.reasons == nil {
			b.reasons = map[string]bool{}
		}
		b.reasons[what] = true
		b.stats.Unsupported = append(b.stats.Unsupported, what)
	}
	return nil
}

// command adds one command of the draw list.
func (b *frameBuilder) command(c gpu.Command) error {
	switch c.Kind {
	case gpu.CommandBeginLayer:
		if b.skipping > 0 {
			b.skipping++
			return nil
		}
		if b.culled > 0 {
			b.culled++
			return nil
		}
		return b.beginLayer(c)
	case gpu.CommandEndLayer:
		switch {
		case b.skipping > 0:
			b.skipping--
		case b.culled > 0:
			b.culled--
		default:
			return b.endLayer()
		}
		return nil
	}
	b.stats.Commands++
	switch {
	case b.skipping > 0:
		b.stats.Skipped++
		return nil
	case b.culled > 0:
		b.stats.Drawn++
		return nil
	}
	drawn, err := b.draw(c)
	if err != nil {
		return err
	}
	if drawn {
		b.stats.Drawn++
	} else {
		b.stats.Skipped++
	}
	return nil
}

// draw makes the instances of a drawing command. It reports whether the command was drawn,
// or at least needed no drawing; false means it is unsupported and was skipped.
func (b *frameBuilder) draw(c gpu.Command) (bool, error) {
	if c.Clip.IsEmpty() {
		return true, nil
	}
	m := c.Transform.Storage()
	if m[3] != 0 || m[7] != 0 || m[15] != 1 {
		return false, b.unsupported("perspective transform")
	}
	// A layer that blurs a shape reaches past the surface, and what is drawn into it does too.
	outer := gpu.PixelRect(c.Clip.Bounds).Intersect(b.target())
	if outer.Empty() {
		return true, nil
	}
	p := gpu.PaintOf(c.Op)
	if p != nil {
		switch {
		case p.ImageFilter() != nil:
			return false, b.unsupported("image filter")
		case p.MaskFilter() != nil:
			return false, b.unsupported("mask filter")
		}
	}
	sc := &shapeContext{b: b, c: c, t: gpu.AffineOf(&c.Transform), outer: outer}
	cb := c.Clip.Bounds
	sc.clip = [4]float32{float32(cb.Left), float32(cb.Top), float32(cb.Right), float32(cb.Bottom)}
	if !c.Clip.IsRect() {
		mask, empty, err := b.res.clipMask(c.Clip, b.w, b.h)
		if err != nil {
			return false, err
		}
		if empty {
			return true, nil
		}
		sc.mask = mask
	}

	var (
		ok  bool
		err error
	)
	switch v := c.Op.(type) {
	case ui.OpDrawRect:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(v.Rect, [4]ui.Radius{}, &v.Paint) })
	case ui.OpDrawRRect:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string {
			return e.RRect(gpu.RectOfRRect(v.RRect), gpu.RadiiOfRRect(v.RRect), &v.Paint)
		})
	case ui.OpDrawOval:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(v.Rect, gpu.OvalRadii(v.Rect), &v.Paint) })
	case ui.OpDrawCircle:
		r := ui.NewRectFromCircle(v.Center, v.Radius)
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(r, gpu.OvalRadii(r), &v.Paint) })
	case ui.OpDrawDRRect:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.DRRect(v.Outer, v.Inner, &v.Paint) })
	case ui.OpDrawRSuperellipse:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.Superellipse(v.RSuperellipse, &v.Paint) })
	case ui.OpDrawLine:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string {
			return e.Segments([]ui.Offset{v.P1, v.P2}, &v.Paint, true, ui.PointModeLines)
		})
	case ui.OpDrawPoints:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.Segments(v.Points, &v.Paint, false, v.Mode) })
	case ui.OpDrawPath, ui.OpDrawArc:
		ok, err = sc.pathMask(c.Op, p)
	case ui.OpDrawColor:
		fill := ui.NewPaint()
		fill.SetColor(v.Color)
		fill.SetBlendMode(v.BlendMode)
		ok, err = sc.shape(fill, func(e *gpu.ShapeEncoder) string { e.Fill(); return "" })
	case ui.OpDrawPaint:
		ok, err = sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { e.Fill(); return "" })
	case ui.OpDrawImage:
		w, h := float64(v.Image.Width()), float64(v.Image.Height())
		ok, err = sc.imageRect(v.Image, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, w, h), &v.Paint)
	case ui.OpDrawImageRect:
		ok, err = sc.imageRect(v.Image, v.Src, v.Dst, &v.Paint)
	case ui.OpDrawImageNine:
		ok, err = sc.imageNine(v)
	case ui.OpDrawAtlas:
		ok, err = sc.imageAtlas(v)
	case ui.OpDrawVertices:
		ok, err = sc.vertices(v)
	default:
		return false, b.unsupported("operation " + fmt.Sprintf("%T", c.Op))
	}
	return ok, err
}

// shapeContext is a drawing command being turned into primitives.
type shapeContext struct {
	b *frameBuilder
	c gpu.Command
	t gpu.Affine2D
	// outer is the pixels the clip reaches, clip its rectangle with fractional edges, and mask
	// the coverage of its shapes (nil when the clip is a rectangle).
	outer image.Rectangle
	clip  [4]float32
	mask  *clipMask
}

// paintInfo is what a paint gives the pipelines.
type paintInfo struct {
	blend gpu.FixedBlend
	color [4]float32 // premultiplied; for an image or a shader, its alpha in all four components
	aa    bool
	spec  *gpu.PaintSpec
}

// paintStatus says what to do with a paint.
type paintStatus int

const (
	paintDraw        paintStatus = iota // draw with the info
	paintNothing                        // the paint draws nothing: the command is done
	paintUnsupported                    // the command is skipped, or failed with the error
)

// paint reads the parts of a paint that the pipelines draw.
func (s *shapeContext) paint(p *ui.Paint, useShader bool) (info paintInfo, status paintStatus, err error) {
	return s.paintWith(p, useShader, s.t)
}

// paintWith reads a paint. With useShader false the paint's shader does not apply (images);
// toDevice is the transform the shader is set under. For paintUnsupported, unsupported has been
// called, and err is its result.
func (s *shapeContext) paintWith(p *ui.Paint, useShader bool, toDevice gpu.Affine2D) (info paintInfo, status paintStatus, err error) {
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return info, paintNothing, nil
	}
	var sp gpu.PaintSpec
	sp.SetColorFilter(p.ColorFilter(), p.InvertColors())
	info.aa = p.IsAntiAlias()
	if !s.b.setBlend(&info, &sp, mode) {
		return info, paintUnsupported, s.b.unsupported("blend mode " + mode.String())
	}
	col := p.Color()
	if sh := p.Shader(); useShader && sh != nil {
		if status, err := s.shaderSpec(&sp, sh, toDevice); status != paintDraw {
			return info, status, err
		}
		a := float32(col.Alpha()) / 255
		info.color = [4]float32{a, a, a, a}
	} else if !useShader {
		a := float32(col.Alpha()) / 255
		info.color = [4]float32{a, a, a, a}
	} else {
		a := col.A
		info.color = [4]float32{col.R * a, col.G * a, col.B * a, a}
	}
	info.spec = s.b.intern(sp)
	return info, paintDraw, nil
}

// setBlend sets how a blend mode is drawn into the paint info and spec: through the blend unit
// if it can, else through the shader that reads the pixels under the draw. False says that the
// mode is not known.
func (b *frameBuilder) setBlend(info *paintInfo, sp *gpu.PaintSpec, mode ui.BlendMode) bool {
	if blend, ok := gpu.FixedBlendOf(mode); ok {
		info.blend = blend
		return true
	}
	idx, ok := gpu.BlendIndex(mode)
	if !ok {
		return false
	}
	info.blend = replaceBlend
	sp.ReadBlend = int32(idx + 1)
	return true
}

// shaderSpec fills the shader part of a spec for the shader sh under the transform that takes the
// shader's local space to the surface.
func (s *shapeContext) shaderSpec(sp *gpu.PaintSpec, sh ui.Shader, toDevice gpu.Affine2D) (paintStatus, error) {
	switch v := sh.(type) {
	case *ui.Gradient:
		r, what := sp.SetGradient(v, toDevice)
		return s.status(r, what)
	case *ui.ImageShader:
		img := v.Image()
		if img == nil || img.IsDisposed() {
			return paintUnsupported, errors.New("GPUImageDisposed", "the image was disposed, or is nil, and cannot be drawn")
		}
		r, err := sp.SetImageShader(v, toDevice, func(img *ui.Image) (any, int, error) {
			entry, err := s.b.res.imageTexture(img)
			if err != nil {
				return nil, 0, err
			}
			return entry.tex, entry.levels, nil
		})
		if err != nil {
			if appErr, isErr := errors.FromError(err); isErr && appErr.Tag == "GPUImageTooLarge" && s.b.skip {
				return paintUnsupported, s.b.unsupported("image larger than the textures of the device")
			}
			return paintUnsupported, err
		}
		return s.status(r, "")
	}
	return paintUnsupported, s.b.unsupported("paint shader")
}

// status turns the result of a spec into what the builder does with the paint.
func (s *shapeContext) status(r gpu.PaintResult, what string) (paintStatus, error) {
	switch r {
	case gpu.PaintNothing:
		return paintNothing, nil
	case gpu.PaintUnsupported:
		return paintUnsupported, s.b.unsupported(what)
	}
	return paintDraw, nil
}

// key makes the call key of a primitive of the pipeline. A draw that reads the pixels under it is
// its own call, and its scissor box is just the pixels it covers, so that only they are copied;
// false says that there is nothing to draw.
func (s *shapeContext) key(pipe pipeKind, info paintInfo, covers image.Rectangle) (callKey, bool) {
	k := callKey{pipe: pipe, blend: info.blend, scissor: s.outer, clip: s.mask, spec: info.spec}
	if info.spec.ReadBlend != 0 {
		r := covers.Intersect(s.outer)
		if r.Empty() {
			return k, false
		}
		k.scissor = r
		s.b.seq++
		k.seq = s.b.seq
	}
	return k, true
}

// shape draws what draw makes with the paint's color and blend. A shape with no analytic
// form is drawn from the coverage atlas.
func (s *shapeContext) shape(p *ui.Paint, draw func(e *gpu.ShapeEncoder) (needsMask string)) (bool, error) {
	info, status, err := s.paint(p, true)
	if status != paintDraw {
		return status == paintNothing, err
	}
	enc := &gpu.ShapeEncoder{
		Transform: &s.c.Transform, T: s.t, Outer: s.outer, Clip: s.clip, ClipBounds: s.c.Clip.Bounds,
		Paint: gpu.ShapePaint{Color: info.color, Antialias: info.aa},
	}
	enc.Emit = func(in *gpu.ShapeInstance, covers image.Rectangle) {
		if key, ok := s.key(pipeShape, info, covers); ok {
			s.b.add(key, in[:])
		}
	}
	if what := draw(enc); what != "" {
		if op := gpu.PaintOf(s.c.Op); op != nil {
			if _, isShape := gpu.ShapeOf(s.c.Op); isShape {
				return s.pathMask(s.c.Op, op)
			}
		}
		return false, s.b.unsupported(what)
	}
	return true, nil
}

// add appends an item as part of the last draw call when it has the same state, else as a
// new one.
func (b *frameBuilder) add(key callKey, floats []float32) { b.addItems(key, floats, 1) }

// addItems appends count items, whose floats are given.
func (b *frameBuilder) addItems(key callKey, floats []float32, count int) {
	first := b.items(key.pipe)
	if n := len(b.calls); n > 0 && key.seq == 0 && key.op == opDraw && b.calls[n-1].callKey == key {
		b.calls[n-1].count += count
	} else {
		b.calls = append(b.calls, drawCall{callKey: key, first: first, count: count})
	}
	b.data[key.pipe] = append(b.data[key.pipe], floats...)
}

// blendState is the blend equation of a fixed blend.
func blendState(b gpu.FixedBlend) BlendState {
	factor := func(f gpu.BlendFactor) BlendFactor {
		switch f {
		case gpu.BlendFactorZero:
			return BlendZero
		case gpu.BlendFactorOneMinusSrcAlpha:
			return BlendOneMinusSrcAlpha
		case gpu.BlendFactorOneMinusDstAlpha:
			return BlendOneMinusDstAlpha
		}
		return BlendOne
	}
	return BlendState{Enabled: true, SrcFactor: factor(b.Src), DstFactor: factor(b.Dst)}
}
