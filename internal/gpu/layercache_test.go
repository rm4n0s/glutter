package gpu

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

type fakeTex struct{}

func release(*fakeTex) {}

func TestLayerCacheEviction(t *testing.T) {
	c := NewLayerCache[*fakeTex](1)
	root := &ui.ContainerLayer{}
	layers := []ui.Layer{&ui.TransformLayer{}, &ui.TransformLayer{}, &ui.TransformLayer{}}
	key := func(w, h int) LayerKey { return LayerKeyOf(ui.NewMatrix4Identity(), image.Rect(0, 0, w, h)) }
	put := func(l ui.Layer, w, h int) {
		t.Helper()
		c.Put(l, key(w, h), &fakeTex{}, release)
	}
	c.BeginFrame(root)
	put(layers[0], 8, 8)
	put(layers[1], 16, 16)
	// An entry used in this frame is kept when another is put for its layer: a draw that
	// waits for it must still find its pixels.
	if c.Put(layers[0], key(4, 4), &fakeTex{}, release) {
		t.Error("an entry used in this frame was replaced")
	}
	if c.Lookup(layers[0], key(8, 8)) == nil || c.Lookup(layers[0], key(8, 9)) != nil {
		t.Error("lookup must match the key exactly")
	}
	c.EndFrame(release)
	if c.bytes != (8*8+16*16)*4 || len(c.entries) != 2 {
		t.Fatalf("bytes %d, entries %d", c.bytes, len(c.entries))
	}
	// A new frame: another entry for a layer that has none, and the first is replaced
	// by a new one now that it is not in use.
	c.BeginFrame(root)
	put(layers[0], 4, 4)
	if c.bytes != (4*4+16*16)*4 {
		t.Errorf("bytes after replacing = %d", c.bytes)
	}
	c.EndFrame(release)
	// Entries that have not been used for a long time go.
	for i := 0; i < LayerCacheIdleFrames; i++ {
		c.BeginFrame(root)
		c.Lookup(layers[0], key(4, 4))
		c.EndFrame(release)
	}
	if len(c.entries) != 1 || c.entries[layers[0]] == nil {
		t.Errorf("entries after the idle frames: %d", len(c.entries))
	}
	// Over the budget, the least recently used entries go first, but never the ones the
	// frame drew.
	c.BeginFrame(root)
	big := 3000
	put(layers[1], big, big)
	put(layers[2], big, big)
	c.EndFrame(release)
	// Over the budget, the entry that was not used in this frame goes, and the two that
	// the frame drew stay, though they are over the budget together.
	if len(c.entries) != 2 || c.entries[layers[0]] != nil {
		t.Errorf("entries after the big frame: %d", len(c.entries))
	}
	c.BeginFrame(root)
	c.Lookup(layers[2], key(big, big))
	c.EndFrame(release)
	if len(c.entries) != 1 || c.entries[layers[2]] == nil {
		t.Errorf("entries = %v", c.entries)
	}
	c.Release(release)
	if len(c.entries) != 0 || c.bytes != 0 {
		t.Errorf("after release: %d entries, %d bytes", len(c.entries), c.bytes)
	}
}

func TestLayerCacheBoundary(t *testing.T) {
	c := NewLayerCache[*fakeTex](3)
	root := &ui.ContainerLayer{}
	l := &ui.TransformLayer{}
	other := &ui.TransformLayer{}
	var got []bool
	frame := func(layers ...ui.Layer) {
		c.BeginFrame(root)
		for _, l := range layers {
			got = append(got, c.Boundary(l))
		}
		c.EndFrame(release)
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
