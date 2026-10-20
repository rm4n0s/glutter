package material

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Deviation: Dart's MaterialPointArcTween, MaterialRectArcTween and
// MaterialRectCenterArcTween extend Tween, whose begin and end are setters that
// mark the tween dirty. Go has no subclassing of Tween, so each is a struct with
// exported Begin and End fields that notices a change when it is used (the
// comparison replaces the dirty flag), and a Tween method that gives the
// *animation.Tween[Opt[Rect]] that a Hero's createRectTween returns.

const arcOnAxisDelta = 2.0

// MaterialPointArcTween interpolates between two points along a quarter circle,
// as the Material motion spec says for a movement that is not on an axis.
//
// Flutter: material/arc.dart MaterialPointArcTween
type MaterialPointArcTween struct {
	// Begin is the point at t = 0.
	//
	// Flutter: animation/tween.dart Tween.begin
	Begin ui.Offset
	// End is the point at t = 1.
	//
	// Flutter: animation/tween.dart Tween.end
	End ui.Offset

	initialized           bool
	initBegin, initEnd    ui.Offset
	center                ui.Offset
	radius                float64
	hasAngles             bool
	beginAngle, endAngle_ float64
}

// NewMaterialPointArcTween creates a tween between the points.
//
// Flutter: material/arc.dart MaterialPointArcTween.new
func NewMaterialPointArcTween(begin, end ui.Offset) *MaterialPointArcTween {
	return &MaterialPointArcTween{Begin: begin, End: end}
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return v
}

func (t *MaterialPointArcTween) initialize() {
	if t.initialized && t.initBegin == t.Begin && t.initEnd == t.End {
		return
	}
	begin, end := t.Begin, t.End
	delta := end.Sub(begin)
	deltaX := math.Abs(delta.Dx)
	deltaY := math.Abs(delta.Dy)
	distanceFromAtoB := delta.Distance()
	c := ui.Offset{Dx: end.Dx, Dy: begin.Dy}
	sweepAngle := func() float64 { return 2.0 * math.Asin(distanceFromAtoB/(2.0*t.radius)) }

	if deltaX > arcOnAxisDelta && deltaY > arcOnAxisDelta {
		if deltaX < deltaY {
			t.radius = distanceFromAtoB * distanceFromAtoB / c.Sub(begin).Distance() / 2.0
			t.center = ui.Offset{Dx: end.Dx + t.radius*sign(begin.Dx-end.Dx), Dy: end.Dy}
			if begin.Dx < end.Dx {
				t.beginAngle = sweepAngle() * sign(begin.Dy-end.Dy)
				t.endAngle_ = 0.0
			} else {
				t.beginAngle = math.Pi + sweepAngle()*sign(end.Dy-begin.Dy)
				t.endAngle_ = math.Pi
			}
		} else {
			t.radius = distanceFromAtoB * distanceFromAtoB / c.Sub(end).Distance() / 2.0
			t.center = ui.Offset{Dx: begin.Dx, Dy: begin.Dy + sign(end.Dy-begin.Dy)*t.radius}
			if begin.Dy < end.Dy {
				t.beginAngle = -math.Pi / 2.0
				t.endAngle_ = t.beginAngle + sweepAngle()*sign(end.Dx-begin.Dx)
			} else {
				t.beginAngle = math.Pi / 2.0
				t.endAngle_ = t.beginAngle + sweepAngle()*sign(begin.Dx-end.Dx)
			}
		}
		t.hasAngles = true
	} else {
		t.hasAngles = false
	}
	t.initialized, t.initBegin, t.initEnd = true, begin, end
}

// Center is the center of the circle of the arc, or none when the movement is on
// an axis.
//
// Flutter: material/arc.dart MaterialPointArcTween.center
func (t *MaterialPointArcTween) Center() foundation.Opt[ui.Offset] {
	t.initialize()
	if !t.hasAngles {
		return foundation.None[ui.Offset]()
	}
	return foundation.Some(t.center)
}

// Radius is the radius of the circle of the arc, or none when the movement is
// on an axis.
//
// Flutter: material/arc.dart MaterialPointArcTween.radius
func (t *MaterialPointArcTween) Radius() foundation.Opt[float64] {
	t.initialize()
	if !t.hasAngles {
		return foundation.None[float64]()
	}
	return foundation.Some(t.radius)
}

// BeginAngle is the angle of the beginning of the arc, or none when the
// movement is on an axis.
//
// Flutter: material/arc.dart MaterialPointArcTween.beginAngle
func (t *MaterialPointArcTween) BeginAngle() foundation.Opt[float64] {
	t.initialize()
	if !t.hasAngles {
		return foundation.None[float64]()
	}
	return foundation.Some(t.beginAngle)
}

// EndAngle is, as in Flutter, the angle of the beginning of the arc: its getter
// returns _beginAngle (it is used only for diagnostics).
//
// Flutter: material/arc.dart MaterialPointArcTween.endAngle
func (t *MaterialPointArcTween) EndAngle() foundation.Opt[float64] { return t.BeginAngle() }

// Lerp is the point at t.
//
// Flutter: material/arc.dart MaterialPointArcTween.lerp
func (t *MaterialPointArcTween) Lerp(at float64) ui.Offset {
	t.initialize()
	if at == 0.0 {
		return t.Begin
	}
	if at == 1.0 {
		return t.End
	}
	if !t.hasAngles {
		return t.Begin.Lerp(t.End, at)
	}
	angle := t.beginAngle + (t.endAngle_-t.beginAngle)*at
	x := math.Cos(angle) * t.radius
	y := math.Sin(angle) * t.radius
	return t.center.Add(ui.Offset{Dx: x, Dy: y})
}

// String describes the tween.
//
// Flutter: material/arc.dart MaterialPointArcTween.toString
func (t *MaterialPointArcTween) String() string {
	t.initialize()
	return fmt.Sprintf("MaterialPointArcTween(%v → %v; center=%v, radius=%v, beginAngle=%v, endAngle=%v)",
		t.Begin, t.End, t.Center(), t.Radius(), t.BeginAngle(), t.EndAngle())
}

type arcCornerID int

const (
	arcCornerTopLeft arcCornerID = iota
	arcCornerTopRight
	arcCornerBottomLeft
	arcCornerBottomRight
)

type arcDiagonal struct{ beginID, endID arcCornerID }

var arcAllDiagonals = []arcDiagonal{
	{arcCornerTopLeft, arcCornerBottomRight},
	{arcCornerBottomRight, arcCornerTopLeft},
	{arcCornerTopRight, arcCornerBottomLeft},
	{arcCornerBottomLeft, arcCornerTopRight},
}

func arcCornerFor(rect ui.Rect, id arcCornerID) ui.Offset {
	switch id {
	case arcCornerTopLeft:
		return rect.TopLeft()
	case arcCornerTopRight:
		return rect.TopRight()
	case arcCornerBottomLeft:
		return rect.BottomLeft()
	}
	return rect.BottomRight()
}

// rectTweenOf makes the *animation.Tween that a Hero's createRectTween returns
// from a function of the begin and end rectangles. A none rect counts as zero
// when the other is some, as RectTween does.
func rectTweenOf(begin, end foundation.Opt[ui.Rect], lerp func(begin, end ui.Rect, t float64) ui.Rect) *animation.Tween[foundation.Opt[ui.Rect]] {
	return animation.NewTweenFunc(begin, end, func(b, e foundation.Opt[ui.Rect], at float64) foundation.Opt[ui.Rect] {
		br, bok := b.Get()
		er, eok := e.Get()
		if bok && eok {
			return foundation.Some(lerp(br, er, at))
		}
		var pb, pe *ui.Rect
		if bok {
			pb = &br
		}
		if eok {
			pe = &er
		}
		if r := ui.RectLerp(pb, pe, at); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[ui.Rect]()
	})
}

// MaterialRectArcTween interpolates between two rectangles by moving the two
// corners of the diagonal that the rectangles move along on quarter circles.
//
// Flutter: material/arc.dart MaterialRectArcTween
type MaterialRectArcTween struct {
	// Begin is the rectangle at t = 0.
	//
	// Flutter: animation/tween.dart Tween.begin
	Begin ui.Rect
	// End is the rectangle at t = 1.
	//
	// Flutter: animation/tween.dart Tween.end
	End ui.Rect

	initialized        bool
	initBegin, initEnd ui.Rect
	beginArc, endArc   *MaterialPointArcTween
}

// NewMaterialRectArcTween creates a tween between the rectangles.
//
// Flutter: material/arc.dart MaterialRectArcTween.new
func NewMaterialRectArcTween(begin, end ui.Rect) *MaterialRectArcTween {
	return &MaterialRectArcTween{Begin: begin, End: end}
}

// MaterialRectArcTweenCreate is a widgets.CreateRectTween: the tween that
// MaterialApp's hero controller gives to the flights of heroes.
//
// Flutter: material/app.dart MaterialApp.createMaterialHeroController
func MaterialRectArcTweenCreate(begin, end foundation.Opt[ui.Rect]) *animation.Tween[foundation.Opt[ui.Rect]] {
	arc := &MaterialRectArcTween{}
	return rectTweenOf(begin, end, func(b, e ui.Rect, at float64) ui.Rect {
		arc.Begin, arc.End = b, e
		return arc.Lerp(at)
	})
}

func (t *MaterialRectArcTween) diagonalSupport(centersVector ui.Offset, d arcDiagonal) float64 {
	delta := arcCornerFor(t.Begin, d.endID).Sub(arcCornerFor(t.Begin, d.beginID))
	length := delta.Distance()
	return centersVector.Dx*delta.Dx/length + centersVector.Dy*delta.Dy/length
}

func (t *MaterialRectArcTween) initialize() {
	if t.initialized && t.initBegin == t.Begin && t.initEnd == t.End {
		return
	}
	centersVector := t.End.Center().Sub(t.Begin.Center())
	diagonal := arcAllDiagonals[0]
	var maxKey float64
	for i, d := range arcAllDiagonals {
		if key := t.diagonalSupport(centersVector, d); i == 0 || key > maxKey {
			diagonal, maxKey = d, key
		}
	}
	t.beginArc = NewMaterialPointArcTween(arcCornerFor(t.Begin, diagonal.beginID), arcCornerFor(t.End, diagonal.beginID))
	t.endArc = NewMaterialPointArcTween(arcCornerFor(t.Begin, diagonal.endID), arcCornerFor(t.End, diagonal.endID))
	t.initialized, t.initBegin, t.initEnd = true, t.Begin, t.End
}

// BeginArc is the arc that the first corner of the diagonal follows.
//
// Flutter: material/arc.dart MaterialRectArcTween.beginArc
func (t *MaterialRectArcTween) BeginArc() *MaterialPointArcTween {
	t.initialize()
	return t.beginArc
}

// EndArc is the arc that the last corner of the diagonal follows.
//
// Flutter: material/arc.dart MaterialRectArcTween.endArc
func (t *MaterialRectArcTween) EndArc() *MaterialPointArcTween {
	t.initialize()
	return t.endArc
}

// Lerp is the rectangle at t.
//
// Flutter: material/arc.dart MaterialRectArcTween.lerp
func (t *MaterialRectArcTween) Lerp(at float64) ui.Rect {
	t.initialize()
	if at == 0.0 {
		return t.Begin
	}
	if at == 1.0 {
		return t.End
	}
	return ui.NewRectFromPoints(t.beginArc.Lerp(at), t.endArc.Lerp(at))
}

// Tween is the tween of the rectangles as a *animation.Tween, which is what
// Hero's createRectTween returns.
//
// Flutter: material/arc.dart MaterialRectArcTween
func (t *MaterialRectArcTween) Tween() *animation.Tween[foundation.Opt[ui.Rect]] {
	return rectTweenOf(foundation.Some(t.Begin), foundation.Some(t.End), func(b, e ui.Rect, at float64) ui.Rect {
		t.Begin, t.End = b, e
		return t.Lerp(at)
	})
}

// String describes the tween.
//
// Flutter: material/arc.dart MaterialRectArcTween.toString
func (t *MaterialRectArcTween) String() string {
	t.initialize()
	return fmt.Sprintf("MaterialRectArcTween(%v → %v; beginArc=%v, endArc=%v)", t.Begin, t.End, t.beginArc, t.endArc)
}

// MaterialRectCenterArcTween interpolates between two rectangles by moving the
// center along a quarter circle and interpolating the size linearly.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween
type MaterialRectCenterArcTween struct {
	// Begin is the rectangle at t = 0.
	//
	// Flutter: animation/tween.dart Tween.begin
	Begin ui.Rect
	// End is the rectangle at t = 1.
	//
	// Flutter: animation/tween.dart Tween.end
	End ui.Rect

	initialized        bool
	initBegin, initEnd ui.Rect
	centerArc          *MaterialPointArcTween
}

// NewMaterialRectCenterArcTween creates a tween between the rectangles.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween.new
func NewMaterialRectCenterArcTween(begin, end ui.Rect) *MaterialRectCenterArcTween {
	return &MaterialRectCenterArcTween{Begin: begin, End: end}
}

func (t *MaterialRectCenterArcTween) initialize() {
	if t.initialized && t.initBegin == t.Begin && t.initEnd == t.End {
		return
	}
	t.centerArc = NewMaterialPointArcTween(t.Begin.Center(), t.End.Center())
	t.initialized, t.initBegin, t.initEnd = true, t.Begin, t.End
}

// CenterArc is the arc that the center follows.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween.centerArc
func (t *MaterialRectCenterArcTween) CenterArc() *MaterialPointArcTween {
	t.initialize()
	return t.centerArc
}

// Lerp is the rectangle at t.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween.lerp
func (t *MaterialRectCenterArcTween) Lerp(at float64) ui.Rect {
	t.initialize()
	if at == 0.0 {
		return t.Begin
	}
	if at == 1.0 {
		return t.End
	}
	center := t.centerArc.Lerp(at)
	width := t.Begin.Width() + (t.End.Width()-t.Begin.Width())*at
	height := t.Begin.Height() + (t.End.Height()-t.Begin.Height())*at
	return ui.NewRectFromLTWH(center.Dx-width/2.0, center.Dy-height/2.0, width, height)
}

// Tween is the tween of the rectangles as a *animation.Tween, which is what
// Hero's createRectTween returns.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween
func (t *MaterialRectCenterArcTween) Tween() *animation.Tween[foundation.Opt[ui.Rect]] {
	return rectTweenOf(foundation.Some(t.Begin), foundation.Some(t.End), func(b, e ui.Rect, at float64) ui.Rect {
		t.Begin, t.End = b, e
		return t.Lerp(at)
	})
}

// String describes the tween.
//
// Flutter: material/arc.dart MaterialRectCenterArcTween.toString
func (t *MaterialRectCenterArcTween) String() string {
	t.initialize()
	return fmt.Sprintf("MaterialRectCenterArcTween(%v → %v; centerArc=%v)", t.Begin, t.End, t.centerArc)
}
