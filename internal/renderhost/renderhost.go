// Package renderhost builds a RendererBinding on a fake engine with one view,
// lays out and paints a render tree through the pipeline, and keeps the scenes
// that the view rendered. The rendering tests (internal/renderingtest) and the
// proof scenes that run a render tree (internal/proof) share it. It has no
// renderer: the caller turns a scene into pixels.
//
// Flutter: no counterpart (test/rendering/rendering_tester.dart)
package renderhost

import (
	"sync"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// Host is the fake engine: it remembers the scenes a view rendered and keeps
// the functions posted to the "UI thread" until the caller runs them.
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
	// TextEngine lays out the text; the default is none.
	TextEngine ui.ParagraphFactory
	// ReportError gets the errors that the binding reports.
	ReportError func(error)
}

// Stack is a rendering binding on a fake engine, with one view.
type Stack struct {
	Host       *Host
	Dispatcher *ui.PlatformDispatcher
	Scheduler  *scheduler.SchedulerBinding
	Gestures   *gestures.GestureBinding
	Binding    *rendering.RendererBinding
	View       *ui.View
	RenderView *rendering.RenderView
	Owner      *rendering.PipelineOwner

	raw time.Duration
}

// New creates the stack with the render view attached to the binding's root
// pipeline owner, ready for SetRoot.
func New(opts ...Opts) *Stack {
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
	s := &Stack{Host: &Host{}}
	s.Dispatcher = ui.NewPlatformDispatcher(s.Host)
	if o.TextEngine != nil {
		s.Dispatcher.SetTextEngine(o.TextEngine)
	}
	s.Scheduler = scheduler.NewSchedulerBinding(s.Dispatcher)
	s.Gestures = gestures.NewGestureBinding(s.Dispatcher, s.Scheduler)
	s.Binding = rendering.NewRendererBinding(s.Dispatcher, s.Scheduler, s.Gestures, rendering.RendererBindingOpts{
		Debug:       o.Debug,
		ReportError: o.ReportError,
	})
	s.View = s.Dispatcher.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: o.Size, DevicePixelRatio: o.DevicePixelRatio})
	s.RenderView = rendering.NewRenderView(s.View)
	s.Owner = rendering.NewPipelineOwner(rendering.PipelineOwnerOpts{})
	s.Binding.RootPipelineOwner().AdoptChild(s.Owner)
	s.Owner.SetRootNode(s.RenderView)
	s.Binding.AddRenderView(s.RenderView)
	s.RenderView.PrepareInitialFrame()
	return s
}

// SetRoot makes the box the child of the render view.
func (s *Stack) SetRoot(box rendering.Box) { s.RenderView.SetChild(box) }

// Pump runs one frame: layout, compositing bits, paint and composite.
func (s *Stack) Pump() {
	s.raw += 16 * time.Millisecond
	s.Scheduler.HandleBeginFrame(foundation.Some(s.raw))
	s.Scheduler.HandleDrawFrame()
}

// FlushPipeline runs layout, compositing bits and paint on the owner only, not
// the composite step.
func (s *Stack) FlushPipeline() {
	s.Owner.FlushLayout()
	s.Owner.FlushCompositingBits()
	s.Owner.FlushPaint()
}

// Scenes are the scenes that the view rendered so far.
func (s *Stack) Scenes() []*ui.Scene {
	s.Host.mu.Lock()
	defer s.Host.mu.Unlock()
	return append([]*ui.Scene(nil), s.Host.scenes...)
}

// LastScene is the scene of the last composited frame, or nil.
func (s *Stack) LastScene() *ui.Scene {
	sc := s.Scenes()
	if len(sc) == 0 {
		return nil
	}
	return sc[len(sc)-1]
}
