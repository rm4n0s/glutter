//go:build linux

package android

import "github.com/rm4n0s/glutter/internal/platform/android/aaudio"

// newAudioBackend makes the AAudio service (SPEC §28.3).
func newAudioBackend(p *Platform) audioBackend {
	return aaudio.New(aaudio.Options{Permissions: audioPermission{b: p.b, h: p.host}, Log: p.log})
}
