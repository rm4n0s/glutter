package gpu

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// Surface is a native drawing surface that a Renderer has attached to: the
// target that Render draws and presents into.
//
// Flutter: no counterpart (flutter::Surface)
type Surface interface {
	// Destroy releases the surface. It must be done before the native window is
	// destroyed, and is safe to do twice.
	Destroy()
}

// Renderer draws scenes. The GLES renderer (internal/gpu/gles) draws on the GPU;
// the software renderer (internal/gpu/soft) is the reference it is compared with.
// Only the UI thread calls a Renderer (SPEC §13).
//
// Flutter: no counterpart (flutter::Rasterizer)
type Renderer interface {
	// Attach starts drawing into a native surface. The first call creates the
	// renderer's device.
	Attach(SurfaceHandle) (Surface, error)
	// Render draws the scene into the surface, at the given size in device pixels,
	// and presents it. dpr is the device pixel ratio, which the scene's
	// coordinates are already scaled by.
	Render(s Surface, scene *ui.Scene, size ui.Size, dpr float32) error
	// Snapshot draws the scene into a new image of the size, without a window:
	// the work of Picture.ToImage and of the raster cache.
	Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error)
	// Close destroys every surface and the device.
	Close() error
}
