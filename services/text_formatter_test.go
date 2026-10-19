package services_test

import (
	"regexp"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// quoted is the pattern that matches the string, as Dart's Pattern of a String.
func quoted(s string) *regexp.Regexp { return regexp.MustCompile(regexp.QuoteMeta(s)) }

func val(text string, sel services.TextSelection) services.TextEditingValue {
	return services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(sel)})
}

func TestTextInputFormatterWithFunction(t *testing.T) {
	var calledOld, calledNew services.TextEditingValue
	f := services.NewTextInputFormatterWithFunction(func(o, n services.TextEditingValue) services.TextEditingValue {
		calledOld, calledNew = o, n
		return services.TextEditingValueEmpty
	})
	old, nw := val("a", services.NewTextSelectionCollapsed(1)), val("ab", services.NewTextSelectionCollapsed(2))
	f.FormatEditUpdate(old, nw)
	if !calledOld.Equal(old) || !calledNew.Equal(nw) {
		t.Errorf("the function got %v and %v", calledOld, calledNew)
	}
}

func TestProvidedFormatters(t *testing.T) {
	// a1b(2c3
	// d4)e5f6
	// where the parentheses are the selection range.
	testNewValue := val("a1b2c3\nd4e5f6", services.NewTextSelection(3, 9))
	old := services.TextEditingValueEmpty
	sel := services.NewTextSelection
	intoTheWoods := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Into the Woods"})
	selectedIntoTheWoods := val("Into the Woods", sel(11, 14))
	re := regexp.MustCompile
	star := services.FilteringTextInputFormatterOpts{ReplacementString: "*"}
	tests := []struct {
		name      string
		formatter services.TextInputFormatter
		in        services.TextEditingValue
		want      services.TextEditingValue
	}{
		{"filtering example, allow o", services.NewFilteringTextInputFormatterAllow(quoted("o"), star), intoTheWoods,
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "*o*oo*"})},
		{"filtering example, deny o", services.NewFilteringTextInputFormatterDeny(quoted("o"), star), intoTheWoods,
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Int* the W**ds"})},
		{"filtering example, allow o+", services.NewFilteringTextInputFormatterAllow(re("o+"), star), intoTheWoods,
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "*o*oo*"})},
		{"filtering example, deny o+", services.NewFilteringTextInputFormatterDeny(re("o+"), star), intoTheWoods,
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Int* the W*ds"})},
		{"filtering example with a selection, allow o", services.NewFilteringTextInputFormatterAllow(quoted("o"), star), selectedIntoTheWoods,
			val("*o*oo*", sel(4, 6))},
		{"filtering example with a selection, deny o", services.NewFilteringTextInputFormatterDeny(quoted("o"), star), selectedIntoTheWoods,
			val("Int* the W**ds", sel(11, 14))},
		{"filtering example with a selection, allow o+", services.NewFilteringTextInputFormatterAllow(re("o+"), star), selectedIntoTheWoods,
			val("*o*oo*", sel(4, 6))},
		{"filtering example with a selection, deny o+", services.NewFilteringTextInputFormatterDeny(re("o+"), star), selectedIntoTheWoods,
			val("Int* the W*ds", sel(11, 13))},
		{"deny mode", services.NewFilteringTextInputFormatterDeny(re(`[a-z]`)), testNewValue, val("123\n456", sel(1, 5))},
		{"single line formatter", services.FilteringTextInputFormatterSingleLine, testNewValue, val("a1b2c3d4e5f6", sel(3, 8))},
		{"allow mode", services.NewFilteringTextInputFormatterAllow(re(`[a-c]`)), testNewValue, val("abc", sel(2, 3))},
		{"digits only", services.FilteringTextInputFormatterDigitsOnly, testNewValue, val("123456", sel(1, 4))},
		{"length limiting", services.NewLengthLimitingTextInputFormatter(foundation.Some(6)), testNewValue, val("a1b2c3", sel(3, 6))},
		{"length limiting, zero-length string", services.NewLengthLimitingTextInputFormatter(foundation.Some(1)),
			val("", sel(0, 0)), val("", sel(0, 0))},
		// The offsets are in bytes: the unicorn face is 4 bytes (SPEC §12.10).
		{"length limiting, non-BMP scalar values", services.NewLengthLimitingTextInputFormatter(foundation.Some(2)),
			val("\U0001f984\U0001f984\U0001f984\U0001f984", services.NewTextSelectionCollapsed(16)),
			val("\U0001f984\U0001f984", services.NewTextSelectionCollapsed(8))},
		{"length limiting, a unicorn and a space", services.NewLengthLimitingTextInputFormatter(foundation.Some(1)),
			val("\U0001F984 ", services.NewTextSelectionCollapsed(4)), val("\U0001F984", services.NewTextSelectionCollapsed(4))},
		{"length limiting, X and Y", services.NewLengthLimitingTextInputFormatter(foundation.Some(1)),
			val("XY", services.NewTextSelectionCollapsed(1)), val("X", services.NewTextSelectionCollapsed(1))},
		{"length limiting, selection off the end", services.NewLengthLimitingTextInputFormatter(foundation.Some(2)),
			testNewValue, val("a1", sel(2, 2))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.formatter.FormatEditUpdate(old, tc.in)
			if !got.Equal(tc.want) {
				t.Errorf("value = %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestLengthLimitingTruncate(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"removes characters from the end", "01234567890", "0123456789"},
		// 😆 is a surrogate pair in Dart, and one character here.
		{"counts a non-BMP character as one", "😆01234567890", "😆012345678"},
		{"counts a grapheme cluster as one", "👨‍👩‍👦01234567890", "👨‍👩‍👦012345678"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := val(tc.text, services.NewTextSelectionCollapsed(len(tc.text)))
			got := services.LengthLimitingTextInputFormatterTruncate(v, 10)
			if got.Text != tc.want {
				t.Fatalf("text = %q, want %q", got.Text, tc.want)
			}
			// The caret that was at the end is at the end of the new string.
			if got.Selection.BaseOffset != len(tc.want) || got.Selection.ExtentOffset != len(tc.want) {
				t.Errorf("selection = %v, want the end (%d)", got.Selection, len(tc.want))
			}
		})
	}
}

func TestLengthLimitingTruncateKeepsTheComposingRange(t *testing.T) {
	v := services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "abcdefghij", Selection: foundation.Some(services.NewTextSelection(2, 9)),
		Composing: foundation.Some(ui.TextRange{Start: 3, End: 9}),
	})
	got := services.LengthLimitingTextInputFormatterTruncate(v, 5)
	if got.Text != "abcde" || got.Composing != (ui.TextRange{Start: 3, End: 5}) || got.Selection.End() != 5 {
		t.Errorf("value = %v", got)
	}
	v.Composing = ui.TextRange{Start: 7, End: 9}
	if got := services.LengthLimitingTextInputFormatterTruncate(v, 5); got.Composing != ui.TextRangeEmpty {
		t.Errorf("a composing range past the end is %v, want empty", got.Composing)
	}
}

func TestLengthLimitingFormatEditUpdate(t *testing.T) {
	const maxLength = 10
	text := func(s string) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: s})
	}
	tests := []struct {
		name     string
		old, new string
		want     string
	}{
		{"passes through when under the limit", "aaa", "aaab", "aaab"},
		{"uses the old value when at the limit", "aaaaaaaaaa", "aaaaabbbbbaaaaa", "aaaaaaaaaa"},
		{"truncates the new value when the old one is over the limit", "aaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbb", "bbbbbbbbbb"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := services.NewLengthLimitingTextInputFormatter(foundation.Some(maxLength))
			if got := f.FormatEditUpdate(text(tc.old), text(tc.new)); got.Text != tc.want {
				t.Errorf("text = %q, want %q", got.Text, tc.want)
			}
		})
	}
}

func TestLengthLimitingEnforcement(t *testing.T) {
	text := func(s string, composing ui.TextRange, sel services.TextSelection) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: s, Composing: foundation.Some(composing), Selection: foundation.Some(sel)})
	}
	caret := services.NewTextSelectionCollapsed
	old := text("aaa", ui.TextRangeEmpty, caret(3))
	composing := text("aaabbbb", ui.TextRange{Start: 3, End: 7}, caret(7))
	tests := []struct {
		name        string
		enforcement services.MaxLengthEnforcement
		old, new    services.TextEditingValue
		want        string
	}{
		{"none lets the text grow", services.MaxLengthEnforcementNone, old, composing, "aaabbbb"},
		{"enforced truncates the composing text", services.MaxLengthEnforcementEnforced, old, composing, "aaabb"},
		{"truncate after the composition keeps the composing text", services.MaxLengthEnforcementTruncateAfterCompositionEnds, old, composing, "aaabbbb"},
		{"truncate after the composition cuts the finished text", services.MaxLengthEnforcementTruncateAfterCompositionEnds, old,
			text("aaabbbb", ui.TextRangeEmpty, caret(7)), "aaabb"},
		{"enforced keeps the old value at the limit with a caret", services.MaxLengthEnforcementEnforced,
			text("aaaaa", ui.TextRangeEmpty, caret(2)), text("aaaaab", ui.TextRangeEmpty, caret(3)), "aaaaa"},
		{"enforced truncates when there is a selection at the limit", services.MaxLengthEnforcementEnforced,
			text("aaaaa", ui.TextRangeEmpty, services.NewTextSelection(0, 5)), text("bbbbbb", ui.TextRangeEmpty, caret(6)), "bbbbb"},
		{"truncate after the composition keeps the old value at the limit", services.MaxLengthEnforcementTruncateAfterCompositionEnds,
			text("aaaaa", ui.TextRangeEmpty, caret(5)), text("aaaaab", ui.TextRangeEmpty, caret(6)), "aaaaa"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := services.NewLengthLimitingTextInputFormatter(foundation.Some(5), services.LengthLimitingTextInputFormatterOpts{MaxLengthEnforcement: tc.enforcement})
			if got := f.FormatEditUpdate(tc.old, tc.new); got.Text != tc.want {
				t.Errorf("text = %q, want %q", got.Text, tc.want)
			}
		})
	}
}

func TestLengthLimitingNoLimit(t *testing.T) {
	long := val("this is a long text", services.NewTextSelectionCollapsed(0))
	for _, n := range []foundation.Opt[int]{foundation.None[int](), foundation.Some(-1)} {
		f := services.NewLengthLimitingTextInputFormatter(n)
		if got := f.FormatEditUpdate(services.TextEditingValueEmpty, long); got.Text != long.Text {
			t.Errorf("limit %v cut the text to %q", n, got.Text)
		}
	}
}

func TestLengthLimitingInvalidLimit(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the checks of Flutter's assertions are for debug builds")
	}
	if tag := panicTag(t, func() { services.NewLengthLimitingTextInputFormatter(foundation.Some(0)) }); tag != "MaxLengthInvalid" {
		t.Errorf("limit 0 panicked with %q", tag)
	}
	if tag := panicTag(t, func() { services.NewLengthLimitingTextInputFormatter(foundation.Some(-2)) }); tag != "MaxLengthInvalid" {
		t.Errorf("limit -2 panicked with %q", tag)
	}
}

func TestDefaultMaxLengthEnforcement(t *testing.T) {
	tests := []struct {
		platform foundation.TargetPlatform
		want     services.MaxLengthEnforcement
	}{
		{foundation.TargetPlatformWindows, services.MaxLengthEnforcementEnforced},
		{foundation.TargetPlatformAndroid, services.MaxLengthEnforcementEnforced},
		{foundation.TargetPlatformMacOS, services.MaxLengthEnforcementTruncateAfterCompositionEnds},
		{foundation.TargetPlatformIOS, services.MaxLengthEnforcementTruncateAfterCompositionEnds},
		{foundation.TargetPlatformLinux, services.MaxLengthEnforcementTruncateAfterCompositionEnds},
		{foundation.TargetPlatformFuchsia, services.MaxLengthEnforcementTruncateAfterCompositionEnds},
	}
	for _, tc := range tests {
		t.Run(tc.platform.Name(), func(t *testing.T) {
			if got := services.LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(tc.platform); got != tc.want {
				t.Errorf("enforcement = %v, want %v", got, tc.want)
			}
		})
	}
	// With no platform, the default target platform (Linux).
	if got := services.LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(); got != services.MaxLengthEnforcementTruncateAfterCompositionEnds {
		t.Errorf("default enforcement = %v", got)
	}
}

func TestFilteringReturnsTheOldTextForANonAllowedCharacter(t *testing.T) {
	f := services.FilteringTextInputFormatterDigitsOnly
	old := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "12345"})
	nw := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "12345@"})
	if got := f.FormatEditUpdate(old, nw).Text; got != "12345" {
		t.Errorf("text = %q, want 12345", got)
	}
}

func TestFilteringMovesTheCursorToTheRightPosition(t *testing.T) {
	f := services.FilteringTextInputFormatterDigitsOnly
	collapsed := func(text string, offset int) services.TextEditingValue {
		return val(text, services.NewTextSelectionCollapsed(offset))
	}
	old, nw := collapsed("123", 0), collapsed("123456", 6)
	if got := f.FormatEditUpdate(old, nw).Selection.BaseOffset; got != 6 {
		t.Errorf("cursor = %d, want the end", got)
	}
	// Move the cursor into the middle of the text and then add the number 9.
	old = nw.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(services.NewTextSelectionCollapsed(4))})
	nw = old.CopyWith(services.TextEditingValueCopy{Text: foundation.Some("1239456")})
	if got := f.FormatEditUpdate(old, nw).Selection.BaseOffset; got != 4 {
		t.Errorf("cursor = %d, want 4 (right after the 9)", got)
	}
}

func TestFilteringIsIndependentOfSelectionAndComposing(t *testing.T) {
	f := services.NewFilteringTextInputFormatterDeny(quoted("abc"), services.FilteringTextInputFormatterOpts{ReplacementString: "*"})
	nw := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "abcabcabc"})
	want := f.FormatEditUpdate(services.TextEditingValueEmpty, nw).Text
	for i := range len(nw.Text) {
		withSel := nw.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(services.NewTextSelectionCollapsed(i))})
		if got := f.FormatEditUpdate(services.TextEditingValueEmpty, withSel).Text; got != want {
			t.Errorf("selection %d: text = %q, want %q", i, got, want)
		}
		withComposing := nw.CopyWith(services.TextEditingValueCopy{Composing: foundation.Some(ui.TextRange{Start: i, End: i})})
		if got := f.FormatEditUpdate(services.TextEditingValueEmpty, withComposing).Text; got != want {
			t.Errorf("composing %d: text = %q, want %q", i, got, want)
		}
	}
}

func TestFilteringBasicFiltering(t *testing.T) {
	f := services.NewFilteringTextInputFormatterAllow(regexp.MustCompile(`[A-Za-z0-9.@-]*`))
	nw := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "ab&&ca@bcabc"})
	if got := f.FormatEditUpdate(services.TextEditingValueEmpty, nw).Text; got != "abca@bcabc" {
		t.Errorf("text = %q", got)
	}
}

func TestFilteringRegions(t *testing.T) {
	nw := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "AAABBBCCC"})
	withSel := func(b, e int) services.TextEditingValue {
		return nw.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(services.NewTextSelection(b, e))})
	}
	withComposing := func(s, e int) services.TextEditingValue {
		return nw.CopyWith(services.TextEditingValueCopy{Composing: foundation.Some(ui.TextRange{Start: s, End: e})})
	}
	deny := func(p, repl string) services.TextInputFormatter {
		return services.NewFilteringTextInputFormatterDeny(quoted(p), services.FilteringTextInputFormatterOpts{ReplacementString: repl})
	}
	allow := func(p, repl string) services.TextInputFormatter {
		return services.NewFilteringTextInputFormatterAllow(quoted(p), services.FilteringTextInputFormatterOpts{ReplacementString: repl})
	}
	empty := services.TextEditingValueEmpty
	selections := []struct {
		name string
		f    services.TextInputFormatter
		in   services.TextEditingValue
		want services.TextSelection
	}{
		// AAA | BBB | CCC => AAA | **** | CCC
		{"deny, the selection is the banned text", deny("BBB", "****"), withSel(6, 3), services.NewTextSelection(7, 3)},
		// AAA | BBB CCC | => AAA | **** CCC |
		{"deny, the selection ends after", deny("BBB", "****"), withSel(9, 3), services.NewTextSelection(10, 3)},
		// AAA BBB | CCC | => AAA **** | CCC |
		{"deny, the selection starts after", deny("BBB", "****"), withSel(9, 6), services.NewTextSelection(10, 7)},
		// Same length replacement, keep the selection at where it is.
		{"deny, same length", deny("BBB", "***"), withSel(5, 4), services.NewTextSelection(5, 4)},
		// AAA | BBB | CCC => AAA | CCC
		{"deny, removed", deny("BBB", ""), withSel(6, 3), services.NewTextSelection(3, 3)},
		// The unfortunate case, we don't know for sure where to put the selection
		// so put it after the replacement string.
		{"deny, the selection is inside the banned text", deny("BBB", "****"), withSel(5, 4), services.NewTextSelection(7, 7)},
		// AAA | BBB | CCC => **** | BBB | ****
		{"allow, the selection is the allowed text", allow("BBB", "****"), withSel(6, 3), services.NewTextSelection(7, 4)},
		// | AAABBBCCC | => | ****BBB**** |
		{"allow, the selection is everything", allow("BBB", "****"), withSel(9, 0), services.NewTextSelection(11, 0)},
		// AAABBB | CCC | => ****BBB | **** |
		{"allow, the selection is the end", allow("BBB", "****"), withSel(9, 6), services.NewTextSelection(11, 7)},
		// Overlapping matches: AAA | BBBBB | CCC => | BBB |
		{"allow, overlapping matches", allow("BBB", ""),
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "AAABBBBBCCC", Selection: foundation.Some(services.NewTextSelection(8, 3))}),
			services.NewTextSelection(3, 0)},
	}
	for _, tc := range selections {
		t.Run("selection/"+tc.name, func(t *testing.T) {
			if got := tc.f.FormatEditUpdate(empty, tc.in).Selection; !got.Equal(tc.want) {
				t.Errorf("selection = %v, want %v", got, tc.want)
			}
		})
	}
	composing := []struct {
		name string
		f    services.TextInputFormatter
		in   services.TextEditingValue
		want ui.TextRange
	}{
		{"deny, the composing text is the banned text", deny("BBB", "****"), withComposing(3, 6), ui.TextRange{Start: 3, End: 7}},
		{"deny, the composing text ends after", deny("BBB", "****"), withComposing(3, 9), ui.TextRange{Start: 3, End: 10}},
		{"deny, the composing text starts after", deny("BBB", "****"), withComposing(6, 9), ui.TextRange{Start: 7, End: 10}},
		{"deny, same length", deny("BBB", "***"), withComposing(4, 5), ui.TextRange{Start: 4, End: 5}},
		{"deny, removed", deny("BBB", ""), withComposing(3, 6), ui.TextRangeEmpty},
	}
	for _, tc := range composing {
		t.Run("composing/"+tc.name, func(t *testing.T) {
			if got := tc.f.FormatEditUpdate(empty, tc.in).Composing; got != tc.want {
				t.Errorf("composing = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilteringKeepsTheAffinityAndDirectional(t *testing.T) {
	in := services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "a1b", Selection: foundation.Some(services.NewTextSelectionCollapsed(1, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream})),
	})
	got := services.FilteringTextInputFormatterDigitsOnly.FormatEditUpdate(services.TextEditingValueEmpty, in)
	if !got.Selection.Affinity.IsUpstream() {
		t.Errorf("selection = %v, want the upstream affinity kept", got.Selection)
	}
}
