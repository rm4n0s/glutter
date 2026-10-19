//go:build !glutter_release && !glutter_profile

package painting

import "testing"

func TestStrutStyleDebugPanics(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"zero font size", "StrutStyleFontSizeInvalid", func() { NewStrutStyle(StrutStyleOpts{FontSize: f64(0)}) }},
		{"negative leading", "StrutStyleLeadingInvalid", func() { NewStrutStyle(StrutStyleOpts{Leading: f64(-1)}) }},
		{"package without font", "StrutStylePackageWithoutFont", func() { NewStrutStyle(StrutStyleOpts{Package: "p"}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}
