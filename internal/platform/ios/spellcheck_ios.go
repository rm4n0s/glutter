//go:build ios

package ios

import (
	"context"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

var _ ui.SpellCheckHost = (*window)(nil)

// nsRange is NSRange: Location and Length in UTF-16 units. NSNotFound is the largest NSUInteger.
type nsRange struct{ Location, Length uint64 }

const nsNotFound = ^uint64(0)

// uiTextChecker is textChecker over UITextChecker. The object is expensive to make, so the platform makes it at the first check and keeps
// it (Flutter does the same).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSpellCheckPlugin.mm FlutterSpellCheckPlugin
type uiTextChecker struct{ obj objc.ID }

var _ textChecker = uiTextChecker{}

// Languages are UITextChecker.availableLanguages.
func (uiTextChecker) Languages() []string {
	list := send[objc.ID](class("UITextChecker"), "availableLanguages")
	var out []string
	if list == 0 {
		return out
	}
	for i := 0; i < objcx.ArrayCount(list); i++ {
		out = append(out, objcx.String(objcx.ArrayAt(list, i)))
	}
	return out
}

// Misspelled is the next misspelled word at or after the UTF-16 offset.
func (c uiTextChecker) Misspelled(text, language string, from int) (utf16Range, bool) {
	t, l := objcx.NewString(text), objcx.NewString(language)
	defer objcx.Release(t)
	defer objcx.Release(l)
	r := send[nsRange](c.obj, "rangeOfMisspelledWordInString:range:startingAt:wrap:language:",
		t, nsRange{0, uint64(u16Len(text))}, int64(from), false, l)
	if r.Location == nsNotFound {
		return utf16Range{}, false
	}
	return utf16Range{int(r.Location), int(r.Length)}, true
}

// Guesses are the words that UITextChecker suggests for the range; it gives an empty array when there are none.
func (c uiTextChecker) Guesses(text, language string, r utf16Range) []string {
	t, l := objcx.NewString(text), objcx.NewString(language)
	defer objcx.Release(t)
	defer objcx.Release(l)
	list := send[objc.ID](c.obj, "guessesForWordRange:inString:language:", nsRange{uint64(r.Location), uint64(r.Length)}, t, l)
	var out []string
	if list == 0 {
		return out
	}
	for i := 0; i < objcx.ArrayCount(list); i++ {
		out = append(out, objcx.String(objcx.ArrayAt(list, i)))
	}
	return out
}

// textChecker is the platform's UITextChecker, made at the first use.
func (p *Platform) textChecker() uiTextChecker {
	if p.checker == 0 {
		p.checker = class("UITextChecker").Send(sel("alloc")).Send(sel("init"))
	}
	return uiTextChecker{p.checker}
}

// SpellCheckSupported says that the platform has a spell checker: UITextChecker is there on every iOS.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.supportsSpellCheck
func (w *window) SpellCheckSupported() bool { return true }

// SpellCheck finds the misspelled words of the text with UITextChecker, on the UI thread (the check of a field's text is short), and
// gives the spans to done after the call has returned (the context is not used: the check is made at once). A locale that UITextChecker has no dictionary for is
// SpellCheckLanguageUnavailable.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSpellCheckPlugin.mm handleMethodCall: SpellCheck.initiateSpellCheck
func (w *window) SpellCheck(_ context.Context, locale, text string, done func([]ui.SpellCheckSpan, error)) {
	if done == nil {
		return
	}
	var spans []ui.SpellCheckSpan
	var err error
	objcx.WithPool(func() { spans, err = checkSpelling(w.p.textChecker(), locale, text) })
	w.p.Post(func() { done(spans, err) })
}
