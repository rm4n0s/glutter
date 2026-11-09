//go:build darwin && !ios

package metal

import (
	"image"
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/ui"
)

// hostLayer makes a plain layer of the size, standing for the layer of a view, with the scale 2 of a Retina display.
func hostLayer(t *testing.T, w, h float64) objc.ID {
	t.Helper()
	if _, err := metalFramework(); err != nil {
		t.Fatal(err)
	}
	l := newObject("CALayer")
	t.Cleanup(func() { release(l) })
	call(l, "setContentsScale:", float64(2))
	call(l, "setFrame:", cgRect{Origin: darwin.CGPoint{X: 0, Y: 0}, Size: darwin.CGSize{Width: w, Height: h}})
	return l
}

func surfaceOf(l objc.ID) gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindMetalLayer, Window: uintptr(l), Surface: uintptr(l)}
}

// drawFrame acquires a drawable, clears it to the color, reads one pixel back and presents.
func drawFrame(t *testing.T, d *Device, sc explicit.Swapchain, c [4]float64) [4]byte {
	t.Helper()
	tex, err := sc.Acquire()
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex, Clear: c}}}).End()
	var px [4]byte
	enc.ReadTexture(tex, image.Rect(0, 0, 1, 1), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
			return
		}
		copy(px[:], img.Pix[:4])
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if err := sc.Present(nil); err != nil {
		t.Fatalf("Present: %v", err)
	}
	return px
}

func TestSwapchainPresentsAndFollowsResizes(t *testing.T) {
	d := openTestDevice(t, Options{Readback: true})
	host := hostLayer(t, 200, 100)
	sc, err := d.NewSwapchain(surfaceOf(host))
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Destroy()
	if sc.PreRotation() != explicit.Rotation0 || sc.Stale() {
		t.Error("a new swap chain is not rotated and not stale")
	}
	// The window changes size as a live resize does: each size is configured, and the frames keep coming.
	for i, size := range []image.Point{{400, 200}, {401, 199}, {300, 150}, {800, 400}, {400, 200}, {1, 1}, {640, 480}} {
		if err := sc.Configure(size.X, size.Y); err != nil {
			t.Fatalf("Configure %v: %v", size, err)
		}
		for frame := 0; frame < 4; frame++ {
			px := drawFrame(t, d, sc, [4]float64{1, 0, 0, 1})
			if px != [4]byte{255, 0, 0, 255} {
				t.Fatalf("size %d %v, frame %d: the pixel is %v, want red", i, size, frame, px)
			}
		}
		tex, err := sc.Acquire()
		if err != nil {
			t.Fatal(err)
		}
		if got := tex.Desc(); got.Width != size.X || got.Height != size.Y || got.Format != explicit.TextureFormatBGRA8Unorm {
			t.Errorf("the drawable is %dx%d %d, want %dx%d BGRA8", got.Width, got.Height, got.Format, size.X, size.Y)
		}
		// Acquire again gives the same drawable until it is presented.
		if again, err := sc.Acquire(); err != nil || again != tex {
			t.Errorf("a second Acquire gave %v, %v", again, err)
		}
		if err := sc.Present(nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := sc.Present(nil); err == nil {
		t.Error("Present without an Acquire did not fail")
	} else {
		wantRoute(t, err, "swapchain.Present.MetalSwapchainFailed")
	}
	if got := send[uint64](sc.(*swapchain).layer, "maximumDrawableCount"); got != 3 {
		t.Errorf("maximumDrawableCount = %d, want 3", got)
	}
	if !sendBool(sc.(*swapchain).layer, "displaySyncEnabled") {
		t.Error("displaySyncEnabled is off")
	}
	if send[float64](sc.(*swapchain).layer, "contentsScale") != 2 {
		t.Error("the layer does not have the scale of the view's layer")
	}
}

func TestSwapchainOfALayerThatIsEmptyGivesNoDrawable(t *testing.T) {
	d := openTestDevice(t, Options{})
	host := hostLayer(t, 0, 0)
	sc, err := d.NewSwapchain(surfaceOf(host))
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Destroy()
	_, err = sc.Acquire()
	wantRoute(t, err, "swapchain.Acquire.MetalDrawableTimeout")
}

func TestSwapchainConfigureNeedsASize(t *testing.T) {
	d := openTestDevice(t, Options{})
	sc, err := d.NewSwapchain(surfaceOf(hostLayer(t, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Destroy()
	wantRoute(t, sc.Configure(0, 5), "swapchain.Configure.MetalSwapchainFailed")
}

func TestSwapchainRenderedWithAPipeline(t *testing.T) {
	// A frame through the explicit renderer into a real layer: the renderer draws, presents, and the drawable has
	// what it drew (the proof scene is compared in the parity tests).
	d := openTestDevice(t, Options{Readback: true})
	host := hostLayer(t, 120, 80)
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return d, nil },
		explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true})
	defer func() { r.Close() }()
	scene := sceneOfRect()
	img, err := r.Draw(surfaceOf(host), 240, 160, scene, true)
	if err != nil {
		t.Fatalf("Draw: %+v", err)
	}
	if img == nil || img.Bounds().Dx() != 240 || img.Bounds().Dy() != 160 {
		t.Fatalf("the frame is %v", img)
	}
	if c := img.RGBAAt(100, 60); c.R != 0x33 || c.G != 0x66 || c.B != 0xCC {
		t.Errorf("the rectangle is %v, want 3366CC", c)
	}
}
