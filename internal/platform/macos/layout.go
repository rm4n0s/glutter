package macos

// This file is how the keyboard layout becomes the logical keys of the typewriter keys (SPEC §29.6). A shortcut
// such as Command+C must work on a French or a Russian layout, so the logical key of a key is not always the
// character that the layout makes: the layout goals say which characters must be found on some key, and the
// layout of the user fills them in. The lookup of the character that a key makes is in keyboard_darwin.go.

// layoutGoal is a logical key that a keyboard layout must offer on some key: the key code of the US layout,
// the character, and whether the character must be found even when it has to be put on a key where the
// layout makes a different one.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/KeyCodeMap_Internal.h LayoutGoal
type layoutGoal struct {
	keyCode   uint16
	logical   uint64
	mandatory bool
}

// layoutClue is the character that a key makes in a layout, without and with Shift. A dead key makes the
// character that stands for it (an accent), and is marked.
//
// Flutter: KeyCodeMap_Internal.h LayoutClue
type layoutClue struct {
	character uint32
	dead      bool
}

// lastTypewriterKeyCode is the last key code of the letters, digits and symbols.
const lastTypewriterKeyCode = 0x32

// isEASCII says that a clue is a character of Latin-1 and not a dead key.
func (c layoutClue) isEASCII() bool { return c.character < 256 && !c.dead }

// buildLayout makes the logical key of each typewriter key of the layout that lookup describes. The logical
// key of a key code is the first of:
//
//   - a mandatory goal that one of the key's two characters meets, so that every letter and digit can be
//     found on some key;
//   - the key of the US layout, when neither character of the key is Latin-1 (the layout of an alphabet that
//     is not Latin), so that there is no logical key outside Latin for a shortcut;
//   - none: the logical key is made at the event from the key code and the characters.
//
// A mandatory goal that no key met is put on the key of the US layout.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/FlutterKeyboardManager.mm
// buildLayout
func buildLayout(goals []layoutGoal, lookup func(keyCode uint16, shift bool) layoutClue) map[uint16]uint64 {
	layout := map[uint16]uint64{}
	mandatoryByChar := map[uint64]layoutGoal{}
	usByKeyCode := map[uint16]layoutGoal{}
	for _, g := range goals {
		if g.mandatory {
			mandatoryByChar[g.logical] = g
		} else {
			usByKeyCode[g.keyCode] = g
		}
	}
	for keyCode := uint16(0); keyCode <= lastTypewriterKeyCode; keyCode++ {
		clues := [2]layoutClue{lookup(keyCode, false), lookup(keyCode, true)}
		for _, clue := range clues {
			var char uint64
			if !clue.dead {
				char = uint64(clue.character)
			}
			if _, ok := mandatoryByChar[char]; ok {
				layout[keyCode] = char
				delete(mandatoryByChar, char)
				break
			}
		}
		if g, ok := usByKeyCode[keyCode]; ok {
			if _, assigned := layout[keyCode]; !assigned && !clues[0].isEASCII() && !clues[1].isEASCII() {
				layout[keyCode] = g.logical
			}
		}
	}
	for _, g := range mandatoryByChar {
		layout[g.keyCode] = g.logical
	}
	return layout
}
