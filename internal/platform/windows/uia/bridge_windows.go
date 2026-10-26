//go:build windows

package uia

import (
	"context"
	"io"
	"log/slog"
	"math"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// Bridge is the accessibility bridge of Windows (SPEC §25.11): platform.Accessibility over UI
// Automation. It holds the Core (the model and the on/off state) and the COM objects of the providers
// that clients hold; the first WM_GETOBJECT of a window turns it on (Answer).
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows
type Bridge struct {
	env  *Env
	core *Core
	log  *slog.Logger

	mu   sync.Mutex
	objs map[NodeRef]win32.Object
	// closed says the bridge was closed: no provider is made.
	closed bool
}

var _ platform.Accessibility = (*Bridge)(nil)

// NewBridge makes the bridge over the env and the windows of the platform. log gets what cannot be
// returned to Windows; nil discards it.
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows::AccessibilityBridgeWindows
func NewBridge(env *Env, win Window, log *slog.Logger) *Bridge {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	b := &Bridge{env: env, log: log, objs: map[NodeRef]win32.Object{}}
	b.core = NewCore(win, (*bridgeSink)(b))
	return b
}

// Core is the bridge without the COM glue, for the tests.
//
// Flutter: no counterpart
func (b *Bridge) Core() *Core { return b.core }

// Start gives the bridge the host. It follows no service: the first WM_GETOBJECT is what turns it on.
//
// Flutter: no counterpart
func (b *Bridge) Start(ctx context.Context, host platform.AccessibilityHost) {
	b.core.Start(host)
	context.AfterFunc(ctx, b.Close)
}

// UpdateSemantics gives the bridge the changes to the semantics tree of a view.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::UpdateSemantics
func (b *Bridge) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	b.core.UpdateSemantics(view, update)
}

// SetWindow tells the title of the window of the view and whether it has the focus.
//
// Flutter: no counterpart
func (b *Bridge) SetWindow(view ui.ViewID, title string, active bool) {
	b.core.SetWindow(view, title, active)
}

// RemoveView forgets the tree of a view that was closed, and drops its fragment root.
//
// Flutter: no counterpart
func (b *Bridge) RemoveView(view ui.ViewID) {
	b.core.RemoveView(view)
	(*bridgeSink)(b).Disconnect(NodeRef{view, FragmentRootID})
}

// Close stops the bridge: the engine is told to stop making the tree and the providers are dropped. It
// may be called twice and from any goroutine.
//
// Flutter: no counterpart
func (b *Bridge) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	b.core.Close()
	b.mu.Lock()
	refs := make([]NodeRef, 0, len(b.objs))
	for ref := range b.objs {
		refs = append(refs, ref)
	}
	b.mu.Unlock()
	for _, ref := range refs {
		(*bridgeSink)(b).Disconnect(ref)
	}
}

// Answer handles a WM_GETOBJECT of the window of the view. When the message asks for the UI Automation
// provider (lParam is UiaRootObjectId), the bridge turns on if it is not, and the result is what the
// window procedure returns: UiaReturnRawElementProvider with the fragment root of the window. handled
// is false for any other request, which DefWindowProc answers (the proxy for MSAA clients), and when the
// bridge cannot answer. It runs on the UI thread, in the window procedure.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetNativeViewAccessible
func (b *Bridge) Answer(view ui.ViewID, hwnd, wparam, lparam uintptr) (result uintptr, handled bool) {
	if !b.core.Ask(int32(lparam)) {
		return 0, false
	}
	root, err := b.element(NodeRef{view, FragmentRootID})
	if err != nil {
		b.log.Error("the UI Automation provider of the window could not be made", "err", err)
		return 0, false
	}
	r, err := win32.UiaReturnRawElementProvider(hwnd, wparam, lparam, root)
	if err != nil {
		b.log.Error("UI Automation did not take the provider of the window", "err", err)
		return 0, false
	}
	return r, true
}

// element returns the object of a node or of the fragment root of a view, made on first use. The
// bridge keeps one reference of it; a method that hands it to Windows asks for its own with
// QueryInterface. It fails when the bridge is closed or Windows has no memory.
func (b *Bridge) element(ref NodeRef) (win32.Object, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if o, ok := b.objs[ref]; ok {
		return o, nil
	}
	if b.closed {
		return win32.Object{}, errors.New(win32.TagWindowsUIAFailed, "the UI Automation bridge is closed", "view", int(ref.View), "node", ref.ID)
	}
	types := b.env.elementTypes
	if ref.ID == FragmentRootID {
		types = b.env.rootTypes
	}
	o, err := b.env.host.NewObject(&elem{b: b, ref: ref}, types...)
	if err != nil {
		return win32.Object{}, err
	}
	b.objs[ref] = o
	return o, nil
}

// pointer returns a new reference to the interface of the element, for Windows to keep.
func (b *Bridge) pointer(ref NodeRef, iface Interface) (uintptr, error) {
	o, err := b.element(ref)
	if err != nil {
		return 0, err
	}
	q, err := o.QueryInterface(&iface.IID)
	if err != nil {
		return 0, err
	}
	return q.Ptr(), nil
}

// bridgeSink is the EventSink of the bridge: it raises the events through uiautomationcore.
type bridgeSink Bridge

func (s *bridgeSink) bridge() *Bridge { return (*Bridge)(s) }

// Listening asks UiaClientsAreListening.
func (s *bridgeSink) Listening() bool { return win32.UiaClientsAreListening() }

// provider returns the object of the node to raise an event on, or false when the bridge cannot make it.
func (s *bridgeSink) provider(ref NodeRef) (win32.Object, bool) {
	o, err := s.bridge().element(ref)
	if err != nil {
		s.log.Debug("no provider for the event", "err", err)
		return win32.Object{}, false
	}
	return o, true
}

// raised logs the failure of a raise: a client that is gone is not an error.
func (s *bridgeSink) raised(what string, err error) {
	if err == nil {
		return
	}
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == win32.TagCOMCallFailed {
		if hr, _ := appErr.Metadata["hresult"].(string); hr == hresult(ErrElementNotAvailable).String() {
			return
		}
	}
	s.log.Debug("a UI Automation event was not raised", "event", what, "err", err)
}

// Structure raises the structure-changed event.
func (s *bridgeSink) Structure(ref NodeRef, kind int) {
	if o, ok := s.provider(ref); ok {
		s.raised("structure", win32.UiaRaiseStructureChangedEvent(o, int32(kind), nil))
	}
}

// PropertyChanged raises the property-changed event.
func (s *bridgeSink) PropertyChanged(ref NodeRef, id PropertyID, old, now Value) {
	o, ok := s.provider(ref)
	if !ok {
		return
	}
	var vo, vn win32.Variant
	if fillVariant(&vo, old).Failed() || fillVariant(&vn, now).Failed() {
		win32.VariantClear(&vo)
		win32.VariantClear(&vn)
		return
	}
	s.raised("property", win32.UiaRaiseAutomationPropertyChangedEvent(o, int32(id), &vo, &vn))
	win32.VariantClear(&vo)
	win32.VariantClear(&vn)
}

// Automation raises a plain event.
func (s *bridgeSink) Automation(ref NodeRef, id EventID) {
	if o, ok := s.provider(ref); ok {
		s.raised("automation", win32.UiaRaiseAutomationEvent(o, int32(id)))
	}
}

// Disconnect drops the provider of a node: UI Automation lets go of it, and so does the bridge. A client
// that still holds the object finds it gone (every method answers UIA_E_ELEMENTNOTAVAILABLE).
func (s *bridgeSink) Disconnect(ref NodeRef) {
	b := s.bridge()
	b.mu.Lock()
	o, ok := b.objs[ref]
	delete(b.objs, ref)
	b.mu.Unlock()
	if !ok {
		return
	}
	s.raised("disconnect", win32.UiaDisconnectProvider(o))
	o.Release()
}

// finite says whether a coordinate that a client gave is a number.
func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }
