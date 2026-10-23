//go:build linux && !android

package engine

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

func TestLinuxAutoTriesVulkanThenGLES(t *testing.T) {
	var names []string
	for _, c := range rendererCandidates() {
		names = append(names, c.Name)
	}
	if got := strings.Join(names, ","); got != "vulkan,gles" {
		t.Errorf("candidates %q, want vulkan,gles (SPEC §9.6)", got)
	}
}

// noVulkan is a Config whose Vulkan library cannot be loaded.
func noVulkan() *Config {
	return &Config{Getenv: func(k string) string {
		if k == "GLUTTER_VULKAN_LIBRARY" {
			return "/nonexistent/libvulkan.so.1"
		}
		return ""
	}}
}

func TestWithoutVulkanAutoUsesGLESAndLogsTheFallbackOnce(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	fake, made := fakeCandidates()
	cands := []rendererCandidate{rendererCandidates()[0], fake[1]} // the real vulkan, then a gles that works
	log, buf := recordingLog()
	c, err := newChooser(noVulkan(), log, "auto", cands)
	if err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 3; frame++ {
		if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
			t.Fatalf("frame %d: %v", frame, err)
		}
	}
	if got := c.Describe(); !strings.HasPrefix(got, "b:") {
		t.Errorf("drawn by %q, want the gles", got)
	}
	if len(*made) != 1 {
		t.Errorf("%d fake renderers made", len(*made))
	}
	if n := strings.Count(buf.String(), "renderer not usable"); n != 1 {
		t.Errorf("the fallback was logged %d times:\n%s", n, buf.String())
	}
	if !strings.Contains(buf.String(), "tag=VulkanLoadFailed") || !strings.Contains(buf.String(), "renderer=vulkan") {
		t.Errorf("the log lacks the renderer or its tag:\n%s", buf.String())
	}
}

func TestVulkanByNameWithoutVulkanIsAnError(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	cfg := noVulkan()
	c, err := newChooser(cfg, nil, "vulkan", rendererCandidates())
	if err != nil {
		t.Fatal(err)
	}
	cfg.rendererNamed = c.named
	_, err = c.Draw(handle, 10, 10, nil, false)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "VulkanLoadFailed" || !appErr.HasRoute("VulkanLoadFailed") {
		t.Fatalf("got %v, want VulkanLoadFailed", err)
	}
}
