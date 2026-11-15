//go:build windows && glutter_wine

package d3d12

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestWineDrawsAFrameThatFillsTheAtlas draws, on a Direct3D 12 device (Wine's vkd3d here, the
// PC's own on Windows), the frame that stopped Fishball (step 46.1): shadows whose masks need
// more tiles at once than the eight pages the coverage atlas keeps. It is the reproduction that
// every platform runs (internal/issues). The renderer is strict, so a frame that the full atlas
// failed, as it did before the step, fails the test in Draw. The frame is drawn with nothing
// left out; a small frame after it frees the pages over the eight, whose textures are
// destroyed; and the first frame again, on pages made again, has the pixels it had.
func TestWineDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	g := newRig(t, Options{}, func(o *explicit.RendererOpts) { o.SkipUnsupported = false })
	rep, first, err := issues.RunAtlasFull(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.AtlasFullStats, error) {
		j := g.draw(scene, w, h, false)
		return j.got, issues.AtlasFullStats{
			TilesRasterized: j.stats.TilesRasterized, AtlasFull: j.stats.AtlasFull, Skipped: j.stats.Skipped, Unsupported: j.stats.Unsupported,
		}, nil
	})
	if err != nil {
		t.Fatalf("the frames: %+v", err)
	}
	dumpImage(t, "d3d12-many-shadows", first)
	for _, line := range rep.Lines() {
		t.Log(line)
	}
	for _, problem := range rep.Problems() {
		t.Error(problem)
	}
}
