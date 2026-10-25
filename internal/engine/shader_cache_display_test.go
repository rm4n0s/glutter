//go:build linux && glutter_display

package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// These tests run the engine in a headless Wayland compositor (scripts/headless-wayland.sh) twice, on the
// same shader cache directory, and read from the log how the shaders of each start were made (SPEC §9.7, step
// 27.10): the second start must find every program in the cache. Each start is a process of its own, as an app
// is.

const cacheChildEnv = "GLUTTER_CACHE_CHILD"

// TestDisplayShaderCacheChild is the app of a start; it prints the "shaders" line of its log. It does nothing in
// a normal run.
func TestDisplayShaderCacheChild(t *testing.T) {
	if os.Getenv(cacheChildEnv) == "" {
		t.Skip("run by startWithCache")
	}
	runtime.LockOSThread() // the EGL context belongs to one thread, as it does for RunApp on the main thread
	defer runtime.UnlockOSThread()
	t.Setenv("GLUTTER_SCREENSHOT", filepath.Join(t.TempDir(), "shot.png"))
	t.Setenv("GLUTTER_BACKEND", "wayland")
	log, buf := recordingLog()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := Run(ctx, Config{
		Logger: log,
		Scene: func(size ui.Size, dpr float64) *ui.Scene {
			rec := ui.NewPictureRecorder()
			c := ui.NewCanvas(rec)
			p := ui.NewPaint()
			p.SetColor(ui.NewColor(0xFF2060C0))
			c.DrawRect(ui.NewRectFromLTWH(10, 10, 100, 60), p)
			c.DrawCircle(ui.Offset{Dx: 200, Dy: 100}, 40, p)
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, rec.EndRecording())
			return b.Build()
		},
	})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, buf.String())
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "stage=shaders") {
			fmt.Println("SHADERS: " + line)
			return
		}
	}
	t.Fatalf("the log has no \"shaders\" line:\n%s", buf.String())
}

// shaderCounts is how the shaders of a start were made.
type shaderCounts struct{ built, cached, refused int }

// startWithCache runs an app that draws one frame, with the renderer and the cache directory, and returns what
// its log says about the shaders.
func startWithCache(t *testing.T, renderer, cacheDir string) shaderCounts {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplayShaderCacheChild$", "-test.count=1")
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), cacheChildEnv+"=1", "GLUTTER_FRAME_STATS=1", "GLUTTER_RENDERER="+renderer, "XDG_CACHE_HOME="+cacheDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the app failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		rest, ok := strings.CutPrefix(line, "SHADERS: ")
		if !ok {
			continue
		}
		var c shaderCounts
		for _, f := range strings.Fields(rest) {
			k, v, ok := strings.Cut(f, "=")
			n, _ := strconv.Atoi(v)
			if !ok {
				continue
			}
			switch k {
			case "built":
				c.built = n
			case "from_cache":
				c.cached = n
			case "refused":
				c.refused = n
			}
		}
		return c
	}
	t.Fatalf("the app did not print its shaders:\n%s", out)
	return shaderCounts{}
}

func TestDisplayASecondStartFindsEveryProgramInTheCache(t *testing.T) {
	cache := t.TempDir()
	first := startWithCache(t, "gles", cache)
	if first.built == 0 || first.cached != 0 {
		t.Fatalf("the first start: %+v, want programs built and none from the cache", first)
	}
	entries, err := filepath.Glob(filepath.Join(cache, "glutter", "shaders", "*.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Skip("the driver of this machine gives no program binaries, so nothing was stored")
	}
	second := startWithCache(t, "gles", cache)
	if second.built != 0 || second.cached != first.built || second.refused != 0 {
		t.Errorf("the second start: %+v, want all %d programs from the cache and none built", second, first.built)
	}
	if len(entries) != first.built {
		t.Errorf("%d files in the cache for %d programs", len(entries), first.built)
	}
}

func TestDisplayTheCacheOff(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("GLUTTER_SHADER_CACHE", "0")
	first := startWithCache(t, "gles", cache)
	second := startWithCache(t, "gles", cache)
	if first.built == 0 || second.built != first.built || second.cached != 0 {
		t.Errorf("starts with the cache off: %+v, %+v", first, second)
	}
	if entries, _ := filepath.Glob(filepath.Join(cache, "glutter", "shaders", "*")); len(entries) != 0 {
		t.Errorf("the cache is off and has files: %v", entries)
	}
}
