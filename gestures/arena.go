package gestures

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
)

// GestureDisposition is how a member of the arena resolves.
//
// Flutter: gestures/arena.dart GestureDisposition
type GestureDisposition int

const (
	// GestureAccepted means the member wins the arena.
	//
	// Flutter: gestures/arena.dart GestureDisposition.accepted
	GestureAccepted GestureDisposition = iota + 1
	// GestureRejected means the member gives up.
	//
	// Flutter: gestures/arena.dart GestureDisposition.rejected
	GestureRejected
)

// Name returns the Dart name of the disposition.
//
// Flutter: gestures/arena.dart GestureDisposition
func (d GestureDisposition) Name() string {
	switch d {
	case GestureAccepted:
		return "accepted"
	case GestureRejected:
		return "rejected"
	}
	return "GestureDisposition(" + strconv.Itoa(int(d)) + ")"
}

// GestureArenaMember is a participant in the arena: it learns whether it won
// or lost.
//
// Flutter: gestures/arena.dart GestureArenaMember
type GestureArenaMember interface {
	// AcceptGesture is called when the member wins the arena for the pointer.
	//
	// Flutter: gestures/arena.dart GestureArenaMember.acceptGesture
	AcceptGesture(pointer int)
	// RejectGesture is called when the member loses the arena for the
	// pointer.
	//
	// Flutter: gestures/arena.dart GestureArenaMember.rejectGesture
	RejectGesture(pointer int)
}

// GestureArenaEntry is a member's handle in the arena of a pointer. The member
// uses it to resolve.
//
// Flutter: gestures/arena.dart GestureArenaEntry
type GestureArenaEntry interface {
	// Resolve accepts or rejects the entry's member in the arena. Resolving
	// again after the arena is resolved has no effect.
	//
	// Flutter: gestures/arena.dart GestureArenaEntry.resolve
	Resolve(disposition GestureDisposition)
}

type arenaEntry struct {
	arena   *GestureArenaManager
	pointer int
	member  GestureArenaMember
	// state is the arena the member joined. A pointer id can be given again
	// to a new pointer, so the entry resolves only in this arena, never in
	// another arena that has the same pointer.
	state *gestureArena
}

func (e *arenaEntry) Resolve(disposition GestureDisposition) {
	e.arena.resolve(e.pointer, e.state, e.member, disposition)
}

type gestureArena struct {
	members         []GestureArenaMember
	isOpen          bool
	isHeld          bool
	hasPendingSweep bool
	eagerWinner     GestureArenaMember
}

func (a *gestureArena) String() string {
	var sb strings.Builder
	if len(a.members) == 0 {
		sb.WriteString("<empty>")
	} else {
		for i, m := range a.members {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(describeMember(m))
			if m == a.eagerWinner {
				sb.WriteString(" (eager winner)")
			}
		}
	}
	if a.isOpen {
		sb.WriteString(" [open]")
	}
	if a.isHeld {
		sb.WriteString(" [held]")
	}
	if a.hasPendingSweep {
		sb.WriteString(" [hasPendingSweep]")
	}
	return sb.String()
}

func describeMember(m GestureArenaMember) string {
	if s, ok := m.(fmt.Stringer); ok {
		return s.String()
	}
	return foundation.ToStringShort(m)
}

// GestureArenaManagerOpts are the optional parameters of
// NewGestureArenaManager.
//
// Flutter: gestures/arena.dart GestureArenaManager.new
type GestureArenaManagerOpts struct {
	// Debug gives the arena its diagnostics flags; nil prints nothing.
	Debug *DebugOptions
}

// GestureArenaManager decides, for each pointer, which of the gestures that
// compete for it wins. The members that joined the arena of a pointer are
// resolved after its up event: the first member wins unless one of them has
// claimed the win or the others gave up earlier.
//
// Flutter: gestures/arena.dart GestureArenaManager
type GestureArenaManager struct {
	scheduleMicrotask func(fn func())
	debug             *DebugOptions
	arenas            map[int]*gestureArena
}

// NewGestureArenaManager creates a manager. scheduleMicrotask runs a function
// after the work that is running now, before the next event; the binding
// passes the dispatcher's Post.
//
// Flutter: gestures/arena.dart GestureArenaManager.new
func NewGestureArenaManager(scheduleMicrotask func(fn func()), opts ...GestureArenaManagerOpts) *GestureArenaManager {
	o := oneOpts(opts)
	return &GestureArenaManager{
		scheduleMicrotask: scheduleMicrotask,
		debug:             o.Debug,
		arenas:            map[int]*gestureArena{},
	}
}

// Add adds the member to the arena of the pointer, opening the arena if it
// does not exist. Members can only join an open arena.
//
// Flutter: gestures/arena.dart GestureArenaManager.add
func (m *GestureArenaManager) Add(pointer int, member GestureArenaMember) GestureArenaEntry {
	state, ok := m.arenas[pointer]
	if !ok {
		state = &gestureArena{isOpen: true}
		m.arenas[pointer] = state
		m.log(pointer, "★ Opening new gesture arena.", nil)
	}
	if foundation.DebugMode && !state.isOpen {
		panic(errors.New(tagGestureArenaClosed, "a member cannot join an arena that is closed", "pointer", pointer))
	}
	state.members = append(state.members, member)
	m.log(pointer, "Adding: "+describeMember(member), nil)
	return &arenaEntry{arena: m, pointer: pointer, member: member, state: state}
}

// Close prevents new members from entering the arena of the pointer, and
// resolves it if it can be resolved. Call it when the pointer's down event has
// been dispatched.
//
// Flutter: gestures/arena.dart GestureArenaManager.close
func (m *GestureArenaManager) Close(pointer int) {
	state := m.arenas[pointer]
	if state == nil {
		return
	}
	state.isOpen = false
	m.log(pointer, "Closing", state)
	m.tryToResolveArena(pointer, state)
}

// Sweep forces the arena of the pointer to resolve: the first member wins and
// the others lose. Call it when the pointer's up event has been dispatched. An
// arena that is held resolves when it is released.
//
// Flutter: gestures/arena.dart GestureArenaManager.sweep
func (m *GestureArenaManager) Sweep(pointer int) {
	state := m.arenas[pointer]
	if state == nil {
		return
	}
	if foundation.DebugMode && state.isOpen {
		panic(errors.New(tagGestureArenaOpenOnSweep, "an arena must be closed before it is swept", "pointer", pointer))
	}
	if state.isHeld {
		state.hasPendingSweep = true
		m.log(pointer, "Delaying sweep", state)
		return
	}
	m.log(pointer, "Sweeping", state)
	delete(m.arenas, pointer)
	if len(state.members) > 0 {
		m.log(pointer, "Winner: "+describeMember(state.members[0]), nil)
		state.members[0].AcceptGesture(pointer)
		for _, member := range state.members[1:] {
			member.RejectGesture(pointer)
		}
	}
}

// Hold keeps the arena of the pointer from being swept, so that a gesture that
// ends later (such as a double tap) can still win. Call Release to let it
// resolve.
//
// Flutter: gestures/arena.dart GestureArenaManager.hold
func (m *GestureArenaManager) Hold(pointer int) {
	state := m.arenas[pointer]
	if state == nil {
		return
	}
	state.isHeld = true
	m.log(pointer, "Holding", state)
}

// Release lets the arena of the pointer be swept again, and sweeps it if a
// sweep was waiting.
//
// Flutter: gestures/arena.dart GestureArenaManager.release
func (m *GestureArenaManager) Release(pointer int) {
	state := m.arenas[pointer]
	if state == nil {
		return
	}
	state.isHeld = false
	m.log(pointer, "Releasing", state)
	if state.hasPendingSweep {
		m.Sweep(pointer)
	}
}

func (m *GestureArenaManager) resolve(pointer int, entryState *gestureArena, member GestureArenaMember, disposition GestureDisposition) {
	state := m.arenas[pointer]
	if state == nil || state != entryState {
		return // The arena of the entry has resolved already (and the pointer may have a new one).
	}
	i := indexOfMember(state.members, member)
	if foundation.DebugMode && i < 0 {
		panic(errors.New(tagGestureArenaMemberUnknown, "the member is not in the arena of the pointer", "pointer", pointer))
	}
	if i < 0 {
		return // The member lost already; it is not told again, and it cannot win.
	}
	switch disposition {
	case GestureAccepted:
		m.log(pointer, "Accepting: "+describeMember(member), nil)
		if state.isOpen {
			if state.eagerWinner == nil {
				state.eagerWinner = member
			}
		} else {
			m.log(pointer, "Self-declared winner: "+describeMember(member), nil)
			m.resolveInFavorOf(pointer, state, member)
		}
	case GestureRejected:
		m.log(pointer, "Rejecting: "+describeMember(member), nil)
		state.members = append(state.members[:i:i], state.members[i+1:]...)
		member.RejectGesture(pointer)
		if !state.isOpen {
			m.tryToResolveArena(pointer, state)
		}
	}
}

func indexOfMember(members []GestureArenaMember, member GestureArenaMember) int {
	for i, m := range members {
		if m == member {
			return i
		}
	}
	return -1
}

func (m *GestureArenaManager) tryToResolveArena(pointer int, state *gestureArena) {
	if m.arenas[pointer] != state {
		// A member's callback resolved the arena while it was being resolved
		// (Flutter asserts that it did not). The pointer may have a new arena
		// by now, which this one must not touch.
		return
	}
	switch {
	case len(state.members) == 1:
		m.scheduleMicrotask(func() { m.resolveByDefault(pointer, state) })
	case len(state.members) == 0:
		delete(m.arenas, pointer)
		m.log(pointer, "Arena empty.", nil)
	case state.eagerWinner != nil:
		m.log(pointer, "Eager winner: "+describeMember(state.eagerWinner), nil)
		m.resolveInFavorOf(pointer, state, state.eagerWinner)
	}
}

func (m *GestureArenaManager) resolveByDefault(pointer int, state *gestureArena) {
	// Flutter checks only that the pointer has an arena. Here the function runs
	// as a post to the UI thread, so other pointer events can run first: the
	// arena it was posted for can have resolved or been emptied, and the pointer
	// can have a new arena by now (a platform gives a new pointer the id of one
	// that is finished).
	if m.arenas[pointer] != state || len(state.members) != 1 {
		return // This arena has already resolved.
	}
	delete(m.arenas, pointer)
	m.log(pointer, "Default winner: "+describeMember(state.members[0]), nil)
	state.members[0].AcceptGesture(pointer)
}

func (m *GestureArenaManager) resolveInFavorOf(pointer int, state *gestureArena, member GestureArenaMember) {
	delete(m.arenas, pointer)
	for _, rejected := range state.members {
		if rejected != member {
			rejected.RejectGesture(pointer)
		}
	}
	member.AcceptGesture(pointer)
}

func (m *GestureArenaManager) log(pointer int, message string, state *gestureArena) {
	if !foundation.DebugMode || m.debug == nil || !m.debug.PrintGestureArenaDiagnostics {
		return
	}
	suffix := ""
	if state != nil {
		count := len(state.members)
		s := "s"
		if count == 1 {
			s = ""
		}
		suffix = fmt.Sprintf(" with %d member%s.", count, s)
	}
	m.debug.print(fmt.Sprintf("Gesture arena %-4d ❙ %s%s", pointer, message, suffix))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGestureArenaClosed        = "GestureArenaClosed"
	tagGestureArenaMemberUnknown = "GestureArenaMemberUnknown"
	tagGestureArenaOpenOnSweep   = "GestureArenaOpenOnSweep"
)
