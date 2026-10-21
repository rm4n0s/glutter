package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
	"image"
)

// The sizes of the renderer's caches.
const (
	// atlasPageSize is the side of a page of the coverage atlas, and atlasMaxPages
	// the number of pages it may grow to: 8-bit pages, so 32 MB at most.
	atlasPageSize = gpu.AtlasPageSize
	atlasMaxPages = gpu.AtlasMaxPages
	// clipMaskBudget is the most memory, in bytes, that clip masks that are not
	// used in the current frame may hold.
	clipMaskBudget = gpu.ClipMaskBudget
	// imageBudget is the same for image textures (RGBA, with mipmaps).
	imageBudget = gpu.ImageBudget
	// idleFrames is how long a cached mask or image may go unused before it is
	// dropped, whatever the budget.
	idleFrames = gpu.IdleFrames
)

// resources are the GPU objects the renderer keeps from frame to frame: the
// coverage atlas for paths, the textures of clip masks and images, and a white
// texel for draws without a clip mask.
type resources struct {
	dev   *Device
	gl    glAPI
	frame uint64

	masks *gpu.MaskCache // the coverage atlas and its tiling
	pages []*Texture     // the atlas pages, in order

	clipMasks map[uint64]*clipMask
	clipBytes int

	images     map[gpu.ImageKey]*imageEntry
	imageBytes int

	white *Texture // a 1x1 R8 texture of 255, for unit 1 when there is no clip mask

	// Counters of work done, for FrameStats (the tiles rasterized are masks.Rasterized).
	clipMasksBuilt, imagesUploaded int
}

func newResources(dev *Device, gl glAPI) (*resources, error) {
	r := &resources{
		dev: dev, gl: gl,
		masks:     gpu.NewMaskCache(atlasPageSize, atlasMaxPages),
		clipMasks: map[uint64]*clipMask{},
		images:    map[gpu.ImageKey]*imageEntry{},
	}
	r.masks.Rasterize = func(op ui.DisplayOp, m *ui.Matrix4, area image.Rectangle) (image.Rectangle, []float32) {
		mask, _ := soft.OpMask(op, m, area)
		if mask == nil {
			return image.Rectangle{}, nil
		}
		return mask.Rect, mask.Cov
	}
	r.masks.NewPage = func() error {
		page, err := dev.AcquireTexture(atlasPageSize, atlasPageSize, TextureFormatR8)
		if err != nil {
			return err
		}
		dev.SetTextureParams(page, glNearest, glNearest, glClampToEdge, glClampToEdge)
		r.pages = append(r.pages, page)
		return nil
	}
	r.masks.Upload = func(page, x, y, w, h int, data []byte) error {
		return dev.UploadTexture(r.pages[page], x, y, w, h, data)
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
	r.masks.BeginFrame()
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
	r.masks.Clear()
	if r.white != nil {
		r.dev.ReleaseTexture(r.white)
		r.white = nil
	}
}
