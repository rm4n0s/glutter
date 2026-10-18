package ui

// DisplayOp is one recorded drawing command of a Canvas. The renderers walk the
// operations of a Picture's DisplayList and draw them; apps do not need it. The
// operations are the exported structs below, so a renderer picks one with a type
// switch. An operation is a value: it is never changed after it is recorded, and
// it holds copies of the paths and paints it was given.
//
// Flutter: no counterpart (Flutter's engine records a C++ DisplayList)
type DisplayOp interface {
	displayOp()
}

// DisplayList is the list of drawing commands a PictureRecorder recorded. It is
// immutable.
//
// Flutter: no counterpart (flutter::DisplayList)
type DisplayList struct {
	ops      []DisplayOp
	cullRect Rect
}

// Ops returns the operations in the order they were recorded. The slice is the
// list's own: callers must not change it.
//
// Flutter: no counterpart
func (d *DisplayList) Ops() []DisplayOp { return d.ops }

// CullRect is the rectangle the canvas was created with, outside of which the
// picture has no content that matters.
//
// Flutter: no counterpart
func (d *DisplayList) CullRect() Rect { return d.cullRect }

// OpSave pushes the transform and the clip, as Canvas.Save does.
//
// Flutter: ui/painting.dart Canvas.save
type OpSave struct{}

// OpSaveLayer pushes the transform and the clip and starts drawing into a layer
// that is composited with the paint at the matching restore.
//
// Flutter: ui/painting.dart Canvas.saveLayer
type OpSaveLayer struct {
	// Bounds limits the layer; HasBounds is false when there is no limit.
	Bounds    Rect
	HasBounds bool
	// Paint says how the layer is composited: its alpha, blend mode, color
	// filter and image filter (which also filters the content behind the layer
	// when it is a backdrop).
	Paint Paint
}

// OpRestore pops the transform, the clip and, if there is one, the layer.
//
// Flutter: ui/painting.dart Canvas.restore
type OpRestore struct{}

// OpTransform multiplies the current transform on the right by a matrix, as
// Canvas.Translate, Scale, Rotate, Skew and Transform do.
//
// Flutter: ui/painting.dart Canvas.transform
type OpTransform struct {
	// Matrix is the matrix to concatenate.
	Matrix Matrix4
}

// OpClipRect clips to a rectangle in the current coordinates.
//
// Flutter: ui/painting.dart Canvas.clipRect
type OpClipRect struct {
	// Rect is the clip shape.
	Rect Rect
	// Op says whether the shape is intersected with the clip or subtracted.
	Op ClipOp
	// AntiAlias says whether the edge is smoothed.
	AntiAlias bool
}

// OpClipRRect clips to a rounded rectangle in the current coordinates.
//
// Flutter: ui/painting.dart Canvas.clipRRect
type OpClipRRect struct {
	// RRect is the clip shape.
	RRect RRect
	// AntiAlias says whether the edge is smoothed.
	AntiAlias bool
}

// OpClipRSuperellipse clips to a rounded superellipse in the current
// coordinates.
//
// Flutter: ui/painting.dart Canvas.clipRSuperellipse
type OpClipRSuperellipse struct {
	// RSuperellipse is the clip shape.
	RSuperellipse RSuperellipse
	// AntiAlias says whether the edge is smoothed.
	AntiAlias bool
}

// OpClipPath clips to a path in the current coordinates.
//
// Flutter: ui/painting.dart Canvas.clipPath
type OpClipPath struct {
	// Path is the clip shape: a copy of the path that was given.
	Path *Path
	// AntiAlias says whether the edge is smoothed.
	AntiAlias bool
}

// OpDrawColor paints the whole clip with a color and a blend mode.
//
// Flutter: ui/painting.dart Canvas.drawColor
type OpDrawColor struct {
	// Color is the source color.
	Color Color
	// BlendMode is how it combines with what is there.
	BlendMode BlendMode
}

// OpDrawPaint fills the whole clip with a paint.
//
// Flutter: ui/painting.dart Canvas.drawPaint
type OpDrawPaint struct {
	// Paint is the paint to fill with.
	Paint Paint
}

// OpDrawLine draws a line between two points.
//
// Flutter: ui/painting.dart Canvas.drawLine
type OpDrawLine struct {
	// P1 is the start of the line.
	P1 Offset
	// P2 is the end of the line.
	P2 Offset
	// Paint is the paint to stroke it with.
	Paint Paint
}

// OpDrawRect draws a rectangle.
//
// Flutter: ui/painting.dart Canvas.drawRect
type OpDrawRect struct {
	// Rect is the rectangle, with its edges in order.
	Rect Rect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawRRect draws a rounded rectangle.
//
// Flutter: ui/painting.dart Canvas.drawRRect
type OpDrawRRect struct {
	// RRect is the rounded rectangle.
	RRect RRect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawDRRect draws the region between two rounded rectangles.
//
// Flutter: ui/painting.dart Canvas.drawDRRect
type OpDrawDRRect struct {
	// Outer is the outer edge of the region.
	Outer RRect
	// Inner is the inner edge of the region.
	Inner RRect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawRSuperellipse draws a rounded superellipse.
//
// Flutter: ui/painting.dart Canvas.drawRSuperellipse
type OpDrawRSuperellipse struct {
	// RSuperellipse is the shape.
	RSuperellipse RSuperellipse
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawOval draws an axis-aligned oval that fills a rectangle.
//
// Flutter: ui/painting.dart Canvas.drawOval
type OpDrawOval struct {
	// Rect is the rectangle the oval fills, with its edges in order.
	Rect Rect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawCircle draws a circle.
//
// Flutter: ui/painting.dart Canvas.drawCircle
type OpDrawCircle struct {
	// Center is the center of the circle.
	Center Offset
	// Radius is its radius.
	Radius float64
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawArc draws an arc of an oval.
//
// Flutter: ui/painting.dart Canvas.drawArc
type OpDrawArc struct {
	// Rect is the rectangle the oval fills.
	Rect Rect
	// StartAngle is where the arc starts, in radians clockwise from the right.
	StartAngle float64
	// SweepAngle is how far it goes, in radians; positive is clockwise.
	SweepAngle float64
	// UseCenter says whether the arc is closed through the center of the oval
	// (a pie slice) or by a chord.
	UseCenter bool
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawPath draws a path.
//
// Flutter: ui/painting.dart Canvas.drawPath
type OpDrawPath struct {
	// Path is a copy of the path that was given.
	Path *Path
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawImage draws an image with its top left corner at an offset.
//
// Flutter: ui/painting.dart Canvas.drawImage
type OpDrawImage struct {
	// Image is the image to draw.
	Image *Image
	// Offset is where its top left corner goes.
	Offset Offset
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawImageRect draws part of an image into a rectangle.
//
// Flutter: ui/painting.dart Canvas.drawImageRect
type OpDrawImageRect struct {
	// Image is the image to draw.
	Image *Image
	// Src is the part of the image to draw, in the pixels of the image.
	Src Rect
	// Dst is where it goes, scaled to fit.
	Dst Rect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawImageNine draws an image stretched like a nine-patch.
//
// Flutter: ui/painting.dart Canvas.drawImageNine
type OpDrawImageNine struct {
	// Image is the image to draw.
	Image *Image
	// Center is the middle cell of the nine-patch, in the pixels of the image.
	Center Rect
	// Dst is where the whole image goes.
	Dst Rect
	// Paint is the paint to draw it with.
	Paint Paint
}

// OpDrawPicture draws another picture.
//
// Flutter: ui/painting.dart Canvas.drawPicture
type OpDrawPicture struct {
	// Picture is the picture to draw.
	Picture *Picture
}

// OpDrawParagraph draws a paragraph of text.
//
// Flutter: ui/painting.dart Canvas.drawParagraph
type OpDrawParagraph struct {
	// Paragraph is the laid-out text.
	Paragraph *Paragraph
	// Offset is where its top left corner goes.
	Offset Offset
}

// OpDrawPoints draws a set of points, lines or a polygon.
//
// Flutter: ui/painting.dart Canvas.drawPoints
type OpDrawPoints struct {
	// Mode says how the points are joined.
	Mode PointMode
	// Points are a copy of the points that were given.
	Points []Offset
	// Paint is the paint to draw them with.
	Paint Paint
}

// OpDrawVertices draws triangles from vertices.
//
// Flutter: ui/painting.dart Canvas.drawVertices
type OpDrawVertices struct {
	// Vertices are the vertices to draw.
	Vertices *Vertices
	// BlendMode is how the vertex colors combine with the paint's color or shader.
	BlendMode BlendMode
	// Paint is the paint to draw them with.
	Paint Paint
}

// OpDrawAtlas draws many parts of an image, each transformed.
//
// Flutter: ui/painting.dart Canvas.drawAtlas
type OpDrawAtlas struct {
	// Atlas is the image that holds the parts.
	Atlas *Image
	// Transforms place each part.
	Transforms []RSTransform
	// Rects are the parts of the atlas, one for each transform.
	Rects []Rect
	// Colors, when there are any, tint each part; as many as the rects.
	Colors []Color
	// BlendMode is how the colors combine with the atlas.
	BlendMode BlendMode
	// CullRect is a bound on everything drawn, for culling; HasCullRect is false
	// when it was not given.
	CullRect    Rect
	HasCullRect bool
	// Paint is the paint to draw them with.
	Paint Paint
}

// OpDrawShadow draws the shadow of a path.
//
// Flutter: ui/painting.dart Canvas.drawShadow
type OpDrawShadow struct {
	// Path is a copy of the path that casts the shadow.
	Path *Path
	// Color is the color of the shadow.
	Color Color
	// Elevation is how far the path is above the surface, in logical pixels.
	Elevation float64
	// TransparentOccluder says that the path is not opaque, so the shadow also
	// shows through it.
	TransparentOccluder bool
}

func (OpSave) displayOp()              {}
func (OpSaveLayer) displayOp()         {}
func (OpRestore) displayOp()           {}
func (OpTransform) displayOp()         {}
func (OpClipRect) displayOp()          {}
func (OpClipRRect) displayOp()         {}
func (OpClipRSuperellipse) displayOp() {}
func (OpClipPath) displayOp()          {}
func (OpDrawColor) displayOp()         {}
func (OpDrawPaint) displayOp()         {}
func (OpDrawLine) displayOp()          {}
func (OpDrawRect) displayOp()          {}
func (OpDrawRRect) displayOp()         {}
func (OpDrawDRRect) displayOp()        {}
func (OpDrawRSuperellipse) displayOp() {}
func (OpDrawOval) displayOp()          {}
func (OpDrawCircle) displayOp()        {}
func (OpDrawArc) displayOp()           {}
func (OpDrawPath) displayOp()          {}
func (OpDrawImage) displayOp()         {}
func (OpDrawImageRect) displayOp()     {}
func (OpDrawImageNine) displayOp()     {}
func (OpDrawPicture) displayOp()       {}
func (OpDrawParagraph) displayOp()     {}
func (OpDrawPoints) displayOp()        {}
func (OpDrawVertices) displayOp()      {}
func (OpDrawAtlas) displayOp()         {}
func (OpDrawShadow) displayOp()        {}

// approximateBytes estimates the memory the operation holds, in bytes.
func approximateBytes(op DisplayOp) int {
	const base = 48
	const paintSize = 96
	switch o := op.(type) {
	case OpClipPath:
		return base + pathBytes(o.Path)
	case OpDrawPath:
		return base + paintSize + pathBytes(o.Path)
	case OpDrawShadow:
		return base + pathBytes(o.Path)
	case OpDrawPoints:
		return base + paintSize + len(o.Points)*16
	case OpDrawVertices:
		v := o.Vertices
		return base + paintSize + len(v.positions)*16 + len(v.colors)*20 + len(v.texCoords)*16 + len(v.indices)*2
	case OpDrawAtlas:
		return base + paintSize + len(o.Transforms)*16 + len(o.Rects)*32 + len(o.Colors)*20
	case OpDrawPicture:
		return base + o.Picture.ApproximateBytesUsed()
	case OpSave, OpRestore:
		return 8
	case OpTransform:
		return base + 64
	}
	return base + paintSize
}

// pathBytes estimates the memory a path holds.
func pathBytes(p *Path) int {
	if p == nil {
		return 0
	}
	return len(p.verbs) + len(p.points)*16 + len(p.weights)*8
}
