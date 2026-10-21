package widgets

import (
	"context"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/ui"
)

// TestASettingsSignalChangesTheMediaQuery follows a change of the system settings
// from the Settings portal (a signal on a fake bus) through the platform's settings
// watcher and the dispatcher to what MediaQuery tells the widgets.
func TestASettingsSignalChangesTheMediaQuery(t *testing.T) {
	bus := fakebus.New(":1.9")
	bus.Handle(dbus.PortalDest, dbus.PortalPath, "org.freedesktop.portal.Settings.ReadAll", func(fakebus.Call) ([]any, error) {
		return []any{map[string]map[string]godbus.Variant{
			"org.freedesktop.appearance": {"color-scheme": godbus.MakeVariant(uint32(2))}, // light
		}}, nil
	})
	type seen struct {
		brightness ui.Brightness
		scale      float64
		noMotion   bool
	}
	var builds []seen
	b, host, err := startFake(t, builderOf(func(bc BuildContext) Widget {
		builds = append(builds, seen{
			MediaQueryPlatformBrightnessOf(bc),
			MediaQueryOf(bc).TextScaler.Scale(10) / 10,
			MediaQueryOf(bc).DisableAnimations,
		})
		return newBox(10, 10)
	}), true, RunAppOpts{}, true)
	if err != nil {
		t.Fatal(err)
	}
	frame := func() {
		t.Helper()
		b.Scheduler().HandleBeginFrame(foundation.Some(time.Duration(0)))
		b.Scheduler().HandleDrawFrame()
	}
	frame()
	if len(builds) != 1 || builds[0] != (seen{ui.BrightnessLight, 1, false}) {
		t.Fatalf("builds = %+v, want one at the light theme, scale 1, animations on", builds)
	}

	// What the engine does with the settings of the platform: the watcher is the
	// platform's, the apply function is the engine's OnSettings. The UI thread of the
	// test is this one: the posted functions are collected and run by hand.
	posted := make(chan func(), 16)
	apply := func(s platform.Settings) {
		b.PlatformDispatcher().UpdatePlatformConfiguration(engine.PlatformConfiguration(s))
	}
	w := &linux.SettingsWatcher{
		Post:   func(fn func()) { posted <- fn },
		Apply:  apply,
		Getenv: func(string) string { return "" },
		Open:   func(context.Context) (dbus.Bus, error) { return bus, nil },
	}
	t.Cleanup(w.Stop)
	initial, fromPortal := w.Start()
	if !fromPortal {
		t.Fatal("the fake portal was not used")
	}
	apply(initial)
	frame()

	emit := func(namespace, key string, value any) {
		bus.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged",
			Body: []any{namespace, key, godbus.MakeVariant(value)}})
		select {
		case fn := <-posted:
			fn()
		case <-time.After(5 * time.Second):
			t.Fatal("the change never reached the UI thread")
		}
		if host.Scheduled < 1 {
			t.Fatal("no frame was asked for")
		}
		frame()
	}
	last := func() seen { return builds[len(builds)-1] }

	emit("org.freedesktop.appearance", "color-scheme", uint32(1)) // dark
	if last().brightness != ui.BrightnessDark {
		t.Errorf("after the dark scheme: %+v", last())
	}
	emit("org.gnome.desktop.interface", "text-scaling-factor", 1.5)
	if last().scale != 1.5 || last().brightness != ui.BrightnessDark {
		t.Errorf("after the text scale: %+v", last())
	}
	emit("org.gnome.desktop.interface", "enable-animations", false)
	if !last().noMotion {
		t.Errorf("after turning animations off: %+v", last())
	}
	emit("org.freedesktop.appearance", "color-scheme", uint32(2)) // back to light
	if last().brightness != ui.BrightnessLight || last().scale != 1.5 || !last().noMotion {
		t.Errorf("after the light scheme: %+v", last())
	}
}
