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
// Flutter: no counterpart (Dart code runs on one isolate)
type UIThread struct {
	// id is atomic: other goroutines read it (SetState checks the thread) while a
	// test driver that runs on a new goroutine adopts the thread.
	id atomic.Uint64
}

// NewUIThread makes a UIThread for the calling goroutine.
//
// Flutter: no counterpart
func NewUIThread() *UIThread {
	t := &UIThread{}
	t.id.Store(goid.ID())
	return t
}

// Adopt makes the calling goroutine the UI thread.
//
// Flutter: no counterpart
func (t *UIThread) Adopt() { t.id.Store(goid.ID()) }

// IsCurrent is whether the calling goroutine is the UI thread. A nil UIThread
// has no opinion: it says yes.
//
// Flutter: no counterpart
func (t *UIThread) IsCurrent() bool { return t == nil || t.id.Load() == goid.ID() }

// AssertCurrent panics with the tag WrongThread in debug builds when the calling
// goroutine is not the UI thread. what names the entry point that was called.
// Other builds do nothing.
//
// Flutter: no counterpart
func (t *UIThread) AssertCurrent(what string) {
	if DebugMode && !t.IsCurrent() {
		panic(errors.New("WrongThread", what+" was called from a goroutine that is not the UI thread; only SetState and Post are safe to call from other goroutines (SPEC §13)",
			"call", what, "uiGoroutine", t.id.Load(), "callingGoroutine", goid.ID()))
	}
}
