//go:build windows && glutter_wine

package windows

import (
	"context"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests change the registry of the Wine prefix (.wine-test/prefix, never the user's) and
// tell the app, as Windows does, with WM_SETTINGCHANGE (SPEC §25.8).

// regSetDWORD writes a DWORD value of HKEY_CURRENT_USER; regDeleteValue removes it.
func regSetDWORD(t testing.TB, subKey, name string, v uint32) {
	t.Helper()
	lib, err := win32.LoadSystemDLL("advapi32.dll")
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lib.Proc("RegSetKeyValueW")
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := win32.UTF16PtrFromString(subKey)
	pn, _ := win32.UTF16PtrFromString(name)
	const regDWORD = 4
	if r, _ := proc.Call(win32.HKeyCurrentUser, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pn)), regDWORD, uintptr(unsafe.Pointer(&v)), 4); r != 0 {
		t.Fatalf("RegSetKeyValueW(%s, %s) = %d", subKey, name, r)
	}
}

func regDeleteValue(t testing.TB, subKey, name string) {
	t.Helper()
	lib, err := win32.LoadSystemDLL("advapi32.dll")
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lib.Proc("RegDeleteKeyValueW")
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := win32.UTF16PtrFromString(subKey)
	pn, _ := win32.UTF16PtrFromString(name)
	_, _ = proc.Call(win32.HKeyCurrentUser, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pn))) // the value may not be there
}

// resetSettings puts the values this file writes back to absent, now and when the test ends.
func resetSettings(t *testing.T) {
	t.Helper()
	clean := func() {
		regDeleteValue(t, personalizeKey, appsUseLightThemeValue)
		regDeleteValue(t, accessibilityKey, textScaleFactorVal)
	}
	clean()
	t.Cleanup(clean)
}

func TestWineSettingsAreReadWhenAsked(t *testing.T) {
	resetSettings(t)
	p, _ := newTestPlatform(t)
	s := p.Settings()
	if s.PlatformBrightness != ui.BrightnessLight || s.TextScaleFactor != 1 {
		t.Errorf("the settings of a prefix with no values: brightness %v, text scale %v", s.PlatformBrightness, s.TextScaleFactor)
	}
	if len(s.Locales) == 0 || s.Locales[0].LanguageCode == "" {
		t.Errorf("locales = %v", s.Locales)
	}
	if s.CaretBlinkTime == 0 || s.DoubleClickTime == 0 {
		t.Errorf("caret blink %v, double click %v: Wine has both", s.CaretBlinkTime, s.DoubleClickTime)
	}
	if s.CursorSize <= 0 || s.UIFontFamily == "" || s.UIFontSize <= 0 || s.MonospaceFontFamily != "Consolas" {
		t.Errorf("cursor %d, interface font %q %v, monospace %q", s.CursorSize, s.UIFontFamily, s.UIFontSize, s.MonospaceFontFamily)
	}
	// Settings is read once: it is the same until a change arrives.
	regSetDWORD(t, personalizeKey, appsUseLightThemeValue, 0)
	if p.Settings().PlatformBrightness != ui.BrightnessLight {
		t.Error("Settings read the registry again by itself")
	}
}

func TestWineAppModeAndTextSizeFollowTheSystemLive(t *testing.T) {
	resetSettings(t)
	p, r := newTestPlatform(t)
	// Two windows: the title bar of each one follows the setting.
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	type bar struct {
		hwnd uintptr
		dark bool
	}
	var bars []bar
	p.titleBarApplied = func(hwnd uintptr, dark bool) { bars = append(bars, bar{hwnd, dark}) }
	_ = p.Settings()

	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		change := func(name string, do func(), check func()) func(next func()) {
			return func(next func()) {
				n := len(r.settings)
				do()
				var wait func(deadline time.Time)
				wait = func(deadline time.Time) {
					if len(r.settings) > n {
						check()
						next()
						return
					}
					if time.Now().After(deadline) {
						t.Errorf("%s: OnSettings never came", name)
						cancel()
						return
					}
					after(p, 20*time.Millisecond, func() { wait(deadline) })
				}
				wait(time.Now().Add(5 * time.Second))
			}
		}
		last := func() platform.Settings { return r.settings[len(r.settings)-1] }
		seq(
			change("dark", func() {
				regSetDWORD(t, personalizeKey, appsUseLightThemeValue, 0)
				bars = nil
				if err := win32.BroadcastSettingChange("ImmersiveColorSet"); err != nil {
					t.Error(err)
				}
			}, func() {
				if last().PlatformBrightness != ui.BrightnessDark {
					t.Errorf("brightness = %v, want dark", last().PlatformBrightness)
				}
				seen := map[uintptr]bool{}
				for _, b := range bars {
					if !b.dark {
						t.Errorf("a title bar was set light: %+v", b)
					}
					seen[b.hwnd] = true
				}
				if !seen[a.hwnd] || !seen[b.hwnd] {
					t.Errorf("the title bars set: %+v, want both windows %#x and %#x", bars, a.hwnd, b.hwnd)
				}
			}),
			change("text size", func() {
				regSetDWORD(t, accessibilityKey, textScaleFactorVal, 150)
				_ = win32.BroadcastSettingChange("WindowMetrics")
			}, func() {
				if last().TextScaleFactor != 1.5 || last().PlatformBrightness != ui.BrightnessDark {
					t.Errorf("text scale %v, brightness %v", last().TextScaleFactor, last().PlatformBrightness)
				}
			}),
			change("light again", func() {
				regSetDWORD(t, personalizeKey, appsUseLightThemeValue, 1)
				bars = nil
				_ = win32.BroadcastSettingChange("ImmersiveColorSet")
			}, func() {
				if last().PlatformBrightness != ui.BrightnessLight {
					t.Errorf("brightness = %v, want light", last().PlatformBrightness)
				}
				for _, b := range bars {
					if b.dark {
						t.Errorf("a title bar stayed dark: %+v", b)
					}
				}
			}),
			func(next func()) {
				// A message that changes nothing reports nothing, and many messages in one turn make
				// one read.
				n := len(r.settings)
				for range 5 {
					_ = win32.BroadcastSettingChange("Environment")
				}
				after(p, 300*time.Millisecond, func() {
					if len(r.settings) != n {
						t.Errorf("%d reports of settings that did not change", len(r.settings)-n)
					}
					next()
				})
			},
			func(next func()) { cancel() },
		)
	})
}

func TestWineSettingsMessagesAreCoalescedIntoOneRead(t *testing.T) {
	resetSettings(t)
	p, r := newTestPlatform(t)
	_ = newWin(t, p, platform.WindowConfig{})
	_ = p.Settings()
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		regSetDWORD(t, accessibilityKey, textScaleFactorVal, 200)
		for range 20 {
			_ = win32.BroadcastSettingChange("WindowMetrics")
		}
		after(p, 300*time.Millisecond, func() {
			defer cancel()
			if len(r.settings) != 1 || r.settings[0].TextScaleFactor != 2 {
				t.Errorf("settings reported: %d, %+v", len(r.settings), r.settings)
			}
		})
	})
}
