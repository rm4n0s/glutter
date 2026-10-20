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
	minSize   ui.Size
	maxSize   ui.Size
	state     platform.WindowState
	cursor    platform.MouseCursor
	textInput ui.TextInputState
	icon      ui.WindowIcon
	content   ui.ContentType
	screenOn  bool
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

// SizeConstraints is the value of an EntrySetSizeConstraints entry.
//
// Flutter: no counterpart
type SizeConstraints struct {
	// Min is the smallest size in logical pixels; zero means no limit.
	Min ui.Size
	// Max is the largest size in logical pixels; zero means no limit.
	Max ui.Size
}

// SetSizeConstraints records the new limits.
//
// Flutter: no counterpart
func (w *Window) SetSizeConstraints(min, max ui.Size) {
	w.set(EntrySetSizeConstraints, SizeConstraints{Min: min, Max: max}, func() { w.minSize, w.maxSize = min, max })
}

// SizeConstraints returns the last limits set; the config's until then.
//
// Flutter: no counterpart
func (w *Window) SizeConstraints() SizeConstraints {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return SizeConstraints{Min: w.minSize, Max: w.maxSize}
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
func (w *Window) SetTextInput(state ui.TextInputState) {
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
func (w *Window) StartDrag(data ui.DragRequest) error {
	w.set(EntryStartDrag, data, func() {})
	return nil
}

// PerformWindowAction records the action.
//
// Flutter: no counterpart
func (w *Window) PerformWindowAction(action ui.WindowAction) {
	w.set(EntryWindowAction, action, func() {})
}

// SetIcon records the icon.
//
// Flutter: no counterpart
func (w *Window) SetIcon(icon ui.WindowIcon) error {
	w.set(EntrySetIcon, icon, func() { w.icon = icon })
	return nil
}

// SetContentType records the hint.
//
// Flutter: no counterpart
func (w *Window) SetContentType(contentType ui.ContentType) {
	w.set(EntrySetContentType, contentType, func() { w.content = contentType })
}

// KeepScreenOn records the request and always succeeds.
//
// Flutter: no counterpart
func (w *Window) KeepScreenOn(on bool) error {
	w.set(EntryKeepScreenOn, on, func() { w.screenOn = on })
	return nil
}

// Icon returns the last icon set; the config's icon until then.
//
// Flutter: no counterpart
func (w *Window) Icon() ui.WindowIcon {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.icon
}

// ContentType returns the last content type set; the config's until then.
//
// Flutter: no counterpart
func (w *Window) ContentType() ui.ContentType {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.content
}

// ScreenKeptOn returns whether the last KeepScreenOn was with true.
//
// Flutter: no counterpart
func (w *Window) ScreenKeptOn() bool {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	return w.screenOn
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
func (w *Window) TextInput() ui.TextInputState {
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
