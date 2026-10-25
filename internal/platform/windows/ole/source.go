package ole

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// Decision is what a drag source answers to IDropSource::QueryContinueDrag.
//
// Flutter: no counterpart
type Decision int

// The answers of QueryContinueDrag.
const (
	// Continue goes on with the drag (S_OK).
	//
	// Flutter: no counterpart
	Continue Decision = iota
	// Drop ends the drag with a drop on the target under the pointer (DRAGDROP_S_DROP).
	//
	// Flutter: no counterpart
	Drop
	// Cancel ends the drag without a drop (DRAGDROP_S_CANCEL).
	//
	// Flutter: no counterpart
	Cancel
)

// HRESULT is the value the method returns for the decision.
//
// Flutter: no counterpart
func (d Decision) HRESULT() win32.HRESULT {
	switch d {
	case Drop:
		return win32.DragDropSDrop
	case Cancel:
		return win32.DragDropSCancel
	}
	return win32.SOK
}

// SourceMachine is the drag source's rule for ending a drag (SPEC §25.10): the drop happens when the
// button that started the drag is released, Escape cancels, and the platform can cancel for its own
// reasons (the window of the drag closes, the app ends). It is the same rule as Explorer's. It
// belongs to the UI thread.
//
// Flutter: no counterpart
type SourceMachine struct {
	// Buttons are the MK_* bits of the pointer buttons that started the drag: the drag goes on while
	// one of them is down.
	Buttons uint32
	// Cancelled is set by the platform to end the drag at the next question.
	Cancelled bool
	// LastEffect is the effect that GiveFeedback saw last: what a drop would do now.
	LastEffect uint32
}

// ButtonKeys turns the pointer buttons of the framework (the bits of ui.PointerData.Buttons: primary
// 1, secondary 2, middle 4) into the MK_* bits that a drop target's key state has.
//
// Flutter: no counterpart
func ButtonKeys(buttons int) uint32 {
	var k uint32
	if buttons&1 != 0 {
		k |= keyLButton
	}
	if buttons&2 != 0 {
		k |= keyRButton
	}
	if buttons&4 != 0 {
		k |= keyMButton
	}
	return k
}

// Query answers QueryContinueDrag: Cancel when Escape was pressed or the platform cancelled, Drop
// when the buttons that started the drag are all up, else Continue. A machine with no buttons drops at
// once, as OLE would for a drag that starts with the button already up.
//
// Flutter: no counterpart
func (m *SourceMachine) Query(escape bool, keyState uint32) Decision {
	switch {
	case escape || m.Cancelled:
		return Cancel
	case keyState&m.Buttons == 0:
		return Drop
	}
	return Continue
}

// Feedback records the effect GiveFeedback was told about and says to leave the cursors of Windows.
//
// Flutter: no counterpart
func (m *SourceMachine) Feedback(effect uint32) win32.HRESULT {
	m.LastEffect = effect
	return win32.DragDropSUseDefaultCursors
}
