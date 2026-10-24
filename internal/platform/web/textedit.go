package web

import "github.com/rm4n0s/glutter/internal/platform/textedit"

// The model of the hidden text field is shared with the Android platform, so it lives in
// internal/platform/textedit (an Android binary has no web backend); these names keep the web code as it
// was.
type (
	// TextEditor is textedit.TextEditor.
	//
	// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy
	TextEditor = textedit.TextEditor
	// DOMWrite is textedit.DOMWrite.
	//
	// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.applyToDomElement
	DOMWrite = textedit.DOMWrite
	// EditingState is textedit.EditingState.
	//
	// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState
	EditingState = textedit.EditingState
	// Composition is textedit.Composition.
	//
	// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart CompositionAwareMixin
	Composition = textedit.Composition
)

// EditingStateFromDOM is textedit.EditingStateFromDOM.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.fromDomElement
func EditingStateFromDOM(value string, selectionStart, selectionEnd int, direction string) EditingState {
	return textedit.EditingStateFromDOM(value, selectionStart, selectionEnd, direction)
}

// NewEditingState is textedit.NewEditingState.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.new
func NewEditingState(text string, base, extent int) EditingState {
	return textedit.NewEditingState(text, base, extent)
}
