package web

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestDetectOS(t *testing.T) {
	const androidUA = "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/120.0 Mobile Safari/537.36"
	const linuxUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36"
	tests := []struct {
		name     string
		platform string
		ua       string
		touch    int
		want     OperatingSystem
	}{
		{"mac", "MacIntel", linuxUA, 0, OSMacOS},
		{"mac with one touch point", "MacIntel", linuxUA, 1, OSMacOS},
		{"mac with two touch points", "MacIntel", linuxUA, 2, OSMacOS},
		{"ipad asking for the desktop site", "MacIntel", linuxUA, 5, OSIOS},
		{"iphone", "iPhone", linuxUA, 5, OSIOS},
		{"ipad", "iPad", linuxUA, 5, OSIOS},
		{"ipod in lower case", "ipod touch", linuxUA, 5, OSIOS},
		{"android reports linux", "Linux armv8l", androidUA, 5, OSAndroid},
		{"linux", "Linux x86_64", linuxUA, 0, OSLinux},
		{"windows", "Win32", linuxUA, 0, OSWindows},
		{"unknown", "FreeBSD amd64", linuxUA, 0, OSUnknown},
		{"nothing", "", "", 0, OSUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectOS(tc.platform, tc.ua, tc.touch); got != tc.want {
				t.Errorf("DetectOS(%q, ..., %d) = %v, want %v", tc.platform, tc.touch, got, tc.want)
			}
		})
	}
}

func TestOperatingSystemTargetPlatform(t *testing.T) {
	tests := []struct {
		os   OperatingSystem
		want foundation.TargetPlatform
	}{
		{OSAndroid, foundation.TargetPlatformAndroid},
		{OSIOS, foundation.TargetPlatformIOS},
		{OSLinux, foundation.TargetPlatformLinux},
		{OSMacOS, foundation.TargetPlatformMacOS},
		{OSWindows, foundation.TargetPlatformWindows},
		{OSUnknown, foundation.TargetPlatformAndroid},
	}
	for _, tc := range tests {
		if got := tc.os.TargetPlatform(); got != tc.want {
			t.Errorf("OperatingSystem(%d).TargetPlatform() = %v, want %v", tc.os, got, tc.want)
		}
	}
}

// TestDetectOSPortedFromFlutter is the table of detectOperatingSystem in Flutter's
// lib/web_ui/test/engine/browser_detect_test.dart: platform "any", user agent "any" and no touch
// points unless a case says otherwise.
func TestDetectOSPortedFromFlutter(t *testing.T) {
	const old = "Mozilla/5.0 (Linux; U; Android 2.2) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1"
	tests := []struct {
		name     string
		platform string
		ua       string
		touch    int
		want     OperatingSystem
	}{
		{"weird values", "any", "any", 0, OSUnknown},
		{"MacIntel", "MacIntel", "any", 0, OSMacOS},
		{"MacAnythingElse", "MacAnythingElse", "any", 0, OSMacOS},
		{"iPhone", "iPhone", "any", 0, OSIOS},
		{"iPhone Simulator", "iPhone Simulator", "any", 0, OSIOS},
		{"iPad", "iPad", "any", 0, OSIOS},
		{"iPad Simulator", "iPad Simulator", "any", 0, OSIOS},
		{"iPod", "iPod", "any", 0, OSIOS},
		{"iPod Simulator", "iPod Simulator", "any", 0, OSIOS},
		{"MacARM", "MacARM", "any", 0, OSMacOS},
		{"MacARM with five touch points", "MacARM", "any", 5, OSIOS},
		{"Android by its user agent", "any", old, 0, OSAndroid},
		{"Linux", "Linux", "any", 0, OSLinux},
		{"Linux armv8l", "Linux armv8l", "any", 0, OSLinux},
		{"Linux x86_64", "Linux x86_64", "any", 0, OSLinux},
		{"Windows", "Windows", "any", 0, OSWindows},
		{"Win32", "Win32", "any", 0, OSWindows},
		{"Win16", "Win16", "any", 0, OSWindows},
		{"WinCE", "WinCE", "any", 0, OSWindows},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectOS(tc.platform, tc.ua, tc.touch); got != tc.want {
				t.Errorf("DetectOS(%q, %q, %d) = %v, want %v", tc.platform, tc.ua, tc.touch, got, tc.want)
			}
		})
	}
}
