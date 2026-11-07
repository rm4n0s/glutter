package web

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestMotionReading(t *testing.T) {
	flat := MotionEvent{HasAcceleration: true, Acceleration: [3]float64{0, 0, 9.81}, HasRotation: true, Alpha: 90, Beta: 180, Gamma: -45, TimeStamp: 1500.5, Interval: 16}
	tests := []struct {
		name string
		kind platform.SensorKind
		e    MotionEvent
		want platform.SensorReading
		ok   bool
	}{
		{"a phone lying flat reads gravity on z", platform.SensorAccelerometer, flat,
			platform.SensorReading{Time: 1500500 * time.Microsecond, Values: [3]float64{0, 0, 9.81}}, true},
		{"the gyroscope: x is beta, y is gamma, z is alpha, in rad/s", platform.SensorGyroscope, flat,
			platform.SensorReading{Time: 1500500 * time.Microsecond, Values: [3]float64{math.Pi, -math.Pi / 4, math.Pi / 2}}, true},
		{"no acceleration in the event", platform.SensorAccelerometer, MotionEvent{HasRotation: true}, platform.SensorReading{}, false},
		{"no rotation in the event", platform.SensorGyroscope, MotionEvent{HasAcceleration: true}, platform.SensorReading{}, false},
		{"a number that is not finite", platform.SensorAccelerometer, MotionEvent{HasAcceleration: true, Acceleration: [3]float64{math.NaN(), 0, 0}}, platform.SensorReading{}, false},
		{"an infinite rotation", platform.SensorGyroscope, MotionEvent{HasRotation: true, Alpha: math.Inf(1)}, platform.SensorReading{}, false},
		{"an unknown kind", platform.SensorKind(9), flat, platform.SensorReading{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MotionReading(tt.kind, tt.e)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.Time != tt.want.Time {
				t.Errorf("Time = %v, want %v", got.Time, tt.want.Time)
			}
			for i := range got.Values {
				if math.Abs(got.Values[i]-tt.want.Values[i]) > 1e-12 {
					t.Errorf("Values = %v, want %v", got.Values, tt.want.Values)
					break
				}
			}
		})
	}
}

func TestMotionRate(t *testing.T) {
	tests := []struct {
		interval, want float64
	}{{16, 62.5}, {10, 100}, {1000, 1}, {0, 0}, {-5, 0}, {math.NaN(), 0}, {math.Inf(1), 0}}
	for _, tt := range tests {
		if got := MotionRate(tt.interval); got != tt.want {
			t.Errorf("MotionRate(%v) = %v, want %v", tt.interval, got, tt.want)
		}
	}
}

func TestHapticMillis(t *testing.T) {
	tests := []struct {
		kind ui.HapticFeedbackKind
		want int
	}{
		{ui.HapticFeedbackVibrate, 50}, {ui.HapticFeedbackLightImpact, 10}, {ui.HapticFeedbackMediumImpact, 20},
		{ui.HapticFeedbackHeavyImpact, 40}, {ui.HapticFeedbackSelectionClick, 5}, {ui.HapticFeedbackUnspecified, 0}, {ui.HapticFeedbackKind(99), 0},
	}
	for _, tt := range tests {
		if got := HapticMillis(tt.kind); got != tt.want {
			t.Errorf("HapticMillis(%d) = %d, want %d", tt.kind, got, tt.want)
		}
	}
	// The kinds are told apart by their lengths: stronger is longer.
	light, medium, heavy := HapticMillis(ui.HapticFeedbackLightImpact), HapticMillis(ui.HapticFeedbackMediumImpact), HapticMillis(ui.HapticFeedbackHeavyImpact)
	if light >= medium || medium >= heavy {
		t.Error("the impacts are not ordered by strength")
	}
}
