package fake

import (
	"context"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// WindowRecord is what the engine told the accessibility bridge about a window.
//
// Flutter: no counterpart
type WindowRecord struct {
	View   ui.ViewID
	Title  string
	Active bool
}

// Accessibility is the accessibility bridge of the fake platform: it keeps what the
// engine gave it, and the test plays the screen reader with ScreenReader and Perform.
//
// Flutter: no counterpart
type Accessibility struct {
	p *Platform

	mu      sync.Mutex
	host    platform.AccessibilityHost
	started bool
	closed  bool
	updates []ViewUpdate
	windows []WindowRecord
	removed []ui.ViewID
}

// ViewUpdate is a semantics update that the engine gave the bridge.
//
// Flutter: no counterpart
type ViewUpdate struct {
	View   ui.ViewID
	Update *ui.SemanticsUpdate
}

var _ platform.Accessibility = (*Accessibility)(nil)

// Start keeps the host.
func (a *Accessibility) Start(_ context.Context, host platform.AccessibilityHost) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.host, a.started = host, true
}

// UpdateSemantics records the update.
func (a *Accessibility) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.updates = append(a.updates, ViewUpdate{view, update})
}

// SetWindow records the window.
func (a *Accessibility) SetWindow(view ui.ViewID, title string, active bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.windows = append(a.windows, WindowRecord{view, title, active})
}

// RemoveView records the view that was removed.
func (a *Accessibility) RemoveView(view ui.ViewID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.removed = append(a.removed, view)
}

// Close records that the engine closed the bridge.
func (a *Accessibility) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
}

// Started says whether the engine started the bridge.
func (a *Accessibility) Started() bool { a.mu.Lock(); defer a.mu.Unlock(); return a.started }

// Closed says whether the engine closed the bridge.
func (a *Accessibility) Closed() bool { a.mu.Lock(); defer a.mu.Unlock(); return a.closed }

// Updates are the semantics updates the bridge got, oldest first.
func (a *Accessibility) Updates() []ViewUpdate {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]ViewUpdate(nil), a.updates...)
}

// Windows are the windows that the bridge was told about, oldest first.
func (a *Accessibility) Windows() []WindowRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]WindowRecord(nil), a.windows...)
}

// Removed are the views that the bridge was told to forget.
func (a *Accessibility) Removed() []ui.ViewID {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]ui.ViewID(nil), a.removed...)
}

// ScreenReader plays a screen reader that starts or stops: the host is told on the
// UI thread. It does nothing before Start.
func (a *Accessibility) ScreenReader(on bool) {
	a.mu.Lock()
	host := a.host
	a.mu.Unlock()
	if host != nil {
		a.p.Post(func() { host.ScreenReaderChanged(on) })
	}
}

// Perform plays an assistive technology that asks for an action on a node.
func (a *Accessibility) Perform(event ui.SemanticsActionEvent) {
	a.mu.Lock()
	host := a.host
	a.mu.Unlock()
	if host != nil {
		a.p.Post(func() { host.PerformAction(event) })
	}
}

// services is the fake platform's Services.
type services struct{ a *Accessibility }

func (s services) Accessibility() platform.Accessibility { return s.a }

// Accessibility is the fake accessibility bridge, which Services gives the engine.
//
// Flutter: no counterpart
func (p *Platform) Accessibility() *Accessibility {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.acc == nil {
		p.acc = &Accessibility{p: p}
	}
	return p.acc
}
