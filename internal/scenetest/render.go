package scenetest

import (
	"image"
	"image/color"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/viewport3d"
)

// Case is a scene that the display tests draw: a name for its golden image, the size of the picture in pixels, and how to make
// the scene and its camera for a device (a texture needs one).
type Case struct {
	Name          string
	Width, Height int
	// Samples is 1 or 4; the default is 4.
	Samples int
	// Build makes the scene and its camera, and a function that releases what it made on the device (nil for none).
	Build func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func())
}

// TagReadbackFailed is the tag of the error of Render when the picture was not read back by the time wait returned.
const TagReadbackFailed = "SceneReadbackFailed"

// Render draws a case with a renderer into a texture of its size on the device, and reads the picture back. wait runs the
// work that the device posted until done says the read has finished, or it gives up (a test makes it from the app's
// RunPosted, and a browser's reads finish in later tasks, so it sleeps between the calls); it is called after the commands are
// submitted and the read is asked for.
func Render(d *gpu.Device, wait func(done func() bool), c Case) (*image.RGBA, error) {
	samples := c.Samples
	if samples == 0 {
		samples = 4
	}
	scene, camera, cleanup := c.Build(d)
	if cleanup != nil {
		defer cleanup()
	}
	r := viewport3d.NewRenderer(scene, camera)
	r.Init(d)
	defer r.Dispose()

	size := gpu.Extent3D{Width: c.Width, Height: c.Height, DepthOrArrayLayers: 1}
	resolve, err := d.NewTexture(gpu.TextureDesc{Label: "scene resolve", Size: size, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageCopySrc | gpu.TextureUsageTextureBinding})
	if err != nil {
		return nil, err
	}
	defer resolve.Release()
	colorTex := resolve
	var multi *gpu.Texture
	if samples > 1 {
		multi, err = d.NewTexture(gpu.TextureDesc{Label: "scene color", Size: size, Format: gpu.TextureFormatRGBA8Unorm, SampleCount: samples,
			Usage: gpu.TextureUsageRenderAttachment})
		if err != nil {
			return nil, err
		}
		defer multi.Release()
		colorTex = multi
	}
	depth, err := d.NewTexture(gpu.TextureDesc{Label: "scene depth", Size: size, Format: gpu.TextureFormatDepth32Float, SampleCount: samples,
		Usage: gpu.TextureUsageRenderAttachment})
	if err != nil {
		return nil, err
	}
	defer depth.Release()
	rowBytes := d.Limits().BytesPerRowAlignment
	row := (c.Width*4 + rowBytes - 1) / rowBytes * rowBytes
	buf, err := d.NewBuffer(gpu.BufferDesc{Label: "scene readback", Size: row * c.Height, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst})
	if err != nil {
		return nil, err
	}
	defer buf.Release()

	enc := d.NewCommandEncoder()
	frame := &gpu.ViewFrame{Device: d, Encoder: enc, Color: colorTex, Depth: depth, Width: c.Width, Height: c.Height, DevicePixelRatio: 1, Time: time.Second}
	if samples > 1 {
		frame.Resolve = resolve
	}
	r.Draw(frame)
	enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: resolve}, gpu.BufferCopy{Buffer: buf, Layout: gpu.DataLayout{BytesPerRow: row}}, size)
	if err := d.Queue().Submit(enc.Finish()); err != nil {
		return nil, err
	}
	var data []byte
	var readErr error
	done := false
	buf.Read(0, row*c.Height, func(b []byte, err error) { data, readErr, done = b, err, true })
	wait(func() bool { return done })
	if !done {
		return nil, errors.New(TagReadbackFailed, "the read of the picture did not finish", "case", c.Name)
	}
	if readErr != nil {
		return nil, readErr
	}
	img := image.NewRGBA(image.Rect(0, 0, c.Width, c.Height))
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			i := y*row + x*4
			// The picture is premultiplied, and opaque where the background is.
			img.SetRGBA(x, y, color.RGBA{R: data[i], G: data[i+1], B: data[i+2], A: data[i+3]})
		}
	}
	return img, nil
}
