package gluttertest

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// pinTB records the cleanups, so that a test can run them when it wants.
type pinTB struct {
	TB
	cleanups []func()
}

func (f *pinTB) Helper()          {}
func (f *pinTB) Cleanup(c func()) { f.cleanups = append(f.cleanups, c) }
func (f *pinTB) runCleanups() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}

func TestPinPlatformFor(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	none := foundation.None[foundation.TargetPlatform]()
	tests := []struct {
		name     string
		host     foundation.TargetPlatform
		existing foundation.Opt[foundation.TargetPlatform]
		want     foundation.Opt[foundation.TargetPlatform] // the override while the test runs
	}{
		{"macOS is pinned to Linux", foundation.TargetPlatformMacOS, none, foundation.Some(foundation.TargetPlatformLinux)},
		{"iOS is pinned to Linux", foundation.TargetPlatformIOS, none, foundation.Some(foundation.TargetPlatformLinux)},
		{"Linux is left alone", foundation.TargetPlatformLinux, none, none},
		{"Windows is left alone", foundation.TargetPlatformWindows, none, none},
		{"Android is left alone", foundation.TargetPlatformAndroid, none, none},
		{"a test's own platform wins on macOS", foundation.TargetPlatformMacOS, foundation.Some(foundation.TargetPlatformIOS), foundation.Some(foundation.TargetPlatformIOS)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(tt.existing)
			defer foundation.SetDebugDefaultTargetPlatformOverride(none)
			ftb := &pinTB{}
			pinPlatformFor(ftb, tt.host)
			if got := foundation.DebugDefaultTargetPlatformOverride(); got != tt.want {
				t.Errorf("override = %v, want %v", got, tt.want)
			}
			ftb.runCleanups()
			if got := foundation.DebugDefaultTargetPlatformOverride(); got != tt.existing {
				t.Errorf("override after the test = %v, want what it was before, %v", got, tt.existing)
			}
		})
	}
	t.Run("a platform that the test sets after the pin is kept", func(t *testing.T) {
		defer foundation.SetDebugDefaultTargetPlatformOverride(none)
		ftb := &pinTB{}
		pinPlatformFor(ftb, foundation.TargetPlatformMacOS)
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformWindows))
		ftb.runCleanups()
		if got := foundation.DebugDefaultTargetPlatformOverride(); got != foundation.Some(foundation.TargetPlatformWindows) {
			t.Errorf("override after the cleanup = %v, want Windows (the test's)", got)
		}
	})
}
