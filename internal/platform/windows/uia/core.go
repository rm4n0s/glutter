package uia

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// FragmentRootID is the ID that names the fragment root of a window (not a node of the semantics tree)
// in a NodeRef.
//
// Flutter: third_party/accessibility/ax/platform/ax_fragment_root_win.cc AXFragmentRootWin
const FragmentRootID = -1

// EventSink is what the core raises UI Automation events through: the COM glue implements it with the
// functions of uiautomationcore, and the tests with a recorder. Its methods are called without the
// core's lock held, because raising an event can make UI Automation call the providers back at once.
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows::DispatchWinAccessibilityEvent
type EventSink interface {
	// Listening says whether a client listens (UiaClientsAreListening). Events are not raised, and
	// providers not made for them, while it is false.
	Listening() bool
	// Structure raises the structure-changed event of the node's children.
	Structure(ref NodeRef, kind int)
	// PropertyChanged raises the property-changed event.
	PropertyChanged(ref NodeRef, id PropertyID, old, now Value)
	// Automation raises a plain event (focus, live region, text).
	Automation(ref NodeRef, id EventID)
	// Disconnect drops the provider of a node that left the tree.
	Disconnect(ref NodeRef)
}

// Window is what the core asks of the windows of the platform.
//
// Flutter: no counterpart
type Window interface {
	// ClientRect is the client area of the view's window on the screen, in physical pixels; ok is
	// false when the window is gone. It is safe from any goroutine.
	ClientRect(view ui.ViewID) (x, y, width, height float64, ok bool)
	// HWND is the window of the view, 0 when it is gone. It is safe from any goroutine.
	HWND(view ui.ViewID) uintptr
	// Post runs fn on the UI thread; it is safe from any goroutine.
	Post(fn func())
}

// Core is the bridge without the COM glue: the model of the trees under a lock, the on/off state, and
// the translation of the changes of the trees into events and of the requests of patterns into the
// framework's actions. Its methods are safe from any goroutine: UI Automation may call the providers on
// the thread of a client, and the framework's calls come on the UI thread. A request only posts the
// action to the UI thread (a provider never touches the framework from another thread, SPEC §25.11).
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows
type Core struct {
	win  Window
	sink EventSink

	enable Enable

	mu    sync.Mutex
	model *Model
	host  platform.AccessibilityHost
}

// NewCore makes the core over the windows of the platform and the sink of the events.
//
// Flutter: no counterpart
func NewCore(win Window, sink EventSink) *Core {
	return &Core{win: win, sink: sink, model: NewModel()}
}

// Start gives the core the host, which it tells when the first client asks for the tree.
//
// Flutter: no counterpart
func (c *Core) Start(host platform.AccessibilityHost) {
	c.mu.Lock()
	c.host = host
	c.mu.Unlock()
	c.enable.Started()
}

// On says whether the bridge is on.
//
// Flutter: no counterpart
func (c *Core) On() bool { return c.enable.On() }

// Ask handles a WM_GETOBJECT (see Enable.Ask). The first one that is answered tells the engine to make
// the semantics tree; it must run on the UI thread, as the window procedure does.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetNativeViewAccessible
func (c *Core) Ask(objectID int32) bool {
	answer, turnedOn := c.enable.Ask(objectID)
	if turnedOn {
		c.mu.Lock()
		host := c.host
		c.mu.Unlock()
		if host != nil {
			host.ScreenReaderChanged(true)
		}
	}
	return answer
}

// Close turns the bridge off: the engine is told to stop making the tree, the providers are dropped and
// the trees forgotten. It may be called twice.
//
// Flutter: no counterpart
func (c *Core) Close() {
	wasOn := c.enable.Close()
	c.mu.Lock()
	host := c.host
	c.host = nil
	var refs []Change
	for _, v := range c.model.Views() {
		refs = append(refs, c.model.RemoveView(v)...)
	}
	c.mu.Unlock()
	for _, ch := range refs {
		if ch.Kind == ChangeNodeRemoved {
			c.sink.Disconnect(ch.Ref)
		}
	}
	if wasOn && host != nil {
		host.ScreenReaderChanged(false)
	}
}

// Read runs fn on the model under the lock. fn must not call into UI Automation or the framework.
//
// Flutter: no counterpart
func (c *Core) Read(fn func(m *Model)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c.model)
}

// ClientRect is the client area of the view's window on the screen, in physical pixels.
//
// Flutter: no counterpart
func (c *Core) ClientRect(view ui.ViewID) (x, y, width, height float64, ok bool) {
	return c.win.ClientRect(view)
}

// HWND is the window of the view, 0 when it is gone.
//
// Flutter: no counterpart
func (c *Core) HWND(view ui.ViewID) uintptr { return c.win.HWND(view) }

// UpdateSemantics takes an update of a view's tree (platform.Accessibility).
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::UpdateSemantics
func (c *Core) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	if !c.enable.On() {
		return
	}
	c.mu.Lock()
	changes := c.model.Apply(view, update)
	c.mu.Unlock()
	c.dispatch(changes)
}

// SetWindow tells the title of the window of a view and whether it is active.
//
// Flutter: no counterpart
func (c *Core) SetWindow(view ui.ViewID, title string, active bool) {
	c.mu.Lock()
	changes := c.model.SetWindow(view, title, active)
	c.mu.Unlock()
	c.dispatch(changes)
}

// RemoveView forgets the tree of a view that was closed.
//
// Flutter: no counterpart
func (c *Core) RemoveView(view ui.ViewID) {
	c.mu.Lock()
	changes := c.model.RemoveView(view)
	c.mu.Unlock()
	c.dispatch(changes)
}

// Perform posts the actions to the UI thread, where the host gives them to the framework.
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows::DispatchAccessibilityAction
func (c *Core) Perform(events []ui.SemanticsActionEvent) {
	if len(events) == 0 {
		return
	}
	c.mu.Lock()
	host := c.host
	c.mu.Unlock()
	if host == nil {
		return
	}
	c.win.Post(func() {
		for _, e := range events {
			host.PerformAction(e)
		}
	})
}

// Request runs a request of a pattern: fn works out the actions on the model under the lock, and they
// are posted to the UI thread. It returns the status.
//
// Flutter: no counterpart
func (c *Core) Request(fn func(m *Model) ([]ui.SemanticsActionEvent, Status)) Status {
	c.mu.Lock()
	events, st := fn(c.model)
	c.mu.Unlock()
	if st == StatusOK {
		c.Perform(events)
	}
	return st
}

// dispatch turns the changes into events. It runs without the lock.
func (c *Core) dispatch(changes []Change) {
	listening := -1 // not asked yet
	hear := func() bool {
		if listening < 0 {
			listening = 0
			if c.sink.Listening() {
				listening = 1
			}
		}
		return listening == 1
	}
	for _, ch := range changes {
		switch ch.Kind {
		case ChangeNodeRemoved:
			c.sink.Disconnect(ch.Ref)
		case ChangeAction:
			c.Perform([]ui.SemanticsActionEvent{ch.Action})
		case ChangeWindowOpened:
			if hear() {
				c.sink.Structure(NodeRef{ch.Ref.View, FragmentRootID}, StructureChildrenInvalidated)
			}
		case ChangeStructure:
			if hear() {
				c.sink.Structure(ch.Ref, ch.Structure)
			}
		case ChangeProperty:
			if hear() {
				c.sink.PropertyChanged(ch.Ref, ch.Property, ch.Old, ch.New)
			}
		case ChangeFocus:
			if hear() {
				c.sink.Automation(ch.Ref, EventAutomationFocusChanged)
			}
		case ChangeLiveRegion:
			if hear() {
				c.sink.Automation(ch.Ref, EventLiveRegionChanged)
			}
		case ChangeTextChanged:
			if hear() {
				c.sink.Automation(ch.Ref, EventTextTextChanged)
			}
		case ChangeTextSelection:
			if hear() {
				c.sink.Automation(ch.Ref, EventTextTextSelectionChanged)
			}
		}
	}
}
