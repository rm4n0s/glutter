// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests port material/spell_check_suggestions_toolbar_test.dart, the cases
// of the group 'Spell check' of widgets/editable_text_test.dart that show the
// Material toolbar, and the two cases of material/text_field_test.dart about the
// defaults of the spell check of a TextField. The offsets of the spans are UTF-8
// bytes (SPEC §12.10). The cases that run with Cupertino in Flutter (iOS) run with
// the Material toolbar here, as there is no Cupertino library. Not ported: 'replacing
// puts cursor at the end of the word' and the cases about 'No Replacements Found',
// which are about the Cupertino toolbar (CupertinoSpellCheckSuggestionsToolbar).

const (
	// Vertical position at which to anchor the toolbar for testing.
	spellCheckTestAnchor = 200.0
	// Amount for toolbar to overlap bottom padding for testing.
	spellCheckTestToolbarOverlap = 10.0
)

// buildSuggestionButtons builds test button items for each of the suggestions
// provided.
func buildSuggestionButtons(suggestions ...string) []widgets.ContextMenuButtonItem {
	items := make([]widgets.ContextMenuButtonItem, 0, len(suggestions)+1)
	for _, suggestion := range suggestions {
		items = append(items, widgets.NewContextMenuButtonItem(func() {}, widgets.ContextMenuButtonItemOpts{Label: foundation.Some(suggestion)}))
	}
	return append(items, widgets.NewContextMenuButtonItem(func() {}, widgets.ContextMenuButtonItemOpts{
		Type: widgets.ContextMenuButtonTypeDelete, Label: foundation.Some("DELETE")}))
}

// findSpellCheckSuggestionsToolbar finds the container of the
// SpellCheckSuggestionsToolbar so that the position of the toolbar itself may be
// determined.
func findSpellCheckSuggestionsToolbar() gluttertest.Finder {
	return findPrivate("spellCheckSuggestionsToolbarContainer")
}

func TestSpellCheckSuggestionsToolbarPosition(t *testing.T) {
	tests := []struct {
		name   string
		anchor float64
		want   float64
	}{
		// We expect the toolbar to be positioned right below the anchor with padding accounted for.
		{"positions toolbar below anchor when it fits above bottom view padding", spellCheckTestAnchor, spellCheckTestAnchor},
		// We expect the toolbar to be positioned _kTestToolbarOverlap pixels above the anchor.
		{"re-positions toolbar higher below anchor when it does not fit above bottom view padding",
			spellCheckTestAnchor - spellCheckTestToolbarOverlap, spellCheckTestAnchor - spellCheckTestToolbarOverlap},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
				Body: NewSpellCheckSuggestionsToolbar(ui.Offset{Dy: tc.anchor}, buildSuggestionButtons("hello", "yellow", "yell")),
			})}))
			if got := ts.TopLeft(findSpellCheckSuggestionsToolbar()).Dy; got != tc.want {
				t.Errorf("toolbar y = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSpellCheckSuggestionsToolbarSize(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []string
		wantHeight  float64
	}{
		{"three suggestions and delete", []string{"hello", "yellow", "yell"}, 193},
		{"two suggestions and delete", []string{"X", "Y"}, 145},
		{"delete only", nil, 49},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
				Body: NewSpellCheckSuggestionsToolbar(ui.Offset{}, buildSuggestionButtons(tc.suggestions...)),
			})}))
			if got := ts.Size(findSpellCheckSuggestionsToolbar()); got != (ui.Size{Width: 165, Height: tc.wantHeight}) {
				t.Errorf("toolbar size = %v, want 165 x %v", got, tc.wantHeight)
			}
		})
	}
}

func TestSpellCheckSuggestionsToolbarMoreThanThreeSuggestionsPanics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name        string
		suggestions []string
		wantPanic   bool
	}{
		{"three suggestions", []string{"hello", "yellow", "yell"}, false},
		{"four suggestions", []string{"hello", "yellow", "yell", "yeller"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v any
			func() {
				defer func() { v = recover() }()
				NewSpellCheckSuggestionsToolbar(ui.Offset{Dy: spellCheckTestAnchor - spellCheckTestToolbarOverlap}, buildSuggestionButtons(tc.suggestions...))
			}()
			if !tc.wantPanic {
				if v != nil {
					t.Fatalf("panicked with %v", v)
				}
				return
			}
			err, _ := v.(error)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", v)
			}
			if !appErr.HasRoute("NewSpellCheckSuggestionsToolbar.SpellCheckSuggestionsToolbarTooManyButtons") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestSpellCheckSuggestionsToolbarDoesNotCrashAtZeroArea(t *testing.T) {
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
		NewSpellCheckSuggestionsToolbar(ui.Offset{Dx: 1, Dy: 1}, buildSuggestionButtons("X", "Y"))))}))
	if got := ts.Size(gluttertest.ByType[*SpellCheckSuggestionsToolbar]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}

func TestSpellCheckSuggestionsToolbarWithoutButtonsBuildsNothing(t *testing.T) {
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewSpellCheckSuggestionsToolbar(ui.Offset{}, nil)})}))
	if ts.Exists(findSpellCheckSuggestionsToolbar()) {
		t.Error("an empty toolbar built a container")
	}
}

func TestSpellCheckSuggestionsToolbarLayoutDelegate(t *testing.T) {
	size := ui.Size{Width: 400, Height: 600}
	child := ui.Size{Width: 165, Height: 193}
	tests := []struct {
		name   string
		anchor ui.Offset
		want   ui.Offset
	}{
		{"below the anchor, centered", ui.Offset{Dx: 200, Dy: 100}, ui.Offset{Dx: 117.5, Dy: 100}},
		{"moved up to fit", ui.Offset{Dx: 200, Dy: 500}, ui.Offset{Dx: 117.5, Dy: 407}},
		{"kept inside on the left", ui.Offset{Dx: 10, Dy: 100}, ui.Offset{Dx: 0, Dy: 100}},
		{"kept inside on the right", ui.Offset{Dx: 395, Dy: 100}, ui.Offset{Dx: 235, Dy: 100}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewSpellCheckSuggestionsToolbarLayoutDelegate(tc.anchor)
			if got := d.GetPositionForChild(size, child); got != tc.want {
				t.Errorf("position = %v, want %v", got, tc.want)
			}
			if got := d.GetConstraintsForChild(rendering.NewBoxConstraintsTight(size)); got.MinWidth != 0 || got.MaxWidth != 400 {
				t.Errorf("constraints = %v, want them loosened", got)
			}
		})
	}
	a := NewSpellCheckSuggestionsToolbarLayoutDelegate(ui.Offset{Dx: 1, Dy: 2})
	if a.ShouldRelayout(NewSpellCheckSuggestionsToolbarLayoutDelegate(ui.Offset{Dx: 1, Dy: 2})) {
		t.Error("ShouldRelayout for the same anchor")
	}
	if !a.ShouldRelayout(NewSpellCheckSuggestionsToolbarLayoutDelegate(ui.Offset{Dx: 1, Dy: 3})) {
		t.Error("no ShouldRelayout for another anchor")
	}
}

func TestSpellCheckSuggestionsToolbarGetToolbarAnchor(t *testing.T) {
	tests := []struct {
		name    string
		anchors widgets.TextSelectionToolbarAnchors
		want    ui.Offset
	}{
		{"the secondary anchor", widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 1, Dy: 2}, SecondaryAnchor: foundation.Some(ui.Offset{Dx: 3, Dy: 4})}, ui.Offset{Dx: 3, Dy: 4}},
		{"the primary anchor without a secondary one", widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 1, Dy: 2}}, ui.Offset{Dx: 1, Dy: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SpellCheckSuggestionsToolbarGetToolbarAnchor(tc.anchors); got != tc.want {
				t.Errorf("anchor = %v, want %v", got, tc.want)
			}
		})
	}
}

// spellCheckService answers the requests with the spans it was given, when flush
// is called.
type spellCheckService struct {
	spans   []services.SuggestionSpan
	pending []func()
}

func (s *spellCheckService) FetchSpellCheckSuggestions(_ context.Context, _ ui.Locale, text string, done services.SpellCheckSuggestionsCallback) {
	spans := s.spans
	s.pending = append(s.pending, func() { done(spans, true, nil) })
}

func (s *spellCheckService) flush() {
	for len(s.pending) > 0 {
		fn := s.pending[0]
		s.pending = s.pending[1:]
		fn()
	}
}

func span(start, end int, suggestions ...string) services.SuggestionSpan {
	return services.NewSuggestionSpan(ui.TextRange{Start: start, End: end}, suggestions)
}

// spellCheckEditable pumps an EditableText with the value and the Material spell
// check toolbar in a Material app.
func spellCheckEditable(t *testing.T, ts *gluttertest.Tester, value services.TextEditingValue, config *widgets.SpellCheckConfiguration, opts ...func(*widgets.EditableTextOpts)) (*widgets.TextEditingController, *widgets.FocusNode) {
	t.Helper()
	controller := widgets.NewTextEditingController("")
	t.Cleanup(controller.Dispose)
	controller.SetValue(value)
	focus := widgets.NewFocusNode()
	t.Cleanup(focus.Dispose)
	o := widgets.EditableTextOpts{SelectionControls: MaterialTextSelectionHandleControls, SpellCheckConfiguration: config}
	for _, f := range opts {
		f(&o)
	}
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewEditableText(controller, focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		ui.NewColor(0xFF0000FF), ui.NewColor(0xFF808080), o)}))
	return controller, focus
}

func spellCheckConfig(service services.SpellCheckService) *widgets.SpellCheckConfiguration {
	return widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{
		SpellCheckService: service, MisspelledTextStyle: TextFieldMaterialMisspelledTextStyle,
		SpellCheckSuggestionsToolbarBuilder: TextFieldDefaultSpellCheckSuggestionsToolbarBuilder,
	})
}

func TestSpellCheckSuggestionsToolbarBuildButtonItems(t *testing.T) {
	tests := []struct {
		name         string
		suggestions  []string
		cursor       int
		wantOK       bool
		wantLabels   []string
		wantLastType widgets.ContextMenuButtonType
	}{
		{"only considers the first three suggestions", []string{"hello", "yellow", "yell", "yeller"}, 0, true,
			[]string{"hello", "yellow", "yell", ""}, widgets.ContextMenuButtonTypeDelete},
		{"builds only a delete button when no suggestions", nil, 0, true, []string{""}, widgets.ContextMenuButtonTypeDelete},
		{"builds nothing when the cursor is not in a misspelled word", []string{"hello"}, 8, false, nil, widgets.ContextMenuButtonTypeUnspecified},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			service := &spellCheckService{}
			spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: "hello world", Selection: foundation.Some(services.NewTextSelectionCollapsed(tc.cursor))}), spellCheckConfig(service))
			state := editableState(ts)
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "hello world", SuggestionSpans: []services.SuggestionSpan{span(0, 5, tc.suggestions...)}})
			items, ok := SpellCheckSuggestionsToolbarBuildButtonItems(state)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			var labels []string
			for _, item := range items {
				labels = append(labels, item.Label.OrElse(""))
			}
			if !slices.Equal(labels, tc.wantLabels) {
				t.Errorf("labels = %q, want %q", labels, tc.wantLabels)
			}
			if ok && items[len(items)-1].Type != tc.wantLastType {
				t.Errorf("last type = %v, want delete", items[len(items)-1].Type)
			}
		})
	}
}

// useSpellChecker makes the platform Android with a spell checker that finds the
// span in every text, as Flutter's tests set nativeSpellCheckServiceDefinedTestValue.
func useSpellChecker(t *testing.T, spans ...ui.SpellCheckSpan) gluttertest.Opts {
	t.Helper()
	withPlatform(t, foundation.TargetPlatformAndroid)
	return gluttertest.Opts{SpellChecker: func(string, string) []ui.SpellCheckSpan { return spans }}
}

func TestEditableTextCanShowSpellCheckSuggestionsToolbarWhenThereAreSpellCheckResults(t *testing.T) {
	ts := newToolbarTester(t, useSpellChecker(t, ui.SpellCheckSpan{Start: 0, End: 4, Suggestions: []string{"test", "sets", "set"}}))
	spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset test test",
		Selection: foundation.Some(services.TextSelection{BaseOffset: 0, ExtentOffset: 4, Affinity: ui.TextAffinityUpstream})}),
		widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{
			MisspelledTextStyle: TextFieldMaterialMisspelledTextStyle, SpellCheckSuggestionsToolbarBuilder: TextFieldDefaultSpellCheckSuggestionsToolbarBuilder}))
	state := editableState(ts)

	// Can't show the toolbar when there's no focus.
	if state.ShowSpellCheckSuggestionsToolbar() {
		t.Error("the toolbar is shown without focus")
	}
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("DELETE")) {
		t.Error("DELETE is shown without focus")
	}

	// Can't show the toolbar when there are no spell check results.
	if state.ShowSpellCheckSuggestionsToolbar() {
		t.Error("the toolbar is shown without results")
	}
	ts.PumpAndSettle()
	for _, label := range []string{"test", "sets", "set", "DELETE"} {
		if ts.Exists(gluttertest.ByText(label)) {
			t.Errorf("%q is shown without results", label)
		}
	}

	// Can show the toolbar when there are spell check results.
	state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "test tset test", SuggestionSpans: []services.SuggestionSpan{span(0, 4, "test", "sets", "set")}})
	editableOf(ts).SelectWordsInRange(ui.Offset{}, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
	ts.PumpAndSettle()
	// The toolbar is only shown on non-web platforms.
	if got := state.ShowSpellCheckSuggestionsToolbar(); got == foundation.IsWeb {
		t.Fatalf("ShowSpellCheckSuggestionsToolbar = %v with results, want %v", got, !foundation.IsWeb)
	}
	ts.PumpAndSettle()
	if foundation.IsWeb {
		for _, label := range []string{"test", "sets", "set", "DELETE"} {
			if ts.Exists(gluttertest.ByText(label)) {
				t.Errorf("%q is shown on the web", label)
			}
		}
		return
	}
	for _, label := range []string{"test", "sets", "set", "DELETE"} {
		if got := ts.Count(gluttertest.ByText(label)); got != 1 {
			t.Errorf("%q found %d times, want 1", label, got)
		}
	}
	if !state.SelectionOverlay().SpellCheckToolbarIsVisible() || !state.SelectionOverlay().ToolbarIsVisible() {
		t.Error("the overlay does not say that the spell check toolbar is visible")
	}
	state.HideToolbar(true)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("DELETE")) || state.SelectionOverlay() != nil && state.SelectionOverlay().ToolbarIsVisible() {
		t.Error("the toolbar is still shown after HideToolbar")
	}
}

func TestEditableTextCannotShowSpellCheckSuggestionsToolbar(t *testing.T) {
	tests := []struct {
		name string
		opts func(*widgets.EditableTextOpts)
	}{
		{"the field is read-only", func(o *widgets.EditableTextOpts) { o.ReadOnly = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t, useSpellChecker(t))
			_, focus := spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset test",
				Selection: foundation.Some(services.NewTextSelectionCollapsed(2))}), spellCheckConfig(&spellCheckService{}), tc.opts)
			focus.RequestFocus()
			ts.Pump()
			state := editableState(ts)
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "tset test", SuggestionSpans: []services.SuggestionSpan{span(0, 4, "test")}})
			if state.ShowSpellCheckSuggestionsToolbar() {
				t.Error("the toolbar is shown")
			}
		})
	}
}

func TestEditableTextMaterialSpellCheckSuggestionsToolbarButtonsChangeTheComposingRegion(t *testing.T) {
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	ts := newToolbarTester(t, useSpellChecker(t, ui.SpellCheckSpan{Start: 0, End: 4, Suggestions: []string{"test", "sets", "set"}}))
	spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset test test",
		Composing: foundation.Some(ui.TextRange{Start: 0, End: 4}),
		Selection: foundation.Some(services.TextSelection{BaseOffset: 0, ExtentOffset: 4, Affinity: ui.TextAffinityUpstream})}),
		widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{
			MisspelledTextStyle: TextFieldMaterialMisspelledTextStyle, SpellCheckSuggestionsToolbarBuilder: TextFieldDefaultSpellCheckSuggestionsToolbarBuilder}))
	state := editableState(ts)
	state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "tset test test", SuggestionSpans: []services.SuggestionSpan{span(0, 4, "test", "sets", "set")}})
	editableOf(ts).SelectWordsInRange(ui.Offset{}, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
	ts.PumpAndSettle()
	if got := state.TextEditingValue().Selection.BaseOffset; got != 0 {
		t.Fatalf("base offset = %d, want 0", got)
	}

	// Test misspelled word replacement buttons.
	state.ShowSpellCheckSuggestionsToolbar()
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByText("sets")); got != 1 {
		t.Fatalf("sets found %d times, want 1", got)
	}
	ts.Tap(gluttertest.ByText("sets"))
	ts.PumpAndSettle()
	if got := state.TextEditingValue().Text; got != "sets test test" {
		t.Errorf("text = %q, want %q", got, "sets test test")
	}
	if got := state.TextEditingValue().Selection.BaseOffset; got != 0 {
		t.Errorf("base offset = %d, want 0", got)
	}

	// Test delete button.
	state.ShowSpellCheckSuggestionsToolbar()
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByText("DELETE")); got != 1 {
		t.Fatalf("DELETE found %d times, want 1", got)
	}
	ts.Tap(gluttertest.ByText("DELETE"))
	ts.PumpAndSettle()
	if got := state.TextEditingValue().Text; got != " test test" {
		t.Errorf("text = %q, want %q", got, " test test")
	}
	if got := state.TextEditingValue().Selection.BaseOffset; got != 0 {
		t.Errorf("base offset = %d, want 0", got)
	}
}

func TestEditableTextTappingOnAMisspelledWordHidesTheHandles(t *testing.T) {
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	ts := newToolbarTester(t, useSpellChecker(t))
	// All misspellings of "test". One the same length, one shorter, and one longer.
	text := "test test testt"
	spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{Text: text,
		Selection: foundation.Some(services.TextSelection{BaseOffset: 0, ExtentOffset: 4, Affinity: ui.TextAffinityUpstream})}),
		widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{
			MisspelledTextStyle: TextFieldMaterialMisspelledTextStyle, SpellCheckSuggestionsToolbarBuilder: TextFieldDefaultSpellCheckSuggestionsToolbarBuilder}),
		func(o *widgets.EditableTextOpts) { o.ShowSelectionHandles = true })
	state := editableState(ts)
	state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: text, SuggestionSpans: []services.SuggestionSpan{span(10, 15, "test")}})

	ts.TapAt(fieldTextOffsetToPosition(ts, 0))
	ts.PumpAndSettle()
	if state.ShowSpellCheckSuggestionsToolbar() {
		t.Error("the toolbar is shown for a correct word")
	}
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("test")) {
		t.Error("a suggestion is shown for a correct word")
	}
	if !state.SelectionOverlay().HandlesAreVisible() {
		t.Error("the handles are hidden")
	}

	ts.TapAt(fieldTextOffsetToPosition(ts, 12))
	ts.PumpAndSettle()
	if !state.ShowSpellCheckSuggestionsToolbar() {
		t.Fatal("the toolbar is not shown for a misspelled word")
	}
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByText("test")); got != 1 {
		t.Errorf("test found %d times, want 1", got)
	}
	if state.SelectionOverlay().HandlesAreVisible() {
		t.Error("the handles are visible with the spell check toolbar")
	}

	ts.TapAt(fieldTextOffsetToPosition(ts, 5))
	ts.PumpAndSettle()
	if state.ShowSpellCheckSuggestionsToolbar() {
		t.Error("the toolbar is shown for a correct word")
	}
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("test")) {
		t.Error("a suggestion is shown for a correct word")
	}
	if !state.SelectionOverlay().HandlesAreVisible() {
		t.Error("the handles are hidden")
	}
}

func TestEditableTextMisspelledSelectionColor(t *testing.T) {
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	misspelled := ui.NewColor(0xFF00FF00)
	selection := ui.NewColor(0xFF112233)
	tests := []struct {
		name  string
		color foundation.Opt[ui.Color]
		want  ui.Color
	}{
		{"the color of the configuration", foundation.Some(misspelled), misspelled},
		{"the selection color of the field when there is none", foundation.None[ui.Color](), selection},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t, useSpellChecker(t))
			controller, _ := spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{Text: "tset test",
				Selection: foundation.Some(services.TextSelection{BaseOffset: 0, ExtentOffset: 4, Affinity: ui.TextAffinityUpstream})}),
				widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{
					MisspelledTextStyle: TextFieldMaterialMisspelledTextStyle, MisspelledSelectionColor: tc.color,
					SpellCheckSuggestionsToolbarBuilder: TextFieldDefaultSpellCheckSuggestionsToolbarBuilder}),
				func(o *widgets.EditableTextOpts) { o.SelectionColor = foundation.Some(selection) })
			state := editableState(ts)
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "tset test", SuggestionSpans: []services.SuggestionSpan{span(0, 4, "test")}})
			editableOf(ts).SelectWordsInRange(ui.Offset{}, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
			ts.PumpAndSettle()
			if got, ok := editableOf(ts).SelectionColor().Get(); !ok || got != selection {
				t.Fatalf("selection color = %v, want %v before the toolbar", got, selection)
			}
			if !state.ShowSpellCheckSuggestionsToolbar() {
				t.Fatal("the toolbar is not shown")
			}
			ts.PumpAndSettle()
			// The field builds again, as it does when the user moves the selection.
			controller.SetSelection(services.TextSelection{BaseOffset: 0, ExtentOffset: 3})
			ts.PumpAndSettle()
			if got, ok := editableOf(ts).SelectionColor().Get(); !ok || got != tc.want {
				t.Errorf("selection color = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextFieldBuildsTheDefaultSpellCheckToolbarOfThePlatform(t *testing.T) {
	// Flutter builds a CupertinoSpellCheckSuggestionsToolbar on iOS and macOS; here
	// every platform builds the Material one.
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformLinux} {
		t.Run(platform.String(), func(t *testing.T) {
			withPlatform(t, platform)
			ts := newToolbarTester(t)
			var builderContext widgets.BuildContext
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				builderContext = bc
				return NewTextField(TextFieldOpts{Autofocus: true, SpellCheckConfiguration: widgets.NewSpellCheckConfiguration(
					widgets.SpellCheckConfigurationOpts{SpellCheckService: &spellCheckService{}})})
			}))})}))
			// Allow the autofocus to take effect.
			ts.Pump()
			state := editableState(ts)
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "", SuggestionSpans: []services.SuggestionSpan{span(0, 0, "something")}})
			if _, ok := TextFieldDefaultSpellCheckSuggestionsToolbarBuilder(builderContext, state).(*SpellCheckSuggestionsToolbar); !ok {
				t.Errorf("the toolbar is a %T, want *SpellCheckSuggestionsToolbar", TextFieldDefaultSpellCheckSuggestionsToolbarBuilder(builderContext, state))
			}
		})
	}
}

func TestTextFieldBuildsTheDefaultSpellCheckConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		platform     foundation.TargetPlatform
		spellChecker bool
	}{
		// The default service is the platform's spell checker, on Android.
		{"Android with the default service", foundation.TargetPlatformAndroid, true},
		// Elsewhere the field needs a service of its own.
		{"iOS with a service", foundation.TargetPlatformIOS, false},
		{"Linux with a service", foundation.TargetPlatformLinux, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withPlatform(t, tc.platform)
			var o gluttertest.Opts
			var service services.SpellCheckService
			if tc.spellChecker {
				o.SpellChecker = func(string, string) []ui.SpellCheckSpan { return nil }
			} else {
				service = &spellCheckService{}
			}
			ts := newToolbarTester(t, o)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				NewTextField(TextFieldOpts{Autofocus: true, SpellCheckConfiguration: widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{SpellCheckService: service})}))})}))
			config := editableState(ts).SpellCheckConfiguration()
			if !config.SpellCheckEnabled() {
				t.Fatal("spell check is disabled")
			}
			if !config.MisspelledTextStyle().Equal(TextFieldMaterialMisspelledTextStyle) {
				t.Errorf("misspelled text style = %v, want the Material one", config.MisspelledTextStyle())
			}
			if config.MisspelledSelectionColor().IsSome() {
				t.Errorf("misspelled selection color = %v, want none", config.MisspelledSelectionColor())
			}
			if tc.spellChecker {
				if _, ok := config.SpellCheckService().(*services.DefaultSpellCheckService); !ok {
					t.Errorf("service = %T, want *DefaultSpellCheckService", config.SpellCheckService())
				}
			} else if config.SpellCheckService() != service {
				t.Errorf("service = %T, want the one given", config.SpellCheckService())
			}
			got := reflect.ValueOf(config.SpellCheckSuggestionsToolbarBuilder()).Pointer()
			if want := reflect.ValueOf(TextFieldDefaultSpellCheckSuggestionsToolbarBuilder).Pointer(); got != want {
				t.Error("the toolbar builder is not the default one")
			}
		})
	}
}

func TestTextFieldInferAndroidSpellCheckConfiguration(t *testing.T) {
	custom := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(30.0)})
	tests := []struct {
		name      string
		in        *widgets.SpellCheckConfiguration
		wantOn    bool
		wantStyle *painting.TextStyle
	}{
		{"nil gives the disabled one", nil, false, nil},
		{"disabled gives the disabled one", widgets.NewSpellCheckConfigurationDisabled(), false, nil},
		{"the missing style is the Material one", widgets.NewSpellCheckConfiguration(), true, TextFieldMaterialMisspelledTextStyle},
		{"a custom style is kept", widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{MisspelledTextStyle: custom}), true, custom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TextFieldInferAndroidSpellCheckConfiguration(tc.in)
			if got.SpellCheckEnabled() != tc.wantOn {
				t.Fatalf("enabled = %v, want %v", got.SpellCheckEnabled(), tc.wantOn)
			}
			if tc.wantOn && (!got.MisspelledTextStyle().Equal(tc.wantStyle) || got.SpellCheckSuggestionsToolbarBuilder() == nil) {
				t.Errorf("configuration = %v", got)
			}
		})
	}
}

// TestTextFieldSpellCheckOnAndroid is the whole path on a phone: the text is typed,
// the engine's spell checker finds the misspelled word, it is marked, a tap on it
// shows the toolbar, and a suggestion replaces the word.
func TestTextFieldSpellCheckOnAndroid(t *testing.T) {
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	withPlatform(t, foundation.TargetPlatformAndroid)
	var requests []string
	ts := newToolbarTester(t, gluttertest.Opts{SpellChecker: func(tag, text string) []ui.SpellCheckSpan {
		requests = append(requests, tag+" "+text)
		if text == "tset" {
			return []ui.SpellCheckSpan{{Start: 0, End: 4, Suggestions: []string{"test", "set"}}}
		}
		return nil
	}})
	controller := widgets.NewTextEditingController("")
	t.Cleanup(controller.Dispose)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		NewTextField(TextFieldOpts{Controller: controller, SpellCheckConfiguration: widgets.NewSpellCheckConfiguration()}))})}))
	state := editableState(ts)
	if !state.SpellCheckEnabled() {
		t.Fatal("spell check is disabled on Android with a spell checker")
	}

	ts.EnterText(findField(), "tset")
	ts.PumpAndSettle()
	if want := []string{"en-US tset"}; !slices.Equal(requests, want) {
		t.Fatalf("requests = %q, want %q", requests, want)
	}
	if r := state.SpellCheckResults(); r == nil || len(r.SuggestionSpans) != 1 {
		t.Fatalf("results = %v, want one misspelled word", r)
	}
	tree := editableOf(ts).Text().(*painting.TextSpan)
	if len(tree.Children) != 1 || tree.Children[0].(*painting.TextSpan).Style().DecorationStyle != ui.TextDecorationStyleWavy {
		t.Errorf("tree = %v, want the word marked with a wavy underline", tree)
	}

	// A tap on the word shows the suggestions.
	ts.TapAt(fieldTextOffsetToPosition(ts, 2))
	ts.PumpAndSettle()
	for _, label := range []string{"test", "set", "DELETE"} {
		if got := ts.Count(gluttertest.ByText(label)); got != 1 {
			t.Fatalf("%q found %d times, want 1", label, got)
		}
	}
	if state.SelectionOverlay().HandlesAreVisible() {
		t.Error("the handles are visible with the spell check toolbar")
	}
	ts.Tap(gluttertest.ByText("test"))
	ts.PumpAndSettle()
	if controller.Text() != "test" {
		t.Errorf("text = %q, want test", controller.Text())
	}
	if ts.Exists(gluttertest.ByText("DELETE")) {
		t.Error("the toolbar is still shown")
	}
}

// TestTextFieldSpellCheckOnIOS is the tap on a misspelled word of the Apple
// platforms: it shows the suggestions once, and the next taps on the word toggle
// the toolbar. The service is the app's, as the platform has no spell checker.
func TestTextFieldSpellCheckOnIOS(t *testing.T) {
	if foundation.IsWeb {
		t.Skip("[intended] the spell check toolbar is not shown on the web: the browser has its own context menu")
	}
	withPlatform(t, foundation.TargetPlatformIOS)
	ts := newToolbarTester(t)
	service := &spellCheckService{spans: []services.SuggestionSpan{span(0, 4, "test")}}
	controller := widgets.NewTextEditingController("")
	t.Cleanup(controller.Dispose)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		NewTextField(TextFieldOpts{Controller: controller, SpellCheckConfiguration: widgets.NewSpellCheckConfiguration(
			widgets.SpellCheckConfigurationOpts{SpellCheckService: service})}))})}))
	state := editableState(ts)

	ts.EnterText(findField(), "tset")
	service.flush()
	ts.PumpAndSettle()
	if r := state.SpellCheckResults(); r == nil || len(r.SuggestionSpans) != 1 {
		t.Fatalf("results = %v, want one misspelled word", r)
	}

	tapWord := func() {
		ts.TapAt(fieldTextOffsetToPosition(ts, 2))
		ts.PumpAndSettle()
	}
	tapWord()
	if got := ts.Count(gluttertest.ByText("test")); got != 1 {
		t.Fatalf("test found %d times after the first tap, want 1", got)
	}
	if !state.SelectionOverlay().SpellCheckToolbarIsVisible() {
		t.Error("the spell check toolbar is not the one that is visible")
	}
	// A second tap on the same word toggles the toolbar off.
	tapWord()
	if ts.Exists(gluttertest.ByText("test")) {
		t.Error("the toolbar is still shown after the second tap")
	}
}

func TestSpellCheckSuggestionsToolbarReplaceNotAllowed(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts func(*widgets.EditableTextOpts)
	}{
		{"a read-only field", func(o *widgets.EditableTextOpts) { o.ReadOnly = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newToolbarTester(t)
			spellCheckEditable(t, ts, services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: "hello", Selection: foundation.Some(services.NewTextSelectionCollapsed(0))}), spellCheckConfig(&spellCheckService{}), tc.opts)
			state := editableState(ts)
			// The buttons are built from results that were set, so the replacement is
			// what is refused.
			state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "hello", SuggestionSpans: []services.SuggestionSpan{span(0, 5, "world")}})
			items, ok := SpellCheckSuggestionsToolbarBuildButtonItems(state)
			if !ok {
				t.Fatal("no buttons")
			}
			var v any
			func() {
				defer func() { v = recover() }()
				items[0].OnPressed()
			}()
			err, _ := v.(error)
			appErr, isErr := errors.FromError(err)
			if !isErr {
				t.Fatalf("expected *errors.Error, got %T", v)
			}
			if !appErr.HasRoute("spellCheckSuggestionsToolbarReplaceText.SpellCheckSuggestionsToolbarReplaceNotAllowed") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
