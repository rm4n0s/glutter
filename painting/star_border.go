package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

const (
	radToDeg = 180 / math.Pi
	degToRad = math.Pi / 180
)

// StarBorder is a border that fits a star or polygon-shaped border within the
// rectangle of the widget it is applied to.
//
// Flutter: painting/star_border.dart StarBorder
type StarBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// Points is the number of points in this star, or sides on a polygon.
	Points float64
	// innerRadiusRatio is nil for a polygon.
	innerRadiusRatio foundation.Opt[float64]
	// PointRounding is how rounded the points of the star are, from 0 to 1.
	PointRounding float64
	// ValleyRounding is how rounded the valleys of the star are, from 0 to 1.
	ValleyRounding  float64
	rotationRadians float64
	// Squash is how much of the aspect ratio of the attached widget to take on,
	// from 0 (a regular star) to 1 (the star takes the aspect ratio).
	Squash float64
}

// StarBorderOpts are the optional parameters of NewStarBorder.
//
// Flutter: painting/star_border.dart StarBorder.new
type StarBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// Points defaults to 5; it must be at least 2.
	Points foundation.Opt[float64]
	// InnerRadiusRatio defaults to 0.4; it must be from 0 to 1.
	InnerRadiusRatio foundation.Opt[float64]
	// PointRounding defaults to 0.
	PointRounding float64
	// ValleyRounding defaults to 0.
	ValleyRounding float64
	// Rotation is the rotation of the star in degrees.
	Rotation float64
	// Squash defaults to 0.
	Squash float64
}

func starCheck(cond bool, what string, vals ...any) {
	if foundation.DebugMode && !cond {
		panic(errors.New(tagStarBorderInvalid, "invalid StarBorder argument: "+what, append([]any{"check", what}, vals...)...))
	}
}

// NewStarBorder creates a star border. In debug builds an argument out of its
// range panics with the tag StarBorderInvalid.
//
// Flutter: painting/star_border.dart StarBorder.new
func NewStarBorder(opts ...StarBorderOpts) StarBorder {
	o := firstOpts(opts)
	points := o.Points.OrElse(5)
	ratio := o.InnerRadiusRatio.OrElse(0.4)
	starCheck(o.Squash >= 0 && o.Squash <= 1, "squash must be from 0 to 1", "squash", o.Squash)
	starCheck(o.PointRounding >= 0 && o.PointRounding <= 1, "pointRounding must be from 0 to 1", "pointRounding", o.PointRounding)
	starCheck(o.ValleyRounding >= 0 && o.ValleyRounding <= 1, "valleyRounding must be from 0 to 1", "valleyRounding", o.ValleyRounding)
	starCheck(o.ValleyRounding+o.PointRounding <= 1,
		"the sum of valleyRounding and pointRounding must not exceed one", "valleyRounding", o.ValleyRounding, "pointRounding", o.PointRounding)
	starCheck(ratio >= 0 && ratio <= 1, "innerRadiusRatio must be from 0 to 1", "innerRadiusRatio", ratio)
	starCheck(points >= 2, "points must be at least 2", "points", points)
	return StarBorder{
		side: o.Side.normalize(), Points: points, innerRadiusRatio: foundation.Some(ratio), PointRounding: o.PointRounding,
		ValleyRounding: o.ValleyRounding, rotationRadians: o.Rotation * degToRad, Squash: o.Squash,
	}
}

// StarBorderPolygonOpts are the optional parameters of NewStarBorderPolygon.
//
// Flutter: painting/star_border.dart StarBorder.polygon
type StarBorderPolygonOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// Sides defaults to 5; it must be at least 2.
	Sides foundation.Opt[float64]
	// PointRounding defaults to 0.
	PointRounding float64
	// Rotation is the rotation of the polygon in degrees.
	Rotation float64
	// Squash defaults to 0.
	Squash float64
}

// NewStarBorderPolygon creates a polygon border. In debug builds an argument
// out of its range panics with the tag StarBorderInvalid.
//
// Flutter: painting/star_border.dart StarBorder.polygon
func NewStarBorderPolygon(opts ...StarBorderPolygonOpts) StarBorder {
	o := firstOpts(opts)
	sides := o.Sides.OrElse(5)
	starCheck(o.Squash >= 0 && o.Squash <= 1, "squash must be from 0 to 1", "squash", o.Squash)
	starCheck(o.PointRounding >= 0 && o.PointRounding <= 1, "pointRounding must be from 0 to 1", "pointRounding", o.PointRounding)
	starCheck(sides >= 2, "sides must be at least 2", "sides", sides)
	return StarBorder{
		side: o.Side.normalize(), Points: sides, PointRounding: o.PointRounding, rotationRadians: o.Rotation * degToRad, Squash: o.Squash,
	}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b StarBorder) Side() BorderSide { return b.side.normalize() }

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b StarBorder) Dimensions() EdgeInsetsGeometry { return OutlinedBorderDimensions(b.Side()) }

// InnerRadiusRatio is the radius of the inner vertices of the star relative to
// the outer ones; for a polygon it is cos(pi / points).
//
// Flutter: painting/star_border.dart StarBorder.innerRadiusRatio
func (b StarBorder) InnerRadiusRatio() float64 {
	return b.innerRadiusRatio.OrElse(math.Cos(math.Pi / b.Points))
}

// Rotation is the rotation of the star in degrees.
//
// Flutter: painting/star_border.dart StarBorder.rotation
func (b StarBorder) Rotation() float64 { return b.rotationRadians * radToDeg }

// Scale scales the side by t.
//
// Flutter: painting/star_border.dart StarBorder.scale
func (b StarBorder) Scale(t float64) ShapeBorder {
	return NewStarBorder(StarBorderOpts{
		Points: foundation.Some(b.Points), Side: b.Side().Scale(t), Rotation: b.Rotation(), InnerRadiusRatio: foundation.Some(b.InnerRadiusRatio()),
		PointRounding: b.PointRounding, ValleyRounding: b.ValleyRounding, Squash: b.Squash,
	})
}

func twoPhaseLerp(t, split float64, first, second func(t float64) ShapeBorder) ShapeBorder {
	if t < split {
		return first(t * (1 / split))
	}
	return second((1 / (1.0 - split)) * (t - split))
}

// LerpFrom interpolates from a StarBorder, CircleBorder, StadiumBorder or
// RoundedRectangleBorder.
//
// Flutter: painting/star_border.dart StarBorder.lerpFrom
func (b StarBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if t == 0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	switch x := a.(type) {
	case StarBorder:
		return NewStarBorder(StarBorderOpts{
			Side: BorderSideLerp(x.Side(), b.Side(), t), Points: foundation.Some(ui.LerpDouble(x.Points, b.Points, t)),
			Rotation:         ui.LerpDouble(x.rotationRadians, b.rotationRadians, t) * radToDeg,
			InnerRadiusRatio: foundation.Some(ui.LerpDouble(x.InnerRadiusRatio(), b.InnerRadiusRatio(), t)),
			PointRounding:    ui.LerpDouble(x.PointRounding, b.PointRounding, t), ValleyRounding: ui.LerpDouble(x.ValleyRounding, b.ValleyRounding, t),
			Squash: ui.LerpDouble(x.Squash, b.Squash, t),
		})
	case StadiumBorder:
		lerpedSide := BorderSideLerp(x.Side(), b.Side(), t)
		circle := CircleBorder{side: lerpedSide}
		return twoPhaseLerp(t, 0.5,
			func(t float64) ShapeBorder { return x.LerpTo(circle, t) },
			func(t float64) ShapeBorder { return b.LerpFrom(circle, t) })
	case RoundedRectangleBorder:
		lerpedSide := BorderSideLerp(x.Side(), b.Side(), t)
		circle := CircleBorder{side: lerpedSide}
		stadium := StadiumBorder{side: lerpedSide}
		return twoPhaseLerp(t, 1.0/3,
			func(t float64) ShapeBorder { return stadium.LerpFrom(x, t) },
			func(t float64) ShapeBorder {
				return twoPhaseLerp(t, 0.5,
					func(t float64) ShapeBorder { return stadium.LerpTo(circle, t) },
					func(t float64) ShapeBorder { return b.LerpFrom(circle, t) })
			})
	}
	if c, ok := asCircleBorder(a); ok {
		if b.Points >= 2.5 {
			lerped := ui.LerpDouble(math.Round(b.Points), b.Points, t)
			return NewStarBorder(StarBorderOpts{
				Side: BorderSideLerp(c.Side(), b.Side(), t), Points: foundation.Some(lerped), Squash: ui.LerpDouble(c.Eccentricity, b.Squash, t),
				Rotation: b.Rotation(), InnerRadiusRatio: foundation.Some(ui.LerpDouble(math.Cos(math.Pi/lerped), b.InnerRadiusRatio(), t)),
				PointRounding: ui.LerpDouble(1.0, b.PointRounding, t), ValleyRounding: ui.LerpDouble(0.0, b.ValleyRounding, t),
			})
		}
		lerped := ui.LerpDouble(b.Points, 2, t)
		return NewStarBorder(StarBorderOpts{
			Side: BorderSideLerp(c.Side(), b.Side(), t), Points: foundation.Some(lerped), Squash: ui.LerpDouble(c.Eccentricity, b.Squash, t),
			Rotation: b.Rotation(), InnerRadiusRatio: foundation.Some(ui.LerpDouble(1, b.InnerRadiusRatio(), t)),
			PointRounding: ui.LerpDouble(0.5, b.PointRounding, t), ValleyRounding: ui.LerpDouble(0.5, b.ValleyRounding, t),
		})
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a StarBorder, CircleBorder, StadiumBorder or
// RoundedRectangleBorder.
//
// Flutter: painting/star_border.dart StarBorder.lerpTo
func (b StarBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if t == 0 {
		return b
	}
	if t == 1.0 {
		return o
	}
	switch x := o.(type) {
	case StarBorder:
		return NewStarBorder(StarBorderOpts{
			Side: BorderSideLerp(b.Side(), x.Side(), t), Points: foundation.Some(ui.LerpDouble(b.Points, x.Points, t)),
			Rotation:         ui.LerpDouble(b.rotationRadians, x.rotationRadians, t) * radToDeg,
			InnerRadiusRatio: foundation.Some(ui.LerpDouble(b.InnerRadiusRatio(), x.InnerRadiusRatio(), t)),
			PointRounding:    ui.LerpDouble(b.PointRounding, x.PointRounding, t), ValleyRounding: ui.LerpDouble(b.ValleyRounding, x.ValleyRounding, t),
			Squash: ui.LerpDouble(b.Squash, x.Squash, t),
		})
	case StadiumBorder:
		lerpedSide := BorderSideLerp(b.Side(), x.Side(), t)
		circle := CircleBorder{side: lerpedSide}
		return twoPhaseLerp(t, 0.5,
			func(t float64) ShapeBorder { return b.LerpTo(circle, t) },
			func(t float64) ShapeBorder { return x.LerpFrom(circle, t) })
	case RoundedRectangleBorder:
		lerpedSide := BorderSideLerp(b.Side(), x.Side(), t)
		circle := CircleBorder{side: lerpedSide}
		stadium := StadiumBorder{side: lerpedSide}
		return twoPhaseLerp(t, 2.0/3,
			func(t float64) ShapeBorder {
				return twoPhaseLerp(t, 0.5,
					func(t float64) ShapeBorder { return b.LerpTo(circle, t) },
					func(t float64) ShapeBorder { return stadium.LerpFrom(circle, t) })
			},
			func(t float64) ShapeBorder { return stadium.LerpTo(x, t) })
	}
	if c, ok := asCircleBorder(o); ok {
		if b.Points >= 2.5 {
			lerped := ui.LerpDouble(b.Points, math.Round(b.Points), t)
			return NewStarBorder(StarBorderOpts{
				Side: BorderSideLerp(b.Side(), c.Side(), t), Points: foundation.Some(lerped), Squash: ui.LerpDouble(b.Squash, c.Eccentricity, t),
				Rotation: b.Rotation(), InnerRadiusRatio: foundation.Some(ui.LerpDouble(b.InnerRadiusRatio(), math.Cos(math.Pi/lerped), t)),
				PointRounding: ui.LerpDouble(b.PointRounding, 1.0, t), ValleyRounding: ui.LerpDouble(b.ValleyRounding, 0.0, t),
			})
		}
		lerped := ui.LerpDouble(b.Points, 2, t)
		return NewStarBorder(StarBorderOpts{
			Side: BorderSideLerp(b.Side(), c.Side(), t), Points: foundation.Some(lerped), Squash: ui.LerpDouble(b.Squash, c.Eccentricity, t),
			Rotation: b.Rotation(), InnerRadiusRatio: foundation.Some(ui.LerpDouble(b.InnerRadiusRatio(), 1, t)),
			PointRounding: ui.LerpDouble(b.PointRounding, 0.5, t), ValleyRounding: ui.LerpDouble(b.ValleyRounding, 0.5, t),
		})
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// StarBorderCopy holds the replacements for StarBorder.CopyWith.
//
// Flutter: painting/star_border.dart StarBorder.copyWith
type StarBorderCopy struct {
	Side             foundation.Opt[BorderSide]
	Points           foundation.Opt[float64]
	InnerRadiusRatio foundation.Opt[float64]
	PointRounding    foundation.Opt[float64]
	ValleyRounding   foundation.Opt[float64]
	Rotation         foundation.Opt[float64]
	Squash           foundation.Opt[float64]
}

// CopyWith returns a copy with the given fields replaced. The copy is a star,
// even if this is a polygon.
//
// Flutter: painting/star_border.dart StarBorder.copyWith
func (b StarBorder) CopyWith(c StarBorderCopy) StarBorder {
	return NewStarBorder(StarBorderOpts{
		Side: c.Side.OrElse(b.Side()), Points: foundation.Some(c.Points.OrElse(b.Points)), Rotation: c.Rotation.OrElse(b.Rotation()),
		InnerRadiusRatio: foundation.Some(c.InnerRadiusRatio.OrElse(b.InnerRadiusRatio())),
		PointRounding:    c.PointRounding.OrElse(b.PointRounding), ValleyRounding: c.ValleyRounding.OrElse(b.ValleyRounding),
		Squash: c.Squash.OrElse(b.Squash),
	})
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b StarBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(StarBorderCopy{Side: foundation.Some(side)})
}

func (b StarBorder) generator() starGenerator {
	return starGenerator{
		points: b.Points, rotation: b.rotationRadians, innerRadiusRatio: b.InnerRadiusRatio(),
		pointRounding: b.PointRounding, valleyRounding: b.ValleyRounding, squash: b.Squash,
	}
}

// GetInnerPath is the star, in the rect deflated by the stroke inset.
//
// Flutter: painting/star_border.dart StarBorder.getInnerPath
func (b StarBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	return b.generator().generate(rect.Deflate(b.Side().StrokeInset()))
}

// GetOuterPath is the star.
//
// Flutter: painting/star_border.dart StarBorder.getOuterPath
func (b StarBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	return b.generator().generate(rect)
}

// Paint strokes the star.
//
// Flutter: painting/star_border.dart StarBorder.paint
func (b StarBorder) Paint(canvas *ui.Canvas, rect ui.Rect, _ ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	canvas.DrawPath(b.generator().generate(rect.Inflate(side.StrokeOffset()/2)), side.ToPaint())
}

// String describes the border.
//
// Flutter: painting/star_border.dart StarBorder.toString
func (b StarBorder) String() string {
	return "StarBorder(" + b.Side().String() + ", points: " + dartDouble(b.Points) + ", innerRadiusRatio: " + dartDouble(b.InnerRadiusRatio()) + ")"
}

type starPointInfo struct {
	valley, point, valleyArc1, pointArc1, pointArc2, valleyArc2 ui.Offset
}

type starGenerator struct {
	points, innerRadiusRatio, pointRounding, valleyRounding, rotation, squash float64
}

func (g starGenerator) generate(rect ui.Rect) *ui.Path {
	radius := rect.ShortestSide() / 2
	center := rect.Center()
	const minInnerRadiusRatio = .002
	mapped := (g.innerRadiusRatio * (1.0 - minInnerRadiusRatio)) + minInnerRadiusRatio
	var points []starPointInfo
	maxDiameter := 2.0 * g.generatePoints(&points, center, radius, radius*mapped)
	path := ui.NewPath()
	g.drawPoints(path, points)
	scale := ui.Offset{Dx: rect.Width() / maxDiameter, Dy: rect.Height() / maxDiameter}
	if rect.ShortestSide() == rect.Width() {
		scale = ui.Offset{Dx: scale.Dx, Dy: g.squash*scale.Dy + (1-g.squash)*scale.Dx}
	} else {
		scale = ui.Offset{Dx: g.squash*scale.Dx + (1-g.squash)*scale.Dy, Dy: scale.Dy}
	}
	m := ui.NewMatrix4TranslationValues(float32(rect.Center().Dx), float32(rect.Center().Dy), 0)
	m.MultiplyBy(ui.NewMatrix4Diagonal3Values(float32(scale.Dx), float32(scale.Dy), 1))
	m.MultiplyBy(ui.NewMatrix4RotationZ(g.rotation))
	m.MultiplyBy(ui.NewMatrix4TranslationValues(float32(-rect.Center().Dx), float32(-rect.Center().Dy), 0))
	return path.Transform(m)
}

func (g starGenerator) generatePoints(pointList *[]starPointInfo, center ui.Offset, radius, innerRadius float64) float64 {
	step := math.Pi / g.points
	angle := -math.Pi/2 - step
	valley := ui.Offset{Dx: center.Dx + math.Cos(angle)*innerRadius, Dy: center.Dy + math.Sin(angle)*innerRadius}

	getCurveMidpoint := func(a, b, c, a1, c1 ui.Offset) ui.Offset {
		ang := starAngle(a, b, c)
		w := starWeight(ang) / 2
		return a1.Div(4).Add(b.Mul(w)).Add(c1.Div(4)).Div(0.5 + w)
	}
	addPoint := func(pointAngle, pointStep, pointRadius, pointInnerRadius float64) float64 {
		pointAngle += pointStep
		point := ui.Offset{Dx: center.Dx + math.Cos(pointAngle)*pointRadius, Dy: center.Dy + math.Sin(pointAngle)*pointRadius}
		pointAngle += pointStep
		nextValley := ui.Offset{Dx: center.Dx + math.Cos(pointAngle)*pointInnerRadius, Dy: center.Dy + math.Sin(pointAngle)*pointInnerRadius}
		*pointList = append(*pointList, starPointInfo{
			valley:     valley,
			point:      point,
			valleyArc1: valley.Add(point.Sub(valley).Mul(g.valleyRounding)),
			pointArc1:  point.Add(valley.Sub(point).Mul(g.pointRounding)),
			pointArc2:  point.Add(nextValley.Sub(point).Mul(g.pointRounding)),
			valleyArc2: nextValley.Add(point.Sub(nextValley).Mul(g.valleyRounding)),
		})
		valley = nextValley
		return pointAngle
	}

	remainder := g.points - math.Trunc(g.points)
	hasIntegerSides := remainder < 1e-6
	wholeSides := g.points
	if !hasIntegerSides {
		wholeSides--
	}
	for i := 0; float64(i) < wholeSides; i++ {
		angle = addPoint(angle, step, radius, innerRadius)
	}
	list := *pointList
	thisPoint, nextPoint := list[0], list[1]
	pointMidpoint := getCurveMidpoint(thisPoint.valley, thisPoint.point, nextPoint.valley, thisPoint.pointArc1, thisPoint.pointArc2)
	valleyMidpoint := getCurveMidpoint(thisPoint.point, nextPoint.valley, nextPoint.point, thisPoint.valleyArc2, nextPoint.valleyArc1)
	valleyRadius := valleyMidpoint.Sub(center).Distance()
	pointRadius := pointMidpoint.Sub(center).Distance()
	if !hasIntegerSides {
		effectiveInnerRadius := math.Max(valleyRadius, innerRadius)
		endingRadius := effectiveInnerRadius + remainder*(radius-effectiveInnerRadius)
		addPoint(angle, step*remainder, endingRadius, innerRadius)
	}
	return ui.ClampDouble(math.Max(valleyRadius, pointRadius), math.SmallestNonzeroFloat64, math.MaxFloat64)
}

func (g starGenerator) drawPoints(path *ui.Path, points []starPointInfo) {
	start := points[0].pointArc1
	path.MoveTo(start.Dx, start.Dy)
	pointAngle := starAngle(points[0].valley, points[0].point, points[1].valley)
	pointWeight := starWeight(pointAngle)
	valleyAngle := starAngle(points[1].point, points[1].valley, points[0].point)
	valleyWeight := starWeight(valleyAngle)
	for i, point := range points {
		next := points[(i+1)%len(points)]
		path.LineTo(point.pointArc1.Dx, point.pointArc1.Dy)
		if pointAngle != 180 && pointAngle != 0 {
			path.ConicTo(point.point.Dx, point.point.Dy, point.pointArc2.Dx, point.pointArc2.Dy, pointWeight)
		} else {
			path.LineTo(point.pointArc2.Dx, point.pointArc2.Dy)
		}
		path.LineTo(point.valleyArc2.Dx, point.valleyArc2.Dy)
		if valleyAngle != 180 && valleyAngle != 0 {
			path.ConicTo(next.valley.Dx, next.valley.Dy, next.valleyArc1.Dx, next.valleyArc1.Dy, valleyWeight)
		} else {
			path.LineTo(next.valleyArc1.Dx, next.valleyArc1.Dy)
		}
	}
	path.Close()
}

func starWeight(angle float64) float64 { return math.Cos(dartMod(angle/2, math.Pi/2)) }

func starAngle(a, b, c ui.Offset) float64 {
	if a == c || b == c || b == a {
		return 0
	}
	u := a.Sub(b)
	v := c.Sub(b)
	dot := u.Dx*v.Dx + u.Dy*v.Dy
	m1 := math.Inf(1)
	if b.Dx != a.Dx {
		m1 = -u.Dy / -u.Dx
	}
	m2 := math.Inf(1)
	if b.Dx != c.Dx {
		m2 = -v.Dy / -v.Dx
	}
	angle := math.Abs(math.Atan2(m1-m2, 1+m1*m2))
	if dot < 0 {
		angle += math.Pi
	}
	return angle
}

var _ OutlinedBorder = StarBorder{}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagStarBorderInvalid = "StarBorderInvalid"
)
