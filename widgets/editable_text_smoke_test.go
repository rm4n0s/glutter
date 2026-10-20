package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func TestEditableTextSmoke(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewTextEditingController("hello")
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "t"})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))}), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)),
		SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
	state.RequestKeyboard()
	ts.Pump()
	if !focus.HasFocus() {
		t.Fatal("no focus")
	}
	state.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "hello!", Selection: foundation.Some(services.NewTextSelectionCollapsed(6))}))
	ts.Pump()
	if controller.Text() != "hello!" {
		t.Errorf("text = %q", controller.Text())
	}
}

func TestEditableTextRemovedWhileFocusedReportsNoError(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewTextEditingController("hello")
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "t"})
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, style, ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)),
		SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]()).RequestKeyboard()
	ts.Pump()
	if !focus.HasFocus() {
		t.Fatal("no focus")
	}
	// The input connection is open: the post-frame callbacks and the callback of
	// the composition of the layer run after the field is gone.
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	ts.Pump()
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("the framework reported %d errors: %v", len(errs), errs[0])
	}
	ts.TakeErrors()
}

func TestEditableTextDeterministicCursorDoesNotBlink(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("DebugOptions.DeterministicCursor has an effect in debug builds only")
	}
	tests := []struct {
		name          string
		deterministic bool
		wantTimer     bool
	}{
		{"blinks by default", false, true},
		{"deterministic cursor", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debug := NewDebugOptions()
			debug.DeterministicCursor = tt.deterministic
			ts := gluttertest.New(t, gluttertest.Opts{Debug: debug})
			controller := NewTextEditingController("hello")
			focus := NewFocusNode(FocusNodeOpts{DebugLabel: "t"})
			style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, style, ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)),
				SizedBoxOpts{Width: foundation.Some(200.0)}))}))
			state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
			state.RequestKeyboard()
			ts.Pump()
			seen := map[bool]bool{}
			for i := 0; i < 6; i++ {
				ts.PumpFrame(state.CursorBlinkInterval())
				seen[state.CursorCurrentlyVisible()] = true
			}
			if blinked := len(seen) == 2; blinked != tt.wantTimer {
				t.Errorf("the caret blinked = %v, want %v (states seen: %v)", blinked, tt.wantTimer, seen)
			}
		})
	}
}
