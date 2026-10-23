package painting

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// BorderStyle is the style of line to draw for a BorderSide in a Border. The
// zero value is BorderStyleUnspecified: the default (solid) when a BorderSide is
// created with NewBorderSide.
//
// Flutter: painting/borders.dart BorderStyle
type BorderStyle int

// The values of BorderStyle follow Flutter's declaration order.
const (
	// BorderStyleUnspecified is the zero value: solid in NewBorderSide.
	//
	// Flutter: no counterpart
	BorderStyleUnspecified BorderStyle = iota
	// BorderStyleNone skips the border.
	//
	// Flutter: painting/borders.dart BorderStyle.none
	BorderStyleNone
	// BorderStyleSolid draws the border as a solid line.
	//
	// Flutter: painting/borders.dart BorderStyle.solid
	BorderStyleSolid
)

// Name returns the name of the value, such as "solid".
//
// Flutter: painting/borders.dart BorderStyle.name
func (s BorderStyle) Name() string {
	switch s {
	case BorderStyleNone:
		return "none"
	case BorderStyleSolid:
		return "solid"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "BorderStyle.solid".
//
// Flutter: painting/borders.dart BorderStyle.toString
func (s BorderStyle) String() string { return "BorderStyle." + s.Name() }

// The values of BorderSide.StrokeAlign.
const (
	// StrokeAlignInside places the stroke inside of the border path.
	//
	// Flutter: painting/borders.dart BorderSide.strokeAlignInside
	StrokeAlignInside = -1.0
	// StrokeAlignCenter centers the stroke on the border path.
	//
	// Flutter: painting/borders.dart BorderSide.strokeAlignCenter
	StrokeAlignCenter = 0.0
	// StrokeAlignOutside places the stroke outside of the border path.
	//
	// Flutter: painting/borders.dart BorderSide.strokeAlignOutside
	StrokeAlignOutside = 1.0
)

var borderSideDefaultColor = ui.NewColor(0xFF000000)

// BorderSide is a side of a border of a box. Make one with NewBorderSide. The
// zero BorderSide, which a literal such as Border{Top: side} leaves in the other
// sides, stands for BorderSideNone.
//
// Flutter: painting/borders.dart BorderSide
type BorderSide struct {
	// Color is the color of this side of the border.
	Color ui.Color
	// Width is the width of this side of the border, in logical pixels.
	Width float64
	// Style is the style of this side of the border.
	Style BorderStyle
	// StrokeAlign is the relative position of the stroke on a BorderSide in an
	// OutlinedBorder or Border, from StrokeAlignInside (-1) to
	// StrokeAlignOutside (1).
	StrokeAlign float64
}

// BorderSideOpts are the optional parameters of NewBorderSide.
//
// Flutter: painting/borders.dart BorderSide.new
type BorderSideOpts struct {
	// Color is the color of the side; the default is black.
	Color foundation.Opt[ui.Color]
	// Width is the width of the side; the default is 1.0.
	Width foundation.Opt[float64]
	// Style is the style of the side; the default is BorderStyleSolid.
	Style BorderStyle
	// StrokeAlign is the relative position of the stroke; the default is
	// StrokeAlignInside.
	StrokeAlign foundation.Opt[float64]
}

// BorderSideNone is a hairline black border that is not rendered.
//
// Flutter: painting/borders.dart BorderSide.none
var BorderSideNone = BorderSide{Color: borderSideDefaultColor, Width: 0, Style: BorderStyleNone, StrokeAlign: StrokeAlignInside}

// normalize returns BorderSideNone for the zero BorderSide, which a struct
// literal such as Border{Top: side} leaves in the other sides.
func (s BorderSide) normalize() BorderSide {
	if s == (BorderSide{}) {
		return BorderSideNone
	}
	return s
}

// NewBorderSide creates the side of a border. A negative width panics with the
// tag BorderSideWidthNegative in debug builds.
//
// Flutter: painting/borders.dart BorderSide.new
func NewBorderSide(opts ...BorderSideOpts) BorderSide {
	o := firstOpts(opts)
	s := BorderSide{
		Color:       o.Color.OrElse(borderSideDefaultColor),
		Width:       o.Width.OrElse(1.0),
		Style:       o.Style,
		StrokeAlign: o.StrokeAlign.OrElse(StrokeAlignInside),
	}
	if s.Style == BorderStyleUnspecified {
		s.Style = BorderStyleSolid
	}
	if foundation.DebugMode && !(s.Width >= 0) {
		panic(errors.New(tagBorderSideWidthNegative, "the width of a border side must not be negative", "width", s.Width))
	}
	return s
}

// BorderSideCopy holds the values BorderSide.CopyWith replaces.
//
// Flutter: painting/borders.dart BorderSide.copyWith
type BorderSideCopy struct {
	Color       foundation.Opt[ui.Color]
	Width       foundation.Opt[float64]
	Style       BorderStyle
	StrokeAlign foundation.Opt[float64]
}

// CopyWith returns a copy with the given values replaced.
//
// Flutter: painting/borders.dart BorderSide.copyWith
func (s BorderSide) CopyWith(c BorderSideCopy) BorderSide {
	style := s.Style
	if c.Style != BorderStyleUnspecified {
		style = c.Style
	}
	return NewBorderSide(BorderSideOpts{Color: foundation.Some(c.Color.OrElse(s.Color)), Width: foundation.Some(c.Width.OrElse(s.Width)),
		Style: style, StrokeAlign: foundation.Some(c.StrokeAlign.OrElse(s.StrokeAlign))})
}

// Scale creates a copy of this border side description but with the width scaled
// by the factor t. The result has no style if t is not positive.
//
// Flutter: painting/borders.dart BorderSide.scale
func (s BorderSide) Scale(t float64) BorderSide {
	style := s.Style
	if t <= 0 {
		style = BorderStyleNone
	}
	return BorderSide{Color: s.Color, Width: math.Max(0, s.Width*t), Style: style, StrokeAlign: StrokeAlignInside}
}

// ToPaint creates a Paint object that, if used to stroke a line, will draw the
// line in this border's style.
//
// Flutter: painting/borders.dart BorderSide.toPaint
func (s BorderSide) ToPaint() *ui.Paint {
	p := ui.NewPaint()
	p.SetStyle(ui.PaintingStyleStroke)
	if s.Style == BorderStyleSolid {
		p.SetColor(s.Color)
		p.SetStrokeWidth(s.Width)
	} else {
		p.SetColor(ui.NewColor(0x00000000))
		p.SetStrokeWidth(0)
	}
	return p
}

func (s BorderSide) isNone() bool { return s.Style != BorderStyleSolid && s.Width == 0 }

// BorderSideCanMerge reports whether a and b can be merged by BorderSideMerge:
// one of them is BorderSideNone-like, or they have the same style and color.
//
// Flutter: painting/borders.dart BorderSide.canMerge
func BorderSideCanMerge(a, b BorderSide) bool {
	if a.isNone() || b.isNone() {
		return true
	}
	return a.Style == b.Style && a.Color == b.Color
}

// BorderSideMerge creates a BorderSide that represents the addition of the two
// given BorderSides. Merging sides that cannot be merged panics with the tag
// BorderSideCannotMerge in debug builds.
//
// Flutter: painting/borders.dart BorderSide.merge
func BorderSideMerge(a, b BorderSide) BorderSide {
	if foundation.DebugMode && !BorderSideCanMerge(a, b) {
		panic(errors.New(tagBorderSideCannotMerge, "the border sides have different styles or colors"))
	}
	aNone, bNone := a.isNone(), b.isNone()
	switch {
	case aNone && bNone:
		return BorderSideNone
	case aNone:
		return b
	case bNone:
		return a
	}
	return BorderSide{Color: a.Color, Width: a.Width + b.Width, Style: a.Style, StrokeAlign: math.Max(a.StrokeAlign, b.StrokeAlign)}
}

// BorderSideLerp linearly interpolates between two border sides.
//
// Flutter: painting/borders.dart BorderSide.lerp
func BorderSideLerp(a, b BorderSide, t float64) BorderSide {
	if a == b {
		return a
	}
	if t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	width := ui.LerpDouble(a.Width, b.Width, t)
	if width < 0 {
		return BorderSideNone
	}
	if a.Style == b.Style && a.StrokeAlign == b.StrokeAlign {
		return BorderSide{Color: a.Color.Lerp(b.Color, t), Width: width, Style: a.Style, StrokeAlign: a.StrokeAlign}
	}
	colorA, colorB := a.Color, b.Color
	if a.Style != BorderStyleSolid {
		colorA = a.Color.WithAlpha(0)
	}
	if b.Style != BorderStyleSolid {
		colorB = b.Color.WithAlpha(0)
	}
	align := a.StrokeAlign
	if a.StrokeAlign != b.StrokeAlign {
		align = ui.LerpDouble(a.StrokeAlign, b.StrokeAlign, t)
	}
	return BorderSide{Color: colorA.Lerp(colorB, t), Width: width, Style: BorderStyleSolid, StrokeAlign: align}
}

// StrokeInset is the amount of the stroke width that lies inside of the
// BorderSide.
//
// Flutter: painting/borders.dart BorderSide.strokeInset
func (s BorderSide) StrokeInset() float64 { return s.Width * (1 - (1+s.StrokeAlign)/2) }

// StrokeOutset is the amount of the stroke width that lies outside of the
// BorderSide.
//
// Flutter: painting/borders.dart BorderSide.strokeOutset
func (s BorderSide) StrokeOutset() float64 { return s.Width * (1 + s.StrokeAlign) / 2 }

// StrokeOffset is the offset of the stroke, taking into account the stroke
// alignment.
//
// Flutter: painting/borders.dart BorderSide.strokeOffset
func (s BorderSide) StrokeOffset() float64 { return s.Width * s.StrokeAlign }

// ToStringShort names the side.
//
// Flutter: painting/borders.dart BorderSide.toStringShort
func (s BorderSide) ToStringShort() string { return "BorderSide" }

// DebugFillProperties adds the color, width, stroke alignment and style.
//
// Flutter: painting/borders.dart BorderSide.debugFillProperties
func (s BorderSide) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("color", s.Color, foundation.DiagnosticsPropertyOpts{DefaultValue: borderSideDefaultColor}))
	p.Add(foundation.NewDoubleProperty("width", foundation.Some(s.Width), foundation.NumberPropertyOpts{DefaultValue: 1.0}))
	p.Add(foundation.NewDoubleProperty("strokeAlign", foundation.Some(s.StrokeAlign), foundation.NumberPropertyOpts{DefaultValue: StrokeAlignInside}))
	p.Add(foundation.NewEnumProperty("style", foundation.Some(s.Style), foundation.EnumPropertyOpts{DefaultValue: BorderStyleSolid}))
}

// ShapeBorder is the base of shapes that can be drawn as borders around a box
// and used as the outline of a shape. The methods Add, LerpFrom and LerpTo are
// protected in Flutter; they are exported so that other packages can implement
// a ShapeBorder. A nil ShapeBorder is Flutter's null. Embed ShapeBorderDefaults
// for the default Add, PaintInterior and PreferPaintInterior.
//
// Flutter: painting/borders.dart ShapeBorder
type ShapeBorder interface {
	// Dimensions is the widths of the border: the insets that make the area a
	// widget can use inside the border.
	Dimensions() EdgeInsetsGeometry
	// Add attempts to create a new object that represents the amalgamation of
	// this border and the other (placed on top of it, or below it if reversed).
	// It returns nil if it can't.
	Add(other ShapeBorder, reversed bool) ShapeBorder
	// Scale creates a copy of the border scaled by the factor t.
	Scale(t float64) ShapeBorder
	// LerpFrom linearly interpolates from another ShapeBorder (possibly nil) to
	// this one; it returns nil if it can't.
	LerpFrom(a ShapeBorder, t float64) ShapeBorder
	// LerpTo linearly interpolates from this to another ShapeBorder (possibly
	// nil); it returns nil if it can't.
	LerpTo(b ShapeBorder, t float64) ShapeBorder
	// GetOuterPath creates a Path that describes the outer edge of the border.
	GetOuterPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path
	// GetInnerPath creates a Path that describes the inner edge of the border.
	GetInnerPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path
	// PaintInterior paints the interior of the border, if PreferPaintInterior
	// is true.
	PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection ui.TextDirection)
	// PreferPaintInterior reports whether PaintInterior is faster than
	// filling the path of GetOuterPath.
	PreferPaintInterior() bool
	// Paint paints the border within the given rect on the given canvas.
	Paint(canvas *ui.Canvas, rect ui.Rect, textDirection ui.TextDirection)
	String() string
}

// ShapeBorderDefaults supplies the default Add (nil), PaintInterior (a debug
// panic) and PreferPaintInterior (false) of ShapeBorder; embed it in a border.
//
// Flutter: painting/borders.dart ShapeBorder
type ShapeBorderDefaults struct{}

// Add returns nil: the borders can't be amalgamated.
//
// Flutter: painting/borders.dart ShapeBorder.add
func (ShapeBorderDefaults) Add(ShapeBorder, bool) ShapeBorder { return nil }

// PreferPaintInterior is false.
//
// Flutter: painting/borders.dart ShapeBorder.preferPaintInterior
func (ShapeBorderDefaults) PreferPaintInterior() bool { return false }

// PaintInterior panics in debug builds with the tag ShapeBorderNoPaintInterior:
// the border said it does not prefer to paint its interior.
//
// Flutter: painting/borders.dart ShapeBorder.paintInterior
func (ShapeBorderDefaults) PaintInterior(*ui.Canvas, ui.Rect, *ui.Paint, ui.TextDirection) {
	if foundation.DebugMode {
		panic(errors.New(tagShapeBorderNoPaintInterior,
			"PreferPaintInterior returns false, so it is an error to call PaintInterior"))
	}
}

// ShapeBorderDefaultLerpFrom is the default LerpFrom of a ShapeBorder: the
// scaled border when a is nil, and nil otherwise.
//
// Flutter: painting/borders.dart ShapeBorder.lerpFrom
func ShapeBorderDefaultLerpFrom(self, a ShapeBorder, t float64) ShapeBorder {
	if a == nil {
		return self.Scale(t)
	}
	return nil
}

// ShapeBorderDefaultLerpTo is the default LerpTo of a ShapeBorder: the scaled
// border when b is nil, and nil otherwise.
//
// Flutter: painting/borders.dart ShapeBorder.lerpTo
func ShapeBorderDefaultLerpTo(self, b ShapeBorder, t float64) ShapeBorder {
	if b == nil {
		return self.Scale(1.0 - t)
	}
	return nil
}

// ShapeBorderPlus creates a new border consisting of the two borders: the other
// border on the outside, then this one.
//
// Flutter: painting/borders.dart ShapeBorder.operator +
func ShapeBorderPlus(a, other ShapeBorder) ShapeBorder {
	if r := a.Add(other, false); r != nil {
		return r
	}
	if r := other.Add(a, true); r != nil {
		return r
	}
	return &compoundBorder{borders: []ShapeBorder{other, a}}
}

// ShapeBorderLerp linearly interpolates between two ShapeBorders. Either may be
// nil. When neither border knows how to interpolate with the other, the result
// is a for t < 0.5 and b after.
//
// Flutter: painting/borders.dart ShapeBorder.lerp
func ShapeBorderLerp(a, b ShapeBorder, t float64) ShapeBorder {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && sameBorder(a, b) {
		return a
	}
	if b != nil {
		if r := b.LerpFrom(a, t); r != nil {
			return r
		}
	}
	if a != nil {
		if r := a.LerpTo(b, t); r != nil {
			return r
		}
	}
	if b != nil {
		if r := b.LerpTo(a, 1.0-t); r != nil {
			return r
		}
	}
	if a != nil {
		if r := a.LerpFrom(b, 1.0-t); r != nil {
			return r
		}
	}
	if t < 0.5 {
		return a
	}
	return b
}

// sameBorder is Dart's identical for borders that are comparable values.
func sameBorder(a, b ShapeBorder) (same bool) {
	defer func() {
		if recover() != nil {
			same = false // not comparable
		}
	}()
	return a == b
}

// OutlinedBorder is a ShapeBorder that draws an outline with the width and color
// specified by its BorderSide.
//
// Flutter: painting/borders.dart OutlinedBorder
type OutlinedBorder interface {
	ShapeBorder
	// Side is the border outline's color and weight.
	Side() BorderSide
	// CopyWithSide returns a copy with the given side. The concrete types have
	// a CopyWith with all their properties.
	CopyWithSide(side BorderSide) OutlinedBorder
}

// OutlinedBorderDimensions is the default Dimensions of an OutlinedBorder: the
// stroke inset of the side on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func OutlinedBorderDimensions(side BorderSide) EdgeInsetsGeometry {
	return NewEdgeInsetsAll(math.Max(side.StrokeInset(), 0))
}

// OutlinedBorderLerp linearly interpolates between two OutlinedBorders.
//
// Flutter: painting/borders.dart OutlinedBorder.lerp
func OutlinedBorderLerp(a, b OutlinedBorder, t float64) OutlinedBorder {
	if a == nil && b == nil {
		return nil
	}
	var sa, sb ShapeBorder
	if a != nil {
		sa = a
	}
	if b != nil {
		sb = b
	}
	if sa != nil && sb != nil && sameBorder(sa, sb) {
		return a
	}
	var result ShapeBorder
	if sb != nil {
		result = sb.LerpFrom(sa, t)
	}
	if result == nil && sa != nil {
		result = sa.LerpTo(sb, t)
	}
	if result == nil && sb != nil {
		result = sb.LerpTo(sa, 1.0-t)
	}
	if result == nil && sa != nil {
		result = sa.LerpFrom(sb, 1.0-t)
	}
	if o, ok := result.(OutlinedBorder); ok && result != nil {
		return o
	}
	if t < 0.5 {
		return a
	}
	return b
}

// compoundBorder is Flutter's _CompoundBorder: several borders painted one inside
// the other.
type compoundBorder struct {
	ShapeBorderDefaults
	borders []ShapeBorder
}

func (c *compoundBorder) Dimensions() EdgeInsetsGeometry {
	var d EdgeInsetsGeometry = EdgeInsetsZero
	for _, b := range c.borders {
		d = d.Add(b.Dimensions())
	}
	return d
}

func (c *compoundBorder) Add(other ShapeBorder, reversed bool) ShapeBorder {
	if oc, ok := other.(*compoundBorder); !ok || oc == nil {
		ours := c.borders[0]
		if reversed {
			ours = c.borders[len(c.borders)-1]
		}
		merged := ours.Add(other, reversed)
		if merged == nil {
			merged = other.Add(ours, !reversed)
		}
		if merged != nil {
			result := append([]ShapeBorder(nil), c.borders...)
			if reversed {
				result[len(result)-1] = merged
			} else {
				result[0] = merged
			}
			return &compoundBorder{borders: result}
		}
	}
	var merged []ShapeBorder
	if reversed {
		merged = append(merged, c.borders...)
	}
	if oc, ok := other.(*compoundBorder); ok {
		merged = append(merged, oc.borders...)
	} else {
		merged = append(merged, other)
	}
	if !reversed {
		merged = append(merged, c.borders...)
	}
	return &compoundBorder{borders: merged}
}

func (c *compoundBorder) Scale(t float64) ShapeBorder {
	scaled := make([]ShapeBorder, len(c.borders))
	for i, b := range c.borders {
		scaled[i] = b.Scale(t)
	}
	return &compoundBorder{borders: scaled}
}

func (c *compoundBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	return lerpCompound(a, c, t)
}

func (c *compoundBorder) LerpTo(b ShapeBorder, t float64) ShapeBorder {
	return lerpCompound(c, b, t)
}

func lerpCompound(a, b ShapeBorder, t float64) *compoundBorder {
	var aList, bList []ShapeBorder
	if ac, ok := a.(*compoundBorder); ok {
		aList = ac.borders
	} else {
		aList = []ShapeBorder{a}
	}
	if bc, ok := b.(*compoundBorder); ok {
		bList = bc.borders
	} else {
		bList = []ShapeBorder{b}
	}
	var results []ShapeBorder
	length := len(aList)
	if len(bList) > length {
		length = len(bList)
	}
	for i := 0; i < length; i++ {
		var la, lb ShapeBorder
		if i < len(aList) {
			la = aList[i]
		}
		if i < len(bList) {
			lb = bList[i]
		}
		if la != nil && lb != nil {
			r := la.LerpTo(lb, t)
			if r == nil {
				r = lb.LerpFrom(la, t)
			}
			if r != nil {
				results = append(results, r)
				continue
			}
		}
		if lb != nil {
			results = append(results, lb.Scale(t))
		}
		if la != nil {
			results = append(results, la.Scale(1.0-t))
		}
	}
	return &compoundBorder{borders: results}
}

func (c *compoundBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	for i := 0; i < len(c.borders)-1; i++ {
		rect = c.borders[i].Dimensions().Resolve(td).DeflateRect(rect)
	}
	return c.borders[len(c.borders)-1].GetInnerPath(rect, td)
}

func (c *compoundBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return c.borders[0].GetOuterPath(rect, td)
}

func (c *compoundBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	c.borders[0].PaintInterior(canvas, rect, paint, td)
}

func (c *compoundBorder) PreferPaintInterior() bool {
	for _, b := range c.borders {
		if !b.PreferPaintInterior() {
			return false
		}
	}
	return true
}

func (c *compoundBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	for _, b := range c.borders {
		b.Paint(canvas, rect, td)
		rect = b.Dimensions().Resolve(td).DeflateRect(rect)
	}
}

func (c *compoundBorder) String() string {
	parts := make([]string, len(c.borders))
	for i, b := range c.borders {
		parts[len(c.borders)-1-i] = b.String()
	}
	return strings.Join(parts, " + ")
}

// BorderPaintOpts are the sides PaintBorder draws; a side that is left out is
// not drawn.
//
// Flutter: painting/borders.dart paintBorder
type BorderPaintOpts struct {
	Top, Right, Bottom, Left BorderSide
}

// PaintBorder paints a border around the given rectangle on the canvas. The four
// sides can have different colors, and the corners are mitered.
//
// Flutter: painting/borders.dart paintBorder
func PaintBorder(canvas *ui.Canvas, rect ui.Rect, opts ...BorderPaintOpts) {
	o := firstOpts(opts)
	top, right, bottom, left := o.Top, o.Right, o.Bottom, o.Left
	paint := ui.NewPaint()
	paint.SetStrokeWidth(0)
	path := ui.NewPath()
	if top.Style == BorderStyleSolid {
		paint.SetColor(top.Color)
		path.Reset()
		path.MoveTo(rect.Left, rect.Top)
		path.LineTo(rect.Right, rect.Top)
		if top.Width == 0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Right-right.Width, rect.Top+top.Width)
			path.LineTo(rect.Left+left.Width, rect.Top+top.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if right.Style == BorderStyleSolid {
		paint.SetColor(right.Color)
		path.Reset()
		path.MoveTo(rect.Right, rect.Top)
		path.LineTo(rect.Right, rect.Bottom)
		if right.Width == 0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Right-right.Width, rect.Bottom-bottom.Width)
			path.LineTo(rect.Right-right.Width, rect.Top+top.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if bottom.Style == BorderStyleSolid {
		paint.SetColor(bottom.Color)
		path.Reset()
		path.MoveTo(rect.Right, rect.Bottom)
		path.LineTo(rect.Left, rect.Bottom)
		if bottom.Width == 0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Left+left.Width, rect.Bottom-bottom.Width)
			path.LineTo(rect.Right-right.Width, rect.Bottom-bottom.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if left.Style == BorderStyleSolid {
		paint.SetColor(left.Color)
		path.Reset()
		path.MoveTo(rect.Left, rect.Bottom)
		path.LineTo(rect.Left, rect.Top)
		if left.Width == 0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Left+left.Width, rect.Top+top.Width)
			path.LineTo(rect.Left+left.Width, rect.Bottom-bottom.Width)
		}
		canvas.DrawPath(path, paint)
	}
}

// String describes the side, such as "BorderSide(color: Color(0xff000000), width: 1.0)".
//
// Flutter: painting/borders.dart BorderSide.toString
func (s BorderSide) String() string { return foundation.DiagnosticableString(s) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBorderSideCannotMerge      = "BorderSideCannotMerge"
	tagBorderSideWidthNegative    = "BorderSideWidthNegative"
	tagShapeBorderNoPaintInterior = "ShapeBorderNoPaintInterior"
)
