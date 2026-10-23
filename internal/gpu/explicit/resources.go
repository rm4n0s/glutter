package explicit

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// resources are the GPU objects the renderer keeps from frame to frame: the coverage atlas
// for paths, the textures of clip masks and images, and a white texel for draws without a
// clip mask (SPEC §9.1). What a frame uses is kept until the frame after it; the textures that
// are old, or over the budget and not used in the current frame, are destroyed (the device
// finishes the frames that use them first).
type resources struct {
	dev   Device
	frame uint64

	masks *gpu.MaskCache
	pages []Texture // the atlas pages, in order

	clipMasks map[uint64]*clipMask
	clipBytes int

	images     map[gpu.ImageKey]*imageEntry
	imageBytes int
	// mips are the images uploaded since the last frame was recorded, whose mip levels are
	// still to be made by the downsampling pass.
	mips []*imageEntry

	white Texture // a 1x1 R8 texture of 255, for draws without a clip mask
	clear Texture // a 1x1 RGBA texture of transparent black, for the image and destination bindings a draw does not use

	// Counters of work done, for FrameStats (the tiles rasterized are masks.Rasterized).
	clipMasksBuilt, imagesUploaded int
}

// clipMask is the texture of the coverage of a clip's shapes over a rectangle of the surface.
type clipMask struct {
	tex        Texture
	x, y, w, h int // the rectangle of the surface the texture covers
	lastUsed   uint64
}

// imageEntry is an image's texture.
type imageEntry struct {
	tex      Texture
	w, h     int
	levels   int    // the number of mip levels, the image itself included
	pix      []byte // kept so that the address in the key is not reused
	bytes    int
	lastUsed uint64
}

func newResources(dev Device) (*resources, error) {
	r := &resources{
		dev:       dev,
		masks:     gpu.NewMaskCache(gpu.AtlasPageSize, gpu.AtlasMaxPages),
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
		page, err := dev.NewTexture(TextureDesc{Width: gpu.AtlasPageSize, Height: gpu.AtlasPageSize, Format: TextureFormatR8Unorm, MipLevels: 1, Usage: TextureSampled})
		if err != nil {
			return err
		}
		r.pages = append(r.pages, page)
		return nil
	}
	r.masks.Upload = func(page, x, y, w, h int, data []byte) error {
		dev.WriteTexture(r.pages[page], 0, image.Rect(x, y, x+w, y+h), data, w)
		return nil
	}
	return r, nil
}

// whiteTexture returns the 1x1 texture of full coverage, made the first time it is asked for.
func (r *resources) whiteTexture() (Texture, error) {
	if r.white == nil {
		white, err := r.dev.NewTexture(TextureDesc{Width: 1, Height: 1, Format: TextureFormatR8Unorm, MipLevels: 1, Usage: TextureSampled})
		if err != nil {
			return nil, err
		}
		r.dev.WriteTexture(white, 0, image.Rect(0, 0, 1, 1), []byte{255}, 1)
		r.white = white
	}
	return r.white, nil
}

// beginFrame starts a frame: what is used from now on survives until the next.
func (r *resources) beginFrame() {
	r.frame++
	r.masks.BeginFrame()
}

// endFrame drops what is old, or over the budget and not used in this frame.
func (r *resources) endFrame() {
	for k, m := range r.clipMasks {
		if r.frame-m.lastUsed > gpu.IdleFrames {
			r.dropClipMask(k, m)
		}
	}
	for r.clipBytes > gpu.ClipMaskBudget {
		if !r.dropOldestClipMask() {
			break
		}
	}
	for k, e := range r.images {
		if r.frame-e.lastUsed > gpu.IdleFrames {
			r.dropImage(k, e)
		}
	}
	for r.imageBytes > gpu.ImageBudget {
		if !r.dropOldestImage() {
			break
		}
	}
}

// release destroys every texture. The device is closed after it.
func (r *resources) release() {
	for k, m := range r.clipMasks {
		r.dropClipMask(k, m)
	}
	for k, e := range r.images {
		r.dropImage(k, e)
	}
	for _, p := range r.pages {
		p.Destroy()
	}
	r.pages = nil
	r.masks.Clear()
	if r.white != nil {
		r.white.Destroy()
		r.white = nil
	}
	if r.clear != nil {
		r.clear.Destroy()
		r.clear = nil
	}
}

// clearTexture returns the 1x1 transparent texture that stands for a texture a draw does not use,
// made the first time it is asked for.
func (r *resources) clearTexture() (Texture, error) {
	if r.clear == nil {
		t, err := r.dev.NewTexture(TextureDesc{Width: 1, Height: 1, Format: TextureFormatRGBA8Unorm, MipLevels: 1, Usage: TextureSampled})
		if err != nil {
			return nil, err
		}
		r.dev.WriteTexture(t, 0, image.Rect(0, 0, 1, 1), []byte{0, 0, 0, 0}, 4)
		r.clear = t
	}
	return r.clear, nil
}

// clipMask returns the mask of the shapes of the clip over the clip's bounds inside a surface
// of w x h pixels, making it if there is none cached. empty says that the shapes leave nothing.
func (r *resources) clipMask(c gpu.Clip, w, h int) (m *clipMask, empty bool, err error) {
	area := gpu.PixelRect(c.Bounds).Intersect(image.Rect(0, 0, w, h))
	if area.Empty() {
		return nil, true, nil
	}
	key := gpu.ClipKey(area, c.Shapes)
	if m, ok := r.clipMasks[key]; ok {
		m.lastUsed = r.frame
		return m, false, nil
	}
	r.clipMasksBuilt++
	data, empty := gpu.ClipCoverage(c, area, func(path *ui.Path, m *ui.Matrix4, aa bool, a image.Rectangle) (image.Rectangle, []float32) {
		mask := soft.ClipMask(path, m, aa, a)
		if mask == nil {
			return image.Rectangle{}, nil
		}
		return mask.Rect, mask.Cov
	})
	if empty {
		return nil, true, nil
	}
	tex, err := r.dev.NewTexture(TextureDesc{Width: area.Dx(), Height: area.Dy(), Format: TextureFormatR8Unorm, MipLevels: 1, Usage: TextureSampled})
	if err != nil {
		return nil, false, err
	}
	r.dev.WriteTexture(tex, 0, image.Rect(0, 0, area.Dx(), area.Dy()), data, area.Dx())
	m = &clipMask{tex: tex, x: area.Min.X, y: area.Min.Y, w: area.Dx(), h: area.Dy(), lastUsed: r.frame}
	r.clipMasks[key] = m
	r.clipBytes += len(data)
	return m, false, nil
}

func (r *resources) dropClipMask(key uint64, m *clipMask) {
	m.tex.Destroy()
	r.clipBytes -= m.w * m.h
	delete(r.clipMasks, key)
}

// dropOldestClipMask drops the least recently used mask that this frame does not use; false
// when there is none.
func (r *resources) dropOldestClipMask() bool {
	var oldestKey uint64
	var oldest *clipMask
	for k, m := range r.clipMasks {
		if m.lastUsed != r.frame && (oldest == nil || m.lastUsed < oldest.lastUsed) {
			oldestKey, oldest = k, m
		}
	}
	if oldest == nil {
		return false
	}
	r.dropClipMask(oldestKey, oldest)
	return true
}

// imageTexture returns the texture of the image, uploading it the first time; its mip levels
// are made when the frame is recorded. It fails with GPUImageTooLarge for an image bigger than
// the device's textures.
func (r *resources) imageTexture(img *ui.Image) (*imageEntry, error) {
	pix := img.Pixels()
	w, h := img.Width(), img.Height()
	key := gpu.KeyOfImage(img)
	if e, ok := r.images[key]; ok {
		e.lastUsed = r.frame
		return e, nil
	}
	if limit := r.dev.Limits().MaxTextureSize; w > limit || h > limit {
		return nil, errors.New(TagGPUImageTooLarge, "the image is larger than the textures of the device", "width", w, "height", h, "limit", limit)
	}
	levels := gpu.MipLevels(w, h)
	usage := TextureSampled
	if levels > 1 {
		usage |= TextureRenderTarget // the downsampling pass draws into the levels
	}
	tex, err := r.dev.NewTexture(TextureDesc{Width: w, Height: h, Format: TextureFormatRGBA8Unorm, MipLevels: levels, Usage: usage})
	if err != nil {
		return nil, err
	}
	r.imagesUploaded++
	r.dev.WriteTexture(tex, 0, image.Rect(0, 0, w, h), pix, w*4)
	e := &imageEntry{tex: tex, w: w, h: h, levels: levels, pix: pix, bytes: len(pix) * 4 / 3, lastUsed: r.frame}
	r.images[key] = e
	r.imageBytes += e.bytes
	if levels > 1 {
		r.mips = append(r.mips, e)
	}
	return e, nil
}

func (r *resources) dropImage(key gpu.ImageKey, e *imageEntry) {
	e.tex.Destroy()
	r.imageBytes -= e.bytes
	delete(r.images, key)
	for i, m := range r.mips {
		if m == e {
			r.mips = append(r.mips[:i], r.mips[i+1:]...)
			break
		}
	}
}

// dropOldestImage drops the least recently used image that this frame does not use; false
// when there is none.
func (r *resources) dropOldestImage() bool {
	var oldestKey gpu.ImageKey
	var oldest *imageEntry
	for k, e := range r.images {
		if e.lastUsed != r.frame && (oldest == nil || e.lastUsed < oldest.lastUsed) {
			oldestKey, oldest = k, e
		}
	}
	if oldest == nil {
		return false
	}
	r.dropImage(oldestKey, oldest)
	return true
}
