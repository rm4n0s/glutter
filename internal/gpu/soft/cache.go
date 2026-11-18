package soft

import (
	"image"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// MipBudget is the most memory, in bytes, that the mipmap levels of a Cache's images that were not
// drawn in the current frame may hold (an image's own pixels, its level 0, are not counted: the image
// has them anyway).
//
// Flutter: no counterpart
const MipBudget = 64 << 20

// Cache keeps what the software renderer makes from the images of a scene and can use again in the
// frames after it: the mipmap chain of each image it scales down (SPEC §9.1, "Images"). A renderer
// that draws frame after frame keeps one Cache, draws every frame with its RenderInto or Render, and
// calls EndFrame after each frame, as the GPU renderers keep the textures of images: a picture under
// an animation is not made smaller again in every frame.
//
// A chain is keyed by the image's pixels (gpu.KeyOfImage), so clones share it, and only the levels
// that a draw needs are made. EndFrame drops the chain of an image that was not drawn for
// gpu.IdleFrames frames, of an image whose handle it was last drawn with is disposed, and the chains
// drawn longest ago while the levels are over MipBudget. A Cache may be used by several goroutines at
// once.
//
// Flutter: no counterpart (Skia keeps the mipmaps of an SkImage with the image)
type Cache struct {
	mu sync.Mutex
	// budget is MipBudget; a test makes it smaller.
	budget int
	frame  int
	chains map[gpu.ImageKey]*mipEntry
	bytes  int
	built  int
	drops  int
}

// mipEntry is the mipmap chain of one image's pixels.
type mipEntry struct {
	// levels holds the image itself first and as many smaller levels as a draw has needed.
	levels []*mipLevel
	// img is the handle the chain was last drawn with.
	img *ui.Image
	// used is the frame of the last draw.
	used int
}

// CacheStats is what a Cache holds and what it has done since it was made.
//
// Flutter: no counterpart
type CacheStats struct {
	// Chains is the number of images whose mipmap chain the cache holds.
	Chains int
	// Bytes is the memory of their levels, without the images' own pixels.
	Bytes int
	// Built is how many chains the cache has started, Dropped how many it has dropped.
	Built, Dropped int
}

// NewCache returns an empty cache.
//
// Flutter: no counterpart
func NewCache() *Cache { return &Cache{budget: MipBudget, chains: map[gpu.ImageKey]*mipEntry{}} }

// RenderInto is the function RenderInto that uses the cache's mipmap chains and keeps the ones it
// makes. It fails as RenderInto does.
//
// Flutter: no counterpart
func (c *Cache) RenderInto(dst *image.RGBA, scene *ui.Scene, region image.Rectangle) error {
	if scene == nil {
		return errors.New(tagSoftSceneNil, "RenderInto needs a scene")
	}
	if dst == nil || dst.Rect.Empty() {
		return errors.New(tagSoftTargetInvalid, "RenderInto needs an image that is not empty.")
	}
	return renderInto(dst, scene, region.Intersect(dst.Rect), c)
}

// Render is the function Render that uses the cache's mipmap chains and keeps the ones it makes. It
// fails as Render does.
//
// Flutter: no counterpart
func (c *Cache) Render(scene *ui.Scene, width, height int) (*image.RGBA, error) {
	if scene == nil {
		return nil, errors.New(tagSoftSceneNil, "Render needs a scene")
	}
	if width <= 0 || height <= 0 || width > maxSurfaceSize || height > maxSurfaceSize {
		return nil, errors.New(tagSoftSizeInvalid, "The surface size is not valid.", "width", width, "height", height)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := renderInto(img, scene, img.Rect, c); err != nil {
		return nil, err
	}
	return img, nil
}

// EndFrame ends a frame: it drops the chains of the images that were not drawn for gpu.IdleFrames
// frames or whose handle is disposed, and then the chains drawn longest ago, but none drawn in this
// frame, while the levels hold more than MipBudget.
//
// Flutter: no counterpart
func (c *Cache) EndFrame() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.chains {
		// A disposed handle was drawn before it was disposed (drawing it fails), so the chain is
		// dropped even after a draw in this frame: a clone that is drawn again makes it again.
		if c.frame-e.used > gpu.IdleFrames || e.img.IsDisposed() {
			c.drop(k, e)
		}
	}
	for c.bytes > c.budget {
		var (
			oldest *mipEntry
			key    gpu.ImageKey
		)
		for k, e := range c.chains {
			if e.used < c.frame && (oldest == nil || e.used < oldest.used) {
				oldest, key = e, k
			}
		}
		if oldest == nil {
			break
		}
		c.drop(key, oldest)
	}
	c.frame++
}

// drop forgets a chain; c.mu is held.
func (c *Cache) drop(k gpu.ImageKey, e *mipEntry) {
	for _, l := range e.levels[1:] {
		c.bytes -= len(l.pix)
	}
	delete(c.chains, k)
	c.drops++
}

// Stats returns what the cache holds and has done.
//
// Flutter: no counterpart
func (c *Cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{Chains: len(c.chains), Bytes: c.bytes, Built: c.built, Dropped: c.drops}
}

// levels returns the mipmap chain of the image, the image itself first, down to level n at least
// (fewer when the chain reaches one pixel before). Each level halves the one before, rounding up,
// and averages 2x2 pixels. The levels are never changed once made.
func (c *Cache) levels(img *ui.Image, n int) []*mipLevel {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := gpu.KeyOfImage(img)
	e, ok := c.chains[k]
	if !ok {
		e = &mipEntry{levels: []*mipLevel{{w: img.Width(), h: img.Height(), pix: img.Pixels()}}}
		c.chains[k] = e
		c.built++
	}
	e.img, e.used = img, c.frame
	for len(e.levels) <= n {
		prev := e.levels[len(e.levels)-1]
		if prev.w == 1 && prev.h == 1 {
			break
		}
		next := halve(prev)
		e.levels = append(e.levels, next)
		c.bytes += len(next.pix)
	}
	return e.levels
}

// halve returns the next smaller level of a mipmap chain.
func halve(prev *mipLevel) *mipLevel {
	next := &mipLevel{w: max(prev.w/2, 1), h: max(prev.h/2, 1)}
	next.pix = make([]byte, 4*next.w*next.h)
	for y := 0; y < next.h; y++ {
		for x := 0; x < next.w; x++ {
			x0, y0 := min(2*x, prev.w-1), min(2*y, prev.h-1)
			x1, y1 := min(2*x+1, prev.w-1), min(2*y+1, prev.h-1)
			for k := 0; k < 4; k++ {
				sum := int(prev.pix[4*(y0*prev.w+x0)+k]) + int(prev.pix[4*(y0*prev.w+x1)+k]) +
					int(prev.pix[4*(y1*prev.w+x0)+k]) + int(prev.pix[4*(y1*prev.w+x1)+k])
				next.pix[4*(y*next.w+x)+k] = byte((sum + 2) / 4)
			}
		}
	}
	return next
}
