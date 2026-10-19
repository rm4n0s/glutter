// Package renderhost builds a RendererBinding on a fake engine with one view,
// lays out and paints a render tree through the pipeline, and keeps the scenes
// that the view rendered. The rendering tests (internal/renderingtest) and the
// proof scenes that run a render tree (internal/proof) share it. It has no
// renderer: the caller turns a scene into pixels.
//
// Flutter: no counterpart (test/rendering/rendering_tester.dart)
package renderhost

import (
	"context"
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
	// ExitRequests counts the calls of RequestAppExit.
	ExitRequests int
	// RenderErr is returned by RenderView, to test a failing renderer.
	RenderErr error
	// SemanticsUpdates are the semantics updates that the view sent.
	SemanticsUpdates []*ui.SemanticsUpdate
	// SemanticsTreeEnabled is what the framework last said about producing a
	// semantics tree.
	SemanticsTreeEnabled bool
	// ActivatedCursors are the system cursors that the framework activated, in
	// order.
	ActivatedCursors []CursorActivation
	// Clipboard is the in-memory clipboard; HasClipboard says whether it holds
	// text. A test can set them to play another application.
	Clipboard    string
	HasClipboard bool
	// TextInputStates are the states of the text field that the framework gave
	// to the input method, in order.
	TextInputStates []ui.TextInputState
}

// SetTextInput keeps the state.
func (h *Host) SetTextInput(state ui.TextInputState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.TextInputStates = append(h.TextInputStates, state)
}

// Lock and Unlock guard the fields of the host that tests set.
func (h *Host) Lock() { h.mu.Lock() }

// Unlock ends what Lock began.
func (h *Host) Unlock() { h.mu.Unlock() }

// SetClipboardText keeps the text. done is posted, like the engine's results.
func (h *Host) SetClipboardText(_ context.Context, text string, done func(error)) {
	h.mu.Lock()
	h.Clipboard, h.HasClipboard = text, true
	h.mu.Unlock()
	if done != nil {
		h.Post(func() { done(nil) })
	}
}

// GetClipboardText posts the text that was kept.
func (h *Host) GetClipboardText(ctx context.Context, done func(string, bool, error)) {
	h.mu.Lock()
	text, ok := h.Clipboard, h.HasClipboard
	h.mu.Unlock()
	h.Post(func() { done(text, ok, ctx.Err()) })
}

// CursorActivation is one call of ActivateSystemCursor.
type CursorActivation struct {
	// Device is the pointer device the cursor is for.
	Device int
	// Kind is the kind of the system cursor, such as "click".
	Kind string
}

// ActivateSystemCursor keeps the activation.
func (h *Host) ActivateSystemCursor(device int, kind string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ActivatedCursors = append(h.ActivatedCursors, CursorActivation{device, kind})
}

// UpdateSemantics keeps the update.
func (h *Host) UpdateSemantics(_ ui.ViewID, update *ui.SemanticsUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.SemanticsUpdates = append(h.SemanticsUpdates, update)
}

// SetSemanticsTreeEnabled keeps the flag.
func (h *Host) SetSemanticsTreeEnabled(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.SemanticsTreeEnabled = enabled
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

// RequestAppExit counts the requests.
func (h *Host) RequestAppExit() { h.mu.Lock(); h.ExitRequests++; h.mu.Unlock() }

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
	s.Owner = rendering.NewViewPipelineOwner()
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

// Scenes are the scenes that the views rendered so far.
func (h *Host) Scenes() []*ui.Scene {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*ui.Scene(nil), h.scenes...)
}
