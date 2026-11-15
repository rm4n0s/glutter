//go:build linux && glutter_display

package vulkan

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// shadowsScene draws n shadows of panels that fill a surface of 1920 x 1080 pixels, each of a
// size of its own, so that no two share their tiles of the coverage atlas.
func shadowsScene(n int) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	for i := 0; i < n; i++ {
		path := ui.NewPath()
		path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20+float64(i), 20+float64(i), 1800-float64(7*i), 1000-float64(5*i)), 12, 12))
		c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false)
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}

// TestVulkanDrawsAFrameThatFillsTheAtlas draws, on a real device with validation on, the frame
// that stopped Fishball (step 46.1): shadows whose masks need more tiles at once than the eight
// pages the coverage atlas keeps. The frame is drawn with nothing left out; a small frame after
// it frees the pages over the eight, whose textures are destroyed; and the first frame again,
// on pages made again, has the pixels it had.
func TestVulkanDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	lavapipe(t)
	const w, h = 1920, 1080
	big, small := shadowsScene(10), shadowsScene(1)
	jobs := []*drawJob{
		{name: "many shadows", scene: big, width: w, height: h},
		{name: "many shadows again", scene: big, width: w, height: h},
		{name: "one shadow", scene: small, width: w, height: h},
		{name: "many shadows on new pages", scene: big, width: w, height: h},
	}
	drawOnWindow(t, jobs)
	for _, j := range jobs {
		if j.stats.AtlasFull != 0 || j.stats.Skipped != 0 || len(j.stats.Unsupported) != 0 {
			t.Errorf("%s: the frame left something out: %+v", j.name, j.stats)
		}
	}
	// More tiles than the pages the atlas keeps hold when they are all of the largest slot.
	slots := gpu.AtlasPageSize / gpu.MaxAtlasTile
	if most := gpu.AtlasMaxPages * slots * slots; jobs[0].stats.TilesRasterized <= most {
		t.Fatalf("the frame rasterized %d tiles, which fit the %d of the pages the atlas keeps: it is not the frame of the issue", jobs[0].stats.TilesRasterized, most)
	}
	if n := jobs[1].stats.TilesRasterized; n != 0 {
		t.Errorf("the same frame again rasterized %d tiles", n)
	}
	// The small frame freed pages, so some tiles of the large one are made again.
	if n := jobs[3].stats.TilesRasterized; n == 0 || n >= jobs[0].stats.TilesRasterized {
		t.Errorf("the frame after the small one rasterized %d tiles, the first %d", n, jobs[0].stats.TilesRasterized)
	}
	dumpImage(t, "vk-many-shadows", jobs[0].got)
	for _, i := range []int{1, 3} {
		if !bytes.Equal(jobs[i].got.Pix, jobs[0].got.Pix) {
			t.Errorf("%s: the pixels are not those of the first frame", jobs[i].name)
		}
	}
	// The shadows are there: a pixel of the first frame just outside the largest panel is darker
	// than the white background.
	if c := jobs[0].got.RGBAAt(1000, 1024); c.R > 250 {
		t.Errorf("no shadow under the panels: %v", c)
	}
}
