//go:build !linux

package android

// newAudioBackend: this package is built for other systems only to be checked, and they have no AAudio.
func newAudioBackend(*Platform) audioBackend { return nil }
