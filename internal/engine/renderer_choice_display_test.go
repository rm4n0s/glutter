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
// check which renderer it chooses (SPEC §9.6, steps 20.7 and 32.3): Vulkan where it works, GLES where it
// does not, never a CPU Vulkan device unless Vulkan was asked for by name, and soft where the only GL is a
// rasterizer on the CPU.

// Each case runs in a process of its own: Mesa's EGL on Wayland does not survive a second display
// connection of the same process reliably (its wl_shm and callback ids collided in about one run
// in three when the cases ran under the load of the other display tests), and an app has one
// connection anyway.

// displayBackend is the display server that the tests run on: "wayland" under scripts/headless-wayland.sh,
// "x11" under scripts/headless-x11.sh, and "" when there is none.
func displayBackend() string {
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "":
		return "wayland"
	case os.Getenv("DISPLAY") != "":
		return "x11"
	}
	return ""
}

// choiceChildEnv makes the test binary run only the child, which is the app of a case; its value is the
// backend of the display server.
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
	t.Setenv("GLUTTER_BACKEND", os.Getenv(choiceChildEnv))
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
		if strings.Contains(line, "renderer not usable") {
			fmt.Println("TRIED: " + line) // what auto gave up on, and why
		}
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
	backend := displayBackend()
	if backend == "" {
		t.Skip("there is no display server (run under scripts/headless-wayland.sh or scripts/headless-x11.sh)")
	}
	icd := lavapipeICD()
	if icd == "" {
		t.Skip("Mesa's lavapipe is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplayChoiceChild$", "-test.count=1")
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), choiceChildEnv+"="+backend)
	for k, v := range env {
		if v == "lavapipe" {
			v = icd
		}
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the app failed: %v\n%s", err, out)
	}
	var tried []string
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(line, "TRIED: "); ok {
			tried = append(tried, rest)
		}
		if rest, ok := strings.CutPrefix(line, "CHOSEN: "); ok {
			// The line that chose the renderer, after the lines of those that were given up.
			return strings.Join(append(tried, rest), "\n")
		}
	}
	t.Fatalf("the app did not say which renderer it chose:\n%s", out)
	return ""
}

// The compositor of these tests is headless with the pixman renderer, and the X server is Xvfb, so the GL
// that the app finds is Mesa's llvmpipe, a rasterizer on the CPU: auto does not use it (SPEC §9.6, step
// 32.3) and draws with soft.

func TestDisplayWithoutVulkanAnAppFallsBackToSoftOnSoftwareGL(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "/nonexistent/icd.json", "GLUTTER_RENDERER": "auto"})
	if !strings.Contains(line, "renderer chosen") || !strings.Contains(line, `renderer="soft:`) {
		t.Errorf("the app used %q, want soft", line)
	}
	// What decided: the GLES that auto tried was llvmpipe, and the log says so.
	if !strings.Contains(line, "renderer=gles") || !strings.Contains(line, "llvmpipe") || !strings.Contains(line, "tag=SoftwareRenderer") {
		t.Errorf("the log does not say that gles was given up for llvmpipe:\n%s", line)
	}
}

func TestDisplayAutoNeverUsesACPUVulkanDevice(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "lavapipe", "GLUTTER_RENDERER": "auto"})
	if !strings.Contains(line, `renderer="soft:`) || strings.Contains(line, `renderer="vulkan`) {
		t.Errorf("the app used %q, want soft", line)
	}
}

func TestDisplayGLESByNameStillUsesLlvmpipe(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "/nonexistent/icd.json", "GLUTTER_RENDERER": "gles"})
	if !strings.Contains(line, `renderer="gles: llvmpipe`) || strings.Contains(line, "SoftwareRenderer") {
		t.Errorf("the app used %q, want gles on llvmpipe", line)
	}
}

func TestDisplayVulkanByNameMayUseTheCPUDevice(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"VK_DRIVER_FILES": "lavapipe", "GLUTTER_RENDERER": "vulkan"})
	if !strings.Contains(line, "vulkan: llvmpipe") {
		t.Errorf("the app used %q, want vulkan: llvmpipe", line)
	}
}

func TestDisplaySoftByNameDrawsIntoSharedMemory(t *testing.T) {
	line := chosenRenderer(t, map[string]string{"GLUTTER_RENDERER": "soft"})
	// On Wayland the pixels go through wl_shm, on X11 through MIT-SHM (step 32.2).
	want := "soft: MIT-SHM (X11)"
	if displayBackend() == "wayland" {
		want = "soft: shared memory (wl_shm)"
	}
	if !strings.Contains(line, want) {
		t.Errorf("the app used %q, want %q", line, want)
	}
}

// lavapipeICD is lavapipe's driver file for the architecture of this test, or "" when
// it is not installed. A machine with Mesa's 32-bit drivers too has lvp_icd.i686.json,
// which sorts first and which a 64-bit process cannot load.
func lavapipeICD() string {
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
	file := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
	if _, err := os.Stat(file); err != nil {
		return ""
	}
	return file
}
