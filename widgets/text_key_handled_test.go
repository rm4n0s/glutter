package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
)

// TestBindingReportsAKeyTheTextFieldTookAsHandled: a character that nothing handles is typed into the
// focused text field by the framework, and the engine is told it was handled, so that a platform with a text
// field of its own (the web's) keeps the browser from typing it a second time. A key the text field cannot
// use is not handled, and neither is a character without a text field.
func TestBindingReportsAKeyTheTextFieldTookAsHandled(t *testing.T) {
	f := newEditableFixture(t, "ab", collapsed(2))
	if err := f.ts.TakeError(); err != nil {
		t.Fatal(err)
	}
	keys := []struct {
		name string
		key  services.LogicalKeyboardKey
		char string
		want bool
	}{
		{"a character", services.LogicalKeyboardKeyKeyC, "c", true},
		{"a key without a character", services.LogicalKeyboardKeyF5, "", false},
	}
	for _, k := range keys {
		if got := f.ts.SendKeyDownEvent(k.key, gluttertest.KeyOpts{Character: k.char}); got != k.want {
			t.Errorf("%s: handled = %v, want %v", k.name, got, k.want)
		}
		f.ts.SendKeyUpEvent(k.key)
	}
	if got := f.controller.Text(); got != "abc" {
		t.Errorf("text = %q, want the character typed once", got)
	}

	f.focus.Unfocus()
	f.ts.Pump()
	if f.ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyD, gluttertest.KeyOpts{Character: "d"}) {
		t.Error("a character was handled with no text field focused")
	}
}
