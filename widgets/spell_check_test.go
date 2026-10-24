package widgets_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests port widgets/spell_check_test.dart and the group 'Spell check' of
// widgets/editable_text_test.dart. The offsets of the spans are UTF-8 bytes, not
// UTF-16 code units (SPEC §12.10). The cases of the group that use the toolbar of
// Material or Cupertino are in material/spell_check_suggestions_toolbar_test.go.

var (
	// composingStyle is the style of the composing region.
	composingStyle = painting.NewTextStyle(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline)})
	// misspelledTextStyle is the style of Android, as the Dart tests use.
	misspelledTextStyle = painting.NewTextStyle(painting.TextStyleOpts{
		Decoration:      foundation.Some(ui.TextDecorationUnderline),
		DecorationColor: foundation.Some(ui.NewColor(0xFFFF0000)),
		DecorationStyle: ui.TextDecorationStyleWavy,
	})
)

func spanOf(start, end int, suggestions ...string) services.SuggestionSpan {
	return services.NewSuggestionSpan(ui.TextRange{Start: start, End: end}, suggestions)
}

// treeOf is the tree that the Dart tests expect: a TextSpan whose children are
// TextSpans of a text and, for the marked ones, a style.
func treeOf(children ...*painting.TextSpan) *painting.TextSpan {
	spans := make([]painting.InlineSpan, len(children))
	for i, c := range children {
		spans[i] = c
	}
	return painting.NewTextSpan(painting.TextSpanOpts{Children: spans})
}

func textOf(text string, style *painting.TextStyle) *painting.TextSpan {
	return painting.NewTextSpan(painting.TextSpanOpts{Style: style, Text: foundation.Some(text)})
}

func TestBuildTextSpanWithSpellCheckSuggestions(t *testing.T) {
	android := []foundation.TargetPlatform{foundation.TargetPlatformAndroid}
	androidAndIOS := []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS}
	tests := []struct {
		name                      string
		platforms                 []foundation.TargetPlatform
		value                     services.TextEditingValue
		composingRegionOutOfRange bool
		resultsText               string
		spans                     []services.SuggestionSpan
		want                      *painting.TextSpan
	}{
		{
			name:      "ignores composing region when composing region out of range",
			platforms: androidAndIOS,
			value:     services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! Hey"}), composingRegionOutOfRange: true,
			resultsText: "Hello, wrold! Hey",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old")},
			want:        treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! Hey", nil)),
		},
		{
			name:      "isolated misspelled word with separate composing region example",
			platforms: android,
			value: services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: "Hello, wrold! Hey", Composing: foundation.Some(ui.TextRange{Start: 14, End: 17})}),
			resultsText: "Hello, wrold! Hey",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old")},
			want:        treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! ", nil), textOf("Hey", composingStyle)),
		},
		{
			name:      "composing region and misspelled words overlap example",
			platforms: android,
			value: services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: "Right worng worng right", Composing: foundation.Some(ui.TextRange{Start: 12, End: 17})}),
			resultsText: "Right worng worng right",
			spans:       []services.SuggestionSpan{spanOf(6, 11, "wrong", "worn", "wrung"), spanOf(12, 17, "wrong", "worn", "wrung")},
			want: treeOf(textOf("Right ", nil), textOf("worng", misspelledTextStyle), textOf(" ", nil),
				textOf("worng", composingStyle), textOf(" right", nil)),
		},
		{
			name:      "consecutive misspelled words example",
			platforms: androidAndIOS,
			value:     services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Right worng worng right"}), composingRegionOutOfRange: true,
			resultsText: "Right worng worng right",
			spans:       []services.SuggestionSpan{spanOf(6, 11, "wrong", "worn", "wrung"), spanOf(12, 17, "wrong", "worn", "wrung")},
			want: treeOf(textOf("Right ", nil), textOf("worng", misspelledTextStyle), textOf(" ", nil),
				textOf("worng", misspelledTextStyle), textOf(" right", nil)),
		},
		{
			name:        "corrects results when they lag, results text shorter than actual text example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! Hey"}),
			resultsText: "Hello, wrold!",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old")},
			want:        treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! Hey", nil)),
		},
		{
			name:        "corrects results when they lag, results text longer with more misspelled words than actual text example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! Hey"}),
			resultsText: "Hello, wrold Hey feirnd!",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old"), spanOf(17, 23, "friend", "fiend", "fern")},
			want:        treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! Hey", nil)),
		},
		{
			name:        "corrects results when they lag, results text mismatched example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! Hey"}),
			resultsText: "Hello, wrild! Hey",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "wild", "world")},
			want:        treeOf(textOf("Hello, wrold! Hey", nil)),
		},
		{
			name:        "corrects results when they lag, results shifted forward example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, there wrold! Hey"}),
			resultsText: "Hello, wrold! Hey",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old")},
			want:        treeOf(textOf("Hello, there ", nil), textOf("wrold", misspelledTextStyle), textOf("! Hey", nil)),
		},
		{
			name:        "corrects results when they lag, results shifted backwards example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! Hey"}),
			resultsText: "Hello, great wrold! Hey",
			spans:       []services.SuggestionSpan{spanOf(13, 18, "world", "word", "old")},
			want:        treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! Hey", nil)),
		},
		{
			name:        "corrects results when they lag, results shifted backwards and forwards example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold! And Hye!"}),
			resultsText: "Hello, great wrold! Hye!",
			spans:       []services.SuggestionSpan{spanOf(13, 18, "world", "word", "old"), spanOf(20, 23, "Hey", "He")},
			want: treeOf(textOf("Hello, ", nil), textOf("wrold", misspelledTextStyle), textOf("! And ", nil),
				textOf("Hye", misspelledTextStyle), textOf("!", nil)),
		},
		{
			name:        "discards result when additions are made to misspelled word example",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wroldd!"}),
			resultsText: "Hello, wrold!",
			spans:       []services.SuggestionSpan{spanOf(7, 12, "world", "word", "old")},
			want:        treeOf(textOf("Hello, wroldd!", nil)),
		},
		{
			// Regression test for https://github.com/flutter/flutter/issues/136032.
			// "ã" is two bytes, so the span 7..12 of UTF-16 is 7..13 of bytes.
			name:        "does not throw when text contains regex reserved characters",
			platforms:   androidAndIOS,
			value:       services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, *ãresaa"}),
			resultsText: "Hello, *ãresa",
			spans:       []services.SuggestionSpan{spanOf(7, 13, "*rangesa")},
			want:        treeOf(textOf("Hello, *ãresaa", nil)),
		},
		{
			// Regression test for https://github.com/flutter/flutter/issues/152272.
			name:      "does not throw when text is being deleted",
			platforms: androidAndIOS,
			value: services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: "Lorem ipsum dolor ", Selection: foundation.Some(services.NewTextSelectionCollapsed(len("Lorem ipsum dolor ")))}),
			resultsText: "Lorem ipsum dolor sit ",
			spans: []services.SuggestionSpan{spanOf(0, 5, "suggestion1", "suggestion2"), spanOf(6, 11, "suggestion1", "suggestion2"),
				spanOf(12, 17, "suggestion1", "suggestion2"), spanOf(18, 21, "suggestion1", "suggestion2")},
			want: treeOf(textOf("Lorem", misspelledTextStyle), textOf(" ", nil), textOf("ipsum", misspelledTextStyle), textOf(" ", nil),
				textOf("dolor", misspelledTextStyle), textOf(" ", nil)),
		},
	}
	for _, tc := range tests {
		for _, platform := range tc.platforms {
			t.Run(tc.name+"/"+platform.String(), func(t *testing.T) {
				usePlatform(t, platform)
				// The value of the Dart tests says that the composing region is in range
				// unless the case says so.
				results := services.NewSpellCheckResults(tc.resultsText, tc.spans)
				got := BuildTextSpanWithSpellCheckSuggestions(tc.value, tc.composingRegionOutOfRange, nil, misspelledTextStyle, results)
				if !got.Equal(tc.want) {
					t.Errorf("tree =\n%v\nwant\n%v", got, tc.want)
				}
			})
		}
	}
}

func TestBuildTextSpanWithSpellCheckSuggestionsKeepsTheStyle(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(20.0)})
	// Without the composing region, the word with the cursor is not marked.
	tests := []struct {
		name   string
		cursor int
		marked bool
	}{
		{"cursor outside the word", 0, true},
		{"cursor in the word", 9, false},
		{"cursor at the end of the word", 12, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Hello, wrold!", Selection: foundation.Some(services.NewTextSelectionCollapsed(tc.cursor))})
			results := services.NewSpellCheckResults("Hello, wrold!", []services.SuggestionSpan{spanOf(7, 12, "world")})
			got := BuildTextSpanWithSpellCheckSuggestions(value, true, style, misspelledTextStyle, results)
			if !got.Style().Equal(style) {
				t.Errorf("the style of the tree = %v, want %v", got.Style(), style)
			}
			if len(got.Children) != 3 {
				t.Fatalf("children = %d, want 3", len(got.Children))
			}
			word := got.Children[1].(*painting.TextSpan)
			wantStyle := style
			if tc.marked {
				wantStyle = style.Merge(misspelledTextStyle)
			}
			if !word.Style().Equal(wantStyle) {
				t.Errorf("the style of the word = %v, want %v", word.Style(), wantStyle)
			}
		})
	}
}

func TestSpellCheckConfiguration(t *testing.T) {
	service := &mockSpellCheckService{}
	other := &mockSpellCheckService{}
	builder := func(BuildContext, *EditableTextState) Widget { return nil }
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(9.0)})
	enabled := NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: style, SpellCheckSuggestionsToolbarBuilder: builder})

	t.Run("enabled and disabled", func(t *testing.T) {
		if !NewSpellCheckConfiguration().SpellCheckEnabled() {
			t.Error("a configuration is disabled")
		}
		if NewSpellCheckConfigurationDisabled().SpellCheckEnabled() {
			t.Error("the disabled configuration is enabled")
		}
	})
	t.Run("Equal", func(t *testing.T) {
		tests := []struct {
			name string
			a, b *SpellCheckConfiguration
			want bool
		}{
			{"nil and nil", nil, nil, true},
			{"nil and a configuration", nil, enabled, false},
			{"disabled and disabled", NewSpellCheckConfigurationDisabled(), NewSpellCheckConfigurationDisabled(), true},
			{"disabled and enabled", NewSpellCheckConfigurationDisabled(), NewSpellCheckConfiguration(), false},
			{"same", enabled, NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: style, SpellCheckSuggestionsToolbarBuilder: builder}), true},
			{"other service", enabled, NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: other, MisspelledTextStyle: style, SpellCheckSuggestionsToolbarBuilder: builder}), false},
			{"other style", enabled, NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, SpellCheckSuggestionsToolbarBuilder: builder}), false},
			{"no builder", enabled, NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: style}), false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := tc.a.Equal(tc.b); got != tc.want {
					t.Errorf("Equal = %v, want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("CopyWith", func(t *testing.T) {
		color := foundation.Some(ui.NewColor(0xFF00FF00))
		got := enabled.CopyWith(SpellCheckConfigurationCopy{SpellCheckService: other, MisspelledSelectionColor: color})
		if got.SpellCheckService() != services.SpellCheckService(other) || got.MisspelledSelectionColor() != color ||
			!got.MisspelledTextStyle().Equal(style) || got.SpellCheckSuggestionsToolbarBuilder() == nil || !got.SpellCheckEnabled() {
			t.Errorf("copy = %v", got)
		}
		// A disabled configuration stays disabled.
		if got := NewSpellCheckConfigurationDisabled().CopyWith(SpellCheckConfigurationCopy{SpellCheckService: other}); got.SpellCheckEnabled() {
			t.Error("a copy of the disabled configuration is enabled")
		}
	})
}

// mockSpellCheckService is the FakeSpellCheckService of the Dart tests: it
// answers with the spans it was given for a text, and none for another text. The
// answer comes when flush is called, as a service's does, never during the call.
type mockSpellCheckService struct {
	suggestionSpansByText map[string][]services.SuggestionSpan
	// cancel makes the service answer that the request was canceled (Dart's null).
	cancel             bool
	fetchCallCount     int
	lastSpellCheckText string
	lastLocale         ui.Locale
	pending            []func()
}

func (m *mockSpellCheckService) FetchSpellCheckSuggestions(_ context.Context, locale ui.Locale, text string, done services.SpellCheckSuggestionsCallback) {
	m.fetchCallCount++
	m.lastSpellCheckText = text
	m.lastLocale = locale
	spans, ok := m.suggestionSpansByText[text]
	if !ok {
		spans = []services.SuggestionSpan{}
	}
	cancel := m.cancel
	m.pending = append(m.pending, func() { done(spans, !cancel, nil) })
}

// flush delivers the answers.
func (m *mockSpellCheckService) flush() {
	for len(m.pending) > 0 {
		fn := m.pending[0]
		m.pending = m.pending[1:]
		fn()
	}
}

// spellCheckApp pumps an EditableText with the text and the options.
func spellCheckApp(ts *gluttertest.Tester, controller *TextEditingController, focus *FocusNode, o EditableTextOpts) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		ui.NewColor(0xFF000000), ui.NewColor(0xFF888888), o), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
}

func spellCheckStateOf(ts *gluttertest.Tester) *EditableTextState {
	return gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
}

func TestEditableTextSpellCheckConfiguration(t *testing.T) {
	underline := painting.NewTextStyle(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline)})
	withService := func(s services.SpellCheckService) *SpellCheckConfiguration {
		return NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: s, MisspelledTextStyle: underline})
	}
	tests := []struct {
		name        string
		opts        EditableTextOpts
		wantEnabled bool
	}{
		{"disabled by default", EditableTextOpts{}, false},
		{"disabled manually", EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfigurationDisabled()}, false},
		{"enabled with a service", EditableTextOpts{SpellCheckConfiguration: withService(&mockSpellCheckService{})}, true},
		{"disabled when obscureText is true", EditableTextOpts{ObscureText: true, SpellCheckConfiguration: withService(&mockSpellCheckService{})}, false},
		{"disabled for visible password input type", EditableTextOpts{
			KeyboardType: foundation.Some(services.TextInputTypeVisiblePassword), SpellCheckConfiguration: withService(&mockSpellCheckService{})}, false},
		{"disabled for password autofill hints", EditableTextOpts{
			AutofillHints: []string{services.AutofillHintPassword}, SpellCheckConfiguration: withService(&mockSpellCheckService{})}, false},
		{"disabled for new password autofill hints", EditableTextOpts{
			AutofillHints: []string{services.AutofillHintNewPassword}, SpellCheckConfiguration: withService(&mockSpellCheckService{})}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			spellCheckApp(ts, NewTextEditingController("A"), NewFocusNode(), tc.opts)
			state := spellCheckStateOf(ts)
			if state.SpellCheckEnabled() != tc.wantEnabled {
				t.Errorf("SpellCheckEnabled = %v, want %v", state.SpellCheckEnabled(), tc.wantEnabled)
			}
			if !tc.wantEnabled && !state.SpellCheckConfiguration().Equal(NewSpellCheckConfigurationDisabled()) {
				t.Errorf("configuration = %v, want the disabled one", state.SpellCheckConfiguration())
			}
		})
	}
}

func TestEditableTextSpellCheckWithServiceOfTheConfiguration(t *testing.T) {
	service := &mockSpellCheckService{}
	ts := gluttertest.New(t)
	spellCheckApp(ts, NewTextEditingController("A"), NewFocusNode(), EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration(
		SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: misspelledTextStyle})})
	if got := spellCheckStateOf(ts).SpellCheckConfiguration().SpellCheckService(); got != services.SpellCheckService(service) {
		t.Errorf("service = %T, want the mock", got)
	}
}

func TestEditableTextSpellCheckMisspelledTextStyleMissing(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func()
		want  bool
	}{
		{"an enabled configuration without a style", func() {
			NewEditableText(NewTextEditingController("A"), NewFocusNode(), painting.NewTextStyle(), ui.NewColor(0xFF0000FF), ui.NewColor(0xFF808080),
				EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration()})
		}, true},
		{"the disabled configuration", func() {
			NewEditableText(NewTextEditingController("A"), NewFocusNode(), painting.NewTextStyle(), ui.NewColor(0xFF0000FF), ui.NewColor(0xFF808080),
				EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfigurationDisabled()})
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := panicOf(tc.build)
			if !tc.want {
				if v != nil {
					t.Fatalf("panicked with %v", v)
				}
				return
			}
			if v == nil {
				t.Fatal("no panic")
			}
			_ = requireErrorTagFromPanic(t, v, "NewEditableText.EditableTextMisspelledTextStyleMissing")
		})
	}
}

func TestEditableTextSpellCheckWithTheDefaultService(t *testing.T) {
	tests := []struct {
		name         string
		platform     foundation.TargetPlatform
		spellChecker bool
		wantEnabled  bool
	}{
		{"Android with a spell checker uses the default service", foundation.TargetPlatformAndroid, true, true},
		{"Android without a spell checker is disabled and reports it", foundation.TargetPlatformAndroid, false, false},
		{"Linux with a spell checker is disabled and reports it", foundation.TargetPlatformLinux, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			usePlatform(t, tc.platform)
			var opts gluttertest.Opts
			if tc.spellChecker {
				opts.SpellChecker = func(string, string) []ui.SpellCheckSpan { return nil }
			}
			ts := gluttertest.New(t, opts)
			spellCheckApp(ts, NewTextEditingController("A"), NewFocusNode(), EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration(
				SpellCheckConfigurationOpts{MisspelledTextStyle: misspelledTextStyle})})
			state := spellCheckStateOf(ts)
			if state.SpellCheckEnabled() != tc.wantEnabled {
				t.Fatalf("SpellCheckEnabled = %v, want %v", state.SpellCheckEnabled(), tc.wantEnabled)
			}
			if tc.wantEnabled {
				if _, ok := state.SpellCheckConfiguration().SpellCheckService().(*services.DefaultSpellCheckService); !ok {
					t.Errorf("service = %T, want *DefaultSpellCheckService", state.SpellCheckConfiguration().SpellCheckService())
				}
				return
			}
			// Spell check was asked for and cannot be done: the framework says so.
			skipUnlessDebug(t)
			err := ts.TakeError()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("EditableTextState.inferSpellCheckConfiguration.EditableTextSpellCheckUnsupported") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if !state.SpellCheckConfiguration().Equal(NewSpellCheckConfigurationDisabled()) {
				t.Errorf("configuration = %v, want the disabled one", state.SpellCheckConfiguration())
			}
		})
	}
}

func TestEditableTextSpellCheckUpdatesWhenObscureTextChanges(t *testing.T) {
	suggestionSpans := []services.SuggestionSpan{spanOf(0, 1, "a")}
	newService := func() *mockSpellCheckService {
		return &mockSpellCheckService{suggestionSpansByText: map[string][]services.SuggestionSpan{"A": suggestionSpans}}
	}
	tests := []struct {
		name         string
		keyboard     foundation.Opt[services.TextInputType]
		wantAfter    bool
		wantFetching bool
	}{
		{"non-password text", foundation.None[services.TextInputType](), true, true},
		{"a visible password stays disabled", foundation.Some(services.TextInputTypeVisiblePassword), false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := newService()
			config := NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: misspelledTextStyle})
			controller := NewTextEditingController("A")
			focus := NewFocusNode()
			ts := gluttertest.New(t)
			pump := func(obscure bool) {
				spellCheckApp(ts, controller, focus, EditableTextOpts{ObscureText: obscure, KeyboardType: tc.keyboard, SpellCheckConfiguration: config})
			}

			pump(false)
			state := spellCheckStateOf(ts)
			if state.SpellCheckEnabled() != tc.wantAfter {
				t.Fatalf("SpellCheckEnabled = %v, want %v", state.SpellCheckEnabled(), tc.wantAfter)
			}
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "A", SuggestionSpans: suggestionSpans})

			pump(true)
			state = spellCheckStateOf(ts)
			if state.SpellCheckEnabled() || !state.SpellCheckConfiguration().Equal(NewSpellCheckConfigurationDisabled()) {
				t.Errorf("spell check is enabled for obscured text")
			}
			if state.SpellCheckResults() != nil {
				t.Errorf("results = %v, want none", state.SpellCheckResults())
			}
			if service.fetchCallCount != 0 {
				t.Errorf("fetch count = %d, want 0", service.fetchCallCount)
			}

			pump(false)
			service.flush()
			state = spellCheckStateOf(ts)
			if state.SpellCheckEnabled() != tc.wantAfter {
				t.Fatalf("SpellCheckEnabled = %v, want %v", state.SpellCheckEnabled(), tc.wantAfter)
			}
			if !tc.wantFetching {
				if service.fetchCallCount != 0 || !state.SpellCheckConfiguration().Equal(NewSpellCheckConfigurationDisabled()) {
					t.Errorf("fetch count = %d, configuration = %v; want none and the disabled one", service.fetchCallCount, state.SpellCheckConfiguration())
				}
				return
			}
			if service.fetchCallCount != 1 || service.lastSpellCheckText != "A" {
				t.Errorf("fetch count = %d, last text = %q; want 1 and A", service.fetchCallCount, service.lastSpellCheckText)
			}
			want := services.NewSpellCheckResults("A", suggestionSpans)
			if r := state.SpellCheckResults(); r == nil || !r.Equal(want) {
				t.Errorf("results = %v, want %v", r, want)
			}
		})
	}
}

func TestEditableTextFindSuggestionSpanAtCursorIndex(t *testing.T) {
	expected := spanOf(20, 23, "Hey", "He")
	spans := []services.SuggestionSpan{spanOf(13, 18, "world", "word", "old"), expected, spanOf(25, 30, "green", "grey", "great")}
	tests := []struct {
		name        string
		results     *services.SpellCheckResults
		cursorIndex int
		want        bool
	}{
		{"cursor in the middle of a word", &services.SpellCheckResults{SuggestionSpans: spans}, 21, true},
		{"cursor on the edge of a word", &services.SpellCheckResults{SuggestionSpans: spans}, 23, true},
		{"cursor on the start of a word", &services.SpellCheckResults{SuggestionSpans: spans}, 20, true},
		{"cursor out of range of the spans", &services.SpellCheckResults{SuggestionSpans: spans}, 33, false},
		{"word correctly spelled", &services.SpellCheckResults{SuggestionSpans: spans}, 5, false},
		{"between two words", &services.SpellCheckResults{SuggestionSpans: spans}, 24, false},
		{"no results", nil, 21, false},
		{"no spans", &services.SpellCheckResults{}, 21, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			usePlatform(t, foundation.TargetPlatformAndroid)
			ts := gluttertest.New(t, gluttertest.Opts{SpellChecker: func(string, string) []ui.SpellCheckSpan { return nil }})
			spellCheckApp(ts, NewTextEditingController("A"), NewFocusNode(), EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration(
				SpellCheckConfigurationOpts{MisspelledTextStyle: misspelledTextStyle})})
			state := spellCheckStateOf(ts)
			state.SetSpellCheckResults(tc.results)
			got, ok := state.FindSuggestionSpanAtCursorIndex(tc.cursorIndex)
			if ok != tc.want || (ok && !got.Equal(expected)) {
				t.Errorf("FindSuggestionSpanAtCursorIndex(%d) = %v, %v; want %v", tc.cursorIndex, got, ok, tc.want)
			}
		})
	}
}

// isMisspelled says whether the span has the style of misspelled words, merged
// into the style of the field.
func isMisspelled(span painting.InlineSpan) bool {
	style := span.(*painting.TextSpan).Style()
	return style != nil && style.Decoration.OrElse(ui.TextDecorationNone) == ui.TextDecorationUnderline &&
		style.DecorationStyle == ui.TextDecorationStyleWavy
}

func TestEditableTextShowsMisspelledWords(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformAndroid)
	service := &mockSpellCheckService{suggestionSpansByText: map[string][]services.SuggestionSpan{
		"Hello, wrold!":  {spanOf(7, 12, "world")},
		"Hello, wrold!!": {spanOf(7, 12, "world")},
	}}
	controller := NewTextEditingController("")
	focus := NewFocusNode()
	ts := gluttertest.New(t)
	spellCheckApp(ts, controller, focus, EditableTextOpts{Locale: foundation.Some(ui.Locale{LanguageCode: "es", CountryCode: "ES"}),
		SpellCheckConfiguration: NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: misspelledTextStyle})})
	state := spellCheckStateOf(ts)
	tree := func() *painting.TextSpan { return state.RenderEditable().Text().(*painting.TextSpan) }
	edit := func(text string) {
		state.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text)))}))
		ts.Pump()
	}

	// An empty text is not checked.
	if service.fetchCallCount != 0 {
		t.Fatalf("fetch count = %d, want 0", service.fetchCallCount)
	}
	edit("Hello, wrold!")
	if service.fetchCallCount != 1 || service.lastSpellCheckText != "Hello, wrold!" {
		t.Fatalf("fetch count = %d, last text = %q", service.fetchCallCount, service.lastSpellCheckText)
	}
	if service.lastLocale != (ui.Locale{LanguageCode: "es", CountryCode: "ES"}) {
		t.Errorf("locale = %v, want the one of the widget", service.lastLocale)
	}
	// The results come later.
	if state.SpellCheckResults() != nil {
		t.Fatal("results before the answer")
	}
	service.flush()
	want := services.NewSpellCheckResults("Hello, wrold!", []services.SuggestionSpan{spanOf(7, 12, "world")})
	if r := state.SpellCheckResults(); r == nil || !r.Equal(want) {
		t.Fatalf("results = %v, want %v", r, want)
	}
	if got := tree(); len(got.Children) != 3 || !isMisspelled(got.Children[1]) || isMisspelled(got.Children[0]) {
		t.Errorf("tree = %v, want the second word marked as misspelled", got)
	}

	// The tree follows the text before the next answer: the span is corrected.
	service.cancel = true
	edit("Hello, wrold!!")
	service.flush()
	if r := state.SpellCheckResults(); r == nil || !r.Equal(want) {
		t.Errorf("a canceled request changed the results to %v", r)
	}
	got := tree()
	if len(got.Children) != 3 || !isMisspelled(got.Children[1]) || got.Children[1].(*painting.TextSpan).Text.Value() != "wrold" {
		t.Errorf("tree = %v, want the word marked in the new text", got)
	}

	// An answer after the field was removed is ignored.
	service.cancel = false
	edit("Hello, wrold!")
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	service.flush()
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("the framework reported %d errors: %v", len(errs), errs[0])
	}
	ts.TakeErrors()
}

func TestEditableTextSpellCheckServiceFailureIsReported(t *testing.T) {
	failing := &failingSpellCheckService{}
	ts := gluttertest.New(t)
	controller := NewTextEditingController("")
	spellCheckApp(ts, controller, NewFocusNode(), EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration(
		SpellCheckConfigurationOpts{SpellCheckService: failing, MisspelledTextStyle: misspelledTextStyle})})
	state := spellCheckStateOf(ts)
	state.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset", Selection: foundation.Some(services.NewTextSelectionCollapsed(4))}))
	ts.Pump()
	err := ts.TakeError()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("EditableTextState.performSpellCheck.func1.EditableTextSpellCheckFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if state.SpellCheckResults() != nil {
		t.Errorf("results = %v, want none", state.SpellCheckResults())
	}
}

// failingSpellCheckService fails every request at once.
type failingSpellCheckService struct{}

func (failingSpellCheckService) FetchSpellCheckSuggestions(_ context.Context, _ ui.Locale, _ string, done services.SpellCheckSuggestionsCallback) {
	done(nil, false, errors.New(tagTestSpellCheckFailed, "the service failed"))
}

func TestEditableTextSpellCheckNeedsALocale(t *testing.T) {
	// Without Localizations above the field and a locale of its own, the field
	// cannot ask for a language: it reports it and does not ask.
	skipUnlessDebug(t)
	service := &mockSpellCheckService{}
	ts := gluttertest.New(t)
	controller := NewTextEditingController("")
	focus := NewFocusNode()
	ts.PumpWidget(ltr(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}}),
		NewFocusScope(NewSizedBox(NewEditableText(controller, focus, painting.NewTextStyle(), ui.NewColor(0xFF0000FF), ui.NewColor(0xFF808080),
			EditableTextOpts{SpellCheckConfiguration: NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: misspelledTextStyle})}),
			SizedBoxOpts{Width: foundation.Some(200.0)})))))
	spellCheckStateOf(ts).UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset", Selection: foundation.Some(services.NewTextSelectionCollapsed(4))}))
	ts.Pump()
	appErr, ok := errors.FromError(ts.TakeError())
	if !ok {
		t.Fatal("no error was reported")
	}
	if !appErr.HasRoute("EditableTextState.performSpellCheck.EditableTextSpellCheckLocaleMissing") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if service.fetchCallCount != 0 {
		t.Errorf("fetch count = %d, want 0", service.fetchCallCount)
	}
}

func TestEditableTextShowSpellCheckSuggestionsToolbarNeedsABuilder(t *testing.T) {
	skipUnlessDebug(t)
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	service := &mockSpellCheckService{}
	ts := gluttertest.New(t)
	spellCheckApp(ts, NewTextEditingController("tset"), NewFocusNode(), EditableTextOpts{
		ContextMenuBuilder:      func(BuildContext, *EditableTextState) Widget { return NewSizedBox(nil) },
		SpellCheckConfiguration: NewSpellCheckConfiguration(SpellCheckConfigurationOpts{SpellCheckService: service, MisspelledTextStyle: misspelledTextStyle}),
	})
	state := spellCheckStateOf(ts)
	// A change of the selection by the user makes the selection overlay.
	state.UserUpdateTextEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "tset", Selection: foundation.Some(services.NewTextSelectionCollapsed(1))}), services.SelectionChangedCauseTap)
	ts.Pump()
	state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "tset", SuggestionSpans: []services.SuggestionSpan{spanOf(0, 4, "test")}})
	v := panicOf(func() { state.ShowSpellCheckSuggestionsToolbar() })
	if v == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTagFromPanic(t, v, "EditableTextState.ShowSpellCheckSuggestionsToolbar.EditableTextSpellCheckToolbarBuilderMissing")
}

// The tags of the errors that the tests make.
const tagTestSpellCheckFailed = "TestSpellCheckFailed"
