//go:build linux && !android

package ime

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/ui"
)

// callTimeout is how long a call to the input method may take. A key whose
// call takes longer is treated as not consumed, so that a stuck input method
// cannot freeze the keyboard.
const callTimeout = time.Second

// Poster runs functions on the UI thread. linux.Loop is one.
//
// Flutter: no counterpart
type Poster interface {
	// Post queues fn to run on the UI thread. It may be called from any
	// goroutine and does not block.
	Post(fn func())
}

// Opener connects to the input method framework. It is called once, on the
// goroutine of the bridge, when the first text field gets the focus. events
// receives the changes the input method asks for.
//
// Flutter: no counterpart
type Opener func(ctx context.Context, events func(dbus.IMEEvent)) (dbus.InputMethod, error)

// Rect is a rectangle in the pixels the input method expects: of the screen
// on X11, of the surface on Wayland.
//
// Flutter: no counterpart
type Rect struct{ X, Y, W, H int }

// Bridge sends the state of the focused text field and the keys to the input
// method, and delivers the input method's changes to the handler. Every
// method but the ones of the worker belongs to the UI thread.
//
// Flutter: no counterpart
type Bridge struct {
	poster  Poster
	open    Opener
	deliver func(ui.TextInputEvent)

	ctx    context.Context
	cancel context.CancelFunc

	// The fields below belong to the UI thread.
	state       ui.TextInputState
	hasPreedit  bool
	unavailable bool
	closed      bool

	mu      sync.Mutex
	queue   []job
	started bool
	wake    chan struct{}
	done    chan struct{}
}

type jobKind int

const (
	jobState jobKind = iota
	jobKey
	jobReset
)

// job is one request to the worker.
type job struct {
	kind  jobKind
	state ui.TextInputState
	caret Rect
	key   dbus.IMEKey
	done  func(handled bool)
}

// New makes a bridge. open connects to the framework when it is first needed;
// deliver receives the changes of the input method on the UI thread.
//
// Flutter: no counterpart
func New(poster Poster, open Opener, deliver func(ui.TextInputEvent)) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bridge{
		poster: poster, open: open, deliver: deliver, ctx: ctx, cancel: cancel,
		wake: make(chan struct{}, 1), done: make(chan struct{}),
	}
}

// Active reports whether a text field has the focus and the input method may
// be there to see the keys. While it is, every key goes through ProcessKey, so
// that the keys keep their order.
//
// Flutter: no counterpart
func (b *Bridge) Active() bool { return b.state.Enabled && !b.unavailable && !b.closed }

// SetState tells the input method about the focused text field: the field
// gets or loses the focus, and the text, the selection and the caret
// rectangle follow. caret is the rectangle of the caret in the pixels the
// input method expects.
//
// Flutter: no counterpart
func (b *Bridge) SetState(st ui.TextInputState, caret Rect) {
	if b.closed {
		return
	}
	// A change that does not come from the input method ends the text being
	// composed there, as clicking in the field does in other toolkits.
	if st.Enabled && b.state.Enabled && b.hasPreedit && !st.CausedByInputMethod &&
		(st.Text != b.state.Text || st.SelectionAnchor != b.state.SelectionAnchor || st.SelectionCursor != b.state.SelectionCursor) {
		b.hasPreedit = false
		b.enqueue(job{kind: jobReset})
	}
	if !st.Enabled {
		b.hasPreedit = false
	}
	if !st.Enabled && !b.state.Enabled {
		b.state = st
		return
	}
	b.state = st
	b.enqueue(job{kind: jobState, state: st, caret: caret})
}

// ProcessKey asks the input method whether it consumes the key. done runs on
// the UI thread, in the order of the calls, with true when the input method
// consumed it. It runs with false at once on the UI thread's next turn if there
// is no input method.
//
// Flutter: no counterpart
func (b *Bridge) ProcessKey(key dbus.IMEKey, done func(handled bool)) {
	if b.closed || b.unavailable {
		b.poster.Post(func() { done(false) })
		return
	}
	b.enqueue(job{kind: jobKey, key: key, done: done})
}

// Close destroys the input context and stops the worker. Pending keys are
// answered as not consumed.
//
// Flutter: no counterpart
func (b *Bridge) Close() {
	if b.closed {
		return
	}
	b.closed = true
	b.cancel()
	b.mu.Lock()
	started := b.started
	b.mu.Unlock()
	if started {
		b.signal()
		<-b.done
	}
}

func (b *Bridge) enqueue(j job) {
	b.mu.Lock()
	b.queue = append(b.queue, j)
	if !b.started {
		b.started = true
		go b.work()
	}
	b.mu.Unlock()
	b.signal()
}

func (b *Bridge) signal() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

// next returns the next job, waiting for one; ok is false once the bridge is
// closed.
func (b *Bridge) next() (j job, ok bool) {
	for {
		if b.ctx.Err() != nil {
			return job{}, false
		}
		b.mu.Lock()
		if len(b.queue) > 0 {
			j = b.queue[0]
			b.queue = b.queue[1:]
			b.mu.Unlock()
			return j, true
		}
		b.mu.Unlock()
		select {
		case <-b.wake:
		case <-b.ctx.Done():
		}
	}
}

// work is the goroutine that talks to the input method.
func (b *Bridge) work() {
	defer close(b.done)
	var w worker
	defer w.close(b)
	for {
		j, ok := b.next()
		if !ok {
			return
		}
		if !w.connected && !w.failed {
			w.connect(b)
		}
		if w.failed {
			// Keys wait for an answer: tell them there is no input method.
			if j.kind == jobKey {
				done := j.done
				b.poster.Post(func() { done(false) })
			}
			continue
		}
		w.run(b, j)
	}
}

// worker is the state of the goroutine of the bridge.
type worker struct {
	im        dbus.InputMethod
	connected bool
	failed    bool
	focused   bool
	applied   ui.TextInputState
	caret     Rect
	content   dbus.IMEContent
	haveCaret bool
	haveText  bool
}

func (w *worker) connect(b *Bridge) {
	im, err := b.open(b.ctx, func(ev dbus.IMEEvent) {
		b.poster.Post(func() { b.onEvent(ev) })
	})
	if err != nil {
		if b.ctx.Err() == nil {
			slog.Info("no input method over D-Bus: the keys go to the app as they are", "err", err)
		}
		w.failed = true
		b.poster.Post(func() { b.unavailable = true })
		return
	}
	w.im, w.connected = im, true
	slog.Debug("input method connected", "framework", im.Name())
}

func (w *worker) close(b *Bridge) {
	if w.im != nil {
		if err := w.im.Close(); err != nil {
			slog.Warn("closing the input method failed", "err", err)
		}
	}
	// The jobs that are left get the answer "not consumed".
	b.mu.Lock()
	left := b.queue
	b.queue = nil
	b.mu.Unlock()
	for _, j := range left {
		if j.kind == jobKey {
			done := j.done
			b.poster.Post(func() { done(false) })
		}
	}
}

func (w *worker) warn(what string, err error) {
	if err != nil {
		slog.Warn("the input method call failed", "call", what, "err", err)
	}
}

func (w *worker) run(b *Bridge, j job) {
	ctx, cancel := context.WithTimeout(b.ctx, callTimeout)
	defer cancel()
	switch j.kind {
	case jobKey:
		handled, err := w.im.ProcessKey(ctx, j.key)
		if err != nil {
			w.warn("ProcessKey", err)
			handled = false
		}
		done := j.done
		b.poster.Post(func() { done(handled) })
	case jobReset:
		w.warn("Reset", w.im.Reset(ctx))
	case jobState:
		w.apply(ctx, j.state, j.caret)
	}
}

// apply makes the input method's view of the field the state, sending only
// what changed.
func (w *worker) apply(ctx context.Context, st ui.TextInputState, caret Rect) {
	if !st.Enabled {
		if w.focused {
			w.focused = false
			w.warn("FocusOut", w.im.FocusOut(ctx))
		}
		return
	}
	content := dbus.IMEContent{
		Purpose: uint32(st.Purpose),
		Hidden:  st.Hints.Has(ui.TextInputHintHiddenText),
	}
	// Purposes past the ones that GTK knows (date and time) are free-form.
	if content.Purpose > uint32(ui.TextInputPurposePIN) {
		content.Purpose = 0
	}
	if !w.focused || content != w.content {
		w.content = content
		w.warn("SetContent", w.im.SetContent(ctx, content))
	}
	if !w.haveText || st.Text != w.applied.Text || st.SelectionAnchor != w.applied.SelectionAnchor ||
		st.SelectionCursor != w.applied.SelectionCursor {
		w.haveText = true
		cursor, anchor := runeIndex(st.Text, st.SelectionCursor), runeIndex(st.Text, st.SelectionAnchor)
		w.warn("SetSurroundingText", w.im.SetSurroundingText(ctx, st.Text, cursor, anchor))
	}
	w.applied = st
	if !w.haveCaret || caret != w.caret {
		w.haveCaret, w.caret = true, caret
		w.warn("SetCursorRect", w.im.SetCursorRect(ctx, caret.X, caret.Y, caret.W, caret.H))
	}
	if !w.focused {
		w.focused = true
		w.warn("FocusIn", w.im.FocusIn(ctx))
	}
}

// onEvent turns a change of the input method into a ui.TextInputEvent. It runs
// on the UI thread.
func (b *Bridge) onEvent(ev dbus.IMEEvent) {
	if b.closed || !b.state.Enabled {
		return
	}
	out := ui.TextInputEvent{View: b.state.View, PreeditCursorBegin: -1, PreeditCursorEnd: -1}
	switch ev.Kind {
	case dbus.IMEEventCommit:
		out.Commit = ev.Text
		// The text being composed is gone once it is committed.
		b.hasPreedit = false
	case dbus.IMEEventPreedit:
		out.Preedit = ev.Text
		if ev.Text != "" && ev.Cursor >= 0 {
			out.PreeditCursorBegin, out.PreeditCursorEnd = ev.Cursor, ev.Cursor
		}
		b.hasPreedit = ev.Text != ""
	case dbus.IMEEventDelete:
		before, after, ok := b.deleteRange(ev.Offset, ev.Count)
		if !ok {
			slog.Debug("the input method asked to delete text that is not around the cursor",
				"offset", ev.Offset, "count", ev.Count)
			return
		}
		out.DeleteBefore, out.DeleteAfter = before, after
		// The delete comes with the text being composed gone.
		out.Preedit = ""
	default:
		return
	}
	b.deliver(out)
}

// deleteRange converts a range of characters, relative to the cursor, into the
// bytes before and after the selection that the field is asked to delete. The
// range is cut to the text that the field has; ok is false when no part of it
// is around the selection.
func (b *Bridge) deleteRange(offset, count int) (before, after int, ok bool) {
	if count <= 0 {
		return 0, 0, false
	}
	text := b.state.Text
	start, end := b.state.SelectionAnchor, b.state.SelectionCursor
	if start > end {
		start, end = end, start
	}
	if start < 0 || end > len(text) {
		return 0, 0, false
	}
	// Characters of the range before the cursor, and after it.
	charsBefore, charsAfter := 0, 0
	switch {
	case offset+count <= 0:
		if offset+count < 0 {
			return 0, 0, false // a gap between the range and the cursor
		}
		charsBefore = count
	case offset >= 0:
		if offset > 0 {
			return 0, 0, false
		}
		charsAfter = count
	default:
		charsBefore, charsAfter = -offset, offset+count
	}
	before = bytesBefore(text[:start], charsBefore)
	after = bytesAfter(text[end:], charsAfter)
	return before, after, before+after > 0
}

// bytesBefore returns the bytes of the last n characters of s.
func bytesBefore(s string, n int) int {
	size := 0
	for ; n > 0 && size < len(s); n-- {
		_, w := utf8.DecodeLastRuneInString(s[:len(s)-size])
		size += w
	}
	return size
}

// bytesAfter returns the bytes of the first n characters of s.
func bytesAfter(s string, n int) int {
	size := 0
	for ; n > 0 && size < len(s); n-- {
		_, w := utf8.DecodeRuneInString(s[size:])
		size += w
	}
	return size
}

// runeIndex returns the number of characters before the byte offset off.
func runeIndex(text string, off int) int {
	if off < 0 {
		return 0
	}
	if off > len(text) {
		off = len(text)
	}
	return utf8.RuneCountInString(text[:off])
}
