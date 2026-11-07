//go:build linux

package android

import "github.com/rm4n0s/glutter/internal/platform/android/asensor"

// newSensorBackend makes the sensors service of the NDK (SPEC §28.4).
func newSensorBackend(p *Platform) sensorBackend {
	return asensor.New(asensor.Options{Log: p.log})
}
