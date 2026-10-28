package foundation

import (
	"sync/atomic"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goid"
)

// UIThread remembers which goroutine is the UI thread: the one that owns the
// element tree, the render tree and the engine (SPEC §13). The binding makes one
// when it is created, on the main goroutine, and hands it to the parts of the
// framework that check or branch on the calling goroutine.
//
// The goroutine is known by its [goid.Token], which [UIThread.IsCurrent]
// compares in about a nanosecond at any stack depth. A token belongs to a
// goroutine only while it lives (the runtime gives the token of an ended
// goroutine to a new one), so the goroutine that is the UI thread must stay alive
// while it is, and a goroutine that takes over calls [UIThread.Adopt] again: the
// binding's goroutine does so by running for the whole life of the app, and a
// test driver that runs on a new goroutine adopts at each of its entry points.
//
// Flutter: no counterpart (Dart code runs on one isolate)
type UIThread struct {
	// cur is atomic: other goroutines read it (SetState checks the thread) while
	// a test driver that runs on a new goroutine adopts the thread. A value is
	// never changed after it is stored.
	cur atomic.Pointer[uiGoroutine]
}

// uiGoroutine is the goroutine that is the UI thread.
type uiGoroutine struct {
	// token is what IsCurrent compares.
	token uint64
	// id is the goroutine's number, for the WrongThread error only. It is read
	// from runtime.Stack, which is slow, so it is kept in debug builds only (the
	// only ones that make that error); it is 0 in the others.
	id uint64
}

// NewUIThread makes a UIThread for the calling goroutine.
//
// Flutter: no counterpart
func NewUIThread() *UIThread {
	t := &UIThread{}
	t.Adopt()
	return t
}

// Adopt makes the calling goroutine the UI thread. A goroutine that takes over
// from one that ended must call it, since a token can be reused once its
// goroutine has ended (see [UIThread]).
//
// Flutter: no counterpart
func (t *UIThread) Adopt() {
	next := &uiGoroutine{token: goid.Token()}
	if DebugMode {
		next.id = goid.ID()
	}
	// Adopt runs at every entry point of a test driver, mostly on the goroutine
	// that is already the UI thread: leave the stored value then.
	if cur := t.cur.Load(); cur != nil && *cur == *next {
		return
	}
	t.cur.Store(next)
}

// IsCurrent is whether the calling goroutine is the UI thread. A nil UIThread
// has no opinion: it says yes. It costs about a nanosecond, so it can run in
// every call to SetState and in the debug checks of the hot paths.
//
// Flutter: no counterpart
func (t *UIThread) IsCurrent() bool {
	if t == nil {
		return true
	}
	cur := t.cur.Load()
	return cur != nil && cur.token == goid.Token()
}

// AssertCurrent panics with the tag WrongThread in debug builds when the calling
// goroutine is not the UI thread. what names the entry point that was called.
// Other builds do nothing. The numbers of the two goroutines in the error's
// metadata are read from runtime.Stack, after the check failed.
//
// Flutter: no counterpart
func (t *UIThread) AssertCurrent(what string) {
	if DebugMode && !t.IsCurrent() {
		var uiID uint64
		if cur := t.cur.Load(); cur != nil {
			uiID = cur.id
		}
		panic(errors.New(tagWrongThread, what+" was called from a goroutine that is not the UI thread; only SetState and Post are safe to call from other goroutines (SPEC §13)",
			"call", what, "uiGoroutine", uiID, "callingGoroutine", goid.ID()))
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWrongThread = "WrongThread"
)
