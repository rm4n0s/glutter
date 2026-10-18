package gles

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// The raster cache (SPEC §9.1): a layer that a scene retains (the same layer in
// consecutive scenes, which is what SceneBuilder.AddRetained does) is drawn once into a
// texture, and the frames after it draw the texture instead of the layer's contents.
// This is the groundwork for RepaintBoundary (step 17.2 sets the rule that decides
// what is cached and when); here the rule is only how many consecutive scenes a layer
// must be in, and zero turns the cache off.
//
// A layer that qualifies becomes a group of the draw list (gpu.FlattenOpts.Boundary),
// drawn into a framebuffer of the pixels its clip's bounds cover, under a clip of whole
// pixels, so the pixels depend only on the transform of the layer's contents and on
// those bounds. The group is composited under the clip it is in, so the clip's shapes
// and fractional edges are applied to the group as a whole, as with a save layer, and
// a change in them does not make the cache miss. The framebuffer is kept, not
// returned to the pool, until the entry is evicted.

const (
	// layerCacheBytes is how much the entries may hold, in bytes of pixels.
	layerCacheBytes = 64 << 20
	// layerCacheIdleFrames is how many frames an entry may go unused.
	layerCacheIdleFrames = 120
)

// layerKey is what the pixels of a cached layer depend on.
type layerKey struct {
	// m is the transform of the layer's contents.
	m [16]float32
	// rect is the pixels of the surface that the group covers.
	rect image.Rectangle
}

// layerEntry is a layer's pixels.
type layerEntry struct {
	layerKey
	fb *Framebuffer
	// used is the frame that last drew it.
	used uint64
}

func (e *layerEntry) bytes() int { return e.rect.Dx() * e.rect.Dy() * 4 }

// layerSeen counts the consecutive scenes a layer was in.
type layerSeen struct {
	last uint64
	run  int
}

// layerCache holds the pixels of retained layers.
type layerCache struct {
	// after is how many consecutive scenes a layer must be in to be cached.
	after   int
	frame   uint64
	root    ui.Layer
	seen    map[ui.Layer]*layerSeen
	entries map[ui.Layer]*layerEntry
	bytes   int
}

func newLayerCache(after int) *layerCache {
	return &layerCache{after: after, seen: map[ui.Layer]*layerSeen{}, entries: map[ui.Layer]*layerEntry{}}
}

// beginFrame starts a frame of the scene whose root layer is given; the root is never
// cached.
func (c *layerCache) beginFrame(root ui.Layer) {
	c.frame++
	c.root = root
}

// boundary says whether the layer is cached: one of the layers that the draw list
// folds, that is not the root, and that has been in the last c.after scenes.
func (c *layerCache) boundary(l ui.Layer) bool {
	if l == c.root {
		return false
	}
	e := c.seen[l]
	switch {
	case e == nil:
		e = &layerSeen{last: c.frame, run: 1}
		c.seen[l] = e
	case e.last == c.frame:
		// The same layer twice in a scene counts once.
	case e.last == c.frame-1:
		e.run++
		e.last = c.frame
	default:
		e.run, e.last = 1, c.frame
	}
	return e.run >= c.after
}

// lookup returns the layer's entry if it has the pixels for the key, and marks it
// used in this frame.
func (c *layerCache) lookup(l ui.Layer, k layerKey) *layerEntry {
	e := c.entries[l]
	if e == nil || e.layerKey != k {
		return nil
	}
	e.used = c.frame
	return e
}

// put keeps the framebuffer as the layer's pixels, and says whether it did. An entry
// that a draw of this frame still waits for is not replaced.
func (c *layerCache) put(dev *Device, l ui.Layer, k layerKey, fb *Framebuffer) bool {
	if old := c.entries[l]; old != nil {
		if old.used == c.frame {
			return false
		}
		c.drop(dev, l, old)
	}
	e := &layerEntry{layerKey: k, fb: fb, used: c.frame}
	c.entries[l] = e
	c.bytes += e.bytes()
	return true
}

func (c *layerCache) drop(dev *Device, l ui.Layer, e *layerEntry) {
	delete(c.entries, l)
	c.bytes -= e.bytes()
	dev.ReleaseFramebuffer(e.fb)
}

// endFrame forgets the layers that were not in the scene, and evicts the entries that
// have not been used for a while, then the least recently used ones while the cache
// holds more than its budget.
func (c *layerCache) endFrame(dev *Device) {
	for l, s := range c.seen {
		if s.last != c.frame {
			delete(c.seen, l)
		}
	}
	for l, e := range c.entries {
		if c.frame-e.used > layerCacheIdleFrames {
			c.drop(dev, l, e)
		}
	}
	for c.bytes > layerCacheBytes {
		var oldest ui.Layer
		var oe *layerEntry
		for l, e := range c.entries {
			if oe == nil || e.used < oe.used {
				oldest, oe = l, e
			}
		}
		if oe == nil || oe.used == c.frame {
			return
		}
		c.drop(dev, oldest, oe)
	}
}

// release returns every framebuffer to the pool.
func (c *layerCache) release(dev *Device) {
	for l, e := range c.entries {
		c.drop(dev, l, e)
	}
}

// keyOf makes the key of a layer's group from the matrix of its contents and the
// pixels it covers.
func keyOf(m *ui.Matrix4, rect image.Rectangle) layerKey {
	k := layerKey{rect: rect}
	copy(k.m[:], m.Storage()[:])
	return k
}
