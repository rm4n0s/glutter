package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// maybeTextDirection is the direction of the closest Directionality above the
// context, or TextDirectionUnspecified. The context depends on it, so the
// widget updates its render object when the direction changes.
func maybeTextDirection(bc BuildContext) ui.TextDirection {
	d, _ := MaybeDirectionalityOf(bc)
	return d
}

// assertFactor checks, in debug builds, that a size factor is not negative
// (Flutter: assert(widthFactor == null || widthFactor >= 0.0)).
func assertFactor(widget, name string, v foundation.Opt[float64]) {
	if f, ok := v.Get(); ok && !(f >= 0) {
		panic(errors.New("FactorNegative", "the "+name+" of a "+widget+" must not be negative", "widget", widget, "factor", name, "value", f))
	}
}

// assertNonNegative checks, in debug builds, that a value is not negative.
func assertNonNegative(widget, name string, v float64) {
	if !(v >= 0) {
		panic(errors.New("ValueNegative", "the "+name+" of a "+widget+" must not be negative", "widget", widget, "name", name, "value", v))
	}
}

// assertNonNegativeOpt is assertNonNegative for a value that may be unset.
func assertNonNegativeOpt(widget, name string, v foundation.Opt[float64]) {
	if f, ok := v.Get(); ok {
		assertNonNegative(widget, name, f)
	}
}

// AxisDirectionFromAxisReverseAndDirectionality is the axis direction for a
// main axis, whether it is reversed and the ambient text direction (the
// direction of the Directionality for a horizontal axis).
//
// Flutter: widgets/basic.dart getAxisDirectionFromAxisReverseAndDirectionality
func AxisDirectionFromAxisReverseAndDirectionality(bc BuildContext, axis painting.Axis, reverse bool) painting.AxisDirection {
	if axis == painting.AxisHorizontal {
		d := painting.TextDirectionToAxisDirection(DirectionalityOf(bc))
		if reverse {
			return painting.FlipAxisDirection(d)
		}
		return d
	}
	if reverse {
		return painting.AxisDirectionUp
	}
	return painting.AxisDirectionDown
}
