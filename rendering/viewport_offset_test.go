package rendering_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

func TestScrollDirection(t *testing.T) {
	tests := []struct {
		d       rendering.ScrollDirection
		str     string
		flipped rendering.ScrollDirection
	}{
		{rendering.ScrollDirectionIdle, "ScrollDirection.idle", rendering.ScrollDirectionIdle},
		{rendering.ScrollDirectionForward, "ScrollDirection.forward", rendering.ScrollDirectionReverse},
		{rendering.ScrollDirectionReverse, "ScrollDirection.reverse", rendering.ScrollDirectionForward},
	}
	for _, tc := range tests {
		if got := tc.d.String(); got != tc.str {
			t.Errorf("%d.String() = %q, want %q", tc.d, got, tc.str)
		}
		if got := rendering.FlipScrollDirection(tc.d); got != tc.flipped {
			t.Errorf("FlipScrollDirection(%v) = %v, want %v", tc.d, got, tc.flipped)
		}
	}
}

func TestFixedViewportOffset(t *testing.T) {
	o := rendering.NewViewportOffsetFixed(42)
	if o.Pixels() != 42 || !o.HasPixels() {
		t.Errorf("pixels %v, has pixels %v", o.Pixels(), o.HasPixels())
	}
	if !o.ApplyViewportDimension(100) || !o.ApplyContentDimensions(0, 500) {
		t.Error("a fixed offset should accept any dimension")
	}
	o.JumpTo(7)
	if o.Pixels() != 42 {
		t.Errorf("JumpTo changed a fixed offset to %v", o.Pixels())
	}
	o.CorrectBy(8)
	if o.Pixels() != 50 {
		t.Errorf("pixels after a correction = %v, want 50", o.Pixels())
	}
	select {
	case <-rendering.ViewportOffsetMoveTo(o, 3).Done():
	default:
		t.Error("moving a fixed offset should complete at once")
	}
	if o.UserScrollDirection() != rendering.ScrollDirectionIdle || o.AllowImplicitScrolling() {
		t.Error("a fixed offset is idle and does not allow implicit scrolling")
	}
	if got := rendering.NewViewportOffsetZero().Pixels(); got != 0 {
		t.Errorf("zero offset = %v", got)
	}
	if got := rendering.ViewportOffsetString(rendering.NewViewportOffsetFixed(5)); got == "" || got[len(got)-len("(offset: 5.0)"):] != "(offset: 5.0)" {
		t.Errorf("ViewportOffsetString = %q", got)
	}
}

func TestViewportOffsetMoveTo(t *testing.T) {
	tests := []struct {
		name         string
		opts         rendering.ViewportOffsetMoveToOpts
		wantJump     bool
		wantAnimates bool
	}{
		{"without a duration", rendering.ViewportOffsetMoveToOpts{}, true, false},
		{"with a zero duration", rendering.ViewportOffsetMoveToOpts{Duration: foundation.Some(time.Duration(0))}, true, false},
		{"with a duration", rendering.ViewportOffsetMoveToOpts{Duration: foundation.Some(time.Second)}, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := newImplicitScrollOffset(0)
			rendering.ViewportOffsetMoveTo(o, 9, tc.opts)
			if (len(o.jumps) == 1) != tc.wantJump || (len(o.animates) == 1) != tc.wantAnimates {
				t.Errorf("jumps %v, animates %v", o.jumps, o.animates)
			}
		})
	}
}
