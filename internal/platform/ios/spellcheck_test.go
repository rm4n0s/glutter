package ios

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func TestSpellLanguageOf(t *testing.T) {
	tests := []struct{ in, want string }{
		{"en-US", "en_US"},
		{"en-us", "en_US"},
		{"pt-br", "pt_BR"},
		{"en", "en"},
		{"zh-Hant-TW", "zh_TW"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := spellLanguageOf(tc.in); got != tc.want {
			t.Errorf("spellLanguageOf(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSpellSpansOf(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		found []spellSpan
		want  []ui.SpellCheckSpan
	}{
		{"none", "hello world", nil, []ui.SpellCheckSpan{}},
		{"ascii", "helo wrld", []spellSpan{{utf16Range{0, 4}, []string{"hello", "help"}}, {utf16Range{5, 4}, []string{"world"}}},
			[]ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"hello", "help"}}, {Start: 5, End: 9, Suggestions: []string{"world"}}}},
		// "é" is two bytes and one unit; the emoji is four bytes and two units.
		{"after non-ascii", "é 😀 wrld", []spellSpan{{utf16Range{5, 4}, []string{"world"}}},
			[]ui.SpellCheckSpan{{Start: 8, End: 12, Suggestions: []string{"world"}}}},
		{"ends the text", "abc xyz", []spellSpan{{utf16Range{4, 3}, nil}}, []ui.SpellCheckSpan{{Start: 4, End: 7}}},
		{"inside an emoji is left out", "😀 x", []spellSpan{{utf16Range{1, 1}, nil}}, []ui.SpellCheckSpan{}},
		{"past the end is left out", "abc", []spellSpan{{utf16Range{2, 5}, nil}, {utf16Range{9, 1}, nil}}, []ui.SpellCheckSpan{}},
		{"negative is left out", "abc", []spellSpan{{utf16Range{-1, 2}, nil}}, []ui.SpellCheckSpan{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := spellSpansOf(tc.text, tc.found); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("spellSpansOf(%q) = %+v, want %+v", tc.text, got, tc.want)
			}
		})
	}
}

// fakeChecker is a textChecker over a list of misspelled words.
type fakeChecker struct {
	languages []string
	// words are the misspelled words; the checker finds them by their text, in UTF-16 units.
	words   map[string][]string
	asked   []int
	stalled bool
}

func (f *fakeChecker) Languages() []string { return f.languages }

func (f *fakeChecker) Misspelled(text, _ string, from int) (utf16Range, bool) {
	f.asked = append(f.asked, from)
	if f.stalled {
		return utf16Range{from, 0}, true
	}
	best, found := utf16Range{}, false
	for w := range f.words {
		i := strings.Index(text, w)
		for i >= 0 {
			loc := u16Len(text[:i])
			if loc >= from && (!found || loc < best.Location) {
				best, found = utf16Range{loc, u16Len(w)}, true
			}
			j := strings.Index(text[i+1:], w)
			if j < 0 {
				break
			}
			i += 1 + j
		}
	}
	return best, found
}

func (f *fakeChecker) Guesses(text, _ string, r utf16Range) []string {
	return f.words[sliceU16(text, r.Location, r.Location+r.Length)]
}

func TestCheckSpelling(t *testing.T) {
	c := &fakeChecker{languages: []string{"en_US", "fr_FR"}, words: map[string][]string{"helo": {"hello", "help"}, "wrld": {"world"}}}
	got, err := checkSpelling(c, "en-US", "é helo and 😀 wrld")
	if err != nil {
		t.Fatalf("checkSpelling: %v", err)
	}
	want := []ui.SpellCheckSpan{{Start: 3, End: 7, Suggestions: []string{"hello", "help"}}, {Start: 17, End: 21, Suggestions: []string{"world"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("spans %+v, want %+v", got, want)
	}
	// The second search starts where the first word ended (UTF-16 units: é, space, helo).
	if len(c.asked) < 2 || c.asked[0] != 0 || c.asked[1] != 6 {
		t.Errorf("the searches started at %v, want 0 then 6", c.asked)
	}
	// No misspelling is no span, and not an error.
	got, err = checkSpelling(&fakeChecker{languages: []string{"en_US"}}, "en-US", "all right")
	if err != nil || len(got) != 0 {
		t.Errorf("a correct text gave %+v, %v", got, err)
	}
}

func TestCheckSpellingLanguageUnavailable(t *testing.T) {
	_, err := checkSpelling(&fakeChecker{languages: []string{"en_US"}}, "de-DE", "hallo")
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagSpellCheckLanguageUnavailable || !appErr.HasRoute("checkSpelling."+tagSpellCheckLanguageUnavailable) {
		t.Errorf("error %v, want SpellCheckLanguageUnavailable", err)
	}
}

func TestCheckSpellingEndsWithACheckerThatDoesNotMoveOn(t *testing.T) {
	c := &fakeChecker{languages: []string{"en_US"}, stalled: true}
	if _, err := checkSpelling(c, "en-US", "abcd"); err != nil {
		t.Fatalf("checkSpelling: %v", err)
	}
	if len(c.asked) > 4 {
		t.Errorf("the checker was asked %d times for 4 units", len(c.asked))
	}
}
