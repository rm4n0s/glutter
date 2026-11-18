//go:build ios

// Command issueapp is the app of the Simulator tests of reported issues (internal/issues; steps 47.1 and 50.1). It runs the iOS
// platform with a handler of its own and, at the first frame, runs the reproduction that GLUTTER_ISSUE (issues.Env) names, prints
// the lines of its report and then issues.DoneLine, or issues.FailedLine with the error:
//
//   - atlas-full (step 47.1): the explicit renderer over the Metal device of the Simulator, opened as the engine opens it and strict
//     (nothing left out), draws the frames of issues.RunAtlasFull into a texture of their size (1920x1080, whatever the phone's
//     screen is) and reads each frame back.
//   - soft-mipmaps (step 50.1): shmwindow, the software renderer of the engine, draws the frames of issues.RunSoftMipmaps into the
//     IOSurfaces of the window, one frame each, with the numbers of its cache of mipmap chains.
package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpu/shmwindow"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/ios"
	"github.com/rm4n0s/glutter/ui"
)

// handler is a platform.Handler that runs the issue at the first frame.
type handler struct {
	platform.NopHandler
	w    platform.Window
	size ui.Size
	ran  bool
}

func (h *handler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) { h.size = m.PhysicalSize }

func (h *handler) OnFrame(_ ui.ViewID, _ time.Duration) {
	if h.ran || h.size.Width == 0 {
		return
	}
	h.ran = true
	lines, err := h.run(os.Getenv(issues.Env))
	for _, line := range lines {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Printf("%s %+v\n", issues.FailedLine, err)
		return
	}
	fmt.Println(issues.DoneLine)
}

// run runs the issue and returns the lines of its report, those of the frames before an error too.
func (h *handler) run(issue string) ([]string, error) {
	switch issue {
	case issues.AtlasFullName:
		return runAtlasFull()
	case issues.SoftMipmapsName:
		return h.runSoftMipmaps()
	}
	return nil, errors.New(tagIssueAppIssueUnknown, "GLUTTER_ISSUE is not an issue that the app reproduces", "value", issue,
		"known", issues.AtlasFullName+" "+issues.SoftMipmapsName)
}

// runAtlasFull draws the frames of the atlas issue with the explicit renderer on Metal, strict, as newMetalRenderer of the engine
// makes it, into an off-screen swap chain of the frame's size.
func runAtlasFull() ([]string, error) {
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := metal.Open(metal.Options{})
		if err != nil {
			return nil, err
		}
		return offscreenDevice{d}, nil
	}, explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true})
	defer r.Close()
	rep, _, err := issues.RunAtlasFull(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.AtlasFullStats, error) {
		img, err := r.Draw(gpu.SurfaceHandle{}, w, h, scene, true)
		if err != nil {
			return nil, issues.AtlasFullStats{}, err
		}
		st := r.LastFrame()
		return img, issues.AtlasFullStats{TilesRasterized: st.TilesRasterized, AtlasFull: st.AtlasFull, Skipped: st.Skipped, Unsupported: st.Unsupported}, nil
	})
	return rep.Lines(), err
}

// runSoftMipmaps draws the frames of the mipmap issue with shmwindow into the window's IOSurfaces, as newSoftWindowRenderer of the
// engine does, one frame each, and reads the renderer's cache after each.
func (h *handler) runSoftMipmaps() ([]string, error) {
	target, ok := h.w.(gpu.PixelTarget)
	if !ok {
		return nil, errors.New(tagIssueAppNoPixels, "the window of the iOS platform is not a gpu.PixelTarget")
	}
	r := shmwindow.NewRenderer(func(gpu.SurfaceHandle) (gpu.PixelTarget, bool) { return target, true })
	defer func() { _ = r.Close() }()
	s, err := r.Attach(h.w.NativeSurface())
	if err != nil {
		return nil, err
	}
	defer s.Destroy()
	rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, w, hgt int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		img, err := r.RenderCapture(s, scene, ui.Size{Width: float64(w), Height: float64(hgt)}, 1, true)
		if err != nil {
			return nil, issues.SoftMipmapsStats{}, err
		}
		st := r.MipmapStats()
		return img, issues.SoftMipmapsStats{Built: st.Built, Chains: st.Chains}, nil
	})
	return rep.Lines(), err
}

// offscreenSwapchain is a swap chain that is one BGRA texture: Acquire returns it, Present does nothing. The frames of the atlas
// issue are larger than the phone's screen, and the renderer reads each one back from the texture.
type offscreenSwapchain struct {
	d    *metal.Device
	tex  explicit.Texture
	w, h int
}

func (s *offscreenSwapchain) Configure(w, h int) error {
	if s.tex != nil && s.w == w && s.h == h {
		return nil
	}
	s.Destroy()
	tex, err := s.d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatBGRA8Unorm, MipLevels: 1,
		Usage: explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource})
	if err != nil {
		return err
	}
	s.tex, s.w, s.h = tex, w, h
	return nil
}
func (s *offscreenSwapchain) PreRotation() explicit.Rotation     { return explicit.Rotation0 }
func (s *offscreenSwapchain) Stale() bool                        { return false }
func (s *offscreenSwapchain) Acquire() (explicit.Texture, error) { return s.tex, nil }
func (s *offscreenSwapchain) Present([]image.Rectangle) error    { return nil }
func (s *offscreenSwapchain) Destroy() {
	if s.tex != nil {
		s.tex.Destroy()
		s.tex = nil
	}
}

// offscreenDevice is a Metal device whose swap chains are offscreenSwapchains.
type offscreenDevice struct{ *metal.Device }

func (o offscreenDevice) NewSwapchain(gpu.SurfaceHandle) (explicit.Swapchain, error) {
	return &offscreenSwapchain{d: o.Device}, nil
}

func main() {
	p, err := ios.New(ios.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "issueapp:", err)
		os.Exit(1)
	}
	w, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "issueapp:", err)
		os.Exit(1)
	}
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			p.Post(w.RequestFrame)
		}
	}()
	if err := p.Run(context.Background(), &handler{w: w}); err != nil {
		fmt.Fprintln(os.Stderr, "issueapp:", err)
		os.Exit(1)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIssueAppIssueUnknown = "IssueAppIssueUnknown"
	tagIssueAppNoPixels     = "IssueAppNoPixels"
)
