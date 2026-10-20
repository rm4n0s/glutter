package ui

import "github.com/rm4n0s/errors"

// WindowKind is the kind of a window that the framework creates.
//
// Flutter: widgets/_window.dart WindowArchetype
type WindowKind int

// The values of WindowKind.
const (
	// WindowKindRegular is an ordinary top-level window.
	//
	// Flutter: widgets/_window.dart WindowArchetype.regular
	WindowKindRegular WindowKind = iota
	// WindowKindDialog is a dialog, modal to its parent when it has one.
	//
	// Flutter: widgets/_window.dart WindowArchetype.dialog
	WindowKindDialog
)

// WindowSpec describes a window that the framework asks the engine to create.
//
// Flutter: widgets/_window.dart WindowingOwner.createRegularWindowController
type WindowSpec struct {
	// Kind is the kind of window.
	Kind WindowKind
	// Title is the title of the window.
	Title string
	// Size is the initial size of the content in logical pixels; zero lets the
	// platform choose.
	Size Size
	// MinSize is the smallest size in logical pixels; zero means no minimum.
	MinSize Size
	// MaxSize is the largest size in logical pixels; zero means no maximum.
	MaxSize Size
	// NotResizable is true for a window the user cannot resize.
	NotResizable bool
	// Parent is the view of the parent of a dialog, when HasParent is true.
	Parent ViewID
	// HasParent is whether the window has a parent. A dialog with a parent is modal
	// to it and closes with it.
	HasParent bool
	// AppDecorations is true when the app draws the title bar of the window, so the
	// platform does not ask the compositor or libdecor for one.
	AppDecorations bool
	// Icon is the icon of the window; the zero value is none.
	Icon WindowIcon
	// ContentType is the hint about what the window shows.
	ContentType ContentType
}

// WindowControlHost is what the engine implements to create, change and destroy
// the windows that the framework asks for (widgets/_window.dart). A host without
// it makes CreateWindow fail with WindowingUnsupported.
//
// Flutter: widgets/_window.dart WindowingOwner
type WindowControlHost interface {
	// CreateWindow creates a window and its view. It returns the ID of the view and
	// its configuration until the platform reports the real one.
	CreateWindow(spec WindowSpec) (ViewID, ViewConfiguration, error)
	// DestroyWindow destroys the window of the view. It is not an error to destroy
	// a window twice.
	DestroyWindow(view ViewID)
	// SetWindowTitle sets the title of the window of the view.
	SetWindowTitle(view ViewID, title string)
	// SetWindowSize asks for a new content size in logical pixels.
	SetWindowSize(view ViewID, size Size)
	// SetWindowConstraints sets the smallest and the largest content size in
	// logical pixels; zero means no limit.
	SetWindowConstraints(view ViewID, min, max Size)
}

// CreateWindow asks the engine for a new window and returns its view. The view's
// configuration is the one the host starts with; the platform reports the real
// one soon after. It fails with WindowingUnsupported when the host cannot create
// windows, and with the host's error otherwise.
//
// Flutter: widgets/_window.dart WindowingOwner.createRegularWindowController
func (d *PlatformDispatcher) CreateWindow(spec WindowSpec) (*View, error) {
	h, ok := d.host.(WindowControlHost)
	if !ok {
		return nil, errors.New("WindowingUnsupported", "the platform cannot create windows")
	}
	id, cfg, err := h.CreateWindow(spec)
	if err != nil {
		return nil, err
	}
	return d.AddView(id, cfg), nil
}

// DestroyWindow destroys the window of the view and removes the view. Rendering
// into a view that was destroyed does nothing.
//
// Flutter: widgets/_window.dart BaseWindowController.destroy
func (d *PlatformDispatcher) DestroyWindow(id ViewID) {
	if _, ok := d.views[id]; !ok {
		return
	}
	if h, ok := d.host.(WindowControlHost); ok {
		h.DestroyWindow(id)
	}
	d.RemoveView(id)
}

// SetWindowTitle sets the title of the window of the view.
//
// Flutter: widgets/_window.dart RegularWindowController.setTitle
func (d *PlatformDispatcher) SetWindowTitle(id ViewID, title string) {
	if h, ok := d.host.(WindowControlHost); ok {
		h.SetWindowTitle(id, title)
	}
}

// SetWindowSize asks for a new content size of the window of the view, in logical
// pixels.
//
// Flutter: widgets/_window.dart RegularWindowController.setSize
func (d *PlatformDispatcher) SetWindowSize(id ViewID, size Size) {
	if h, ok := d.host.(WindowControlHost); ok {
		h.SetWindowSize(id, size)
	}
}

// SetWindowConstraints sets the limits on the content size of the window of the
// view, in logical pixels; zero means no limit.
//
// Flutter: widgets/_window.dart RegularWindowController.setConstraints
func (d *PlatformDispatcher) SetWindowConstraints(id ViewID, min, max Size) {
	if h, ok := d.host.(WindowControlHost); ok {
		h.SetWindowConstraints(id, min, max)
	}
}

// ViewFocusState is whether a view has the focus.
//
// Flutter: ui/platform_dispatcher.dart ViewFocusState
type ViewFocusState int

// The values of ViewFocusState.
const (
	// ViewFocusStateUnfocused is a view that lost the focus.
	//
	// Flutter: ui/platform_dispatcher.dart ViewFocusState.unfocused
	ViewFocusStateUnfocused ViewFocusState = iota
	// ViewFocusStateFocused is a view that has the focus.
	//
	// Flutter: ui/platform_dispatcher.dart ViewFocusState.focused
	ViewFocusStateFocused
)

// ViewFocusDirection is the direction of a focus change between views.
//
// Flutter: ui/platform_dispatcher.dart ViewFocusDirection
type ViewFocusDirection int

// The values of ViewFocusDirection.
const (
	// ViewFocusDirectionUndefined is a focus change with no direction, as when the
	// user clicks the window.
	//
	// Flutter: ui/platform_dispatcher.dart ViewFocusDirection.undefined
	ViewFocusDirectionUndefined ViewFocusDirection = iota
	// ViewFocusDirectionForward is a focus change that moves to the next widget.
	//
	// Flutter: ui/platform_dispatcher.dart ViewFocusDirection.forward
	ViewFocusDirectionForward
	// ViewFocusDirectionBackward is a focus change that moves to the previous
	// widget.
	//
	// Flutter: ui/platform_dispatcher.dart ViewFocusDirection.backward
	ViewFocusDirectionBackward
)

// ViewFocusEvent says that a view got or lost the focus.
//
// Flutter: ui/platform_dispatcher.dart ViewFocusEvent
type ViewFocusEvent struct {
	// ViewID is the view whose focus changed.
	ViewID ViewID
	// State is whether the view has the focus now.
	State ViewFocusState
	// Direction is where the focus came from.
	Direction ViewFocusDirection
}

// ViewFocusHost is what the engine implements to give the focus to a native
// window when the framework moves the focus into its view.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.requestViewFocusChange
type ViewFocusHost interface {
	// RequestViewFocus asks for the window of the view to get or lose the focus.
	RequestViewFocus(view ViewID, state ViewFocusState, direction ViewFocusDirection)
}

// DispatchViewFocus tells OnViewFocusChange that a view got or lost the focus.
// Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._dispatchViewFocusEvent
func (d *PlatformDispatcher) DispatchViewFocus(event ViewFocusEvent) {
	if d.OnViewFocusChange != nil {
		d.OnViewFocusChange(event)
	}
}

// RequestViewFocusChange asks the engine to give the focus to the native window of
// a view, or to take it away. The engine answers with a view focus event once
// the window has it.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.requestViewFocusChange
func (d *PlatformDispatcher) RequestViewFocusChange(id ViewID, state ViewFocusState, direction ViewFocusDirection) {
	if h, ok := d.host.(ViewFocusHost); ok {
		h.RequestViewFocus(id, state, direction)
	}
}
