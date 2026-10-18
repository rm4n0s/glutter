package gles

import (
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
)

// The sizes of the renderer's caches.
const (
	// atlasPageSize is the side of a page of the coverage atlas, and atlasMaxPages
	// the number of pages it may grow to: 8-bit pages, so 32 MB at most.
	atlasPageSize = 2048
	atlasMaxPages = 8
	// clipMaskBudget is the most memory, in bytes, that clip masks that are not
	// used in the current frame may hold.
	clipMaskBudget = 64 << 20
	// imageBudget is the same for image textures (RGBA, with mipmaps).
	imageBudget = 256 << 20
	// idleFrames is how long a cached mask or image may go unused before it is
	// dropped, whatever the budget.
	idleFrames = 240
)

// resources are the GPU objects the renderer keeps from frame to frame: the
// coverage atlas for paths, the textures of clip masks and images, and a white
// texel for draws without a clip mask.
type resources struct {
	dev   *Device
	gl    glAPI
	frame uint64

	atlas *gpu.Atlas[maskKey]
	pages []*Texture // the atlas pages, in order

	clipMasks map[uint64]*clipMask
	clipBytes int

	images     map[imageKey]*imageEntry
	imageBytes int

	white *Texture // a 1x1 R8 texture of 255, for unit 1 when there is no clip mask

	// Counters of work done, for FrameStats.
	tilesRasterized, clipMasksBuilt, imagesUploaded int
}

func newResources(dev *Device, gl glAPI) (*resources, error) {
	r := &resources{
		dev: dev, gl: gl,
		atlas:     gpu.NewAtlas[maskKey](atlasPageSize, atlasMaxPages),
		clipMasks: map[uint64]*clipMask{},
		images:    map[imageKey]*imageEntry{},
	}
	white, err := dev.AcquireTexture(1, 1, TextureFormatR8)
	if err != nil {
		return nil, err
	}
	dev.SetTextureParams(white, glNearest, glNearest, glClampToEdge, glClampToEdge)
	if err := dev.UploadTexture(white, 0, 0, 1, 1, []byte{255}); err != nil {
		return nil, err
	}
	r.white = white
	return r, nil
}

// beginFrame starts a frame: what is used from now on survives until the next.
func (r *resources) beginFrame() {
	r.frame++
	r.atlas.BeginFrame()
}

// endFrame drops what is old, or over the budget and not used in this frame.
func (r *resources) endFrame() {
	for k, m := range r.clipMasks {
		if r.frame-m.lastUsed > idleFrames {
			r.dropClipMask(k, m)
		}
	}
	for r.clipBytes > clipMaskBudget {
		if !r.dropOldestClipMask() {
			break
		}
	}
	for k, e := range r.images {
		if r.frame-e.lastUsed > idleFrames {
			r.dropImage(k, e)
		}
	}
	for r.imageBytes > imageBudget {
		if !r.dropOldestImage() {
			break
		}
	}
}

// release gives every texture back to the device's pool. The device is closed
// after it.
func (r *resources) release() {
	for k, m := range r.clipMasks {
		r.dropClipMask(k, m)
	}
	for k, e := range r.images {
		r.dropImage(k, e)
	}
	for _, p := range r.pages {
		r.dev.ReleaseTexture(p)
	}
	r.pages = nil
	r.atlas.Clear()
	if r.white != nil {
		r.dev.ReleaseTexture(r.white)
		r.white = nil
	}
}

// hasher is a 64-bit FNV-1a hash that the cache keys are made with.
type hasher uint64

func newHasher() hasher { return 14695981039346656037 }

func (h *hasher) u64(v uint64) {
	x := uint64(*h)
	for i := 0; i < 8; i++ {
		x ^= v & 0xff
		x *= 1099511628211
		v >>= 8
	}
	*h = hasher(x)
}

func (h *hasher) f64(v float64) { h.u64(math.Float64bits(v)) }
func (h *hasher) i(v int)       { h.u64(uint64(int64(v))) }
func (h *hasher) b(v bool) {
	if v {
		h.u64(1)
	} else {
		h.u64(0)
	}
}
