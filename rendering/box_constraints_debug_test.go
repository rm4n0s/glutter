//go:build !glutter_release && !glutter_profile

package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/rendering"
)

func TestBoxConstraintsDebugAssertIsValid(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name string
		c    rendering.BoxConstraints
		opts rendering.DebugAssertIsValidOpts
		tag  string
	}{
		{"nan", rendering.BoxConstraints{MinWidth: nan, MaxWidth: 1, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNaN"},
		{"two nan", rendering.BoxConstraints{MinWidth: nan, MaxWidth: nan, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNaN"},
		{"three nan", rendering.BoxConstraints{MinWidth: nan, MaxWidth: nan, MinHeight: nan, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNaN"},
		{"negative width", rendering.BoxConstraints{MinWidth: -1, MaxWidth: 1, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNegative"},
		{"negative height", rendering.BoxConstraints{MaxWidth: 1, MinHeight: -1, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNegative"},
		{"both negative", rendering.BoxConstraints{MinWidth: -1, MaxWidth: 1, MinHeight: -1, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNegative"},
		{"not normalized width", rendering.BoxConstraints{MinWidth: 2, MaxWidth: 1, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNotNormalized"},
		{"not normalized height", rendering.BoxConstraints{MaxWidth: 1, MinHeight: 2, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNotNormalized"},
		{"not normalized both", rendering.BoxConstraints{MinWidth: 2, MaxWidth: 1, MinHeight: 2, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{}, "BoxConstraintsNotNormalized"},
		{"forced infinite width", rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MaxHeight: 1}, rendering.DebugAssertIsValidOpts{IsAppliedConstraint: true}, "BoxConstraintsInfinite"},
		{"forced infinite height", rendering.BoxConstraints{MaxWidth: 1, MinHeight: inf, MaxHeight: inf}, rendering.DebugAssertIsValidOpts{IsAppliedConstraint: true}, "BoxConstraintsInfinite"},
		{"forced infinite both", rendering.NewBoxConstraintsExpand(), rendering.DebugAssertIsValidOpts{IsAppliedConstraint: true}, "BoxConstraintsInfinite"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := panicOf(func() { tc.c.DebugAssertIsValid(tc.opts) })
			err, _ := v.(error)
			requireRoute(t, err, tc.tag)
		})
	}
	if v := panicOf(func() { rendering.NewBoxConstraintsExpand().DebugAssertIsValid() }); v != nil {
		t.Errorf("forced infinite constraints are valid unless applied, got panic %v", v)
	}
}

func TestBoxConstraintsDebugCollector(t *testing.T) {
	v := panicOf(func() {
		rendering.BoxConstraints{MinWidth: -1}.DebugAssertIsValid(rendering.DebugAssertIsValidOpts{
			InformationCollector: func() []string { return []string{"extra line"} },
		})
	})
	err, _ := v.(error)
	requireRoute(t, err, "BoxConstraintsNegative")
}

func TestBoxConstraintsLerpUnbounded(t *testing.T) {
	c1 := &rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MinHeight: inf, MaxHeight: inf}
	c2 := &rendering.BoxConstraints{MinWidth: 20, MaxWidth: 30, MinHeight: inf, MaxHeight: inf}
	c3 := &rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MinHeight: 20, MaxHeight: 30}
	for _, pair := range [][2]*rendering.BoxConstraints{{c1, c2}, {c1, c3}, {c2, c3}} {
		err, _ := panicOf(func() { rendering.BoxConstraintsLerp(pair[0], pair[1], 0.5) }).(error)
		requireRoute(t, err, "BoxConstraintsLerpUnbounded")
	}
}
