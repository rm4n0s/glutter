//go:build glutter_android

package androidtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
)

// TestAndroidTellsWhetherTheDesktopShowsTrayIcons is the issue "The tray's availability" (step 60.1) on the
// emulator: examples/dev/trayavailable asks desktop.TrayAvailable from a goroutine, places a SystemTray and logs the
// report of internal/issues. Android has no tray, so the answer is no and the tray reports TrayUnavailable (its cause
// DBusUnavailableOnAndroid: D-Bus is never tried), which matches desktop.TagTrayUnavailable.
func TestAndroidTellsWhetherTheDesktopShowsTrayIcons(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/dev/trayavailable")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}); err != nil {
		t.Fatal(Describe(err))
	}
	log, err := e.WaitLog(ctx, appID, issues.DoneLine, 2*time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	rep, err := issues.ParseTrayAvailable(strings.Split(log, "\n"))
	if err != nil {
		t.Fatal(Describe(err))
	}
	t.Logf("%s", strings.Join(rep.Lines(), "\n"))
	if rep.Platform != "android" {
		t.Errorf("the app ran as %q, want android", rep.Platform)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
	if t.Failed() {
		t.Logf("the log:\n%s", log)
	}
}
