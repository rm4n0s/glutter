package android

import (
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// decodeAction turns an action of the Java side (Native.a11yAction) into the event for the
// framework. action is the ui.SemanticsAction bit that AccessibilityBridge.performAction chose for
// the request of the system; args depends on it, little-endian:
//
//	MoveCursor*       u8 extendSelection (0 or 1)             -> bool
//	SetSelection      i32 base, i32 extent, UTF-8 byte offsets  -> map[string]int{"base", "extent"}
//	SetText           the new text, UTF-8                       -> string
//	CustomAction      i32 id of the custom action                -> int
//	ScrollToOffset    f32 dx, f32 dy                              -> []float64{dx, dy}
//	every other       no bytes                                    -> nil
//
// The Java side counts offsets in UTF-16 units and converts them to bytes of the node's value before
// it sends them, because it holds the value. It fails with AndroidSemanticsActionInvalid on an
// action that is not one bit of ui.SemanticsAction, or on arguments of another size.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/AccessibilityChannel.java AccessibilityChannel.dispatchSemanticsAction
func decodeAction(nodeID, action int, args []byte) (ui.SemanticsActionEvent, error) {
	act := ui.SemanticsAction(action)
	if !slices.Contains(ui.SemanticsActionValues(), act) {
		return ui.SemanticsActionEvent{}, errors.New(tagAndroidSemanticsActionInvalid, "the action is not one semantics action", "action", action)
	}
	ev := ui.SemanticsActionEvent{ViewID: ui.ImplicitViewID, NodeID: nodeID, Type: act}
	le32 := func(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24 }
	want := -1 // the size of the arguments, or -1 for any
	switch act {
	case ui.SemanticsActionMoveCursorForwardByCharacter, ui.SemanticsActionMoveCursorBackwardByCharacter,
		ui.SemanticsActionMoveCursorForwardByWord, ui.SemanticsActionMoveCursorBackwardByWord:
		want = 1
	case ui.SemanticsActionSetSelection, ui.SemanticsActionScrollToOffset:
		want = 8
	case ui.SemanticsActionCustomAction:
		want = 4
	case ui.SemanticsActionSetText:
	default:
		want = 0
	}
	if want >= 0 && len(args) != want {
		return ui.SemanticsActionEvent{}, errors.New(tagAndroidSemanticsActionInvalid, "the arguments of the action have another size",
			"action", act.Name(), "size", len(args), "want", want)
	}
	switch act {
	case ui.SemanticsActionMoveCursorForwardByCharacter, ui.SemanticsActionMoveCursorBackwardByCharacter,
		ui.SemanticsActionMoveCursorForwardByWord, ui.SemanticsActionMoveCursorBackwardByWord:
		ev.Arguments = args[0] != 0
	case ui.SemanticsActionSetSelection:
		ev.Arguments = map[string]int{"base": int(int32(le32(args))), "extent": int(int32(le32(args[4:])))}
	case ui.SemanticsActionSetText:
		ev.Arguments = string(args)
	case ui.SemanticsActionCustomAction:
		ev.Arguments = int(int32(le32(args)))
	case ui.SemanticsActionScrollToOffset:
		ev.Arguments = []float64{float64(math.Float32frombits(le32(args))), float64(math.Float32frombits(le32(args[4:])))}
	}
	return ev, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagAndroidSemanticsActionInvalid = "AndroidSemanticsActionInvalid"
