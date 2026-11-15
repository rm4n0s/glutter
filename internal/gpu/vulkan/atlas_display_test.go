//go:build linux && glutter_display

package vulkan

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/issues"
)

// TestVulkanDrawsAFrameThatFillsTheAtlas draws, on a real device with validation on, the frame
// that stopped Fishball (step 46.1): shadows whose masks need more tiles at once than the eight
// pages the coverage atlas keeps. It is the reproduction that every platform runs
// (internal/issues): the frame is drawn with nothing left out; a small frame after it frees the
// pages over the eight, whose textures are destroyed; and the first frame again, on pages made
// again, has the pixels it had.
func TestVulkanDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	lavapipe(t)
	var jobs []*drawJob
	for _, sc := range issues.AtlasFullScenes() {
		jobs = append(jobs, &drawJob{name: sc.Name, scene: sc.Scene, width: issues.AtlasFullWidth, height: issues.AtlasFullHeight})
	}
	drawOnWindow(t, jobs)
	var rep issues.AtlasFullReport
	for _, j := range jobs {
		rep.Frames = append(rep.Frames, issues.AtlasFullFrameOf(j.name, j.got, issues.AtlasFullStats{
			TilesRasterized: j.stats.TilesRasterized, AtlasFull: j.stats.AtlasFull, Skipped: j.stats.Skipped, Unsupported: j.stats.Unsupported,
		}))
	}
	dumpImage(t, "vk-many-shadows", jobs[0].got)
	for _, line := range rep.Lines() {
		t.Log(line)
	}
	for _, problem := range rep.Problems() {
		t.Error(problem)
	}
}
