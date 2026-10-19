package dartfmt

import (
	"math"
	"testing"
	"time"
)

func TestToStringAsFixed(t *testing.T) {
	tests := []struct {
		in     float64
		digits int
		want   string
	}{
		{0, 1, "0.0"},
		{1.25, 1, "1.3"}, // an exact tie rounds away from zero
		{-1.25, 1, "-1.3"},
		{0.35, 1, "0.3"}, // 0.35 is a little below the tie
		{1.0 / 3, 1, "0.3"},
		{42, 1, "42.0"},
		{2.5, 0, "3"},
		{9.95, 1, "9.9"},
		{0.5, 0, "1"},
		{99.95, 1, "100.0"},
		{math.NaN(), 1, "NaN"},
		{math.Inf(1), 1, "Infinity"},
		{math.Inf(-1), 1, "-Infinity"},
	}
	for _, tt := range tests {
		if got := ToStringAsFixed(tt.in, tt.digits); got != tt.want {
			t.Errorf("ToStringAsFixed(%v, %d) = %q, want %q", tt.in, tt.digits, got, tt.want)
		}
	}
}

func TestDouble(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{1, "1.0"},
		{-2.5, "-2.5"},
		{0.1, "0.1"},
		{1e-5, "1e-5"},
		{1e21, "1e+21"},
		{123456789.5, "123456789.5"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, tt := range tests {
		if got := Double(tt.in); got != tt.want {
			t.Errorf("Double(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00:00.000000"},
		{1500 * time.Millisecond, "0:00:01.500000"},
		{time.Microsecond, "0:00:00.000001"},
		{26*time.Hour + 3*time.Minute + 4*time.Second, "26:03:04.000000"},
		{-2 * time.Second, "-0:00:02.000000"},
	}
	for _, tt := range tests {
		if got := Duration(tt.in); got != tt.want {
			t.Errorf("Duration(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
