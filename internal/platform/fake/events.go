package fake

import (
	"fmt"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// EntryKind names what an Entry in the log records.
//
// Flutter: no counterpart
type EntryKind string

// The kinds of log entries.
const (
	// EntryNewWindow records NewWindow; Value is the platform.WindowConfig.
	//
	// Flutter: no counterpart
	EntryNewWindow EntryKind = "NewWindow"
	// EntryEvent records an injected event delivered to the handler; Value is
	// the Event.
	//
	// Flutter: no counterpart
	EntryEvent EntryKind = "Event"
	// EntryCloseDecision records the handler's answer to a close request;
	// Value is a bool, true when the window was closed.
	//
	// Flutter: no counterpart
	EntryCloseDecision EntryKind = "CloseDecision"
	// EntrySetTitle records Window.SetTitle; Value is the string.
	//
	// Flutter: no counterpart
	EntrySetTitle EntryKind = "SetTitle"
	// EntrySetSize records Window.SetSize; Value is the ui.Size.
	//
	// Flutter: no counterpart
	EntrySetSize EntryKind = "SetSize"
	// EntrySetState records Window.SetState; Value is the platform.WindowState.
	//
	// Flutter: no counterpart
	EntrySetState EntryKind = "SetState"
	// EntrySetCursor records Window.SetCursor; Value is the
	// platform.MouseCursor.
	//
	// Flutter: no counterpart
	EntrySetCursor EntryKind = "SetCursor"
	// EntrySetTextInput records Window.SetTextInput; Value is the
	// ui.TextInputState.
	//
	// Flutter: no counterpart
	EntrySetTextInput EntryKind = "SetTextInput"
	// EntryRequestFrame records Window.RequestFrame; Value is nil.
	//
	// Flutter: no counterpart
	EntryRequestFrame EntryKind = "RequestFrame"
	// EntryStartDrag records Window.StartDrag; Value is the platform.DragData.
	//
	// Flutter: no counterpart
	EntryStartDrag EntryKind = "StartDrag"
	// EntrySetSizeConstraints records Window.SetSizeConstraints; Value is a
	// SizeConstraints.
	//
	// Flutter: no counterpart
	EntrySetSizeConstraints EntryKind = "SetSizeConstraints"
	// EntryWindowAction records Window.PerformWindowAction; Value is the
	// ui.WindowAction.
	//
	// Flutter: no counterpart
	EntryWindowAction EntryKind = "WindowAction"
	// EntryClose records Window.Close; Value is nil.
	//
	// Flutter: no counterpart
	EntryClose EntryKind = "Close"
)

// Entry is one line of the log of a fake Platform.
//
// Flutter: no counterpart
type Entry struct {
	// Kind says what happened.
	Kind EntryKind
	// View is the view the entry is about, or ui.ImplicitViewID when it is
	// about no particular view.
	View ui.ViewID
	// Value carries the details; its type depends on Kind.
	Value any
}

// Event is an input or system event that Platform.Inject can queue. The set of
// events is closed: it is the methods of platform.Handler.
//
// Flutter: no counterpart
type Event interface {
	// view returns the view the event is about.
	view() ui.ViewID
	// dispatch delivers the event to h.
	dispatch(p *Platform, h platform.Handler)
}

// PointerEvent delivers a batch of pointer reports to Handler.OnPointer.
//
// Flutter: no counterpart
type PointerEvent struct {
	// Data is the batch.
	Data []ui.PointerData
}

func (e PointerEvent) view() ui.ViewID {
	if len(e.Data) > 0 {
		return e.Data[0].ViewID
	}
	return ui.ImplicitViewID
}
func (e PointerEvent) dispatch(_ *Platform, h platform.Handler) { h.OnPointer(e.Data) }

// KeyEvent delivers a key event to Handler.OnKey.
//
// Flutter: no counterpart
type KeyEvent struct {
	// Data is the key event.
	Data ui.KeyData
}

func (e KeyEvent) view() ui.ViewID                          { return ui.ImplicitViewID }
func (e KeyEvent) dispatch(_ *Platform, h platform.Handler) { h.OnKey(e.Data) }

// TextInputEvent delivers an IME event to Handler.OnTextInput.
//
// Flutter: no counterpart
type TextInputEvent struct {
	// Event is the IME event.
	Event ui.TextInputEvent
}

func (e TextInputEvent) view() ui.ViewID                          { return ui.ImplicitViewID }
func (e TextInputEvent) dispatch(_ *Platform, h platform.Handler) { h.OnTextInput(e.Event) }

// FrameEvent delivers a frame request to Handler.OnFrame.
//
// Flutter: no counterpart
type FrameEvent struct {
	// View is the view to draw a frame of.
	View ui.ViewID
	// FrameTime is the frame time since an arbitrary fixed point.
	FrameTime time.Duration
}

func (e FrameEvent) view() ui.ViewID                          { return e.View }
func (e FrameEvent) dispatch(_ *Platform, h platform.Handler) { h.OnFrame(e.View, e.FrameTime) }

// MetricsEvent delivers new view metrics to Handler.OnMetrics.
//
// Flutter: no counterpart
type MetricsEvent struct {
	// View is the view whose metrics changed.
	View ui.ViewID
	// Metrics is the new metrics.
	Metrics platform.ViewMetrics
}

func (e MetricsEvent) view() ui.ViewID                          { return e.View }
func (e MetricsEvent) dispatch(_ *Platform, h platform.Handler) { h.OnMetrics(e.View, e.Metrics) }

// ViewFocusEvent delivers a keyboard focus change to Handler.OnViewFocus.
//
// Flutter: no counterpart
type ViewFocusEvent struct {
	// View is the view whose focus changed.
	View ui.ViewID
	// Focused is whether the view has the focus now.
	Focused bool
}

func (e ViewFocusEvent) view() ui.ViewID                          { return e.View }
func (e ViewFocusEvent) dispatch(_ *Platform, h platform.Handler) { h.OnViewFocus(e.View, e.Focused) }

// LifecycleEvent delivers a lifecycle change to Handler.OnLifecycle.
//
// Flutter: no counterpart
type LifecycleEvent struct {
	// State is the new lifecycle state.
	State ui.AppLifecycleState
}

func (e LifecycleEvent) view() ui.ViewID                          { return ui.ImplicitViewID }
func (e LifecycleEvent) dispatch(_ *Platform, h platform.Handler) { h.OnLifecycle(e.State) }

// DropEvent delivers a drag-and-drop event to Handler.OnDrop.
//
// Flutter: no counterpart
type DropEvent struct {
	// View is the view the drop is over.
	View ui.ViewID
	// Event is the drop event.
	Event platform.DropEvent
}

func (e DropEvent) view() ui.ViewID                          { return e.View }
func (e DropEvent) dispatch(_ *Platform, h platform.Handler) { h.OnDrop(e.View, e.Event) }

// SettingsEvent changes the settings that Platform.Settings reports and then
// delivers them to Handler.OnSettings.
//
// Flutter: no counterpart
type SettingsEvent struct {
	// Settings is the new settings.
	Settings platform.Settings
}

func (e SettingsEvent) view() ui.ViewID { return ui.ImplicitViewID }
func (e SettingsEvent) dispatch(p *Platform, h platform.Handler) {
	p.SetSettings(e.Settings)
	h.OnSettings(e.Settings)
}

// DisplaysEvent changes the displays that Platform.Displays reports and then tells
// Handler.OnDisplaysChanged.
//
// Flutter: no counterpart
type DisplaysEvent struct {
	// Displays is the new list of displays.
	Displays []platform.Display
}

func (e DisplaysEvent) view() ui.ViewID { return ui.ImplicitViewID }
func (e DisplaysEvent) dispatch(p *Platform, h platform.Handler) {
	p.SetDisplays(e.Displays)
	h.OnDisplaysChanged()
}

// CloseRequestEvent delivers a close request to Handler.OnCloseRequest. When
// the handler returns true, the fake closes the view's window.
//
// Flutter: no counterpart
type CloseRequestEvent struct {
	// View is the view the user asked to close.
	View ui.ViewID
}

func (e CloseRequestEvent) view() ui.ViewID { return e.View }
func (e CloseRequestEvent) dispatch(p *Platform, h platform.Handler) {
	closed := h.OnCloseRequest(e.View)
	p.record(Entry{Kind: EntryCloseDecision, View: e.View, Value: closed})
	if !closed {
		return
	}
	for _, w := range p.Windows() {
		if w.ID() == e.View {
			w.Close()
		}
	}
}

// String describes the entry, for test failure messages.
//
// Flutter: no counterpart
func (e Entry) String() string {
	return fmt.Sprintf("%s(view %d): %+v", e.Kind, e.View, e.Value)
}
