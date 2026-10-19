// Package renderingtest is a temporary harness for the tests of the rendering
// library: it builds a RendererBinding on a fake engine, lays out and paints a
// render tree through the pipeline, and renders the scene with the software
// renderer. It stands in for gluttertest (step 7.5), which replaces it.
//
// Flutter: no counterpart (test/rendering/rendering_tester.dart)
package renderingtest

import (
	"image"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// Host is the fake engine: it remembers the last scene a view rendered and
// keeps the functions posted to the "UI thread" until the test runs them.
type Host struct {
	mu        sync.Mutex
	posted    []func()
	scenes    []*ui.Scene
	Scheduled int
	// RenderErr is returned by RenderView, to test a failing renderer.
	RenderErr error
}

// ScheduleFrame counts the frames asked for.
func (h *Host) ScheduleFrame() { h.mu.Lock(); h.Scheduled++; h.mu.Unlock() }

// RenderView keeps the scene.
func (h *Host) RenderView(_ ui.ViewID, scene *ui.Scene) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.scenes = append(h.scenes, scene)
	return h.RenderErr
}

// RequestAppExit does nothing.
func (h *Host) RequestAppExit() {}

// Post keeps the function.
func (h *Host) Post(fn func()) { h.mu.Lock(); h.posted = append(h.posted, fn); h.mu.Unlock() }

// RunPosted runs what was posted, including what that posts.
func (h *Host) RunPosted() int {
	n := 0
	for {
		h.mu.Lock()
		if len(h.posted) == 0 {
			h.mu.Unlock()
			return n
		}
		fn := h.posted[0]
		h.posted = h.posted[1:]
		h.mu.Unlock()
		fn()
		n++
	}
}

// Opts are the optional parameters of New.
type Opts struct {
	// Size is the size of the view in physical pixels; the default is 800x600.
	Size ui.Size
	// DevicePixelRatio defaults to 1.
	DevicePixelRatio float64
	// Debug are the debug flags; the default is a fresh set.
	Debug *rendering.DebugOptions
}

// Harness is a rendering binding on a fake engine, with one view.
type Harness struct {
	t          *testing.T
	Host       *Host
	Dispatcher *ui.PlatformDispatcher
	Scheduler  *scheduler.SchedulerBinding
	Gestures   *gestures.GestureBinding
	Binding    *rendering.RendererBinding
	View       *ui.View
	RenderView *rendering.RenderView
	Owner      *rendering.PipelineOwner
	// Errors collects the errors that the binding reported.
	Errors []error

	raw time.Duration
}

// New creates the harness with the render view attached to the binding's root
// pipeline owner, ready for SetRoot.
func New(t *testing.T, opts ...Opts) *Harness {
	t.Helper()
	var o Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Size == (ui.Size{}) {
		o.Size = ui.Size{Width: 800, Height: 600}
	}
	if o.DevicePixelRatio == 0 {
		o.DevicePixelRatio = 1
	}
	h := &Harness{t: t, Host: &Host{}}
	h.Dispatcher = ui.NewPlatformDispatcher(h.Host)
	h.Scheduler = scheduler.NewSchedulerBinding(h.Dispatcher)
	h.Gestures = gestures.NewGestureBinding(h.Dispatcher, h.Scheduler)
	h.Binding = rendering.NewRendererBinding(h.Dispatcher, h.Scheduler, h.Gestures, rendering.RendererBindingOpts{
		Debug:       o.Debug,
		ReportError: func(err error) { h.Errors = append(h.Errors, err) },
	})
	h.View = h.Dispatcher.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: o.Size, DevicePixelRatio: o.DevicePixelRatio})
	h.RenderView = rendering.NewRenderView(h.View)
	h.Owner = rendering.NewPipelineOwner(rendering.PipelineOwnerOpts{})
	h.Binding.RootPipelineOwner().AdoptChild(h.Owner)
	h.Owner.SetRootNode(h.RenderView)
	h.Binding.AddRenderView(h.RenderView)
	h.RenderView.PrepareInitialFrame()
	return h
}

// SetRoot makes the box the child of the render view.
func (h *Harness) SetRoot(box rendering.Box) { h.RenderView.SetChild(box) }

// Pump runs one frame: layout, compositing bits, paint and composite.
func (h *Harness) Pump() {
	h.t.Helper()
	h.raw += 16 * time.Millisecond
	h.Scheduler.HandleBeginFrame(foundation.Some(h.raw))
	h.Scheduler.HandleDrawFrame()
}

// FlushPipeline runs layout, compositing bits and paint on the owner only, not
// the composite step.
func (h *Harness) FlushPipeline() {
	h.Owner.FlushLayout()
	h.Owner.FlushCompositingBits()
	h.Owner.FlushPaint()
}

// Scenes are the scenes that the view rendered so far.
func (h *Harness) Scenes() []*ui.Scene {
	h.Host.mu.Lock()
	defer h.Host.mu.Unlock()
	return append([]*ui.Scene(nil), h.Host.scenes...)
}

// LastScene is the scene of the last composited frame, or nil.
func (h *Harness) LastScene() *ui.Scene {
	s := h.Scenes()
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

// Image renders the last scene with the software renderer.
func (h *Harness) Image() (*image.RGBA, error) {
	s := h.LastScene()
	if s == nil {
		return nil, errors.New("RenderingTestNoScene", "no frame was composited yet; call Pump first")
	}
	size := h.View.PhysicalSize()
	return soft.Render(s, int(size.Width), int(size.Height))
}
