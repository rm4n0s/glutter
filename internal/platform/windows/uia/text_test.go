package uia

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/ui"
)

func TestUTF16OffsetsOfTheTextOfAField(t *testing.T) {
	// "aé😀b": 1 + 1 + 2 UTF-16 units; 1 + 2 + 4 + 1 bytes.
	const s = "aé😀b"
	tests := []struct {
		name     string
		bytes    int
		wantU16  int
		backFrom int // UTF-16 offset to convert back
		wantByte int
	}{
		{"the start", 0, 0, 0, 0},
		{"after the a", 1, 1, 1, 1},
		{"after the é", 3, 2, 2, 3},
		{"in the middle of the é counts it", 2, 2, 3, 3 + 0},
		{"after the emoji", 7, 4, 4, 7},
		{"in the middle of the emoji's pair", 5, 4, 3, 3},
		{"the end", 8, 5, 5, 8},
		{"past the end", 99, 5, 99, 8},
		{"before the start", -4, 0, -4, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UTF16Offset(s, tt.bytes); got != tt.wantU16 {
				t.Errorf("UTF16Offset(%d) = %d, want %d", tt.bytes, got, tt.wantU16)
			}
			if got := ByteOffset(s, tt.backFrom); got != tt.wantByte {
				t.Errorf("ByteOffset(%d) = %d, want %d", tt.backFrom, got, tt.wantByte)
			}
		})
	}
	if got := UTF16Len(s); got != 5 {
		t.Errorf("UTF16Len = %d, want 5", got)
	}
}

func TestSelectionOfAField(t *testing.T) {
	field := semtree.N(1, "Name", 0, 0, 1, 1)
	field.Flags.IsTextField = true
	field.Value = "aé😀b"
	tests := []struct {
		name       string
		base, ext  int
		want       TextRange
		wantExists bool
	}{
		{"no selection", -1, -1, TextRange{}, false},
		{"a caret", 3, 3, TextRange{2, 2}, true},
		{"a selection", 1, 7, TextRange{1, 4}, true},
		{"a backwards selection", 7, 1, TextRange{1, 4}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := field
			f.TextSelectionBase, f.TextSelectionExtent = tt.base, tt.ext
			m := NewModel()
			m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1), f))
			got, ok := Selection(m.Node(NodeRef{1, 1}))
			if ok != tt.wantExists || got != tt.want {
				t.Errorf("selection = %v, %v; want %v, %v", got, ok, tt.want, tt.wantExists)
			}
		})
	}
}

func TestSelectRangeSetsTheSelectionInBytes(t *testing.T) {
	field := semtree.N(1, "Name", 0, 0, 1, 1)
	field.Flags.IsTextField = true
	field.Value = "aé😀b"
	field.Actions = ui.SemanticsActionSetSelection
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1), field))
	ev, st := m.SelectRange(NodeRef{1, 1}, TextRange{2, 4})
	if st != StatusOK || len(ev) != 1 {
		t.Fatalf("SelectRange = %v, %v", ev, st)
	}
	args, _ := ev[0].Arguments.(map[string]int)
	if args["base"] != 3 || args["extent"] != 7 {
		t.Errorf("arguments = %v, want base 3 and extent 7 (bytes)", ev[0].Arguments)
	}
	if ev, st := m.SelectRange(NodeRef{1, 1}, TextRange{-5, 99}); st != StatusOK || ev[0].Arguments.(map[string]int)["extent"] != 8 {
		t.Errorf("a range outside the text is cut: %v, %v", ev, st)
	}
	field.Actions = 0
	m.Apply(1, semtree.Update(field))
	if _, st := m.SelectRange(NodeRef{1, 1}, TextRange{0, 1}); st != StatusNotSupported {
		t.Errorf("without the action the status is %v, want not supported", st)
	}
}

func TestTextRangeOperations(t *testing.T) {
	const doc = "Hello big world\nSecond line"
	tx := NewText(doc)
	tests := []struct {
		name string
		got  TextRange
		want TextRange
	}{
		{"expand a caret to a character", func() TextRange { return tx.Expand(TextRange{3, 3}, TextUnitCharacter) }(), TextRange{3, 4}},
		{"expand to a word with its space", func() TextRange { return tx.Expand(TextRange{7, 7}, TextUnitWord) }(), TextRange{6, 10}},
		{"expand a range over two words", func() TextRange { return tx.Expand(TextRange{2, 8}, TextUnitWord) }(), TextRange{0, 10}},
		{"expand to a line with its break", func() TextRange { return tx.Expand(TextRange{3, 3}, TextUnitLine) }(), TextRange{0, 16}},
		{"expand to the last line", func() TextRange { return tx.Expand(TextRange{20, 20}, TextUnitLine) }(), TextRange{16, 27}},
		{"expand to the document", func() TextRange { return tx.Expand(TextRange{3, 3}, TextUnitDocument) }(), TextRange{0, 27}},
		{"a caret at the end expands to the last character", func() TextRange { return tx.Expand(TextRange{27, 27}, TextUnitCharacter) }(), TextRange{26, 27}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("= %v, want %v", tt.got, tt.want)
			}
		})
	}
	moves := []struct {
		name      string
		start     TextRange
		unit      int
		count     int
		wantRange TextRange
		wantMoved int
	}{
		{"one word forward", TextRange{0, 6}, TextUnitWord, 1, TextRange{6, 10}, 1},
		{"two words forward", TextRange{0, 6}, TextUnitWord, 2, TextRange{10, 16}, 2},
		{"back from the second word", TextRange{6, 10}, TextUnitWord, -1, TextRange{0, 6}, -1},
		{"back past the start stops", TextRange{6, 10}, TextUnitWord, -5, TextRange{0, 6}, -1},
		{"forward past the end stops", TextRange{16, 23}, TextUnitWord, 9, TextRange{23, 27}, 1},
		{"a character", TextRange{0, 1}, TextUnitCharacter, 3, TextRange{3, 4}, 3},
		{"a line", TextRange{0, 5}, TextUnitLine, 1, TextRange{16, 27}, 1},
		{"zero units", TextRange{2, 5}, TextUnitWord, 0, TextRange{2, 5}, 0},
	}
	for _, tt := range moves {
		t.Run("move "+tt.name, func(t *testing.T) {
			got, moved := tx.Move(tt.start, tt.unit, tt.count)
			if got != tt.wantRange || moved != tt.wantMoved {
				t.Errorf("Move = %v, %d; want %v, %d", got, moved, tt.wantRange, tt.wantMoved)
			}
		})
	}
	ends := []struct {
		name      string
		start     TextRange
		endpoint  int
		count     int
		wantRange TextRange
		wantMoved int
	}{
		{"the end one word forward", TextRange{0, 6}, EndpointEnd, 1, TextRange{0, 10}, 1},
		{"the start one word forward", TextRange{0, 6}, EndpointStart, 1, TextRange{6, 6}, 1},
		{"the start past the end drags the end", TextRange{0, 6}, EndpointStart, 2, TextRange{10, 10}, 2},
		{"the end back past the start drags the start", TextRange{6, 10}, EndpointEnd, -3, TextRange{0, 0}, -2},
		{"the end at the end of the text", TextRange{20, 27}, EndpointEnd, 5, TextRange{20, 27}, 0},
	}
	for _, tt := range ends {
		t.Run("endpoint "+tt.name, func(t *testing.T) {
			got, moved := tx.MoveEndpoint(tt.start, tt.endpoint, TextUnitWord, tt.count)
			if got != tt.wantRange || moved != tt.wantMoved {
				t.Errorf("MoveEndpoint = %v, %d; want %v, %d", got, moved, tt.wantRange, tt.wantMoved)
			}
		})
	}
}

func TestTextSubstringFindAndCompare(t *testing.T) {
	tx := NewText("Hello 😀 Hello")
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"all", tx.Substring(tx.Document(), -1), "Hello 😀 Hello"},
		{"cut", tx.Substring(tx.Document(), 3), "Hel"},
		{"never in the middle of a pair", tx.Substring(TextRange{0, 14}, 7), "Hello "},
		{"a sub range", tx.Substring(TextRange{6, 8}, -1), "😀"},
		{"outside", tx.Substring(TextRange{-3, 99}, -1), "Hello 😀 Hello"},
		{"empty", tx.Substring(TextRange{4, 4}, -1), ""},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
	finds := []struct {
		name       string
		needle     string
		backward   bool
		ignoreCase bool
		want       TextRange
		ok         bool
	}{
		{"first", "Hello", false, false, TextRange{0, 5}, true},
		{"last", "Hello", true, false, TextRange{9, 14}, true},
		{"case", "hello", false, false, TextRange{}, false},
		{"ignoring case", "hello", false, true, TextRange{0, 5}, true},
		{"the emoji", "😀", false, false, TextRange{6, 8}, true},
		{"empty", "", false, false, TextRange{}, false},
	}
	for _, tt := range finds {
		got, ok := tx.Find(tx.Document(), tt.needle, tt.backward, tt.ignoreCase)
		if ok != tt.ok || got != tt.want {
			t.Errorf("Find %s = %v, %v; want %v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
	a, b := TextRange{2, 5}, TextRange{4, 9}
	if CompareEndpoints(a, EndpointStart, b, EndpointStart) != -1 || CompareEndpoints(a, EndpointEnd, b, EndpointStart) != 1 ||
		CompareEndpoints(a, EndpointEnd, TextRange{0, 5}, EndpointEnd) != 0 {
		t.Error("CompareEndpoints gave the wrong order")
	}
	if got := MoveEndpointToEndpoint(a, EndpointStart, b, EndpointEnd); got != (TextRange{9, 9}) {
		t.Errorf("start to the end of the other = %v, want a caret at 9", got)
	}
	if got := MoveEndpointToEndpoint(a, EndpointEnd, b, EndpointStart); got != (TextRange{2, 4}) {
		t.Errorf("end to the start of the other = %v", got)
	}
	if got := MoveEndpointToEndpoint(a, EndpointEnd, TextRange{0, 1}, EndpointStart); got != (TextRange{0, 0}) {
		t.Errorf("end before the start drags the start: %v", got)
	}
}
