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
	"strconv"
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
	// ClipboardPNG and ClipboardURIs are the other formats of the in-memory
	// system clipboard; Primary is the in-memory primary selection.
	ClipboardPNG  []byte
	ClipboardURIs []string
	Primary       ui.ClipboardContent
	// WindowActions are the actions that a title bar asked of the window, in order.
	WindowActions []ui.WindowAction
	// ViewActions are the same actions with the view they were for.
	ViewActions []ViewAction
	// TextInputStates are the states of the text field that the framework gave
	// to the input method, in order.
	TextInputStates []ui.TextInputState
	// Windows are the windows the framework asked for, in order, destroyed ones
	// included.
	Windows []*FakeWindow
	// CreateErr is returned by CreateWindow, to test a platform that cannot make
	// windows.
	CreateErr error
	// Drags are the drag and drop operations the framework started, in order.
	Drags []ui.DragRequest
	// DragErr is returned by StartDrag, to test a platform that cannot drag.
	DragErr error
	// Icons, ContentTypes and ScreenOn are what the framework set for each view: the
	// window icon, the content type hint, and whether the screen is kept on.
	Icons        map[ui.ViewID]ui.WindowIcon
	ContentTypes map[ui.ViewID]ui.ContentType
	ScreenOn     map[ui.ViewID]bool
	// ParentHandle is the handle of a view for the portals; the default is
	// "test:" and the view ID. ActivationTokenValue is the activation token; the
	// default is "test-token". Both are called on the thread that runs the posted
	// functions.
	ParentHandle         func(view ui.ViewID) string
	ActivationTokenValue string
	// AppIDValue is the application ID of the app; the default is empty.
	AppIDValue string
	// KeepScreenOnErr is returned by KeepScreenOn, to test a platform that cannot
	// keep the screen on.
	KeepScreenOnErr error
	// FocusEvents are the focus changes the framework asked the platform for, in
	// order.
	FocusEvents []ui.ViewFocusEvent

	nextView ui.ViewID
	byView   map[ui.ViewID][]*ui.Scene
}

// FakeWindow is a window the framework asked the fake engine for.
type FakeWindow struct {
	// View is the view of the window.
	View ui.ViewID
	// Spec is what the framework asked for.
	Spec ui.WindowSpec
	// Title is the last title set; the spec's until then.
	Title string
	// Size is the last size asked for; the spec's until then.
	Size ui.Size
	// Min and Max are the last limits set; the spec's until then.
	Min, Max ui.Size
	// Destroyed is whether the framework destroyed the window.
	Destroyed bool
}

// CreateWindow records the window and gives it the next view ID, after the
// implicit one. The view starts at the size asked for, or 800x600.
func (h *Host) CreateWindow(spec ui.WindowSpec) (ui.ViewID, ui.ViewConfiguration, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.CreateErr != nil {
		return 0, ui.ViewConfiguration{}, h.CreateErr
	}
	if err := spec.Icon.Validate(); err != nil {
		return 0, ui.ViewConfiguration{}, err
	}
	h.nextView++
	size := spec.Size
	if size == (ui.Size{}) {
		size = ui.Size{Width: 800, Height: 600}
	}
	h.Windows = append(h.Windows, &FakeWindow{View: h.nextView, Spec: spec, Title: spec.Title, Size: size, Min: spec.MinSize, Max: spec.MaxSize})
	return h.nextView, ui.ViewConfiguration{PhysicalSize: size, DevicePixelRatio: 1}, nil
}

// window finds the window of the view.
func (h *Host) window(view ui.ViewID) *FakeWindow {
	for _, w := range h.Windows {
		if w.View == view {
			return w
		}
	}
	return nil
}

// DestroyWindow marks the window destroyed.
func (h *Host) DestroyWindow(view ui.ViewID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.window(view); w != nil {
		w.Destroyed = true
	}
}

// SetWindowTitle keeps the title.
func (h *Host) SetWindowTitle(view ui.ViewID, title string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.window(view); w != nil {
		w.Title = title
	}
}

// SetWindowSize keeps the size.
func (h *Host) SetWindowSize(view ui.ViewID, size ui.Size) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.window(view); w != nil {
		w.Size = size
	}
}

// SetWindowConstraints keeps the limits.
func (h *Host) SetWindowConstraints(view ui.ViewID, min, max ui.Size) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.window(view); w != nil {
		w.Min, w.Max = min, max
	}
}

// RequestViewFocus keeps the request.
func (h *Host) RequestViewFocus(view ui.ViewID, state ui.ViewFocusState, direction ui.ViewFocusDirection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.FocusEvents = append(h.FocusEvents, ui.ViewFocusEvent{ViewID: view, State: state, Direction: direction})
}

// ScenesOf are the scenes that the view rendered so far.
func (h *Host) ScenesOf(view ui.ViewID) []*ui.Scene {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*ui.Scene(nil), h.byView[view]...)
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
	h.Clipboard, h.HasClipboard, h.ClipboardPNG, h.ClipboardURIs = text, true, nil, nil
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

// SetClipboard keeps the content on the selection of the kind. done is posted.
func (h *Host) SetClipboard(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	h.mu.Lock()
	if kind == ui.ClipboardKindPrimary {
		h.Primary = content
	} else {
		h.Clipboard, h.HasClipboard, h.ClipboardPNG, h.ClipboardURIs = content.Text, content.HasText, content.PNG, content.URIs
	}
	h.mu.Unlock()
	if done != nil {
		h.Post(func() { done(nil) })
	}
}

// GetClipboard posts the content that was kept, in the format.
func (h *Host) GetClipboard(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	h.mu.Lock()
	content := h.Primary
	if kind != ui.ClipboardKindPrimary {
		content = ui.ClipboardContent{Text: h.Clipboard, HasText: h.HasClipboard, PNG: h.ClipboardPNG, URIs: h.ClipboardURIs}
	}
	h.mu.Unlock()
	var out ui.ClipboardContent
	switch format {
	case ui.ClipboardFormatText:
		out.Text, out.HasText = content.Text, content.HasText
	case ui.ClipboardFormatPNG:
		out.PNG = content.PNG
	case ui.ClipboardFormatURIList:
		out.URIs = content.URIs
	}
	h.Post(func() { done(out, ctx.Err()) })
}

// StartDrag records the drag; it fails with DragErr when that is set.
func (h *Host) StartDrag(_ ui.ViewID, request ui.DragRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.DragErr != nil {
		return h.DragErr
	}
	h.Drags = append(h.Drags, request)
	return nil
}

// SetWindowIcon keeps the icon of the view.
func (h *Host) SetWindowIcon(view ui.ViewID, icon ui.WindowIcon) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.Icons == nil {
		h.Icons = map[ui.ViewID]ui.WindowIcon{}
	}
	h.Icons[view] = icon
	return nil
}

// SetWindowContentType keeps the content type of the view.
func (h *Host) SetWindowContentType(view ui.ViewID, contentType ui.ContentType) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ContentTypes == nil {
		h.ContentTypes = map[ui.ViewID]ui.ContentType{}
	}
	h.ContentTypes[view] = contentType
}

// KeepScreenOn keeps the request of the view; it fails with KeepScreenOnErr when
// that is set.
func (h *Host) KeepScreenOn(view ui.ViewID, on bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.KeepScreenOnErr != nil {
		return h.KeepScreenOnErr
	}
	if h.ScreenOn == nil {
		h.ScreenOn = map[ui.ViewID]bool{}
	}
	h.ScreenOn[view] = on
	return nil
}

// PortalParentHandle answers with the handle of the view, posted.
func (h *Host) PortalParentHandle(view ui.ViewID, done func(handle string)) {
	h.mu.Lock()
	f := h.ParentHandle
	h.mu.Unlock()
	handle := "test:" + strconv.Itoa(int(view))
	if f != nil {
		handle = f(view)
	}
	h.Post(func() { done(handle) })
}

// ActivationToken answers with the token, posted.
func (h *Host) ActivationToken(_ ui.ViewID, done func(token string)) {
	h.mu.Lock()
	token := h.ActivationTokenValue
	h.mu.Unlock()
	if token == "" {
		token = "test-token"
	}
	h.Post(func() { done(token) })
}

// AppID is the application ID that the test set.
func (h *Host) AppID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.AppIDValue
}

// PerformWindowAction keeps the action.
func (h *Host) PerformWindowAction(view ui.ViewID, action ui.WindowAction) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.WindowActions = append(h.WindowActions, action)
	h.ViewActions = append(h.ViewActions, ViewAction{View: view, Action: action})
}

// ViewAction is an action of a title bar and the view it was for.
type ViewAction struct {
	// View is the view of the window.
	View ui.ViewID
	// Action is what was asked.
	Action ui.WindowAction
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
func (h *Host) RenderView(view ui.ViewID, scene *ui.Scene) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.scenes = append(h.scenes, scene)
	if h.byView == nil {
		h.byView = map[ui.ViewID][]*ui.Scene{}
	}
	h.byView[view] = append(h.byView[view], scene)
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
