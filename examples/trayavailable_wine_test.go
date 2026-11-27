//go:build linux && glutter_wine

package examples

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// TestWineTellsWhetherTheDesktopShowsTrayIcons is the issue "The tray's availability" (step 60.1) on Windows, under
// Wine: examples/dev/trayavailable, built for Windows, asks desktop.TrayAvailable from a goroutine, places a
// SystemTray (a notify icon of the notification area, Shell_NotifyIconW) and prints the report of internal/issues.
// Windows has a notification area, so the answer is yes and the tray is shown (it reports nothing). Run it with
// `go test -tags glutter_wine -p 1 -run TellsWhetherTheDesktopShowsTrayIcons -v ./examples`.
func TestWineTellsWhetherTheDesktopShowsTrayIcons(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := h.Stop(stop); err != nil {
			t.Errorf("Stop: %v", err)
		}
		if left := winetest.Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	res, err := wintool.Build(ctx, wintool.ExecRunner{}, wintool.BuildOptions{
		Debug:   true,
		Package: "./examples/dev/trayavailable", Dir: h.Root, Out: filepath.Join(t.TempDir(), "trayavailable.exe"),
	})
	if err != nil {
		t.Fatalf("glutter-windows build ./examples/dev/trayavailable: %v", err)
	}
	run, err := h.Run(ctx, res.Exes[0].Path, winetest.RunOptions{Env: []string{"GLUTTER_RENDERER=soft"}, Timeout: 5 * time.Minute})
	if err != nil || run.ExitCode != 0 {
		t.Fatalf("the app: %v, status %d\n%s", err, run.ExitCode, run.Output)
	}
	lines := strings.Split(run.Output, "\n")
	if !strings.Contains(run.Output, issues.DoneLine) {
		t.Fatalf("the app did not finish the reproduction:\n%s", run.Output)
	}
	rep, err := issues.ParseTrayAvailable(lines)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", strings.Join(rep.Lines(), "\n"))
	if rep.Platform != "windows" {
		t.Errorf("the app ran as %q, want windows", rep.Platform)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
	if t.Failed() {
		t.Logf("the output:\n%s", run.Output)
	}
}
