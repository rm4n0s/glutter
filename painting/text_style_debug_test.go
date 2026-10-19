//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestTextStyleDebugPanics(t *testing.T) {
	inheritA := NewTextStyle(TextStyleOpts{Inherit: foundation.Some(true)})
	inheritB := NewTextStyle(TextStyleOpts{Inherit: foundation.Some(false), FontSize: f64(3)})
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"color and foreground", "TextStyleColorAndForeground", func() {
			NewTextStyle(TextStyleOpts{Color: col(0xFF000000), Foreground: ui.NewPaint()})
		}},
		{"copy: color and foreground", "TextStyleColorAndForeground", func() {
			NewTextStyle().CopyWith(TextStyleCopy{Color: col(0xFF000000), Foreground: ui.NewPaint()})
		}},
		{"background and backgroundColor", "TextStyleBackgroundAndBackgroundColor", func() {
			NewTextStyle(TextStyleOpts{BackgroundColor: col(0xFF000000), Background: ui.NewPaint()})
		}},
		{"apply a factor to a null size", "TextStyleApplyNullField", func() {
			NewTextStyle().Apply(TextStyleApply{FontSizeFactor: f64(2)})
		}},
		{"apply a weight delta to a null weight", "TextStyleApplyNullField", func() {
			NewTextStyle().Apply(TextStyleApply{FontWeightDelta: 1})
		}},
		{"lerp with different inherit", "TextStyleLerpInheritMismatch", func() { TextStyleLerp(inheritA, inheritB, 0.5) }},
		{"zero max lines", "ParagraphMaxLinesInvalid", func() {
			NewTextStyle().GetParagraphStyle(GetParagraphStyleOpts{MaxLines: foundation.Some(0)})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}

func TestTextStyleDebugLabels(t *testing.T) {
	a := NewTextStyle(TextStyleOpts{DebugLabel: "a"})
	b := NewTextStyle(TextStyleOpts{DebugLabel: "b"})
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"copyWith keeps a label", a.CopyWith(TextStyleCopy{FontSize: f64(2)}).DebugLabel, "(a).copyWith"},
		{"copyWith sets a label", a.CopyWith(TextStyleCopy{DebugLabel: "z"}).DebugLabel, "z"},
		{"apply", a.Apply(TextStyleApply{}).DebugLabel, "(a).apply"},
		{"merge", a.Merge(b).DebugLabel, "(a).merge(b)"},
		{"merge with no labels", NewTextStyle().Merge(b).DebugLabel, "(unknown).merge(b)"},
		{"lerp", TextStyleLerp(a, b, 0.5).DebugLabel, "lerp(a ⎯0.5→ b)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: label = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}
