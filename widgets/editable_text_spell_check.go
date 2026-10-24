// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
)

// SpellCheckConfiguration is the configuration that determines how spell check
// is performed: the one of the widget, with the spell check service inferred if
// it was not specified.
//
// Flutter: widgets/editable_text.dart EditableTextState.spellCheckConfiguration
func (s *EditableTextState) SpellCheckConfiguration() *SpellCheckConfiguration {
	return s.spellCheckConfiguration
}

// SpellCheckEnabled is whether spell check is enabled: a configuration was
// specified and the field is not a password field.
//
// Flutter: widgets/editable_text.dart EditableTextState.spellCheckEnabled
func (s *EditableTextState) SpellCheckEnabled() bool {
	return s.spellCheckConfiguration != nil && s.spellCheckConfiguration.SpellCheckEnabled()
}

// SpellCheckResults are the most up-to-date spell check results for the text
// input; nil when there are none. The results are updated by calls to the
// SpellCheckService, and used to build the TextSpan tree of the text and the
// menus with replacements for misspelled words.
//
// Flutter: widgets/editable_text.dart EditableTextState.spellCheckResults
func (s *EditableTextState) SpellCheckResults() *services.SpellCheckResults {
	return s.spellCheckResults
}

// SetSpellCheckResults sets the spell check results, as the field's spell check
// does when the service answers; nil clears them. The text is not rebuilt until
// the next build.
//
// Flutter: widgets/editable_text.dart EditableTextState.spellCheckResults
func (s *EditableTextState) SetSpellCheckResults(results *services.SpellCheckResults) {
	s.spellCheckResults = results
}

// spellCheckResultsReceived is whether spell check is enabled and results with
// at least one misspelled span have been received.
//
// Flutter: widgets/editable_text.dart EditableTextState._spellCheckResultsReceived
func (s *EditableTextState) spellCheckResultsReceived() bool {
	return s.SpellCheckEnabled() && s.spellCheckResults != nil && len(s.spellCheckResults.SuggestionSpans) > 0
}

// FindSuggestionSpanAtCursorIndex finds the span that matches the index, using a
// binary search, and says whether there is one. The index is a byte offset into
// the text that the spell check results are for.
//
// Flutter: widgets/editable_text.dart EditableTextState.findSuggestionSpanAtCursorIndex
func (s *EditableTextState) FindSuggestionSpanAtCursorIndex(cursorIndex int) (services.SuggestionSpan, bool) {
	if !s.spellCheckResultsReceived() ||
		s.spellCheckResults.SuggestionSpans[len(s.spellCheckResults.SuggestionSpans)-1].Range.End < cursorIndex {
		// No spell check results have been received or the cursor index is out
		// of range that suggestionSpans covers.
		return services.SuggestionSpan{}, false
	}

	suggestionSpans := s.spellCheckResults.SuggestionSpans
	leftIndex := 0
	rightIndex := len(suggestionSpans) - 1

	for leftIndex <= rightIndex {
		midIndex := (leftIndex + rightIndex) / 2
		currentSpanStart := suggestionSpans[midIndex].Range.Start
		currentSpanEnd := suggestionSpans[midIndex].Range.End

		if cursorIndex <= currentSpanEnd && cursorIndex >= currentSpanStart {
			return suggestionSpans[midIndex], true
		} else if cursorIndex <= currentSpanStart {
			rightIndex = midIndex - 1
		} else {
			leftIndex = midIndex + 1
		}
	}
	return services.SuggestionSpan{}, false
}

// nativeSpellCheckServiceDefined is whether the platform has a spell checker for
// the DefaultSpellCheckService.
//
// Deviation: Flutter reads PlatformDispatcher.nativeSpellCheckServiceDefined, which
// the engine sets on Android and iOS; here it is the Android target platform with
// an engine that has a spell checker (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart EditableTextState._inferSpellCheckConfiguration (nativeSpellCheckServiceDefined)
func (s *EditableTextState) nativeSpellCheckServiceDefined() bool {
	return services.DefaultSpellCheckServiceSupportsSpellCheck(s.binding().PlatformDispatcher())
}

// inferSpellCheckConfiguration infers the SpellCheckConfiguration used to perform
// spell check. If spell check is enabled, this tries to infer a value for the
// SpellCheckService if it was left unspecified. When it cannot be enabled, it is
// the disabled configuration; asking for spell check on a platform without a
// spell checker, and without a service, is reported (the tag
// EditableTextSpellCheckUnsupported).
//
// Flutter: widgets/editable_text.dart EditableTextState._inferSpellCheckConfiguration
func (s *EditableTextState) inferSpellCheckConfiguration(configuration *SpellCheckConfiguration, obscureText bool,
	keyboardType services.TextInputType, autofillHints []string) *SpellCheckConfiguration {
	var spellCheckService services.SpellCheckService
	if configuration != nil {
		spellCheckService = configuration.SpellCheckService()
	}
	spellCheckAutomaticallyDisabled := isPasswordInput(obscureText, keyboardType, autofillHints) ||
		configuration == nil || configuration.Equal(NewSpellCheckConfigurationDisabled())
	spellCheckServiceIsConfigured := spellCheckService != nil || s.nativeSpellCheckServiceDefined()
	if spellCheckAutomaticallyDisabled || !spellCheckServiceIsConfigured {
		// Only enable spell check if a non-disabled configuration is provided
		// for non-password input and, if that configuration does not specify a
		// spell check service, a native spell checker must be supported.
		if foundation.DebugMode && !spellCheckAutomaticallyDisabled && !spellCheckServiceIsConfigured {
			s.binding().ErrorReporter()(errors.New(tagEditableTextSpellCheckUnsupported,
				"Spell check was enabled with spellCheckConfiguration, but the current platform does not have a supported spell check "+
					"service, and none was provided. Consider disabling spell check for this platform or passing a "+
					"SpellCheckConfiguration with a specified spell check service."))
		}
		return NewSpellCheckConfigurationDisabled()
	}

	if spellCheckService == nil {
		spellCheckService = services.NewDefaultSpellCheckService(s.binding().PlatformDispatcher())
	}
	return configuration.CopyWith(SpellCheckConfigurationCopy{SpellCheckService: spellCheckService})
}

// isPasswordInput is whether the field is for a password, which is not spell
// checked.
//
// Flutter: widgets/editable_text.dart EditableTextState._isPasswordInput
func isPasswordInput(obscureText bool, keyboardType services.TextInputType, autofillHints []string) bool {
	if obscureText || keyboardType == services.TextInputTypeVisiblePassword {
		return true
	}
	for _, hint := range autofillHints {
		if hint == services.AutofillHintPassword || hint == services.AutofillHintNewPassword {
			return true
		}
	}
	return false
}

// performSpellCheck asks the spell check service for the misspelled words of the
// text, and shows them when the answer comes, if the field is still mounted, spell
// check is still enabled and the request was not canceled.
//
// Deviation: Flutter awaits a Future; here the service calls back on the UI thread
// (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart EditableTextState._performSpellCheck
func (s *EditableTextState) performSpellCheck(text string) {
	localeForSpellChecking, ok := s.Widget().o.Locale.Get()
	if !ok {
		localeForSpellChecking, ok = LocalizationsMaybeLocaleOf(s.Context())
	}
	if !ok {
		s.binding().ErrorReporter()(errors.New(tagEditableTextSpellCheckLocaleMissing,
			"Locale must be specified in widget or Localization widget must be in scope"))
		return
	}

	s.spellCheckConfiguration.SpellCheckService().FetchSpellCheckSuggestions(s.Lifetime(), localeForSpellChecking, text,
		func(suggestions []services.SuggestionSpan, ok bool, err error) {
			if err != nil {
				s.binding().ErrorReporter()(errors.NewErr(tagEditableTextSpellCheckFailed, err).SetMetadata("context", "while performing spell check"))
				return
			}
			if !ok || !s.Mounted() || !s.SpellCheckEnabled() {
				// The request to fetch spell check suggestions was canceled due to ongoing request,
				// the widget was unmounted, or spell check was disabled before the
				// request completed.
				return
			}

			results := services.NewSpellCheckResults(text, suggestions)
			s.spellCheckResults = &results
			if r := s.RenderEditable(); r != nil {
				r.SetText(s.BuildTextSpan())
			}
		})
}

// webContextMenuEnabled is whether the browser's context menu is on.
//
// Flutter: widgets/editable_text.dart EditableTextState._webContextMenuEnabled
func (s *EditableTextState) webContextMenuEnabled() bool {
	return foundation.IsWeb && s.binding().BrowserContextMenu().Enabled()
}

// ShowSpellCheckSuggestionsToolbar shows the toolbar with the spell check
// suggestions of misspelled words that are available for click-and-replace, and
// says whether it did. It does not when spell check is off, on the web with the
// browser's context menu, for a read-only field, without a selection overlay,
// without results, or when the cursor is not in a misspelled word. The
// configuration needs a SpellCheckSuggestionsToolbarBuilder (the panic
// EditableTextSpellCheckToolbarBuilderMissing in debug builds).
//
// Flutter: widgets/editable_text.dart EditableTextState.showSpellCheckSuggestionsToolbar
func (s *EditableTextState) ShowSpellCheckSuggestionsToolbar() bool {
	// Spell check suggestions toolbars are intended to be shown on non-web
	// platforms.
	if !s.SpellCheckEnabled() || s.webContextMenuEnabled() || s.Widget().o.ReadOnly || s.selectionOverlay == nil ||
		!s.spellCheckResultsReceived() {
		return false
	}
	if _, found := s.FindSuggestionSpanAtCursorIndex(s.value().Selection.ExtentOffset); !found {
		// Only attempt to show the spell check suggestions toolbar if there
		// is a toolbar specified and spell check suggestions available to show.
		return false
	}

	builder := s.spellCheckConfiguration.SpellCheckSuggestionsToolbarBuilder()
	if foundation.DebugMode && builder == nil {
		panic(errors.New(tagEditableTextSpellCheckToolbarBuilderMissing,
			"spellCheckSuggestionsToolbarBuilder must be defined in SpellCheckConfiguration to show a toolbar with spell check suggestions"))
	}

	s.selectionOverlay.ShowSpellCheckSuggestionsToolbar(func(bc BuildContext) Widget {
		return s.spellCheckConfiguration.SpellCheckSuggestionsToolbarBuilder()(bc, s)
	})
	return true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEditableTextSpellCheckFailed                = "EditableTextSpellCheckFailed"
	tagEditableTextSpellCheckLocaleMissing         = "EditableTextSpellCheckLocaleMissing"
	tagEditableTextSpellCheckToolbarBuilderMissing = "EditableTextSpellCheckToolbarBuilderMissing"
	tagEditableTextSpellCheckUnsupported           = "EditableTextSpellCheckUnsupported"
)
