package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from the asserts of EditableText's constructor
// (packages/flutter/lib/src/widgets/editable_text.dart) that
// editable_text_test.dart checks, and test the tags of its actions.

func TestEditableTextConstructorAsserts(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		tag  string
		opts EditableTextOpts
	}{
		{"an obscuring character of two characters", "EditableTextObscuringCharacterInvalid", EditableTextOpts{ObscuringCharacter: "ab"}},
		{"minLines of zero", "EditableTextMinLinesInvalid", EditableTextOpts{MinLines: foundation.Some(0), MaxLines: foundation.Some(3)}},
		{"minLines more than maxLines", "EditableTextMinLinesGreaterThanMaxLines", EditableTextOpts{MinLines: foundation.Some(3), MaxLines: foundation.Some(2)}},
		{"expands with the default maxLines", "EditableTextExpandsWithLines", EditableTextOpts{Expands: true}},
		{"expands with minLines", "EditableTextExpandsWithLines", EditableTextOpts{Expands: true, MinLines: foundation.Some(2), MaxLines: foundation.Some(0)}},
		{"an obscured multiline field", "EditableTextObscuredMultiline", EditableTextOpts{ObscureText: true, MaxLines: foundation.Some(2)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller := NewTextEditingController()
			focus := NewFocusNode()
			defer controller.Dispose()
			defer focus.Dispose()
			requireTag(t, tc.tag, func() {
				NewEditableText(controller, focus, painting.NewTextStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888), tc.opts)
			})
		})
	}
	t.Run("valid combinations", func(t *testing.T) {
		controller := NewTextEditingController()
		focus := NewFocusNode()
		defer controller.Dispose()
		defer focus.Dispose()
		for _, o := range []EditableTextOpts{
			{Expands: true, MaxLines: foundation.Some(0)},
			{ObscureText: true},
			{MinLines: foundation.Some(2), MaxLines: foundation.Some(4)},
			{ObscuringCharacter: "👨‍👩‍👦"},
		} {
			NewEditableText(controller, focus, painting.NewTextStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888), o)
		}
	})
}
