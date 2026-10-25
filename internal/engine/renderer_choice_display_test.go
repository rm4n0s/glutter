//go:build linux && glutter_display

package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// These tests run the engine in a headless Wayland compositor (scripts/headless-wayland.sh) and
// check which renderer it chooses (SPEC §9.6, step 20.7): Vulkan where it works, GLES where it
// does not, and never a CPU Vulkan device unless Vulkan was asked for by name.

// Each case runs in a process of its own: Mesa's EGL on Wayland does not survive a second display
// connection of the same process reliably (its wl_shm and callback ids collided in about one run
// in three when the cases ran under the load of the other display tests), and an app has one
// connection anyway.

// choiceChildEnv makes the test binary run only the child, which is the app of a case.
const choiceChildEnv = "GLUTTER_CHOICE_CHILD"

// TestDisplayChoiceChild is the app of a case, started by chosenRenderer; it prints what the log
// says about the renderer. It does nothing in a normal run.
func TestDisplayChoiceChild(t *testing.T) {
	if os.Getenv(choiceChildEnv) == "" {
		t.Skip("run by chosenRenderer")
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
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, rec.EndRecording())
			return b.Build()
		},
	})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, buf.String())
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "renderer chosen") {
			fmt.Println("CHOSEN: " + line)
			return
		}
	}
	t.Fatalf("the log has no \"renderer chosen\" line:\n%s", buf.String())
}

// chosenRenderer runs an app that draws one frame, saves it and exits, in a process of its own
// with the environment, and returns what its log says about the renderer.
func chosenRenderer(t *testing.T, env map[string]string) string {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	files, _ := filepath.Glob("/usr/share/vulkan/icd.d/lvp_icd.*.json")
	if len(files) == 0 {
		t.Skip("Mesa's lavapipe is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplayChoiceChild$", "-test.count=1")
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), choiceChildEnv+"=1")
	for k, v := range env {
		if v == "lavapipe" {
			v = files[0]
		}
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the app failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(line, "CHOSEN: "); ok {
			return rest
		}
	}
	t.Fatalf("the app did not say which renderer it chose:\n%s", out)
	return ""
}

func TestDisplayWithoutVulkanAnAppFallsBackToGLES(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "/nonexistent/icd.json", "GLUTTER_RENDERER": "auto"})
	if !strings.Contains(line, "gles") {
		t.Errorf("the app used %q, want gles", line)
	}
}

func TestDisplayAutoNeverUsesACPUVulkanDevice(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "lavapipe", "GLUTTER_RENDERER": "auto"})
	if !strings.Contains(line, "gles") || strings.Contains(line, "vulkan") {
		t.Errorf("the app used %q, want gles", line)
	}
}

func TestDisplayVulkanByNameMayUseTheCPUDevice(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "lavapipe", "GLUTTER_RENDERER": "vulkan"})
	if !strings.Contains(line, "vulkan: llvmpipe") {
		t.Errorf("the app used %q, want vulkan: llvmpipe", line)
	}
}
