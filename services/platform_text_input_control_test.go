package services_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// session is a text input over a fake backend with a client attached and the
// input method shown.
type session struct {
	ti      *services.TextInput
	backend *fakeBackend
	client  *fakeClient
	conn    *services.TextInputConnection
}

func newSession(t *testing.T, cfg services.TextInputConfiguration, value services.TextEditingValue) *session {
	t.Helper()
	s := &session{backend: &fakeBackend{}, client: newFakeClient(value)}
	s.ti = services.NewTextInput(services.TextInputOpts{Backend: s.backend})
	s.conn = s.ti.Attach(s.client, cfg)
	s.conn.SetEditingState(value)
	s.conn.Show()
	return s
}

func caretAt(text string, offset int) services.TextEditingValue {
	return val(text, services.NewTextSelectionCollapsed(offset))
}

func TestPlatformControlTurnsTheInputMethodOnAndOff(t *testing.T) {
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend})
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	if backend.last().Enabled {
		t.Error("the input method is on before Show")
	}
	conn.SetEditingState(caretAt("abc", 3))
	conn.Show()
	got := backend.last()
	if !got.Enabled || got.Text != "abc" || got.SelectionAnchor != 3 || got.SelectionCursor != 3 {
		t.Errorf("state = %+v", got)
	}
	ti.HandleRequestExistingInputState() // attach again: still on
	if !backend.last().Enabled {
		t.Error("the input method went off")
	}
	conn.Close()
	if backend.last().Enabled {
		t.Error("the input method is on after Close")
	}
}

func TestPlatformControlStateContentTypes(t *testing.T) {
	tests := []struct {
		name    string
		cfg     services.TextInputConfigurationOpts
		purpose ui.TextInputPurpose
		hints   ui.TextInputHints
	}{
		{"text", services.TextInputConfigurationOpts{}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"no autocorrect, suggestions or learning", services.TextInputConfigurationOpts{DisableAutocorrect: true, DisableSuggestions: true, DisableIMEPersonalizedLearning: true},
			ui.TextInputPurposeUnspecified, ui.TextInputHintSensitiveData},
		{"multiline", services.TextInputConfigurationOpts{InputType: services.TextInputTypeMultiline}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintMultiline},
		{"number", services.TextInputConfigurationOpts{InputType: services.TextInputTypeNumber}, ui.TextInputPurposeDigits,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"signed number", services.TextInputConfigurationOpts{InputType: services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberOpts{Signed: true})},
			ui.TextInputPurposeNumber, ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"decimal number", services.TextInputConfigurationOpts{InputType: services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberOpts{Decimal: true})},
			ui.TextInputPurposeNumber, ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"phone", services.TextInputConfigurationOpts{InputType: services.TextInputTypePhone}, ui.TextInputPurposePhone,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"datetime", services.TextInputConfigurationOpts{InputType: services.TextInputTypeDatetime}, ui.TextInputPurposeDatetime,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"email", services.TextInputConfigurationOpts{InputType: services.TextInputTypeEmailAddress}, ui.TextInputPurposeEmail,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"url", services.TextInputConfigurationOpts{InputType: services.TextInputTypeURL}, ui.TextInputPurposeURL,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"visible password", services.TextInputConfigurationOpts{InputType: services.TextInputTypeVisiblePassword}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintSensitiveData},
		{"name", services.TextInputConfigurationOpts{InputType: services.TextInputTypeName}, ui.TextInputPurposeName,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion},
		{"obscured text", services.TextInputConfigurationOpts{ObscureText: true}, ui.TextInputPurposePassword,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintHiddenText | ui.TextInputHintSensitiveData},
		{"capitalize sentences", services.TextInputConfigurationOpts{TextCapitalization: services.TextCapitalizationSentences}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintAutoCapitalization},
		{"capitalize words", services.TextInputConfigurationOpts{TextCapitalization: services.TextCapitalizationWords}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintTitlecase},
		{"capitalize characters", services.TextInputConfigurationOpts{TextCapitalization: services.TextCapitalizationCharacters}, ui.TextInputPurposeUnspecified,
			ui.TextInputHintSpellcheck | ui.TextInputHintCompletion | ui.TextInputHintUppercase},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, services.NewTextInputConfiguration(tc.cfg), caretAt("a", 1))
			got := s.backend.last()
			if !got.Enabled || got.Purpose != tc.purpose || got.Hints != tc.hints {
				t.Errorf("purpose = %v hints = %#x, want %v %#x", got.Purpose, uint32(got.Hints), tc.purpose, uint32(tc.hints))
			}
		})
	}
}

func TestPlatformControlStaysOffForTheTypeNone(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(services.TextInputConfigurationOpts{InputType: services.TextInputTypeNone}), caretAt("a", 1))
	if s.backend.last().Enabled {
		t.Error("the input method is on for a field of the type none")
	}
}

func TestPlatformControlStaysOffWhileTheAppHasItsOwnControl(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(), caretAt("a", 1))
	if !s.backend.last().Enabled {
		t.Fatal("the input method is off")
	}
	s.ti.SetInputControl(&fakeControl{})
	s.conn.SetEditingState(caretAt("ab", 2))
	if s.backend.last().Enabled {
		t.Error("the input method is on with a control of the app")
	}
	s.ti.RestorePlatformInputControl()
	s.conn.SetEditingState(caretAt("abc", 3))
	if !s.backend.last().Enabled {
		t.Error("the input method is off after the platform control was restored")
	}
}

func TestPlatformControlStateViewAndConfigUpdate(t *testing.T) {
	cfg := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{ViewID: foundation.Some(7)})
	s := newSession(t, cfg, caretAt("a", 1))
	if got := s.backend.last().View; got != 7 {
		t.Errorf("view = %v, want 7", got)
	}
	s.conn.UpdateConfig(cfg.CopyWith(services.TextInputConfigurationCopy{InputType: foundation.Some(services.TextInputTypeEmailAddress)}))
	if got := s.backend.last().Purpose; got != ui.TextInputPurposeEmail {
		t.Errorf("purpose after UpdateConfig = %v", got)
	}
	s.conn.UpdateConfig(cfg.CopyWith(services.TextInputConfigurationCopy{InputType: foundation.Some(services.TextInputTypeNone)}))
	if s.backend.last().Enabled {
		t.Error("the input method stays on after the type became none")
	}
}

func TestPlatformControlCaretRectIsInTheViewsCoordinates(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(), caretAt("a", 1))
	m := ui.NewMatrix4Identity()
	m.Translate(100, 50, 0)
	s.conn.SetEditableSizeAndTransform(ui.Size{Width: 200, Height: 30}, m)
	s.conn.SetCaretRect(ui.Rect{Left: 4, Top: 2, Right: 5, Bottom: 18})
	if got, want := s.backend.last().CaretRect, (ui.Rect{Left: 104, Top: 52, Right: 105, Bottom: 68}); got != want {
		t.Errorf("caret = %v, want %v", got, want)
	}
	// A rectangle that is not finite is none.
	s.conn.SetCaretRect(ui.Rect{Left: 0, Top: 0, Right: 0, Bottom: 0}.Shift(ui.Offset{Dx: 1, Dy: 1}))
	if !s.backend.last().Enabled {
		t.Error("the input method went off")
	}
}

func TestPlatformControlSurroundingTextLeavesOutTheComposingText(t *testing.T) {
	tests := []struct {
		name           string
		value          services.TextEditingValue
		text           string
		anchor, cursor int
	}{
		{"no composing text", caretAt("hello", 5), "hello", 5, 5},
		{"composing text before the caret", services.NewTextEditingValue(services.TextEditingValueOpts{
			Text: "hi nihao", Selection: foundation.Some(services.NewTextSelectionCollapsed(8)), Composing: foundation.Some(ui.TextRange{Start: 3, End: 8})}),
			"hi ", 3, 3},
		{"the caret inside the composing text", services.NewTextEditingValue(services.TextEditingValueOpts{
			Text: "hi nihao", Selection: foundation.Some(services.NewTextSelectionCollapsed(5)), Composing: foundation.Some(ui.TextRange{Start: 3, End: 8})}),
			"hi ", 3, 3},
		{"text after the composing text", services.NewTextEditingValue(services.TextEditingValueOpts{
			Text: "hi ni!", Selection: foundation.Some(services.NewTextSelectionCollapsed(5)), Composing: foundation.Some(ui.TextRange{Start: 3, End: 5})}),
			"hi !", 3, 3},
		{"a selection", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello", Selection: foundation.Some(services.NewTextSelection(1, 4))}),
			"hello", 1, 4},
		{"no selection", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello"}), "hello", 0, 0},
		{"a composing range outside the text is ignored", services.NewTextEditingValue(services.TextEditingValueOpts{
			Text: "ab", Selection: foundation.Some(services.NewTextSelectionCollapsed(2)), Composing: foundation.Some(ui.TextRange{Start: 1, End: 9})}),
			"ab", 2, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, services.NewTextInputConfiguration(), tc.value)
			got := s.backend.last()
			if got.Text != tc.text || got.SelectionAnchor != tc.anchor || got.SelectionCursor != tc.cursor {
				t.Errorf("state = %q %d %d, want %q %d %d", got.Text, got.SelectionAnchor, got.SelectionCursor, tc.text, tc.anchor, tc.cursor)
			}
		})
	}
}

func TestPlatformControlAppliesTheEventsOfTheInputMethod(t *testing.T) {
	collapsed := services.NewTextSelectionCollapsed
	composing := func(text string, caret int, c ui.TextRange) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(collapsed(caret)), Composing: foundation.Some(c)})
	}
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	tests := []struct {
		name  string
		start services.TextEditingValue
		event ui.TextInputEvent
		want  services.TextEditingValue
	}{
		{"commit at the caret", caretAt("ab", 1), ui.TextInputEvent{Commit: "XY"}, composing("aXYb", 3, ui.TextRangeEmpty)},
		{"commit in place of the selection", val("abcd", services.NewTextSelection(1, 3)), ui.TextInputEvent{Commit: "X"}, composing("aXd", 2, ui.TextRangeEmpty)},
		{"commit a character of several bytes", caretAt("a", 1), ui.TextInputEvent{Commit: "你"}, composing("a你", 4, ui.TextRangeEmpty)},
		{"preedit", caretAt("ab", 2), ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2},
			composing("abni", 4, rng(2, 4))},
		{"preedit with the cursor inside", caretAt("ab", 2), ui.TextInputEvent{Preedit: "nihao", PreeditCursorBegin: 2, PreeditCursorEnd: 4},
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "abnihao", Selection: foundation.Some(services.NewTextSelection(4, 6)), Composing: foundation.Some(rng(2, 7))})},
		{"preedit with the cursor hidden", caretAt("ab", 2), ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
			composing("abni", 4, rng(2, 4))},
		{"preedit in place of the selection", val("abcd", services.NewTextSelection(1, 3)), ui.TextInputEvent{Preedit: "n", PreeditCursorBegin: 1, PreeditCursorEnd: 1},
			composing("and", 2, rng(1, 2))},
		{"the next preedit replaces the old one", composing("abni", 4, rng(2, 4)), ui.TextInputEvent{Preedit: "nih", PreeditCursorBegin: 3, PreeditCursorEnd: 3},
			composing("abnih", 5, rng(2, 5))},
		{"commit replaces the preedit", composing("abni", 4, rng(2, 4)), ui.TextInputEvent{Commit: "你", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
			composing("ab你", 5, ui.TextRangeEmpty)},
		{"commit and a new preedit", composing("abni", 4, rng(2, 4)), ui.TextInputEvent{Commit: "你", Preedit: "h", PreeditCursorBegin: 1, PreeditCursorEnd: 1},
			composing("ab你h", 6, rng(5, 6))},
		{"the preedit goes", composing("abni", 4, rng(2, 4)), ui.TextInputEvent{PreeditCursorBegin: -1, PreeditCursorEnd: -1},
			composing("ab", 2, ui.TextRangeEmpty)},
		{"delete around the caret", caretAt("abcdef", 3), ui.TextInputEvent{DeleteBefore: 2, DeleteAfter: 1}, composing("aef", 1, ui.TextRangeEmpty)},
		{"delete before the caret", caretAt("abcdef", 3), ui.TextInputEvent{DeleteBefore: 1}, composing("abdef", 2, ui.TextRangeEmpty)},
		{"delete past the start and the end", caretAt("abc", 1), ui.TextInputEvent{DeleteBefore: 9, DeleteAfter: 9}, composing("", 0, ui.TextRangeEmpty)},
		{"delete leaves the selection", val("abcdef", services.NewTextSelection(2, 4)), ui.TextInputEvent{DeleteBefore: 1, DeleteAfter: 1},
			services.NewTextEditingValue(services.TextEditingValueOpts{Text: "acdf", Selection: foundation.Some(services.NewTextSelection(1, 3))})},
		{"delete a whole character of several bytes", caretAt("a你b", 4), ui.TextInputEvent{DeleteBefore: 1}, composing("ab", 1, ui.TextRangeEmpty)},
		{"delete then commit", caretAt("abc", 3), ui.TextInputEvent{DeleteBefore: 1, Commit: "X"}, composing("abX", 3, ui.TextRangeEmpty)},
		{"delete counts from the start of the preedit", composing("abni", 4, rng(2, 4)), ui.TextInputEvent{DeleteBefore: 1, Commit: "X", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
			composing("aX", 2, ui.TextRangeEmpty)},
		{"a value without a selection", services.NewTextEditingValue(services.TextEditingValueOpts{Text: "ab"}), ui.TextInputEvent{Commit: "X"},
			composing("abX", 3, ui.TextRangeEmpty)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, services.NewTextInputConfiguration(), tc.start)
			s.ti.HandleTextInput(tc.event)
			if len(s.client.values) != 1 {
				t.Fatalf("the client got %d values", len(s.client.values))
			}
			if got := s.client.values[0]; !got.Equal(tc.want) {
				t.Errorf("value = %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestPlatformControlEventsAreRememberedForTheNext(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(), caretAt("", 0))
	s.ti.HandleTextInput(ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2})
	s.ti.HandleTextInput(ui.TextInputEvent{Commit: "你", PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	want := []string{"ni", "你"}
	var got []string
	for _, v := range s.client.values {
		got = append(got, v.Text)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("texts = %q, want %q", got, want)
	}
	// And the platform was told the new state, from the input method.
	last := s.backend.last()
	if last.Text != "你" || !last.CausedByInputMethod {
		t.Errorf("state = %+v", last)
	}
	// The app's own change is not caused by the input method.
	s.conn.SetEditingState(caretAt("你x", 4))
	if s.backend.last().CausedByInputMethod {
		t.Error("a change of the app is marked as the input method's")
	}
}

func TestPlatformControlSendsStateWithoutTheComposingText(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(), caretAt("ab", 2))
	s.ti.HandleTextInput(ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2})
	if got := s.backend.last(); got.Text != "ab" || got.SelectionCursor != 2 {
		t.Errorf("state = %+v, want the text without the preedit", got)
	}
}

func TestPlatformControlIgnoresEventsWithoutAClient(t *testing.T) {
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend})
	ti.HandleTextInput(ui.TextInputEvent{Commit: "x"}) // must not panic
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	conn.Close()
	ti.HandleTextInput(ui.TextInputEvent{Commit: "x"})
	if len(client.values) != 0 {
		t.Errorf("a client that was closed got %v", client.values)
	}
}

func TestPlatformControlOtherControlsHearOfTheEvents(t *testing.T) {
	s := newSession(t, services.NewTextInputConfiguration(), caretAt("", 0))
	other := &fakeControl{}
	s.ti.SetInputControl(other)
	other.calls = nil
	s.ti.HandleTextInput(ui.TextInputEvent{Commit: "x"})
	if want := []string{"setEditingState"}; !reflect.DeepEqual(other.calls, want) {
		t.Errorf("calls = %v, want %v", other.calls, want)
	}
	if len(s.client.values) != 1 {
		t.Errorf("client values = %v", s.client.values)
	}
}

func TestPlatformControlDeliversDeltasToADeltaClient(t *testing.T) {
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend})
	client := fakeDeltaClient{newFakeClient(caretAt("ab", 2))}
	cfg := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{EnableDeltaModel: true})
	conn := ti.Attach(client, cfg)
	conn.SetEditingState(caretAt("ab", 2))
	conn.Show()

	tests := []struct {
		name  string
		event ui.TextInputEvent
		want  services.TextEditingDelta
	}{
		{"commit is an insertion", ui.TextInputEvent{Commit: "X"},
			services.NewTextEditingDeltaInsertion("ab", "X", 2, services.NewTextSelectionCollapsed(3), ui.TextRangeEmpty)},
		{"delete is a deletion", ui.TextInputEvent{DeleteBefore: 1},
			services.NewTextEditingDeltaDeletion("abX", ui.TextRange{Start: 2, End: 3}, services.NewTextSelectionCollapsed(2), ui.TextRangeEmpty)},
		{"preedit is an insertion with a composing range", ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2},
			services.NewTextEditingDeltaInsertion("ab", "ni", 2, services.NewTextSelectionCollapsed(4), ui.TextRange{Start: 2, End: 4})},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "delete is a deletion" {
				// Start from "abX" with the caret after the X.
				conn.SetEditingState(caretAt("abX", 3))
			}
			if tc.name == "preedit is an insertion with a composing range" {
				conn.SetEditingState(caretAt("ab", 2))
			}
			ti.HandleTextInput(tc.event)
			if len(client.deltas) != i+1 || len(client.deltas[i]) != 1 {
				t.Fatalf("deltas = %v", client.deltas)
			}
			if got := client.deltas[i][0]; !reflect.DeepEqual(got, tc.want) {
				t.Errorf("delta = %#v\nwant %#v", got, tc.want)
			}
		})
	}
	if len(client.values) != 0 {
		t.Errorf("a delta client also got the values %v", client.values)
	}
}

func TestPlatformControlDeltaOfAPreeditThatChangesTheText(t *testing.T) {
	ti := services.NewTextInput()
	client := fakeDeltaClient{newFakeClient(services.TextEditingValueEmpty)}
	conn := ti.Attach(client, services.NewTextInputConfiguration(services.TextInputConfigurationOpts{EnableDeltaModel: true}))
	conn.SetEditingState(val("hello ni", services.NewTextSelectionCollapsed(8)))
	// The preedit "ni" is replaced by "nihao": the old text is applied, so the delta must be
	// a change of the text that apply turns into the new value.
	conn.SetEditingState(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "hello ni", Selection: foundation.Some(services.NewTextSelectionCollapsed(8)), Composing: foundation.Some(ui.TextRange{Start: 6, End: 8})}))
	ti.HandleTextInput(ui.TextInputEvent{Preedit: "nihao", PreeditCursorBegin: 5, PreeditCursorEnd: 5})
	if len(client.deltas) != 1 {
		t.Fatalf("deltas = %v", client.deltas)
	}
	got := client.deltas[0][0].Apply(services.TextEditingValueEmpty)
	want := services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "hello nihao", Selection: foundation.Some(services.NewTextSelectionCollapsed(11)), Composing: foundation.Some(ui.TextRange{Start: 6, End: 11})})
	if !got.Equal(want) {
		t.Errorf("applied = %v\nwant %v", got, want)
	}
}

func TestPlatformControlDeltaOfANonTextEvent(t *testing.T) {
	ti := services.NewTextInput()
	client := fakeDeltaClient{newFakeClient(services.TextEditingValueEmpty)}
	conn := ti.Attach(client, services.NewTextInputConfiguration(services.TextInputConfigurationOpts{EnableDeltaModel: true}))
	conn.SetEditingState(caretAt("abc", 3))
	ti.HandleTextInput(ui.TextInputEvent{PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	if len(client.deltas) != 1 {
		t.Fatalf("deltas = %v", client.deltas)
	}
	if _, ok := client.deltas[0][0].(services.TextEditingDeltaNonTextUpdate); !ok {
		t.Errorf("delta = %#v, want a non text update", client.deltas[0][0])
	}
}

func TestPlatformControlDeltaWithMultibyteText(t *testing.T) {
	ti := services.NewTextInput()
	client := fakeDeltaClient{newFakeClient(services.TextEditingValueEmpty)}
	conn := ti.Attach(client, services.NewTextInputConfiguration(services.TextInputConfigurationOpts{EnableDeltaModel: true}))
	conn.SetEditingState(caretAt("é你é", 7))
	ti.HandleTextInput(ui.TextInputEvent{DeleteBefore: 2})
	if len(client.deltas) != 1 {
		t.Fatalf("deltas = %v", client.deltas)
	}
	got := client.deltas[0][0].Apply(services.TextEditingValueEmpty)
	if got.Text != "é你" {
		t.Errorf("applied text = %q, want é你", got.Text)
	}
}

func TestHandleUnhandledKeyTypesOrActs(t *testing.T) {
	key := func(logical services.LogicalKeyboardKey, char string, typ ui.KeyEventType) ui.KeyData {
		return ui.KeyData{Type: typ, Logical: uint64(logical), Character: char}
	}
	multiline := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{InputType: services.TextInputTypeMultiline})
	tests := []struct {
		name        string
		cfg         services.TextInputConfiguration
		data        ui.KeyData
		controlHeld bool
		wantText    string // "" when the client gets no value
		wantActions []services.TextInputAction
	}{
		{"a character is typed", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyKeyC, "c", ui.KeyEventTypeDown), false, "abc", nil},
		{"a repeat is typed", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyKeyC, "c", ui.KeyEventTypeRepeat), false, "abc", nil},
		{"a character of several bytes", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyKeyC, "你", ui.KeyEventTypeDown), false, "ab你", nil},
		{"a release is ignored", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyKeyC, "c", ui.KeyEventTypeUp), false, "", nil},
		{"a key with control held is ignored", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyKeyC, "c", ui.KeyEventTypeDown), true, "", nil},
		{"a key without a character is ignored", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyArrowLeft, "", ui.KeyEventTypeDown), false, "", nil},
		{"a control character is ignored", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyBackspace, "\b", ui.KeyEventTypeDown), false, "", nil},
		{"enter is done in a single line field", services.NewTextInputConfiguration(), key(services.LogicalKeyboardKeyEnter, "\r", ui.KeyEventTypeDown), false, "",
			[]services.TextInputAction{services.TextInputActionDone}},
		{"enter is the action of the field", services.NewTextInputConfiguration(services.TextInputConfigurationOpts{InputAction: services.TextInputActionSearch}),
			key(services.LogicalKeyboardKeyNumpadEnter, "", ui.KeyEventTypeDown), false, "", []services.TextInputAction{services.TextInputActionSearch}},
		{"enter is a new line in a multiline field", multiline, key(services.LogicalKeyboardKeyEnter, "\r", ui.KeyEventTypeDown), false, "ab\n",
			[]services.TextInputAction{services.TextInputActionNewline}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, tc.cfg, caretAt("ab", 2))
			s.ti.HandleUnhandledKey(tc.data, tc.controlHeld)
			if tc.wantText == "" {
				if len(s.client.values) != 0 {
					t.Errorf("the client got %v", s.client.values)
				}
			} else if len(s.client.values) != 1 || s.client.values[0].Text != tc.wantText {
				t.Errorf("the client got %v, want the text %q", s.client.values, tc.wantText)
			}
			if !reflect.DeepEqual(s.client.actions, tc.wantActions) {
				t.Errorf("actions = %v, want %v", s.client.actions, tc.wantActions)
			}
		})
	}
	t.Run("without a client nothing happens", func(t *testing.T) {
		ti := services.NewTextInput(services.TextInputOpts{Backend: &fakeBackend{}})
		ti.HandleUnhandledKey(key(services.LogicalKeyboardKeyKeyC, "c", ui.KeyEventTypeDown), false)
	})
}
