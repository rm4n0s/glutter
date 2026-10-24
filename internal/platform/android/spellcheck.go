package android

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// opSpellCheck is the operation of the activity's command call that asks the spell checker (SPEC
// §24.6): A is the number of the request and S the locale (a BCP 47 tag), a line feed and the text.
// The answer is Native.spellCheckResult. It is done on the main thread by SpellCheck.java.
const opSpellCheck = 24

// The codes of Native.spellCheckResult.
const (
	// spellCheckOK: the text is the misspelled ranges, one per line: the start and the end in UTF-16
	// offsets and the suggestions, separated by tabs.
	spellCheckOK = 0
	// spellCheckFailed: the text is the reason.
	spellCheckFailed = 1
)

// spellChecker is the spell checker of Android: TextServicesManager's service, asked through the
// activity. A request is a command and an event that brings the answer, which the UI thread gives to
// the function of the call, never before the call has returned.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/SpellCheckPlugin.java SpellCheckPlugin
type spellChecker struct {
	p *Platform
	// supported is whether the activity said that a spell checker can be asked (the user has not
	// turned it off); it is told again when the window gets the focus.
	supported bool
	// requests wait for the activity, by number; next is the number of the next.
	requests map[int]*spellRequest
	next     int
}

// spellRequest is a request that waits for the answer. text is what was asked, to turn the ranges
// of the answer into byte offsets; stop ends the watch on the context.
type spellRequest struct {
	text string
	done func([]ui.SpellCheckSpan, error)
	stop func() bool
}

var _ ui.SpellCheckHost = (*window)(nil)

// SpellCheckSupported reports whether a spell checker can be asked (ui.SpellCheckHost).
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.supportsSpellCheck
func (w *window) SpellCheckSupported() bool { return w.p.spell != nil && w.p.spell.supported }

// SpellCheck asks the system's spell checker for the misspelled words of the text in the locale
// (ui.SpellCheckHost). done is called later on the UI thread. A request that ctx ends before the
// activity answers fails with AndroidSpellCheckEnded, once.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/SpellCheckPlugin.java SpellCheckPlugin.initiateSpellCheck
func (w *window) SpellCheck(ctx context.Context, locale, text string, done func([]ui.SpellCheckSpan, error)) {
	if done == nil {
		return
	}
	s := w.p.spellChecker()
	if ctx.Err() != nil {
		err := errors.NewErr(tagAndroidSpellCheckEnded, ctx.Err())
		w.p.Post(func() { done(nil, err) })
		return
	}
	s.next++
	id := s.next
	// The end of the context is told to the UI thread, which gives the answer, if the activity has not.
	stop := context.AfterFunc(ctx, func() {
		w.p.Post(func() {
			if r, ok := s.requests[id]; ok {
				delete(s.requests, id)
				r.done(nil, errors.NewErr(tagAndroidSpellCheckEnded, ctx.Err()))
			}
		})
	})
	s.requests[id] = &spellRequest{text: text, done: done, stop: stop}
	w.p.host.Command(command{Op: opSpellCheck, A: int64(id), S: locale + "\n" + text})
}

// spellChecker returns the spell checker of the platform.
func (p *Platform) spellChecker() *spellChecker {
	if p.spell == nil {
		p.spell = &spellChecker{p: p, requests: map[int]*spellRequest{}}
	}
	return p.spell
}

// spellCheckSupport is the activity's report of whether a spell checker can be asked
// (Native.spellCheckSupport).
func (p *Platform) spellCheckSupport(supported bool) { p.spellChecker().supported = supported }

// spellCheckResult is the activity's answer to a request (Native.spellCheckResult), on the UI
// thread: a request that the context ended is not answered again.
func (p *Platform) spellCheckResult(id, code int, text string) {
	s := p.spellChecker()
	r, ok := s.requests[id]
	if !ok {
		return
	}
	delete(s.requests, id)
	r.stop()
	if code != spellCheckOK {
		r.done(nil, errors.New(tagAndroidSpellCheckFailed, "the spell checker failed", "reason", text))
		return
	}
	spans, err := parseSpellCheck(r.text, text)
	if err != nil {
		r.done(nil, err)
		return
	}
	r.done(spans, nil)
}

// failSpellChecks gives up the requests that wait: the activity that was asked is gone, and with it
// the session of its spell checker.
func (p *Platform) failSpellChecks() {
	if p.spell == nil {
		return
	}
	for id, r := range p.spell.requests {
		delete(p.spell.requests, id)
		r.stop()
		r.done(nil, errors.New(tagAndroidSpellCheckEnded, "the activity that was asked is gone"))
	}
}

// parseSpellCheck reads the lines of Native.spellCheckResult: "start\tend\tsuggestion\t..." with
// UTF-16 offsets into the text that was asked, which become byte offsets. A line that is not that,
// or whose range is not inside the text, fails with AndroidSpellCheckInvalid.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/SpellCheckPlugin.java SpellCheckPlugin.onGetSentenceSuggestions
func parseSpellCheck(asked, text string) ([]ui.SpellCheckSpan, error) {
	var spans []ui.SpellCheckSpan
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			return nil, errors.New(tagAndroidSpellCheckInvalid, "a range has no suggestions", "line", line)
		}
		start, err1 := strconv.Atoi(fields[0])
		end, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil || start < 0 || end < start {
			return nil, errors.New(tagAndroidSpellCheckInvalid, "a range is not two offsets", "line", line)
		}
		from, ok1 := utf16ToByte(asked, start)
		to, ok2 := utf16ToByte(asked, end)
		if !ok1 || !ok2 {
			return nil, errors.New(tagAndroidSpellCheckInvalid, "a range is not inside the text", "line", line, "length", len(asked))
		}
		spans = append(spans, ui.SpellCheckSpan{Start: from, End: to, Suggestions: fields[2:]})
	}
	return spans, nil
}

// utf16ToByte is the byte offset of the UTF-16 offset in s, and false when s is shorter or the
// offset is inside a surrogate pair.
func utf16ToByte(s string, offset int) (int, bool) {
	units := 0
	for i := 0; ; {
		if units == offset {
			return i, true
		}
		if i >= len(s) || units > offset {
			return 0, false
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		i += n
		if r >= 0x10000 {
			units += 2
		} else {
			units++
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidSpellCheckEnded   = "AndroidSpellCheckEnded"
	tagAndroidSpellCheckFailed  = "AndroidSpellCheckFailed"
	tagAndroidSpellCheckInvalid = "AndroidSpellCheckInvalid"
)
