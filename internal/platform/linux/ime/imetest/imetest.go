//go:build linux && !android

package imetest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/dbus"
)

// IM is a dbus.InputMethod that records the calls as text such as
// "SetCursorRect(1,2,3,4)" and answers ProcessKey with Handled.
//
// Flutter: no counterpart
type IM struct {
	mu    sync.Mutex
	calls []string
	// Handled decides whether ProcessKey consumes a key; nil consumes none.
	Handled func(dbus.IMEKey) bool
	// Fail makes the calls whose name is a key fail with the error.
	Fail map[string]error
	// Events is the function the bridge gave to open; call it to send the
	// bridge a change of the input method.
	Events func(dbus.IMEEvent)
	// Block, when set, makes ProcessKey wait for it to be closed (or for its
	// context to end).
	Block chan struct{}
}

func (f *IM) rec(format string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	call := fmt.Sprintf(format, args...)
	f.calls = append(f.calls, call)
	name, _, _ := strings.Cut(call, "(")
	return f.Fail[name]
}

// Log returns the calls so far.
//
// Flutter: no counterpart
func (f *IM) Log() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// WaitCalls waits until the input method has had n calls, and returns them.
//
// Flutter: no counterpart
func (f *IM) WaitCalls(t *testing.T, n int) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if calls := f.Log(); len(calls) >= n {
			return calls
		}
		if time.Now().After(deadline) {
			t.Fatalf("calls = %v, want %d", f.Log(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

// Open is an ime.Opener that returns f, and keeps the events function.
//
// Flutter: no counterpart
func (f *IM) Open(_ context.Context, events func(dbus.IMEEvent)) (dbus.InputMethod, error) {
	f.Events = events
	return f, nil
}

// Name is "fake".
func (f *IM) Name() string { return "fake" }

// FocusIn records the call.
func (f *IM) FocusIn(context.Context) error { return f.rec("FocusIn()") }

// FocusOut records the call.
func (f *IM) FocusOut(context.Context) error { return f.rec("FocusOut()") }

// Reset records the call.
func (f *IM) Reset(context.Context) error { return f.rec("Reset()") }

// SetCursorRect records the call.
func (f *IM) SetCursorRect(_ context.Context, x, y, w, h int) error {
	return f.rec("SetCursorRect(%d,%d,%d,%d)", x, y, w, h)
}

// SetContent records the call.
func (f *IM) SetContent(_ context.Context, c dbus.IMEContent) error {
	return f.rec("SetContent(%d,%v)", c.Purpose, c.Hidden)
}

// SetSurroundingText records the call.
func (f *IM) SetSurroundingText(_ context.Context, text string, cursor, anchor int) error {
	return f.rec("SetSurroundingText(%q,%d,%d)", text, cursor, anchor)
}

// ProcessKey records the call as "ProcessKey(<char>,<keycode>,<state>,<release>)".
func (f *IM) ProcessKey(ctx context.Context, k dbus.IMEKey) (bool, error) {
	if f.Block != nil {
		select {
		case <-f.Block:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	if err := f.rec("ProcessKey(%c,%d,%d,%v)", rune(k.Keysym), k.Keycode, k.State, k.Release); err != nil {
		return false, err
	}
	return f.Handled != nil && f.Handled(k), nil
}

// Close records the call.
func (f *IM) Close() error { return f.rec("Close()") }

var _ dbus.InputMethod = (*IM)(nil)

// Poster is an ime.Poster that keeps the posted functions until the test runs
// them on its own goroutine.
//
// Flutter: no counterpart
type Poster struct {
	ch chan func()
}

// NewPoster makes a Poster.
//
// Flutter: no counterpart
func NewPoster() *Poster { return &Poster{ch: make(chan func(), 64)} }

// Post keeps fn.
func (p *Poster) Post(fn func()) { p.ch <- fn }

// Run runs the next posted function, waiting up to two seconds for it.
//
// Flutter: no counterpart
func (p *Poster) Run(t *testing.T) {
	t.Helper()
	select {
	case fn := <-p.ch:
		fn()
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was posted")
	}
}

// Idle fails the test if something is posted within a short time.
//
// Flutter: no counterpart
func (p *Poster) Idle(t *testing.T) {
	t.Helper()
	select {
	case <-p.ch:
		t.Fatal("something was posted")
	case <-time.After(30 * time.Millisecond):
	}
}
