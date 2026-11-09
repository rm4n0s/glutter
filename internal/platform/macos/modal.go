package macos

import "github.com/rm4n0s/glutter/ui"

// This file is the state machine of the modal windows (SPEC §29.4). A modal window is run with its parent
// disabled, not as a sheet: the parent takes no input, the focus goes to the modal window, and the parent has
// the keys again when the modal window closes. It is plain Go, so that its rules are tested on every host;
// modal_darwin.go has the glue that drops the events and moves the focus.

// modalEntry is an open modal window and the window it blocks.
type modalEntry struct {
	modal, parent ui.ViewID
}

// modalStack is the open modal windows, in the order they were opened.
//
// Flutter: no counterpart
type modalStack struct {
	entries []modalEntry
}

// open adds a modal window that blocks its parent. A modal window that is open already is not added again.
func (s *modalStack) open(modal, parent ui.ViewID) {
	for _, e := range s.entries {
		if e.modal == modal {
			return
		}
	}
	s.entries = append(s.entries, modalEntry{modal: modal, parent: parent})
}

// close removes a modal window and returns the window that it blocked. ok is false when the window was
// not a modal one that was open.
func (s *modalStack) close(modal ui.ViewID) (parent ui.ViewID, ok bool) {
	for i, e := range s.entries {
		if e.modal == modal {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return e.parent, true
		}
	}
	return 0, false
}

// blocker is the modal window that blocks the view: the one that was opened last of those that have the
// view as their parent.
func (s *modalStack) blocker(view ui.ViewID) (ui.ViewID, bool) {
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].parent == view {
			return s.entries[i].modal, true
		}
	}
	return 0, false
}

// blocked says that the view takes no input because a modal window is open over it.
func (s *modalStack) blocked(view ui.ViewID) bool {
	_, ok := s.blocker(view)
	return ok
}

// target is the window that gets the input and the focus in place of the view: the view itself when
// nothing blocks it, else the innermost modal window of the chain that blocks it (a modal window that opened
// another one is blocked too).
func (s *modalStack) target(view ui.ViewID) ui.ViewID {
	for range len(s.entries) + 1 {
		next, ok := s.blocker(view)
		if !ok {
			return view
		}
		view = next
	}
	return view
}

// open reports whether any modal window is open.
func (s *modalStack) any() bool { return len(s.entries) > 0 }
