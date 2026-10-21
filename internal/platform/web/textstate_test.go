package web

import "testing"

func TestUTF16Conversions(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		byteO int
		u16   int
	}{
		{"empty", "", 0, 0},
		{"ascii", "hello", 3, 3},
		{"two byte character", "é!", 2, 1},
		{"three byte character", "你好", 3, 1},
		{"surrogate pair counts two", "a😀b", 5, 3},
		{"end of a surrogate pair", "😀", 4, 2},
		{"end of the text", "a😀b", 6, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := utf16Offset(tc.text, tc.byteO); got != tc.u16 {
				t.Errorf("utf16Offset(%q, %d) = %d, want %d", tc.text, tc.byteO, got, tc.u16)
			}
			if got := byteOffset(tc.text, tc.u16); got != tc.byteO {
				t.Errorf("byteOffset(%q, %d) = %d, want %d", tc.text, tc.u16, got, tc.byteO)
			}
			if tc.u16 > 0 && utf16Len(tc.text) < tc.u16 {
				t.Errorf("utf16Len(%q) = %d, shorter than %d", tc.text, utf16Len(tc.text), tc.u16)
			}
		})
	}
	// Offsets out of the text and in the middle of a character move to a character.
	edge := []struct {
		name string
		got  int
		want int
	}{
		{"negative byte offset", utf16Offset("abc", -2), 0},
		{"byte offset past the end", utf16Offset("abc", 9), 3},
		{"byte offset inside a character", utf16Offset("é", 1), 1},
		{"negative utf16 offset", byteOffset("abc", -1), 0},
		{"utf16 offset past the end", byteOffset("abc", 9), 3},
		{"utf16 offset inside a surrogate pair", byteOffset("😀", 1), 0},
		{"rune start of a continuation byte", alignRuneStart("é", 1), 0},
	}
	for _, tc := range edge {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
	if got := sliceUTF16("a😀bc", 1, 3); got != "😀" {
		t.Errorf("sliceUTF16 = %q, want the emoji", got)
	}
	if got := sliceUTF16("abc", 2, 1); got != "" {
		t.Errorf("sliceUTF16 with the end before the start = %q, want empty", got)
	}
}

// TestEditingStatePortedFromFlutter has the cases of the EditingState group of text_editing_test.dart
// that do not need a DOM element: 'Fix flipped base and extent offsets', 'Sets default composing offsets
// if none given', 'Correctly identifies min and max offsets', 'Get Editing State from input element' and
// 'from text area element' (a DOM element gives its value, selectionStart, selectionEnd and
// selectionDirection), the three 'comparing editing states' and the negative offsets that are handled.
func TestEditingStatePortedFromFlutter(t *testing.T) {
	t.Run("flipped base and extent offsets are equal", func(t *testing.T) {
		if !NewEditingState("", 10, 4).Equal(NewEditingState("", 4, 10)) {
			t.Error("a flipped selection is a different state")
		}
	})
	t.Run("default composing offsets", func(t *testing.T) {
		s := NewEditingState("Test", 2, 4)
		if s.ComposingBase != -1 || s.ComposingExtent != -1 {
			t.Errorf("composing = %d..%d, want -1..-1", s.ComposingBase, s.ComposingExtent)
		}
	})
	t.Run("min and max offsets", func(t *testing.T) {
		flipped, normal := NewEditingState("", 10, 4), NewEditingState("", 2, 6)
		if flipped.MinOffset() != 4 || flipped.MaxOffset() != 10 || normal.MinOffset() != 2 || normal.MaxOffset() != 6 {
			t.Errorf("min/max = %d/%d and %d/%d", flipped.MinOffset(), flipped.MaxOffset(), normal.MinOffset(), normal.MaxOffset())
		}
	})
	t.Run("negative offsets become 0", func(t *testing.T) {
		s := NewEditingState("xyz", -1, -1)
		if s.Base != 0 || s.Extent != 0 || !s.IsValid() {
			t.Errorf("state = %+v, want the offsets 0", s)
		}
	})
	// The element 'Get Editing State from input element' and '... text area element' read: the value
	// 'Test' with the selection (1, 2), forward and backward.
	for _, tag := range []string{"input", "textarea"} {
		t.Run("read from "+tag, func(t *testing.T) {
			fwd := EditingStateFromDOM("Test", 1, 2, "forward")
			if fwd.Text != "Test" || fwd.Base != 1 || fwd.Extent != 2 {
				t.Errorf("forward = %+v, want base 1 extent 2", fwd)
			}
			back := EditingStateFromDOM("Test", 1, 2, "backward")
			if back.Base != 2 || back.Extent != 1 {
				t.Errorf("backward = %+v, want base 2 extent 1", back)
			}
			if none := EditingStateFromDOM("Test", 1, 2, "none"); none.Base != 1 || none.Extent != 2 {
				t.Errorf("a selection without a direction = %+v, want base 1 extent 2", none)
			}
		})
	}
	t.Run("comparing states from the dom", func(t *testing.T) {
		s1 := EditingStateFromDOM("Test", 1, 2, "forward")
		s2 := EditingStateFromDOM("Test", 1, 2, "forward")
		s3 := EditingStateFromDOM("Test", 1, 3, "forward")
		if !s1.Equal(s2) || s1.Equal(s3) {
			t.Error("equal states differ or different states are equal")
		}
	})
	t.Run("comparing takes the composition range into account", func(t *testing.T) {
		s1 := NewEditingState("", 0, 0).WithComposing(1, 2)
		s2 := NewEditingState("", 0, 0).WithComposing(1, 2)
		s3 := NewEditingState("", 0, 0).WithComposing(4, 8)
		if !s1.Equal(s2) || s1.Equal(s3) {
			t.Error("the composing range is not compared")
		}
	})
}

// compositionState is what the DOM holds in a case of composition_test.dart: text and caret.
func compositionState(text string, base, extent int) EditingState {
	return NewEditingState(text, base, extent)
}

// TestCompositionPortedFromFlutter has the cases of composition_test.dart for CompositionAwareMixin: the
// three event handlers and the nine 'determine composition state' cases.
func TestCompositionPortedFromFlutter(t *testing.T) {
	const fake = "ImComposingText"
	t.Run("composition end resets the composing text", func(t *testing.T) {
		c := Composition{}
		c.Update(fake)
		c.End()
		if c.Active() {
			t.Error("still composing after compositionend")
		}
	})
	t.Run("composition start resets the composing text", func(t *testing.T) {
		c := Composition{}
		c.Update(fake)
		c.Start()
		if c.Active() {
			t.Error("still composing after compositionstart")
		}
	})
	t.Run("composition update sets the composing text", func(t *testing.T) {
		c := Composition{}
		c.Update(fake)
		c.Update("IAmComposingThis")
		if !c.Active() || c.Text() != "IAmComposingThis" {
			t.Errorf("composing %v %q, want the text of the event", c.Active(), c.Text())
		}
	})

	tests := []struct {
		name  string
		state EditingState
		text  string // the composing text; "" with active false means none
		none  bool
		// want is the composing range of the result, -1 for none.
		wantBase, wantExtent int
	}{
		{"extent offset smaller than the composing text begins before the text", compositionState("Test", 0, 0), "Test", false, -1, -1},
		{"composing text is null", compositionState("Test", 0, 4), "", true, -1, -1},
		{"text is empty", compositionState("", 0, 0), "Test", false, -1, -1},
		{"extent offset is smaller than the composing text length", compositionState("Test", 0, 4), "composeMe", false, -1, -1},
		{"compositing the middle of the text", compositionState("Testing", 7, 7), "Test", false, 3, 7},
		{"compositing from the beginning of the text", compositionState("今日は", 0, 3), "今日は", false, 0, 3},
	}
	for _, tc := range tests {
		t.Run("determine: "+tc.name, func(t *testing.T) {
			c := Composition{}
			if !tc.none {
				c.Update(tc.text)
			}
			got := c.Determine(tc.state)
			want := tc.state.WithComposing(tc.wantBase, tc.wantExtent)
			if !got.Equal(want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
	t.Run("determine: the composing base is kept while the text changes", func(t *testing.T) {
		c := Composition{}
		c.Update("今日は寒い日です")
		state := compositionState("今日は寒い日です", 0, 8)
		for _, sel := range [][2]int{{0, 8}, {0, 3}, {3, 6}} {
			state.Base, state.Extent = sel[0], sel[1]
			got := c.Determine(state)
			if got.ComposingBase != 0 || got.ComposingExtent != 8 {
				t.Errorf("selection %v: composing %d..%d, want 0..8", sel, got.ComposingBase, got.ComposingExtent)
			}
		}
	})
}
