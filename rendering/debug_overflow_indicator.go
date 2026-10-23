package rendering

import (
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

const (
	overflowIndicatorFraction = 0.1
	overflowFontSizePixels    = 7.5
	overflowLabelPadding      = 1.0
)

type overflowSide int

const (
	overflowLeft overflowSide = iota
	overflowTop
	overflowBottom
	overflowRight
)

type overflowRegion struct {
	rect        ui.Rect
	label       string
	labelOffset ui.Offset
	rotation    float64
	side        overflowSide
}

// DebugOverflowIndicator paints a yellow and black striped pattern on the edges
// of a render object whose children overflow it, and reports the overflow once.
// Embed it in the render objects that overflow.
//
// Deviation: the labels ("RIGHT OVERFLOWED BY 25 PIXELS") are not drawn, because
// a render object has no text engine; the stripes and the error report are
// (SPEC §12.10).
//
// Flutter: rendering/debug_overflow_indicator.dart DebugOverflowIndicatorMixin
type DebugOverflowIndicator struct {
	overflowReported bool
}

// ResetOverflowReport makes the next overflow be reported again; call it from
// Reassemble.
//
// Flutter: rendering/debug_overflow_indicator.dart DebugOverflowIndicatorMixin.reassemble
func (d *DebugOverflowIndicator) ResetOverflowReport() { d.overflowReported = false }

func formatPixels(value float64) string {
	switch {
	case value > 10:
		return strconv.FormatFloat(value, 'f', 0, 64)
	case value > 1:
		return strconv.FormatFloat(value, 'f', 1, 64)
	}
	return strconv.FormatFloat(value, 'g', 3, 64)
}

func calculateOverflowRegions(overflow RelativeRect, container ui.Rect) []overflowRegion {
	var regions []overflowRegion
	if overflow.Left > 0 {
		marker := ui.NewRectFromLTWH(0, 0, container.Width()*overflowIndicatorFraction, container.Height())
		regions = append(regions, overflowRegion{rect: marker, label: "LEFT OVERFLOWED BY " + formatPixels(overflow.Left) + " PIXELS",
			labelOffset: marker.CenterLeft().Add(ui.Offset{Dx: overflowFontSizePixels + overflowLabelPadding}),
			rotation:    math.Pi / 2, side: overflowLeft})
	}
	if overflow.Right > 0 {
		marker := ui.NewRectFromLTWH(container.Width()*(1-overflowIndicatorFraction), 0, container.Width()*overflowIndicatorFraction, container.Height())
		regions = append(regions, overflowRegion{rect: marker, label: "RIGHT OVERFLOWED BY " + formatPixels(overflow.Right) + " PIXELS",
			labelOffset: marker.CenterRight().Sub(ui.Offset{Dx: overflowFontSizePixels + overflowLabelPadding}),
			rotation:    -math.Pi / 2, side: overflowRight})
	}
	if overflow.Top > 0 {
		marker := ui.NewRectFromLTWH(0, 0, container.Width(), container.Height()*overflowIndicatorFraction)
		regions = append(regions, overflowRegion{rect: marker, label: "TOP OVERFLOWED BY " + formatPixels(overflow.Top) + " PIXELS",
			labelOffset: marker.TopCenter().Add(ui.Offset{Dy: overflowLabelPadding}), side: overflowTop})
	}
	if overflow.Bottom > 0 {
		marker := ui.NewRectFromLTWH(0, container.Height()*(1-overflowIndicatorFraction), container.Width(), container.Height()*overflowIndicatorFraction)
		regions = append(regions, overflowRegion{rect: marker, label: "BOTTOM OVERFLOWED BY " + formatPixels(overflow.Bottom) + " PIXELS",
			labelOffset: marker.BottomCenter().Sub(ui.Offset{Dy: overflowFontSizePixels + overflowLabelPadding}), side: overflowBottom})
	}
	return regions
}

// overflowDescription returns the text of the overflow error and its hints.
func overflowDescription(self RenderObject, overflow RelativeRect, hints []string) (text string, allHints []string) {
	if len(hints) == 0 {
		name := foundation.RuntimeTypeName(self)
		hints = []string{
			"The edge of the " + name + " that is overflowing has been marked in the rendering with a yellow and black " +
				"striped pattern. This is usually caused by the contents being too big for the " + name + ".",
			"This is considered an error condition because it indicates that there is content that cannot be seen. " +
				"If the content is legitimately bigger than the available space, consider clipping it with a ClipRect " +
				"widget before putting it in the " + name + ", or using a scrollable container, like a ListView.",
		}
	}
	var overflows []string
	if overflow.Left > 0 {
		overflows = append(overflows, formatPixels(overflow.Left)+" pixels on the left")
	}
	if overflow.Top > 0 {
		overflows = append(overflows, formatPixels(overflow.Top)+" pixels on the top")
	}
	if overflow.Bottom > 0 {
		overflows = append(overflows, formatPixels(overflow.Bottom)+" pixels on the bottom")
	}
	if overflow.Right > 0 {
		overflows = append(overflows, formatPixels(overflow.Right)+" pixels on the right")
	}
	switch len(overflows) {
	case 0:
		text = "nothing"
	case 1:
		text = overflows[0]
	case 2:
		text = overflows[0] + " and " + overflows[1]
	default:
		overflows[len(overflows)-1] = "and " + overflows[len(overflows)-1]
		text = strings.Join(overflows, ", ")
	}
	return text, hints
}

// PaintOverflowIndicator paints the striped pattern on the edges of the
// container that the child overflows, and reports the overflow once. The rects
// are in the coordinates of the render object.
//
// Flutter: rendering/debug_overflow_indicator.dart DebugOverflowIndicatorMixin.paintOverflowIndicator
func (d *DebugOverflowIndicator) PaintOverflowIndicator(self RenderObject, context *PaintingContext, offset ui.Offset,
	containerRect, childRect ui.Rect, overflowHints ...string) {
	overflow := NewRelativeRectFromRect(containerRect, childRect)
	if overflow.Left <= 0 && overflow.Right <= 0 && overflow.Top <= 0 && overflow.Bottom <= 0 {
		return
	}
	black, yellow := ui.NewColor(0xBF000000), ui.NewColor(0xBFFFFF00)
	paint := ui.NewPaint()
	paint.SetShader(ui.NewGradientLinear(ui.OffsetZero, ui.Offset{Dx: 10, Dy: 10}, []ui.Color{black, yellow, yellow, black},
		ui.GradientLinearOpts{ColorStops: []float64{0.25, 0.25, 0.75, 0.75}, TileMode: ui.TileModeRepeated}))
	for _, region := range calculateOverflowRegions(overflow, containerRect) {
		context.Canvas().DrawRect(region.rect.Shift(offset), paint)
	}
	if !d.overflowReported {
		d.overflowReported = true
		text, hints := overflowDescription(self, overflow, overflowHints)
		err := errors.New(tagRenderOverflow, "A "+foundation.RuntimeTypeName(self)+" overflowed by "+text+".",
			"library", "rendering library", "context", "during layout", "hints", strings.Join(hints, "\n"),
			"object", foundation.DescribeIdentity(self))
		o := self.DebugOptions()
		if o.ReportError != nil {
			o.ReportError(err)
		} else {
			o.DebugPrint(err.Error())
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRenderOverflow = "RenderOverflow"
)
