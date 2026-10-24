package android

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

func TestParseSpellCheckTurnsUTF16RangesIntoByteOffsets(t *testing.T) {
	tests := []struct {
		name   string
		asked  string
		answer string
		want   []ui.SpellCheckSpan
	}{
		{"nothing is misspelled", "hello", "", nil},
		{"one word with its suggestions", "helo world", "0\t4\thello\thelp\n", []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"hello", "help"}}}},
		{"after an accent a unit is two bytes", "é wrld", "2\t6\tworld\n", []ui.SpellCheckSpan{{Start: 3, End: 7, Suggestions: []string{"world"}}}},
		{"after an emoji two units are four bytes", "😀 wrld", "3\t7\tworld\n", []ui.SpellCheckSpan{{Start: 5, End: 9, Suggestions: []string{"world"}}}},
		{"two ranges", "helo wrld", "0\t4\thello\n5\t9\tworld\n", []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"hello"}}, {Start: 5, End: 9, Suggestions: []string{"world"}}}},
		{"a range at the end of the text", "wrld", "0\t4\tworld\n", []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"world"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSpellCheck(tt.asked, tt.answer)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("spans = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseSpellCheckFailsWithAndroidSpellCheckInvalid(t *testing.T) {
	tests := []struct {
		name   string
		asked  string
		answer string
	}{
		{"no suggestion", "helo", "0\t4\n"},
		{"not a number", "helo", "a\tb\thello\n"},
		{"the end before the start", "helo", "3\t1\thello\n"},
		{"past the text", "helo", "0\t9\thello\n"},
		{"inside a surrogate pair", "😀x", "1\t3\thello\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSpellCheck(tt.asked, tt.answer)
			if !hasRoute(err, "parseSpellCheck."+tagAndroidSpellCheckInvalid) {
				t.Errorf("the answer failed with %v", err)
			}
		})
	}
}

type spellAnswer struct {
	spans []ui.SpellCheckSpan
	err   error
}

func TestSpellCheckAsksTheActivityAndGivesTheAnswer(t *testing.T) {
	p, host, _ := backPlatform(t)
	w := ui.SpellCheckHost(p.win)
	if w.SpellCheckSupported() {
		t.Fatal("spell check is supported before the activity said so")
	}
	p.spellCheckSupport(true)
	if !w.SpellCheckSupported() {
		t.Fatal("spell check is not supported after the activity said so")
	}
	answers := make(chan spellAnswer, 3)
	ask := func(text string) {
		w.SpellCheck(context.Background(), "en-US", text, func(s []ui.SpellCheckSpan, err error) { answers <- spellAnswer{s, err} })
	}
	ask("helo")
	ask("wrld")
	want := []command{{Op: opSpellCheck, A: 1, S: "en-US\nhelo"}, {Op: opSpellCheck, A: 2, S: "en-US\nwrld"}}
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("the commands are %+v, want %+v", got, want)
	}
	// The answers come in any order, once each; an answer for no request is dropped.
	p.spellCheckResult(2, spellCheckOK, "0\t4\tworld\n")
	p.spellCheckResult(1, spellCheckFailed, "no spell checker")
	p.spellCheckResult(1, spellCheckOK, "0\t4\thello\n")
	p.spellCheckResult(9, spellCheckOK, "")
	first, second := <-answers, <-answers
	if first.err != nil || !reflect.DeepEqual(first.spans, []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"world"}}}) {
		t.Errorf("the first answer is %+v", first)
	}
	if !hasRoute(second.err, "Platform.spellCheckResult."+tagAndroidSpellCheckFailed) {
		t.Errorf("the second answer is %+v", second)
	}
	select {
	case a := <-answers:
		t.Errorf("a third answer: %+v", a)
	default:
	}
}

func TestSpellCheckGivesUpWhenTheActivityChanges(t *testing.T) {
	p, _, _ := backPlatform(t)
	p.spellCheckSupport(true)
	answers := make(chan spellAnswer, 1)
	ui.SpellCheckHost(p.win).SpellCheck(context.Background(), "en-US", "helo", func(s []ui.SpellCheckSpan, err error) { answers <- spellAnswer{s, err} })
	p.activityCreated()
	a := <-answers
	if !hasRoute(a.err, "Platform.activityCreated->Platform.failSpellChecks."+tagAndroidSpellCheckEnded) {
		t.Errorf("the request ended with %v", a.err)
	}
	// The answer of the activity that is gone is dropped.
	p.spellCheckResult(1, spellCheckOK, "0\t4\thello\n")
	select {
	case a := <-answers:
		t.Errorf("a second answer: %+v", a)
	default:
	}
}

func TestSpellCheckIsEndedByItsContext(t *testing.T) {
	h := newHarness(t)
	h.run()
	w := h.win.(ui.SpellCheckHost)
	ctx, cancel := context.WithCancel(context.Background())
	answers := make(chan spellAnswer, 2)
	h.sync(func() {
		w.SpellCheck(ctx, "en-US", "helo", func(s []ui.SpellCheckSpan, err error) { answers <- spellAnswer{s, err} })
	})
	cancel()
	if a := <-answers; !hasTag(a.err, tagAndroidSpellCheckEnded) {
		t.Errorf("the request ended with %v", a.err)
	}
	// The answer that comes later is not given again.
	h.b.spellCheckResult(1, spellCheckOK, "0\t4\thello\n")
	h.sync(func() {})
	select {
	case a := <-answers:
		t.Errorf("a second answer: %+v", a)
	case <-time.After(50 * time.Millisecond):
	}
	// A context that has ended is told at once, without asking the activity.
	n := len(h.host.sentCommands())
	h.sync(func() {
		w.SpellCheck(ctx, "en-US", "helo", func(s []ui.SpellCheckSpan, err error) { answers <- spellAnswer{s, err} })
	})
	if a := <-answers; !hasTag(a.err, tagAndroidSpellCheckEnded) {
		t.Errorf("a request with an ended context: %v", a.err)
	}
	if len(h.host.sentCommands()) != n {
		t.Error("the activity was asked for a request that was over")
	}
}

func TestSpellCheckSupportReachesTheUIThreadThroughTheBridge(t *testing.T) {
	h := newHarness(t)
	h.run()
	w := h.win.(ui.SpellCheckHost)
	supported := func() (ok bool) {
		h.sync(func() { ok = w.SpellCheckSupported() })
		return ok
	}
	if supported() {
		t.Fatal("supported before the activity said so")
	}
	h.b.spellCheckSupport(true)
	if !supported() {
		t.Error("not supported after the activity said so")
	}
	h.b.spellCheckSupport(false)
	if supported() {
		t.Error("supported after the activity said it is not")
	}
}
