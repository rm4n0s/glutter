package painting

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ShapeDecoration is an immutable description of how to paint an arbitrary
// shape.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration
type ShapeDecoration struct {
	DecorationDefaults
	// Color is the color to fill in the background of the shape.
	Color foundation.Opt[ui.Color]
	// Image is the image to paint inside the shape, above Color and Gradient.
	Image DecorationImageSource
	// Gradient is the gradient to use when filling the shape.
	Gradient Gradient
	// Shadows is a list of shadows cast by this shape behind the shape. Nil is
	// Flutter's null.
	Shadows []BoxShadow
	// Shape is the shape to fill the Color, Gradient and Image into and to cast
	// as the Shadows.
	Shape ShapeBorder
}

// ShapeDecorationOpts are the optional parameters of NewShapeDecoration.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.new
type ShapeDecorationOpts struct {
	Color    foundation.Opt[ui.Color]
	Image    DecorationImageSource
	Gradient Gradient
	Shadows  []BoxShadow
}

// NewShapeDecoration creates a shape decoration. In debug builds a color
// together with a gradient panics with the tag ShapeDecorationColorAndGradient.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.new
func NewShapeDecoration(shape ShapeBorder, opts ...ShapeDecorationOpts) *ShapeDecoration {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Color.IsSome() && o.Gradient != nil {
		panic(errors.New(tagShapeDecorationColorAndGradient, "a ShapeDecoration can't have both a color and a gradient"))
	}
	return &ShapeDecoration{Color: o.Color, Image: o.Image, Gradient: o.Gradient, Shadows: o.Shadows, Shape: shape}
}

// NewShapeDecorationFromBoxDecoration creates a shape decoration out of a box
// decoration. In debug builds a non-uniform border where a shape needs a
// uniform one panics with the tag ShapeDecorationBorderNotUniform.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.fromBoxDecoration
func NewShapeDecorationFromBoxDecoration(source *BoxDecoration) *ShapeDecoration {
	var shape ShapeBorder
	uniform := func() {
		if foundation.DebugMode && source.Border != nil && !source.Border.IsUniform() {
			panic(errors.New(tagShapeDecorationBorderNotUniform, "the border of the BoxDecoration must be uniform to become a shape"))
		}
	}
	switch source.Shape {
	case BoxShapeCircle:
		if source.Border != nil {
			uniform()
			shape = NewCircleBorder(CircleBorderOpts{Side: source.Border.TopSide()})
		} else {
			shape = NewCircleBorder()
		}
	default:
		if source.BorderRadius != nil {
			uniform()
			side := BorderSideNone
			if source.Border != nil {
				side = source.Border.TopSide()
			}
			shape = NewRoundedRectangleBorder(RoundedRectangleBorderOpts{Side: side, BorderRadius: source.BorderRadius})
		} else if source.Border != nil {
			shape = source.Border
		} else {
			shape = Border{}
		}
	}
	return NewShapeDecoration(shape, ShapeDecorationOpts{Color: source.Color, Image: source.Image, Gradient: source.Gradient, Shadows: source.BoxShadow})
}

// GetClipPath is the outer path of the shape.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.getClipPath
func (d *ShapeDecoration) GetClipPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return d.Shape.GetOuterPath(rect, td)
}

// Padding is the dimensions of the shape.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.padding
func (d *ShapeDecoration) Padding() EdgeInsetsGeometry { return d.Shape.Dimensions() }

// IsComplex is whether the decoration has shadows.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.isComplex
func (d *ShapeDecoration) IsComplex() bool { return d.Shadows != nil }

// LerpFrom interpolates from a BoxDecoration, a ShapeDecoration or nil.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.lerpFrom
func (d *ShapeDecoration) LerpFrom(a Decoration, t float64) Decoration {
	switch x := a.(type) {
	case nil:
		return ShapeDecorationLerp(nil, d, t)
	case *BoxDecoration:
		return ShapeDecorationLerp(NewShapeDecorationFromBoxDecoration(x), d, t)
	case *ShapeDecoration:
		return ShapeDecorationLerp(x, d, t)
	}
	return nil
}

// LerpTo interpolates to a BoxDecoration, a ShapeDecoration or nil.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.lerpTo
func (d *ShapeDecoration) LerpTo(b Decoration, t float64) Decoration {
	switch x := b.(type) {
	case nil:
		return ShapeDecorationLerp(d, nil, t)
	case *BoxDecoration:
		return ShapeDecorationLerp(d, NewShapeDecorationFromBoxDecoration(x), t)
	case *ShapeDecoration:
		return ShapeDecorationLerp(d, x, t)
	}
	return nil
}

// ShapeDecorationLerp linearly interpolates between two shape decorations.
// Either may be nil.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.lerp
func ShapeDecorationLerp(a, b *ShapeDecoration, t float64) *ShapeDecoration {
	if a == b {
		return a
	}
	if a != nil && b != nil {
		if t == 0.0 {
			return a
		}
		if t == 1.0 {
			return b
		}
	}
	var aGradient, bGradient Gradient
	var aColor, bColor foundation.Opt[ui.Color]
	var aImage, bImage DecorationImageSource
	var aShadows, bShadows []BoxShadow
	var aShape, bShape ShapeBorder
	if a != nil {
		aGradient, aColor, aImage, aShadows, aShape = a.Gradient, a.Color, a.Image, a.Shadows, a.Shape
	}
	if b != nil {
		bGradient, bColor, bImage, bShadows, bShape = b.Gradient, b.Color, b.Image, b.Shadows, b.Shape
	}
	if aGradient == nil && bGradient != nil && aColor.IsSome() {
		aGradient = bGradient.FromColor(aColor.Value())
	} else if bGradient == nil && aGradient != nil && bColor.IsSome() {
		bGradient = aGradient.FromColor(bColor.Value())
	}
	gradient := GradientLerp(aGradient, bGradient, t)
	var color foundation.Opt[ui.Color]
	if gradient == nil {
		color = lerpOptColor(aColor, bColor, t)
	}
	return NewShapeDecoration(ShapeBorderLerp(aShape, bShape, t), ShapeDecorationOpts{
		Color: color, Gradient: gradient, Image: DecorationImageLerp(aImage, bImage, t), Shadows: BoxShadowLerpList(aShadows, bShadows, t),
	})
}

func lerpOptColor(a, b foundation.Opt[ui.Color], t float64) foundation.Opt[ui.Color] {
	if c := ui.ColorLerp(optColorPtr(a), optColorPtr(b), t); c != nil {
		return foundation.Some(*c)
	}
	return foundation.None[ui.Color]()
}

// Equal reports whether the other decoration is an equal ShapeDecoration.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.operator ==
func (d *ShapeDecoration) Equal(other *ShapeDecoration) bool {
	if d == other {
		return true
	}
	if d == nil || other == nil || d.Color != other.Color || !decorationImageEqual(d.Image, other.Image) {
		return false
	}
	if (d.Gradient == nil) != (other.Gradient == nil) || (d.Gradient != nil && !d.Gradient.Equal(other.Gradient)) {
		return false
	}
	if (d.Shadows == nil) != (other.Shadows == nil) || len(d.Shadows) != len(other.Shadows) {
		return false
	}
	for i := range d.Shadows {
		if d.Shadows[i] != other.Shadows[i] {
			return false
		}
	}
	return sameBorder(d.Shape, other.Shape)
}

// ToStringShort names the decoration.
//
// Flutter: painting/decoration.dart Decoration.toStringShort
func (d *ShapeDecoration) ToStringShort() string { return "ShapeDecoration" }

// String describes the decoration and its properties.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.toString
func (d *ShapeDecoration) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the color, gradient, image, shadows and shape.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.debugFillProperties
func (d *ShapeDecoration) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	null := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	p.Add(NewColorProperty("color", d.Color, null))
	p.Add(foundation.NewDiagnosticsProperty("gradient", d.Gradient, null))
	p.Add(foundation.NewDiagnosticsProperty("image", d.Image, null))
	p.Add(foundation.NewIterableProperty("shadows", d.Shadows,
		foundation.IterablePropertyOpts{DefaultValue: foundation.DefaultNull, Style: foundation.DiagnosticsTreeStyleWhitespace}))
	p.Add(foundation.NewDiagnosticsProperty("shape", d.Shape))
}

// HitTest tests whether the position hits the outer path of the shape.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.hitTest
func (d *ShapeDecoration) HitTest(size ui.Size, position ui.Offset, td ui.TextDirection) bool {
	return d.Shape.GetOuterPath((ui.Offset{}).And(size), td).Contains(position)
}

// CreateBoxPainter returns a painter for the decoration. In debug builds it
// panics with the tag ShapeDecorationOnChangedRequired if there is an image but
// no onChanged callback.
//
// Flutter: painting/shape_decoration.dart ShapeDecoration.createBoxPainter
func (d *ShapeDecoration) CreateBoxPainter(onChanged func()) BoxPainter {
	if foundation.DebugMode && onChanged == nil && d.Image != nil {
		panic(errors.New(tagShapeDecorationOnChangedRequired, "a ShapeDecoration with an image needs an onChanged callback"))
	}
	return &shapeDecorationPainter{decoration: d, onChanged: onChanged}
}

type shapeDecorationPainter struct {
	decoration *ShapeDecoration
	onChanged  func()

	lastRect          *ui.Rect
	lastTextDirection ui.TextDirection
	outerPath         *ui.Path
	innerPath         *ui.Path
	interiorPaint     *ui.Paint
	hasShadows        bool
	shadowBounds      []ui.Rect
	shadowPaths       []*ui.Path
	shadowPaints      []*ui.Paint
	imagePainter      DecorationImagePainter
}

func (s *shapeDecorationPainter) precache(rect ui.Rect, td ui.TextDirection) {
	if s.lastRect != nil && *s.lastRect == rect && s.lastTextDirection == td {
		return
	}
	d := s.decoration
	if s.interiorPaint == nil && (d.Color.IsSome() || d.Gradient != nil) {
		s.interiorPaint = ui.NewPaint()
		if c, ok := d.Color.Get(); ok {
			s.interiorPaint.SetColor(c)
		}
	}
	if d.Gradient != nil {
		s.interiorPaint.SetShader(d.Gradient.CreateShader(rect, td))
	}
	if d.Shadows != nil {
		if !s.hasShadows {
			s.hasShadows = true
			s.shadowPaints = make([]*ui.Paint, len(d.Shadows))
			for i, shadow := range d.Shadows {
				s.shadowPaints[i] = shadow.ToPaint()
			}
		}
		if d.Shape.PreferPaintInterior() {
			s.shadowBounds = make([]ui.Rect, len(d.Shadows))
			for i, shadow := range d.Shadows {
				s.shadowBounds[i] = rect.Shift(shadow.Offset).Inflate(shadow.SpreadRadius)
			}
		} else {
			s.shadowPaths = make([]*ui.Path, len(d.Shadows))
			for i, shadow := range d.Shadows {
				s.shadowPaths[i] = d.Shape.GetOuterPath(rect.Shift(shadow.Offset).Inflate(shadow.SpreadRadius), td)
			}
		}
	}
	if !d.Shape.PreferPaintInterior() && (s.interiorPaint != nil || s.hasShadows) {
		s.outerPath = d.Shape.GetOuterPath(rect, td)
	}
	if d.Image != nil {
		s.innerPath = d.Shape.GetInnerPath(rect, td)
	}
	r := rect
	s.lastRect = &r
	s.lastTextDirection = td
}

func (s *shapeDecorationPainter) paintShadows(canvas *ui.Canvas, td ui.TextDirection) {
	if !s.hasShadows {
		return
	}
	for i := range s.decoration.Shadows {
		if s.decoration.Shape.PreferPaintInterior() {
			s.decoration.Shape.PaintInterior(canvas, s.shadowBounds[i], s.shadowPaints[i], td)
		} else {
			canvas.DrawPath(s.shadowPaths[i], s.shadowPaints[i])
		}
	}
}

func (s *shapeDecorationPainter) paintInterior(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	if s.interiorPaint == nil {
		return
	}
	if s.decoration.Shape.PreferPaintInterior() {
		s.decoration.Shape.PaintInterior(canvas, s.adjustedRectOnOutlinedBorder(rect), s.interiorPaint, td)
	} else {
		canvas.DrawPath(s.outerPath, s.interiorPaint)
	}
}

func (s *shapeDecorationPainter) adjustedRectOnOutlinedBorder(rect ui.Rect) ui.Rect {
	if o, ok := s.decoration.Shape.(OutlinedBorder); ok && s.decoration.Color.IsSome() {
		side := o.Side()
		if side.Color.Alpha() == 255 && side.Style == BorderStyleSolid {
			return rect.Deflate(side.StrokeInset() / 2)
		}
	}
	return rect
}

func (s *shapeDecorationPainter) paintImage(canvas *ui.Canvas, configuration ImageConfiguration) {
	if s.decoration.Image == nil {
		return
	}
	if s.imagePainter == nil {
		s.imagePainter = s.decoration.Image.CreatePainter(s.onChanged)
	}
	s.imagePainter.Paint(canvas, *s.lastRect, s.innerPath, configuration)
}

// Dispose disposes the image painter.
//
// Flutter: painting/shape_decoration.dart _ShapeDecorationPainter.dispose
func (s *shapeDecorationPainter) Dispose() {
	if s.imagePainter != nil {
		s.imagePainter.Dispose()
	}
}

// Paint paints shadows, interior, image and the border of the shape. In debug
// builds it panics with the tag ShapeDecorationNoSize if the configuration has
// no size.
//
// Flutter: painting/shape_decoration.dart _ShapeDecorationPainter.paint
func (s *shapeDecorationPainter) Paint(canvas *ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	size, ok := configuration.Size.Get()
	if !ok {
		if foundation.DebugMode {
			panic(errors.New(tagShapeDecorationNoSize, "the ImageConfiguration given to a ShapeDecoration painter has no size"))
		}
		return
	}
	rect := offset.And(size)
	td := configuration.TextDirection
	s.precache(rect, td)
	s.paintShadows(canvas, td)
	s.paintInterior(canvas, rect, td)
	s.paintImage(canvas, configuration)
	s.decoration.Shape.Paint(canvas, rect, td)
}

var _ Decoration = (*ShapeDecoration)(nil)

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShapeDecorationBorderNotUniform  = "ShapeDecorationBorderNotUniform"
	tagShapeDecorationColorAndGradient  = "ShapeDecorationColorAndGradient"
	tagShapeDecorationNoSize            = "ShapeDecorationNoSize"
	tagShapeDecorationOnChangedRequired = "ShapeDecorationOnChangedRequired"
)
