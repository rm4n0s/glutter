package services

import (
	"context"
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// These tests port services/spell_check_test.dart, and add the cases of
// DefaultSpellCheckService, which Flutter tests through the method channel.

func TestSpellCheckToString(t *testing.T) {
	suggestionSpan := NewSuggestionSpan(ui.TextRange{Start: 12, End: 17}, []string{"weird"})
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"SuggestionSpan.toString", suggestionSpan.String(),
			"SuggestionSpan(range: TextRange(start: 12, end: 17), suggestions: [weird])"},
		{"SpellCheckResults.toString", NewSpellCheckResults("i before e except after c is so wierd.", []SuggestionSpan{suggestionSpan}).String(),
			"SpellCheckResults(spellCheckText: i before e except after c is so wierd., suggestionSpans: [SuggestionSpan(range: TextRange(start: 12, end: 17), suggestions: [weird])])"},
		{"several suggestions", NewSuggestionSpan(ui.TextRange{Start: 0, End: 3}, []string{"a", "b"}).String(),
			"SuggestionSpan(range: TextRange(start: 0, end: 3), suggestions: [a, b])"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestSpellCheckEqual(t *testing.T) {
	a := NewSuggestionSpan(ui.TextRange{Start: 1, End: 4}, []string{"x", "y"})
	tests := []struct {
		name string
		a, b SuggestionSpan
		want bool
	}{
		{"same", a, NewSuggestionSpan(ui.TextRange{Start: 1, End: 4}, []string{"x", "y"}), true},
		{"other start", a, NewSuggestionSpan(ui.TextRange{Start: 2, End: 4}, []string{"x", "y"}), false},
		{"other end", a, NewSuggestionSpan(ui.TextRange{Start: 1, End: 5}, []string{"x", "y"}), false},
		{"other suggestions", a, NewSuggestionSpan(ui.TextRange{Start: 1, End: 4}, []string{"x"}), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal = %v, want %v", got, tc.want)
			}
			ra := NewSpellCheckResults("text", []SuggestionSpan{tc.a})
			rb := NewSpellCheckResults("text", []SuggestionSpan{tc.b})
			if got := ra.Equal(rb); got != tc.want {
				t.Errorf("results Equal = %v, want %v", got, tc.want)
			}
		})
	}
	if NewSpellCheckResults("a", nil).Equal(NewSpellCheckResults("b", nil)) {
		t.Error("results of different texts are equal")
	}
}

func TestDefaultSpellCheckServiceMergeResults(t *testing.T) {
	span := func(start, end int, s ...string) SuggestionSpan {
		return NewSuggestionSpan(ui.TextRange{Start: start, End: end}, s)
	}
	tests := []struct {
		name     string
		old, new []SuggestionSpan
		want     []SuggestionSpan
	}{
		{"both empty", nil, nil, []SuggestionSpan{}},
		{"old only", []SuggestionSpan{span(0, 2, "a")}, nil, []SuggestionSpan{span(0, 2, "a")}},
		{"new only", nil, []SuggestionSpan{span(0, 2, "b")}, []SuggestionSpan{span(0, 2, "b")}},
		{"same start keeps the old", []SuggestionSpan{span(0, 2, "a")}, []SuggestionSpan{span(0, 2, "b")}, []SuggestionSpan{span(0, 2, "a")}},
		{"interleaved", []SuggestionSpan{span(0, 2, "a"), span(10, 12, "c")}, []SuggestionSpan{span(5, 7, "b"), span(20, 22, "d")},
			[]SuggestionSpan{span(0, 2, "a"), span(5, 7, "b"), span(10, 12, "c"), span(20, 22, "d")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DefaultSpellCheckServiceMergeResults(tc.old, tc.new)
			if !slices.EqualFunc(got, tc.want, SuggestionSpan.Equal) {
				t.Errorf("merged = %v, want %v", got, tc.want)
			}
		})
	}
}

// fakeSpellCheckBackend answers a request when run is called, like the engine.
type fakeSpellCheckBackend struct {
	supported bool
	spans     []ui.SpellCheckSpan
	err       error
	requests  []string
	pending   []func()
}

func (b *fakeSpellCheckBackend) SpellCheckSupported() bool { return b.supported }

func (b *fakeSpellCheckBackend) SpellCheck(_ context.Context, locale, text string, done func([]ui.SpellCheckSpan, error)) {
	b.requests = append(b.requests, locale+" "+text)
	spans, err := b.spans, b.err
	b.pending = append(b.pending, func() { done(spans, err) })
}

func (b *fakeSpellCheckBackend) run() {
	for len(b.pending) > 0 {
		fn := b.pending[0]
		b.pending = b.pending[1:]
		fn()
	}
}

type spellCheckAnswer struct {
	spans []SuggestionSpan
	ok    bool
	err   error
}

func TestDefaultSpellCheckServiceFetchSpellCheckSuggestions(t *testing.T) {
	en := ui.Locale{LanguageCode: "en", CountryCode: "US"}
	tests := []struct {
		name      string
		spans     []ui.SpellCheckSpan
		err       error
		wantOK    bool
		wantSpans []SuggestionSpan
		wantSaved bool
	}{
		{"misspelled words", []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"test"}}}, nil, true,
			[]SuggestionSpan{NewSuggestionSpan(ui.TextRange{Start: 0, End: 4}, []string{"test"})}, true},
		{"no misspelled words", nil, nil, true, []SuggestionSpan{}, true},
		{"a canceled request gives no result", nil, errors.New(tagTestSpellCheckCanceled, "canceled by a newer request"), false, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := &fakeSpellCheckBackend{supported: true, spans: tc.spans, err: tc.err}
			service := NewDefaultSpellCheckService(backend)
			var got *spellCheckAnswer
			service.FetchSpellCheckSuggestions(context.Background(), en, "tset", func(spans []SuggestionSpan, ok bool, err error) {
				got = &spellCheckAnswer{spans, ok, err}
			})
			if got != nil {
				t.Fatal("the answer arrived before the engine's")
			}
			backend.run()
			if got == nil {
				t.Fatal("no answer")
			}
			if got.err != nil || got.ok != tc.wantOK || !slices.EqualFunc(got.spans, tc.wantSpans, SuggestionSpan.Equal) {
				t.Errorf("answer = %+v, want ok %v spans %v", *got, tc.wantOK, tc.wantSpans)
			}
			if want := []string{"en-US tset"}; !slices.Equal(backend.requests, want) {
				t.Errorf("requests = %v, want %v", backend.requests, want)
			}
			if saved := service.LastSavedResults() != nil; saved != tc.wantSaved {
				t.Errorf("results saved = %v, want %v", saved, tc.wantSaved)
			}
		})
	}
}

func TestDefaultSpellCheckServiceKeepsTheLastResults(t *testing.T) {
	en := ui.Locale{LanguageCode: "en"}
	backend := &fakeSpellCheckBackend{supported: true, spans: []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"test"}}}}
	service := NewDefaultSpellCheckService(backend)
	var spans []SuggestionSpan
	fetch := func(text string) {
		service.FetchSpellCheckSuggestions(context.Background(), en, text, func(s []SuggestionSpan, _ bool, _ error) { spans = s })
		backend.run()
	}
	fetch("tset tset")
	first := service.LastSavedResults()
	if first == nil || first.SpellCheckedText != "tset tset" || len(first.SuggestionSpans) != 1 {
		t.Fatalf("saved = %v", first)
	}
	// The same text and the same spans merge into the same spans.
	fetch("tset tset")
	if len(spans) != 1 || !spans[0].Equal(first.SuggestionSpans[0]) {
		t.Errorf("merged = %v, want %v", spans, first.SuggestionSpans)
	}
	// Another text replaces the results.
	backend.spans = []ui.SpellCheckSpan{{Start: 5, End: 9, Suggestions: []string{"test"}}}
	fetch("tset tset!")
	if got := service.LastSavedResults(); got.SpellCheckedText != "tset tset!" || got.SuggestionSpans[0].Range.Start != 5 {
		t.Errorf("saved = %v", got)
	}
}

func TestDefaultSpellCheckServiceWithoutBackend(t *testing.T) {
	var got error
	NewDefaultSpellCheckService(nil).FetchSpellCheckSuggestions(context.Background(), ui.Locale{LanguageCode: "en"}, "tset", func(_ []SuggestionSpan, ok bool, err error) {
		got = err
		if ok {
			t.Error("a service without a backend gave a result")
		}
	})
	appErr, ok := errors.FromError(got)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", got)
	}
	if !appErr.HasRoute("DefaultSpellCheckService.FetchSpellCheckSuggestions.SpellCheckServiceUnavailable") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestDefaultSpellCheckServiceSupportsSpellCheck(t *testing.T) {
	tests := []struct {
		name      string
		platform  foundation.TargetPlatform
		supported bool
		nilBack   bool
		want      bool
	}{
		{"Android with a spell checker", foundation.TargetPlatformAndroid, true, false, true},
		{"Android without one", foundation.TargetPlatformAndroid, false, false, false},
		{"Android without a backend", foundation.TargetPlatformAndroid, true, true, false},
		{"Linux with a spell checker", foundation.TargetPlatformLinux, true, false, false},
		{"iOS with a spell checker", foundation.TargetPlatformIOS, true, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("needs the debug-only platform override")
			}
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(tc.platform))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			var backend SpellCheckBackend
			if !tc.nilBack {
				backend = &fakeSpellCheckBackend{supported: tc.supported}
			}
			if got := DefaultSpellCheckServiceSupportsSpellCheck(backend); got != tc.want {
				t.Errorf("supports = %v, want %v", got, tc.want)
			}
		})
	}
}

// The tags of the errors that the tests make.
const tagTestSpellCheckCanceled = "TestSpellCheckCanceled"
