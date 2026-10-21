package gles

import (
	"fmt"
	"image"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The shape pipeline draws rectangles, rounded rectangles, ovals, circles,
// rounded superellipses, the region between two rounded rectangles, lines and
// points, and fills with a color. Each shape is one instance of a quad whose
// fragment shader finds the coverage of every pixel from a signed distance
// field (shaders/shape.frag.glsl). There is no multisampling.

// The shape instance and its constants are shared with the explicit renderer (gpu/shapes.go).
const (
	shapeFloats = gpu.ShapeFloats
	shapeStride = gpu.ShapeStride
)

// The kinds of shape of the shader.
const (
	kindRRect   = gpu.ShapeKindRRect
	kindSuper   = gpu.ShapeKindSuper
	kindSegment = gpu.ShapeKindSegment
	kindFill    = gpu.ShapeKindFill
	kindBox     = gpu.ShapeKindBox
	kindOutline = gpu.ShapeKindOutline
	kindShadow  = gpu.ShapeKindShadow
)

// The flags of the shader.
const (
	flagHasB       = gpu.ShapeFlagHasB
	flagAntialias  = gpu.ShapeFlagAntialias
	flagSimilarity = gpu.ShapeFlagSimilarity
)

// blendFunc is a pair of glBlendFuncSeparate factors.
type blendFunc struct{ srcRGB, dstRGB, srcA, dstA uint32 }

// blendFuncs are the blend modes that glBlendFunc makes exactly when the shape's
// coverage scales the source (gpu.FixedBlendOf). The others read a copy of the pixels
// under the draw in the shader (paintSpec.readBlend), which writes the whole result:
// replaceBlend.
var blendFuncs = fixedBlendFuncs()

func fixedBlendFuncs() map[ui.BlendMode]blendFunc {
	factor := map[gpu.BlendFactor]uint32{
		gpu.BlendFactorZero: glZero, gpu.BlendFactorOne: glOne,
		gpu.BlendFactorOneMinusSrcAlpha: glOneMinusSrcAlpha, gpu.BlendFactorOneMinusDstAlpha: glOneMinusDstAlpha,
	}
	m := map[ui.BlendMode]blendFunc{}
	for _, mode := range []ui.BlendMode{ui.BlendModeSrcOver, ui.BlendModeDstOver, ui.BlendModePlus, ui.BlendModeDstOut} {
		b, _ := gpu.FixedBlendOf(mode)
		s, d := factor[b.Src], factor[b.Dst]
		m[mode] = blendFunc{s, d, s, d}
	}
	return m
}

// replaceBlend writes the shader's output as it is.
var replaceBlend = blendFunc{glOne, glZero, glOne, glZero}

// affine2D and its helpers are shared with the explicit renderer (gpu/affine.go).
type affine2D = gpu.Affine2D

func affineOf(m *ui.Matrix4) affine2D { return gpu.AffineOf(m) }

func translation(x, y float64) affine2D { return gpu.Translation(x, y) }

// frameBuilder turns the commands of a draw list into instances. It does not
// call GL.
type frameBuilder struct {
	w, h  int
	skip  bool // unsupported commands are skipped and counted instead of failing
	frame frame
	// res holds the caches that paths, clip shapes and images need; without it
	// (in tests of the builder alone) they are unsupported.
	res   *resources
	stats FrameStats
	// skipping is the depth of the unsupported group being skipped; 0 when none.
	skipping int
	// reasons are the unsupported features seen, to count each once.
	reasons map[string]bool
	// specs interns the paint specs of the frame.
	specs map[paintSpec]*paintSpec
	// seq numbers the draw calls that read the pixels under them, which are never
	// merged.
	seq int
	// layers are the groups being drawn, innermost last; culled is the depth of a
	// group with nothing to draw, whose commands are dropped.
	layers []*layerRec
	culled int
	// cache is the raster cache, or nil without one.
	cache *layerCache
}

// FrameStats says what the last frame did.
//
// Flutter: no counterpart
type FrameStats struct {
	// Commands is the number of drawing commands in the scene.
	Commands int
	// Drawn is the number of commands drawn.
	Drawn int
	// Skipped is the number of commands left out because the renderer cannot draw
	// them yet; only a renderer with SkipUnsupported skips.
	Skipped int
	// Instances is the number of instanced primitives sent to the GPU (shapes,
	// coverage tiles and image rectangles), Vertices the number of vertices, and
	// Batches the number of draw calls that drew them.
	Instances, Vertices, Batches int
	// TilesRasterized, ClipMasksBuilt and ImagesUploaded count the work the frame
	// did that an earlier frame's cache could have saved: coverage tiles
	// rasterized, clip masks made and images uploaded.
	TilesRasterized, ClipMasksBuilt, ImagesUploaded int
	// CachedLayersDrawn is the number of retained layers drawn from the raster
	// cache's textures, and CachedLayersMade the number drawn into them.
	CachedLayersDrawn, CachedLayersMade int
	// DamagedPixels is the number of pixels the frame told the compositor to
	// update, and DamageRects the number of rectangles they were in: the whole
	// window and 0 when the frame was presented whole. Both are 0 unless
	// RendererOpts.Damage is set and the display can swap with damage.
	DamagedPixels, DamageRects int
	// Unsupported lists the features that caused skipped commands, once each.
	Unsupported []string
}

// unsupported is called for a feature the renderer cannot draw. In strict mode it
// returns GLNotImplemented; when skipping it records the feature and returns nil
// and the caller leaves the command out.
func (b *frameBuilder) unsupported(what string) error {
	if !b.skip {
		return errors.New("GLNotImplemented", "the GPU renderer cannot draw this yet", "feature", what)
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

// draw makes the instances of a drawing command. It reports whether the command
// was drawn, or at least needed no drawing; false means it is unsupported and
// was skipped.
func (b *frameBuilder) draw(c gpu.Command) (bool, error) {
	if c.Clip.IsEmpty() {
		return true, nil
	}
	m := c.Transform.Storage()
	if m[3] != 0 || m[7] != 0 || m[15] != 1 {
		return false, b.unsupported("perspective transform")
	}
	switch c.Op.(type) {
	case ui.OpDrawPath, ui.OpDrawArc, ui.OpDrawImage, ui.OpDrawImageRect, ui.OpDrawImageNine, ui.OpDrawAtlas, ui.OpDrawVertices:
		// These need the caches of the coverage atlas, images and vertices.
		if b.res == nil {
			return false, b.unsupported("operation " + typeName(c.Op))
		}
	}
	if p, with := paintAccess(c.Op); p != nil && p.ImageFilter() != nil {
		return b.filteredDraw(c, p, with)
	}
	ctx := &shapeContext{b: b, c: c, t: affineOf(&c.Transform)}
	if done, ok, err := ctx.setupClip(); done {
		return ok, err
	}
	if p, with := paintAccess(c.Op); p != nil && p.MaskFilter() != nil {
		if handled, ok, err := ctx.maskFiltered(p, with); handled {
			return ok, err
		}
	}
	var (
		ok  bool
		err error
	)
	switch v := c.Op.(type) {
	case ui.OpDrawShadow:
		ok, err = ctx.shadow(v)
	case ui.OpDrawRect:
		ok, err = ctx.rrect(v.Rect, [4]ui.Radius{}, &v.Paint)
	case ui.OpDrawRRect:
		ok, err = ctx.rrect(rectOfRRect(v.RRect), radiiOf(v.RRect), &v.Paint)
	case ui.OpDrawOval:
		ok, err = ctx.rrect(v.Rect, ovalRadii(v.Rect), &v.Paint)
	case ui.OpDrawCircle:
		r := ui.NewRectFromCircle(v.Center, v.Radius)
		ok, err = ctx.rrect(r, ovalRadii(r), &v.Paint)
	case ui.OpDrawDRRect:
		ok, err = ctx.drrect(v)
	case ui.OpDrawRSuperellipse:
		ok, err = ctx.superellipse(v)
	case ui.OpDrawLine:
		ok, err = ctx.segments([]ui.Offset{v.P1, v.P2}, &v.Paint, c.Op, ui.PointModeLines)
	case ui.OpDrawPoints:
		ok, err = ctx.segments(v.Points, &v.Paint, c.Op, v.Mode)
	case ui.OpDrawPath:
		ok, err = ctx.pathMask(c.Op, &v.Paint)
	case ui.OpDrawArc:
		ok, err = ctx.pathMask(c.Op, &v.Paint)
	case ui.OpDrawImage:
		if err := checkImage(v.Image); err != nil {
			return false, err
		}
		w, h := float64(v.Image.Width()), float64(v.Image.Height())
		ok, err = ctx.imageRect(v.Image, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, w, h), &v.Paint)
	case ui.OpDrawImageRect:
		ok, err = ctx.imageRect(v.Image, v.Src, v.Dst, &v.Paint)
	case ui.OpDrawImageNine:
		ok, err = ctx.imageNine(v)
	case ui.OpDrawAtlas:
		ok, err = ctx.imageAtlas(v)
	case ui.OpDrawVertices:
		ok, err = ctx.vertices(v)
	case ui.OpDrawColor:
		p := ui.NewPaint()
		p.SetColor(v.Color)
		p.SetBlendMode(v.BlendMode)
		ok, err = ctx.fill(p)
	case ui.OpDrawPaint:
		ok, err = ctx.fill(&v.Paint)
	default:
		return false, b.unsupported("operation " + typeName(c.Op))
	}
	if err == errUseMask {
		// The shape has no analytic form: its coverage comes from the atlas.
		return ctx.pathMask(c.Op, paintOfOp(c.Op))
	}
	return ok, err
}

// errUseMask is returned by the analytic shapes for a shape or stroke they cannot
// draw, which the coverage atlas draws instead.
var errUseMask = errors.New("UseMask", "the shape has no analytic form")

// paintOfOp returns the paint of the shape-drawing operations.
func paintOfOp(op ui.DisplayOp) *ui.Paint {
	switch v := op.(type) {
	case ui.OpDrawRect:
		return &v.Paint
	case ui.OpDrawRRect:
		return &v.Paint
	case ui.OpDrawOval:
		return &v.Paint
	case ui.OpDrawCircle:
		return &v.Paint
	case ui.OpDrawDRRect:
		return &v.Paint
	case ui.OpDrawRSuperellipse:
		return &v.Paint
	case ui.OpDrawLine:
		return &v.Paint
	case ui.OpDrawPoints:
		return &v.Paint
	}
	return ui.NewPaint()
}

// maskOr is for a shape the analytic pipeline cannot draw: the coverage atlas
// draws it, or without one the feature is unsupported.
func (s *shapeContext) maskOr(what string) (bool, error) {
	if s.b.res != nil {
		return false, errUseMask
	}
	return false, s.b.unsupported(what)
}

// shapeContext is a drawing command being turned into primitives.
type shapeContext struct {
	b *frameBuilder
	c gpu.Command
	t affine2D

	// What the clip leaves, set by setupClip: the pixels it reaches (the scissor
	// box), its rectangle with fractional edges, and the mask of its shapes.
	scissor [4]int
	outer   image.Rectangle
	clip    [4]float32
	mask    *clipMask
}

// setupClip works out the scissor box, the clip rectangle and the mask of the
// command's clip. done says that the command needs no more work: ok tells
// whether it counts as drawn.
func (s *shapeContext) setupClip() (done, ok bool, err error) {
	// A layer that blurs a shape reaches past the surface, and what is drawn into it
	// does too.
	s.outer = pixelRect(s.c.Clip.Bounds).Intersect(s.b.target())
	if s.outer.Empty() {
		return true, true, nil
	}
	s.scissor = [4]int{s.outer.Min.X, s.outer.Min.Y, s.outer.Dx(), s.outer.Dy()}
	cb := s.c.Clip.Bounds
	s.clip = [4]float32{float32(cb.Left), float32(cb.Top), float32(cb.Right), float32(cb.Bottom)}
	if s.c.Clip.IsRect() {
		return false, false, nil
	}
	if s.b.res == nil {
		return true, false, s.b.unsupported("clip shape")
	}
	mask, empty, err := s.b.res.clipMask(s.c.Clip, s.b.w, s.b.h)
	if err != nil {
		return true, false, err
	}
	if empty {
		return true, true, nil
	}
	s.mask = mask
	return false, false, nil
}

// scale is the factor that lengths in the shape's units grow by.
func (s *shapeContext) scale() float64 { return math.Sqrt(math.Abs(s.t.Det())) }

// paintInfo is what a paint gives the pipelines.
type paintInfo struct {
	color [4]float32 // premultiplied; for a shader, its alpha in all four components
	blend blendFunc
	aa    bool
	spec  *paintSpec
}

// paintStatus says what to do with a paint.
type paintStatus int

const (
	paintDraw        paintStatus = iota // draw with the info
	paintNothing                        // the paint draws nothing: the command is done
	paintUnsupported                    // the command is skipped, or failed with the error
)

// paint reads the parts of a paint that the shape pipelines draw. For
// paintUnsupported, unsupported has been called, and err is its result.
func (s *shapeContext) paint(p *ui.Paint) (info paintInfo, status paintStatus, err error) {
	return s.paintWith(p, true, s.t)
}

// paintWith reads a paint. With useShader false the paint's shader and mask
// filter do not apply (images); toDevice is the transform the shader is set
// under.
func (s *shapeContext) paintWith(p *ui.Paint, useShader bool, toDevice affine2D) (info paintInfo, status paintStatus, err error) {
	// The paint's mask filter and image filter are not part of the info: the commands
	// that have them are drawn through layers (effects.go).
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return info, paintNothing, nil
	}
	var sp paintSpec
	filterSpec(&sp, p.ColorFilter(), p.InvertColors())
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

// key makes the call key of a primitive that covers the pixels of bounds. A draw
// that reads the pixels under it is its own call, and its scissor box is just
// those pixels, so that only they are copied; false says that there is nothing to
// draw.
func (s *shapeContext) key(pipe pipeline, info paintInfo, tex *Texture, bounds image.Rectangle) (callKey, bool) {
	k := callKey{pipe: pipe, blend: info.blend, scissor: s.scissor, tex: tex, clip: s.mask, spec: info.spec}
	if info.spec.readBlend != 0 {
		r := bounds.Intersect(s.outer)
		if r.Empty() {
			return k, false
		}
		k.scissor = [4]int{r.Min.X, r.Min.Y, r.Dx(), r.Dy()}
		s.b.seq++
		k.seq = s.b.seq
	}
	return k, true
}

// instance is the attributes of one shape, in the order of the shader.
type instance = gpu.ShapeInstance

// encoder returns the shape encoder of the command, which adds each instance it makes to
// the frame under the key of info.
func (s *shapeContext) encoder(info paintInfo) *gpu.ShapeEncoder {
	return &gpu.ShapeEncoder{
		Transform: &s.c.Transform, T: s.t, Outer: s.outer, Clip: s.clip, ClipBounds: s.c.Clip.Bounds,
		Paint: gpu.ShapePaint{Color: info.color, Antialias: info.aa},
		Emit: func(in *instance, covers image.Rectangle) {
			if key, ok := s.key(pipeShape, info, nil, covers); ok {
				s.b.frame.add(key, in[:], 1)
			}
		},
	}
}

// placePadded finishes an instance (gpu.ShapeEncoder.PlacePadded) and adds it to the frame.
func (s *shapeContext) placePadded(in *instance, local ui.Rect, kind, flags int, info paintInfo, alpha float32, pad float64) (bool, error) {
	s.encoder(info).PlacePadded(in, local, kind, flags, alpha, pad)
	return true, nil
}

// analytic reports the result of an encoder method: a shape that needs the coverage atlas
// is drawn by it, or without one is unsupported.
func (s *shapeContext) analytic(needsMask string) (bool, error) {
	if needsMask == "" {
		return true, nil
	}
	return s.maskOr(needsMask)
}

// rrectParts, partsOf and the helpers of rounded rectangles are shared with the explicit
// renderer (gpu/shapes.go).
type rrectParts = gpu.RRectParts

func rectOfRRect(r ui.RRect) ui.Rect { return gpu.RectOfRRect(r) }

func radiiOf(r ui.RRect) [4]ui.Radius { return gpu.RadiiOfRRect(r) }

func ovalRadii(r ui.Rect) [4]ui.Radius { return gpu.OvalRadii(r) }

// rrect draws a rounded rectangle, filled or stroked.
func (s *shapeContext) rrect(r ui.Rect, radii [4]ui.Radius, p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	return s.analytic(s.encoder(info).RRect(r, radii, p))
}

// drrect draws the region between two rounded rectangles.
func (s *shapeContext) drrect(v ui.OpDrawDRRect) (bool, error) {
	info, status, err := s.paint(&v.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	return s.analytic(s.encoder(info).DRRect(v.Outer, v.Inner, &v.Paint))
}

// superellipse draws a filled rounded superellipse.
func (s *shapeContext) superellipse(v ui.OpDrawRSuperellipse) (bool, error) {
	info, status, err := s.paint(&v.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	return s.analytic(s.encoder(info).Superellipse(v.RSuperellipse, &v.Paint))
}

// segments draws lines and points with the paint's width and cap.
func (s *shapeContext) segments(pts []ui.Offset, p *ui.Paint, op ui.DisplayOp, mode ui.PointMode) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	_, isLine := op.(ui.OpDrawLine)
	return s.analytic(s.encoder(info).Segments(pts, p, isLine, mode))
}

// fill paints the whole clip with the paint's color.
func (s *shapeContext) fill(p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	s.encoder(info).Fill()
	return true, nil
}

// typeName is the name of the dynamic type of an operation, for the list of
// unsupported features.
func typeName(op ui.DisplayOp) string { return fmt.Sprintf("%T", op) }
