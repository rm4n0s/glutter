//go:build darwin && !ios && glutter_display

package main

import (
	"image"
	"strconv"
	"testing"
)

// The editor of the tests of this file takes its text from the input method of macOS (SPEC §29.7): the
// scripted steps text, composing and compose call the NSTextInputClient methods of the view as an input
// method does. Each frame, after the steps, must show in the text area what the frame of an editor that
// starts with that text and selection shows (GLUTTER_DEV_TEXT and GLUTTER_DEV_SELECT), pixel for pixel; a
// script ends with a selection, because a caret blinks and two runs would show it in different phases.
// Run them with `go test -tags glutter_display ./examples/editor`.

var macTextArea = image.Rect(16, 36, 480, 412) // the text area of the 820 x 460 window, without the status bar

func TestMacOSScriptedTextAndCompositionsEditTheText(t *testing.T) {
	run := macEditor(t)
	tests := []struct {
		name   string
		script string
		text   string
		sel    string
	}{
		{"typed text", "text:Hello world;key:KeyA,meta", "Hello world", "0,11"},
		{"text with a blank at the end and an emoji", "text:ab 😀 ;key:KeyA,meta", "ab 😀 ", "0,8"},
		{"a dead key and the letter that it accents", "text:caf;compose:´,é;key:KeyA,meta", "café", "0,5"},
		{"a pinyin composition that is committed", "text:Say ;compose:n,ni,你;key:KeyA,meta", "Say 你", "0,7"},
		{"a composition that is committed in the middle of the text", "text:ad;key:ArrowLeft;compose:b,bc,bc;key:KeyA,meta", "abcd", "0,4"},
		{"two compositions one after the other", "compose:ka,か;compose:na,な;key:KeyA,meta", "かな", "0,6"},
		{"Backspace after a committed text", "text:abc;key:Backspace;key:KeyA,meta", "ab", "0,2"},
		{"Return in a multiline field is a line break", "text:ab;key:Enter;text:c;key:KeyA,meta", "ab\nc", "0,4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run("ime", "GLUTTER_DEV_EVENTS=wait:300ms;"+tt.script+";wait:300ms")
			want := run("start", "GLUTTER_DEV_TEXT="+tt.text, "GLUTTER_DEV_SELECT="+tt.sel)
			if n := macDiff(want, got, macTextArea); n != 0 {
				t.Errorf("the text area differs from the editor that starts with %q selected at %s in %d pixels", tt.text, tt.sel, n)
			}
		})
	}
}

// TestMacOSComposingTextIsUnderlined checks that the text being composed is drawn differently from the
// same text once it is committed: Flutter underlines it.
func TestMacOSComposingTextIsUnderlined(t *testing.T) {
	run := macEditor(t)
	composing := run("composing", "GLUTTER_DEV_EVENTS=wait:300ms;text:Hello ;composing:nǐ,nǐhǎo;wait:300ms")
	committed := run("committed", "GLUTTER_DEV_EVENTS=wait:300ms;text:Hello nǐhǎo;wait:300ms")
	// The caret blinks in one and the composition has a selection in the other: only the rows under the text
	// are compared, where the underline is.
	if n := macDiff(composing, committed, image.Rect(16, 62, 480, 72)); n == 0 {
		t.Error("the composing text is drawn like the committed text: no underline")
	}
}

func TestMacOSClipboardKeysCopyCutAndPaste(t *testing.T) {
	run := macEditor(t)
	tests := []struct {
		name   string
		script string
		text   string
	}{
		{"copy everything and paste it at the end", "text:Hello;key:KeyA,meta;key:KeyC,meta;wait:300ms;key:ArrowRight;key:KeyV,meta;wait:500ms", "HelloHello"},
		{"cut everything and paste it back", "text:Hello;key:KeyA,meta;key:KeyX,meta;wait:500ms;key:KeyV,meta;wait:500ms", "Hello"},
		{"copy a word and paste it at the start", "text:Hello world;wait:300ms;key:ArrowLeft,alt+shift;key:KeyC,meta;wait:300ms;key:ArrowLeft,meta;key:KeyV,meta;wait:500ms", "worldHello world"},
		{"paste replaces the selection", "text:Hello world;wait:300ms;key:ArrowLeft,alt+shift;key:KeyC,meta;wait:300ms;key:KeyA,meta;key:KeyV,meta;wait:500ms", "world"},
		{"paste twice", "text:ab;key:KeyA,meta;key:KeyC,meta;wait:300ms;key:ArrowRight;key:KeyV,meta;wait:500ms;key:KeyV,meta;wait:500ms", "ababab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each script ends with Command+A: a caret blinks, a selection does not.
			got := run("clip", "GLUTTER_DEV_EVENTS=wait:300ms;"+tt.script+";key:KeyA,meta;wait:300ms")
			want := run("start", "GLUTTER_DEV_TEXT="+tt.text, "GLUTTER_DEV_SELECT=0,"+strconv.Itoa(len(tt.text)))
			if n := macDiff(want, got, macTextArea); n != 0 {
				t.Errorf("the text area differs from the editor that starts with %q in %d pixels", tt.text, n)
			}
		})
	}
}
