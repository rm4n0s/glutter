package painting

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PositionDependentBoxOpts are the parameters of PositionDependentBox.
//
// Flutter: painting/geometry.dart positionDependentBox
type PositionDependentBoxOpts struct {
	// Size is the size of the container.
	Size ui.Size
	// ChildSize is the size of the box to position.
	ChildSize ui.Size
	// Target is the point the box should be near, in the container.
	Target ui.Offset
	// PreferBelow says whether the box goes below the target when it fits both
	// above and below.
	PreferBelow bool
	// VerticalOffset is the distance between the target and the box.
	VerticalOffset float64
	// Margin is the minimum distance to the edges of the container; the default
	// is 10.
	Margin foundation.Opt[float64]
}

// PositionDependentBox positions a box of ChildSize in a container of Size so it
// is near the target, such as a tooltip, and returns its top left.
//
// Flutter: painting/geometry.dart positionDependentBox
func PositionDependentBox(o PositionDependentBoxOpts) ui.Offset {
	margin := o.Margin.OrElse(10.0)
	fitsBelow := o.Target.Dy+o.VerticalOffset+o.ChildSize.Height <= o.Size.Height-margin
	fitsAbove := o.Target.Dy-o.VerticalOffset-o.ChildSize.Height >= margin
	below := o.PreferBelow
	if fitsAbove != fitsBelow {
		below = fitsBelow
	}
	var y float64
	if below {
		y = math.Min(o.Target.Dy+o.VerticalOffset, o.Size.Height-margin)
	} else {
		y = math.Max(o.Target.Dy-o.VerticalOffset-o.ChildSize.Height, margin)
	}
	flexibleSpace := o.Size.Width - o.ChildSize.Width
	var x float64
	if flexibleSpace <= 2*margin {
		x = flexibleSpace / 2.0
	} else {
		x = ui.ClampDouble(o.Target.Dx-o.ChildSize.Width/2, margin, flexibleSpace-margin)
	}
	return ui.Offset{Dx: x, Dy: y}
}
