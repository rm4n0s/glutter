package ui

import (
	"math"
	"testing"
)

// There is no Dart test for PlatformDispatcher.scaleFontSize in the pinned sources (the engine
// tests it with an embedder), so these cases follow its algorithm.
func TestScaleFontSize(t *testing.T) {
	// platform converts 1 -> 1.5, 2 -> 3.5 and so on: a curve that is not linear.
	curve := func(size float64) float64 { return size*size/10 + size*1.4 }
	tests := []struct {
		name   string
		factor float64
		hook   func(*[]float64) func(float64) float64
		size   float64
		want   float64
		calls  []float64
	}{
		{"no hook is linear", 2, nil, 14.5, 29, nil},
		{"factor 1 returns the size", 1, func(c *[]float64) func(float64) float64 {
			return func(s float64) float64 { *c = append(*c, s); return curve(s) }
		}, 14.5, 14.5, nil},
		{"whole size", 1.5, func(c *[]float64) func(float64) float64 {
			return func(s float64) float64 { *c = append(*c, s); return curve(s) }
		}, 10, curve(10), []float64{10}},
		{"fraction is interpolated", 1.5, func(c *[]float64) func(float64) float64 {
			return func(s float64) float64 { *c = append(*c, s); return curve(s) }
		}, 10.25, curve(10) + (curve(11)-curve(10))*0.25, []float64{10, 11}},
		{"a failing platform falls back to linear", 1.5, func(c *[]float64) func(float64) float64 {
			return func(s float64) float64 { *c = append(*c, s); return -1 }
		}, 10.5, 15.75, []float64{10, 11}},
		{"zero", 1.5, func(c *[]float64) func(float64) float64 {
			return func(s float64) float64 { *c = append(*c, s); return curve(s) }
		}, 0, 0, []float64{0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls []float64
			cfg := PlatformConfiguration{TextScaleFactor: tc.factor}
			if tc.hook != nil {
				cfg.ScaleFontSize = tc.hook(&calls)
			}
			d := NewPlatformDispatcher(nil)
			d.UpdatePlatformConfiguration(cfg)
			if got := d.ScaleFontSize(tc.size); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("ScaleFontSize(%v) = %v, want %v", tc.size, got, tc.want)
			}
			if len(calls) != len(tc.calls) {
				t.Fatalf("platform called with %v, want %v", calls, tc.calls)
			}
			for i := range calls {
				if calls[i] != tc.calls[i] {
					t.Errorf("platform called with %v, want %v", calls, tc.calls)
				}
			}
		})
	}
}

func TestScaleFontSizeMemoizesUntilTheConfigurationChanges(t *testing.T) {
	calls := 0
	cfg := PlatformConfiguration{TextScaleFactor: 2, ScaleFontSize: func(s float64) float64 { calls++; return s * 2 }}
	d := NewPlatformDispatcher(nil)
	changes := 0
	d.OnTextScaleFactorChanged = func() { changes++ }
	d.UpdatePlatformConfiguration(cfg)
	d.ScaleFontSize(12.5)
	d.ScaleFontSize(12.5)
	d.ScaleFontSize(12)
	if calls != 2 {
		t.Errorf("the platform converted %d sizes, want 2 (12 and 13, each once)", calls)
	}
	d.UpdatePlatformConfiguration(cfg)
	d.ScaleFontSize(12)
	if calls != 3 {
		t.Errorf("the platform converted %d sizes after a new configuration, want 3", calls)
	}
	if changes != 2 {
		t.Errorf("OnTextScaleFactorChanged called %d times, want 2 (a scaler function counts as a change)", changes)
	}
}

func TestAlwaysUse24HourFormat(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	changed := 0
	d.OnPlatformConfigurationChanged = func() { changed++ }
	if d.AlwaysUse24HourFormat() {
		t.Error("the default is the 12-hour format")
	}
	tests := []struct {
		name        string
		cfg         PlatformConfiguration
		want        bool
		wantChanged int
	}{
		{"on", PlatformConfiguration{AlwaysUse24HourFormat: true}, true, 1},
		{"unchanged", PlatformConfiguration{AlwaysUse24HourFormat: true}, true, 0},
		{"off", PlatformConfiguration{}, false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed = 0
			d.UpdatePlatformConfiguration(tc.cfg)
			if d.AlwaysUse24HourFormat() != tc.want || changed != tc.wantChanged {
				t.Errorf("got %v with %d change callbacks, want %v with %d", d.AlwaysUse24HourFormat(), changed, tc.want, tc.wantChanged)
			}
		})
	}
}
