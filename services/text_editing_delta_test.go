package services_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func TestTextEditingDeltaFromReplacement(t *testing.T) {
	collapsed := services.NewTextSelectionCollapsed
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	tests := []struct {
		name       string
		oldText    string
		start, end int
		text       string
		selection  services.TextSelection
		composing  ui.TextRange
		want       services.TextEditingDelta
	}{
		{"insertion at a collapsed selection", "", 0, 0, "let there be text", collapsed(17), ui.TextRangeEmpty,
			services.NewTextEditingDeltaInsertion("", "let there be text", 0, collapsed(17), ui.TextRangeEmpty)},
		{"insertion at the end of the composing region", "hello worl", 6, 10, "world", collapsed(11), rng(6, 11),
			services.NewTextEditingDeltaInsertion("hello worl", "d", 10, collapsed(11), rng(6, 11))},
		{"deletion", "let there be text.", 1, 2, "", collapsed(1), ui.TextRangeEmpty,
			services.NewTextEditingDeltaDeletion("let there be text.", rng(1, 2), collapsed(1), ui.TextRangeEmpty)},
		{"deletion at the end of the composing region", "hello world", 6, 11, "worl", collapsed(10), rng(6, 10),
			services.NewTextEditingDeltaDeletion("hello world", rng(10, 11), collapsed(10), rng(6, 10))},
		{"replacement with longer", "hello worfi", 6, 11, "working", collapsed(13), rng(6, 13),
			services.NewTextEditingDeltaReplacement("hello worfi", "working", rng(6, 11), collapsed(13), rng(6, 13))},
		{"replacement with shorter", "hello world", 6, 11, "h", collapsed(7), rng(6, 7),
			services.NewTextEditingDeltaReplacement("hello world", "h", rng(6, 11), collapsed(7), rng(6, 7))},
		{"replacement with the same length", "hello world", 6, 11, "words", collapsed(11), rng(6, 11),
			services.NewTextEditingDeltaReplacement("hello world", "words", rng(6, 11), collapsed(11), rng(6, 11))},
		{"non text update", "hello world", -1, -1, "", collapsed(10), rng(6, 11),
			services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(10), rng(6, 11))},
		{"the same text is a non text update", "hello", 1, 3, "el", collapsed(3), ui.TextRangeEmpty,
			services.NewTextEditingDeltaNonTextUpdate("hello", collapsed(3), ui.TextRangeEmpty)},
		// A deletion deletes one character, not one byte (SPEC §12.10).
		{"deletion of a character of two bytes", "héé", 0, 5, "hé", collapsed(3), rng(0, 3),
			services.NewTextEditingDeltaDeletion("héé", rng(3, 5), collapsed(3), rng(0, 3))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := services.NewTextEditingDeltaFromReplacement(tc.oldText, tc.start, tc.end, tc.text, tc.selection, tc.composing)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("delta = %#v\nwant %#v", got, tc.want)
			}
		})
	}
}

func TestTextEditingDeltaAccessors(t *testing.T) {
	ins := services.NewTextEditingDeltaInsertion("hello worl", "d", 10, services.NewTextSelectionCollapsed(11), ui.TextRangeEmpty)
	if ins.OldText() != "hello worl" || ins.TextInserted() != "d" || ins.InsertionOffset() != 10 ||
		!ins.Selection().Equal(services.NewTextSelectionCollapsed(11)) || ins.Composing() != ui.TextRangeEmpty {
		t.Errorf("insertion = %+v", ins)
	}
	del := services.NewTextEditingDeltaDeletion("let there be text.", ui.TextRange{Start: 1, End: 2}, services.NewTextSelectionCollapsed(1), ui.TextRangeEmpty)
	if del.TextDeleted() != "e" || del.DeletedRange() != (ui.TextRange{Start: 1, End: 2}) {
		t.Errorf("deletion = %+v", del)
	}
	rep := services.NewTextEditingDeltaReplacement("hello world", "h", ui.TextRange{Start: 6, End: 11}, services.NewTextSelectionCollapsed(7), ui.TextRange{Start: 6, End: 7})
	if rep.TextReplaced() != "world" || rep.ReplacementText() != "h" || rep.ReplacedRange() != (ui.TextRange{Start: 6, End: 11}) {
		t.Errorf("replacement = %+v", rep)
	}
}

func TestTextEditingDeltaApply(t *testing.T) {
	collapsed := services.NewTextSelectionCollapsed
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	// The text of the value is not used: the delta applies to its own old text.
	value := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "stale"})
	tests := []struct {
		name  string
		delta services.TextEditingDelta
		want  services.TextEditingValue
	}{
		{"insertion", services.NewTextEditingDeltaInsertion("hello worl", "d", 10, collapsed(11), rng(6, 11)),
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello world", Selection: foundation.Some(collapsed(11)), Composing: foundation.Some(rng(6, 11))})},
		{"deletion", services.NewTextEditingDeltaDeletion("hello world", rng(10, 11), collapsed(10), ui.TextRangeEmpty),
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello worl", Selection: foundation.Some(collapsed(10))})},
		{"replacement", services.NewTextEditingDeltaReplacement("hello world", "h", rng(6, 11), collapsed(7), rng(6, 7)),
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello h", Selection: foundation.Some(collapsed(7)), Composing: foundation.Some(rng(6, 7))})},
		{"non text update", services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(7), rng(6, 7)),
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello world", Selection: foundation.Some(collapsed(7)), Composing: foundation.Some(rng(6, 7))})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.delta.Apply(value); !got.Equal(tc.want) {
				t.Errorf("Apply = %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestTextEditingDeltaInvalidFailsToApply(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the checks of Flutter's assertions are for debug builds")
	}
	collapsed := services.NewTextSelectionCollapsed
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	tests := []struct {
		name  string
		delta services.TextEditingDelta
		tag   string
	}{
		{"insertion", services.NewTextEditingDeltaInsertion("hello worl", "d", 11, collapsed(11), ui.TextRangeEmpty), "TextEditingDeltaInsertionInvalid"},
		{"insertion selection", services.NewTextEditingDeltaInsertion("hello worl", "d", 10, collapsed(99), ui.TextRangeEmpty), "TextEditingDeltaInsertionInvalid"},
		{"deletion", services.NewTextEditingDeltaDeletion("hello world", rng(5, 12), collapsed(5), ui.TextRangeEmpty), "TextEditingDeltaDeletionInvalid"},
		{"replacement", services.NewTextEditingDeltaReplacement("hello worl", "world", rng(5, 11), collapsed(11), ui.TextRangeEmpty), "TextEditingDeltaReplacementInvalid"},
		{"non text update", services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(12), ui.TextRangeEmpty), "TextEditingDeltaNonTextUpdateInvalid"},
		{"non text update composing", services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(1), rng(0, 40)), "TextEditingDeltaNonTextUpdateInvalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tag := panicTag(t, func() { tc.delta.Apply(services.TextEditingValueEmpty) }); tag != tc.tag {
				t.Errorf("Apply panicked with %q, want %q", tag, tc.tag)
			}
		})
	}
}

func TestTextEditingDeltaFromReplacementRangeInvalid(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the checks of Flutter's assertions are for debug builds")
	}
	collapsed := services.NewTextSelectionCollapsed
	tests := []struct {
		name       string
		start, end int
		text       string
		sel        services.TextSelection
	}{
		{"range outside the text", 0, 99, "x", collapsed(0)},
		{"selection outside the new text", 0, 0, "x", collapsed(40)},
		{"non text update with a selection outside the text", -1, -1, "", collapsed(40)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := panicTag(t, func() {
				services.NewTextEditingDeltaFromReplacement("abc", tc.start, tc.end, tc.text, tc.sel, ui.TextRangeEmpty)
			})
			if tag != "TextEditingDeltaRangeInvalid" {
				t.Errorf("panicked with %q", tag)
			}
		})
	}
}

func TestTextEditingDeltaDebugFillProperties(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the checks of Flutter's assertions are for debug builds")
	}
	collapsed := services.NewTextSelectionCollapsed
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	tests := []struct {
		name  string
		delta services.TextEditingDelta
		want  []string
	}{
		{"insertion", services.NewTextEditingDeltaInsertion("hello worl", "d", 10, collapsed(11), ui.TextRangeEmpty), []string{
			"oldText: hello worl", "textInserted: d", "insertionOffset: 10",
			"selection: TextSelection.collapsed(offset: 11, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: -1, end: -1)"}},
		{"deletion", services.NewTextEditingDeltaDeletion("hello world", rng(6, 10), collapsed(6), ui.TextRangeEmpty), []string{
			"oldText: hello world", "textDeleted: worl", "deletedRange: TextRange(start: 6, end: 10)",
			"selection: TextSelection.collapsed(offset: 6, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: -1, end: -1)"}},
		{"replacement", services.NewTextEditingDeltaReplacement("hello world", "h", rng(6, 11), collapsed(7), rng(6, 7)), []string{
			"oldText: hello world", "textReplaced: world", "replacementText: h", "replacedRange: TextRange(start: 6, end: 11)",
			"selection: TextSelection.collapsed(offset: 7, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: 6, end: 7)"}},
		{"non text update", services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(7), rng(6, 7)), []string{
			"oldText: hello world",
			"selection: TextSelection.collapsed(offset: 7, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: 6, end: 7)"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := foundation.NewDiagnosticPropertiesBuilder()
			tc.delta.DebugFillProperties(b)
			var got []string
			for _, n := range b.Properties {
				if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
					got = append(got, n.String())
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("properties = %q\nwant %q", got, tc.want)
			}
		})
	}
}
