package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

func TestApplyGrowthDirectionToAxisDirection(t *testing.T) {
	directions := []painting.AxisDirection{painting.AxisDirectionUp, painting.AxisDirectionRight, painting.AxisDirectionDown, painting.AxisDirectionLeft}
	for _, d := range directions {
		if got := rendering.ApplyGrowthDirectionToAxisDirection(d, rendering.GrowthDirectionForward); got != d {
			t.Errorf("forward growth of %v = %v, want %v", d, got, d)
		}
	}
	reverse := []struct{ in, want painting.AxisDirection }{
		{painting.AxisDirectionUp, painting.AxisDirectionDown},
		{painting.AxisDirectionDown, painting.AxisDirectionUp},
		{painting.AxisDirectionLeft, painting.AxisDirectionRight},
		{painting.AxisDirectionRight, painting.AxisDirectionLeft},
	}
	for _, tc := range reverse {
		if got := rendering.ApplyGrowthDirectionToAxisDirection(tc.in, rendering.GrowthDirectionReverse); got != tc.want {
			t.Errorf("reverse growth of %v = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestApplyGrowthDirectionToScrollDirection(t *testing.T) {
	tests := []struct {
		scroll rendering.ScrollDirection
		growth rendering.GrowthDirection
		want   rendering.ScrollDirection
	}{
		{rendering.ScrollDirectionIdle, rendering.GrowthDirectionReverse, rendering.ScrollDirectionIdle},
		{rendering.ScrollDirectionForward, rendering.GrowthDirectionForward, rendering.ScrollDirectionForward},
		{rendering.ScrollDirectionForward, rendering.GrowthDirectionReverse, rendering.ScrollDirectionReverse},
		{rendering.ScrollDirectionReverse, rendering.GrowthDirectionReverse, rendering.ScrollDirectionForward},
		{rendering.ScrollDirectionReverse, rendering.GrowthDirectionForward, rendering.ScrollDirectionReverse},
	}
	for _, tc := range tests {
		if got := rendering.ApplyGrowthDirectionToScrollDirection(tc.scroll, tc.growth); got != tc.want {
			t.Errorf("ApplyGrowthDirectionToScrollDirection(%v, %v) = %v, want %v", tc.scroll, tc.growth, got, tc.want)
		}
	}
}

func TestSliverConstraintsCopy(t *testing.T) {
	original := rendering.SliverConstraints{
		AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward,
		UserScrollDirection: rendering.ScrollDirectionIdle, CrossAxisDirection: painting.AxisDirectionRight,
	}
	cp := original.CopyWith(rendering.SliverConstraintsCopy{})
	if original != cp {
		t.Errorf("copy = %v, want %v", cp, original)
	}
	if original.String() != cp.String() {
		t.Errorf("copy description = %q, want %q", cp.String(), original.String())
	}
	if original.NormalizedGrowthDirection() != rendering.GrowthDirectionForward {
		t.Errorf("normalizedGrowthDirection = %v, want forward", original.NormalizedGrowthDirection())
	}
}

func TestSliverConstraintsNormalizedGrowthDirection(t *testing.T) {
	a := rendering.SliverConstraints{
		AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward,
		UserScrollDirection: rendering.ScrollDirectionIdle, CrossAxisDirection: painting.AxisDirectionRight,
	}
	c := a.CopyWith(rendering.SliverConstraintsCopy{
		AxisDirection:          painting.AxisDirectionUp,
		GrowthDirection:        rendering.GrowthDirectionReverse,
		UserScrollDirection:    rendering.ScrollDirectionForward,
		ScrollOffset:           foundation.Some(10.0),
		Overlap:                foundation.Some(20.0),
		RemainingPaintExtent:   foundation.Some(30.0),
		CrossAxisExtent:        foundation.Some(40.0),
		ViewportMainAxisExtent: foundation.Some(30.0),
	})
	d := rendering.SliverConstraints{
		AxisDirection: painting.AxisDirectionUp, GrowthDirection: rendering.GrowthDirectionReverse,
		UserScrollDirection: rendering.ScrollDirectionForward, ScrollOffset: 10, Overlap: 20, RemainingPaintExtent: 30,
		CrossAxisExtent: 40, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 30,
	}
	if c != d {
		t.Errorf("copy = %v, want %v", c, d)
	}
	tests := []struct {
		name string
		c    rendering.SliverConstraints
		want rendering.GrowthDirection
	}{
		{"c", c, rendering.GrowthDirectionForward},
		{"d", d, rendering.GrowthDirectionForward},
		{"e", d.CopyWith(rendering.SliverConstraintsCopy{AxisDirection: painting.AxisDirectionRight}), rendering.GrowthDirectionReverse},
		{"f", d.CopyWith(rendering.SliverConstraintsCopy{AxisDirection: painting.AxisDirectionLeft}), rendering.GrowthDirectionForward},
		{"g", d.CopyWith(rendering.SliverConstraintsCopy{GrowthDirection: rendering.GrowthDirectionForward}), rendering.GrowthDirectionReverse},
	}
	for _, tc := range tests {
		if got := tc.c.NormalizedGrowthDirection(); got != tc.want {
			t.Errorf("%s: normalizedGrowthDirection = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestSliverGeometryValidity(t *testing.T) {
	skipUnlessDebug(t)
	if !rendering.SliverGeometryZero.DebugAssertIsValid() {
		t.Error("SliverGeometry with no arguments is not valid")
	}
	tests := []struct {
		name string
		g    rendering.SliverGeometry
	}{
		{"layoutExtent exceeds paintExtent", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{LayoutExtent: foundation.Some(10.0), PaintExtent: 9})},
		{"maxPaintExtent is less than paintExtent", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{PaintExtent: 9, MaxPaintExtent: 8})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requireTag(t, "SliverGeometryInvalid", func() { tc.g.DebugAssertIsValid() })
		})
	}
}
