package gles

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// retainedScene builds a scene with the retained layer (or, for the first scene, a
// new one) pushed at the offset, with a rectangle in it, and a rectangle outside it.
func retainedScene(kept *ui.OffsetEngineLayer, dx, dy float64) (*ui.Scene, *ui.OffsetEngineLayer) {
	pic := func(rect ui.Rect, argb uint32) *ui.Picture {
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawRect(rect, solidPaint(argb))
		return rec.EndRecording()
	}
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, pic(ui.NewRectFromLTWH(0, 0, 4, 4), 0xFF0000FF))
	b.PushOffset(dx, dy)
	if kept == nil {
		kept = b.PushOffset(0, 0)
		b.AddPicture(ui.Offset{}, pic(ui.NewRectFromLTWH(0, 0, 10, 10), 0xFFFF0000))
		b.AddPicture(ui.Offset{}, pic(ui.NewRectFromLTWH(5, 5, 10, 10), 0x80FFFFFF))
		b.Pop()
	} else {
		b.AddRetained(kept)
	}
	b.Pop()
	return b.Build(), kept
}

func TestRasterCacheDrawsARetainedLayerFromItsTexture(t *testing.T) {
	r, _, gl := newTestRenderer(RendererOpts{RasterCacheAfter: 2})
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	size := ui.Size{Width: 64, Height: 64}
	render := func(scene *ui.Scene) FrameStats {
		t.Helper()
		gl.resetCalls()
		if err := r.Render(s, scene, size, 1); err != nil {
			t.Fatal(err)
		}
		return r.LastFrame()
	}
	scene, kept := retainedScene(nil, 20, 20)
	// The layer is new: it is drawn as it is, with no group.
	if st := render(scene); st.CachedLayersDrawn != 0 || st.CachedLayersMade != 0 || st.Instances != 3 || gl.calls["genFramebuffer"] != 0 {
		t.Errorf("first frame: %+v, %d framebuffers", st, gl.calls["genFramebuffer"])
	}
	// The second scene has it again, and it is drawn into a framebuffer that the
	// cache keeps: three rectangles and the composite, with the layer's contents
	// drawn into the group.
	second, _ := retainedScene(kept, 20, 20)
	st := render(second)
	if st.CachedLayersMade != 1 || st.CachedLayersDrawn != 0 || st.Instances != 4 || gl.calls["genFramebuffer"] != 1 {
		t.Errorf("second frame: %+v, %d framebuffers", st, gl.calls["genFramebuffer"])
	}
	if ds := r.ctx.dev.Stats(); ds.Framebuffers != 1 || ds.FreeFramebuffers != 0 {
		t.Errorf("the cache should hold the framebuffer: %+v", ds)
	}
	// From the third on, one instance draws the whole layer, and the rectangles
	// inside it are not drawn again; they count as drawn.
	third, _ := retainedScene(kept, 20, 20)
	st = render(third)
	if st.CachedLayersDrawn != 1 || st.CachedLayersMade != 0 || st.Instances != 2 || st.Commands != 3 || st.Drawn != 3 || gl.calls["genFramebuffer"] != 0 {
		t.Errorf("third frame: %+v, %d framebuffers", st, gl.calls["genFramebuffer"])
	}
	if gl.calls["drawArraysInstanced"] != 2 {
		t.Errorf("draws = %d", gl.calls["drawArraysInstanced"])
	}
	// A layer that moves by a whole pixel has other pixels in its contents'
	// transform, so it is drawn again and the entry is replaced.
	moved, _ := retainedScene(kept, 21, 20)
	st = render(moved)
	if st.CachedLayersMade != 1 || st.CachedLayersDrawn != 0 {
		t.Errorf("moved: %+v", st)
	}
	// The new pixels were drawn while the old ones were held; the old framebuffer is
	// back in the pool.
	if ds := r.ctx.dev.Stats(); ds.Framebuffers != 2 || ds.FreeFramebuffers != 1 {
		t.Errorf("one framebuffer is kept and one is free: %+v", ds)
	}
	again, _ := retainedScene(kept, 21, 20)
	if st := render(again); st.CachedLayersDrawn != 1 {
		t.Errorf("after the move: %+v", st)
	}
	// A scene without it loses the run: it is not cached again before two scenes.
	gone, _ := retainedScene(nil, 21, 20)
	render(gone)
	back, _ := retainedScene(kept, 21, 20)
	if st := render(back); st.CachedLayersDrawn != 0 || st.CachedLayersMade != 0 {
		t.Errorf("after a scene without it: %+v", st)
	}
	// Closing returns the framebuffer and deletes everything.
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if gl.calls["deleteFramebuffer"] == 0 {
		t.Error("the cached framebuffer was not deleted")
	}
}

func TestRasterCacheIsOffByDefaultAndForSnapshots(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, _ := r.Attach(waylandHandle)
	scene, kept := retainedScene(nil, 20, 20)
	for i := 0; i < 4; i++ {
		if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
			t.Fatal(err)
		}
		scene, _ = retainedScene(kept, 20, 20)
		if st := r.LastFrame(); st.CachedLayersDrawn != 0 || st.CachedLayersMade != 0 || gl.calls["genFramebuffer"] != 0 {
			t.Fatalf("frame %d: %+v", i, st)
		}
	}
	// Snapshots do not count as frames of the cache.
	on, _, _ := newTestRenderer(RendererOpts{RasterCacheAfter: 1})
	if _, err := on.Attach(waylandHandle); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := on.Snapshot(scene, ui.Size{Width: 64, Height: 64}); err != nil {
			t.Fatal(err)
		}
		if st := on.LastFrame(); st.CachedLayersMade != 0 || st.CachedLayersDrawn != 0 {
			t.Fatalf("snapshot %d: %+v", i, st)
		}
	}
}

// TestRasterCacheRuleOfThreeFramesNeverRasterizesACachedLayerAgain is the rule the
// engine uses (SPEC §9.1): a retained layer is drawn as it is for two frames, drawn
// into a texture in the third, and from the fourth on one instance draws it; the
// coverage tiles of the path inside it are made once, and the layer's contents are not
// walked again.
func TestRasterCacheRuleOfThreeFramesNeverRasterizesACachedLayerAgain(t *testing.T) {
	r, _, gl := newTestRenderer(RendererOpts{RasterCacheAfter: 3})
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	star := ui.NewPath()
	star.MoveTo(10, 0)
	star.LineTo(13, 8)
	star.LineTo(22, 8)
	star.LineTo(15, 13)
	star.LineTo(18, 22)
	star.LineTo(10, 16)
	star.LineTo(2, 22)
	star.LineTo(5, 13)
	star.LineTo(-2, 8)
	star.LineTo(7, 8)
	star.Close()
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawPath(star, solidPaint(0xFF00AA44))
	inside := rec.EndRecording()
	sceneWith := func(kept *ui.OffsetEngineLayer) (*ui.Scene, *ui.OffsetEngineLayer) {
		b := ui.NewSceneBuilder()
		if kept == nil {
			kept = b.PushOffset(5, 5)
			b.AddPicture(ui.Offset{}, inside)
			b.Pop()
		} else {
			b.AddRetained(kept)
		}
		return b.Build(), kept
	}
	type frame struct {
		tiles, made, drawn, instances, framebuffers int
	}
	var frames []frame
	scene, kept := sceneWith(nil)
	for i := 0; i < 8; i++ {
		gl.resetCalls()
		if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
			t.Fatal(err)
		}
		st := r.LastFrame()
		frames = append(frames, frame{st.TilesRasterized, st.CachedLayersMade, st.CachedLayersDrawn, st.Instances, gl.calls["genFramebuffer"]})
		scene, _ = sceneWith(kept)
	}
	if frames[0].tiles == 0 {
		t.Fatalf("the path made no coverage tiles in the first frame: %+v", frames[0])
	}
	for i, f := range frames {
		switch {
		case i < 2: // drawn as it is
			if f.made != 0 || f.drawn != 0 {
				t.Errorf("frame %d: %+v, want it drawn as it is", i+1, f)
			}
		case i == 2: // the third frame makes the texture
			if f.made != 1 || f.drawn != 0 || f.framebuffers != 1 {
				t.Errorf("frame 3: %+v, want the texture made", f)
			}
		default: // one instance, no tiles, no new framebuffer
			if f.made != 0 || f.drawn != 1 || f.tiles != 0 || f.framebuffers != 0 || f.instances != 1 {
				t.Errorf("frame %d: %+v, want the layer drawn from its texture alone", i+1, f)
			}
		}
		if i > 0 && f.tiles != 0 {
			t.Errorf("frame %d made %d coverage tiles: the path was rasterized again", i+1, f.tiles)
		}
	}
}
