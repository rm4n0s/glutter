package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// StretchEffectOpts are the optional parameters of NewStretchEffect.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.new
type StretchEffectOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// StretchStrength is the overscroll strength applied for the stretching
	// effect, between -1.0 and 1.0 inclusive; the default is 0. For the
	// horizontal axis, positive values apply a pull/stretch from left to right,
	// where 1.0 is the maximum stretch to the right, and negative values from
	// right to left. For the vertical axis, positive values apply a pull/stretch
	// from top to bottom, where 1.0 is the maximum stretch downward, and
	// negative values from bottom to top.
	StretchStrength float64
}

// StretchEffect is a widget that applies a stretching visual effect to its
// child, when the user overscrolls horizontally or vertically.
//
// Flutter replicates the native Android stretch overscroll effect with a
// fragment shader when the engine supports shader image filters, and
// approximates it with a matrix transform otherwise. StretchingOverscrollIndicator
// uses it.
//
// Deviation: ui has no shader image filters (custom fragment shaders are a
// non-goal, SPEC §2), so the effect is always the matrix transform of Flutter's
// fallback (SPEC §12.10).
//
// Flutter: widgets/stretch_effect.dart StretchEffect
type StretchEffect struct {
	StatelessWidget
	axis            painting.Axis
	child           Widget
	stretchStrength float64
}

// NewStretchEffect creates a widget that applies a stretch effect along the
// axis. In debug builds it panics with the tag StretchEffectStrengthInvalid when
// the strength is not between -1.0 and 1.0.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.new
func NewStretchEffect(axis painting.Axis, child Widget, opts ...StretchEffectOpts) *StretchEffect {
	o := firstOpts(opts)
	if foundation.DebugMode && !(o.StretchStrength >= -1.0 && o.StretchStrength <= 1.0) {
		panic(errors.New(tagStretchEffectStrengthInvalid, "stretchStrength must be between -1.0 and 1.0",
			"stretchStrength", o.StretchStrength))
	}
	return &StretchEffect{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), axis: axis, child: child, stretchStrength: o.StretchStrength}
}

// StretchStrength is the overscroll strength applied for the stretching effect,
// between -1.0 and 1.0.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.stretchStrength
func (w *StretchEffect) StretchStrength() float64 { return w.stretchStrength }

// Axis is the axis along which the stretching overscroll effect is applied.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.axis
func (w *StretchEffect) Axis() painting.Axis { return w.axis }

// Child is the child widget that the stretching overscroll effect applies to.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.child
func (w *StretchEffect) Child() Widget { return w.child }

func (w *StretchEffect) alignment(direction ui.TextDirection) painting.AlignmentGeometry {
	isForward := w.stretchStrength > 0

	if w.axis == painting.AxisVertical {
		if isForward {
			return painting.AlignmentDirectionalTopCenter
		}
		return painting.AlignmentDirectionalBottomCenter
	}

	// RTL horizontal.
	if direction == ui.TextDirectionRTL {
		if isForward {
			return painting.AlignmentDirectionalCenterEnd
		}
		return painting.AlignmentDirectionalCenterStart
	}
	if isForward {
		return painting.AlignmentDirectionalCenterStart
	}
	return painting.AlignmentDirectionalCenterEnd
}

// Build scales the child along the axis around the edge it is pulled from.
//
// Flutter: widgets/stretch_effect.dart StretchEffect.build
func (w *StretchEffect) Build(bc BuildContext) Widget {
	textDirection := DirectionalityOf(bc)
	x, y := 1.0, 1.0

	stretch := w.stretchStrength
	if stretch < 0 {
		stretch = -stretch
	}
	switch w.axis {
	case painting.AxisHorizontal:
		x += stretch
	case painting.AxisVertical:
		y += stretch
	}

	filterQuality := foundation.Some(ui.FilterQualityMedium)
	if w.stretchStrength == 0 {
		filterQuality = foundation.None[ui.FilterQuality]()
	}
	return NewTransform(ui.NewMatrix4Diagonal3Values(float32(x), float32(y), 1.0), w.child, TransformOpts{
		Alignment:     w.alignment(textDirection),
		FilterQuality: filterQuality,
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagStretchEffectStrengthInvalid = "StretchEffectStrengthInvalid"
)
