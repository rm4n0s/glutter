package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/editable_text_test.dart:
// "Asserts if composing text is not valid", "Preserves composing range if cursor
// moves within that range", "Clears composing range if cursor moves outside that
// range" (both cases), "TextEditingController.text set to empty string clears
// field", "TextEditingController.buildTextSpan receives build context" and
// "Composing text is underlined and underline is cleared when losing focus".

func findRenderEditable(ts *gluttertest.Tester) *rendering.RenderEditable {
	e := ts.Element(gluttertest.ByElementPredicate("a RenderEditable", func(e Element) bool {
		_, ok := e.RenderObject().(*rendering.RenderEditable)
		return ok
	}))
	return e.RenderObject().(*rendering.RenderEditable)
}

func TestTextEditingControllerAssertsIfComposingIsNotValid(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name   string
		value  services.TextEditingValue
		assert bool
	}{
		{"empty", services.NewTextEditingValue(), false},
		{"reversed", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "test", Composing: foundation.Some(ui.TextRange{Start: 1, End: 0})}), true},
		{"past the end", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "test", Composing: foundation.Some(ui.TextRange{Start: 1, End: 9})}), true},
		{"not a range", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "test", Composing: foundation.Some(ui.TextRange{Start: -1, End: 9})}), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for name, build := range map[string]func(){
				"from value": func() { NewTextEditingControllerFromValue(tc.value).Dispose() },
				"set value":  func() { c := NewTextEditingController(); defer c.Dispose(); c.SetValue(tc.value) },
			} {
				t.Run(name, func(t *testing.T) {
					if tc.assert {
						requireTag(t, "TextEditingControllerComposingInvalid", build)
					} else {
						build()
					}
				})
			}
		})
	}
}

func TestTextEditingControllerSelectionOutsideTheText(t *testing.T) {
	c := NewTextEditingController("abc")
	defer c.Dispose()
	requireTag(t, "TextEditingControllerSelectionInvalid", func() { c.SetSelection(services.NewTextSelectionCollapsed(4)) })
	requireTag(t, "TextEditingControllerSelectionInvalid", func() { c.SetSelection(services.NewTextSelection(1, 5)) })
	c.SetSelection(services.NewTextSelection(1, 3))
	if got := c.Selection(); got.BaseOffset != 1 || got.ExtentOffset != 3 {
		t.Errorf("selection = %v", got)
	}
}

func TestTextEditingControllerValues(t *testing.T) {
	composing := ui.TextRange{Start: 1, End: 3}
	withComposing := func() *TextEditingController {
		return NewTextEditingControllerFromValue(services.NewTextEditingValue(services.TextEditingValueOpts{
			Text: "abcd", Selection: foundation.Some(services.NewTextSelectionCollapsed(2)), Composing: foundation.Some(composing)}))
	}
	tests := []struct {
		name     string
		do       func(c *TextEditingController)
		text     string
		sel      services.TextSelection
		composed ui.TextRange
	}{
		{"new without a text", func(c *TextEditingController) {}, "", services.NewTextSelectionCollapsed(-1), ui.TextRangeEmpty},
		{"set text clears the selection and the composing range", func(c *TextEditingController) { c.SetText("xyz") },
			"xyz", services.NewTextSelectionCollapsed(-1), ui.TextRangeEmpty},
		{"clear", func(c *TextEditingController) { c.Clear() }, "", services.NewTextSelectionCollapsed(0), ui.TextRangeEmpty},
		{"clear the composing range", func(c *TextEditingController) { c.ClearComposing() }, "abcd", services.NewTextSelectionCollapsed(2), ui.TextRangeEmpty},
		{"selection within the composing range keeps it", func(c *TextEditingController) { c.SetSelection(services.NewTextSelectionCollapsed(1)) },
			"abcd", services.NewTextSelectionCollapsed(1), composing},
		{"selection outside of the composing range clears it", func(c *TextEditingController) { c.SetSelection(services.NewTextSelectionCollapsed(4)) },
			"abcd", services.NewTextSelectionCollapsed(4), ui.TextRangeEmpty},
		{"selection that is partly outside of the composing range clears it", func(c *TextEditingController) { c.SetSelection(services.NewTextSelection(0, 2)) },
			"abcd", services.NewTextSelection(0, 2), ui.TextRangeEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := withComposing()
			if tc.name == "new without a text" {
				c = NewTextEditingController()
			}
			defer c.Dispose()
			notified := 0
			c.AddListener(func() { notified++ })
			tc.do(c)
			v := c.Value()
			if v.Text != tc.text || !v.Selection.Equal(tc.sel) || v.Composing != tc.composed {
				t.Errorf("value = %v, want text %q selection %v composing %v", v, tc.text, tc.sel, tc.composed)
			}
			if tc.name != "new without a text" && notified != 1 {
				t.Errorf("listener called %d times", notified)
			}
		})
	}
}

func TestTextEditingControllerBuildTextSpan(t *testing.T) {
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(10.0)})
	value := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "foo composing bar",
		Selection: foundation.Some(services.NewTextSelectionCollapsed(4)), Composing: foundation.Some(ui.TextRange{Start: 4, End: 12})})
	ts := gluttertest.New(t)
	var bc BuildContext
	ts.PumpWidget(NewBuilder(func(c BuildContext) Widget { bc = c; return NewContainer(nil) }))

	t.Run("without composing", func(t *testing.T) {
		c := NewTextEditingControllerFromValue(value)
		defer c.Dispose()
		span := c.BuildTextSpan(bc, style, false)
		if span.Text.OrElse("") != "foo composing bar" || len(span.Children) != 0 || span.Style() != style {
			t.Errorf("span = %v", span)
		}
	})
	t.Run("with composing, the composing range is underlined", func(t *testing.T) {
		c := NewTextEditingControllerFromValue(value)
		defer c.Dispose()
		span := c.BuildTextSpan(bc, style, true)
		if len(span.Children) != 3 {
			t.Fatalf("children = %d, want 3", len(span.Children))
		}
		want := []string{"foo ", "composin", "g bar"}
		for i, ch := range span.Children {
			child := ch.(*painting.TextSpan)
			if child.Text.OrElse("") != want[i] {
				t.Errorf("child %d = %q, want %q", i, child.Text.OrElse(""), want[i])
			}
			underlined := child.Style() != nil && child.Style().Decoration.OrElse(ui.TextDecorationNone) == ui.TextDecorationUnderline
			if underlined != (i == 1) {
				t.Errorf("child %d underlined = %v", i, underlined)
			}
		}
		if got := span.Style(); got != style {
			t.Errorf("the span has the style %v", got)
		}
	})
	t.Run("an invalid composing range", func(t *testing.T) {
		skipUnlessDebug(t)
		c := NewTextEditingController("test")
		defer c.Dispose()
		c.ValueNotifier.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "test", Composing: foundation.Some(ui.TextRange{Start: 1, End: 9})}))
		requireTag(t, "TextEditingControllerComposingInvalid", func() { c.BuildTextSpan(bc, style, true) })
		if span := c.BuildTextSpan(bc, style, false); span.Text.OrElse("") != "test" {
			t.Errorf("span = %v", span)
		}
	})
	t.Run("a function replaces it and gets the context", func(t *testing.T) {
		c := NewTextEditingController("a")
		defer c.Dispose()
		var got BuildContext
		c.SetBuildTextSpan(func(c BuildContext, s *painting.TextStyle, withComposing bool) *painting.TextSpan {
			got = c
			return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("custom"), Style: s})
		})
		if span := c.BuildTextSpan(bc, style, true); span.Text.OrElse("") != "custom" || got != bc {
			t.Errorf("span = %v, context %v", span, got)
		}
		c.SetBuildTextSpan(nil)
		if span := c.BuildTextSpan(bc, style, true); span.Text.OrElse("") != "a" {
			t.Errorf("span = %v", span)
		}
	})
}

func TestEditableTextPreservesOrClearsTheComposingRange(t *testing.T) {
	composing := ui.TextRange{Start: 4, End: 12}
	text := "foo composing bar"
	set := func(sel services.TextSelection) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(sel), Composing: foundation.Some(composing)})
	}
	tests := []struct {
		name string
		move services.TextSelection
		want ui.TextRange
	}{
		{"a cursor within the range", services.NewTextSelectionCollapsed(5), composing},
		{"a cursor at the start of the range", services.NewTextSelectionCollapsed(4), composing},
		{"a cursor before the range", services.NewTextSelectionCollapsed(2), ui.TextRangeEmpty},
		{"a cursor after the range", services.NewTextSelectionCollapsed(14), ui.TextRangeEmpty},
		{"a selection before the range", services.NewTextSelection(1, 2), ui.TextRangeEmpty},
		{"a selection within the range", services.NewTextSelection(5, 8), composing},
		{"a selection across the start of the range", services.NewTextSelection(2, 6), ui.TextRangeEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newEditableFixture(t, "", collapsed(0))
			state := gluttertest.StateOf[*EditableTextState](f.ts, gluttertest.ByType[*EditableText]())
			state.UpdateEditingValue(set(services.NewTextSelectionCollapsed(4)))
			f.ts.Pump()
			if v, _ := state.CurrentTextEditingValue(); v.Composing != composing {
				t.Fatalf("composing = %v, want %v", v.Composing, composing)
			}
			f.controller.SetSelection(tc.move)
			if v, _ := state.CurrentTextEditingValue(); v.Composing != tc.want {
				t.Errorf("composing = %v, want %v", v.Composing, tc.want)
			}
		})
	}
}

func TestEditableTextShowsAndUnderlinesTheComposingText(t *testing.T) {
	f := newEditableFixture(t, "", collapsed(0))
	state := gluttertest.StateOf[*EditableTextState](f.ts, gluttertest.ByType[*EditableText]())
	f.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "foo composing bar",
		Selection: foundation.Some(services.NewTextSelectionCollapsed(4)), Composing: foundation.Some(ui.TextRange{Start: 4, End: 12})}))
	f.ts.Pump()
	state.RequestKeyboard()
	f.ts.Pump()
	span := findRenderEditable(f.ts).Text().(*painting.TextSpan)
	if len(span.Children) != 3 || span.Children[1].(*painting.TextSpan).Style().Decoration.OrElse(ui.TextDecorationNone) != ui.TextDecorationUnderline {
		t.Fatalf("the composing text is not underlined: %v", span)
	}
	// The underline is cleared when the field loses focus.
	f.focus.Unfocus()
	f.ts.Pump()
	span = findRenderEditable(f.ts).Text().(*painting.TextSpan)
	if len(span.Children) != 0 {
		t.Errorf("the text is still marked as composing after the focus was lost: %v", span)
	}
}

func TestTextEditingControllerSetToEmptyTextClearsTheField(t *testing.T) {
	f := newEditableFixture(t, "", collapsed(0))
	f.controller.SetText("...")
	f.ts.Pump()
	if got := findRenderEditable(f.ts).PlainText(); got != "..." {
		t.Errorf("text = %q", got)
	}
	f.controller.SetText("")
	f.ts.Pump()
	if got := findRenderEditable(f.ts).PlainText(); got != "" {
		t.Errorf("text = %q", got)
	}
}

func TestEditableTextBuildsTheSpanWithTheContextOfTheController(t *testing.T) {
	f := newEditableFixture(t, "a", collapsed(0))
	color := ui.NewColor(0xFF010203)
	f.controller.SetBuildTextSpan(func(bc BuildContext, s *painting.TextStyle, withComposing bool) *painting.TextSpan {
		if bc == nil {
			t.Error("the context is nil")
		}
		return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("a"), Style: s.CopyWith(painting.TextStyleCopy{Color: foundation.Some(color)})})
	})
	f.ts.PumpWidget(f.build())
	span := findRenderEditable(f.ts).Text().(*painting.TextSpan)
	if got := span.Style().Color; got.OrElse(ui.Color{}) != color {
		t.Errorf("color = %v, want %v", got, color)
	}
}

func TestEditableTextShowsThePreeditOfTheInputMethod(t *testing.T) {
	f := newEditableFixture(t, "ab", collapsed(2))
	f.ts.Pump()
	input := f.ts.Binding.TextInput()
	// The input method composes "ni": the preedit is in the text, underlined.
	input.HandleTextInput(ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2})
	f.ts.Pump()
	if got := f.controller.Text(); got != "abni" {
		t.Fatalf("text = %q, want the preedit in it", got)
	}
	if got := f.controller.Value().Composing; got != (ui.TextRange{Start: 2, End: 4}) {
		t.Fatalf("composing = %v", got)
	}
	span := findRenderEditable(f.ts).Text().(*painting.TextSpan)
	if len(span.Children) != 3 || span.Children[1].(*painting.TextSpan).Style().Decoration.OrElse(ui.TextDecorationNone) != ui.TextDecorationUnderline {
		t.Errorf("the preedit is not underlined: %v", span)
	}
	// It is replaced by what the input method commits.
	input.HandleTextInput(ui.TextInputEvent{Commit: "你", PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	f.ts.Pump()
	if got := f.controller.Text(); got != "ab你" {
		t.Errorf("text = %q, want the commit", got)
	}
	if got := f.controller.Value().Composing; got.IsValid() {
		t.Errorf("composing = %v after the commit", got)
	}
	if got := f.controller.Selection(); got.BaseOffset != len("ab你") {
		t.Errorf("selection = %v, want the caret after the commit", got)
	}
	span = findRenderEditable(f.ts).Text().(*painting.TextSpan)
	if len(span.Children) != 0 {
		t.Errorf("the committed text is still marked: %v", span)
	}
}

func TestEditableTextTypesTheKeysTheShortcutsDidNotTake(t *testing.T) {
	f := newEditableFixture(t, "ab", collapsed(2))
	f.ts.SendKeyEvent(services.LogicalKeyboardKeyKeyC, gluttertest.KeyOpts{Character: "c"})
	f.ts.Pump()
	f.expect(t, "abc", collapsed(3))
	// Enter in a single line field does not add a line.
	f.ts.SendKeyEvent(services.LogicalKeyboardKeyEnter, gluttertest.KeyOpts{Character: "\r"})
	f.ts.Pump()
	f.expect(t, "abc", collapsed(3))
}
