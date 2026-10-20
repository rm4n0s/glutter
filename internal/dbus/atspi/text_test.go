package atspi

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestRuneAndByteOffsets(t *testing.T) {
	const s = "aé😀b" // 1 + 2 + 4 + 1 bytes, 4 characters
	tests := []struct {
		bytes, runes int
	}{{-3, 0}, {0, 0}, {1, 1}, {2, 2}, {3, 2}, {4, 3}, {5, 3}, {6, 3}, {7, 3}, {8, 4}, {100, 4}}
	for _, tt := range tests {
		if got := runeOffset(s, tt.bytes); got != tt.runes {
			t.Errorf("runeOffset(%d) = %d, want %d", tt.bytes, got, tt.runes)
		}
	}
	back := []struct{ runes, bytes int }{{-1, 0}, {0, 0}, {1, 1}, {2, 3}, {3, 7}, {4, 8}, {9, 8}}
	for _, tt := range back {
		if got := byteOffset(s, tt.runes); got != tt.bytes {
			t.Errorf("byteOffset(%d) = %d, want %d", tt.runes, got, tt.bytes)
		}
	}
	if runeOffset("", 3) != 0 || byteOffset("", 3) != 0 {
		t.Error("an empty text has no offsets")
	}
}

func TestCaretOf(t *testing.T) {
	n := &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Value: "aé😀b", TextSelectionBase: 1, TextSelectionExtent: 7}}
	if caret, base := caretOf(n); caret != 3 || base != 1 {
		t.Errorf("caret %d base %d", caret, base)
	}
	n.TextSelectionBase = -1
	if caret, base := caretOf(n); caret != -1 || base != -1 {
		t.Errorf("no selection: caret %d base %d", caret, base)
	}
}

func TestTextPieces(t *testing.T) {
	const text = "Hello big world. Next one!  Last\nsecond line\n\nend"
	tests := []struct {
		name       string
		u          unit
		offset     int
		delta      int
		want       string
		start, end int
	}{
		{"char", unitChar, 1, 0, "e", 1, 2},
		{"the char before", unitChar, 1, -1, "H", 0, 1},
		{"the char after", unitChar, 1, 1, "l", 2, 3},
		{"the char at the end", unitChar, len([]rune(text)), 0, "d", 48, 49},
		{"word at its start", unitWord, 0, 0, "Hello ", 0, 6},
		{"word in the middle", unitWord, 8, 0, "big ", 6, 10},
		{"word before", unitWord, 8, -1, "Hello ", 0, 6},
		{"word after", unitWord, 8, 1, "world. ", 10, 17},
		{"word before the first", unitWord, 2, -1, "", 2, 2},
		{"word after the last", unitWord, 47, 1, "", 47, 47},
		{"word before the last", unitWord, 47, -1, "line\n\n", 40, 46},
		{"sentence", unitSentence, 3, 0, "Hello big world. ", 0, 17},
		{"sentence after", unitSentence, 3, 1, "Next one!  ", 17, 28},
		{"the last sentence", unitSentence, 48, 0, "end", 46, 49},
		{"sentence that ends with a new line", unitSentence, 36, 0, "second line\n", 33, 45},
		{"line", unitLine, 3, 0, "Hello big world. Next one!  Last\n", 0, 33},
		{"line after", unitLine, 3, 1, "second line\n", 33, 45},
		{"an empty line", unitLine, 45, 0, "\n", 45, 46},
		{"paragraph", unitParagraph, 36, 0, "second line\n", 33, 45},
	}
	runes := []rune(text)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, start, end := pick(runes, tt.u, tt.offset, tt.delta)
			if got != tt.want || start != tt.start || end != tt.end {
				t.Errorf("pick = %q [%d, %d), want %q [%d, %d)", got, start, end, tt.want, tt.start, tt.end)
			}
		})
	}
	if got, s, e := pick(nil, unitWord, 0, 0); got != "" || s != 0 || e != 0 {
		t.Errorf("empty text: %q %d %d", got, s, e)
	}
	if got, _, _ := pick(runes, unitWord, -1, 0); got != "" {
		t.Errorf("a negative offset gave %q", got)
	}
	if got, _, _ := pick(runes, unitWord, 1000, 0); got != "" {
		t.Errorf("an offset past the end gave %q", got)
	}
}

func TestSegmentsCoverTheText(t *testing.T) {
	for _, text := range []string{"", "a", "  a  b ", "one. two! three?\nfour\n\n", "é😀 x\n"} {
		runes := []rune(text)
		for u := unitChar; u <= unitParagraph; u++ {
			pos := 0
			for _, s := range segments(runes, u) {
				if s.start != pos || s.end <= s.start {
					t.Errorf("%q unit %d: segment %v after %d", text, u, s, pos)
				}
				pos = s.end
			}
			if pos != len(runes) {
				t.Errorf("%q unit %d: covers %d of %d", text, u, pos, len(runes))
			}
		}
	}
}

func TestGranularityAndBoundary(t *testing.T) {
	for g := uint32(0); g <= 4; g++ {
		if u, ok := granularityUnit(g); !ok || uint32(u) != g {
			t.Errorf("granularity %d = %d, %v", g, u, ok)
		}
	}
	if _, ok := granularityUnit(5); ok {
		t.Error("granularity 5")
	}
	want := map[uint32]unit{0: unitChar, 1: unitWord, 2: unitWord, 3: unitSentence, 4: unitSentence, 5: unitLine, 6: unitLine}
	for b, u := range want {
		if got, ok := boundaryUnit(b); !ok || got != u {
			t.Errorf("boundary %d = %d, %v", b, got, ok)
		}
	}
	if _, ok := boundaryUnit(7); ok {
		t.Error("boundary 7")
	}
}

func TestTextDiff(t *testing.T) {
	tests := []struct {
		old, now          string
		start             int
		deleted, inserted string
	}{
		{"hello", "hello!", 5, "", "!"},
		{"hello", "hell", 4, "o", ""},
		{"hello", "heXllo", 2, "", "X"},
		{"hello", "hllo", 1, "e", ""},
		{"hello", "jello", 0, "h", "j"},
		{"hello world", "hello there", 6, "world", "there"},
		{"", "abc", 0, "", "abc"},
		{"abc", "", 0, "abc", ""},
		{"same", "same", 4, "", ""},
		{"aaa", "aa", 2, "a", ""},
		{"é😀", "é", 1, "😀", ""},
	}
	for _, tt := range tests {
		start, del, ins := textDiff([]rune(tt.old), []rune(tt.now))
		if start != tt.start || string(del) != tt.deleted || string(ins) != tt.inserted {
			t.Errorf("%q -> %q: start %d, deleted %q, inserted %q; want %d %q %q",
				tt.old, tt.now, start, string(del), string(ins), tt.start, tt.deleted, tt.inserted)
		}
	}
}

func TestValueOf(t *testing.T) {
	slider := func(value, min, max string) *Node {
		n := &Node{}
		n.Flags.IsSlider = true
		n.Value, n.MinValue, n.MaxValue = value, min, max
		return n
	}
	tests := []struct {
		name string
		n    *Node
		want numericValue
		ok   bool
	}{
		{"a percentage", slider("50%", "", ""), numericValue{50, 0, 100, 1}, true},
		{"a percentage ignores the limits", slider("20%", "5", "10"), numericValue{20, 0, 100, 1}, true},
		{"a number with limits", slider("3", "0", "10"), numericValue{3, 0, 10, 0.1}, true},
		{"a number without limits", slider("42", "", ""), numericValue{42, 0, 100, 1}, true},
		{"a big range has a step of one", slider("500", "0", "1000"), numericValue{500, 0, 1000, 1}, true},
		{"a value that is not a number", slider("high", "", ""), numericValue{0, 0, 100, 1}, true},
		{"an empty range", slider("1", "5", "5"), numericValue{1, 5, 5, 1}, true},
		{"a spin button", &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Role: ui.SemanticsRoleSpinButton, Value: "7"}}, numericValue{7, 0, 100, 1}, true},
		{"a progress bar", &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Role: ui.SemanticsRoleProgressBar, Value: "30%"}}, numericValue{30, 0, 100, 1}, true},
		{"a button", &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Value: "7"}}, numericValue{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := valueOf(tt.n)
			if ok != tt.ok || (ok && !reflect.DeepEqual(got, tt.want)) {
				t.Errorf("valueOf = %+v, %v; want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestHasText(t *testing.T) {
	field := &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Actions: ui.SemanticsActionSetText}}
	field.Flags.IsTextField = true
	if !hasText(field) || !hasEditableText(field) {
		t.Error("a text field is editable text")
	}
	field.Flags.IsReadOnly = true
	if !hasText(field) || hasEditableText(field) {
		t.Error("a read-only field has text only")
	}
	field.Flags.IsReadOnly = false
	field.Actions = 0
	if !hasEditableText(field) {
		t.Error("a field that has no set-text action yet (it is not focused) is still editable")
	}
	if hasText(&Node{}) {
		t.Error("a plain node has text")
	}
}
