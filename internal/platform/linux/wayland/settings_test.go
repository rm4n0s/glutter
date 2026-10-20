package wayland

import (
	"context"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// The system settings of the Wayland backend: the portal, the environment, and what
// follows a change (SPEC §8.8).

// portal makes a fake session bus whose Settings portal answers with all.
func portal(all map[string]map[string]godbus.Variant) *fakebus.Bus {
	b := fakebus.New(":1.7")
	b.Handle(dbus.PortalDest, dbus.PortalPath, "org.freedesktop.portal.Settings.ReadAll", func(fakebus.Call) ([]any, error) {
		return []any{all}, nil
	})
	return b
}

func emit(b *fakebus.Bus, namespace, key string, value any) {
	b.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged",
		Body: []any{namespace, key, godbus.MakeVariant(value)}})
}

func TestSettingsComeFromThePortalAndFollowItsChanges(t *testing.T) {
	bus := portal(map[string]map[string]godbus.Variant{
		"org.freedesktop.appearance":  {"color-scheme": godbus.MakeVariant(uint32(1))},
		"org.gnome.desktop.interface": {"cursor-theme": godbus.MakeVariant("Bibata"), "cursor-size": godbus.MakeVariant(int32(32))},
	})
	t.Setenv("LANG", "de_DE.UTF-8")
	p, _, err := connectWith(t, fakecomp.Config{}, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	p.settings.Open = func(context.Context) (dbus.Bus, error) { return bus, nil }

	// Before Run: the portal's settings, the locale of the environment, and the cursors
	// use the portal's theme and size.
	got := p.Settings()
	if got.PlatformBrightness != ui.BrightnessDark || got.CursorTheme != "Bibata" || got.CursorSize != 32 {
		t.Errorf("Settings() = %+v", got)
	}
	if len(got.Locales) != 1 || got.Locales[0] != ui.NewLocale("de", "DE") {
		t.Errorf("locales = %v, want de_DE", got.Locales)
	}
	if p.cursors.themeName != "Bibata" || p.cursors.baseSize != 32 {
		t.Errorf("the cursors have theme %q and size %d", p.cursors.themeName, p.cursors.baseSize)
	}

	h := &frameHandler{p: p}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	t.Cleanup(func() {
		cancel()
		<-done
		p.settings.Stop()
	})

	// The user switches to the light theme, a bigger cursor and a smaller text.
	emit(bus, "org.freedesktop.appearance", "color-scheme", uint32(2))
	waitUntil(t, func() bool { return len(h.settingsSeen()) == 1 }, "OnSettings was not called for the color scheme")
	if s := h.settingsSeen()[0]; s.PlatformBrightness != ui.BrightnessLight || s.CursorTheme != "Bibata" {
		t.Errorf("after the change: %+v", s)
	}
	emit(bus, "org.gnome.desktop.interface", "cursor-size", int32(48))
	emit(bus, "org.gnome.desktop.interface", "text-scaling-factor", 1.25)
	waitUntil(t, func() bool { return len(h.settingsSeen()) == 3 }, "OnSettings was not called for the cursor size and the text scale")
	last := h.settingsSeen()[2]
	if last.CursorSize != 48 || last.TextScaleFactor != 1.25 || last.PlatformBrightness != ui.BrightnessLight {
		t.Errorf("last settings = %+v", last)
	}
	// The cursors followed (on the UI thread).
	var size int32
	finished := make(chan struct{})
	p.Post(func() { size = p.cursors.baseSize; close(finished) })
	select {
	case <-finished:
	case <-time.After(testTimeout):
		t.Fatal("the loop did not run a posted function")
	}
	if size != 48 {
		t.Errorf("the cursor size is %d, want 48", size)
	}
	// A value that did not change is not passed on again.
	emit(bus, "org.gnome.desktop.interface", "cursor-size", int32(48))
	time.Sleep(50 * time.Millisecond)
	if n := len(h.settingsSeen()); n != 3 {
		t.Errorf("%d OnSettings calls, want 3", n)
	}
}

func TestSettingsWithoutAPortalComeFromTheEnvironment(t *testing.T) {
	t.Setenv("XCURSOR_SIZE", "40")
	t.Setenv("XCURSOR_THEME", "Adwaita")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "fr_CA.UTF-8")
	p, _, err := connectWith(t, fakecomp.Config{}, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	// A bus with no portal on it.
	p.settings.Open = func(context.Context) (dbus.Bus, error) { return fakebus.New(":1.8"), nil }
	t.Cleanup(p.settings.Stop)
	want := platform.Settings{
		PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 40, CursorTheme: "Adwaita",
		Locales: []ui.Locale{ui.NewLocale("fr", "CA")},
	}
	got := p.Settings()
	if got.CursorSize != want.CursorSize || got.CursorTheme != want.CursorTheme || got.TextScaleFactor != 1 ||
		got.PlatformBrightness != want.PlatformBrightness || len(got.Locales) != 1 || got.Locales[0] != want.Locales[0] {
		t.Errorf("Settings() = %+v, want %+v", got, want)
	}
	// Asking again does not read again.
	if again := p.Settings(); again.CursorSize != 40 {
		t.Errorf("second Settings() = %+v", again)
	}
}
