//go:build linux

package decor

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// libraryName is the versioned name of libdecor; the unversioned symlink exists
// only when the -devel package is installed (SPEC §8.1).
const libraryName = "libdecor-0.so.0"

// LibraryEnv is the environment variable that holds the absolute path of the
// libdecor to load, for unusual setups.
//
// Flutter: no counterpart
const LibraryEnv = "GLUTTER_LIBDECOR_LIBRARY"

// Error is a libdecor_error: what libdecor reports through its error callback.
//
// Flutter: no counterpart
type Error int32

// The values of Error.
const (
	// ErrorCompositorIncompatible is LIBDECOR_ERROR_COMPOSITOR_INCOMPATIBLE: the
	// compositor lacks a protocol that libdecor needs.
	ErrorCompositorIncompatible Error = 0
	// ErrorInvalidFrameConfiguration is LIBDECOR_ERROR_INVALID_FRAME_CONFIGURATION:
	// a frame was committed with a configuration that is not valid.
	ErrorInvalidFrameConfiguration Error = 1
)

// State is a bit set of libdecor_window_state.
//
// Flutter: no counterpart
type State uint32

// The bits of State.
const (
	// StateActive is the window has the focus.
	StateActive State = 1 << 0
	// StateMaximized is the window is maximized.
	StateMaximized State = 1 << 1
	// StateFullscreen is the window is fullscreen.
	StateFullscreen State = 1 << 2
	// StateTiledLeft is the left edge touches another window or the screen.
	StateTiledLeft State = 1 << 3
	// StateTiledRight is the right edge is tiled.
	StateTiledRight State = 1 << 4
	// StateTiledTop is the top edge is tiled.
	StateTiledTop State = 1 << 5
	// StateTiledBottom is the bottom edge is tiled.
	StateTiledBottom State = 1 << 6
	// StateSuspended is the compositor hides the window.
	StateSuspended State = 1 << 7
)

// ResizeEdge is a libdecor_resize_edge.
//
// Flutter: no counterpart
type ResizeEdge int32

// The values of ResizeEdge.
const (
	// ResizeNone is no edge.
	ResizeNone ResizeEdge = 0
	// ResizeTop is the top edge.
	ResizeTop ResizeEdge = 1
	// ResizeBottom is the bottom edge.
	ResizeBottom ResizeEdge = 2
	// ResizeLeft is the left edge.
	ResizeLeft ResizeEdge = 3
	// ResizeTopLeft is the top left corner.
	ResizeTopLeft ResizeEdge = 4
	// ResizeTopRight is the top right corner.
	ResizeTopRight ResizeEdge = 5
	// ResizeRight is the right edge.
	ResizeRight ResizeEdge = 6
	// ResizeBottomLeft is the bottom left corner.
	ResizeBottomLeft ResizeEdge = 7
	// ResizeBottomRight is the bottom right corner.
	ResizeBottomRight ResizeEdge = 8
)

// Capabilities is a bit set of libdecor_capabilities: what the user may do with
// the window through its decorations.
//
// Flutter: no counterpart
type Capabilities uint32

// The bits of Capabilities.
const (
	// CanMove lets the user move the window.
	CanMove Capabilities = 1 << 0
	// CanResize lets the user resize the window.
	CanResize Capabilities = 1 << 1
	// CanMinimize lets the user minimize the window.
	CanMinimize Capabilities = 1 << 2
	// CanFullscreen lets the user make the window fullscreen.
	CanFullscreen Capabilities = 1 << 3
	// CanClose lets the user close the window.
	CanClose Capabilities = 1 << 4
)

// interfaceStruct mirrors struct libdecor_interface: the error callback and ten
// reserved pointers.
type interfaceStruct struct {
	errorFn  uintptr
	reserved [10]uintptr
}

// frameInterfaceStruct mirrors struct libdecor_frame_interface: the four frame
// callbacks and nine reserved pointers.
type frameInterfaceStruct struct {
	configure    uintptr
	closeFn      uintptr
	commit       uintptr
	dismissPopup uintptr
	reserved     [9]uintptr
}

// api is the set of libdecor functions that this package calls.
type api struct {
	new      func(display, iface uintptr) uintptr
	unref    func(ctx uintptr)
	getFD    func(ctx uintptr) int32
	dispatch func(ctx uintptr, timeout int32) int32

	decorate func(ctx, surface, iface, userData uintptr) uintptr
	frameRef func(frame uintptr)
	unrefFr  func(frame uintptr)
	setVis   func(frame uintptr, visible bool)
	setPar   func(frame, parent uintptr)
	setTitle func(frame, title uintptr)
	setAppID func(frame, appID uintptr)
	setCaps  func(frame uintptr, caps uint32)
	unsetCap func(frame uintptr, caps uint32)
	hasCap   func(frame uintptr, caps uint32) bool
	getCaps  func(frame uintptr) uint32
	setMin   func(frame uintptr, w, h int32)
	setMax   func(frame uintptr, w, h int32)
	showMenu func(frame, seat uintptr, serial uint32, x, y int32)
	resize   func(frame, seat uintptr, serial uint32, edge int32)
	move     func(frame, seat uintptr, serial uint32)
	commit   func(frame, state, configuration uintptr)
	setMini  func(frame uintptr)
	setMaxd  func(frame uintptr)
	unsetMax func(frame uintptr)
	setFull  func(frame, output uintptr)
	unsetFul func(frame uintptr)
	isFloat  func(frame uintptr) bool
	closeFr  func(frame uintptr)
	mapFrame func(frame uintptr)
	getState func(frame uintptr) uint32
	getTop   func(frame uintptr) uintptr

	stateNew    func(w, h int32) uintptr
	stateFree   func(state uintptr)
	confSize    func(conf, frame, w, h uintptr) bool
	confWinSate func(conf, state uintptr) bool
}

var loadAPI = sync.OnceValues(func() (*api, error) {
	lib, err := dl.OpenOverride(LibraryEnv, libraryName)
	if err != nil {
		return nil, err
	}
	a := new(api)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&a.new, "libdecor_new"}, {&a.unref, "libdecor_unref"}, {&a.getFD, "libdecor_get_fd"}, {&a.dispatch, "libdecor_dispatch"},
		{&a.decorate, "libdecor_decorate"}, {&a.frameRef, "libdecor_frame_ref"}, {&a.unrefFr, "libdecor_frame_unref"},
		{&a.setVis, "libdecor_frame_set_visibility"}, {&a.setPar, "libdecor_frame_set_parent"},
		{&a.setTitle, "libdecor_frame_set_title"}, {&a.setAppID, "libdecor_frame_set_app_id"},
		{&a.setCaps, "libdecor_frame_set_capabilities"}, {&a.unsetCap, "libdecor_frame_unset_capabilities"},
		{&a.hasCap, "libdecor_frame_has_capability"}, {&a.getCaps, "libdecor_frame_get_capabilities"},
		{&a.setMin, "libdecor_frame_set_min_content_size"}, {&a.setMax, "libdecor_frame_set_max_content_size"},
		{&a.showMenu, "libdecor_frame_show_window_menu"}, {&a.resize, "libdecor_frame_resize"}, {&a.move, "libdecor_frame_move"},
		{&a.commit, "libdecor_frame_commit"}, {&a.setMini, "libdecor_frame_set_minimized"},
		{&a.setMaxd, "libdecor_frame_set_maximized"}, {&a.unsetMax, "libdecor_frame_unset_maximized"},
		{&a.setFull, "libdecor_frame_set_fullscreen"}, {&a.unsetFul, "libdecor_frame_unset_fullscreen"},
		{&a.isFloat, "libdecor_frame_is_floating"}, {&a.closeFr, "libdecor_frame_close"}, {&a.mapFrame, "libdecor_frame_map"},
		{&a.getState, "libdecor_frame_get_window_state"}, {&a.getTop, "libdecor_frame_get_xdg_toplevel"},
		{&a.stateNew, "libdecor_state_new"}, {&a.stateFree, "libdecor_state_free"},
		{&a.confSize, "libdecor_configuration_get_content_size"}, {&a.confWinSate, "libdecor_configuration_get_window_state"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return a, nil
})

// Available reports whether libdecor can be loaded.
//
// Flutter: no counterpart
func Available() bool {
	_, err := loadAPI()
	return err == nil
}

// registry is the process-wide table that the callbacks use to find the Go
// objects. The callbacks are created once (SPEC §8.1 rule 2); a context is
// found by its pointer, a frame by the user data, which is its handle.
type registry struct {
	mu       sync.Mutex
	contexts map[uintptr]*Context
	frames   map[uintptr]*Frame
	nextID   uintptr

	once     sync.Once
	iface    uintptr // struct libdecor_interface, in C memory
	frameIfc uintptr // struct libdecor_frame_interface, in C memory
}

// callbacks is the registry of the process: it only maps the pointers that C
// holds to the Go objects that own them.
var callbacks = &registry{contexts: map[uintptr]*Context{}, frames: map[uintptr]*Frame{}}

// setup makes the callbacks and the C structs that hold them, once.
func (r *registry) setup() {
	r.once.Do(func() {
		r.iface = dl.Calloc(1, unsafe.Sizeof(interfaceStruct{}))
		(*interfaceStruct)(dl.Pointer(r.iface)).errorFn = dl.Callback(onError)
		r.frameIfc = dl.Calloc(1, unsafe.Sizeof(frameInterfaceStruct{}))
		f := (*frameInterfaceStruct)(dl.Pointer(r.frameIfc))
		f.configure = dl.Callback(onConfigure)
		f.closeFn = dl.Callback(onClose)
		f.commit = dl.Callback(onCommit)
		f.dismissPopup = dl.Callback(onDismissPopup)
	})
}

func (r *registry) frame(userData uintptr) *Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.frames[userData]
}

// guard keeps a panic of a Go handler from unwinding through the C frames of
// libdecor: the first one is kept in the context and raised again when the call
// into libdecor returns (Dispatch).
func (c *Context) guard() {
	if r := recover(); r != nil && c.panicValue == nil {
		c.panicValue = &panicBox{value: r}
	}
}

type panicBox struct{ value any }

func onError(ctx uintptr, code int32, message uintptr) {
	callbacks.mu.Lock()
	c := callbacks.contexts[ctx]
	callbacks.mu.Unlock()
	if c == nil || c.onError == nil {
		return
	}
	defer c.guard()
	c.onError(Error(code), dl.GoString(message))
}

func onConfigure(frame, configuration, userData uintptr) {
	f := callbacks.frame(userData)
	if f == nil {
		return
	}
	defer f.ctx.guard()
	f.handler.Configure(Configuration{ptr: configuration, frame: f})
}

func onClose(frame, userData uintptr) {
	f := callbacks.frame(userData)
	if f == nil {
		return
	}
	defer f.ctx.guard()
	f.handler.Close()
}

func onCommit(frame, userData uintptr) {
	f := callbacks.frame(userData)
	if f == nil {
		return
	}
	defer f.ctx.guard()
	f.handler.Commit()
}

func onDismissPopup(frame, seatName, userData uintptr) {
	f := callbacks.frame(userData)
	if f == nil {
		return
	}
	defer f.ctx.guard()
	f.handler.DismissPopup(dl.GoString(seatName))
}

// Handler receives the events of a frame. Its methods are called inside
// Context.Dispatch, on the UI thread.
//
// Flutter: no counterpart
type Handler interface {
	// Configure is called when the compositor or libdecor wants a new size or state.
	// The window must answer with Frame.Commit, passing the configuration.
	Configure(c Configuration)
	// Close is called when the user asked to close the window.
	Close()
	// Commit is called when the window must commit its wl_surface because the
	// decoration changed.
	Commit()
	// DismissPopup is called when a popup of the decoration (the window menu)
	// must go away, for the seat with the name.
	DismissPopup(seat string)
}

// Context is a libdecor context: the connection of libdecor to the display.
//
// Flutter: no counterpart
type Context struct {
	api        *api
	ptr        uintptr
	onError    func(code Error, message string)
	panicValue *panicBox
}

// New makes the libdecor context for a wl_display. It fails with
// LibraryLoadFailed or SymbolMissing when libdecor cannot be used, and with
// LibdecorInitFailed when libdecor_new returns NULL (for example when no plugin
// loads). onError, when not nil, gets the errors that libdecor reports later.
//
// Flutter: no counterpart
func New(display uintptr, onError func(code Error, message string)) (*Context, error) {
	a, err := loadAPI()
	if err != nil {
		return nil, err
	}
	return newContext(a, display, onError)
}

func newContext(a *api, display uintptr, onError func(code Error, message string)) (*Context, error) {
	callbacks.setup()
	c := &Context{api: a, onError: onError}
	// libdecor calls the error callback from inside libdecor_new, before the
	// pointer is known; the callback finds the context only afterwards, which is
	// fine: initialisation errors make libdecor_new return NULL.
	c.ptr = a.new(display, callbacks.iface)
	if c.ptr == 0 {
		return nil, errors.New("LibdecorInitFailed", "libdecor_new failed: no plugin could be loaded or the compositor is not supported")
	}
	callbacks.mu.Lock()
	callbacks.contexts[c.ptr] = c
	callbacks.mu.Unlock()
	return c, nil
}

// FD is the file descriptor to wait on for the events of libdecor.
//
// Flutter: no counterpart
func (c *Context) FD() int { return int(c.api.getFD(c.ptr)) }

// Dispatch handles the events that are ready, without waiting. A panic in a
// handler is raised here, after libdecor returns.
//
// Flutter: no counterpart
func (c *Context) Dispatch() error {
	if c.api.dispatch(c.ptr, 0) < 0 {
		return errors.New("LibdecorDispatchFailed", "libdecor_dispatch failed")
	}
	if p := c.panicValue; p != nil {
		c.panicValue = nil
		panic(p.value)
	}
	return nil
}

// Close releases the context. Close the frames first.
//
// Flutter: no counterpart
func (c *Context) Close() {
	if c.ptr == 0 {
		return
	}
	callbacks.mu.Lock()
	delete(callbacks.contexts, c.ptr)
	callbacks.mu.Unlock()
	c.api.unref(c.ptr)
	c.ptr = 0
}

// Frame is a decorated window.
//
// Flutter: no counterpart
type Frame struct {
	ctx     *Context
	ptr     uintptr
	id      uintptr
	handler Handler
}

// Decorate makes the frame for the wl_surface (a pointer). It creates the
// xdg_surface and the xdg_toplevel of the window. Set the title, the app ID and
// the size limits, then Map the frame. It fails with LibdecorDecorateFailed when
// libdecor_decorate returns NULL.
//
// Flutter: no counterpart
func (c *Context) Decorate(surface uintptr, h Handler) (*Frame, error) {
	f := &Frame{ctx: c, handler: h}
	callbacks.mu.Lock()
	callbacks.nextID++
	f.id = callbacks.nextID
	callbacks.frames[f.id] = f
	callbacks.mu.Unlock()
	f.ptr = c.api.decorate(c.ptr, surface, callbacks.frameIfc, f.id)
	if f.ptr == 0 {
		callbacks.mu.Lock()
		delete(callbacks.frames, f.id)
		callbacks.mu.Unlock()
		return nil, errors.New("LibdecorDecorateFailed", "libdecor_decorate failed for the surface")
	}
	return f, nil
}

func (f *Frame) a() *api { return f.ctx.api }

// Free releases the frame; its events stop.
//
// Flutter: no counterpart
func (f *Frame) Free() {
	if f.ptr == 0 {
		return
	}
	callbacks.mu.Lock()
	delete(callbacks.frames, f.id)
	callbacks.mu.Unlock()
	f.a().unrefFr(f.ptr)
	f.ptr = 0
}

// SetTitle sets the title.
//
// Flutter: no counterpart
func (f *Frame) SetTitle(title string) {
	s := dl.CString(title)
	defer dl.Free(s)
	f.a().setTitle(f.ptr, s)
}

// SetAppID sets the application ID.
//
// Flutter: no counterpart
func (f *Frame) SetAppID(id string) {
	s := dl.CString(id)
	defer dl.Free(s)
	f.a().setAppID(f.ptr, s)
}

// SetParent makes the frame a child of another one.
//
// Flutter: no counterpart
func (f *Frame) SetParent(parent *Frame) { f.a().setPar(f.ptr, parent.ptr) }

// SetMinContentSize sets the smallest size of the content; zero is no limit.
//
// Flutter: no counterpart
func (f *Frame) SetMinContentSize(w, h int) { f.a().setMin(f.ptr, int32(w), int32(h)) }

// SetMaxContentSize sets the largest size of the content; zero is no limit.
//
// Flutter: no counterpart
func (f *Frame) SetMaxContentSize(w, h int) { f.a().setMax(f.ptr, int32(w), int32(h)) }

// SetCapabilities says what the user may do with the window.
//
// Flutter: no counterpart
func (f *Frame) SetCapabilities(c Capabilities) { f.a().setCaps(f.ptr, uint32(c)) }

// UnsetCapabilities takes capabilities away.
//
// Flutter: no counterpart
func (f *Frame) UnsetCapabilities(c Capabilities) { f.a().unsetCap(f.ptr, uint32(c)) }

// HasCapability reports whether the user may do all of c.
//
// Flutter: no counterpart
func (f *Frame) HasCapability(c Capabilities) bool { return f.a().hasCap(f.ptr, uint32(c)) }

// Map shows the window: the first configure follows.
//
// Flutter: no counterpart
func (f *Frame) Map() { f.a().mapFrame(f.ptr) }

// SetVisibility shows or hides the decorations.
//
// Flutter: no counterpart
func (f *Frame) SetVisibility(visible bool) { f.a().setVis(f.ptr, visible) }

// Move starts moving the window, for the seat and the serial of the input event.
//
// Flutter: no counterpart
func (f *Frame) Move(seat uintptr, serial uint32) { f.a().move(f.ptr, seat, serial) }

// Resize starts resizing the window from the edge.
//
// Flutter: no counterpart
func (f *Frame) Resize(seat uintptr, serial uint32, edge ResizeEdge) {
	f.a().resize(f.ptr, seat, serial, int32(edge))
}

// ShowWindowMenu opens the window menu at x, y of the surface.
//
// Flutter: no counterpart
func (f *Frame) ShowWindowMenu(seat uintptr, serial uint32, x, y int) {
	f.a().showMenu(f.ptr, seat, serial, int32(x), int32(y))
}

// SetMinimized minimizes the window.
//
// Flutter: no counterpart
func (f *Frame) SetMinimized() { f.a().setMini(f.ptr) }

// SetMaximized maximizes the window.
//
// Flutter: no counterpart
func (f *Frame) SetMaximized() { f.a().setMaxd(f.ptr) }

// UnsetMaximized restores a maximized window.
//
// Flutter: no counterpart
func (f *Frame) UnsetMaximized() { f.a().unsetMax(f.ptr) }

// SetFullscreen makes the window fullscreen on the output (a pointer; zero lets
// the compositor choose).
//
// Flutter: no counterpart
func (f *Frame) SetFullscreen(output uintptr) { f.a().setFull(f.ptr, output) }

// UnsetFullscreen leaves fullscreen.
//
// Flutter: no counterpart
func (f *Frame) UnsetFullscreen() { f.a().unsetFul(f.ptr) }

// IsFloating reports whether the window is neither maximized, fullscreen nor tiled.
//
// Flutter: no counterpart
func (f *Frame) IsFloating() bool { return f.a().isFloat(f.ptr) }

// WindowState is the state of the window.
//
// Flutter: no counterpart
func (f *Frame) WindowState() State { return State(f.a().getState(f.ptr)) }

// Toplevel is the pointer of the xdg_toplevel that libdecor made.
//
// Flutter: no counterpart
func (f *Frame) Toplevel() uintptr { return f.a().getTop(f.ptr) }

// Commit applies the content size and the state: it is the answer to a
// configure (pass its configuration) or a request for a new size (pass the zero
// Configuration).
//
// Flutter: no counterpart
func (f *Frame) Commit(width, height int, c Configuration) {
	state := f.a().stateNew(int32(width), int32(height))
	f.a().commit(f.ptr, state, c.ptr)
	f.a().stateFree(state)
}

// Configuration is a configure event that the window has to answer.
//
// Flutter: no counterpart
type Configuration struct {
	ptr   uintptr
	frame *Frame
}

// ContentSize is the size of the content that the compositor asks for. The
// result is false when the compositor leaves the size to the window.
//
// Flutter: no counterpart
func (c Configuration) ContentSize() (width, height int, ok bool) {
	buf := dl.Calloc(2, 4)
	defer dl.Free(buf)
	if !c.frame.a().confSize(c.ptr, c.frame.ptr, buf, buf+4) {
		return 0, 0, false
	}
	b := dl.Bytes(buf, 8)
	return int(int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)),
		int(int32(uint32(b[4]) | uint32(b[5])<<8 | uint32(b[6])<<16 | uint32(b[7])<<24)), true
}

// WindowState is the state that the configure asks for. The result is false when
// the configure carries none.
//
// Flutter: no counterpart
func (c Configuration) WindowState() (State, bool) {
	buf := dl.Calloc(1, 4)
	defer dl.Free(buf)
	if !c.frame.a().confWinSate(c.ptr, buf) {
		return 0, false
	}
	b := dl.Bytes(buf, 4)
	return State(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24), true
}
