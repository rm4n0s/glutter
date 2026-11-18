//go:build android

// Command android-parity is a development example (step 25.4): the parity program of the Android
// device tests. It draws every proof scene (internal/proof, the goldens of the software renderer)
// with the renderer that GLUTTER_RENDERER names, through the Android platform, and saves the images
// as PNG files in the app's files directory, for the host to compare with the software renderer's
// (internal/gpu/gputest). Nothing it draws is meant to be looked at.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/dev/android-parity
//
// Environment: GLUTTER_RENDERER is gles (the default), vulkan or soft (the software renderer of
// Android, into the window's buffers). The images are in
// files/parity/<renderer>/<scene>.png, with <scene>.txt that says how many commands the renderer
// skipped, and files/parity/<renderer>/DONE when the whole run finished. The same program draws every
// scene with the software renderer too, into files/parity/soft: the host's fonts are not the
// device's, so the device's own software images are what the GPU images are compared with.
//
// With GLUTTER_ISSUE=<name> it runs the reproduction of a reported issue instead (internal/issues;
// atlas-full is the frame of many shadows of step 46.1; soft-mipmaps, a spinner over a picture of
// step 49.1, is the software renderer's), with the same renderer, and writes
// files/issues/<name>/<renderer>/report.txt (the lines of the issue's report), frame.png (the first
// frame) and DONE.
package main

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/nativewindow"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("android-parity failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	renderer := os.Getenv("GLUTTER_RENDERER")
	if renderer == "" {
		renderer = "gles"
	}
	p, err := android.New()
	if err != nil {
		return err
	}
	win, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &parity{win: win, renderer: renderer, cancel: cancel}
	if err := p.Run(ctx, h); err != nil {
		return err
	}
	return h.err
}

// parity draws the scenes at the first frame, when the window has a surface.
type parity struct {
	platform.NopHandler
	win      platform.Window
	renderer string
	cancel   context.CancelFunc
	done     bool
	err      error
}

// OnFrame is the first frame: the surface exists. It draws everything and ends the run.
func (h *parity) OnFrame(ui.ViewID, time.Duration) {
	if h.done {
		return
	}
	h.done = true
	h.err = h.drawAll()
	if h.err != nil {
		slog.Error("android-parity: the run failed", "err", h.err)
	}
	h.cancel()
}

// sceneDrawer draws a scene of the given size and returns its pixels and what the renderer says
// of the frame: how many commands it skipped, and the numbers that the reproduction of an issue
// reports.
type sceneDrawer func(scene *ui.Scene, width, height int) (*image.RGBA, issues.AtlasFullStats, error)

// drawAll attaches the renderer to the window and draws every scene.
func (h *parity) drawAll() error {
	issue := os.Getenv(issues.Env)
	dir := filepath.Join(os.Getenv("HOME"), "parity", h.renderer)
	softDir := filepath.Join(os.Getenv("HOME"), "parity", "soft")
	dirs := []string{dir, softDir}
	if issue != "" {
		dir = filepath.Join(os.Getenv("HOME"), "issues", issue, h.renderer)
		dirs = []string{dir}
	}
	for _, d := range dirs {
		if err := os.RemoveAll(d); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", d)
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", d)
		}
	}
	var (
		draw sceneDrawer
		// mipDraw draws a frame of the reproduction of soft-mipmaps: only the software renderer has it.
		mipDraw issues.SoftMipmapsDraw
		closeFn func()
		device  string
		vkDev   *vulkan.Device
	)
	switch h.renderer {
	case "gles":
		r := gles.NewRenderer()
		closeFn = func() { _ = r.Close() }
		surface, err := r.Attach(h.win.NativeSurface())
		if err != nil {
			closeFn()
			return err
		}
		prev := closeFn
		closeFn = func() { surface.Destroy(); prev() }
		device = r.DeviceName()
		// One frame through the window itself first (drawing the scenes takes longer than the
		// platform waits for the UI thread to let go of a lost surface): it shows that the surface works.
		if _, err := r.RenderCapture(surface, nil, ui.Size{Width: 64, Height: 64}, 1); err != nil {
			closeFn()
			return err
		}
		draw = func(scene *ui.Scene, width, height int) (*image.RGBA, issues.AtlasFullStats, error) {
			img, err := r.Snapshot(scene, ui.Size{Width: float64(width), Height: float64(height)})
			st := r.LastFrame()
			return img, issues.AtlasFullStats{TilesRasterized: st.TilesRasterized, AtlasFull: st.AtlasFull, Skipped: st.Skipped, Unsupported: st.Unsupported}, err
		}
	case "vulkan":
		window := h.win.NativeSurface()
		r := explicit.NewRenderer(func(sh gpu.SurfaceHandle) (explicit.Device, error) {
			d, err := vulkan.Open(vulkan.Options{Surface: sh, AllowCPU: true})
			vkDev = d
			if err != nil {
				return nil, err
			}
			return d, nil
		}, explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true})
		closeFn = r.Close
		// Every scene is drawn through the window: the frame is blitted into the swapchain image,
		// rotated by its pre-transform, and presented; the capture is the frame texture.
		draw = func(scene *ui.Scene, width, height int) (*image.RGBA, issues.AtlasFullStats, error) {
			img, err := r.Draw(window, width, height, scene, true)
			st := r.LastFrame()
			return img, issues.AtlasFullStats{TilesRasterized: st.TilesRasterized, AtlasFull: st.AtlasFull, Skipped: st.Skipped, Unsupported: st.Unsupported}, err
		}
		device = "vulkan"
	case "soft":
		presenter, err := nativewindow.NewPresenter()
		if err != nil {
			return err
		}
		r := nativewindow.NewRenderer(presenter, nativewindow.Options{Background: ui.NewColor(0xFFFFFFFF)})
		surface, err := r.Attach(h.win.NativeSurface())
		if err != nil {
			_ = r.Close()
			return err
		}
		closeFn = func() { surface.Destroy(); _ = r.Close() }
		device = "soft"
		// Every frame is drawn through the window, as the engine draws it.
		draw = func(scene *ui.Scene, width, height int) (*image.RGBA, issues.AtlasFullStats, error) {
			img, err := r.RenderCapture(surface, scene, ui.Size{Width: float64(width), Height: float64(height)}, 1)
			return img, issues.AtlasFullStats{}, err
		}
		mipDraw = func(scene *ui.Scene, width, height int) (*image.RGBA, issues.SoftMipmapsStats, error) {
			img, err := r.RenderCapture(surface, scene, ui.Size{Width: float64(width), Height: float64(height)}, 1)
			st := r.MipmapStats()
			return img, issues.SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, err
		}
	default:
		return errors.New(tagParityRendererUnknown, "GLUTTER_RENDERER is gles, vulkan or soft", "value", h.renderer)
	}
	defer closeFn()
	slog.Info("android-parity: renderer ready", "renderer", h.renderer, "device", device)
	done := proof.Names()
	if issue != "" {
		done = []string{issue}
		if err := drawIssue(issue, dir, draw, mipDraw); err != nil {
			return err
		}
		if vkDev != nil {
			device = vkDev.Description()
			slog.Info("android-parity: Vulkan device", "device", device)
		}
	}
	for _, name := range proof.Names() {
		if issue != "" {
			break
		}
		sc, err := proof.Lookup(name)
		if err != nil {
			return err
		}
		img, st, err := draw(sc.Build(), sc.Width, sc.Height)
		if err != nil {
			return err
		}
		skipped := st.Skipped
		if vkDev != nil && device == "vulkan" {
			device = vkDev.Description()
			slog.Info("android-parity: Vulkan device", "device", device)
		}
		if err := goldens.WritePNG(filepath.Join(dir, name+".png"), img); err != nil {
			return err
		}
		ref, err := soft.Render(sc.Build(), sc.Width, sc.Height)
		if err != nil {
			return err
		}
		if err := goldens.WritePNG(filepath.Join(softDir, name+".png"), ref); err != nil {
			return err
		}
		note := fmt.Sprintf("skipped %d\n", skipped)
		if err := os.WriteFile(filepath.Join(dir, name+".txt"), []byte(note), 0o644); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("scene", name)
		}
		slog.Info("android-parity: scene drawn", "scene", name, "skipped", skipped)
	}
	// What the validation layer said, one message per line (none when it is off or content).
	var messages []string
	if vkDev != nil {
		messages = vkDev.ValidationErrors()
	}
	if err := os.WriteFile(filepath.Join(dir, "validation.txt"), []byte(strings.Join(messages, "\n")), 0o644); err != nil {
		return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", dir)
	}
	if err := os.WriteFile(filepath.Join(dir, "device.txt"), []byte(device+"\n"), 0o644); err != nil {
		return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", dir)
	}
	if err := os.WriteFile(filepath.Join(dir, "DONE"), []byte(strings.Join(done, "\n")+"\n"), 0o644); err != nil {
		return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", dir)
	}
	return nil
}

// drawIssue runs the reproduction of the issue with the renderer (mipDraw is the software
// renderer's, nil for another) and writes its report and its first frame into the directory. An
// error of a frame, which is the issue itself on a renderer without its fix, ends the run with no
// report.
func drawIssue(issue, dir string, draw sceneDrawer, mipDraw issues.SoftMipmapsDraw) error {
	var (
		lines []string
		first *image.RGBA
	)
	switch issue {
	case issues.AtlasFullName:
		rep, img, err := issues.RunAtlasFull(issues.AtlasFullDraw(draw))
		if err != nil {
			return err
		}
		lines, first = rep.Lines(), img
	case issues.SoftMipmapsName:
		if mipDraw == nil {
			return errors.New(tagParityIssueRenderer, "the issue is the software renderer's: run it with GLUTTER_RENDERER=soft", "issue", issue)
		}
		rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, width, height int) (*image.RGBA, issues.SoftMipmapsStats, error) {
			img, st, err := mipDraw(scene, width, height)
			if first == nil {
				first = img
			}
			return img, st, err
		})
		if err != nil {
			return err
		}
		lines = rep.Lines()
	default:
		return errors.New(tagParityIssueUnknown, "GLUTTER_ISSUE is not an issue that the program reproduces", "value", issue, "known", issues.AtlasFullName+" "+issues.SoftMipmapsName)
	}
	for _, line := range lines {
		slog.Info("android-parity: " + line)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return errors.NewErr(tagParityWriteFailed, err).SetMetadata("issue", issue)
	}
	return goldens.WritePNG(filepath.Join(dir, "frame.png"), first)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagParityRendererUnknown = "ParityRendererUnknown"
	tagParityIssueUnknown    = "ParityIssueUnknown"
	tagParityIssueRenderer   = "ParityIssueRenderer"
	tagParityWriteFailed     = "ParityWriteFailed"
)
