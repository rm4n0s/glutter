package gles

import (
	"image"
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

func TestLayerCacheEviction(t *testing.T) {
	_, dev, _ := newTestResources(t)
	c := newLayerCache(1)
	root := &ui.ContainerLayer{}
	layers := []ui.Layer{&ui.TransformLayer{}, &ui.TransformLayer{}, &ui.TransformLayer{}}
	key := func(w, h int) layerKey { return keyOf(ui.NewMatrix4Identity(), image.Rect(0, 0, w, h)) }
	put := func(l ui.Layer, w, h int) {
		t.Helper()
		fb, err := dev.AcquireFramebuffer(w, h, false)
		if err != nil {
			t.Fatal(err)
		}
		if !c.put(dev, l, key(w, h), fb) {
			dev.ReleaseFramebuffer(fb)
		}
	}
	c.beginFrame(root)
	put(layers[0], 8, 8)
	put(layers[1], 16, 16)
	// An entry used in this frame is kept when another is put for its layer: a draw that
	// waits for it must still find its pixels.
	fb, _ := dev.AcquireFramebuffer(4, 4, false)
	if c.put(dev, layers[0], key(4, 4), fb) {
		t.Error("an entry used in this frame was replaced")
	}
	dev.ReleaseFramebuffer(fb)
	if c.lookup(layers[0], key(8, 8)) == nil || c.lookup(layers[0], key(8, 9)) != nil {
		t.Error("lookup must match the key exactly")
	}
	c.endFrame(dev)
	if c.bytes != (8*8+16*16)*4 || len(c.entries) != 2 {
		t.Fatalf("bytes %d, entries %d", c.bytes, len(c.entries))
	}
	// A new frame: another entry for a layer that has none, and the first is replaced
	// by a new one now that it is not in use.
	c.beginFrame(root)
	put(layers[0], 4, 4)
	if c.bytes != (4*4+16*16)*4 {
		t.Errorf("bytes after replacing = %d", c.bytes)
	}
	c.endFrame(dev)
	// Entries that have not been used for a long time go.
	for i := 0; i < layerCacheIdleFrames; i++ {
		c.beginFrame(root)
		c.lookup(layers[0], key(4, 4))
		c.endFrame(dev)
	}
	if len(c.entries) != 1 || c.entries[layers[0]] == nil {
		t.Errorf("entries after the idle frames: %d", len(c.entries))
	}
	// Over the budget, the least recently used entries go first, but never the ones the
	// frame drew.
	c.beginFrame(root)
	big := 3000
	put(layers[1], big, big)
	put(layers[2], big, big)
	c.endFrame(dev)
	// Over the budget, the entry that was not used in this frame goes, and the two that
	// the frame drew stay, though they are over the budget together.
	if len(c.entries) != 2 || c.entries[layers[0]] != nil {
		t.Errorf("entries after the big frame: %d", len(c.entries))
	}
	c.beginFrame(root)
	c.lookup(layers[2], key(big, big))
	c.endFrame(dev)
	if len(c.entries) != 1 || c.entries[layers[2]] == nil {
		t.Errorf("entries = %v", c.entries)
	}
	c.release(dev)
	if len(c.entries) != 0 || c.bytes != 0 {
		t.Errorf("after release: %d entries, %d bytes", len(c.entries), c.bytes)
	}
}

func TestLayerCacheBoundary(t *testing.T) {
	c := newLayerCache(3)
	root := &ui.ContainerLayer{}
	l := &ui.TransformLayer{}
	other := &ui.TransformLayer{}
	var got []bool
	frame := func(layers ...ui.Layer) {
		c.beginFrame(root)
		for _, l := range layers {
			got = append(got, c.boundary(l))
		}
		c.endFrame(nil)
	}
	frame(root, l, l) // the root is never cached; a layer twice in a scene counts once
	frame(l, other)
	frame(l)
	frame(other) // l was missing: its run starts again, and other's too
	frame(l)
	want := []bool{false, false, false, false, false, true, false, false}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("answers = %v, want %v", got, want)
			break
		}
	}
}
