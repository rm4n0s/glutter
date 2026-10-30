package android

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestEditorConfigOf(t *testing.T) {
	const (
		multi   = ui.TextInputHintMultiline
		suggest = ui.TextInputHintCompletion
	)
	tests := []struct {
		name  string
		state ui.TextInputState
		want  editorConfig
	}{
		{"a plain field", ui.TextInputState{Hints: suggest},
			editorConfig{inputType: typeClassText, imeOptions: imeActionDone}},
		{"autocorrect and sentences", ui.TextInputState{Hints: suggest | ui.TextInputHintSpellcheck | ui.TextInputHintAutoCapitalization},
			editorConfig{inputType: typeClassText | typeTextFlagAutoCorrect | typeTextFlagCapSentences, imeOptions: imeActionDone}},
		{"words and characters", ui.TextInputState{Hints: suggest | ui.TextInputHintTitlecase},
			editorConfig{inputType: typeClassText | typeTextFlagCapWords, imeOptions: imeActionDone}},
		{"characters", ui.TextInputState{Hints: suggest | ui.TextInputHintUppercase},
			editorConfig{inputType: typeClassText | typeTextFlagCapCharacters, imeOptions: imeActionDone}},
		{"no suggestions is a visible password", ui.TextInputState{},
			editorConfig{inputType: typeClassText | typeTextFlagNoSuggestions | typeTextVariationVisiblePassword, imeOptions: imeActionDone}},
		{"multiline has no action", ui.TextInputState{Hints: suggest | multi},
			editorConfig{inputType: typeClassText | typeTextFlagMultiLine, imeOptions: imeActionNone, multiline: true}},
		{"a password", ui.TextInputState{Purpose: ui.TextInputPurposePassword, Hints: ui.TextInputHintHiddenText | ui.TextInputHintSensitiveData},
			editorConfig{inputType: typeClassText | typeTextFlagNoSuggestions | typeTextVariationPassword, imeOptions: imeActionDone | imeFlagNoPersonalizedLearning}},
		{"an email address, next", ui.TextInputState{Purpose: ui.TextInputPurposeEmail, Hints: suggest, EnterKey: ui.TextInputEnterKeyNext},
			editorConfig{inputType: typeClassText | typeTextVariationEmailAddress, imeOptions: imeActionNext}},
		{"a url, go", ui.TextInputState{Purpose: ui.TextInputPurposeURL, Hints: suggest, EnterKey: ui.TextInputEnterKeyGo},
			editorConfig{inputType: typeClassText | typeTextVariationURI, imeOptions: imeActionGo}},
		{"a name, search", ui.TextInputState{Purpose: ui.TextInputPurposeName, Hints: suggest, EnterKey: ui.TextInputEnterKeySearch},
			editorConfig{inputType: typeClassText | typeTextVariationPersonName, imeOptions: imeActionSearch}},
		{"send and previous", ui.TextInputState{Hints: suggest, EnterKey: ui.TextInputEnterKeySend},
			editorConfig{inputType: typeClassText, imeOptions: imeActionSend}},
		{"previous", ui.TextInputState{Hints: suggest, EnterKey: ui.TextInputEnterKeyPrevious},
			editorConfig{inputType: typeClassText, imeOptions: imeActionPrevious}},
		{"enter in a single line field", ui.TextInputState{Hints: suggest, EnterKey: ui.TextInputEnterKeyEnter},
			editorConfig{inputType: typeClassText, imeOptions: imeActionNone}},
		{"digits", ui.TextInputState{Purpose: ui.TextInputPurposeDigits},
			editorConfig{inputType: typeClassNumber, imeOptions: imeActionDone}},
		{"a number with a sign and a decimal point", ui.TextInputState{Purpose: ui.TextInputPurposeNumber},
			editorConfig{inputType: typeClassNumber | typeNumberFlagSigned | typeNumberFlagDecimal, imeOptions: imeActionDone}},
		{"a pin", ui.TextInputState{Purpose: ui.TextInputPurposePIN},
			editorConfig{inputType: typeClassNumber | typeNumberVariationPasswd, imeOptions: imeActionDone}},
		{"a phone number", ui.TextInputState{Purpose: ui.TextInputPurposePhone},
			editorConfig{inputType: typeClassPhone, imeOptions: imeActionDone}},
		{"a date and time", ui.TextInputState{Purpose: ui.TextInputPurposeDatetime},
			editorConfig{inputType: typeClassDatetime, imeOptions: imeActionDone}},
		{"a date", ui.TextInputState{Purpose: ui.TextInputPurposeDate},
			editorConfig{inputType: typeClassDatetime | typeDatetimeVariationDate, imeOptions: imeActionDone}},
		{"a time", ui.TextInputState{Purpose: ui.TextInputPurposeTime},
			editorConfig{inputType: typeClassDatetime | typeDatetimeVariationTime, imeOptions: imeActionDone}},
	}
	for _, tt := range tests {
		if got := editorConfigOf(tt.state); got != tt.want {
			t.Errorf("%s: editorConfigOf = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// textHandler is a handler that takes the text input at once, as the engine's does.
type textHandler struct {
	recorder
	syncCalls int
}

func (h *textHandler) OnKeySync(ui.KeyData) bool           { return false }
func (h *textHandler) OnPointerSync([]ui.PointerData)      {}
func (h *textHandler) OnTextInputSync(e ui.TextInputEvent) { h.syncCalls++; h.OnTextInput(e) }

// textPlatform is a platform with its window and a handler, not running.
func textPlatform(t *testing.T) (*Platform, *fakeHost, *textHandler) {
	t.Helper()
	host := newFakeHost()
	p := newPlatform(newBridge(), host, nil)
	if _, err := p.NewWindow(platform.WindowConfig{}); err != nil {
		t.Fatal(err)
	}
	h := &textHandler{}
	p.handler = h
	return p, host, h
}

func plainField(text string, anchor, cursor int) ui.TextInputState {
	return ui.TextInputState{Enabled: true, Hints: ui.TextInputHintCompletion, Text: text, SelectionAnchor: anchor, SelectionCursor: cursor}
}

// TestTextInputStartsWritesAndStops checks what the activity is told as the framework's text field
// gets the focus, changes and loses the focus.
func TestTextInputStartsWritesAndStops(t *testing.T) {
	p, host, _ := textPlatform(t)
	w := p.win
	w.SetTextInput(plainField("héllo", 3, 3))
	want := []command{
		{Op: opTextStart, A: typeClassText, B: imeActionDone},
		{Op: opTextState, A: 0, B: 2, C: 2, S: "héllo"},
	}
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("a new field: %+v, want %+v", got, want)
	}
	// The same state again writes nothing; a new text and caret write; the same configuration does not restart.
	w.SetTextInput(plainField("héllo", 3, 3))
	w.SetTextInput(plainField("héllo!", 7, 7))
	want = append(want, command{Op: opTextState, A: 0, B: 6, C: 6, S: "héllo!"})
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("a changed field: %+v, want %+v", got, want)
	}
	// Another configuration restarts the input method with a new editing state.
	f := plainField("x", 1, 1)
	f.Purpose = ui.TextInputPurposeEmail
	w.SetTextInput(f)
	want = append(want,
		command{Op: opTextStart, A: typeClassText | typeTextVariationEmailAddress, B: imeActionDone},
		command{Op: opTextState, A: 0, B: 1, C: 1, S: "x"})
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("a field of another kind: %+v, want %+v", got, want)
	}
	// No field: the input method stops, once.
	w.SetTextInput(ui.TextInputState{})
	w.SetTextInput(ui.TextInputState{})
	want = append(want, command{Op: opTextStop})
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("no field: %+v, want %+v", got, want)
	}
}

// TestTheInputMethodsEditsBecomeEvents has the edits that an input method makes (typing, composing, deleting, an
// emoji with a surrogate pair), in the UTF-16 units of the Java side, turn into the events of the contract in
// UTF-8, (the echo is checked below).
func TestTheInputMethodsEditsBecomeEvents(t *testing.T) {
	type edit struct {
		seq                              int
		text                             string
		base, extent, composeB, composeE int
	}
	tests := []struct {
		name   string
		field  ui.TextInputState
		edit   edit
		want   ui.TextInputEvent
		wantOK bool
	}{
		{"typing a letter", plainField("ab", 2, 2), edit{1, "abc", 3, 3, -1, -1},
			ui.TextInputEvent{Commit: "c", PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true},
		{"composing a letter", plainField("ab", 2, 2), edit{1, "abk", 3, 3, 2, 3},
			ui.TextInputEvent{Preedit: "k", PreeditCursorBegin: 1, PreeditCursorEnd: 1}, true},
		{"deleting before the caret", plainField("abc", 3, 3), edit{1, "ab", 2, 2, -1, -1},
			ui.TextInputEvent{DeleteBefore: 1, PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true},
		{"an emoji is one rune and two units", plainField("a", 1, 1), edit{1, "a😀", 3, 3, -1, -1},
			ui.TextInputEvent{Commit: "😀", PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true},
		{"replacing a selection", plainField("abcd", 1, 3), edit{1, "aXd", 2, 2, -1, -1},
			ui.TextInputEvent{Commit: "X", PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true},
		{"deleting a selection", plainField("abcd", 1, 3), edit{1, "ad", 1, 1, -1, -1},
			ui.TextInputEvent{DeleteSelection: true, PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true},
		{"a move of the caret alone is not sent", plainField("abc", 3, 3), edit{1, "abc", 1, 1, -1, -1}, ui.TextInputEvent{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _, h := textPlatform(t)
			p.win.SetTextInput(tt.field)
			p.textEdit(tt.edit.seq, tt.edit.text, tt.edit.base, tt.edit.extent, tt.edit.composeB, tt.edit.composeE)
			got := h.textEvents()
			if !tt.wantOK {
				if len(got) != 0 {
					t.Fatalf("an event was sent: %+v", got)
				}
				return
			}
			if len(got) != 1 || h.syncCalls != 1 {
				t.Fatalf("%d events, %d at once; want one, at once", len(got), h.syncCalls)
			}
			tt.want.View = p.win.id
			if got[0] != tt.want {
				t.Errorf("event %+v, want %+v", got[0], tt.want)
			}
		})
	}
}

// TestAnEchoOfTheFrameworkIsNotWrittenBack checks the exchange of the framework's state and the edits of the input
// method: the state that follows an older edit than the last one carries that edit's number (the Java side
// ignores it), the echo of an edit writes nothing, and a change of the app writes.
func TestAnEchoOfTheFrameworkIsNotWrittenBack(t *testing.T) {
	p, host, h := textPlatform(t)
	w := p.win
	w.SetTextInput(plainField("ab", 2, 2))
	start := len(host.sentCommands())
	p.textEdit(5, "abc", 3, 3, -1, -1)
	if got := h.textEvents(); len(got) != 1 || got[0].Commit != "c" {
		t.Fatalf("events %+v", got)
	}
	// The framework applied the event and says what its field holds now: the same as the editing state.
	w.SetTextInput(plainField("abc", 3, 3))
	if n := len(host.sentCommands()); n != start {
		t.Errorf("the echo was written back: %+v", host.sentCommands()[start:])
	}
	// The app changes the text: it is written, with the number of the last edit.
	w.SetTextInput(plainField("abc!", 4, 4))
	got := host.sentCommands()
	if len(got) != start+1 || got[start] != (command{Op: opTextState, A: 5, B: 4, C: 4, S: "abc!"}) {
		t.Errorf("the app's change: %+v", got[start:])
	}
	// An edit when no field has the focus is dropped.
	w.SetTextInput(ui.TextInputState{})
	before := len(h.textEvents())
	p.textEdit(6, "zzz", 3, 3, -1, -1)
	if len(h.textEvents()) != before {
		t.Error("an edit without a text field was sent")
	}
}

// TestTextEditsReachTheEngineThroughTheBridge runs the platform: an edit of the Java side is queued and
// handled on the UI thread.
func TestTextEditsReachTheEngineThroughTheBridge(t *testing.T) {
	h := newHarness(t)
	h.run()
	h.sync(func() { h.win.SetTextInput(plainField("", 0, 0)) })
	h.b.textEdit(1, "hi", 2, 2, -1, -1)
	waitFor(t, "the text event", func() bool { return len(h.h.textEvents()) == 1 })
	if got := h.h.textEvents()[0]; got.Commit != "hi" || got.View != h.win.ID() {
		t.Errorf("the event is %+v", got)
	}
}

func TestClipboardWritesAndReads(t *testing.T) {
	h := newHarness(t)
	h.run()
	clip := h.p.Clipboard()
	ctx := context.Background()

	done := make(chan error, 1)
	h.sync(func() { clip.SetText(ctx, "héllo", func(err error) { done <- err }) })
	if err := <-done; err != nil {
		t.Errorf("SetText: %v", err)
	}
	if got := h.host.sentCommands(); len(got) != 1 || got[0] != (command{Op: opClipboardSet, S: "héllo"}) {
		t.Fatalf("the commands are %+v", got)
	}

	type read struct {
		text string
		ok   bool
		err  error
	}
	reads := make(chan read, 1)
	h.sync(func() { clip.GetText(ctx, func(s string, ok bool, err error) { reads <- read{s, ok, err} }) })
	if got := h.host.sentCommands(); len(got) != 2 || got[1] != (command{Op: opClipboardGet, A: 1}) {
		t.Fatalf("the commands are %+v", got)
	}
	h.b.clipboardText(1, true, "from another app")
	if r := <-reads; r.text != "from another app" || !r.ok || r.err != nil {
		t.Errorf("GetText = %+v", r)
	}
	// An answer that comes twice, or for no read, is dropped; a clipboard without text says so.
	h.b.clipboardText(1, true, "again")
	h.sync(func() { clip.GetText(ctx, func(s string, ok bool, err error) { reads <- read{s, ok, err} }) })
	h.b.clipboardText(2, false, "")
	if r := <-reads; r.text != "" || r.ok || r.err != nil {
		t.Errorf("an empty clipboard: %+v", r)
	}
}

func TestClipboardReadIsEndedByItsContext(t *testing.T) {
	h := newHarness(t)
	h.run()
	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 2)
	h.sync(func() { h.p.Clipboard().GetText(ctx, func(_ string, _ bool, err error) { errs <- err }) })
	cancel()
	err := <-errs
	if !hasRoute(err, "AfterFunc."+tagClipboardReadTimeout) && !hasRoute(err, tagClipboardReadTimeout) {
		t.Errorf("the read ended with %v", err)
	}
	// The answer of the activity that comes later is not given again.
	h.b.clipboardText(1, true, "late")
	h.sync(func() {})
	select {
	case err := <-errs:
		t.Errorf("a second answer: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	// A context that has ended is a timeout at once, without asking the activity.
	n := len(h.host.sentCommands())
	h.sync(func() { h.p.Clipboard().GetText(ctx, func(_ string, _ bool, err error) { errs <- err }) })
	if err := <-errs; !hasTag(err, tagClipboardReadTimeout) {
		t.Errorf("a read with an ended context: %v", err)
	}
	if len(h.host.sentCommands()) != n {
		t.Error("the activity was asked for a read that was over")
	}
}

func hasTag(err error, tag string) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.Tag == tag
}

func TestClipboardKindsAndFormats(t *testing.T) {
	h := newHarness(t)
	h.run()
	clip := h.p.Clipboard()
	ctx := context.Background()
	errs := make(chan error, 1)
	contents := make(chan ui.ClipboardContent, 1)
	h.sync(func() {
		clip.SetContent(ctx, ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}, func(err error) { errs <- err })
	})
	if err := <-errs; !hasTag(err, tagClipboardPrimaryUnavailable) {
		t.Errorf("the primary selection: %v", err)
	}
	h.sync(func() {
		clip.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{PNG: []byte{1}}, func(err error) { errs <- err })
	})
	if err := <-errs; !hasTag(err, tagAndroidClipboardFormatUnsupported) {
		t.Errorf("an image: %v", err)
	}
	h.sync(func() {
		clip.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: "ok", HasText: true}, func(err error) { errs <- err })
	})
	if err := <-errs; err != nil {
		t.Errorf("text: %v", err)
	}
	h.sync(func() {
		clip.GetContent(ctx, ui.ClipboardKindSystem, "image/png", func(c ui.ClipboardContent, err error) {
			if err != nil {
				errs <- err
			}
			contents <- c
		})
	})
	if c := <-contents; !c.IsEmpty() {
		t.Errorf("another format reads as %+v", c)
	}
	h.sync(func() {
		clip.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(c ui.ClipboardContent, err error) {
			if err != nil {
				errs <- err
			}
			contents <- c
		})
	})
	h.b.clipboardText(1, true, "pasted")
	if c := <-contents; c.Text != "pasted" || !c.HasText {
		t.Errorf("text reads as %+v", c)
	}
}

// TestClipboardHTML: a content with HTML is one command (the HTML, then the text, and the length of
// the HTML in UTF-16 units, which Java cuts it by), and the HTML is read with a command of its own whose
// answer is the HTML.
func TestClipboardHTML(t *testing.T) {
	writes := []struct {
		name    string
		content ui.ClipboardContent
		want    command
	}{
		{"text and HTML", ui.ClipboardContent{Text: "café", HasText: true, HTML: "<b>café</b>"},
			command{Op: opClipboardSetHTML, A: 11, B: 1, S: "<b>café</b>café"}},
		{"HTML only", ui.ClipboardContent{HTML: "<i>x</i>"}, command{Op: opClipboardSetHTML, A: 8, B: 0, S: "<i>x</i>"}},
		{"characters outside the BMP count two units", ui.ClipboardContent{Text: "\U0001F600", HasText: true, HTML: "<p>\U0001F600</p>"},
			command{Op: opClipboardSetHTML, A: 9, B: 1, S: "<p>\U0001F600</p>\U0001F600"}},
		{"HTML with an image: the image is left out", ui.ClipboardContent{HTML: "<u>", PNG: []byte{1}}, command{Op: opClipboardSetHTML, A: 3, B: 0, S: "<u>"}},
		{"invalid UTF-8 is replaced", ui.ClipboardContent{HTML: "a\xffb"}, command{Op: opClipboardSetHTML, A: 3, B: 0, S: "a�b"}},
	}
	for _, tt := range writes {
		t.Run("write "+tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.run()
			done := make(chan error, 1)
			h.sync(func() {
				h.p.Clipboard().SetContent(context.Background(), ui.ClipboardKindSystem, tt.content, func(err error) { done <- err })
			})
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if got := h.host.sentCommands(); len(got) != 1 || got[0] != tt.want {
				t.Errorf("the commands are %+v, want %+v", got, tt.want)
			}
		})
	}

	reads := []struct {
		name string
		ok   bool
		html string
		want ui.ClipboardContent
	}{
		{"a clip with HTML", true, "<b>café</b>", ui.ClipboardContent{HTML: "<b>café</b>"}},
		{"a clip without HTML", false, "", ui.ClipboardContent{}},
	}
	for _, tt := range reads {
		t.Run("read "+tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.run()
			type result struct {
				c   ui.ClipboardContent
				err error
			}
			got := make(chan result, 1)
			h.sync(func() {
				h.p.Clipboard().GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatHTML, func(c ui.ClipboardContent, err error) { got <- result{c, err} })
			})
			if cmds := h.host.sentCommands(); len(cmds) != 1 || cmds[0] != (command{Op: opClipboardGetHTML, A: 1}) {
				t.Fatalf("the commands are %+v", cmds)
			}
			h.b.clipboardText(1, tt.ok, tt.html)
			if r := <-got; r.err != nil || !reflect.DeepEqual(r.c, tt.want) {
				t.Errorf("GetContent = %+v, want %+v", r, tt.want)
			}
		})
	}

	t.Run("a read of the HTML ends with its context", func(t *testing.T) {
		h := newHarness(t)
		h.run()
		ctx, cancel := context.WithCancel(context.Background())
		errs := make(chan error, 1)
		h.sync(func() {
			h.p.Clipboard().GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatHTML, func(_ ui.ClipboardContent, err error) { errs <- err })
		})
		cancel()
		if err := <-errs; !hasTag(err, tagClipboardReadTimeout) {
			t.Errorf("the read ended with %v", err)
		}
	})
}
