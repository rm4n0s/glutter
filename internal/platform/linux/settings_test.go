//go:build linux

package linux

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func noEnv(string) string { return "" }

// settingsRecorder collects what the watcher passes on.
type settingsRecorder struct {
	mu  sync.Mutex
	got []platform.Settings
}

func (r *settingsRecorder) apply(s platform.Settings) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, s)
}

func (r *settingsRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.got)
}

func portalBus(color uint32) *fakebus.Bus {
	b := fakebus.New(":1.5")
	b.Handle(dbus.PortalDest, dbus.PortalPath, "org.freedesktop.portal.Settings.ReadAll", func(fakebus.Call) ([]any, error) {
		return []any{map[string]map[string]godbus.Variant{
			"org.freedesktop.appearance": {"color-scheme": godbus.MakeVariant(color)},
		}}, nil
	})
	return b
}

func TestSettingsWatcherFollowsThePortal(t *testing.T) {
	bus := portalBus(1)
	var rec settingsRecorder
	w := &SettingsWatcher{
		Post: func(fn func()) { fn() }, Apply: rec.apply, Getenv: noEnv,
		Open: func(context.Context) (dbus.Bus, error) { return bus, nil },
	}
	s, fromPortal := w.Start()
	if !fromPortal || s.PlatformBrightness != ui.BrightnessDark {
		t.Fatalf("Start() = %+v, %v", s, fromPortal)
	}
	// A second Start reads nothing again.
	if again, _ := w.Start(); again.PlatformBrightness != ui.BrightnessDark {
		t.Errorf("second Start() = %+v", again)
	}
	bus.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged",
		Body: []any{"org.freedesktop.appearance", "color-scheme", godbus.MakeVariant(uint32(2))}})
	deadline := time.Now().Add(5 * time.Second)
	for rec.count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Apply was not called")
		}
		time.Sleep(time.Millisecond)
	}
	if w.Current().PlatformBrightness != ui.BrightnessLight {
		t.Errorf("Current() = %+v", w.Current())
	}
	w.Stop()
	w.Stop() // twice is fine
}

func TestSettingsWatcherWithoutAPortal(t *testing.T) {
	tests := []struct {
		name string
		open func(context.Context) (dbus.Bus, error)
	}{
		{"the bus cannot be opened", func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagNoBus, "no bus") }},
		{"the bus has no portal", func(context.Context) (dbus.Bus, error) { return fakebus.New(":1.6"), nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec settingsRecorder
			w := &SettingsWatcher{Post: func(fn func()) { fn() }, Apply: rec.apply, Open: tt.open,
				Getenv: func(k string) string {
					return map[string]string{"XCURSOR_SIZE": "30", "LANG": "es_ES.UTF-8"}[k]
				}}
			s, fromPortal := w.Start()
			if fromPortal || s.CursorSize != 30 || len(s.Locales) != 1 || s.Locales[0] != ui.NewLocale("es", "ES") {
				t.Errorf("Start() = %+v, %v", s, fromPortal)
			}
			w.Stop()
		})
	}
}

func TestSettingsWatcherSetAndReplace(t *testing.T) {
	var rec settingsRecorder
	w := &SettingsWatcher{Post: func(fn func()) { fn() }, Apply: rec.apply, Getenv: noEnv,
		Open: func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagNoBus, "no bus") }}
	first, _ := w.Start()
	dark := first
	dark.PlatformBrightness = ui.BrightnessDark
	dark.Locales = []ui.Locale{ui.NewLocale("fr")}

	w.Replace(dark) // the first reading of another source: nobody is told
	if rec.count() != 0 || w.Current().PlatformBrightness != ui.BrightnessDark {
		t.Errorf("Replace told %d times; current %+v", rec.count(), w.Current())
	}
	w.Set(dark) // the same, locales compared by value
	dark.Locales = []ui.Locale{ui.NewLocale("fr")}
	w.Set(dark)
	if rec.count() != 0 {
		t.Errorf("an unchanged Set told %d times", rec.count())
	}
	light := dark
	light.PlatformBrightness = ui.BrightnessLight
	w.Set(light)
	if rec.count() != 1 || rec.got[0].PlatformBrightness != ui.BrightnessLight {
		t.Errorf("Set told %+v", rec.got)
	}
}

func TestSessionBusPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bus"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"an address", map[string]string{"DBUS_SESSION_BUS_ADDRESS": "unix:path=/x"}, true},
		{"the socket of the user bus", map[string]string{"XDG_RUNTIME_DIR": dir}, true},
		{"neither", map[string]string{"XDG_RUNTIME_DIR": t.TempDir()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionBusPresent(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("sessionBusPresent = %v, want %v", got, tt.want)
			}
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNoBus = "NoBus"
)
