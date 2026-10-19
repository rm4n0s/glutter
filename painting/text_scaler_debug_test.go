//go:build !glutter_release && !glutter_profile

package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestTextScalerDebugPanics(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"negative factor", "TextScalerInvalid", func() { NewTextScalerLinear(-1) }},
		{"negative font size", "TextScalerFontSizeInvalid", func() { NewTextScalerLinear(2).Scale(-1) }},
		{"infinite font size", "TextScalerFontSizeInvalid", func() { NewTextScalerLinear(2).Scale(math.Inf(1)) }},
		{"max below min", "TextScalerClampInvalid", func() {
			NewTextScalerLinear(2).Clamp(TextScalerClampOpts{MinScaleFactor: 2, MaxScaleFactor: foundation.Some(1.0)})
		}},
		{"negative min", "TextScalerClampInvalid", func() { NewTextScalerLinear(2).Clamp(TextScalerClampOpts{MinScaleFactor: -1}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}
