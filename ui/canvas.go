package ui

import (
	"math"

	"github.com/rm4n0s/errors"
)

// Canvas is an interface for recording graphical operations. It holds the state
// a drawing needs (the current transform and clip, and a stack of saved ones)
// and records every drawing command into a display list, which the
// PictureRecorder it was created for turns into a Picture.
//
// A Canvas is not safe for use from several goroutines.
//
// Flutter: ui/painting.dart Canvas
type Canvas struct {
	recorder *PictureRecorder
	ops      []DisplayOp
	cullRect Rect
	state    canvasState
	stack    []canvasState
}

// canvasState is what Save pushes and Restore pops.
type canvasState struct {
	ctm Matrix4
	// clip is the bounds of the clip in device coordinates: the cull rectangle
	// intersected with the bounds of every clip shape. A zero rectangle means
	// everything is clipped away.
	clip Rect
}

// CanvasOpts are the optional arguments of NewCanvas.
//
// Flutter: ui/painting.dart Canvas
type CanvasOpts struct {
	// CullRect is a bound on what will be drawn, in the coordinates the canvas
	// starts with; the display list may leave out what is outside it. Nil means
	// Rect.largest, as in Flutter.
	CullRect *Rect
}

// NewCanvas creates a canvas that records into the recorder, which starts
// recording. It panics with the tag CanvasRecorderBusy when the recorder is
// already recording.
//
// Flutter: ui/painting.dart Canvas
func NewCanvas(recorder *PictureRecorder, opts ...CanvasOpts) *Canvas {
	o := oneOpts(opts)
	if recorder.IsRecording() {
		panic(errors.New(tagCanvasRecorderBusy, `"recorder" must not already be associated with another Canvas.`))
	}
	cull := RectLargest
	if o.CullRect != nil {
		cull = *o.CullRect
	}
	c := &Canvas{recorder: recorder, cullRect: cull}
	c.state = canvasState{ctm: *NewMatrix4Identity(), clip: cull}
	recorder.canvas = c
	return c
}

// record adds an operation, unless the recording has ended.
func (c *Canvas) record(op DisplayOp) {
	if c.recorder != nil {
		c.ops = append(c.ops, op)
	}
}

// paintOrDefault is the paint to record: the default paint for nil.
func paintOrDefault(p *Paint) Paint {
	if p == nil {
		return Paint{}
	}
	return *p
}

// Save saves a copy of the current transform and clip on the save stack. Call
// Restore to pop the save stack.
//
// Flutter: ui/painting.dart Canvas.save
func (c *Canvas) Save() {
	c.stack = append(c.stack, c.state)
	c.record(OpSave{})
}

// SaveLayer saves a copy of the current transform and clip on the save stack,
// and then creates a new group which subsequent calls will become a part of.
// When the save stack is later popped, the group will be flattened into a layer
// and have the given paint's color filter, blend mode and alpha applied. The
// bounds, when not nil, limit the layer.
//
// Flutter: ui/painting.dart Canvas.saveLayer
func (c *Canvas) SaveLayer(bounds *Rect, paint *Paint) {
	op := OpSaveLayer{Paint: paintOrDefault(paint)}
	c.stack = append(c.stack, c.state)
	if bounds != nil {
		op.Bounds, op.HasBounds = *bounds, true
		c.intersectClip(c.state.ctm.TransformRect(*bounds))
	}
	c.record(op)
}

// Restore pops the transform and clip states stored by the last Save or
// SaveLayer. Nothing happens when the stack is empty.
//
// Flutter: ui/painting.dart Canvas.restore
func (c *Canvas) Restore() {
	n := len(c.stack)
	if n == 0 {
		return
	}
	c.state = c.stack[n-1]
	c.stack = c.stack[:n-1]
	c.record(OpRestore{})
}

// RestoreToCount efficiently calls Restore until GetSaveCount returns count. A
// count less than 1 is 1, and a count greater than the current one does
// nothing.
//
// Flutter: ui/painting.dart Canvas.restoreToCount
func (c *Canvas) RestoreToCount(count int) {
	if count < 1 {
		count = 1
	}
	for c.GetSaveCount() > count {
		c.Restore()
	}
}

// GetSaveCount returns the number of items on the save stack, including the
// initial state: it is 1 on a new canvas, and every Save and SaveLayer adds one.
//
// Flutter: ui/painting.dart Canvas.getSaveCount
func (c *Canvas) GetSaveCount() int { return len(c.stack) + 1 }

// concat multiplies the transform on the right by m and records it, unless m is
// not finite or does nothing.
func (c *Canvas) concat(m *Matrix4) {
	for _, v := range m.storage {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return
		}
	}
	if m.IsIdentity() {
		return
	}
	c.state.ctm.MultiplyBy(m)
	c.record(OpTransform{Matrix: *m})
}

// Translate adds a translation to the current transform, shifting the
// coordinate space horizontally by the first argument and vertically by the
// second argument.
//
// Flutter: ui/painting.dart Canvas.translate
func (c *Canvas) Translate(dx, dy float64) {
	c.concat(NewMatrix4TranslationValues(float32(dx), float32(dy), 0))
}

// Scale adds an axis-aligned scale to the current transform, scaling by the
// first argument in the horizontal direction and the second in the vertical
// direction.
//
// Flutter: ui/painting.dart Canvas.scale
func (c *Canvas) Scale(sx, sy float64) {
	c.concat(NewMatrix4Diagonal3Values(float32(sx), float32(sy), 1))
}

// ScaleUniform adds a scale of s in both directions to the current transform:
// Flutter's scale called with one argument.
//
// Flutter: ui/painting.dart Canvas.scale
func (c *Canvas) ScaleUniform(s float64) { c.Scale(s, s) }

// Rotate adds a rotation to the current transform. The argument is in radians
// clockwise.
//
// Flutter: ui/painting.dart Canvas.rotate
func (c *Canvas) Rotate(radians float64) {
	if math.Mod(radians*180/math.Pi, 360) == 0 {
		return
	}
	c.concat(NewMatrix4RotationZ(radians))
}

// Skew adds an axis-aligned skew to the current transform, with the first
// argument being the horizontal skew in rise over run units clockwise around the
// origin, and the second argument being the vertical skew in rise over run units
// clockwise around the origin.
//
// Flutter: ui/painting.dart Canvas.skew
func (c *Canvas) Skew(sx, sy float64) {
	m := NewMatrix4Identity()
	m.storage[4], m.storage[1] = float32(sx), float32(sy)
	c.concat(m)
}

// Transform multiplies the current transform by the specified 4⨉4 transformation
// matrix.
//
// Flutter: ui/painting.dart Canvas.transform
func (c *Canvas) Transform(matrix *Matrix4) { c.concat(matrix) }

// GetTransform returns a copy of the current transform.
//
// Flutter: ui/painting.dart Canvas.getTransform
func (c *Canvas) GetTransform() *Matrix4 { return c.state.ctm.Clone() }

// intersectClip narrows the device clip bounds.
func (c *Canvas) intersectClip(device Rect) {
	if device.IsEmpty() || c.state.clip.IsEmpty() {
		c.state.clip = Rect{}
		return
	}
	i := c.state.clip.Intersect(device)
	if i.IsEmpty() {
		i = Rect{}
	}
	c.state.clip = i
}

// differenceClip cuts a device rectangle out of the device clip bounds when the
// result is still a rectangle.
func (c *Canvas) differenceClip(cut Rect) {
	clip := c.state.clip
	if cut.IsEmpty() || clip.IsEmpty() || !clip.Overlaps(cut) {
		return
	}
	i := clip.Intersect(cut)
	switch {
	case i == clip:
		c.state.clip = Rect{}
	case i.Left == clip.Left && i.Right == clip.Right:
		if i.Top <= clip.Top {
			clip.Top = i.Bottom
		} else if i.Bottom >= clip.Bottom {
			clip.Bottom = i.Top
		}
		c.state.clip = clip
	case i.Top == clip.Top && i.Bottom == clip.Bottom:
		if i.Left <= clip.Left {
			clip.Left = i.Right
		} else if i.Right >= clip.Right {
			clip.Right = i.Left
		}
		c.state.clip = clip
	}
}

// preservesRects reports whether the transform keeps rectangles axis-aligned: it
// has no rotation (other than quarter turns), skew or perspective.
func (m *Matrix4) preservesRects() bool {
	s := &m.storage
	if m.hasPerspective() {
		return false
	}
	return (s[1] == 0 && s[4] == 0) || (s[0] == 0 && s[5] == 0)
}

// ClipRectOpts are the optional arguments of ClipRect.
//
// Flutter: ui/painting.dart Canvas.clipRect
type ClipRectOpts struct {
	// ClipOp says whether the rectangle is intersected with the clip (the zero
	// value, Flutter's default) or subtracted from it.
	ClipOp ClipOp
	// DoAntiAlias says whether the edge is smoothed; nil means true, Flutter's
	// default.
	DoAntiAlias *bool
}

// ClipOpts are the optional arguments of ClipRRect, ClipRSuperellipse and
// ClipPath.
//
// Flutter: ui/painting.dart Canvas.clipRRect
type ClipOpts struct {
	// DoAntiAlias says whether the edge is smoothed; nil means true, Flutter's
	// default.
	DoAntiAlias *bool
}

func antiAlias(p *bool) bool { return p == nil || *p }

// ClipRect reduces the clip region to the intersection of the current clip and
// the given rectangle (or, with ClipOpDifference, to the part of the current
// clip outside it). If the rectangle is empty, everything is clipped away until
// the next Restore.
//
// Flutter: ui/painting.dart Canvas.clipRect
func (c *Canvas) ClipRect(rect Rect, opts ...ClipRectOpts) {
	o := oneOpts(opts)
	op := o.ClipOp
	if op == ClipOpUnspecified {
		op = ClipOpIntersect
	}
	if rect.IsEmpty() {
		rect = Rect{min(rect.Left, rect.Right), min(rect.Top, rect.Bottom), max(rect.Left, rect.Right), max(rect.Top, rect.Bottom)}
	}
	c.record(OpClipRect{Rect: rect, Op: op, AntiAlias: antiAlias(o.DoAntiAlias)})
	if op == ClipOpIntersect {
		c.intersectClip(c.state.ctm.TransformRect(rect))
	} else if c.state.ctm.preservesRects() {
		c.differenceClip(c.state.ctm.TransformRect(rect))
	}
}

// ClipRRect reduces the clip region to the intersection of the current clip and
// the given rounded rectangle.
//
// Flutter: ui/painting.dart Canvas.clipRRect
func (c *Canvas) ClipRRect(rrect RRect, opts ...ClipOpts) {
	o := oneOpts(opts)
	c.record(OpClipRRect{RRect: rrect, AntiAlias: antiAlias(o.DoAntiAlias)})
	c.intersectClip(c.state.ctm.TransformRect(rrect.OuterRect()))
}

// ClipRSuperellipse reduces the clip region to the intersection of the current
// clip and the given rounded superellipse.
//
// Flutter: ui/painting.dart Canvas.clipRSuperellipse
func (c *Canvas) ClipRSuperellipse(rs RSuperellipse, opts ...ClipOpts) {
	o := oneOpts(opts)
	c.record(OpClipRSuperellipse{RSuperellipse: rs, AntiAlias: antiAlias(o.DoAntiAlias)})
	c.intersectClip(c.state.ctm.TransformRect(rs.OuterRect()))
}

// ClipPath reduces the clip region to the intersection of the current clip and
// the given path. The path is copied.
//
// Flutter: ui/painting.dart Canvas.clipPath
func (c *Canvas) ClipPath(path *Path, opts ...ClipOpts) {
	o := oneOpts(opts)
	p := NewPathFrom(path)
	c.record(OpClipPath{Path: p, AntiAlias: antiAlias(o.DoAntiAlias)})
	c.intersectClip(c.state.ctm.TransformRect(p.GetBounds()))
}

// GetLocalClipBounds returns the conservative bounds of the total clip in the
// current coordinate system: the rectangle that contains the area that can still
// be drawn into. It is meant for culling, not to be exact.
//
// Flutter: ui/painting.dart Canvas.getLocalClipBounds
func (c *Canvas) GetLocalClipBounds() Rect {
	if c.state.clip.IsEmpty() {
		return Rect{}
	}
	inv := c.state.ctm.Clone()
	if inv.Invert() == 0 {
		return Rect{}
	}
	return inv.TransformRect(c.state.clip)
}

// GetDestinationClipBounds returns the conservative bounds of the total clip in
// device coordinates: the coordinates of the target the picture will be drawn
// onto.
//
// Flutter: ui/painting.dart Canvas.getDestinationClipBounds
func (c *Canvas) GetDestinationClipBounds() Rect { return c.state.clip }

// sortedRect swaps the edges of an empty rectangle that has them backwards.
func sortedRect(r Rect) Rect {
	if r.IsEmpty() {
		return Rect{min(r.Left, r.Right), min(r.Top, r.Bottom), max(r.Left, r.Right), max(r.Top, r.Bottom)}
	}
	return r
}

// DrawColor paints the given color onto the canvas, applying the given blend
// mode, with the given color being the source and the background being the
// destination.
//
// Flutter: ui/painting.dart Canvas.drawColor
func (c *Canvas) DrawColor(color Color, blendMode BlendMode) {
	if blendMode == BlendModeUnspecified {
		blendMode = BlendModeSrcOver
	}
	c.record(OpDrawColor{Color: color, BlendMode: blendMode})
}

// DrawLine draws a line between the given points using the given paint. The line
// is stroked, the value of the paint's style is ignored for this call.
//
// Flutter: ui/painting.dart Canvas.drawLine
func (c *Canvas) DrawLine(p1, p2 Offset, paint *Paint) {
	c.record(OpDrawLine{P1: p1, P2: p2, Paint: paintOrDefault(paint)})
}

// DrawPaint fills the canvas with the given Paint. To fill the canvas with a
// solid color and blend mode, consider DrawColor instead.
//
// Flutter: ui/painting.dart Canvas.drawPaint
func (c *Canvas) DrawPaint(paint *Paint) { c.record(OpDrawPaint{Paint: paintOrDefault(paint)}) }

// DrawRect draws a rectangle with the given paint. Whether the rectangle is
// filled or stroked (or both) is controlled by the paint's style. A filled empty
// rectangle is not recorded.
//
// Flutter: ui/painting.dart Canvas.drawRect
func (c *Canvas) DrawRect(rect Rect, paint *Paint) {
	rect = sortedRect(rect)
	p := paintOrDefault(paint)
	if p.Style() != PaintingStyleFill || !rect.IsEmpty() {
		c.record(OpDrawRect{Rect: rect, Paint: p})
	}
}

// DrawRRect draws a rounded rectangle with the given paint.
//
// Flutter: ui/painting.dart Canvas.drawRRect
func (c *Canvas) DrawRRect(rrect RRect, paint *Paint) {
	c.record(OpDrawRRect{RRect: rrect, Paint: paintOrDefault(paint)})
}

// DrawDRRect draws a shape consisting of the difference between two rounded
// rectangles with the given paint. The outer rectangle must contain the inner
// one.
//
// Flutter: ui/painting.dart Canvas.drawDRRect
func (c *Canvas) DrawDRRect(outer, inner RRect, paint *Paint) {
	c.record(OpDrawDRRect{Outer: outer, Inner: inner, Paint: paintOrDefault(paint)})
}

// DrawRSuperellipse draws a rounded superellipse with the given paint.
//
// Flutter: ui/painting.dart Canvas.drawRSuperellipse
func (c *Canvas) DrawRSuperellipse(rs RSuperellipse, paint *Paint) {
	c.record(OpDrawRSuperellipse{RSuperellipse: rs, Paint: paintOrDefault(paint)})
}

// DrawOval draws an axis-aligned oval that fills the given axis-aligned
// rectangle with the given paint. A filled empty oval is not recorded.
//
// Flutter: ui/painting.dart Canvas.drawOval
func (c *Canvas) DrawOval(rect Rect, paint *Paint) {
	rect = sortedRect(rect)
	p := paintOrDefault(paint)
	if p.Style() != PaintingStyleFill || !rect.IsEmpty() {
		c.record(OpDrawOval{Rect: rect, Paint: p})
	}
}

// DrawCircle draws a circle centered at the point given by the first argument
// and that has the radius given by the second argument, with the paint given by
// the third argument.
//
// Flutter: ui/painting.dart Canvas.drawCircle
func (c *Canvas) DrawCircle(center Offset, radius float64, paint *Paint) {
	c.record(OpDrawCircle{Center: center, Radius: radius, Paint: paintOrDefault(paint)})
}

// DrawArc draws an arc scaled to fit inside the given rectangle. It starts from
// startAngle radians around the oval up to startAngle + sweepAngle radians around
// the oval, with zero radians being the point on the right hand side of the oval
// that crosses the horizontal line that intersects the center of the rectangle
// and with positive angles going clockwise around the oval. If useCenter is
// true, the arc is closed back to the center, forming a circle sector.
// Otherwise, the arc is not closed, forming a circle segment.
//
// Flutter: ui/painting.dart Canvas.drawArc
func (c *Canvas) DrawArc(rect Rect, startAngle, sweepAngle float64, useCenter bool, paint *Paint) {
	c.record(OpDrawArc{Rect: rect, StartAngle: startAngle, SweepAngle: sweepAngle, UseCenter: useCenter, Paint: paintOrDefault(paint)})
}

// DrawPath draws the given Path with the given paint. The path is copied.
//
// Flutter: ui/painting.dart Canvas.drawPath
func (c *Canvas) DrawPath(path *Path, paint *Paint) {
	c.record(OpDrawPath{Path: NewPathFrom(path), Paint: paintOrDefault(paint)})
}

// DrawImage draws the given Image into the canvas with its top-left corner at
// the given offset.
//
// Flutter: ui/painting.dart Canvas.drawImage
func (c *Canvas) DrawImage(image *Image, offset Offset, paint *Paint) {
	c.record(OpDrawImage{Image: image, Offset: offset, Paint: paintOrDefault(paint)})
}

// DrawImageRect draws the subset of the given image described by the src
// argument into the canvas in the axis-aligned rectangle given by the dst
// argument.
//
// Flutter: ui/painting.dart Canvas.drawImageRect
func (c *Canvas) DrawImageRect(image *Image, src, dst Rect, paint *Paint) {
	c.record(OpDrawImageRect{Image: image, Src: src, Dst: dst, Paint: paintOrDefault(paint)})
}

// DrawImageNine draws the given Image into the canvas using the given Paint.
//
// The image is drawn in nine portions described by splitting the image by
// drawing two horizontal lines and two vertical lines, where the center argument
// describes the rectangle formed by the four points where these four lines
// intersect each other. The corners are drawn into the corresponding corners of
// dst without scaling, and the edges and the center are stretched.
//
// Flutter: ui/painting.dart Canvas.drawImageNine
func (c *Canvas) DrawImageNine(image *Image, center, dst Rect, paint *Paint) {
	c.record(OpDrawImageNine{Image: image, Center: center, Dst: dst, Paint: paintOrDefault(paint)})
}

// DrawPicture draws the given Picture onto the canvas. To create a picture, see
// PictureRecorder.
//
// Flutter: ui/painting.dart Canvas.drawPicture
func (c *Canvas) DrawPicture(picture *Picture) { c.record(OpDrawPicture{Picture: picture}) }

// DrawParagraph draws a sequence of text.
//
// Flutter: ui/painting.dart Canvas.drawParagraph
func (c *Canvas) DrawParagraph(paragraph *Paragraph, offset Offset) {
	c.record(OpDrawParagraph{Paragraph: paragraph, Offset: offset})
}

// DrawPoints draws a sequence of points according to the given PointMode. The
// points are copied.
//
// Flutter: ui/painting.dart Canvas.drawPoints
func (c *Canvas) DrawPoints(mode PointMode, points []Offset, paint *Paint) {
	if mode == PointModeUnspecified {
		mode = PointModePoints
	}
	c.record(OpDrawPoints{Mode: mode, Points: append([]Offset(nil), points...), Paint: paintOrDefault(paint)})
}

// DrawVertices draws a set of Vertices onto the canvas as one or more triangles.
// The Paint.color property specifies the default color to use for the triangles
// drawn when the vertices have no colors, and its shader, when set, is used
// with the texture coordinates. The blend mode combines the vertex colors with
// the paint's color or shader.
//
// Flutter: ui/painting.dart Canvas.drawVertices
func (c *Canvas) DrawVertices(vertices *Vertices, blendMode BlendMode, paint *Paint) {
	if blendMode == BlendModeUnspecified {
		blendMode = BlendModeSrcOver
	}
	c.record(OpDrawVertices{Vertices: vertices, BlendMode: blendMode, Paint: paintOrDefault(paint)})
}

// DrawAtlas draws many parts of an image (the atlas) to the canvas. Each part
// is the rectangle in rects at the same index, placed by the transform at that
// index. The colors, when there are any, tint the parts with the blend mode
// (BlendModeSrc when unspecified). The optional cullRect is a bound on
// everything drawn, which can speed up culling. Lists whose lengths do not match
// panic with the tag AtlasTransformCountMismatch or AtlasColorCountMismatch.
//
// Flutter: ui/painting.dart Canvas.drawAtlas
func (c *Canvas) DrawAtlas(atlas *Image, transforms []RSTransform, rects []Rect, colors []Color, blendMode BlendMode, cullRect *Rect, paint *Paint) {
	if len(transforms) != len(rects) {
		panic(errors.New(tagAtlasTransformCountMismatch, `"transforms" and "rects" lengths must match.`,
			"transforms", len(transforms), "rects", len(rects)))
	}
	if len(colors) != 0 && len(colors) != len(rects) {
		panic(errors.New(tagAtlasColorCountMismatch, `If non-null, "colors" length must match that of "transforms" and "rects".`,
			"colors", len(colors), "rects", len(rects)))
	}
	if blendMode == BlendModeUnspecified {
		blendMode = BlendModeSrc
	}
	op := OpDrawAtlas{
		Atlas:      atlas,
		Transforms: append([]RSTransform(nil), transforms...),
		Rects:      append([]Rect(nil), rects...),
		BlendMode:  blendMode,
		Paint:      paintOrDefault(paint),
	}
	if len(colors) != 0 {
		op.Colors = append([]Color(nil), colors...)
	}
	if cullRect != nil {
		op.CullRect, op.HasCullRect = *cullRect, true
	}
	c.record(op)
}

// DrawShadow draws a shadow for a Path representing the given material
// elevation. The transparentOccluder argument should be true if the occluding
// object is not opaque.
//
// The elevation is in logical pixels; the renderer scales it by the device pixel
// ratio of the view.
//
// Flutter: ui/painting.dart Canvas.drawShadow
func (c *Canvas) DrawShadow(path *Path, color Color, elevation float64, transparentOccluder bool) {
	c.record(OpDrawShadow{Path: NewPathFrom(path), Color: color, Elevation: elevation, TransparentOccluder: transparentOccluder})
}

// String describes the canvas, as Dart's toString does.
//
// Flutter: ui/painting.dart Canvas.toString
func (c *Canvas) String() string {
	return "Canvas(recording: " + boolString(c.recorder != nil) + ")"
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAtlasColorCountMismatch     = "AtlasColorCountMismatch"
	tagAtlasTransformCountMismatch = "AtlasTransformCountMismatch"
	tagCanvasRecorderBusy          = "CanvasRecorderBusy"
)
