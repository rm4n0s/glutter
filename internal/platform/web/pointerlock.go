package web

import "github.com/rm4n0s/glutter/ui"

// The pointer lock of the page (SPEC §27.6): the plain-Go part. The browser's Pointer Lock API hides
// the cursor and reports the motion of the mouse as the movementX and movementY of its events, with
// a position that does not change; the app sees a pointer held where it was, with the motion as its
// delta. pointerlock_js.go calls the API.

// pointerLockInput is the state of PointerInput while the pointer is locked.
type pointerLockInput struct {
	// x and y are the held place, in CSS pixels of the view.
	x, y float64
}

// Lock holds the mouse where it is: the place it was last seen at, or the one given when it has not been
// seen yet (the middle of the view). It returns the pointer data that places the pointer there, with
// no delta. Locking a locked input does nothing.
//
// Flutter: no counterpart
func (in *PointerInput) Lock(x, y, dpr float64) []ui.PointerData {
	if in.lock != nil {
		return nil
	}
	if in.seen {
		x, y = in.lastX, in.lastY
	}
	in.lock = &pointerLockInput{x: x, y: y}
	buttons := in.mouseButtons()
	e := PointerEvent{PointerType: "mouse", PointerID: 1, Buttons: buttons, X: x, Y: y}
	data := in.convert(nil, e, in.sanitizer(mouseDeviceID).Move(buttons), mouseDeviceID, dpr)
	for i := range data {
		data[i].PhysicalDeltaX, data[i].PhysicalDeltaY = 0, 0
	}
	return data
}

// Unlock lets the mouse go. The next event of the mouse has its own place.
//
// Flutter: no counterpart
func (in *PointerInput) Unlock() { in.lock = nil }

// Locked says whether the pointer is locked.
//
// Flutter: no counterpart
func (in *PointerInput) Locked() bool { return in.lock != nil }

// mouseButtons are the buttons of the mouse that the sanitizer saw go down and not up.
func (in *PointerInput) mouseButtons() int {
	if s, ok := in.sanitizers[mouseDeviceID]; ok {
		return s.pressedButtons
	}
	return 0
}

// held gives an event of the mouse the held place while the pointer is locked: the browser reports
// the place the lock began at, but a button event just after the lock may still carry the old one.
func (in *PointerInput) held(e PointerEvent) PointerEvent {
	if in.lock != nil && pointerKind(e.PointerType) == ui.PointerDeviceKindMouse {
		e.X, e.Y = in.lock.x, in.lock.y
	}
	return e
}

// lockedMove converts a pointermove of a locked mouse: the hover (or the move, with a button down) is
// at the held place, and its delta is the motion the event reports, in physical pixels.
func (in *PointerInput) lockedMove(e PointerEvent, dpr float64) []ui.PointerData {
	events := e.Coalesced
	if len(events) == 0 {
		events = []PointerEvent{e}
	}
	device := deviceOf(e)
	s := in.sanitizer(device)
	var data []ui.PointerData
	for _, ev := range events {
		if ev.PointerType == "" {
			ev.PointerType = e.PointerType
		}
		ev = in.held(ev)
		in.checkModifiers(ev)
		if ev.MovementX == 0 && ev.MovementY == 0 {
			continue
		}
		n := len(data)
		data = in.convert(data, ev, s.Move(ev.Buttons), device, dpr)
		for i := n; i < len(data); i++ {
			data[i].PhysicalDeltaX, data[i].PhysicalDeltaY = ev.MovementX*dpr, ev.MovementY*dpr
		}
	}
	return data
}

// lockRequest says what the platform decided about a request for the lock.
type lockRequest int

const (
	// lockAsk means the request goes to the browser.
	lockAsk lockRequest = iota
	// lockUnsupported means the browser has no Pointer Lock API.
	lockUnsupported
	// lockNoGesture means the page has no user activation, which the browser requires.
	lockNoGesture
)

// pointerLockPhase is where the lock is.
type pointerLockPhase int

const (
	lockIdle pointerLockPhase = iota
	// lockRequested is a request the browser has not answered.
	lockRequested
	// lockHeld is a lock the browser granted.
	lockHeld
)

// pointerLockState follows the lock through the events of the browser: the request, the answer
// (pointerlockchange or pointerlockerror, or the rejection of the promise), and the end, which the
// user (Escape), the browser or the app causes. It tells which of the ends the app has to hear of.
type pointerLockState struct {
	phase pointerLockPhase
	// leaving says the app asked to unlock a lock that was granted, and pointerlockchange has not
	// come yet.
	leaving bool
}

// request decides what to do with a request for the lock. It asks the browser only with user
// activation (the browser would refuse otherwise, asynchronously).
func (s *pointerLockState) request(supported, activation bool) lockRequest {
	switch {
	case s.phase != lockIdle:
		return lockAsk
	case !supported:
		return lockUnsupported
	case !activation:
		return lockNoGesture
	}
	s.phase = lockRequested
	return lockAsk
}

// pending says whether the lock is requested or held.
func (s *pointerLockState) pending() bool { return s.phase != lockIdle }

// unlock records that the app unlocks, and says whether the browser has to be told (exitPointerLock).
func (s *pointerLockState) unlock() (exit bool) {
	switch s.phase {
	case lockHeld:
		s.leaving = true
		s.phase = lockIdle
		return true
	case lockRequested:
		// The browser may still grant it: changed(true) then asks to leave.
		s.phase = lockIdle
	}
	return false
}

// changed handles pointerlockchange: ours says whether the pointer lock element is the canvas now. It
// reports whether the lock ended by itself (the app has to hear of it), and whether a lock that the
// app gave up in the meantime was granted and has to be left.
func (s *pointerLockState) changed(ours bool) (ended, leave bool) {
	if ours {
		if s.phase == lockIdle {
			return false, true
		}
		s.phase = lockHeld
		return false, false
	}
	if s.leaving {
		s.leaving = false
		return false, false
	}
	if s.phase != lockIdle {
		s.phase = lockIdle
		return true, false
	}
	return false, false
}

// failed handles pointerlockerror, or the rejection of the request: a request that the browser
// refused ends the lock the app thinks it has.
func (s *pointerLockState) failed() (ended bool) {
	if s.phase == lockIdle {
		return false
	}
	s.phase = lockIdle
	return true
}
