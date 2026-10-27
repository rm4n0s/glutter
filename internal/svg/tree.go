package svg

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// NodeKind is the kind of a Node.
type NodeKind uint8

// The kinds of nodes.
const (
	// NodeGroup draws its children in order: g, a, a use (its content is the child), and the child chosen by a
	// switch.
	NodeGroup NodeKind = iota
	// NodeSvg is a viewport: the root svg element, a nested svg, or a symbol that a use instantiated.
	NodeSvg
	// NodeShape is a path, rect, circle, ellipse, line, polyline or polygon, as a ui.Path in user units.
	NodeShape
	// NodeText is a text element.
	NodeText
	// NodeImage is an image with a data: URL.
	NodeImage
)

// Node is one drawable element of the tree. The parser has resolved its style, its geometry and its references, so
// that the painter draws it without reading any text again.
//
// A node is drawn like this (SVG 1.1: render.html#RenderingOrder): apply Transform; clip to the viewport if it is a
// NodeSvg; if the node has ClipPath, Mask, Filters or an Opacity below 1, draw the content into a layer that these
// are applied to; draw the content: a shape fills and strokes Path with Style, a group draws its Children.
type Node struct {
	Kind NodeKind
	// Name is the element's name in the source ("g", "rect", "use").
	Name string
	// ID is the id attribute, or "". A use gives its copies the id of the use (the copies of the referenced element
	// have none).
	ID string
	// Transform is the transform attribute (or the CSS transform property); for a use it includes the translation
	// by x and y.
	Transform Affine
	// Style is the computed style that inherits. It is shared between nodes that have the same one: do not change
	// it.
	Style *Style
	// Opacity is the opacity property (1 by default); it does not inherit.
	Opacity float64
	// ClipPath is the clip-path property, or nil. Missing references are ignored.
	ClipPath *ClipPath
	// Mask is the mask property, or nil.
	Mask *Mask
	// Filters are the url() references of the filter property, in order. References to a filter that does not
	// exist are dropped; an element with a filter function (blur(), ...) is drawn as if it had no filter.
	Filters []*Filter
	// VectorEffect is the vector-effect property; it does not inherit.
	VectorEffect VectorEffect
	// Children are the nodes drawn inside a NodeGroup or a NodeSvg, in order.
	Children []*Node
	// Path is the geometry of a NodeShape. Its fill type is the fill-rule of the style (the clip-rule inside a
	// clip path).
	Path *ui.Path
	// Shape says which element the NodeShape came from.
	Shape ShapeKind
	// Vertices are the vertices of a path, line, polyline or polygon, where its markers go; set only when the style has
	// a marker property.
	Vertices []Vertex
	// Viewport is set for a NodeSvg.
	Viewport *Viewport
	// Text is set for a NodeText.
	Text *TextSpan
	// Image is set for a NodeImage.
	Image *Image
}

// ShapeKind is the element a NodeShape came from.
type ShapeKind uint8

// The shape elements.
const (
	ShapePath ShapeKind = iota
	ShapeRect
	ShapeCircle
	ShapeEllipse
	ShapeLine
	ShapePolyline
	ShapePolygon
)

// VectorEffect is the vector-effect property.
//
// SVG 2: coords.html#VectorEffectProperty
type VectorEffect uint8

const (
	// VectorEffectNone is none.
	VectorEffectNone VectorEffect = iota
	// VectorEffectNonScalingStroke is non-scaling-stroke: the stroke width is in the device's pixels.
	VectorEffectNonScalingStroke
)

// Viewport is the viewport that an svg element, or a symbol, establishes.
//
// SVG 1.1: coords.html#EstablishingANewSVGViewport
type Viewport struct {
	// X and Y are the position of the viewport in the user space of its parent (0 for the root and for a symbol).
	X, Y float64
	// Width and Height are the size of the viewport in the parent's user units. For the root they are the document's
	// size (Document.Size); for a nested svg or a symbol they are width and height resolved against the parent
	// viewport (100% by default).
	Width, Height float64
	// ViewBox is the viewBox, or nil.
	ViewBox *ViewBox
	// PreserveAspectRatio says how ViewBox fits in the viewport.
	PreserveAspectRatio PreserveAspectRatio
	// Overflow is true if the content is not clipped to the viewport (overflow: visible or auto). It is false by
	// default for a nested svg and a symbol; the root is not clipped by the document, the widget clips it.
	Overflow bool
	// Root is true for the root svg element.
	Root bool
}

// Image is an image element whose href is a data: URL.
//
// SVG 1.1: struct.html#ImageElement
type Image struct {
	// X, Y, Width and Height are the rectangle the image is fitted in. Width or Height is 0 when it is auto (not
	// given): the painter then takes the size of the image.
	X, Y, Width, Height float64
	// PreserveAspectRatio says how the image fits in the rectangle.
	PreserveAspectRatio PreserveAspectRatio
	// MIME is the media type of the data URL, in lower case ("image/png", "image/svg+xml").
	MIME string
	// Data is the decoded content of the URL.
	Data []byte
}

// FillRule is fill-rule and clip-rule.
//
// SVG 1.1: painting.html#FillRuleProperty
type FillRule uint8

const (
	// FillRuleNonZero is nonzero, the default.
	FillRuleNonZero FillRule = iota
	// FillRuleEvenOdd is evenodd.
	FillRuleEvenOdd
)

// LineCap is stroke-linecap.
//
// SVG 1.1: painting.html#StrokeLinecapProperty
type LineCap uint8

const (
	// LineCapButt is butt, the default.
	LineCapButt LineCap = iota
	// LineCapRound is round.
	LineCapRound
	// LineCapSquare is square.
	LineCapSquare
)

// LineJoin is stroke-linejoin.
//
// SVG 1.1: painting.html#StrokeLinejoinProperty
type LineJoin uint8

const (
	// LineJoinMiter is miter, the default (miter-clip and arcs are drawn as miter).
	LineJoinMiter LineJoin = iota
	// LineJoinRound is round.
	LineJoinRound
	// LineJoinBevel is bevel.
	LineJoinBevel
)

// TextAnchor is text-anchor.
//
// SVG 1.1: text.html#TextAnchorProperty
type TextAnchor uint8

const (
	// TextAnchorStart is start, the default.
	TextAnchorStart TextAnchor = iota
	// TextAnchorMiddle is middle.
	TextAnchorMiddle
	// TextAnchorEnd is end.
	TextAnchorEnd
)

// FontStyle is font-style.
type FontStyle uint8

const (
	// FontStyleNormal is normal, the default.
	FontStyleNormal FontStyle = iota
	// FontStyleItalic is italic.
	FontStyleItalic
	// FontStyleOblique is oblique.
	FontStyleOblique
)

// TextDecoration is the set of lines of text-decoration.
type TextDecoration uint8

// The lines of a text decoration; they combine.
const (
	TextDecorationUnderline TextDecoration = 1 << iota
	TextDecorationOverline
	TextDecorationLineThrough
)

// ColorInterpolation is color-interpolation-filters.
//
// SVG 1.1: painting.html#ColorInterpolationFiltersProperty
type ColorInterpolation uint8

const (
	// ColorInterpolationLinearRGB is linearRGB, the default of filters.
	ColorInterpolationLinearRGB ColorInterpolation = iota
	// ColorInterpolationSRGB is sRGB (and auto).
	ColorInterpolationSRGB
)

// ShapeRendering is shape-rendering.
//
// SVG 1.1: painting.html#ShapeRenderingProperty
type ShapeRendering uint8

const (
	// ShapeRenderingAuto is auto (and optimizeSpeed and geometricPrecision), the default.
	ShapeRenderingAuto ShapeRendering = iota
	// ShapeRenderingCrispEdges is crispEdges: no anti-aliasing.
	ShapeRenderingCrispEdges
)

// PaintLayer is one of the three things that paint-order orders.
type PaintLayer uint8

const (
	// PaintLayerFill is fill.
	PaintLayerFill PaintLayer = iota
	// PaintLayerStroke is stroke.
	PaintLayerStroke
	// PaintLayerMarkers is markers.
	PaintLayerMarkers
)

// DefaultPaintOrder is fill, stroke, markers.
var DefaultPaintOrder = [3]PaintLayer{PaintLayerFill, PaintLayerStroke, PaintLayerMarkers}

// PaintKind is the kind of a Paint.
type PaintKind uint8

const (
	// PaintKindNone draws nothing: the value none, a missing server without a fallback, or an invalid value.
	PaintKindNone PaintKind = iota
	// PaintKindColor is a color (currentColor is resolved to a color).
	PaintKindColor
	// PaintKindServer is a gradient or a pattern.
	PaintKindServer
)

// Paint is the value of fill and stroke.
//
// SVG 1.1: painting.html#SpecifyingPaint
type Paint struct {
	Kind  PaintKind
	Color ui.Color
	// Server is a *Gradient or a *Pattern when Kind is PaintKindServer.
	Server PaintServer
}

// PaintServer is a Gradient or a Pattern.
type PaintServer interface{ paintServer() }

// Style is the computed style of the properties that inherit. Nodes that inherit everything share one Style, so it
// is never changed after the parser has made it.
//
// SVG 1.1: styling.html; CSS 2.1: cascade.html#inheritance
type Style struct {
	// Color is the color property, the value of currentColor (Theme.CurrentColor at the root).
	Color ui.Color
	// Fill and Stroke are the paints; the initial Fill is black and the initial Stroke is none.
	Fill, Stroke Paint
	// FillOpacity and StrokeOpacity are in [0, 1].
	FillOpacity, StrokeOpacity float64
	// FillRule and ClipRule are the winding rules.
	FillRule, ClipRule FillRule
	// StrokeWidth is in user units (1 by default).
	StrokeWidth float64
	// LineCap and LineJoin are the stroke's.
	LineCap  LineCap
	LineJoin LineJoin
	// MiterLimit is stroke-miterlimit (4 by default).
	MiterLimit float64
	// DashArray is stroke-dasharray in user units: nil for none, otherwise an even number of values that are not all
	// zero (an odd list is repeated twice, as the specification says).
	DashArray []float64
	// DashOffset is stroke-dashoffset in user units.
	DashOffset float64
	// Visible is false for visibility: hidden and collapse.
	Visible bool
	// ShapeRendering is shape-rendering.
	ShapeRendering ShapeRendering
	// PaintOrder is paint-order (DefaultPaintOrder by default).
	PaintOrder [3]PaintLayer
	// MarkerStart, MarkerMid and MarkerEnd are the marker properties, or nil.
	MarkerStart, MarkerMid, MarkerEnd *Marker
	// ColorInterpolationFilters is color-interpolation-filters.
	ColorInterpolationFilters ColorInterpolation

	// FontFamily is the font-family list, quotes removed; generic names such as sans-serif are kept as written.
	FontFamily []string
	// FontSize is the computed font size in user units.
	FontSize float64
	// FontWeight is the weight from 1 to 1000 (400 normal, 700 bold).
	FontWeight int
	// FontStyle is font-style.
	FontStyle FontStyle
	// TextAnchor is text-anchor.
	TextAnchor TextAnchor
	// TextDecoration is the lines of text-decoration. Deviation: the property does not inherit in CSS, but the lines
	// are drawn on the descendants of the text, so it is kept here.
	TextDecoration TextDecoration
	// LetterSpacing and WordSpacing are in user units (0 for normal).
	LetterSpacing, WordSpacing float64
}

// Gradient is a linearGradient or a radialGradient with the attributes it inherits through href already merged in:
// every field has its final value.
//
// SVG 1.1: pservers.html#Gradients
type Gradient struct {
	// Radial is true for a radialGradient.
	Radial bool
	// X1, Y1, X2 and Y2 are the vector of a linear gradient (0%, 0%, 100%, 0% by default).
	X1, Y1, X2, Y2 Length
	// CX, CY, R, FX, FY and FR are the circles of a radial gradient (50%, 50%, 50%, then CX, CY and 0 by default).
	CX, CY, R, FX, FY, FR Length
	// Units says what the coordinates are relative to; a percentage under ObjectBoundingBox is a fraction of the
	// bounding box (Length.Resolve with ref 1), under UserSpaceOnUse of the viewport.
	Units Units
	// Transform is gradientTransform.
	Transform Affine
	// Spread is spreadMethod.
	Spread Spread
	// Stops are the stops with offsets in [0, 1] that never decrease, and the stop-opacity applied to the color.
	Stops []Stop
}

func (*Gradient) paintServer() {}

// Stop is a gradient stop.
//
// SVG 1.1: pservers.html#GradientStops
type Stop struct {
	Offset float64
	Color  ui.Color
}

// Units is the value of gradientUnits, patternUnits, clipPathUnits, maskUnits and the like.
//
// SVG 1.1: coords.html#ObjectBoundingBox
type Units uint8

const (
	// UnitsObjectBoundingBox is objectBoundingBox.
	UnitsObjectBoundingBox Units = iota
	// UnitsUserSpaceOnUse is userSpaceOnUse.
	UnitsUserSpaceOnUse
)

// Spread is spreadMethod.
//
// SVG 1.1: pservers.html#GradientSpreadMethod
type Spread uint8

const (
	// SpreadPad is pad, the default.
	SpreadPad Spread = iota
	// SpreadReflect is reflect.
	SpreadReflect
	// SpreadRepeat is repeat.
	SpreadRepeat
)

// Pattern is a pattern with the attributes and the children it inherits through href already merged in.
//
// SVG 1.1: pservers.html#Patterns
type Pattern struct {
	// X, Y, Width and Height are the tile (0 by default).
	X, Y, Width, Height Length
	// Units is patternUnits (objectBoundingBox by default) and ContentUnits is patternContentUnits (userSpaceOnUse by
	// default).
	Units, ContentUnits Units
	// Transform is patternTransform.
	Transform Affine
	// ViewBox and PreserveAspectRatio place the content in the tile.
	ViewBox             *ViewBox
	PreserveAspectRatio PreserveAspectRatio
	// Children are the content of the tile.
	Children []*Node
}

func (*Pattern) paintServer() {}

// ClipPath is a clipPath element.
//
// SVG 1.1: masking.html#EstablishingANewClippingPath
type ClipPath struct {
	// Units is clipPathUnits (userSpaceOnUse by default).
	Units Units
	// Transform is the transform of the clipPath element.
	Transform Affine
	// ClipPath is the clip-path property of the clipPath element itself, or nil: the intersection of both clips.
	ClipPath *ClipPath
	// Children are the shapes, texts and uses that make the clip, as nodes. Their Path has the clip-rule as its fill
	// type, and the nodes with display: none or visibility: hidden are not in the list.
	Children []*Node
}

// MaskType is mask-type.
//
// SVG 2: masking.html#MaskTypeProperty
type MaskType uint8

const (
	// MaskLuminance is luminance, the default.
	MaskLuminance MaskType = iota
	// MaskAlpha is alpha.
	MaskAlpha
)

// Mask is a mask element.
//
// SVG 1.1: masking.html#Masking
type Mask struct {
	// X, Y, Width and Height are the region of the mask (-10%, -10%, 120%, 120% by default).
	X, Y, Width, Height Length
	// Units is maskUnits (objectBoundingBox by default) and ContentUnits is maskContentUnits (userSpaceOnUse by
	// default).
	Units, ContentUnits Units
	// Type is the mask-type property of the mask element.
	Type MaskType
	// Mask is the mask property of the mask element itself, or nil.
	Mask *Mask
	// Children are the content of the mask.
	Children []*Node
}

// Filter is a filter element.
//
// SVG 1.1: filters.html#FilterElement
type Filter struct {
	// X, Y, Width and Height are the filter region (-10%, -10%, 120%, 120% by default).
	X, Y, Width, Height Length
	// Units is filterUnits (objectBoundingBox by default) and PrimitiveUnits is primitiveUnits (userSpaceOnUse by
	// default).
	Units, PrimitiveUnits Units
	// Primitives are the filter primitives in order, with the attributes and children of a filter that this one
	// refers to with href merged in as the specification says.
	Primitives []*FilterPrimitive
}

// FilterPrimitive is one filter primitive element (feGaussianBlur, feOffset, ...).
//
// SVG 1.1: filters.html#FilterPrimitivesOverview
type FilterPrimitive struct {
	// Name is the element's name ("feGaussianBlur").
	Name string
	// In and In2 are the in and in2 attributes ("" if not given: the previous result, or SourceGraphic for the
	// first primitive), and Result is the result attribute.
	In, In2, Result string
	// X, Y, Width and Height are the subregion; a nil pointer is not given (the default is the filter region, or the
	// union of the inputs' subregions).
	X, Y, Width, Height *Length
	// Attrs are the other attributes of the element as written ("stdDeviation", "dx", "values", "operator"), for the
	// painter to read with ParseNumberList, ParseLength and the like.
	Attrs map[string]string
	// Style has the color-interpolation-filters of the primitive (inherited from its filter), and Color,
	// Opacity are the flood-color and flood-opacity, or the lighting-color, resolved (black and 1 by default).
	Style        *Style
	FloodColor   ui.Color
	FloodOpacity float64
	// Children are the feMergeNode and feFunc elements, the light sources of lighting filters.
	Children []*FilterPrimitive
}

// Marker is a marker element.
//
// SVG 1.1: painting.html#MarkerElement
type Marker struct {
	// RefX and RefY are the point of the marker's content that is placed on the vertex, in the marker's content
	// units (viewBox units if there is a viewBox).
	RefX, RefY float64
	// Width and Height are markerWidth and markerHeight (3 by default).
	Width, Height float64
	// StrokeWidthUnits is true for markerUnits="strokeWidth" (the default): the marker is scaled by the stroke
	// width.
	StrokeWidthUnits bool
	// Orient is the orient attribute: OrientAuto, OrientAutoStartReverse or an angle in degrees in Angle.
	Orient OrientKind
	Angle  float64
	// ViewBox and PreserveAspectRatio place the content in Width and Height.
	ViewBox             *ViewBox
	PreserveAspectRatio PreserveAspectRatio
	// Overflow is true if the content is not clipped to the marker viewport.
	Overflow bool
	// Children are the content.
	Children []*Node
}

// OrientKind is the kind of value that the orient attribute of a marker has.
type OrientKind uint8

const (
	// OrientAngle is a fixed angle (Marker.Angle).
	OrientAngle OrientKind = iota
	// OrientAuto is auto.
	OrientAuto
	// OrientAutoStartReverse is auto-start-reverse.
	OrientAutoStartReverse
)

// TextSpan is a text, tspan, textPath or tref: a run of text with its position lists and its own style.
//
// SVG 1.1: text.html#TextElement
type TextSpan struct {
	// Name is "text", "tspan", "textPath" or "tref".
	Name string
	// X, Y, Dx, Dy and Rotate are the lists of the attributes, in user units (degrees for Rotate); the nth value
	// applies to the nth character of the element and of its descendants, counting from the start of the text
	// element. A nil list is an absent attribute.
	X, Y, Dx, Dy, Rotate []float64
	// TextLength is the textLength attribute, or 0 if not given, and AdjustGlyphs is lengthAdjust="spacingAndGlyphs".
	TextLength   float64
	AdjustGlyphs bool
	// Style is the computed style of the span.
	Style *Style
	// Items are the content in order: strings and spans.
	Items []TextItem
	// PathData is the geometry that a textPath follows, in the user space of the text element (the transform of the
	// path element applied), or nil. StartOffset is startOffset in user units, or a fraction of the path's length
	// if StartOffsetFraction is true.
	PathData            *ui.Path
	StartOffset         float64
	StartOffsetFraction bool
}

// TextItem is a string or a span inside a span. The strings have the white space handled as xml:space says
// (collapsed by default).
type TextItem struct {
	Text string
	Span *TextSpan
}

// Theme is the part of the theme of the widget (flutter_svg's SvgTheme) that the parser uses.
//
// SVG 1.1: styling.html#ColorProperty
type Theme struct {
	// CurrentColor is the value of currentColor at the root (black by default: the zero Color means the default).
	CurrentColor ui.Color
	// FontSize is the font size of the root in user units and the value of 1em there (14 by default).
	FontSize float64
	// XHeight is the value of 1ex at the root (half of the font size by default).
	XHeight float64
}

func (t Theme) withDefaults() Theme {
	if t.CurrentColor == (ui.Color{}) {
		t.CurrentColor = ui.NewColorFromARGB(255, 0, 0, 0)
	}
	if t.FontSize <= 0 || math.IsNaN(t.FontSize) {
		t.FontSize = 14
	}
	if t.XHeight <= 0 || math.IsNaN(t.XHeight) {
		t.XHeight = t.FontSize / 2
	}
	return t
}
