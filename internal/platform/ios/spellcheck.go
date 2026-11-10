package ios

import (
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the spell check of iOS (SPEC §30.6): the language that UITextChecker is asked for and the
// conversion of its answers, which count UTF-16 units, into the byte ranges of Glutter. The calls to UITextChecker are in
// spellcheck_ios.go.

// spellLanguageOf is the language of a BCP 47 tag as UITextChecker names it: "en-US" is "en_US" (the region in upper case), and a
// tag with no region is left as it is.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSpellCheckPlugin.mm findAllSpellCheckSuggestionsForText:inLanguage:
func spellLanguageOf(locale string) string {
	if !strings.Contains(locale, "-") {
		return locale
	}
	parts := strings.Split(locale, "-")
	return parts[0] + "_" + strings.ToUpper(parts[len(parts)-1])
}

// utf16Range is a range of UTF-16 units, as an NSRange: Location and Length.
type utf16Range struct{ Location, Length int }

// spellSpan is a misspelled word with its guesses, as UITextChecker gives them.
type spellSpan struct {
	Range       utf16Range
	Suggestions []string
}

// spellSpansOf turns the misspelled words into spans over the bytes of text. A range that does not fall on the characters of the
// text (UITextChecker answered for another text) is left out.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSpellCheckPlugin.mm toDictionary
func spellSpansOf(text string, found []spellSpan) []ui.SpellCheckSpan {
	out := make([]ui.SpellCheckSpan, 0, len(found))
	for _, f := range found {
		start, ok := byteOfUnit(text, f.Range.Location)
		if !ok {
			continue
		}
		end, ok := byteOfUnit(text, f.Range.Location+f.Range.Length)
		if !ok || end < start {
			continue
		}
		out = append(out, ui.SpellCheckSpan{Start: start, End: end, Suggestions: f.Suggestions})
	}
	return out
}

// byteOfUnit is the byte offset of the UTF-16 offset in text, and false when the offset is past the end or inside a character
// that is two units long.
func byteOfUnit(text string, unit int) (int, bool) {
	if unit < 0 {
		return 0, false
	}
	units := 0
	for i, r := range text {
		if units == unit {
			return i, true
		}
		if units > unit {
			return 0, false
		}
		if r >= 0x10000 {
			units += 2
		} else {
			units++
		}
	}
	if units == unit {
		return len(text), true
	}
	return 0, false
}

// textChecker is what the spell check asks of UITextChecker.
type textChecker interface {
	// Languages are the languages that the checker has a dictionary for, as it names them ("en_US").
	Languages() []string
	// Misspelled is the first misspelled word of the text that starts at or after the UTF-16 offset, and false when there is none.
	Misspelled(text, language string, from int) (utf16Range, bool)
	// Guesses are the words that the checker suggests for the misspelled range, the best first.
	Guesses(text, language string, r utf16Range) []string
}

// checkSpelling finds every misspelled word of the text in the locale, as FlutterSpellCheckPlugin does: each search goes on after the
// end of the word that was found. A language that the checker has no dictionary for is SpellCheckLanguageUnavailable.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSpellCheckPlugin.mm findAllSpellCheckSuggestionsForText:inLanguage:
func checkSpelling(c textChecker, locale, text string) ([]ui.SpellCheckSpan, error) {
	language := spellLanguageOf(locale)
	if !slices.Contains(c.Languages(), language) {
		return nil, errors.New(tagSpellCheckLanguageUnavailable, "the spell checker has no dictionary for the language", "language", language)
	}
	var found []spellSpan
	for next, total := 0, u16Len(text); next < total; {
		r, ok := c.Misspelled(text, language, next)
		if !ok {
			break
		}
		found = append(found, spellSpan{Range: r, Suggestions: c.Guesses(text, language, r)})
		end := r.Location + r.Length
		if end <= next { // a checker that does not move on would never end
			end = next + 1
		}
		next = end
	}
	return spellSpansOf(text, found), nil
}

// The tag of the error that this file makes (rm4n0s/errors, R5).
const tagSpellCheckLanguageUnavailable = "SpellCheckLanguageUnavailable"
