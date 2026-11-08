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
	if got := strings.Join(names, ","); got != "vulkan,gles,soft" {
		t.Errorf("candidates %q, want vulkan,gles,soft (SPEC §9.6)", got)
	}
}

// TestSoftIsTheLastRendererOfAutoOnLinux checks that auto tries soft after vulkan and gles (step 32.3), and
// that a name chooses it alone.
func TestSoftIsTheLastRendererOfAutoOnLinux(t *testing.T) {
	cfg := &Config{}
	auto, err := newChooser(cfg, quiet(), "auto", rendererCandidates())
	if err != nil {
		t.Fatal(err)
	}
	last := auto.cand[len(auto.cand)-1]
	if last.Name != "soft" || (last.Skip != nil && last.Skip(cfg)) {
		t.Errorf("the last candidate of auto is %+v, want soft and not skipped", last)
	}
	named, err := newChooser(cfg, quiet(), "soft", rendererCandidates())
	if err != nil {
		t.Fatal(err)
	}
	if len(named.cand) != 1 || named.cand[0].Name != "soft" || !named.named {
		t.Errorf("candidates of the name soft: %+v, named %v", named.cand, named.named)
	}
}

// TestGLESRejectsASoftwareRasterizerOnlyInAuto checks that the gles of Linux gives up on llvmpipe and the
// like when it was not asked for by name, so that auto goes on to soft (step 32.3).
func TestGLESRejectsASoftwareRasterizerOnlyInAuto(t *testing.T) {
	for _, named := range []bool{false, true} {
		cfg := &Config{rendererNamed: named}
		r, ok := newGLESRenderer(cfg, quiet()).(*glesRenderer)
		if !ok {
			t.Fatalf("newGLESRenderer made a %T", newGLESRenderer(cfg, quiet()))
		}
		if r.rejectSoftware == named {
			t.Errorf("rendererNamed = %v: rejectSoftware = %v", named, r.rejectSoftware)
		}
	}
}

// TestAutoGoesFromVulkanAndGLESToSoft checks the choice and its log: a Vulkan with only a CPU device and a
// GLES on llvmpipe both fail with their own tags, soft is chosen, and the log says what decided each.
func TestAutoGoesFromVulkanAndGLESToSoft(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	cands, made := fakeCandidates("a", "b")
	cands[0].Name, cands[1].Name, cands[2].Name = "vulkan", "gles", "soft"
	log, buf := recordingLog()
	c, err := newChooser(&Config{}, log, "auto", cands)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := c.Describe(); !strings.Contains(got, "c: device of") {
		t.Errorf("auto chose %q, want the last candidate", got)
	}
	if len(*made) != 3 {
		t.Errorf("%d renderers were made, want 3", len(*made))
	}
	out := buf.String()
	for _, want := range []string{"renderer not usable, trying the next", "renderer=vulkan", "renderer=gles", "renderer chosen"} {
		if !strings.Contains(out, want) {
			t.Errorf("the log lacks %q:\n%s", want, out)
		}
	}
}

// TestOnlySoftByNameMakesPixelWindows checks which choices have the X11 windows made for pixels (step
// 32.2): the name soft, and no other choice, so that the others still ask EGL for a visual.
func TestOnlySoftByNameMakesPixelWindows(t *testing.T) {
	tests := []struct {
		want string
		soft bool
	}{
		{"soft", true},
		{"", false},
		{"auto", false},
		{"gles", false},
		{"vulkan", false},
	}
	for _, tt := range tests {
		t.Run("renderer "+tt.want, func(t *testing.T) {
			c, err := newChooser(&Config{}, quiet(), tt.want, rendererCandidates())
			if err != nil {
				t.Fatal(err)
			}
			if got := c.asksFor("soft"); got != tt.soft {
				t.Errorf("asksFor(soft) = %v, want %v", got, tt.soft)
			}
		})
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
