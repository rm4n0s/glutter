//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The text tests build internal/platform/web/texttest, a small app that dispatches synthetic DOM events at
// the hidden text field of the platform and at the host, replaces navigator.clipboard by a stub with
// promises that it settles by hand, and posts what it saw (SPEC §23.5): beforeinput and input events typing
// into a TextField, a composition committing its text, a paste event, and keys sent while a clipboard read is
// pending.

var (
	textOnce sync.Once
	textDir  string
	textErr  error
)

func textSite(t *testing.T) string {
	t.Helper()
	textOnce.Do(func() {
		if textDir, textErr = os.MkdirTemp("", "glutter-web-texttest-"); textErr != nil {
			return
		}
		_, textErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/platform/web/texttest", Dir: webtest.ModuleRoot(t), OutDir: textDir,
		})
	})
	if textErr != nil {
		t.Fatalf("building the text test app for the web: %v", textErr)
	}
	return textDir
}

func cleanupTextSite() {
	if textDir != "" {
		_ = os.RemoveAll(textDir) // a temporary directory: a leftover is harmless
	}
}

// textField and textState mirror what texttest posts.
type textField struct {
	Present        bool   `json:"present"`
	Tag            string `json:"tag"`
	Type           string `json:"type"`
	InputMode      string `json:"inputMode"`
	EnterKeyHint   string `json:"enterKeyHint"`
	Autocapitalize string `json:"autocapitalize"`
	Focused        bool   `json:"focused"`
	Value          string `json:"value"`
}

type textState struct {
	Text           string `json:"text"`
	ComposingValid bool   `json:"composingValid"`
	ComposingStart int    `json:"composingStart"`
	ComposingEnd   int    `json:"composingEnd"`
	Selection      int    `json:"selection"`
}

type textObservations struct {
	AfterTap              textField `json:"afterTap"`
	Typed                 textState `json:"typed"`
	TypedElement          textField `json:"typedElement"`
	Backspaced            textState `json:"backspaced"`
	MidComposition        textState `json:"midComposition"`
	MidCompositionElement textField `json:"midCompositionElement"`
	Composed              textState `json:"composed"`
	ComposedElement       textField `json:"composedElement"`
	PastePrevented        bool      `json:"pastePrevented"`
	Pasted                textState `json:"pasted"`
	PastedElement         textField `json:"pastedElement"`
	KeyTypedPrevented     bool      `json:"keyTypedPrevented"`
	KeyTyped              textState `json:"keyTyped"`
	PasteKeyPrevented     bool      `json:"pasteKeyPrevented"`
	PasteKeyState         textState `json:"pasteKeyState"`
	PendingKeysPrevented  []bool    `json:"pendingKeysPrevented"`
	PendingAnswered       bool      `json:"pendingAnswered"`
	PendingText           string    `json:"pendingText"`
	ReadText              string    `json:"readText"`
	ReadError             string    `json:"readError"`
	RejectedTag           string    `json:"rejectedTag"`
	Written               string    `json:"written"`
	WriteError            string    `json:"writeError"`
	Copied                string    `json:"copied"`
	Secret                textState `json:"secret"`
	SecretElement         textField `json:"secretElement"`
	MultiElement          textField `json:"multiElement"`
	Multi                 textState `json:"multi"`
	EnterPrevented        bool      `json:"enterPrevented"`
	Gone                  textField `json:"gone"`
	HostFocused           bool      `json:"hostFocused"`
}

// TestBrowserTextInput: in Chromium and Firefox, a tap on a TextField makes the hidden text field, synthetic
// beforeinput and input events type into it (and a deleteContentBackward deletes), a compositionstart,
// compositionupdate and compositionend sequence shows the composing text and then commits it, a paste event
// brings its text and is prevented, a character key is typed once by the framework and prevented, the
// browser's paste shortcut is left alone, keys sent while a clipboard read is pending are handled and the
// read is answered from the promise's callback (no deadlock), and the password, the multiline field and
// the loss of the focus give a password input, a textarea and no field.
func TestBrowserTextInput(t *testing.T) {
	dir := textSite(t)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Width: 480, Height: 640}, "text")
		if err != nil {
			t.Fatalf("Await in %s: %v", b, err)
		}
		var o textObservations
		if err := json.Unmarshal(res.Data, &o); err != nil {
			t.Fatalf("the result is not the observations: %v\n%s", err, res.Data)
		}
		t.Logf("%+v", o)
		check := func(name string, got, want any) {
			t.Helper()
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s = %+v, want %+v", name, got, want)
			}
		}
		// The hidden text field.
		check("the hidden field after the tap is an input", o.AfterTap.Tag, "INPUT")
		check("the hidden field has the focus", o.AfterTap.Focused, true)
		check("no inputmode on a desktop browser", o.AfterTap.InputMode, "")
		check("autocapitalize off", o.AfterTap.Autocapitalize, "off")

		// Typing with beforeinput and input events.
		check("text after typing hello", o.Typed.Text, "hello")
		check("caret after typing hello", o.Typed.Selection, 5)
		check("the element after typing hello", o.TypedElement.Value, "hello")
		check("text after the backspace", o.Backspaced.Text, "hell")

		// Composition.
		check("text while composing ni", o.MidComposition.Text, "hellni")
		check("the composing range while composing ni", [3]any{o.MidComposition.ComposingValid, o.MidComposition.ComposingStart, o.MidComposition.ComposingEnd}, [3]any{true, 4, 6})
		check("the element while composing ni", o.MidCompositionElement.Value, "hellni")
		check("text after the commit", o.Composed.Text, "hell你好")
		check("composing after the commit", o.Composed.ComposingValid, false)
		check("the element after the commit", o.ComposedElement.Value, "hell你好")

		// Paste through the paste event.
		check("the paste event's default prevented", o.PastePrevented, true)
		check("text after the paste", o.Pasted.Text, "hell你好 world")
		check("the element after the paste", o.PastedElement.Value, "hell你好 world")

		// A key is typed once.
		check("the A key prevented", o.KeyTypedPrevented, true)
		check("text after the A key", o.KeyTyped.Text, "hell你好 worlda")
		check("Control+V prevented", o.PasteKeyPrevented, false)
		check("text after Control+V", o.PasteKeyState.Text, "hell你好 worlda")

		// Keys while a clipboard read is pending.
		check("keys sent while the read is pending are handled", o.PendingKeysPrevented, []bool{true, true, true})
		check("the read is not answered yet", o.PendingAnswered, false)
		check("text after the keys of the pending read", o.PendingText, "hell你好 worldaxyz")
		check("the read's text once the promise was resolved", o.ReadText, "from clipboard")
		check("the read's error", o.ReadError, "")
		check("a refused read", o.RejectedTag, "WebClipboardReadFailed")
		check("SetData wrote", o.Written, "to clipboard")
		check("SetData's error", o.WriteError, "")
		check("Control+C copied the selection", o.Copied, "hell你好 worldaxyz")

		// Other fields.
		check("the password field's input type", o.SecretElement.Type, "password")
		check("the password field's tag", o.SecretElement.Tag, "INPUT")
		check("the text typed in the password field", o.Secret.Text, "s3cret")
		check("the multiline field's tag", o.MultiElement.Tag, "TEXTAREA")
		check("Enter in the multiline field prevented", o.EnterPrevented, true)
		check("the multiline text", o.Multi.Text, "one\ntwo")
		check("the hidden field after the focus was lost", o.Gone.Present, false)
		check("the host has the focus again", o.HostFocused, true)
	})
}
