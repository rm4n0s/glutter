package ui

import "github.com/rm4n0s/errors"

// DropActions is a set of the actions a drag and drop operation may perform.
//
// Flutter: no counterpart (Flutter has no drag and drop between apps)
type DropActions uint8

// The actions of a drag and drop operation.
const (
	// DropActionCopy copies the data to the target.
	//
	// Flutter: no counterpart
	DropActionCopy DropActions = 1 << iota
	// DropActionMove moves the data to the target: the source removes it.
	//
	// Flutter: no counterpart
	DropActionMove
	// DropActionLink makes a link to the data at the target.
	//
	// Flutter: no counterpart
	DropActionLink

	// DropActionNone is no action: the drop is not accepted.
	//
	// Flutter: no counterpart
	DropActionNone DropActions = 0
)

// Has reports whether the set contains all the actions of a.
//
// Flutter: no counterpart
func (s DropActions) Has(a DropActions) bool { return s&a == a && a != 0 }

// DropPhase is what a DropEvent says happened.
//
// Flutter: no counterpart
type DropPhase int

// The phases of a drag and drop operation over a view.
const (
	// DropPhaseEnter is a drag that came into the view.
	//
	// Flutter: no counterpart
	DropPhaseEnter DropPhase = iota
	// DropPhaseMove is a drag that moved over the view.
	//
	// Flutter: no counterpart
	DropPhaseMove
	// DropPhaseLeave is a drag that left the view without a drop.
	//
	// Flutter: no counterpart
	DropPhaseLeave
	// DropPhaseDrop is a drag that was dropped on the view; the event has the data.
	//
	// Flutter: no counterpart
	DropPhaseDrop
)

// DropEvent is a drag and drop operation of another application (or of this one,
// through the platform) over a view.
//
// Flutter: no counterpart
type DropEvent struct {
	// ViewID is the view the drag is over.
	ViewID ViewID
	// Phase says what happened.
	Phase DropPhase
	// Position is where the pointer is over the view, in physical pixels.
	Position Offset
	// Types are the MIME types that the source offers, such as "text/uri-list".
	Types []string
	// Actions are the actions the source allows.
	Actions DropActions
	// Text is the text of the drag (text/plain), in a drop.
	Text string
	// URIs are the URIs of the drag (text/uri-list), in a drop: the files that a file
	// manager drags are file:// URIs.
	URIs []string
	// LocalID is not 0 when the source is this app: the ID of the DragRequest that
	// started it.
	LocalID uint64
}

// DropHost is what the engine implements to start a drag and drop operation and to
// take part in the ones of other applications.
//
// Flutter: no counterpart
type DropHost interface {
	// StartDrag starts a drag from the window of the view. It must be called while
	// the pointer button that started the drag is down. The result of the operation
	// arrives through PlatformDispatcher.DispatchDragEnd.
	StartDrag(view ViewID, request DragRequest) error
}

// DragRequest is a drag and drop operation that the app starts.
//
// Flutter: no counterpart
type DragRequest struct {
	// ID identifies the request; the DropEvents of the operation that reach this app
	// carry it as LocalID. It is not 0.
	ID uint64
	// Text is the text to offer as text/plain.
	Text string
	// URIs are the URIs to offer as text/uri-list (file:// URIs for files).
	URIs []string
	// Actions are the actions the app allows; none means copy.
	Actions DropActions
}

// DragResult is how a drag and drop operation that the app started ended.
//
// Flutter: no counterpart
type DragResult struct {
	// ID is the ID of the DragRequest.
	ID uint64
	// Action is the action that the target performed; DropActionNone when the drag
	// was cancelled or dropped where it is not accepted.
	Action DropActions
}

// StartDrag starts a drag and drop operation from the window of a view. The result
// comes to OnDragEnd. It fails with DragUnsupported when the host cannot, and with
// the host's error when the platform refuses.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) StartDrag(view ViewID, request DragRequest) error {
	h, ok := d.host.(DropHost)
	if !ok {
		return errors.New(tagDragUnsupported, "the platform cannot start a drag and drop operation")
	}
	return h.StartDrag(view, request)
}

// DispatchDrop passes a drop event to OnDrop and returns the action that the
// framework accepts, or DropActionNone. Called by the engine.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) DispatchDrop(event DropEvent) DropActions {
	if d.OnDrop == nil {
		return DropActionNone
	}
	return d.OnDrop(event)
}

// DispatchDragEnd passes the end of a drag that the app started to OnDragEnd.
// Called by the engine.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) DispatchDragEnd(result DragResult) {
	if d.OnDragEnd != nil {
		d.OnDragEnd(result)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragUnsupported = "DragUnsupported"
)
