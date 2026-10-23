package services

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
)

// MouseCursorSession is a session of a mouse cursor on one device: from the time
// the cursor becomes the active one until another replaces it.
//
// Flutter: services/mouse_cursor.dart MouseCursorSession
type MouseCursorSession interface {
	// Cursor is the cursor of the session.
	Cursor() MouseCursor
	// Device is the device the session controls.
	Device() int
	// Activate makes the cursor the one of the device.
	Activate()
	// Dispose is called when another cursor replaces this one.
	Dispose()
}

// MouseCursor is the appearance of the mouse cursor over a region. A value must
// be comparable: the manager skips an update whose cursor equals the active one.
//
// Flutter: services/mouse_cursor.dart MouseCursor
type MouseCursor interface {
	// CreateSession creates the session that makes the cursor active on a
	// device.
	//
	// Flutter: services/mouse_cursor.dart MouseCursor.createSession
	CreateSession(device int) MouseCursorSession
	// DebugDescription is a short description of the cursor.
	//
	// Flutter: services/mouse_cursor.dart MouseCursor.debugDescription
	DebugDescription() string
}

type deferringMouseCursor struct{}

func (deferringMouseCursor) CreateSession(int) MouseCursorSession {
	panic(errors.New(tagMouseCursorDeferCreatedSession, "the defer cursor can not create a session"))
}
func (deferringMouseCursor) DebugDescription() string { return "defer" }

type noopMouseCursor struct{}

func (c noopMouseCursor) CreateSession(device int) MouseCursorSession {
	return noopMouseCursorSession{device: device}
}
func (noopMouseCursor) DebugDescription() string { return "uncontrolled" }

type noopMouseCursorSession struct{ device int }

func (noopMouseCursorSession) Cursor() MouseCursor { return noopMouseCursor{} }
func (s noopMouseCursorSession) Device() int       { return s.device }
func (noopMouseCursorSession) Activate()           {}
func (noopMouseCursorSession) Dispose()            {}

// MouseCursorDefer defers the choice of the cursor to the next region under the
// pointer; the fallback cursor is used if none chooses.
//
// Flutter: services/mouse_cursor.dart MouseCursor.defer
var MouseCursorDefer MouseCursor = deferringMouseCursor{}

// MouseCursorUncontrolled leaves the cursor to the platform: no cursor is set.
//
// Flutter: services/mouse_cursor.dart MouseCursor.uncontrolled
var MouseCursorUncontrolled MouseCursor = noopMouseCursor{}

// MouseCursorFirstNonDeferred returns the first cursor that is not
// MouseCursorDefer, and false if there is none.
//
// Flutter: services/mouse_cursor.dart _DeferringMouseCursor.firstNonDeferred
func MouseCursorFirstNonDeferred(cursors []MouseCursor) (MouseCursor, bool) {
	for _, c := range cursors {
		if c != MouseCursorDefer {
			return c, true
		}
	}
	return nil, false
}

// MouseCursorManager tracks the cursor of each device and activates a new one
// when it changes.
//
// Flutter: services/mouse_cursor.dart MouseCursorManager
type MouseCursorManager struct {
	// FallbackMouseCursor is the cursor used when all the candidates defer.
	FallbackMouseCursor MouseCursor
	// ActivateSystemCursor shows a system cursor on a device; the sessions of the
	// system cursors call it. The binding sets it to the platform's. Without it,
	// activating a system cursor does nothing.
	//
	// Deviation: Flutter's sessions call SystemChannels.mouseCursor, a platform
	// channel (SPEC §12.10).
	//
	// Flutter: services/mouse_cursor.dart _SystemMouseCursorSession.activate
	ActivateSystemCursor SystemCursorActivator
	lastSession          map[int]MouseCursorSession
}

// SystemCursorActivator shows the system cursor of the kind (the kind of a
// SystemMouseCursor, such as "click") on the device.
//
// Flutter: services/mouse_cursor.dart _SystemMouseCursorSession.activate
type SystemCursorActivator func(device int, kind string)

// systemCursorSession is a session that needs the activator of the manager.
type systemCursorSession interface {
	bindActivator(SystemCursorActivator)
}

// NewMouseCursorManager creates a manager. The fallback must not be
// MouseCursorDefer.
//
// Flutter: services/mouse_cursor.dart MouseCursorManager.new
func NewMouseCursorManager(fallback MouseCursor) *MouseCursorManager {
	if foundation.DebugMode && fallback == MouseCursorDefer {
		panic(errors.New(tagMouseCursorFallbackDefers, "the fallback mouse cursor can not defer"))
	}
	return &MouseCursorManager{FallbackMouseCursor: fallback, lastSession: map[int]MouseCursorSession{}}
}

// DebugDeviceActiveCursor is the cursor of the last session of a device; it is
// only available in debug mode.
//
// Flutter: services/mouse_cursor.dart MouseCursorManager.debugDeviceActiveCursor
func (m *MouseCursorManager) DebugDeviceActiveCursor(device int) (MouseCursor, bool) {
	if !foundation.DebugMode {
		return nil, false
	}
	s, ok := m.lastSession[device]
	if !ok {
		return nil, false
	}
	return s.Cursor(), true
}

// HandleDeviceCursorUpdate activates the first cursor of the candidates that
// does not defer, if it differs from the active one of the device. A removed
// event forgets the device.
//
// Flutter: services/mouse_cursor.dart MouseCursorManager.handleDeviceCursorUpdate
func (m *MouseCursorManager) HandleDeviceCursorUpdate(device int, triggeringEvent gestures.PointerEvent, cursorCandidates []MouseCursor) {
	if _, ok := triggeringEvent.(*gestures.PointerRemovedEvent); ok {
		delete(m.lastSession, device)
		return
	}
	last := m.lastSession[device]
	next, ok := MouseCursorFirstNonDeferred(cursorCandidates)
	if !ok {
		next = m.FallbackMouseCursor
	}
	if last != nil && last.Cursor() == next {
		return
	}
	nextSession := next.CreateSession(device)
	if bound, ok := nextSession.(systemCursorSession); ok {
		bound.bindActivator(m.ActivateSystemCursor)
	}
	m.lastSession[device] = nextSession
	if last != nil {
		last.Dispose()
	}
	nextSession.Activate()
}

// SystemMouseCursor is a cursor that the system draws, one of the SystemMouseCursors.
// Two system cursors are equal when their kinds are.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursor
type SystemMouseCursor struct{ kind string }

// Kind is the name of the cursor that the platform shows, such as "click".
//
// Flutter: services/mouse_cursor.dart SystemMouseCursor.kind
func (c SystemMouseCursor) Kind() string { return c.kind }

// DebugDescription describes the cursor with its kind.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursor.debugDescription
func (c SystemMouseCursor) DebugDescription() string { return "SystemMouseCursor(" + c.kind + ")" }

// String is the description.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursor.toString
func (c SystemMouseCursor) String() string { return c.DebugDescription() }

// CreateSession creates the session that shows the cursor on the device.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursor.createSession
func (c SystemMouseCursor) CreateSession(device int) MouseCursorSession {
	return &systemMouseCursorSession{cursor: c, device: device}
}

type systemMouseCursorSession struct {
	cursor   SystemMouseCursor
	device   int
	activate SystemCursorActivator
}

func (s *systemMouseCursorSession) Cursor() MouseCursor                   { return s.cursor }
func (s *systemMouseCursorSession) Device() int                           { return s.device }
func (s *systemMouseCursorSession) Dispose()                              {}
func (s *systemMouseCursorSession) bindActivator(a SystemCursorActivator) { s.activate = a }

// Activate shows the cursor on the device.
//
// Flutter: services/mouse_cursor.dart _SystemMouseCursorSession.activate
func (s *systemMouseCursorSession) Activate() {
	if s.activate != nil {
		s.activate(s.device, s.cursor.kind)
	}
}

type systemMouseCursors struct {
	// None hides the cursor.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.none
	None SystemMouseCursor
	// Basic is the platform's default arrow.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.basic
	Basic SystemMouseCursor
	// Click is the pointing hand, for something that can be clicked.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.click
	Click SystemMouseCursor
	// Forbidden says the action is not allowed.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.forbidden
	Forbidden SystemMouseCursor
	// Wait says the app is busy and cannot be used.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.wait
	Wait SystemMouseCursor
	// Progress says the app is busy but can still be used.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.progress
	Progress SystemMouseCursor
	// ContextMenu says a context menu is available.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.contextMenu
	ContextMenu SystemMouseCursor
	// Help says help is available.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.help
	Help SystemMouseCursor
	// Text is the I-beam, for text that can be selected.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.text
	Text SystemMouseCursor
	// VerticalText is the I-beam for vertical text.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.verticalText
	VerticalText SystemMouseCursor
	// Cell is the cross of a spreadsheet cell.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.cell
	Cell SystemMouseCursor
	// Precise is the crosshair.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.precise
	Precise SystemMouseCursor
	// Move says an item can be moved.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.move
	Move SystemMouseCursor
	// Grab is the open hand, for something that can be grabbed.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.grab
	Grab SystemMouseCursor
	// Grabbing is the closed hand, for something being dragged.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.grabbing
	Grabbing SystemMouseCursor
	// NoDrop says the item cannot be dropped here.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.noDrop
	NoDrop SystemMouseCursor
	// Alias says a shortcut will be made.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.alias
	Alias SystemMouseCursor
	// Copy says the item will be copied.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.copy
	Copy SystemMouseCursor
	// Disappearing says the item will disappear.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.disappearing
	Disappearing SystemMouseCursor
	// AllScroll says the content can scroll in every direction.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.allScroll
	AllScroll SystemMouseCursor
	// ResizeLeftRight is the horizontal double arrow.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeLeftRight
	ResizeLeftRight SystemMouseCursor
	// ResizeUpDown is the vertical double arrow.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpDown
	ResizeUpDown SystemMouseCursor
	// ResizeUpLeftDownRight is the diagonal double arrow, top left to bottom right.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpLeftDownRight
	ResizeUpLeftDownRight SystemMouseCursor
	// ResizeUpRightDownLeft is the diagonal double arrow, top right to bottom left.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpRightDownLeft
	ResizeUpRightDownLeft SystemMouseCursor
	// ResizeUp is the arrow that resizes the top edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUp
	ResizeUp SystemMouseCursor
	// ResizeDown is the arrow that resizes the bottom edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDown
	ResizeDown SystemMouseCursor
	// ResizeLeft is the arrow that resizes the left edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeLeft
	ResizeLeft SystemMouseCursor
	// ResizeRight is the arrow that resizes the right edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeRight
	ResizeRight SystemMouseCursor
	// ResizeUpLeft is the arrow that resizes the top left corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpLeft
	ResizeUpLeft SystemMouseCursor
	// ResizeUpRight is the arrow that resizes the top right corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpRight
	ResizeUpRight SystemMouseCursor
	// ResizeDownLeft is the arrow that resizes the bottom left corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDownLeft
	ResizeDownLeft SystemMouseCursor
	// ResizeDownRight is the arrow that resizes the bottom right corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDownRight
	ResizeDownRight SystemMouseCursor
	// ResizeColumn is the arrow that resizes a column.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeColumn
	ResizeColumn SystemMouseCursor
	// ResizeRow is the arrow that resizes a row.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeRow
	ResizeRow SystemMouseCursor
	// ZoomIn is the magnifier with a plus.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.zoomIn
	ZoomIn SystemMouseCursor
	// ZoomOut is the magnifier with a minus.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.zoomOut
	ZoomOut SystemMouseCursor
}

// SystemMouseCursors are the cursors that the system draws: the same set on every
// platform. It is read-only.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursors
var SystemMouseCursors = systemMouseCursors{
	None:                  SystemMouseCursor{"none"},
	Basic:                 SystemMouseCursor{"basic"},
	Click:                 SystemMouseCursor{"click"},
	Forbidden:             SystemMouseCursor{"forbidden"},
	Wait:                  SystemMouseCursor{"wait"},
	Progress:              SystemMouseCursor{"progress"},
	ContextMenu:           SystemMouseCursor{"contextMenu"},
	Help:                  SystemMouseCursor{"help"},
	Text:                  SystemMouseCursor{"text"},
	VerticalText:          SystemMouseCursor{"verticalText"},
	Cell:                  SystemMouseCursor{"cell"},
	Precise:               SystemMouseCursor{"precise"},
	Move:                  SystemMouseCursor{"move"},
	Grab:                  SystemMouseCursor{"grab"},
	Grabbing:              SystemMouseCursor{"grabbing"},
	NoDrop:                SystemMouseCursor{"noDrop"},
	Alias:                 SystemMouseCursor{"alias"},
	Copy:                  SystemMouseCursor{"copy"},
	Disappearing:          SystemMouseCursor{"disappearing"},
	AllScroll:             SystemMouseCursor{"allScroll"},
	ResizeLeftRight:       SystemMouseCursor{"resizeLeftRight"},
	ResizeUpDown:          SystemMouseCursor{"resizeUpDown"},
	ResizeUpLeftDownRight: SystemMouseCursor{"resizeUpLeftDownRight"},
	ResizeUpRightDownLeft: SystemMouseCursor{"resizeUpRightDownLeft"},
	ResizeUp:              SystemMouseCursor{"resizeUp"},
	ResizeDown:            SystemMouseCursor{"resizeDown"},
	ResizeLeft:            SystemMouseCursor{"resizeLeft"},
	ResizeRight:           SystemMouseCursor{"resizeRight"},
	ResizeUpLeft:          SystemMouseCursor{"resizeUpLeft"},
	ResizeUpRight:         SystemMouseCursor{"resizeUpRight"},
	ResizeDownLeft:        SystemMouseCursor{"resizeDownLeft"},
	ResizeDownRight:       SystemMouseCursor{"resizeDownRight"},
	ResizeColumn:          SystemMouseCursor{"resizeColumn"},
	ResizeRow:             SystemMouseCursor{"resizeRow"},
	ZoomIn:                SystemMouseCursor{"zoomIn"},
	ZoomOut:               SystemMouseCursor{"zoomOut"},
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMouseCursorDeferCreatedSession = "MouseCursorDeferCreatedSession"
	tagMouseCursorFallbackDefers      = "MouseCursorFallbackDefers"
)
