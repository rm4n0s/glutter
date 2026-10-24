package ui

import (
	"context"

	"github.com/rm4n0s/errors"
)

// SpellCheckSpan is a misspelled range of the text that was checked, with the words the spell
// checker suggests for it. Start and End are UTF-8 byte offsets into the text.
//
// Deviation: Flutter's SuggestionSpan has the UTF-16 range of a TextRange; Glutter's text offsets
// are bytes (SPEC §12.10), and the platform converts the ranges it gets.
//
// Flutter: services/spell_check.dart SuggestionSpan
type SpellCheckSpan struct {
	// Start is the byte offset of the first byte of the misspelled range.
	Start int
	// End is the byte offset after the last byte of the misspelled range.
	End int
	// Suggestions are the replacements, the best first.
	Suggestions []string
}

// SpellCheckHost is what the engine implements to give access to the spell checker of the
// platform: Android's TextServicesManager (SPEC §24.6). The dispatcher uses it when the host has
// it; a host without it has no spell check, as Linux and the web do not.
//
// Flutter: no counterpart (the engine's side of the flutter/spellcheck channel)
type SpellCheckHost interface {
	// SpellCheckSupported says whether the platform has a spell checker that can be asked.
	SpellCheckSupported() bool
	// SpellCheck asks for the misspelled words of the text in the locale (a BCP 47 tag, such as
	// "en-US"). done is called later on the UI thread with the misspelled ranges in order, or the
	// error. It may be called from any goroutine's work: the result is posted to the UI thread.
	SpellCheck(ctx context.Context, locale, text string, done func(spans []SpellCheckSpan, err error))
}

// SpellCheckSupported reports whether the engine has a spell checker.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.supportsSpellCheck
func (d *PlatformDispatcher) SpellCheckSupported() bool {
	if h, ok := d.host.(SpellCheckHost); ok {
		return h.SpellCheckSupported()
	}
	return false
}

// SpellCheck asks the engine to check the spelling of the text in the locale. done is called
// later on the UI thread with the misspelled ranges, or with the error: a host without a spell
// checker fails with SpellCheckUnavailable.
//
// Deviation: Flutter sends it as the SpellCheck.initiateSpellCheck call of the flutter/spellcheck
// channel (SPEC §12.10).
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.fetchSpellCheckSuggestions
func (d *PlatformDispatcher) SpellCheck(ctx context.Context, locale, text string, done func(spans []SpellCheckSpan, err error)) {
	if h, ok := d.host.(SpellCheckHost); ok && h.SpellCheckSupported() {
		h.SpellCheck(ctx, locale, text, done)
		return
	}
	err := errors.New(tagSpellCheckUnavailable, "the engine has no spell checker")
	d.Post(func() { done(nil, err) })
}

const tagSpellCheckUnavailable = "SpellCheckUnavailable"
