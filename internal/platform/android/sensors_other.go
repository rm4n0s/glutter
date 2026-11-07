//go:build !linux

package android

// newSensorBackend: this package is built for other systems only to be checked, and they have no sensor manager.
func newSensorBackend(*Platform) sensorBackend { return nil }
