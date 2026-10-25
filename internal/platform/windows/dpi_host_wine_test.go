//go:build linux && glutter_wine

package windows_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/winetest"
)

// TestScaleFollowsTheDPIOfTheDisplay: a prefix with LogPixels 144 gives a scale of 1.5 and a window of
// the right physical size (SPEC §25.13). It runs on the host: it writes the registry value, stops the
// Wine server (the monitors take the DPI when it starts) and runs the Windows test binary that opens
// the window. Run it with `go test -tags glutter_wine -p 1 ./internal/platform/windows`.
func TestScaleFollowsTheDPIOfTheDisplay(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "windows.test.exe")
	build := exec.Command("go", "test", "-c", "-tags", "glutter_wine", "-o", exe, "./internal/platform/windows")
	build.Dir = h.Root
	build.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go test -c: %v\n%s", err, out)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	const key = `HKCU\Control Panel\Desktop`
	regRun := func(args ...string) {
		t.Helper()
		res, err := h.Run(ctx, "reg", winetest.RunOptions{Args: args, Timeout: time.Minute})
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("reg %v: %v, status %d\n%s", args, err, res.ExitCode, res.Output)
		}
	}
	t.Cleanup(func() {
		// The prefix is shared by every test: the value goes away.
		stop, cancelStop := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelStop()
		res, err := h.Run(stop, "reg", winetest.RunOptions{Args: []string{"delete", key, "/v", "LogPixels", "/f"}, Timeout: time.Minute})
		if err != nil {
			t.Errorf("reg delete: %v\n%s", err, res.Output)
		}
		if err := h.Stop(stop); err != nil {
			t.Errorf("Stop: %v", err)
		}
		if left := winetest.Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	tests := []struct {
		dpi              int
		scale            string
		clientW, clientH int
	}{{96, "1", 320, 240}, {144, "1.5", 480, 360}, {120, "1.25", 400, 300}, {192, "2", 640, 480}}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d dpi", tt.dpi), func(t *testing.T) {
			regRun("add", key, "/v", "LogPixels", "/t", "REG_DWORD", "/d", strconv.Itoa(tt.dpi), "/f")
			res, err := h.Run(ctx, exe, winetest.RunOptions{
				Args: []string{"-test.run", "^TestDPIChild$", "-test.v", "-test.count=1"},
				Env:  []string{"GLUTTER_DPI_CHILD=1"}, Timeout: time.Minute,
			})
			if err != nil || res.ExitCode != 0 {
				t.Fatalf("the program: %v, status %d\n%s", err, res.ExitCode, res.Output)
			}
			want := fmt.Sprintf("RESULT scale=%s client=%dx%d metrics=%dx%d dpr=%s displays=%s", tt.scale, tt.clientW, tt.clientH, tt.clientW, tt.clientH, tt.scale, tt.scale)
			if !strings.Contains(res.Output, want) {
				t.Errorf("the program did not print %q:\n%s", want, res.Output)
			}
		})
	}
}
