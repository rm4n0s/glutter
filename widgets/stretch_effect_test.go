package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// StretchEffect has no test of its own in Flutter's repository (the tests of
// overscroll_stretch_indicator_test.dart go through StretchingOverscrollIndicator).
// These check the transform of the fallback of stretch_effect.dart, which is the
// only path of Glutter (SPEC §2: no shader image filters).

func TestStretchEffectTransform(t *testing.T) {
	tests := []struct {
		name            string
		axis            painting.Axis
		direction       ui.TextDirection
		strength        float64
		wantScaleX      float32
		wantScaleY      float32
		wantAlignment   painting.AlignmentGeometry
		wantFilterQuali bool
	}{
		{"vertical forward", painting.AxisVertical, ui.TextDirectionLTR, 0.5, 1, 1.5, painting.AlignmentDirectionalTopCenter, true},
		{"vertical backward", painting.AxisVertical, ui.TextDirectionLTR, -0.25, 1, 1.25, painting.AlignmentDirectionalBottomCenter, true},
		{"vertical at rest", painting.AxisVertical, ui.TextDirectionLTR, 0, 1, 1, painting.AlignmentDirectionalBottomCenter, false},
		{"horizontal forward", painting.AxisHorizontal, ui.TextDirectionLTR, 1, 2, 1, painting.AlignmentDirectionalCenterStart, true},
		{"horizontal backward", painting.AxisHorizontal, ui.TextDirectionLTR, -0.5, 1.5, 1, painting.AlignmentDirectionalCenterEnd, true},
		{"horizontal forward rtl", painting.AxisHorizontal, ui.TextDirectionRTL, 0.5, 1.5, 1, painting.AlignmentDirectionalCenterEnd, true},
		{"horizontal backward rtl", painting.AxisHorizontal, ui.TextDirectionRTL, -0.5, 1.5, 1, painting.AlignmentDirectionalCenterStart, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(tt.direction,
				NewStretchEffect(tt.axis, NewSizedBox(nil), StretchEffectOpts{StretchStrength: tt.strength})))
			transform := ts.Widget(gluttertest.ByType[*Transform]()).(*Transform)
			if got := transform.Transform(); got.Entry(0, 0) != tt.wantScaleX || got.Entry(1, 1) != tt.wantScaleY || got.Entry(2, 2) != 1 {
				t.Errorf("transform scales (%v, %v, %v), want (%v, %v, 1)", got.Entry(0, 0), got.Entry(1, 1), got.Entry(2, 2), tt.wantScaleX, tt.wantScaleY)
			}
			if got := transform.Alignment(); got != tt.wantAlignment {
				t.Errorf("alignment = %v, want %v", got, tt.wantAlignment)
			}
			if got := transform.FilterQuality(); got.IsSome() != tt.wantFilterQuali || (got.IsSome() && got.Value() != ui.FilterQualityMedium) {
				t.Errorf("filter quality = %v, want medium %v", got, tt.wantFilterQuali)
			}
			if got := ts.Widget(gluttertest.ByType[*StretchEffect]()).(*StretchEffect).StretchStrength(); got != tt.strength {
				t.Errorf("stretchStrength = %v, want %v", got, tt.strength)
			}
		})
	}
}

func TestStretchEffectStrengthMustBeInRange(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		strength float64
		panics   bool
	}{
		{-1.0, false}, {0.0, false}, {1.0, false}, {1.0001, true}, {-2, true},
	}
	for _, tt := range tests {
		v := panicOf(func() { NewStretchEffect(painting.AxisVertical, nil, StretchEffectOpts{StretchStrength: tt.strength}) })
		if (v != nil) != tt.panics {
			t.Errorf("strength %v: panic %v, want panic %v", tt.strength, v, tt.panics)
			continue
		}
		if tt.panics {
			err, _ := v.(error)
			appErr := requireErrorTag(t, err, "StretchEffectStrengthInvalid")
			if !appErr.HasRoute("NewStretchEffect.StretchEffectStrengthInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		}
	}
}
