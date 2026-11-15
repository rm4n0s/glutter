//go:build glutter_display

package gles

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplayDrawsAFrameThatFillsTheAtlas draws, with a real OpenGL ES context, the frame that
// stopped Fishball (step 46.1): shadows whose masks need more tiles at once than the eight
// pages the coverage atlas keeps. It is the reproduction that every platform runs
// (internal/issues): the frame is drawn with nothing left out; a small frame after it frees the
// pages over the eight, whose textures go back to the device; and the first frame again, on
// pages made again, has the pixels it had. Here the pages and their textures are counted too.
func TestDisplayDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	onDisplay(t, func(handle gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(handle); err != nil {
			return err
		}
		// Whether the atlas has more pages than it keeps after each frame.
		over := []bool{true, true, false, true}
		frame := 0
		rep, first, err := issues.RunAtlasFull(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.AtlasFullStats, error) {
			got, err := renderGPU(r, scene, w, h)
			if err != nil {
				return nil, issues.AtlasFullStats{}, err
			}
			st, atlas := r.LastFrame(), r.res.masks.Atlas().Stats()
			t.Logf("frame %d: %d tiles rasterized, %d pages", frame, st.TilesRasterized, atlas.Pages)
			if (atlas.PagesOverBudget > 0) != over[frame] || atlas.Pages > gpu.AtlasFramePages {
				t.Errorf("frame %d: the atlas has %d pages, %d over what it keeps", frame, atlas.Pages, atlas.PagesOverBudget)
			}
			textures := 0
			for _, p := range r.res.pages {
				if p != nil {
					textures++
				}
			}
			if textures != atlas.Pages {
				t.Errorf("frame %d: %d page textures for %d pages", frame, textures, atlas.Pages)
			}
			frame++
			return got, issues.AtlasFullStats{
				TilesRasterized: st.TilesRasterized, AtlasFull: st.AtlasFull, Skipped: st.Skipped, Unsupported: st.Unsupported,
			}, nil
		})
		if err != nil {
			t.Errorf("the frames: %+v", err)
			return nil
		}
		dumpImage(t, "gles-many-shadows", first)
		for _, line := range rep.Lines() {
			t.Log(line)
		}
		for _, problem := range rep.Problems() {
			t.Error(problem)
		}
		return nil
	})
}
