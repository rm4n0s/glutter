//go:build darwin

package coreaudio

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

// TestRealLibraryBinds binds Core Audio and AudioToolbox, and lists the devices of this machine (a Mac with no audio device lists
// none, which is no error).
func TestRealLibraryBinds(t *testing.T) {
	lib, err := loadLibrary()
	if err != nil {
		t.Fatal(err)
	}
	if lib.outputCB == 0 || lib.inputCB == 0 || lib.listenerCB == 0 {
		t.Error("a callback was not made")
	}
	s := New(Options{})
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		list, err := s.Devices(kind)
		if err != nil {
			t.Fatalf("Devices(%d): %v", kind, err)
		}
		defaults := 0
		for _, d := range list {
			if d.ID == "" || d.Name == "" {
				t.Errorf("a device with no ID or name: %+v", d)
			}
			if d.Default {
				defaults++
			}
			t.Logf("kind %d: %q (%s) default %v format %+v", kind, d.Name, d.ID, d.Default, d.Format)
		}
		if defaults > 1 {
			t.Errorf("%d default devices of kind %d", defaults, kind)
		}
	}
}
