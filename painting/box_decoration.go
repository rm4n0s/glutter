package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// DecorationImageSource is the image of a BoxDecoration or ShapeDecoration: an
// image, how to fit it, and so on. Step 6.2 adds *DecorationImage, which
// implements it. A nil DecorationImageSource is Flutter's null.
//
// Flutter: painting/decoration_image.dart DecorationImage
type DecorationImageSource interface {
	// CreatePainter creates a painter that draws the image and calls onChanged
	// when it must be redrawn.
	CreatePainter(onChanged func()) DecorationImagePainter
	// Blended returns the image that interpolates from a to b at t, where one
	// of them is the receiver. It is used by DecorationImageLerp when the two
	// images differ.
	Blended(a, b DecorationImageSource, t float64) DecorationImageSource
	// Equal reports whether the other image is equal.
	Equal(other DecorationImageSource) bool
}

// DecorationImagePainter draws a DecorationImage.
//
// Flutter: painting/decoration_image.dart DecorationImagePainter
type DecorationImagePainter interface {
	// Paint draws the image onto the canvas in the rect, clipped to clipPath
	// when it is not nil.
	Paint(canvas *ui.Canvas, rect ui.Rect, clipPath *ui.Path, configuration ImageConfiguration)
	// Dispose discards the resources used by the painter.
	Dispose()
}

// DecorationImageLerp linearly interpolates between two decoration images,
// either of which may be nil. It paints both on top of each other, blended
// together, when they differ.
//
// Flutter: painting/decoration_image.dart DecorationImage.lerp
func DecorationImageLerp(a, b DecorationImageSource, t float64) DecorationImageSource {
	if a == nil && b == nil {
		return nil
	}
	if (a != nil && b != nil && a.Equal(b)) || t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	if b != nil {
		return b.Blended(a, b, t)
	}
	return a.Blended(a, b, t)
}

func decorationImageEqual(a, b DecorationImageSource) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

// BoxDecoration is an immutable description of how to paint a box.
//
// Flutter: painting/box_decoration.dart BoxDecoration
type BoxDecoration struct {
	DecorationDefaults
	// Color is the color to fill in the background of the box.
	Color foundation.Opt[ui.Color]
	// Image is the image to paint inside the box, above Color and Gradient.
	Image DecorationImageSource
	// Border is the border to draw above the background.
	Border BoxBorder
	// BorderRadius is if non-nil, the corners of this box are rounded.
	BorderRadius BorderRadiusGeometry
	// BoxShadow is a list of shadows cast by this box behind the box. Nil is
	// Flutter's null.
	BoxShadow []BoxShadow
	// Gradient is the gradient to use when filling the box.
	Gradient Gradient
	// BackgroundBlendMode is the blend mode applied to the Color or Gradient
	// background; ui.BlendModeUnspecified for none.
	BackgroundBlendMode ui.BlendMode
	// Shape is the shape to fill the background Color, Gradient and Image into
	// and to cast as the BoxShadow.
	Shape BoxShape
}

// BoxDecorationOpts are the optional parameters of NewBoxDecoration.
//
// Flutter: painting/box_decoration.dart BoxDecoration.new
type BoxDecorationOpts struct {
	Color               foundation.Opt[ui.Color]
	Image               DecorationImageSource
	Border              BoxBorder
	BorderRadius        BorderRadiusGeometry
	BoxShadow           []BoxShadow
	Gradient            Gradient
	BackgroundBlendMode ui.BlendMode
	// Shape defaults to BoxShapeRectangle.
	Shape BoxShape
}

// NewBoxDecoration creates a box decoration. In debug builds it panics with the
// tag BoxDecorationBlendModeWithoutBackground if there is a background blend
// mode but no color or gradient.
//
// Flutter: painting/box_decoration.dart BoxDecoration.new
func NewBoxDecoration(opts ...BoxDecorationOpts) *BoxDecoration {
	o := firstOpts(opts)
	if foundation.DebugMode && o.BackgroundBlendMode != ui.BlendModeUnspecified && o.Color.IsNone() && o.Gradient == nil {
		panic(errors.New("BoxDecorationBlendModeWithoutBackground",
			"backgroundBlendMode applies to BoxDecoration's background color or gradient, but no color or gradient was provided."))
	}
	d := &BoxDecoration{
		Color: o.Color, Image: o.Image, Border: o.Border, BorderRadius: o.BorderRadius, BoxShadow: o.BoxShadow,
		Gradient: o.Gradient, BackgroundBlendMode: o.BackgroundBlendMode, Shape: o.Shape,
	}
	if d.Shape == BoxShapeUnspecified {
		d.Shape = BoxShapeRectangle
	}
	return d
}

// BoxDecorationCopy holds the replacements for BoxDecoration.CopyWith. A field
// that is None, or a nil interface, keeps the old value.
//
// Flutter: painting/box_decoration.dart BoxDecoration.copyWith
type BoxDecorationCopy struct {
	Color               foundation.Opt[ui.Color]
	Image               DecorationImageSource
	Border              BoxBorder
	BorderRadius        BorderRadiusGeometry
	BoxShadow           foundation.Opt[[]BoxShadow]
	Gradient            Gradient
	BackgroundBlendMode ui.BlendMode
	Shape               BoxShape
}

// CopyWith returns a copy of the decoration with the given fields replaced.
//
// Flutter: painting/box_decoration.dart BoxDecoration.copyWith
func (d *BoxDecoration) CopyWith(c BoxDecorationCopy) *BoxDecoration {
	out := *d
	if c.Color.IsSome() {
		out.Color = c.Color
	}
	if c.Image != nil {
		out.Image = c.Image
	}
	if c.Border != nil {
		out.Border = c.Border
	}
	if c.BorderRadius != nil {
		out.BorderRadius = c.BorderRadius
	}
	if c.BoxShadow.IsSome() {
		out.BoxShadow = c.BoxShadow.Value()
	}
	if c.Gradient != nil {
		out.Gradient = c.Gradient
	}
	if c.BackgroundBlendMode != ui.BlendModeUnspecified {
		out.BackgroundBlendMode = c.BackgroundBlendMode
	}
	if c.Shape != BoxShapeUnspecified {
		out.Shape = c.Shape
	}
	return NewBoxDecoration(BoxDecorationOpts{
		Color: out.Color, Image: out.Image, Border: out.Border, BorderRadius: out.BorderRadius, BoxShadow: out.BoxShadow,
		Gradient: out.Gradient, BackgroundBlendMode: out.BackgroundBlendMode, Shape: out.Shape,
	})
}

func boxDecorationCircleCheck(shape BoxShape, borderRadius BorderRadiusGeometry) {
	if foundation.DebugMode && shape == BoxShapeCircle && borderRadius != nil {
		panic(errors.New("BoxDecorationCircleBorderRadius",
			"A circle cannot have a border radius. Remove either the shape or the borderRadius argument."))
	}
}

// DebugAssertIsValid panics with the tag BoxDecorationCircleBorderRadius, in
// debug builds, if the shape is a circle and there is a border radius.
//
// Flutter: painting/box_decoration.dart BoxDecoration.debugAssertIsValid
func (d *BoxDecoration) DebugAssertIsValid() bool {
	boxDecorationCircleCheck(d.Shape, d.BorderRadius)
	return d.DecorationDefaults.DebugAssertIsValid()
}

// Padding is the insets of the border, or zero without one.
//
// Flutter: painting/box_decoration.dart BoxDecoration.padding
func (d *BoxDecoration) Padding() EdgeInsetsGeometry {
	if d.Border == nil {
		return EdgeInsetsZero
	}
	return d.Border.Dimensions()
}

// GetClipPath returns the outline of the box as a path.
//
// Flutter: painting/box_decoration.dart BoxDecoration.getClipPath
func (d *BoxDecoration) GetClipPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if d.Shape == BoxShapeCircle {
		p.AddOval(ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0))
		return p
	}
	if d.BorderRadius != nil {
		p.AddRRect(d.BorderRadius.Resolve(td).ToRRect(rect))
		return p
	}
	p.AddRect(rect)
	return p
}

// Scale returns a new box decoration with its colors and border scaled by the
// factor.
//
// Flutter: painting/box_decoration.dart BoxDecoration.scale
func (d *BoxDecoration) Scale(factor float64) *BoxDecoration {
	var color foundation.Opt[ui.Color]
	if c := ui.ColorLerp(nil, optColorPtr(d.Color), factor); c != nil {
		color = foundation.Some(*c)
	}
	var grad Gradient
	if d.Gradient != nil {
		grad = d.Gradient.Scale(factor)
	}
	return NewBoxDecoration(BoxDecorationOpts{
		Color: color, Image: DecorationImageLerp(nil, d.Image, factor), Border: BoxBorderLerp(nil, d.Border, factor),
		BorderRadius: BorderRadiusGeometryLerp(nil, d.BorderRadius, factor), BoxShadow: BoxShadowLerpList(nil, d.BoxShadow, factor),
		Gradient: grad, Shape: d.Shape,
	})
}

// IsComplex is whether the decoration has shadows.
//
// Flutter: painting/box_decoration.dart BoxDecoration.isComplex
func (d *BoxDecoration) IsComplex() bool { return d.BoxShadow != nil }

// LerpFrom interpolates from another decoration to this one.
//
// Flutter: painting/box_decoration.dart BoxDecoration.lerpFrom
func (d *BoxDecoration) LerpFrom(a Decoration, t float64) Decoration {
	switch x := a.(type) {
	case nil:
		return d.Scale(t)
	case *BoxDecoration:
		return BoxDecorationLerp(x, d, t)
	}
	return nil
}

// LerpTo interpolates from this decoration to another.
//
// Flutter: painting/box_decoration.dart BoxDecoration.lerpTo
func (d *BoxDecoration) LerpTo(b Decoration, t float64) Decoration {
	switch x := b.(type) {
	case nil:
		return d.Scale(1.0 - t)
	case *BoxDecoration:
		return BoxDecorationLerp(d, x, t)
	}
	return nil
}

// BoxDecorationLerp linearly interpolates between two box decorations. Either
// may be nil.
//
// Flutter: painting/box_decoration.dart BoxDecoration.lerp
func BoxDecorationLerp(a, b *BoxDecoration, t float64) *BoxDecoration {
	if a == b {
		return a
	}
	if a == nil {
		return b.Scale(t)
	}
	if b == nil {
		return a.Scale(1.0 - t)
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	var color foundation.Opt[ui.Color]
	if c := ui.ColorLerp(optColorPtr(a.Color), optColorPtr(b.Color), t); c != nil {
		color = foundation.Some(*c)
	}
	shape := b.Shape
	if t < 0.5 {
		shape = a.Shape
	}
	return NewBoxDecoration(BoxDecorationOpts{
		Color: color, Image: DecorationImageLerp(a.Image, b.Image, t), Border: BoxBorderLerp(a.Border, b.Border, t),
		BorderRadius: BorderRadiusGeometryLerp(a.BorderRadius, b.BorderRadius, t),
		BoxShadow:    BoxShadowLerpList(a.BoxShadow, b.BoxShadow, t), Gradient: GradientLerp(a.Gradient, b.Gradient, t), Shape: shape,
	})
}

// Equal reports whether the other decoration is an equal BoxDecoration.
//
// Flutter: painting/box_decoration.dart BoxDecoration.operator ==
func (d *BoxDecoration) Equal(other *BoxDecoration) bool {
	if d == other {
		return true
	}
	if d == nil || other == nil {
		return false
	}
	if d.Color != other.Color || !decorationImageEqual(d.Image, other.Image) || d.BackgroundBlendMode != other.BackgroundBlendMode || d.Shape != other.Shape {
		return false
	}
	if (d.Border == nil) != (other.Border == nil) || (d.Border != nil && !sameValue(d.Border, other.Border)) {
		return false
	}
	if !borderRadiusEqualOrNil(d.BorderRadius, other.BorderRadius) {
		return false
	}
	if (d.BoxShadow == nil) != (other.BoxShadow == nil) || len(d.BoxShadow) != len(other.BoxShadow) {
		return false
	}
	for i := range d.BoxShadow {
		if d.BoxShadow[i] != other.BoxShadow[i] {
			return false
		}
	}
	if (d.Gradient == nil) != (other.Gradient == nil) || (d.Gradient != nil && !d.Gradient.Equal(other.Gradient)) {
		return false
	}
	return true
}

func borderRadiusEqualOrNil(a, b BorderRadiusGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

// ToStringShort names the decoration.
//
// Flutter: painting/decoration.dart Decoration.toStringShort
func (d *BoxDecoration) ToStringShort() string { return "BoxDecoration" }

// String describes the decoration and its properties.
//
// Flutter: painting/box_decoration.dart BoxDecoration.toString
func (d *BoxDecoration) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the color, image, border, border radius, shadows,
// gradient and shape.
//
// Flutter: painting/box_decoration.dart BoxDecoration.debugFillProperties
func (d *BoxDecoration) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	p.EmptyBodyDescription = foundation.Some("<no decorations specified>")
	null := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	p.Add(NewColorProperty("color", d.Color, null))
	p.Add(foundation.NewDiagnosticsProperty("image", d.Image, null))
	p.Add(foundation.NewDiagnosticsProperty("border", d.Border, null))
	p.Add(foundation.NewDiagnosticsProperty("borderRadius", d.BorderRadius, null))
	p.Add(foundation.NewIterableProperty("boxShadow", d.BoxShadow,
		foundation.IterablePropertyOpts{DefaultValue: foundation.DefaultNull, Style: foundation.DiagnosticsTreeStyleWhitespace}))
	p.Add(foundation.NewDiagnosticsProperty("gradient", d.Gradient, null))
	p.Add(foundation.NewEnumProperty("shape", foundation.Some(d.Shape), foundation.EnumPropertyOpts{DefaultValue: BoxShapeRectangle}))
}

// HitTest tests whether the position, on a rectangle of the given size, hits
// the shape of the box. In debug builds it panics with the tag
// DecorationHitTestOutside if the position is outside the rectangle.
//
// Flutter: painting/box_decoration.dart BoxDecoration.hitTest
func (d *BoxDecoration) HitTest(size ui.Size, position ui.Offset, td ui.TextDirection) bool {
	if foundation.DebugMode && !(ui.Offset{}).And(size).Contains(position) {
		panic(errors.New("DecorationHitTestOutside", "the position given to hitTest is outside the rectangle of the given size",
			"size", size.String(), "position", position.String()))
	}
	if d.Shape == BoxShapeCircle {
		center := size.Center(ui.Offset{})
		return position.Sub(center).Distance() <= math.Min(size.Width, size.Height)/2.0
	}
	if d.BorderRadius != nil {
		return d.BorderRadius.Resolve(td).ToRRect((ui.Offset{}).And(size)).Contains(position)
	}
	return true
}

// CreateBoxPainter returns a painter for the decoration. In debug builds it
// panics with the tag BoxDecorationOnChangedRequired if there is an image but
// no onChanged callback.
//
// Flutter: painting/box_decoration.dart BoxDecoration.createBoxPainter
func (d *BoxDecoration) CreateBoxPainter(onChanged func()) BoxPainter {
	if foundation.DebugMode && onChanged == nil && d.Image != nil {
		panic(errors.New("BoxDecorationOnChangedRequired", "a BoxDecoration with an image needs an onChanged callback"))
	}
	return &boxDecorationPainter{decoration: d, onChanged: onChanged}
}

type boxDecorationPainter struct {
	decoration *BoxDecoration
	onChanged  func()

	cachedBackgroundPaint        *ui.Paint
	rectForCachedBackgroundPaint *ui.Rect
	imagePainter                 DecorationImagePainter
}

func (b *boxDecorationPainter) backgroundPaint(rect ui.Rect, td ui.TextDirection) *ui.Paint {
	d := b.decoration
	if b.cachedBackgroundPaint == nil || (d.Gradient != nil && (b.rectForCachedBackgroundPaint == nil || *b.rectForCachedBackgroundPaint != rect)) {
		paint := ui.NewPaint()
		if d.BackgroundBlendMode != ui.BlendModeUnspecified {
			paint.SetBlendMode(d.BackgroundBlendMode)
		}
		if c, ok := d.Color.Get(); ok {
			paint.SetColor(c)
		}
		if d.Gradient != nil {
			paint.SetShader(d.Gradient.CreateShader(rect, td))
			r := rect
			b.rectForCachedBackgroundPaint = &r
		}
		b.cachedBackgroundPaint = paint
	}
	return b.cachedBackgroundPaint
}

func (b *boxDecorationPainter) paintBox(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	d := b.decoration
	if d.Shape == BoxShapeCircle {
		boxDecorationCircleCheck(d.Shape, d.BorderRadius)
		canvas.DrawCircle(rect.Center(), rect.ShortestSide()/2.0, paint)
		return
	}
	if d.BorderRadius == nil || d.BorderRadius.Equal(BorderRadiusZero) {
		canvas.DrawRect(rect, paint)
		return
	}
	canvas.DrawRRect(d.BorderRadius.Resolve(td).ToRRect(rect), paint)
}

func (b *boxDecorationPainter) paintShadows(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	for _, shadow := range b.decoration.BoxShadow {
		bounds := rect.Shift(shadow.Offset).Inflate(shadow.SpreadRadius)
		b.paintBox(canvas, bounds, shadow.ToPaint(), td)
	}
}

func (b *boxDecorationPainter) paintBackgroundColor(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	if b.decoration.Color.IsSome() || b.decoration.Gradient != nil {
		adjusted := b.adjustedRectOnOutlinedBorder(rect, td)
		b.paintBox(canvas, adjusted, b.backgroundPaint(rect, td), td)
	}
}

func calculateAdjustedSide(side BorderSide) float64 {
	side = side.normalize()
	if side.Color.Alpha() == 255 && side.Style == BorderStyleSolid {
		return side.StrokeInset()
	}
	return 0
}

func (b *boxDecorationPainter) adjustedRectOnOutlinedBorder(rect ui.Rect, td ui.TextDirection) ui.Rect {
	var insets EdgeInsets
	switch border := b.decoration.Border.(type) {
	case Border:
		insets = NewEdgeInsetsFromLTRB(calculateAdjustedSide(border.Left), calculateAdjustedSide(border.Top),
			calculateAdjustedSide(border.Right), calculateAdjustedSide(border.Bottom)).Div(2)
	case BorderDirectional:
		if td == ui.TextDirectionUnspecified {
			return rect
		}
		left, right := border.Start, border.End
		if td == ui.TextDirectionRTL {
			left, right = border.End, border.Start
		}
		insets = NewEdgeInsetsFromLTRB(calculateAdjustedSide(left), calculateAdjustedSide(border.Top),
			calculateAdjustedSide(right), calculateAdjustedSide(border.Bottom)).Div(2)
	default:
		return rect
	}
	return ui.NewRectFromLTRB(rect.Left+insets.Left, rect.Top+insets.Top, rect.Right-insets.Right, rect.Bottom-insets.Bottom)
}

func (b *boxDecorationPainter) paintBackgroundImage(canvas *ui.Canvas, rect ui.Rect, configuration ImageConfiguration) {
	d := b.decoration
	if d.Image == nil {
		return
	}
	if b.imagePainter == nil {
		b.imagePainter = d.Image.CreatePainter(b.onChanged)
	}
	var clipPath *ui.Path
	if d.Shape == BoxShapeCircle {
		boxDecorationCircleCheck(d.Shape, d.BorderRadius)
		clipPath = ui.NewPath()
		clipPath.AddOval(ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0))
	} else if d.BorderRadius != nil {
		clipPath = ui.NewPath()
		clipPath.AddRRect(d.BorderRadius.Resolve(configuration.TextDirection).ToRRect(rect))
	}
	b.imagePainter.Paint(canvas, rect, clipPath, configuration)
}

// Dispose disposes the image painter.
//
// Flutter: painting/box_decoration.dart _BoxDecorationPainter.dispose
func (b *boxDecorationPainter) Dispose() {
	if b.imagePainter != nil {
		b.imagePainter.Dispose()
	}
}

// Paint paints shadows, background, image and border. In debug builds it
// panics with the tag BoxDecorationNoSize if the configuration has no size.
//
// Flutter: painting/box_decoration.dart _BoxDecorationPainter.paint
func (b *boxDecorationPainter) Paint(canvas *ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	size, ok := configuration.Size.Get()
	if !ok {
		if foundation.DebugMode {
			panic(errors.New("BoxDecorationNoSize", "the ImageConfiguration given to a BoxDecoration painter has no size"))
		}
		return
	}
	rect := offset.And(size)
	td := configuration.TextDirection
	b.paintShadows(canvas, rect, td)
	b.paintBackgroundColor(canvas, rect, td)
	b.paintBackgroundImage(canvas, rect, configuration)
	if border := b.decoration.Border; border != nil {
		opts := BoxPaintOpts{TextDirection: td, Shape: b.decoration.Shape}
		if b.decoration.BorderRadius != nil {
			opts.BorderRadius = foundation.Some(b.decoration.BorderRadius.Resolve(td))
		}
		border.PaintBox(canvas, rect, opts)
	}
}

// String describes the painter.
func (b *boxDecorationPainter) String() string { return "BoxPainter for " + b.decoration.String() }

var _ Decoration = (*BoxDecoration)(nil)

func optColorPtr(c foundation.Opt[ui.Color]) *ui.Color {
	if v, ok := c.Get(); ok {
		return &v
	}
	return nil
}
