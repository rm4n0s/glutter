// Package opsig makes signatures of drawing operations: a 64-bit number that is the same
// for two operations that draw the same pixels under the same state, so that the damage
// tracker can tell that a picture that was recorded again draws what the picture before it
// drew (SPEC §9.1, "Damage"). A signature is made of the contents of the operation: its
// geometry, its paint with the contents of the paint's gradient and filters, a path by its
// fingerprint, a paragraph by what it draws. What has no cheap contents is taken by its
// identity (the pixels of an image, a Vertices, a shader of an unknown kind): the signature
// then holds an address, and means something only while the object is alive, which it is
// for as long as the picture that draws it is kept.
//
// Two different operations have the same signature with the probability of a collision of
// a 64-bit hash, which the damage tracker accepts for its tiles already.
//
// Flutter: no counterpart (DisplayList::Equals, which flutter::DiffContext compares the
// display lists of two layers with)
package opsig

import (
	"math"
	"reflect"

	"github.com/rm4n0s/glutter/ui"
)

// Signed is a paragraph layout that knows the signature of what it draws, without drawing
// it. The layout of the text engine is one (it keeps the signature until it is laid out
// again), and the frozen paragraph of the offscreen detacher.
//
// Flutter: no counterpart
type Signed interface {
	// PaintSignature is the signature of what PaintParagraph emits (Draws), and false
	// when something it emits has none.
	PaintSignature() (uint64, bool)
}

// hash is a 64-bit hash that takes a word at a time, mixing every bit of the word into
// every bit of the state (the mixing of gpu.Hasher, which this package cannot import).
type hash uint64

const seed hash = 14695981039346656037

func (h *hash) u(v uint64) {
	x := (uint64(*h) ^ v) * 0x9E3779B97F4A7C15
	x ^= x >> 32
	x *= 0xD6E8FEB86659FD93
	x ^= x >> 29
	*h = hash(x)
}

func (h *hash) f(vs ...float64) {
	for _, v := range vs {
		h.u(math.Float64bits(v))
	}
}

func (h *hash) f32(vs ...float32) {
	for _, v := range vs {
		h.u(uint64(math.Float32bits(v)))
	}
}

func (h *hash) b(v bool) {
	if v {
		h.u(1)
	} else {
		h.u(0)
	}
}

func (h *hash) rect(r ui.Rect)     { h.f(r.Left, r.Top, r.Right, r.Bottom) }
func (h *hash) offset(o ui.Offset) { h.f(o.Dx, o.Dy) }

func (h *hash) rrect(r ui.RRect) {
	h.f(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY,
		r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
}

func (h *hash) color(c ui.Color) {
	h.f32(c.A, c.R, c.G, c.B)
	h.u(uint64(c.ColorSpace))
}

func (h *hash) matrix(m *ui.Matrix4) {
	s := m.Storage()
	h.f32(s[:]...)
}

// path adds the contents of a path: its fingerprint, which the path keeps up to date.
func (h *hash) path(p *ui.Path) {
	if p == nil {
		h.u(0)
		return
	}
	h.u(1)
	h.u(p.Fingerprint())
}

// address adds the identity of what a pointer, a map or a function refers to, and says
// false for a value that has none.
func (h *hash) address(x any) bool {
	if x == nil {
		h.u(0)
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Func, reflect.UnsafePointer:
		h.u(uint64(v.Pointer()))
		return true
	}
	return false
}

// image adds the identity of the pixels of an image: every handle on the same pixels (a
// clone, as the offscreen detacher makes) gives the same. The pixels of an image are never
// written.
func (h *hash) image(img *ui.Image) {
	if img == nil {
		h.u(0)
		return
	}
	h.u(1)
	h.u(uint64(img.Width()))
	h.u(uint64(img.Height()))
	if pix := img.Pixels(); len(pix) > 0 {
		h.u(uint64(reflect.ValueOf(pix).Pointer()))
		return
	}
	h.u(uint64(reflect.ValueOf(img).Pointer()))
}

// The kinds of things a signature is made of; each is added before the contents, so that
// two kinds with the same numbers differ.
const (
	kindNone = iota
	kindSave
	kindSaveLayer
	kindRestore
	kindTransform
	kindClipRect
	kindClipRRect
	kindClipRSuperellipse
	kindClipPath
	kindDrawColor
	kindDrawPaint
	kindDrawLine
	kindDrawRect
	kindDrawRRect
	kindDrawDRRect
	kindDrawRSuperellipse
	kindDrawOval
	kindDrawCircle
	kindDrawArc
	kindDrawPath
	kindDrawImage
	kindDrawImageRect
	kindDrawImageNine
	kindDrawPicture
	kindDrawParagraph
	kindDrawPoints
	kindDrawVertices
	kindDrawAtlas
	kindDrawShadow
	kindGradient
	kindImageShader
	kindOtherShader
	kindColorFilter
	kindBlur
	kindDilate
	kindErode
	kindMatrixFilter
	kindCompose
	kindOtherFilter
)

// shader adds a shader: a gradient by its contents, an image shader by its image and its
// parameters, any other by its identity.
func (h *hash) shader(sh ui.Shader) bool {
	switch v := sh.(type) {
	case nil:
		h.u(kindNone)
	case *ui.Gradient:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindGradient)
		h.u(uint64(v.Kind()))
		h.offset(v.From())
		h.offset(v.To())
		h.offset(v.Center())
		h.offset(v.Focal())
		h.f(v.Radius(), v.FocalRadius(), v.StartAngle(), v.EndAngle())
		colors := v.Colors()
		h.u(uint64(len(colors)))
		for _, c := range colors {
			h.color(c)
		}
		stops := v.ColorStops()
		h.u(uint64(len(stops)))
		h.f(stops...)
		h.u(uint64(v.TileMode()))
		m, has := v.Matrix4()
		h.b(has)
		if has {
			h.matrix(m)
		}
	case *ui.ImageShader:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindImageShader)
		h.image(v.Image())
		h.u(uint64(v.TileModeX()))
		h.u(uint64(v.TileModeY()))
		h.u(uint64(v.FilterQuality()))
		h.matrix(v.Matrix4())
	default:
		h.u(kindOtherShader)
		return h.address(sh)
	}
	return true
}

func (h *hash) colorFilter(f *ui.ColorFilter) {
	if f == nil {
		h.u(kindNone)
		return
	}
	h.u(kindColorFilter)
	h.u(uint64(f.Kind()))
	h.color(f.Color())
	h.u(uint64(f.BlendMode()))
	m := f.Matrix()
	h.f(m[:]...)
}

// imageFilter adds an image filter: the filters of the engine by their contents, any other
// by its identity.
func (h *hash) imageFilter(f ui.ImageFilter) bool {
	switch v := f.(type) {
	case nil:
		h.u(kindNone)
	case *ui.ColorFilter:
		h.colorFilter(v)
	case *ui.BlurImageFilter:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindBlur)
		h.f(v.SigmaX(), v.SigmaY())
		h.u(uint64(v.TileMode()))
		bounds, has := v.Bounds()
		h.b(has)
		h.rect(bounds)
	case *ui.DilateImageFilter:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindDilate)
		h.f(v.RadiusX(), v.RadiusY())
	case *ui.ErodeImageFilter:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindErode)
		h.f(v.RadiusX(), v.RadiusY())
	case *ui.MatrixImageFilter:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindMatrixFilter)
		h.matrix(v.Matrix())
		h.u(uint64(v.FilterQuality()))
	case *ui.ComposeImageFilter:
		if v == nil {
			h.u(kindNone)
			return true
		}
		h.u(kindCompose)
		return h.imageFilter(v.Outer()) && h.imageFilter(v.Inner())
	default:
		h.u(kindOtherFilter)
		return h.address(f)
	}
	return true
}

// paint adds everything of a paint that a drawing can depend on, read through the getters,
// so that a paint with a default left unset equals one with the default set.
func (h *hash) paint(p *ui.Paint) bool {
	h.b(p.IsAntiAlias())
	h.color(p.Color())
	h.u(uint64(p.BlendMode()))
	h.u(uint64(p.Style()))
	h.f(p.StrokeWidth(), p.StrokeMiterLimit())
	h.u(uint64(p.StrokeCap()))
	h.u(uint64(p.StrokeJoin()))
	h.u(uint64(p.FilterQuality()))
	h.b(p.InvertColors())
	if mf := p.MaskFilter(); mf != nil {
		h.u(1)
		h.u(uint64(mf.Style()))
		h.f(mf.Sigma())
	} else {
		h.u(0)
	}
	h.colorFilter(p.ColorFilter())
	return h.shader(p.Shader()) && h.imageFilter(p.ImageFilter())
}

// op adds an operation; false for one that has no signature.
func (h *hash) op(op ui.DisplayOp) bool {
	switch v := op.(type) {
	case ui.OpSave:
		h.u(kindSave)
	case ui.OpSaveLayer:
		h.u(kindSaveLayer)
		h.b(v.HasBounds)
		h.rect(v.Bounds)
		return h.paint(&v.Paint)
	case ui.OpRestore:
		h.u(kindRestore)
	case ui.OpTransform:
		h.u(kindTransform)
		h.matrix(&v.Matrix)
	case ui.OpClipRect:
		h.u(kindClipRect)
		h.rect(v.Rect)
		h.u(uint64(v.Op))
		h.b(v.AntiAlias)
	case ui.OpClipRRect:
		h.u(kindClipRRect)
		h.rrect(v.RRect)
		h.b(v.AntiAlias)
	case ui.OpClipRSuperellipse:
		h.u(kindClipRSuperellipse)
		h.rrect(ui.RRect(v.RSuperellipse))
		h.b(v.AntiAlias)
	case ui.OpClipPath:
		h.u(kindClipPath)
		h.path(v.Path)
		h.b(v.AntiAlias)
	case ui.OpDrawColor:
		h.u(kindDrawColor)
		h.color(v.Color)
		h.u(uint64(v.BlendMode))
	case ui.OpDrawPaint:
		h.u(kindDrawPaint)
		return h.paint(&v.Paint)
	case ui.OpDrawLine:
		h.u(kindDrawLine)
		h.offset(v.P1)
		h.offset(v.P2)
		return h.paint(&v.Paint)
	case ui.OpDrawRect:
		h.u(kindDrawRect)
		h.rect(v.Rect)
		return h.paint(&v.Paint)
	case ui.OpDrawRRect:
		h.u(kindDrawRRect)
		h.rrect(v.RRect)
		return h.paint(&v.Paint)
	case ui.OpDrawDRRect:
		h.u(kindDrawDRRect)
		h.rrect(v.Outer)
		h.rrect(v.Inner)
		return h.paint(&v.Paint)
	case ui.OpDrawRSuperellipse:
		h.u(kindDrawRSuperellipse)
		h.rrect(ui.RRect(v.RSuperellipse))
		return h.paint(&v.Paint)
	case ui.OpDrawOval:
		h.u(kindDrawOval)
		h.rect(v.Rect)
		return h.paint(&v.Paint)
	case ui.OpDrawCircle:
		h.u(kindDrawCircle)
		h.offset(v.Center)
		h.f(v.Radius)
		return h.paint(&v.Paint)
	case ui.OpDrawArc:
		h.u(kindDrawArc)
		h.rect(v.Rect)
		h.f(v.StartAngle, v.SweepAngle)
		h.b(v.UseCenter)
		return h.paint(&v.Paint)
	case ui.OpDrawPath:
		h.u(kindDrawPath)
		h.path(v.Path)
		return h.paint(&v.Paint)
	case ui.OpDrawImage:
		h.u(kindDrawImage)
		h.image(v.Image)
		h.offset(v.Offset)
		return h.paint(&v.Paint)
	case ui.OpDrawImageRect:
		h.u(kindDrawImageRect)
		h.image(v.Image)
		h.rect(v.Src)
		h.rect(v.Dst)
		return h.paint(&v.Paint)
	case ui.OpDrawImageNine:
		h.u(kindDrawImageNine)
		h.image(v.Image)
		h.rect(v.Center)
		h.rect(v.Dst)
		return h.paint(&v.Paint)
	case ui.OpDrawPicture:
		h.u(kindDrawPicture)
		sig, ok := Picture(v.Picture)
		h.u(sig)
		return ok
	case ui.OpDrawParagraph:
		h.u(kindDrawParagraph)
		h.offset(v.Offset)
		sig, ok := Paragraph(v.Paragraph)
		h.u(sig)
		return ok
	case ui.OpDrawPoints:
		h.u(kindDrawPoints)
		h.u(uint64(v.Mode))
		h.u(uint64(len(v.Points)))
		for _, p := range v.Points {
			h.offset(p)
		}
		return h.paint(&v.Paint)
	case ui.OpDrawVertices:
		h.u(kindDrawVertices)
		h.u(uint64(v.BlendMode))
		// A Vertices is immutable, and can be large: its identity stands for it.
		if v.Vertices == nil {
			h.u(0)
		} else {
			h.u(uint64(reflect.ValueOf(v.Vertices).Pointer()))
		}
		return h.paint(&v.Paint)
	case ui.OpDrawAtlas:
		h.u(kindDrawAtlas)
		h.image(v.Atlas)
		h.u(uint64(len(v.Transforms)))
		for _, t := range v.Transforms {
			h.f32(t.Scos, t.Ssin, t.Tx, t.Ty)
		}
		h.u(uint64(len(v.Rects)))
		for _, r := range v.Rects {
			h.rect(r)
		}
		h.u(uint64(len(v.Colors)))
		for _, c := range v.Colors {
			h.color(c)
		}
		h.u(uint64(v.BlendMode))
		h.b(v.HasCullRect)
		h.rect(v.CullRect)
		return h.paint(&v.Paint)
	case ui.OpDrawShadow:
		h.u(kindDrawShadow)
		h.path(v.Path)
		h.color(v.Color)
		h.f(v.Elevation)
		h.b(v.TransparentOccluder)
	default:
		return false
	}
	return true
}

// Op returns the signature of an operation of a display list, and false for an operation
// that has none: a kind this package does not know, a paint with a shader or a filter that
// has no address, a paragraph that cannot draw itself. A picture and a paragraph are taken
// by what they draw.
//
// Flutter: no counterpart
func Op(op ui.DisplayOp) (uint64, bool) {
	h := seed
	ok := h.op(op)
	return uint64(h), ok
}

// Picture returns the signature of everything a picture records, in order, and false when
// one of its operations has none. It reads every operation: the caller keeps the result for
// a picture it sees again.
//
// Flutter: no counterpart
func Picture(p *ui.Picture) (uint64, bool) {
	h := seed
	if p == nil {
		return uint64(h), true
	}
	ops := p.DisplayList().Ops()
	h.u(uint64(len(ops)))
	for _, op := range ops {
		if !h.op(op) {
			return uint64(h), false
		}
	}
	return uint64(h), true
}

// Paragraph returns the signature of what a paragraph draws as it is laid out now, at its
// own origin, and false for a paragraph that cannot draw itself or that draws something
// without a signature. A layout that is Signed is asked; any other is painted, which costs
// the length of its text.
//
// Flutter: no counterpart
func Paragraph(p *ui.Paragraph) (uint64, bool) {
	if p == nil {
		return uint64(seed), true
	}
	layout := p.Layouter()
	if s, ok := layout.(Signed); ok {
		return s.PaintSignature()
	}
	painter, ok := layout.(ui.ParagraphPainter)
	if !ok {
		return 0, false
	}
	var d Draws
	painter.PaintParagraph(d.Add)
	return d.Sum()
}

// Draws makes the signature of the drawing of a paragraph from the draws that its painter
// emits, one after another. The zero value has no draws.
//
// Flutter: no counterpart
type Draws struct {
	h      hash
	n      uint64
	failed bool
}

// Add adds a draw.
//
// Flutter: no counterpart
func (d *Draws) Add(draw ui.ParagraphDraw) {
	if d.n == 0 {
		d.h = seed
	}
	d.n++
	if !d.h.op(draw.Op) {
		d.failed = true
	}
	d.h.offset(draw.Offset)
	d.h.b(draw.Snap)
	if draw.Matrix != nil {
		d.h.u(1)
		d.h.matrix(draw.Matrix)
	} else {
		d.h.u(0)
	}
}

// Sum returns the signature of the draws added so far, and false when one of them has
// none.
//
// Flutter: no counterpart
func (d *Draws) Sum() (uint64, bool) {
	h := d.h
	if d.n == 0 {
		h = seed
	}
	h.u(d.n)
	return uint64(h), !d.failed
}
