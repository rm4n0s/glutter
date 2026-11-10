//go:build ios

// Command sceneapp is the app of the Simulator tests of step 35.2: it runs the iOS platform with a handler of its own that draws a
// flat color into the window's pixels and prints a line for each metrics change, frame and lifecycle change, so that the test sees
// them in the console of the app. A goroutine asks for a frame ten times a second, so a window that may draw shows frames and one
// that may not shows none.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/iostest/scenelines"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/ios"
	"github.com/rm4n0s/glutter/ui"
)

// handler is a platform.Handler that draws a flat color and says what happens.
type handler struct {
	platform.NopHandler
	p      *ios.Platform
	w      platform.Window
	size   ui.Size
	frames int
	turned bool
}

func (h *handler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	h.size = m.PhysicalSize
	fmt.Printf("%s %dx%d scale=%g padding=%g,%g,%g,%g\n", scenelines.Metrics, int(m.PhysicalSize.Width), int(m.PhysicalSize.Height),
		m.DevicePixelRatio, m.ViewPadding.Left, m.ViewPadding.Top, m.ViewPadding.Right, m.ViewPadding.Bottom)
}

func (h *handler) OnLifecycle(state ui.AppLifecycleState) {
	fmt.Printf("%s %s\n", scenelines.Lifecycle, scenelines.StateName(state))
}

func (h *handler) OnFrame(_ ui.ViewID, _ time.Duration) {
	target, ok := h.w.(gpu.PixelTarget)
	if !ok || h.size.Width == 0 {
		return
	}
	buf, err := target.AcquirePixels(int(h.size.Width), int(h.size.Height))
	if err != nil {
		fmt.Printf("%s acquire: %v\n", scenelines.Failed, err)
		return
	}
	for y := 0; y < buf.Height; y++ {
		row := buf.Pix[y*buf.Stride : y*buf.Stride+buf.Width*4]
		for x := 0; x < len(row); x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = 0x80, 0x40, 0x10, 0xff // BGRX: a dark blue
		}
	}
	if err := target.PresentPixels(nil); err != nil {
		fmt.Printf("%s present: %v\n", scenelines.Failed, err)
	}
	h.frames++
	fmt.Printf("%s %d %dx%d\n", scenelines.Frame, h.frames, buf.Width, buf.Height)
	if os.Getenv(scenelines.RotateEnv) == "landscape" && !h.turned {
		h.turned = true
		h.p.RequestOrientations(ios.OrientationLandscape)
	}
}

func main() {
	p, err := ios.New(ios.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sceneapp:", err)
		os.Exit(1)
	}
	w, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sceneapp:", err)
		os.Exit(1)
	}
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			p.Post(w.RequestFrame)
		}
	}()
	h := &handler{p: p, w: w}
	if err := p.Run(context.Background(), h); err != nil {
		fmt.Fprintln(os.Stderr, "sceneapp:", err)
		os.Exit(1)
	}
}
