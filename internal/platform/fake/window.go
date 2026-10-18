package fake

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// Window is a fake platform.Window. It records the calls made on it and keeps
// the last values set, so tests can inspect them.
//
// Flutter: no counterpart
type Window struct {
	p      *Platform
	id     ui.ViewID
	config platform.WindowConfig

	// The fields below are guarded by p.mu.
	title     string
	size      ui.Size
	state     platform.WindowState
	cursor    platform.MouseCursor
	textInput platform.TextInputState
	closed    bool
}

var _ platform.Window = (*Window)(nil)

// ID returns the window's view ID.
//
// Flutter: no counterpart
func (w *Window) ID() ui.ViewID { return w.id }

// NativeSurface returns a handle of kind gpu.SurfaceKindFake whose Window is the
// view ID.
//
// Flutter: no counterpart
func (w *Window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: uintptr(w.id)}
}

// set records a call on the window and updates its state under the lock.
func (w *Window) set(kind EntryKind, value any, update func()) {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	update()
	w.p.log = append(w.p.log, Entry{Kind: kind, View: w.id, Value: value})
}

// SetTitle records the new title.
//
// Flutter: no counterpart
func (w *Window) SetTitle(title string) {
	w.set(EntrySetTitle, title, func() { w.title = title })
}

// SetSize records the new size.
//
// Flutter: no counterpart
func (w *Window) SetSize(size ui.Size) {
	w.set(EntrySetSize, size, func() { w.size = size })
}

// SetState records the new state.
//
// Flutter: no counterpart
func (w *Window) SetState(state platform.WindowState) {
	w.set(EntrySetState, state, func() { w.state = state })
}

// SetCursor records the new cursor.
//
// Flutter: no counterpart
func (w *Window) SetCursor(cursor platform.MouseCursor) {
	w.set(EntrySetCursor, cursor, func() { w.cursor = cursor })
}

// SetTextInput records the new text input state.
//
// Flutter: no counterpart
func (w *Window) SetTextInput(state platform.TextInputState) {
	w.set(EntrySetTextInput, state, func() { w.textInput = state })
}

// RequestFrame records the request.
//
// Flutter: no counterpart
func (w *Window) RequestFrame() {
	w.set(EntryRequestFrame, nil, func() {})
}

// StartDrag records the drag and always succeeds.
//
// Flutter: no counterpart
func (w *Window) StartDrag(data platform.DragData) error {
	w.set(EntryStartDrag, data, func() {})
	return nil
}

// Close marks the window closed and records it. Closing a closed window
// records nothing.
//
// Flutter: no counterpart
func (w *Window) Close() {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	w.p.log = append(w.p.log, Entry{Kind: EntryClose, View: w.id})
}

// Config returns the config the window was created with.
//
// Flutter: no counterpart
func (w *Window) Config() platform.WindowConfig { return w.config }

// Title returns the last title set; the config's title until then.
//
// Flutter: no counterpart
func (w *Window) Title() string {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.title
}

// Size returns the last size set; the config's size until then.
//
// Flutter: no counterpart
func (w *Window) Size() ui.Size {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.size
}

// State returns the last state set; the config's state until then.
//
// Flutter: no counterpart
func (w *Window) State() platform.WindowState {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.state
}

// Cursor returns the last cursor set.
//
// Flutter: no counterpart
func (w *Window) Cursor() platform.MouseCursor {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.cursor
}

// TextInput returns the last text input state set.
//
// Flutter: no counterpart
func (w *Window) TextInput() platform.TextInputState {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.textInput
}

// Closed returns whether Close was called.
//
// Flutter: no counterpart
func (w *Window) Closed() bool {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.closed
}
