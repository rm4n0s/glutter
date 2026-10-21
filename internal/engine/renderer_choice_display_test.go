//go:build linux && glutter_display

package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// These tests run the engine in a headless Wayland compositor (scripts/headless-wayland.sh) and
// check which renderer it chooses (SPEC §9.6, step 20.7): Vulkan where it works, GLES where it
// does not, and never a CPU Vulkan device unless Vulkan was asked for by name.

// chosenRenderer runs an app that draws one frame, saves it and exits, and returns what the log
// says about the renderer.
func chosenRenderer(t *testing.T, env map[string]string) string {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	files, _ := filepath.Glob("/usr/share/vulkan/icd.d/lvp_icd.*.json")
	if len(files) == 0 {
		t.Skip("Mesa's lavapipe is not installed")
	}
	for k, v := range env {
		if v == "lavapipe" {
			v = files[0]
		}
		t.Setenv(k, v)
	}
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
			return line
		}
	}
	t.Fatalf("the log has no \"renderer chosen\" line:\n%s", buf.String())
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
