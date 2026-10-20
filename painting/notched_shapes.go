package painting

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// NotchedShape specifies a shape with a notch: a rectangle with a cut-out for
// another shape, such as a floating action button docked in a bar.
//
// Flutter: painting/notched_shapes.dart NotchedShape
type NotchedShape interface {
	// GetOuterPath creates a path of the host with a notch for the guest, which is
	// nil for none.
	//
	// Flutter: painting/notched_shapes.dart NotchedShape.getOuterPath
	GetOuterPath(host ui.Rect, guest foundation.Opt[ui.Rect]) *ui.Path
}

// CircularNotchedRectangle is a rectangle with a smooth circular notch.
//
// Flutter: painting/notched_shapes.dart CircularNotchedRectangle
type CircularNotchedRectangle struct {
	// Inverted makes the notch in the bottom edge instead of the top.
	Inverted bool
}

// GetOuterPath creates the rectangle's path with the notch for the guest.
//
// Flutter: painting/notched_shapes.dart CircularNotchedRectangle.getOuterPath
func (c CircularNotchedRectangle) GetOuterPath(host ui.Rect, guestOpt foundation.Opt[ui.Rect]) *ui.Path {
	guest, ok := guestOpt.Get()
	path := ui.NewPath()
	if !ok || !host.Overlaps(guest) {
		path.AddRect(host)
		return path
	}
	r := guest.Width() / 2.0
	invertMultiplier := 1.0
	if c.Inverted {
		invertMultiplier = -1.0
	}
	const s1, s2 = 15.0, 1.0
	a := -r - s2
	edge := host.Top
	if c.Inverted {
		edge = host.Bottom
	}
	b := edge - guest.Center().Dy
	n2 := math.Sqrt(b * b * r * r * (a*a + b*b - r*r))
	p2xA := ((a * r * r) - n2) / (a*a + b*b)
	p2xB := ((a * r * r) + n2) / (a*a + b*b)
	p2yA := math.Sqrt(r*r-p2xA*p2xA) * invertMultiplier
	p2yB := math.Sqrt(r*r-p2xB*p2xB) * invertMultiplier
	var p [6]ui.Offset
	p[0] = ui.Offset{Dx: a - s1, Dy: b}
	p[1] = ui.Offset{Dx: a, Dy: b}
	cmp := 1.0
	if b < 0 {
		cmp = -1.0
	}
	if cmp*p2yA > cmp*p2yB {
		p[2] = ui.Offset{Dx: p2xA, Dy: p2yA}
	} else {
		p[2] = ui.Offset{Dx: p2xB, Dy: p2yB}
	}
	p[3] = ui.Offset{Dx: -1.0 * p[2].Dx, Dy: p[2].Dy}
	p[4] = ui.Offset{Dx: -1.0 * p[1].Dx, Dy: p[1].Dy}
	p[5] = ui.Offset{Dx: -1.0 * p[0].Dx, Dy: p[0].Dy}
	for i := range p {
		p[i] = p[i].Add(guest.Center())
	}
	notch := ui.ArcToPointOpts{Radius: ui.NewRadiusCircular(r), Clockwise: ui.Ptr(false)}
	path.MoveTo(host.Left, host.Top)
	if !c.Inverted {
		path.LineTo(p[0].Dx, p[0].Dy)
		path.QuadraticBezierTo(p[1].Dx, p[1].Dy, p[2].Dx, p[2].Dy)
		path.ArcToPoint(p[3], notch)
		path.QuadraticBezierTo(p[4].Dx, p[4].Dy, p[5].Dx, p[5].Dy)
		path.LineTo(host.Right, host.Top)
		path.LineTo(host.Right, host.Bottom)
		path.LineTo(host.Left, host.Bottom)
	} else {
		path.LineTo(host.Right, host.Top)
		path.LineTo(host.Right, host.Bottom)
		path.LineTo(p[5].Dx, p[5].Dy)
		path.QuadraticBezierTo(p[4].Dx, p[4].Dy, p[3].Dx, p[3].Dy)
		path.ArcToPoint(p[2], notch)
		path.QuadraticBezierTo(p[1].Dx, p[1].Dy, p[0].Dx, p[0].Dy)
		path.LineTo(host.Left, host.Bottom)
	}
	path.Close()
	return path
}

// AutomaticNotchedShape is a NotchedShape defined by a ShapeBorder: the shape
// of the widget that uses the notched shape, typically a BottomAppBar.
//
// Deviation: Flutter's second ShapeBorder, the guest that is subtracted from the
// host to make the notch, is not ported because ui.Path has no boolean
// operations yet (SPEC §12.10); the guest rectangle is ignored, as it is in
// Flutter when the guest shape is null.
//
// Flutter: painting/notched_shapes.dart AutomaticNotchedShape
type AutomaticNotchedShape struct {
	// Host is the shape of the widget that uses the notched shape. It cannot
	// depend on the text direction, as no text direction is available.
	Host ShapeBorder
}

// GetOuterPath is the outer path of the host shape in the host rectangle.
//
// Flutter: painting/notched_shapes.dart AutomaticNotchedShape.getOuterPath
func (s AutomaticNotchedShape) GetOuterPath(host ui.Rect, _ foundation.Opt[ui.Rect]) *ui.Path {
	return s.Host.GetOuterPath(host, ui.TextDirectionUnspecified)
}
