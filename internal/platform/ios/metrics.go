package ios

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the view's metrics (SPEC §30.4): the arithmetic that turns what UIKit reports, in points, into the
// ViewMetrics of the framework, in physical pixels. It has no OS tag, so a table of cases tests it on every host.

// physicalSize is the size in physical pixels of a size in points at the scale. The buffer that a view draws into has whole pixels,
// rounded to the nearest.
func physicalSize(points ui.Size, scale float64) ui.Size {
	if scale <= 0 {
		scale = 1
	}
	return ui.Size{Width: math.Round(points.Width * scale), Height: math.Round(points.Height * scale)}
}

// edgeInsets are insets in points, as UIKit's UIEdgeInsets give them.
type edgeInsets struct{ Top, Left, Bottom, Right float64 }

// viewPadding is the padding in physical pixels of insets in points at the scale: the safe area of a view (the notch, the Dynamic
// Island, the home indicator and the rounded corners) is the framework's ViewPadding. A negative inset is no inset.
func viewPadding(in edgeInsets, scale float64) ui.ViewPadding {
	if scale <= 0 {
		scale = 1
	}
	px := func(v float64) float64 { return math.Max(0, v) * scale }
	return ui.ViewPadding{Left: px(in.Left), Top: px(in.Top), Right: px(in.Right), Bottom: px(in.Bottom)}
}
