package gpu

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// The raster cache (SPEC §9.1): a layer that a scene retains (the same layer in consecutive
// scenes, which is what SceneBuilder.AddRetained does) is drawn once into a texture, and the
// frames after it draw the texture instead of the layer's contents. This is the groundwork for
// RepaintBoundary (step 17.2 sets the rule that decides what is cached and when); here the rule
// is only how many consecutive scenes a layer must be in, and zero turns the cache off.
//
// A layer that qualifies becomes a group of the draw list (FlattenOpts.Boundary), drawn into a
// texture of the pixels its clip's bounds cover, under a clip of whole pixels, so the pixels
// depend only on the transform of the layer's contents and on those bounds. The group is
// composited under the clip it is in, so the clip's shapes and fractional edges are applied to
// the group as a whole, as with a save layer, and a change in them does not make the cache miss.
// The texture is kept, not returned to the pool, until the entry is evicted.
//
// The cache is the policy only: T is the renderer's texture, and the functions that release
// textures are given by the renderer.

const (
	// LayerCacheBytes is how much the entries may hold, in bytes of pixels.
	//
	// Flutter: no counterpart
	LayerCacheBytes = 64 << 20
	// LayerCacheIdleFrames is how many frames an entry may go unused.
	//
	// Flutter: no counterpart
	LayerCacheIdleFrames = 120
)

// LayerKey is what the pixels of a cached layer depend on.
//
// Flutter: no counterpart
type LayerKey struct {
	// M is the transform of the layer's contents.
	M [16]float32
	// Rect is the pixels of the surface that the group covers.
	Rect image.Rectangle
}

// LayerKeyOf makes the key of a layer's group from the matrix of its contents and the pixels it
// covers.
//
// Flutter: no counterpart
func LayerKeyOf(m *ui.Matrix4, rect image.Rectangle) LayerKey {
	k := LayerKey{Rect: rect}
	copy(k.M[:], m.Storage()[:])
	return k
}

// LayerEntry is a layer's pixels.
//
// Flutter: no counterpart
type LayerEntry[T any] struct {
	LayerKey
	// Tex is the renderer's texture of the pixels.
	Tex T
	// used is the frame that last drew it.
	used uint64
}

func (e *LayerEntry[T]) bytes() int { return e.Rect.Dx() * e.Rect.Dy() * 4 }

// layerSeen counts the consecutive scenes a layer was in.
type layerSeen struct {
	last uint64
	run  int
}

// LayerCache holds the pixels of retained layers.
//
// Flutter: no counterpart
type LayerCache[T any] struct {
	// after is how many consecutive scenes a layer must be in to be cached.
	after   int
	frame   uint64
	root    ui.Layer
	seen    map[ui.Layer]*layerSeen
	entries map[ui.Layer]*LayerEntry[T]
	bytes   int
}

// NewLayerCache makes a cache that keeps the layers that were in the last after scenes.
//
// Flutter: no counterpart
func NewLayerCache[T any](after int) *LayerCache[T] {
	return &LayerCache[T]{after: after, seen: map[ui.Layer]*layerSeen{}, entries: map[ui.Layer]*LayerEntry[T]{}}
}

// Len returns the number of entries.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Len() int { return len(c.entries) }

// Bytes returns the bytes of pixels the entries hold.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Bytes() int { return c.bytes }

// BeginFrame starts a frame of the scene whose root layer is given; the root is never cached.
//
// Flutter: no counterpart
func (c *LayerCache[T]) BeginFrame(root ui.Layer) {
	c.frame++
	c.root = root
}

// Boundary says whether the layer is cached: one of the layers that the draw list folds, that is
// not the root, and that has been in the last c.after scenes.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Boundary(l ui.Layer) bool {
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

// Lookup returns the layer's entry if it has the pixels for the key, and marks it used in this
// frame.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Lookup(l ui.Layer, k LayerKey) *LayerEntry[T] {
	e := c.entries[l]
	if e == nil || e.LayerKey != k {
		return nil
	}
	e.used = c.frame
	return e
}

// Put keeps the texture as the layer's pixels, and says whether it did; release frees the
// texture of the entry it replaces. An entry that a draw of this frame still waits for is not
// replaced.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Put(l ui.Layer, k LayerKey, tex T, release func(T)) bool {
	if old := c.entries[l]; old != nil {
		if old.used == c.frame {
			return false
		}
		c.drop(l, old, release)
	}
	e := &LayerEntry[T]{LayerKey: k, Tex: tex, used: c.frame}
	c.entries[l] = e
	c.bytes += e.bytes()
	return true
}

func (c *LayerCache[T]) drop(l ui.Layer, e *LayerEntry[T], release func(T)) {
	delete(c.entries, l)
	c.bytes -= e.bytes()
	release(e.Tex)
}

// EndFrame forgets the layers that were not in the scene, and evicts the entries that have not
// been used for a while, then the least recently used ones while the cache holds more than its
// budget.
//
// Flutter: no counterpart
func (c *LayerCache[T]) EndFrame(release func(T)) {
	for l, s := range c.seen {
		if s.last != c.frame {
			delete(c.seen, l)
		}
	}
	for l, e := range c.entries {
		if c.frame-e.used > LayerCacheIdleFrames {
			c.drop(l, e, release)
		}
	}
	for c.bytes > LayerCacheBytes {
		var oldest ui.Layer
		var oe *LayerEntry[T]
		for l, e := range c.entries {
			if oe == nil || e.used < oe.used {
				oldest, oe = l, e
			}
		}
		if oe == nil || oe.used == c.frame {
			return
		}
		c.drop(oldest, oe, release)
	}
}

// Release frees every texture.
//
// Flutter: no counterpart
func (c *LayerCache[T]) Release(release func(T)) {
	for l, e := range c.entries {
		c.drop(l, e, release)
	}
}
