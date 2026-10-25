package ole

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// The bits of the key state that a drop target gets (MK_*); the buttons are the first ones. MK_ALT is
// 0x20, the value that MK_XBUTTON1 has in the mouse messages.
const (
	keyLButton = win32.MKLButton
	keyRButton = win32.MKRButton
	keyMButton = win32.MKMButton
	keyShift   = win32.MKShift
	keyControl = win32.MKControl
	keyAlt     = win32.MKAlt
)

// ActionsOf turns the effects of OLE (DROPEFFECT_*) into the actions of the framework. DROPEFFECT_SCROLL
// and the bits that are not effects are dropped.
//
// Flutter: no counterpart
func ActionsOf(effect uint32) ui.DropActions {
	var a ui.DropActions
	if effect&win32.DropEffectCopy != 0 {
		a |= ui.DropActionCopy
	}
	if effect&win32.DropEffectMove != 0 {
		a |= ui.DropActionMove
	}
	if effect&win32.DropEffectLink != 0 {
		a |= ui.DropActionLink
	}
	return a
}

// EffectOf turns the actions of the framework into the effects of OLE; no action is DROPEFFECT_NONE.
//
// Flutter: no counterpart
func EffectOf(a ui.DropActions) uint32 {
	var e uint32
	if a.Has(ui.DropActionCopy) {
		e |= win32.DropEffectCopy
	}
	if a.Has(ui.DropActionMove) {
		e |= win32.DropEffectMove
	}
	if a.Has(ui.DropActionLink) {
		e |= win32.DropEffectLink
	}
	return e
}

// OfferedActions are the actions the framework may choose from for a drag over a target: the actions
// the source allows, narrowed to the one that the keys ask for when the source allows it. The
// convention of Windows is Ctrl for copy, Shift for move, and Ctrl+Shift or Alt for link; with no key
// the source's whole set is offered, and the framework chooses (copy first).
//
// Flutter: no counterpart
func OfferedActions(keyState uint32, allowed ui.DropActions) ui.DropActions {
	var want ui.DropActions
	switch ctrl, shift, alt := keyState&keyControl != 0, keyState&keyShift != 0, keyState&keyAlt != 0; {
	case ctrl && shift, alt:
		want = ui.DropActionLink
	case ctrl:
		want = ui.DropActionCopy
	case shift:
		want = ui.DropActionMove
	}
	if want != ui.DropActionNone && allowed.Has(want) {
		return want
	}
	return allowed
}

// EffectFor is the effect to return to OLE for the action the framework accepted, among the effects
// the source allows: the accepted action when the source allows it, else the first of copy, move and
// link that both the framework accepted and the source allows, else DROPEFFECT_NONE.
//
// Flutter: no counterpart
func EffectFor(accepted ui.DropActions, allowed uint32) uint32 {
	both := EffectOf(accepted) & allowed
	switch {
	case both&win32.DropEffectCopy != 0:
		return win32.DropEffectCopy
	case both&win32.DropEffectMove != 0:
		return win32.DropEffectMove
	case both&win32.DropEffectLink != 0:
		return win32.DropEffectLink
	}
	return win32.DropEffectNone
}
