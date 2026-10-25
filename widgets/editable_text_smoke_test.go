package widgets_test

import (
	"testing"
	"time"

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

// pumpedBlinkField pumps an EditableText with the focus, after the platform said how long the
// caret stays on, and returns its state.
func pumpedBlinkField(t *testing.T, blink time.Duration) (*gluttertest.Tester, *EditableTextState) {
	t.Helper()
	ts := gluttertest.New(t)
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.CaretBlinkTime = blink })
	controller := NewTextEditingController("hello")
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "t"})
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, style, ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)),
		SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
	state.RequestKeyboard()
	ts.Pump()
	return ts, state
}

// TestEditableTextCaretBlinksAtTheRateOfTheSystem checks the deviation of SPEC §12.10: the caret is
// on, and then off, for the caret blink time of the system (Windows: the control panel), 500 ms
// when the platform says nothing, and it stays on when the user turned the blinking off.
func TestEditableTextCaretBlinksAtTheRateOfTheSystem(t *testing.T) {
	tests := []struct {
		name     string
		blink    time.Duration
		interval time.Duration
		blinks   bool
	}{
		{"the platform says nothing", 0, 500 * time.Millisecond, true},
		{"530 ms, the default of Windows", 530 * time.Millisecond, 530 * time.Millisecond, true},
		{"250 ms", 250 * time.Millisecond, 250 * time.Millisecond, true},
		{"1200 ms", 1200 * time.Millisecond, 1200 * time.Millisecond, true},
		{"blinking turned off", ui.CaretBlinkNever, 500 * time.Millisecond, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, state := pumpedBlinkField(t, tt.blink)
			if got := state.CursorBlinkInterval(); got != tt.interval {
				t.Errorf("CursorBlinkInterval = %v, want %v", got, tt.interval)
			}
			if !state.CursorCurrentlyVisible() {
				t.Fatal("the caret is off when the field gets the focus")
			}
			if !tt.blinks {
				for range 6 {
					ts.PumpFrame(tt.interval)
					if !state.CursorCurrentlyVisible() {
						t.Fatal("the caret went off, but the user turned the blinking off")
					}
				}
				return
			}
			// It stays on for the whole period, goes off at its end, and on again one period later.
			ts.PumpFrame(tt.interval - time.Millisecond)
			if !state.CursorCurrentlyVisible() {
				t.Errorf("the caret went off before %v", tt.interval)
			}
			ts.PumpFrame(2 * time.Millisecond)
			if state.CursorCurrentlyVisible() {
				t.Errorf("the caret is still on after %v", tt.interval)
			}
			ts.PumpFrame(tt.interval)
			if !state.CursorCurrentlyVisible() {
				t.Errorf("the caret is not on again after %v more", tt.interval)
			}
		})
	}
}

func TestEditableTextCaretFollowsAChangeOfTheBlinkTime(t *testing.T) {
	change := func(ts *gluttertest.Tester, blink time.Duration) {
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.CaretBlinkTime = blink })
	}
	t.Run("faster", func(t *testing.T) {
		ts, state := pumpedBlinkField(t, 500*time.Millisecond)
		change(ts, 200*time.Millisecond)
		ts.PumpFrame(500 * time.Millisecond) // the tick of the old period starts the new one
		ts.PumpFrame(199 * time.Millisecond)
		if !state.CursorCurrentlyVisible() {
			t.Error("the caret went off before the new period")
		}
		ts.PumpFrame(2 * time.Millisecond)
		if state.CursorCurrentlyVisible() {
			t.Error("the caret is still on after the new period")
		}
		ts.PumpFrame(200 * time.Millisecond)
		if !state.CursorCurrentlyVisible() {
			t.Error("the caret did not come back after one more period")
		}
	})
	t.Run("turned off", func(t *testing.T) {
		ts, state := pumpedBlinkField(t, 500*time.Millisecond)
		change(ts, ui.CaretBlinkNever)
		ts.PumpFrame(500 * time.Millisecond)
		for range 6 {
			ts.PumpFrame(500 * time.Millisecond)
			if !state.CursorCurrentlyVisible() {
				t.Fatal("the caret went off after the user turned the blinking off")
			}
		}
	})
	t.Run("turned on again", func(t *testing.T) {
		ts, state := pumpedBlinkField(t, ui.CaretBlinkNever)
		change(ts, 300*time.Millisecond)
		ts.PumpFrame(500 * time.Millisecond) // the next tick sees the setting
		seen := map[bool]bool{}
		for range 6 {
			ts.PumpFrame(300 * time.Millisecond)
			seen[state.CursorCurrentlyVisible()] = true
		}
		if len(seen) != 2 {
			t.Errorf("the caret did not blink after the setting came back: %v", seen)
		}
	})
}
