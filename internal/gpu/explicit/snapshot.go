package explicit

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

var _ ui.Rasterizer = (*Renderer)(nil)

// Snapshot draws the scene into a new image of the size, without a window: the work of
// Picture.ToImage and Scene.ToImage. The scene is drawn over transparent pixels into a pooled
// texture, which is read back; the pixels are premultiplied, top row first. The device must
// exist, so a window has to have been drawn into at some point; the raster cache is neither
// used nor fed. It fails with RendererClosed, RendererNotAttached, RendererSizeInvalid,
// GPUNotImplemented as Draw does, and the errors of the device.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) {
	if r.closed {
		return nil, errors.New(tagRendererClosed, "the renderer was closed")
	}
	if r.dev == nil {
		return nil, errors.New(tagRendererNotAttached, "a snapshot needs the renderer's device, which the first frame of a window creates")
	}
	w, h := int(size.Width), int(size.Height)
	if w <= 0 || h <= 0 {
		return nil, errors.New(tagRendererSizeInvalid, "the size to render must be positive", "width", size.Width, "height", size.Height)
	}
	if w > r.dev.Limits().MaxTextureSize || h > r.dev.Limits().MaxTextureSize {
		return nil, errors.New(tagRendererSizeInvalid, "the size to render is larger than the textures of the device", "width", w, "height", h, "limit", r.dev.Limits().MaxTextureSize)
	}
	defer func() {
		if r.res != nil {
			r.res.endFrame()
			r.sweepLayers()
		}
	}()
	// The last frame's numbers stay the window's: a snapshot is not a frame.
	last := r.last
	defer func() { r.last = last }()
	fb, err := r.build(scene, w, h, false)
	if err != nil {
		return nil, err
	}
	tex, err := r.layerTexture(w, h)
	if err != nil {
		return nil, err
	}
	defer r.releaseLayer(tex)
	enc, err := r.dev.Begin()
	if err != nil {
		return nil, err
	}
	slot := &r.slots[r.frameNo%FramesInFlight]
	r.frameNo++
	if err := r.drawScene(enc, slot, tex, fb, w, h, ui.Color{}); err != nil {
		_ = enc.Submit()
		return nil, err
	}
	var shot *image.RGBA
	var readErr error
	enc.ReadTexture(tex, image.Rect(0, 0, w, h), func(i *image.RGBA, err error) { shot, readErr = i, err })
	if err := enc.Submit(); err != nil {
		return nil, err
	}
	if readErr != nil {
		return nil, readErr
	}
	if shot == nil {
		return nil, errors.New(tagRendererCaptureFailed, "the device did not deliver the snapshot it was asked to read back")
	}
	return shot, nil
}

// RasterizeScene draws the scene into a new image of width x height pixels, as Snapshot does: the
// ui.Rasterizer of the GPU renderer, for Picture.ToImage and Scene.ToImage.
//
// Flutter: no counterpart
func (r *Renderer) RasterizeScene(scene *ui.Scene, width, height int) (*ui.Image, error) {
	img, err := r.Snapshot(scene, ui.Size{Width: float64(width), Height: float64(height)})
	if err != nil {
		return nil, err
	}
	return ui.NewImageFromRGBA(img.Pix, width, height), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRendererNotAttached = "RendererNotAttached"
)
