//go:build linux && glutter_display

package vulkan

import (
	"image"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
)

// The paint shaders, color filters, blend modes and layers of step 20.5 through the explicit
// renderer on Vulkan, compared with the software renderer as the tests of gles are.

// blendModeNames are the 29 modes in the order of BlendMode, as the cells of the blend-modes
// scene are.
var blendModeNames = []string{
	"clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn", "srcOut", "dstOut", "srcATop",
	"dstATop", "xor", "plus", "modulate", "screen", "overlay", "darken", "lighten", "colorDodge", "colorBurn",
	"hardLight", "softLight", "difference", "exclusion", "multiply", "hue", "saturation", "color", "luminosity",
}

func TestVulkanGradientAndBlendScenesMatchSoftware(t *testing.T) {
	lavapipe(t)
	var jobs []*drawJob
	for _, name := range []string{"gradients", "blend-modes"} {
		sc, err := proof.Lookup(name)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, &drawJob{name: name, scene: sc.Build(), width: sc.Width, height: sc.Height})
	}
	drawOnWindow(t, jobs)
	for _, j := range jobs {
		t.Run(j.name, func(t *testing.T) {
			want, err := soft.Render(j.scene, j.width, j.height)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "vk-"+j.name+"-gpu", j.got)
			dumpImage(t, "vk-"+j.name+"-soft", want)
			t.Logf("frame: %+v", j.stats)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			if j.name == "gradients" {
				gputest.Tolerance{Over8Permille: 10, Over32Permille: 3}.Check(t, j.name, gputest.DiffRect(j.got, want, j.got.Bounds()))
				return
			}
			for i, mode := range blendModeNames {
				cell := image.Rect(i%6*106, i/6*96, i%6*106+106, i/6*96+96)
				gputest.Tolerance{Over8Permille: 15, Over32Permille: 5}.Check(t, mode, gputest.DiffRect(j.got, want, cell))
			}
		})
	}
}

func TestVulkanEffectScenesMatchSoftware(t *testing.T) {
	lavapipe(t)
	scenes := gputest.EffectScenes()
	jobs := make([]*drawJob, len(scenes))
	for i, sc := range scenes {
		jobs[i] = &drawJob{name: sc.Name, scene: sc.Build(), width: gputest.EffectSize, height: gputest.EffectSize}
	}
	drawOnWindow(t, jobs)
	for i, sc := range scenes {
		j := jobs[i]
		t.Run(sc.Name, func(t *testing.T) {
			want, err := soft.Render(j.scene, j.width, j.height)
			if err != nil {
				t.Fatal(err)
			}
			slug := strings.NewReplacer(" ", "-", ",", "", ":", "").Replace(sc.Name)
			dumpImage(t, "vk-"+slug+"-gpu", j.got)
			dumpImage(t, "vk-"+slug+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance.Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}
