// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"context"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SuggestionSpan represents a span of text that has been marked as misspelled,
// with the words that the spell checker suggests for it.
//
// Deviation: the range is in UTF-8 bytes, like every text offset of Glutter, not
// in UTF-16 code units (SPEC §12.10).
//
// Flutter: services/spell_check.dart SuggestionSpan
type SuggestionSpan struct {
	// Range is the misspelled range of text.
	//
	// Flutter: services/spell_check.dart SuggestionSpan.range
	Range ui.TextRange
	// Suggestions are the suggestions for the misspelled range, the best first.
	//
	// Flutter: services/spell_check.dart SuggestionSpan.suggestions
	Suggestions []string
}

// NewSuggestionSpan creates a span of misspelled text with its suggestions.
//
// Flutter: services/spell_check.dart SuggestionSpan.new
func NewSuggestionSpan(textRange ui.TextRange, suggestions []string) SuggestionSpan {
	return SuggestionSpan{Range: textRange, Suggestions: suggestions}
}

// Equal reports whether the spans have the same range and the same suggestions.
//
// Flutter: services/spell_check.dart SuggestionSpan.==
func (s SuggestionSpan) Equal(other SuggestionSpan) bool {
	return s.Range.Start == other.Range.Start && s.Range.End == other.Range.End &&
		slices.Equal(s.Suggestions, other.Suggestions)
}

// String describes the span.
//
// Flutter: services/spell_check.dart SuggestionSpan.toString
func (s SuggestionSpan) String() string {
	return "SuggestionSpan(range: " + s.Range.String() + ", suggestions: " + listString(s.Suggestions) + ")"
}

// listString prints a list the way Dart does: [a, b].
func listString(list []string) string {
	return "[" + strings.Join(list, ", ") + "]"
}

// SpellCheckResults are the results of a spell check request: the text that was
// checked and the misspelled ranges that were found in it.
//
// Flutter: services/spell_check.dart SpellCheckResults
type SpellCheckResults struct {
	// SpellCheckedText is the text that the spell check results are for.
	//
	// Flutter: services/spell_check.dart SpellCheckResults.spellCheckedText
	SpellCheckedText string
	// SuggestionSpans are the spans of misspelled text with their suggestions,
	// in the order of the text.
	//
	// Flutter: services/spell_check.dart SpellCheckResults.suggestionSpans
	SuggestionSpans []SuggestionSpan
}

// NewSpellCheckResults creates the results of the spell check of a text.
//
// Flutter: services/spell_check.dart SpellCheckResults.new
func NewSpellCheckResults(spellCheckedText string, suggestionSpans []SuggestionSpan) SpellCheckResults {
	return SpellCheckResults{SpellCheckedText: spellCheckedText, SuggestionSpans: suggestionSpans}
}

// Equal reports whether the results are for the same text and have the same
// spans.
//
// Flutter: services/spell_check.dart SpellCheckResults.==
func (r SpellCheckResults) Equal(other SpellCheckResults) bool {
	return r.SpellCheckedText == other.SpellCheckedText &&
		slices.EqualFunc(r.SuggestionSpans, other.SuggestionSpans, SuggestionSpan.Equal)
}

// String describes the results.
//
// Flutter: services/spell_check.dart SpellCheckResults.toString
func (r SpellCheckResults) String() string {
	spans := make([]string, len(r.SuggestionSpans))
	for i, s := range r.SuggestionSpans {
		spans[i] = s.String()
	}
	return "SpellCheckResults(spellCheckText: " + r.SpellCheckedText + ", suggestionSpans: " + listString(spans) + ")"
}

// SpellCheckSuggestionsCallback receives the result of
// SpellCheckService.FetchSpellCheckSuggestions. ok is false when the request was
// canceled or did not give a result, which is Dart's null: the spans are then
// not used and the old results stay. A non-nil err is a failure of the service,
// which the text field reports. It runs on the UI thread, never before the
// request call has returned.
//
// Flutter: services/spell_check.dart SpellCheckService.fetchSpellCheckSuggestions
type SpellCheckSuggestionsCallback func(spans []SuggestionSpan, ok bool, err error)

// SpellCheckService is the interface to implement for a spell check service to
// be used with an EditableText: it gets the misspelled words of a text.
//
// Deviation: Flutter's method returns a Future; here it takes a callback that
// runs later on the UI thread, and the request is bound to a context (SPEC
// §12.10).
//
// Flutter: services/spell_check.dart SpellCheckService
type SpellCheckService interface {
	// FetchSpellCheckSuggestions starts the spell check of the text in the locale
	// and calls done with the misspelled ranges of the text and their suggestions.
	//
	// Flutter: services/spell_check.dart SpellCheckService.fetchSpellCheckSuggestions
	FetchSpellCheckSuggestions(ctx context.Context, locale ui.Locale, text string, done SpellCheckSuggestionsCallback)
}

// SpellCheckBackend is what the DefaultSpellCheckService asks of the engine: the
// platform dispatcher (ui.PlatformDispatcher) is one. done is called later on
// the UI thread, never before the method has returned.
//
// Flutter: no counterpart (the flutter/spellcheck channel)
type SpellCheckBackend interface {
	// SpellCheckSupported says whether the engine has a spell checker.
	SpellCheckSupported() bool
	// SpellCheck asks for the misspelled words of the text in the locale (a BCP 47
	// tag, such as "en-US").
	SpellCheck(ctx context.Context, locale, text string, done func(spans []ui.SpellCheckSpan, err error))
}

// DefaultSpellCheckService is the spell check service that uses the spell
// checker of the platform: Android's TextServicesManager. It keeps the last
// results to merge them with the next ones when only the spell checker's
// answer changed.
//
// Deviation: Flutter talks to the flutter/spellcheck method channel; here the
// requests go through the platform dispatcher (ui.PlatformDispatcher.SpellCheck).
//
// Flutter: services/spell_check.dart DefaultSpellCheckService
type DefaultSpellCheckService struct {
	backend          SpellCheckBackend
	lastSavedResults *SpellCheckResults
}

// NewDefaultSpellCheckService creates the service on a backend, usually the
// platform dispatcher of the binding.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.new
func NewDefaultSpellCheckService(backend SpellCheckBackend) *DefaultSpellCheckService {
	return &DefaultSpellCheckService{backend: backend}
}

var _ SpellCheckService = (*DefaultSpellCheckService)(nil)

// LastSavedResults are the results that the last request gave; nil before the
// first one.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.lastSavedResults
func (s *DefaultSpellCheckService) LastSavedResults() *SpellCheckResults { return s.lastSavedResults }

// DefaultSpellCheckServiceSupportsSpellCheck says whether the platform has a spell
// checker that the service can ask: the target platform is Android and the engine has a spell
// checker.
//
// Deviation: Flutter's engine sets PlatformDispatcher.nativeSpellCheckServiceDefined
// on Android and iOS; Glutter has the spell checker on Android only, and the
// dispatcher says whether the host has one, so Linux and the web keep spell check
// off as in Flutter (SPEC §12.10).
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.supportsSpellCheck
func DefaultSpellCheckServiceSupportsSpellCheck(backend SpellCheckBackend) bool {
	return foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid &&
		backend != nil && backend.SpellCheckSupported()
}

// DefaultSpellCheckServiceMergeResults merges two lists of spans that are sorted
// by their start, keeping the old span when both start at the same index.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.mergeResults
func DefaultSpellCheckServiceMergeResults(oldResults, newResults []SuggestionSpan) []SuggestionSpan {
	mergedResults := []SuggestionSpan{}
	oldSpanPointer := 0
	newSpanPointer := 0

	for oldSpanPointer < len(oldResults) && newSpanPointer < len(newResults) {
		oldSpan := oldResults[oldSpanPointer]
		newSpan := newResults[newSpanPointer]

		if oldSpan.Range.Start == newSpan.Range.Start {
			mergedResults = append(mergedResults, oldSpan)
			oldSpanPointer++
			newSpanPointer++
		} else if oldSpan.Range.Start < newSpan.Range.Start {
			mergedResults = append(mergedResults, oldSpan)
			oldSpanPointer++
		} else {
			mergedResults = append(mergedResults, newSpan)
			newSpanPointer++
		}
	}

	mergedResults = append(mergedResults, oldResults[oldSpanPointer:]...)
	mergedResults = append(mergedResults, newResults[newSpanPointer:]...)
	return mergedResults
}

// FetchSpellCheckSuggestions asks the platform to spell check the text. A
// request that the platform canceled because a newer one replaced it, or that
// failed, gives no result (ok is false), as Flutter's returns null. A service
// without a backend calls done at once with the error SpellCheckServiceUnavailable.
//
// Flutter: services/spell_check.dart DefaultSpellCheckService.fetchSpellCheckSuggestions
func (s *DefaultSpellCheckService) FetchSpellCheckSuggestions(ctx context.Context, locale ui.Locale, text string, done SpellCheckSuggestionsCallback) {
	if s.backend == nil {
		done(nil, false, errors.New(tagSpellCheckServiceUnavailable, "the spell check service has no backend"))
		return
	}
	s.backend.SpellCheck(ctx, locale.LanguageTag(), text, func(spans []ui.SpellCheckSpan, err error) {
		if err != nil {
			// Spell check request canceled due to pending request. Flutter's service
			// catches every error of the channel and gives no result.
			done(nil, false, nil)
			return
		}
		suggestionSpans := make([]SuggestionSpan, len(spans))
		for i, span := range spans {
			suggestionSpans[i] = SuggestionSpan{
				Range:       ui.TextRange{Start: span.Start, End: span.End},
				Suggestions: slices.Clone(span.Suggestions),
			}
		}

		if s.lastSavedResults != nil {
			// Merge current and previous spell check results if between requests,
			// the text has not changed but the spell check results have.
			textHasNotChanged := s.lastSavedResults.SpellCheckedText == text
			spansHaveChanged := slices.EqualFunc(s.lastSavedResults.SuggestionSpans, suggestionSpans, SuggestionSpan.Equal)

			if textHasNotChanged && spansHaveChanged {
				suggestionSpans = DefaultSpellCheckServiceMergeResults(s.lastSavedResults.SuggestionSpans, suggestionSpans)
			}
		}
		saved := NewSpellCheckResults(text, suggestionSpans)
		s.lastSavedResults = &saved

		done(suggestionSpans, true, nil)
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSpellCheckServiceUnavailable = "SpellCheckServiceUnavailable"
)
