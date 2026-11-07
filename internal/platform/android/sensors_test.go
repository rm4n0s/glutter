//go:build linux

package android

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// fakeSensors is a sensors backend that counts its closes.
type fakeSensors struct{ closed int }

func (*fakeSensors) Available(platform.SensorKind) bool { return true }
func (*fakeSensors) Open(context.Context, platform.SensorKind, platform.SensorOpenOptions, platform.SensorHandler) (platform.SensorDevice, error) {
	return nil, nil
}
func (f *fakeSensors) Close() { f.closed++ }

func TestTheServicesHaveTheSensorsOfTheSystem(t *testing.T) {
	h := newHarness(t)
	got := h.p.Services().Sensors()
	if got == nil {
		t.Fatal("the services have no sensors service")
	}
	if h.p.Services().Sensors() != got {
		t.Error("the sensors service was made again")
	}
	// The desktop has no libandroid: the service is there, and says so when a sensor is asked for.
	t.Setenv("GLUTTER_ANDROID_LIBRARY", "/nonexistent/libandroid.so")
	_, err := got.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, nil)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagSensorUnavailable {
		t.Errorf("Open without libandroid: %v", err)
	}
}

func TestClosingThePlatformClosesTheSensorsService(t *testing.T) {
	h := newHarness(t)
	fake := &fakeSensors{}
	h.p.sensors.once.Do(func() { h.p.sensors.svc = fake })
	if h.p.Services().Sensors() == nil {
		t.Fatal("the services have no sensors service")
	}
	if err := h.p.Close(); err != nil {
		t.Fatal(err)
	}
	if fake.closed != 1 || h.p.sensors.svc != nil {
		t.Errorf("closed %d times, service kept %v", fake.closed, h.p.sensors.svc != nil)
	}
}
