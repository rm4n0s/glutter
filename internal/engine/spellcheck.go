package engine

import (
	"context"

	"github.com/rm4n0s/glutter/ui"
)

var _ ui.SpellCheckHost = (*handler)(nil)

// spellCheckHostOf is the window of the implicit view when it has a spell checker.
func (h *handler) spellCheckHostOf() ui.SpellCheckHost {
	w := h.windowOf(ui.ImplicitViewID)
	if w == nil {
		return nil
	}
	s, _ := w.Window.(ui.SpellCheckHost)
	return s
}

// SpellCheckSupported says whether the window has a spell checker (ui.SpellCheckHost); the desktop's
// and the web's have none.
func (h *handler) SpellCheckSupported() bool {
	s := h.spellCheckHostOf()
	return s != nil && s.SpellCheckSupported()
}

// SpellCheck asks the window's spell checker for the misspelled words of the text
// (ui.SpellCheckHost).
func (h *handler) SpellCheck(ctx context.Context, locale, text string, done func([]ui.SpellCheckSpan, error)) {
	if s := h.spellCheckHostOf(); s != nil {
		s.SpellCheck(ctx, locale, text, done)
	}
}
