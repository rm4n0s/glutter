package offscreen

import (
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// Rasterizer is a ui.Rasterizer that draws on the CPU with Glutter's software
// renderer: apps give it to Scene.ToImage, Picture.ToImage and
// RenderRepaintBoundary.ToImage (ToImageOpts.Rasterizer) to turn what they draw into
// pixels, with no window and no GPU, on any platform. It keeps nothing after
// RasterizeScene returns, so one Rasterizer may be used by many goroutines at once. A
// scene made by the framework is drawn on the UI thread; to draw it on another
// goroutine, detach it first (Detacher).
//
// Flutter: no counterpart
type Rasterizer struct{}

var _ ui.Rasterizer = (*Rasterizer)(nil)

// NewRasterizer returns a software rasterizer.
//
// Flutter: no counterpart
func NewRasterizer() *Rasterizer { return &Rasterizer{} }

// RasterizeScene draws the scene into a new image of width x height pixels, in
// premultiplied RGBA with a transparent background. The errors are those of
// internal/gpu/soft: a size that is not valid, a nil scene, a layer or an operation the
// software renderer cannot draw, an image that was disposed.
//
// Flutter: no counterpart
func (*Rasterizer) RasterizeScene(scene *ui.Scene, width, height int) (*ui.Image, error) {
	img, err := soft.Render(scene, width, height)
	if err != nil {
		return nil, err
	}
	return ui.NewImageFromRGBA(img.Pix, width, height), nil
}
