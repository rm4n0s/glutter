package explicit

import (
	"fmt"
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The frame is what a draw list becomes before any device call is made: the data of the
// shape instances and the draw calls that use them, in order. Building it first lets the
// renderer fail on an unsupported feature before it draws anything, and upload the
// instances once.

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
	// Instances is the number of instanced shapes sent to the GPU, and Batches the number
	// of draw calls that drew them.
	Instances, Batches int
	// Unsupported lists the features that caused skipped commands, once each.
	Unsupported []string
}

// callKey is what two draws must share to be one draw call.
type callKey struct {
	blend   gpu.FixedBlend
	scissor image.Rectangle // from the top left of the target
}

// drawCall is a run of instances drawn with one state.
type drawCall struct {
	callKey
	first, count int
}

// frameBuilder turns the commands of a draw list into instances. It does not call the
// device.
type frameBuilder struct {
	w, h int
	skip bool // unsupported commands are skipped and counted instead of failing

	data  []float32 // the instances, gpu.ShapeFloats each
	calls []drawCall
	stats FrameStats
	// skipping is the depth of the unsupported group being skipped; 0 when none.
	skipping int
	// reasons are the unsupported features seen, to count each once.
	reasons map[string]bool
}

// instances returns the number of instances.
func (b *frameBuilder) instances() int { return len(b.data) / gpu.ShapeFloats }

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
		b.skipping = 1
		return b.unsupported("layer")
	case gpu.CommandEndLayer:
		if b.skipping > 0 {
			b.skipping--
		}
		return nil
	}
	b.stats.Commands++
	if b.skipping > 0 {
		b.stats.Skipped++
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
	if !c.Clip.IsRect() {
		return false, b.unsupported("clip shape")
	}
	outer := gpu.PixelRect(c.Clip.Bounds).Intersect(image.Rect(0, 0, b.w, b.h))
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

	switch v := c.Op.(type) {
	case ui.OpDrawRect:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(v.Rect, [4]ui.Radius{}, &v.Paint) })
	case ui.OpDrawRRect:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string {
			return e.RRect(gpu.RectOfRRect(v.RRect), gpu.RadiiOfRRect(v.RRect), &v.Paint)
		})
	case ui.OpDrawOval:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(v.Rect, gpu.OvalRadii(v.Rect), &v.Paint) })
	case ui.OpDrawCircle:
		r := ui.NewRectFromCircle(v.Center, v.Radius)
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.RRect(r, gpu.OvalRadii(r), &v.Paint) })
	case ui.OpDrawDRRect:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.DRRect(v.Outer, v.Inner, &v.Paint) })
	case ui.OpDrawRSuperellipse:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.Superellipse(v.RSuperellipse, &v.Paint) })
	case ui.OpDrawLine:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string {
			return e.Segments([]ui.Offset{v.P1, v.P2}, &v.Paint, true, ui.PointModeLines)
		})
	case ui.OpDrawPoints:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { return e.Segments(v.Points, &v.Paint, false, v.Mode) })
	case ui.OpDrawColor:
		fill := ui.NewPaint()
		fill.SetColor(v.Color)
		fill.SetBlendMode(v.BlendMode)
		return sc.shape(fill, func(e *gpu.ShapeEncoder) string { e.Fill(); return "" })
	case ui.OpDrawPaint:
		return sc.shape(&v.Paint, func(e *gpu.ShapeEncoder) string { e.Fill(); return "" })
	}
	return false, b.unsupported("operation " + fmt.Sprintf("%T", c.Op))
}

// shapeContext is a drawing command being turned into instances.
type shapeContext struct {
	b *frameBuilder
	c gpu.Command
	t gpu.Affine2D
	// outer is the pixels the clip reaches, and clip its rectangle with fractional edges.
	outer image.Rectangle
	clip  [4]float32
}

// shape draws what draw makes, with the paint's color and blend. A paint that draws
// nothing, or one the renderer cannot draw, ends the command before it makes anything.
func (s *shapeContext) shape(p *ui.Paint, draw func(e *gpu.ShapeEncoder) (needsMask string)) (bool, error) {
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return true, nil
	}
	blend, ok := gpu.FixedBlendOf(mode)
	switch {
	case !ok:
		return false, s.b.unsupported("blend mode " + mode.String())
	case p.Shader() != nil:
		return false, s.b.unsupported("shader")
	case p.ColorFilter() != nil || p.InvertColors():
		return false, s.b.unsupported("color filter")
	}
	col := p.Color()
	a := col.A
	enc := &gpu.ShapeEncoder{
		Transform: &s.c.Transform, T: s.t, Outer: s.outer, Clip: s.clip, ClipBounds: s.c.Clip.Bounds,
		Paint: gpu.ShapePaint{Color: [4]float32{col.R * a, col.G * a, col.B * a, a}, Antialias: p.IsAntiAlias()},
	}
	key := callKey{blend: blend, scissor: s.outer}
	enc.Emit = func(in *gpu.ShapeInstance, _ image.Rectangle) {
		s.b.add(key, in[:])
	}
	if what := draw(enc); what != "" {
		// A shape with no analytic form is drawn from the coverage atlas (step 20.4).
		return false, s.b.unsupported(what)
	}
	return true, nil
}

// add appends an instance as part of the last draw call when it has the same state, else
// as a new one.
func (b *frameBuilder) add(key callKey, floats []float32) {
	first := b.instances()
	if n := len(b.calls); n > 0 && b.calls[n-1].callKey == key {
		b.calls[n-1].count++
	} else {
		b.calls = append(b.calls, drawCall{callKey: key, first: first, count: 1})
	}
	b.data = append(b.data, floats...)
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
