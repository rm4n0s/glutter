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
	panic(errors.New("MouseCursorDeferCreatedSession", "the defer cursor can not create a session"))
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
	lastSession         map[int]MouseCursorSession
}

// NewMouseCursorManager creates a manager. The fallback must not be
// MouseCursorDefer.
//
// Flutter: services/mouse_cursor.dart MouseCursorManager.new
func NewMouseCursorManager(fallback MouseCursor) *MouseCursorManager {
	if foundation.DebugMode && fallback == MouseCursorDefer {
		panic(errors.New("MouseCursorFallbackDefers", "the fallback mouse cursor can not defer"))
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
	m.lastSession[device] = nextSession
	if last != nil {
		last.Dispose()
	}
	nextSession.Activate()
}
