package webgpu

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// Renderer is the explicit renderer over a WebGPU device (SPEC §9.5). It draws as the Vulkan
// renderer does, with one difference: a snapshot (Picture.ToImage) is drawn by the software
// renderer. A WebGPU read-back finishes in a later browser task and the UI goroutine does not
// wait for one (SPEC §23.3), while ToImage returns at once (SPEC §12.10).
//
// Flutter: no counterpart (flutter::Rasterizer)
type Renderer struct {
	*explicit.Renderer
}

var _ ui.Rasterizer = (*Renderer)(nil)

// NewRenderer returns a renderer that opens its device with open when it draws its first frame:
// the Open method of a Provider, or a function that gives another device. More than one options
// value panics with MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(open func(gpu.SurfaceHandle) (explicit.Device, error), opts ...explicit.RendererOpts) *Renderer {
	return &Renderer{Renderer: explicit.NewRenderer(open, opts...)}
}

// Snapshot draws the scene into a new image of the size with the software renderer, over
// transparent pixels, premultiplied, top row first: the pixels of Picture.ToImage. It fails
// with RendererSizeInvalid for a size that is not positive.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) {
	w, h := int(size.Width), int(size.Height)
	if w <= 0 || h <= 0 {
		return nil, errors.New(tagRendererSizeInvalid, "the size to render must be positive", "width", size.Width, "height", size.Height)
	}
	return soft.Render(scene, w, h)
}

// RasterizeScene draws the scene into a new image of width x height pixels, as Snapshot does: the
// ui.Rasterizer for Picture.ToImage and Scene.ToImage.
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
	tagRendererSizeInvalid = "RendererSizeInvalid"
)
