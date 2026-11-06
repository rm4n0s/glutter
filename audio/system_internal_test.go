package audio

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

// TestSystemWithoutAPlatformService: a platform that has no audio service gives a System that lists nothing and fails to open.
func TestSystemWithoutAPlatformService(t *testing.T) {
	s := newSystem(nil, func(fn func()) { fn() }, nil, nil)
	if s.Available() || s.Devices(Playback) != nil {
		t.Errorf("available %v, devices %v", s.Available(), s.Devices(Playback))
	}
	for name, open := range map[string]func() (*Device, error){
		"playback": func() (*Device, error) { return s.OpenPlayback(context.Background()) },
		"capture":  func() (*Device, error) { return s.OpenCapture(context.Background()) },
	} {
		d, err := open()
		appErr, ok := errors.FromError(err)
		if d != nil || !ok || appErr.Tag != TagUnavailable || !appErr.HasRoute("System.open.AudioUnavailable") {
			t.Errorf("%s: %v, %v", name, d, err)
		}
	}
	remove := s.AddListener(func() {})
	remove()
	s.Close()
}
