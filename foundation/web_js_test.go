//go:build js && wasm

package foundation

import "testing"

const webTarget = true

// TestSetBrowserTargetPlatform checks that the platform the browser reports is the
// one the framework behaves like, and that Linux is the default until it is told.
func TestSetBrowserTargetPlatform(t *testing.T) {
	t.Cleanup(func() { SetBrowserTargetPlatform(TargetPlatformUnspecified) })
	tests := []struct {
		name string
		set  TargetPlatform
		want TargetPlatform
	}{
		{"not told yet", TargetPlatformUnspecified, TargetPlatformLinux},
		{"android", TargetPlatformAndroid, TargetPlatformAndroid},
		{"iOS", TargetPlatformIOS, TargetPlatformIOS},
		{"macOS", TargetPlatformMacOS, TargetPlatformMacOS},
		{"windows", TargetPlatformWindows, TargetPlatformWindows},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if DebugDefaultTargetPlatformOverride().IsSome() {
				t.Skip("a test override is in force")
			}
			SetBrowserTargetPlatform(tc.set)
			if got := DefaultTargetPlatform(); got != tc.want {
				t.Errorf("DefaultTargetPlatform() = %v, want %v", got, tc.want)
			}
		})
	}
}
