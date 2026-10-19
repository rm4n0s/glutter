//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestOutlinedBordersDebugPanics(t *testing.T) {
	f := foundation.Some[float64]
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"circle eccentricity high", "BorderEccentricityOutOfRange", func() { NewCircleBorder(CircleBorderOpts{Eccentricity: 1.5}) }},
		{"circle eccentricity low", "BorderEccentricityOutOfRange", func() { NewCircleBorder(CircleBorderOpts{Eccentricity: -0.5}) }},
		{"oval eccentricity", "BorderEccentricityOutOfRange", func() { NewOvalBorder(OvalBorderOpts{Eccentricity: f(2)}) }},
		{"star points", "StarBorderInvalid", func() { NewStarBorder(StarBorderOpts{Points: f(1)}) }},
		{"star inner radius ratio", "StarBorderInvalid", func() { NewStarBorder(StarBorderOpts{InnerRadiusRatio: f(1.5)}) }},
		{"star squash", "StarBorderInvalid", func() { NewStarBorder(StarBorderOpts{Squash: 2}) }},
		{"star rounding sum", "StarBorderInvalid", func() { NewStarBorder(StarBorderOpts{PointRounding: 0.6, ValleyRounding: 0.6}) }},
		{"polygon sides", "StarBorderInvalid", func() { NewStarBorderPolygon(StarBorderPolygonOpts{Sides: f(1)}) }},
		{"polygon point rounding", "StarBorderInvalid", func() { NewStarBorderPolygon(StarBorderPolygonOpts{PointRounding: 2}) }},
		{"linear edge size", "LinearBorderEdgeSizeOutOfRange", func() { NewLinearBorderEdge(LinearBorderEdgeOpts{Size: f(1.5)}) }},
		{"two options values", "MultipleOptions", func() { NewStadiumBorder(StadiumBorderOpts{}, StadiumBorderOpts{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, tt.tag, tt.call)
		})
	}
}
