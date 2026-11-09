package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// TestEditableTextDoesTheSelectorsOfTheInputMethod is the platform asking, as macOS does through
// doCommandBySelector:, for the commands of the Cocoa text system: the selector becomes an intent that is
// invoked in the focused node (EditableTextState.performSelector).
//
// Flutter: widgets/editable_text.dart EditableTextState.performSelector
func TestEditableTextDoesTheSelectorsOfTheInputMethod(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		sel      services.TextSelection
		text     string
		wantSel  services.TextSelection
		wantText string
	}{
		{"moveLeft:", "moveLeft:", collapsed(5), "hello world", collapsed(4), "hello world"},
		{"moveRight:", "moveRight:", collapsed(5), "hello world", collapsed(6), "hello world"},
		{"moveToBeginningOfDocument:", "moveToBeginningOfDocument:", collapsed(5), "hello world", collapsed(0), "hello world"},
		{"moveToEndOfDocument:", "moveToEndOfDocument:", collapsed(5), "hello world", collapsed(11), "hello world"},
		{"moveWordRight:", "moveWordRight:", collapsed(0), "hello world", collapsed(5), "hello world"},
		{"moveLeftAndModifySelection:", "moveLeftAndModifySelection:", collapsed(5), "hello world", rangeSel(5, 4), "hello world"},
		{"deleteBackward:", "deleteBackward:", collapsed(5), "hello world", collapsed(4), "hell world"},
		{"deleteForward:", "deleteForward:", collapsed(5), "hello world", collapsed(5), "helloworld"},
		{"deleteWordBackward:", "deleteWordBackward:", collapsed(5), "hello world", collapsed(0), " world"},
		{"a selector that is not known", "noSuchSelector:", collapsed(5), "hello world", collapsed(5), "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newEditableFixture(t, tt.text, tt.sel)
			f.focus.RequestFocus()
			f.ts.Pump()
			f.ts.SendTextInputEvent(ui.TextInputEvent{Selector: tt.selector})
			f.ts.Pump()
			f.expect(t, tt.wantText, tt.wantSel)
		})
	}
}
