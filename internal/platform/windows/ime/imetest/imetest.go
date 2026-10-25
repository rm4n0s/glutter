// Package imetest is a fake of the input method manager (ime.IMM) for the tests of the text input of
// Windows, in the ime package and in the window code that uses it. It answers from its fields and
// records the calls that matter.
//
// Flutter: no counterpart
package imetest

import (
	"strconv"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// Fake is an input method manager that answers from fields.
//
// Flutter: no counterpart
type Fake struct {
	// Off says ImmGetContext gives no context (the input method is off for the window).
	Off bool
	// Comp and Result are the composition string and the result string of the input context, and
	// Cursor the caret in the composition string, in UTF-16 units.
	Comp, Result []uint16
	Cursor       int32
	// FocusWindow is the window that has the keyboard focus.
	FocusWindow uintptr

	// Calls are the calls that change something, as text: "associate 0", "associate-ex 0 16",
	// "create-caret 1x1", "destroy-caret", "notify 21 4 0", "clear".
	Calls []string
	// CompForm and CandForm are the last places given to the windows of the input method, and
	// Placements counts the calls that gave them.
	CompForm   *win32.CompositionForm
	CandForm   *win32.CandidateForm
	Placements int
	// CaretPos is the last place of the system caret.
	CaretPos win32.Point
	// Contexts counts the contexts asked for and Released the contexts given back; ValueCalls counts
	// the reads of a number of the composition (the caret).
	Contexts, Released, ValueCalls int
}

// New makes a fake with the keyboard focus in the window hwnd.
//
// Flutter: no counterpart
func New(hwnd uintptr) *Fake { return &Fake{FocusWindow: hwnd} }

// Context is the input context 77, or 0 while Off.
func (f *Fake) Context(uintptr) uintptr {
	f.Contexts++
	if f.Off {
		return 0
	}
	return 77
}

// Release counts the context given back.
func (f *Fake) Release(uintptr, uintptr) { f.Released++ }

// String is the composition string or the result string.
func (f *Fake) String(_ uintptr, index uint32) ([]uint16, bool) {
	switch index {
	case win32.GCSCompStr:
		return f.Comp, true
	case win32.GCSResultStr:
		return f.Result, true
	}
	return nil, false
}

// Value is the caret in the composition string.
func (f *Fake) Value(_ uintptr, index uint32) int32 {
	f.ValueCalls++
	if index == win32.GCSCursorPos {
		return f.Cursor
	}
	return -1
}

// SetCompositionWindow keeps the form.
func (f *Fake) SetCompositionWindow(_ uintptr, form *win32.CompositionForm) bool {
	c := *form
	f.CompForm = &c
	f.Placements++
	return true
}

// SetCandidateWindow keeps the form.
func (f *Fake) SetCandidateWindow(_ uintptr, form *win32.CandidateForm) bool {
	c := *form
	f.CandForm = &c
	return true
}

// Notify records the call.
func (f *Fake) Notify(_ uintptr, action, index, value uint32) bool {
	f.Calls = append(f.Calls, "notify "+u(action)+" "+u(index)+" "+u(value))
	return true
}

// SetCompositionString records that the string was cleared.
func (f *Fake) SetCompositionString(uintptr, []uint16, []uint16) bool {
	f.Calls = append(f.Calls, "clear")
	return true
}

// AssociateContext records the call.
func (f *Fake) AssociateContext(_, ctx uintptr) uintptr {
	f.Calls = append(f.Calls, "associate "+u(uint32(ctx)))
	f.Off = ctx == 0
	return 77
}

// AssociateContextEx records the call.
func (f *Fake) AssociateContextEx(_, ctx uintptr, flags uint32) bool {
	f.Calls = append(f.Calls, "associate-ex "+u(uint32(ctx))+" "+u(flags))
	f.Off = false
	return true
}

// CreateCaret records the call.
func (f *Fake) CreateCaret(_ uintptr, w, h int32) bool {
	f.Calls = append(f.Calls, "create-caret "+strconv.Itoa(int(w))+"x"+strconv.Itoa(int(h)))
	return true
}

// DestroyCaret records the call.
func (f *Fake) DestroyCaret() bool {
	f.Calls = append(f.Calls, "destroy-caret")
	return true
}

// SetCaretPos keeps the place.
func (f *Fake) SetCaretPos(x, y int32) bool {
	f.CaretPos = win32.Point{X: x, Y: y}
	return true
}

// Focus is the window with the keyboard focus.
func (f *Fake) Focus() uintptr { return f.FocusWindow }

func u(n uint32) string { return strconv.FormatUint(uint64(n), 10) }
