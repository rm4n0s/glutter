//go:build glutter_display

package gles

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// shadowsScene draws n shadows of panels that fill a surface of 1920 x 1080 pixels, each of a
// size of its own, so that no two share their tiles of the coverage atlas.
func shadowsScene(n int) *ui.Scene {
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
		c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
		for i := 0; i < n; i++ {
			path := ui.NewPath()
			path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20+float64(i), 20+float64(i), 1800-float64(7*i), 1000-float64(5*i)), 12, 12))
			c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false)
		}
	}))
	return b.Build()
}

// TestDisplayDrawsAFrameThatFillsTheAtlas draws, with a real OpenGL ES context, the frame that
// stopped Fishball (step 46.1): shadows whose masks need more tiles at once than the eight
// pages the coverage atlas keeps. The frame is drawn with nothing left out; a small frame after
// it frees the pages over the eight, whose textures go back to the device; and the first frame
// again, on pages made again, has the pixels it had.
func TestDisplayDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	const w, h = 1920, 1080
	big, small := shadowsScene(10), shadowsScene(1)
	onDisplay(t, func(handle gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(handle); err != nil {
			return err
		}
		frames := []struct {
			name  string
			scene *ui.Scene
			over  bool // the atlas has more pages than it keeps after the frame
			same  bool // the pixels are those of the first frame
		}{
			{"many shadows", big, true, true},
			{"many shadows again", big, true, true},
			{"one shadow", small, false, false},
			{"many shadows on new pages", big, true, true},
		}
		var first []byte
		for i, fr := range frames {
			got, err := renderGPU(r, fr.scene, w, h)
			if err != nil {
				t.Errorf("%s: the frame failed: %v", fr.name, err)
				return nil
			}
			st, atlas := r.LastFrame(), r.res.masks.Atlas().Stats()
			t.Logf("%s: %d tiles rasterized, %d pages", fr.name, st.TilesRasterized, atlas.Pages)
			if st.AtlasFull != 0 || st.Skipped != 0 || len(st.Unsupported) != 0 {
				t.Errorf("%s: the frame left something out: %+v", fr.name, st)
			}
			if (atlas.PagesOverBudget > 0) != fr.over || atlas.Pages > gpu.AtlasFramePages {
				t.Errorf("%s: the atlas has %d pages, %d over what it keeps", fr.name, atlas.Pages, atlas.PagesOverBudget)
			}
			textures := 0
			for _, p := range r.res.pages {
				if p != nil {
					textures++
				}
			}
			if textures != atlas.Pages {
				t.Errorf("%s: %d page textures for %d pages", fr.name, textures, atlas.Pages)
			}
			if i == 0 {
				first = bytes.Clone(got.Pix)
				dumpImage(t, "gles-many-shadows", got)
				// The shadows are there: a pixel just under the largest panel is darker than the
				// white around it.
				if c := got.RGBAAt(1000, 1024); c.R > 250 {
					t.Errorf("no shadow under the panels: %v", c)
				}
			}
			if i == 1 && st.TilesRasterized != 0 {
				t.Errorf("%s: %d tiles rasterized", fr.name, st.TilesRasterized)
			}
			if fr.same && !bytes.Equal(got.Pix, first) {
				t.Errorf("%s: the pixels are not those of the first frame", fr.name)
			}
		}
		return nil
	})
}
