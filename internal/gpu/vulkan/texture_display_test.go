//go:build linux && glutter_display

package vulkan

import (
	"image"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/ui"
)

// The display test of step 30.5: a registered texture is shown by the explicit renderer on lavapipe (validation
// on) under a rotated, clipped, half transparent layer, and the frame matches what the software renderer draws for
// the same pixels as an image (SPEC §27.4).

// textureHandler runs a function at the first frame of the window.
type textureHandler struct {
	platform.NopHandler
	r    *explicit.Renderer
	win  platform.Window
	run  func(r *explicit.Renderer, h platform.Window)
	mu   sync.Mutex
	done bool
}

func (h *textureHandler) OnFrame(ui.ViewID, time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return
	}
	h.done = true
	h.run(h.r, h.win)
}

func (h *textureHandler) finished() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.done
}

// onWindow opens a window and a renderer with a texture registry, and calls run on the platform's goroutine with
// the renderer, the device (open: an empty frame has been drawn) and the registry.
func onWindow(t *testing.T, run func(r *explicit.Renderer, win platform.Window, dev explicit.Device, reg *explicit.TextureRegistry)) {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	reg := explicit.NewTextureRegistry()
	r, dev := newWindowRenderer(t, func(o *explicit.RendererOpts) {
		o.Background = ui.NewColor(0xFFFFFFFF)
		o.Textures = reg
	})
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "vulkan texture test"
	cfg.Size = ui.Size{Width: 128, Height: 128}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &textureHandler{r: r, win: win}
	h.run = func(r *explicit.Renderer, win platform.Window) {
		// The first frame opens the device.
		if _, err := r.Draw(win.NativeSurface(), 128, 128, ui.NewSceneBuilder().Build(), false); err != nil {
			t.Error(err)
			return
		}
		run(r, win, dev, reg)
	}
	stop := runPlatform(t, p, h)
	waitUntil(t, h.finished, "the textures were not drawn")
	stop()
	checkValidation(t, r, dev)
}

// textureOf makes a texture of the case, fills it and registers it.
func textureOf(t *testing.T, dev explicit.Device, reg *explicit.TextureRegistry, c gputest.TextureCase) int64 {
	t.Helper()
	format := map[gputest.TextureKind]explicit.TextureFormat{gputest.TextureRGBA8: explicit.TextureFormatRGBA8Unorm,
		gputest.TextureBGRA8: explicit.TextureFormatBGRA8Unorm, gputest.TextureRGBA16F: explicit.TextureFormatRGBA16Float}[c.Kind]
	tex, err := dev.NewTexture(explicit.TextureDesc{Width: gputest.TextureSize, Height: gputest.TextureSize, Format: format, MipLevels: 1, Usage: explicit.TextureSampled})
	if err != nil {
		t.Error(err)
		return 0
	}
	dev.WriteTexture(tex, 0, image.Rect(0, 0, gputest.TextureSize, gputest.TextureSize), c.Texels, gputest.TextureSize*c.Kind.TexelBytes())
	return reg.Register(dev, tex)
}

func TestVulkanATextureLayerMatchesTheImage(t *testing.T) {
	for _, tt := range gputest.TextureCases() {
		t.Run(tt.Name, func(t *testing.T) {
			var got *image.RGBA
			onWindow(t, func(r *explicit.Renderer, win platform.Window, dev explicit.Device, reg *explicit.TextureRegistry) {
				id := textureOf(t, dev, reg, tt)
				var err error
				got, err = r.Draw(win.NativeSurface(), gputest.TextureWindow, gputest.TextureWindow, gputest.TextureScene(id, tt.Quality), true)
				if err != nil {
					t.Error(err)
					return
				}
				if st := r.LastFrame(); st.Skipped != 0 {
					t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
				}
			})
			if got == nil {
				t.Fatal("no frame")
			}
			want, err := soft.Render(gputest.ImageScene(tt.Reference, tt.Quality), gputest.TextureWindow, gputest.TextureWindow)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "vk-texture-"+tt.Name+"-gpu", got)
			dumpImage(t, "vk-texture-"+tt.Name+"-soft", want)
			tt.Tolerance.Check(t, tt.Name, gputest.DiffRect(got, want, got.Bounds()))
			// The texture is there at all: the middle of the layer is not the background.
			c := got.PixOffset(60, 60)
			if got.Pix[c] == 0xE8 && got.Pix[c+1] == 0xE8 && got.Pix[c+2] == 0xF0 {
				t.Errorf("the middle of the texture layer shows the background")
			}
		})
	}
}
