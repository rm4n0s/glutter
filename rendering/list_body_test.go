package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// listBodyRoot wraps the list body in an overflow box that gives it unbounded
// space along its main axis.
func listBodyRoot(body *rendering.RenderListBody, axis painting.Axis, crossExtent float64) rendering.Box {
	opts := rendering.RenderConstrainedOverflowBoxOpts{Alignment: painting.AlignmentTopLeft}
	if axis == painting.AxisHorizontal {
		opts.MinWidth, opts.MaxWidth = some(0), some(inf)
		opts.MinHeight, opts.MaxHeight = some(crossExtent), some(crossExtent)
	} else {
		opts.MinHeight, opts.MaxHeight = some(0), some(inf)
		opts.MinWidth, opts.MaxWidth = some(crossExtent), some(crossExtent)
	}
	return rendering.NewRenderConstrainedOverflowBox(body, opts)
}

func TestListBodyLayout(t *testing.T) {
	tests := []struct {
		name    string
		axisDir painting.AxisDirection
		want    [3]ui.Offset
		size    ui.Size
	}{
		{"down", painting.AxisDirectionDown, [3]ui.Offset{off(0, 0), off(0, 10), off(0, 30)}, ui.Size{Width: 50, Height: 60}},
		{"unspecified is down", painting.AxisDirectionUnspecified, [3]ui.Offset{off(0, 0), off(0, 10), off(0, 30)}, ui.Size{Width: 50, Height: 60}},
		{"up", painting.AxisDirectionUp, [3]ui.Offset{off(0, 50), off(0, 30), off(0, 0)}, ui.Size{Width: 50, Height: 60}},
		{"right", painting.AxisDirectionRight, [3]ui.Offset{off(0, 0), off(10, 0), off(30, 0)}, ui.Size{Width: 60, Height: 50}},
		{"left", painting.AxisDirectionLeft, [3]ui.Offset{off(50, 0), off(30, 0), off(0, 0)}, ui.Size{Width: 60, Height: 50}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			horizontal := tc.axisDir == painting.AxisDirectionRight || tc.axisDir == painting.AxisDirectionLeft
			mk := func(extent float64) rendering.Box {
				// Boxes of the given main extent that fill the cross axis.
				b := newTBox(extent, extent)
				b.pref = ui.Size{Width: extent, Height: extent}
				return b
			}
			boxes := []rendering.Box{mk(10), mk(20), mk(30)}
			body := rendering.NewRenderListBody(boxes, rendering.RenderListBodyOpts{AxisDirection: tc.axisDir})
			axis := body.MainAxis()
			layout(t, newHarness(t), listBodyRoot(body, axis, 50))
			if body.Size() != tc.size {
				t.Errorf("size = %v, want %v", body.Size(), tc.size)
			}
			for i, b := range boxes {
				if got := offsetOf(b); got != tc.want[i] {
					t.Errorf("box %d offset = %v, want %v", i, got, tc.want[i])
				}
				want := ui.Size{Width: 50, Height: float64(10 * (i + 1))}
				if horizontal {
					want = ui.Size{Width: float64(10 * (i + 1)), Height: 50}
				}
				if got := b.Size(); got != want {
					t.Errorf("box %d size = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestListBodyIntrinsicsAndBaseline(t *testing.T) {
	a, b := newBaselineGridBox(1, atTen), newBaselineGridBox(2, atBottom)
	vertical := rendering.NewRenderListBody([]rendering.Box{a, b})
	if got := vertical.GetMinIntrinsicWidth(inf); got != 10 {
		t.Errorf("vertical min intrinsic width = %v, want 10", got)
	}
	if got := vertical.GetMaxIntrinsicWidth(inf); got != 20 {
		t.Errorf("vertical max intrinsic width = %v, want 20", got)
	}
	// Flutter takes the maximum of the heights of a vertical list body, not their sum.
	if got := vertical.GetMinIntrinsicHeight(100); got != 10 {
		t.Errorf("vertical min intrinsic height = %v, want 10", got)
	}
	horizontal := rendering.NewRenderListBody([]rendering.Box{newBaselineGridBox(1, atTen), newBaselineGridBox(2, atBottom)}, rendering.RenderListBodyOpts{AxisDirection: painting.AxisDirectionRight})
	if got := horizontal.GetMaxIntrinsicWidth(inf); got != 30 {
		t.Errorf("horizontal max intrinsic width = %v, want 30", got)
	}
	if horizontal.MainAxis() != painting.AxisHorizontal {
		t.Errorf("main axis = %v", horizontal.MainAxis())
	}
	skipUnlessDebug(t)
	body := rendering.NewRenderListBody([]rendering.Box{newBaselineGridBox(1, atTen), newBaselineGridBox(1, atTen)})
	c := rendering.BoxConstraints{MinWidth: 50, MaxWidth: 50, MaxHeight: inf}
	if got := body.GetDryBaseline(c, ui.TextBaselineAlphabetic); got != foundation.Some(10.0) {
		t.Errorf("dry baseline = %v, want 10", got)
	}
	if got := body.GetDryLayout(c); got != (ui.Size{Width: 50, Height: 20}) {
		t.Errorf("dry layout = %v, want 50x20", got)
	}
}

func TestListBodyConstraintErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("bounded main axis", func(t *testing.T) {
		body := rendering.NewRenderListBody([]rendering.Box{newTBox(1, 1)})
		requireTag(t, "ListBodyBoundedMainAxis", func() {
			layout(t, newHarness(t), body, withConstraints(rendering.BoxConstraints{MaxWidth: 50, MaxHeight: 50}))
		})
	})
	t.Run("unbounded cross axis", func(t *testing.T) {
		body := rendering.NewRenderListBody([]rendering.Box{newTBox(1, 1)})
		parent := rendering.NewRenderConstrainedOverflowBox(body, rendering.RenderConstrainedOverflowBoxOpts{
			MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(inf)})
		requireTag(t, "ListBodyUnboundedCrossAxis", func() { layout(t, newHarness(t), parent) })
	})
}

func TestListBodySetAxisDirection(t *testing.T) {
	body := rendering.NewRenderListBody(nil)
	body.SetAxisDirection(painting.AxisDirectionLeft)
	if body.AxisDirection() != painting.AxisDirectionLeft {
		t.Errorf("axis direction = %v", body.AxisDirection())
	}
	body.SetAxisDirection(painting.AxisDirectionUnspecified)
	if body.AxisDirection() != painting.AxisDirectionDown {
		t.Errorf("axis direction = %v, want down", body.AxisDirection())
	}
}
