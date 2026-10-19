package dbus_test

import (
	"context"
	"reflect"
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

const settingsMethod = "org.freedesktop.portal.Settings.ReadAll"

type vars = map[string]map[string]godbus.Variant

func v(x any) godbus.Variant { return godbus.MakeVariant(x) }

// portalWith makes a fake bus whose Settings portal answers ReadAll with all.
func portalWith(all vars) *fakebus.Bus {
	b := fakebus.New(":1.42")
	b.Handle(dbus.PortalDest, dbus.PortalPath, settingsMethod, func(fakebus.Call) ([]any, error) {
		return []any{all}, nil
	})
	return b
}

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func TestReadSettings(t *testing.T) {
	base := platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 24}
	tests := []struct {
		name string
		all  vars
		want func(*platform.Settings)
	}{
		{"empty keeps the base", vars{}, func(*platform.Settings) {}},
		{"dark scheme", vars{"org.freedesktop.appearance": {"color-scheme": v(uint32(1))}},
			func(s *platform.Settings) { s.PlatformBrightness = ui.BrightnessDark }},
		{"light scheme", vars{"org.freedesktop.appearance": {"color-scheme": v(uint32(2))}}, func(*platform.Settings) {}},
		{"no preference is light", vars{"org.freedesktop.appearance": {"color-scheme": v(uint32(0))}}, func(*platform.Settings) {}},
		{"accent color", vars{"org.freedesktop.appearance": {"accent-color": v([]any{1.0, 0.0, 0.5})}},
			func(s *platform.Settings) {
				s.HasAccentColor = true
				s.AccentColor = ui.NewColorFrom(1, 1, 0, 0.5)
			}},
		{"accent color out of range is none", vars{"org.freedesktop.appearance": {"accent-color": v([]any{-1.0, 0.0, 0.5})}},
			func(*platform.Settings) {}},
		{"more contrast", vars{"org.freedesktop.appearance": {"contrast": v(uint32(1))}},
			func(s *platform.Settings) { s.HighContrast = true }},
		{"text scale", vars{"org.gnome.desktop.interface": {"text-scaling-factor": v(1.25)}},
			func(s *platform.Settings) { s.TextScaleFactor = 1.25 }},
		{"animations off", vars{"org.gnome.desktop.interface": {"enable-animations": v(false)}},
			func(s *platform.Settings) { s.DisableAnimations = true }},
		{"animations on", vars{"org.gnome.desktop.interface": {"enable-animations": v(true)}}, func(*platform.Settings) {}},
		{"cursor", vars{"org.gnome.desktop.interface": {"cursor-size": v(int32(32)), "cursor-theme": v("Bibata")}},
			func(s *platform.Settings) { s.CursorSize = 32; s.CursorTheme = "Bibata" }},
		{"font rendering", vars{"org.gnome.desktop.interface": {"font-antialiasing": v("rgba"), "font-hinting": v("slight")}},
			func(s *platform.Settings) {
				s.FontAntialiasing = platform.FontAntialiasingRGBA
				s.FontHinting = platform.FontHintingSlight
			}},
		{"nested variants are unwrapped", vars{"org.gnome.desktop.interface": {"text-scaling-factor": v(v(1.5))}},
			func(s *platform.Settings) { s.TextScaleFactor = 1.5 }},
		{"wrong types and unknown keys are ignored", vars{
			"org.gnome.desktop.interface": {"text-scaling-factor": v("big"), "cursor-size": v("x"), "unknown": v(1)},
			"org.freedesktop.appearance":  {"color-scheme": v("dark")},
			"other.namespace":             {"color-scheme": v(uint32(1))},
		}, func(*platform.Settings) {}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := base
			tc.want(&want)
			got, err := dbus.ReadSettings(context.Background(), portalWith(tc.all), base)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("settings = %+v, want %+v", got, want)
			}
		})
	}
}

func TestReadSettingsAsksForBothNamespaces(t *testing.T) {
	b := portalWith(vars{})
	if _, err := dbus.ReadSettings(context.Background(), b, platform.Settings{}); err != nil {
		t.Fatal(err)
	}
	calls := b.Calls()
	if len(calls) != 1 || calls[0].Method != settingsMethod {
		t.Fatalf("calls = %+v", calls)
	}
	want := []string{"org.freedesktop.appearance", "org.gnome.desktop.interface"}
	if !reflect.DeepEqual(calls[0].Args, []any{want}) {
		t.Errorf("args = %#v, want %#v", calls[0].Args, want)
	}
}

func TestReadSettingsErrors(t *testing.T) {
	t.Run("portal missing", func(t *testing.T) {
		_, err := dbus.ReadSettings(context.Background(), fakebus.New(":1.1"), platform.Settings{})
		assertRoute(t, err, "ReadSettings->Bus.Call.DBusCallFailed")
	})
	t.Run("no values", func(t *testing.T) {
		b := fakebus.New(":1.1")
		b.Handle(dbus.PortalDest, dbus.PortalPath, settingsMethod, func(fakebus.Call) ([]any, error) { return nil, nil })
		_, err := dbus.ReadSettings(context.Background(), b, platform.Settings{})
		assertRoute(t, err, "ReadSettings.SettingsReplyInvalid")
	})
	t.Run("wrong type", func(t *testing.T) {
		b := fakebus.New(":1.1")
		b.Handle(dbus.PortalDest, dbus.PortalPath, settingsMethod, func(fakebus.Call) ([]any, error) { return []any{"x"}, nil })
		_, err := dbus.ReadSettings(context.Background(), b, platform.Settings{})
		assertRoute(t, err, "ReadSettings.SettingsReplyInvalid")
	})
}

func TestFallbackSettings(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want platform.Settings
	}{
		{"defaults", nil, platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 24}},
		{"cursor from the environment", map[string]string{"XCURSOR_SIZE": "48", "XCURSOR_THEME": "Adwaita"},
			platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 48, CursorTheme: "Adwaita"}},
		{"bad size", map[string]string{"XCURSOR_SIZE": "big"},
			platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 24}},
		{"zero size", map[string]string{"XCURSOR_SIZE": "0"},
			platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 24}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := dbus.FallbackSettings(env(tc.env)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("settings = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// delivery collects what WatchSettings hands to OnSettings.
type delivery struct {
	mu  sync.Mutex
	got []platform.Settings
	ch  chan struct{}
}

func newDelivery() *delivery { return &delivery{ch: make(chan struct{}, 16)} }

// post runs fn at once, standing for Platform.Post.
func (d *delivery) post(fn func()) { fn() }

func (d *delivery) onSettings(s platform.Settings) {
	d.mu.Lock()
	d.got = append(d.got, s)
	d.mu.Unlock()
	d.ch <- struct{}{}
}

func (d *delivery) wait(t *testing.T) platform.Settings {
	t.Helper()
	select {
	case <-d.ch:
	case <-time.After(5 * time.Second):
		t.Fatal("OnSettings was not called")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.got[len(d.got)-1]
}

func (d *delivery) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.got)
}

func changed(namespace, key string, value any) *godbus.Signal {
	return &godbus.Signal{
		Sender: ":1.7",
		Path:   dbus.PortalPath,
		Name:   "org.freedesktop.portal.Settings.SettingChanged",
		Body:   []any{namespace, key, v(value)},
	}
}

func TestWatchSettingsFollowsChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := portalWith(vars{"org.freedesktop.appearance": {"color-scheme": v(uint32(2))}})
	d := newDelivery()

	initial := dbus.WatchSettings(ctx, b, env(nil), d.post, d.onSettings)
	if initial.PlatformBrightness != ui.BrightnessLight {
		t.Fatalf("initial brightness = %v", initial.PlatformBrightness)
	}

	b.Emit(changed("org.freedesktop.appearance", "color-scheme", uint32(1)))
	got := d.wait(t)
	if got.PlatformBrightness != ui.BrightnessDark {
		t.Errorf("brightness after the change = %v, want dark", got.PlatformBrightness)
	}

	b.Emit(changed("org.gnome.desktop.interface", "text-scaling-factor", 1.5))
	b.Emit(changed("org.gnome.desktop.interface", "enable-animations", false))
	d.wait(t)
	got = d.wait(t)
	if got.TextScaleFactor != 1.5 || !got.DisableAnimations || got.PlatformBrightness != ui.BrightnessDark {
		t.Errorf("settings = %+v, want scale 1.5, animations off and dark kept", got)
	}
}

func TestWatchSettingsSkipsUnchangedAndUnknownSignals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := portalWith(vars{"org.gnome.desktop.interface": {"text-scaling-factor": v(1.25)}})
	d := newDelivery()
	dbus.WatchSettings(ctx, b, env(nil), d.post, d.onSettings)

	b.Emit(changed("org.gnome.desktop.interface", "text-scaling-factor", 1.25))                                             // same value
	b.Emit(changed("org.gnome.desktop.interface", "gtk-theme", "Adwaita"))                                                  // key not used
	b.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged", Body: []any{"x"}}) // malformed
	b.Emit(changed("org.gnome.desktop.interface", "text-scaling-factor", 2.0))
	if got := d.wait(t); got.TextScaleFactor != 2 {
		t.Errorf("scale = %v, want 2", got.TextScaleFactor)
	}
	if n := d.count(); n != 1 {
		t.Errorf("OnSettings was called %d times, want 1", n)
	}
}

func TestWatchSettingsStopsWhenTheContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := portalWith(vars{})
	dbus.WatchSettings(ctx, b, env(nil), func(fn func()) { fn() }, func(platform.Settings) {})
	if n := b.Subscribers(); n != 1 {
		t.Fatalf("subscribers = %d, want 1", n)
	}
	cancel()
	waitForNoSubscribers(t, b)
}

// waitForNoSubscribers waits for the subscriptions of b to close: they close
// when their context is done, in another goroutine.
func waitForNoSubscribers(t *testing.T, b *fakebus.Bus) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for b.Subscribers() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the subscription was not closed")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestWatchSettingsFallsBackWhenThePortalIsMissing(t *testing.T) {
	want := platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 40, CursorTheme: "X"}
	e := env(map[string]string{"XCURSOR_SIZE": "40", "XCURSOR_THEME": "X"})
	d := newDelivery()

	t.Run("no bus", func(t *testing.T) {
		if got := dbus.WatchSettings(context.Background(), nil, e, d.post, d.onSettings); !reflect.DeepEqual(got, want) {
			t.Errorf("settings = %+v, want %+v", got, want)
		}
	})
	t.Run("no portal", func(t *testing.T) {
		b := fakebus.New(":1.1")
		if got := dbus.WatchSettings(context.Background(), b, e, d.post, d.onSettings); !reflect.DeepEqual(got, want) {
			t.Errorf("settings = %+v, want %+v", got, want)
		}
		waitForNoSubscribers(t, b)
	})
	if n := d.count(); n != 0 {
		t.Errorf("OnSettings was called %d times, want 0", n)
	}
}
