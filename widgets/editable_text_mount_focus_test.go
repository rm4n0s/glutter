package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// lastInputState is the last state the framework gave to the input method of
// the fake platform.
func lastInputState(ts *gluttertest.Tester) (ui.TextInputState, bool) {
	states := ts.TextInputStates()
	if len(states) == 0 {
		return ui.TextInputState{}, false
	}
	return states[len(states)-1], true
}

// typeIntoField plays the input method: it commits the text, and reports whether
// the field took it.
func typeIntoField(ts *gluttertest.Tester, controller *TextEditingController, text string) bool {
	before := controller.Text()
	ts.SendTextInputEvent(ui.TextInputEvent{Commit: text, PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	ts.Pump()
	return controller.Text() != before
}

func TestEditableTextMountedWithTheFocusTakesKeys(t *testing.T) {
	type shape int
	const (
		stackToPlain shape = iota
		keyChanged
		plainToStack
	)
	tests := []struct {
		name        string
		text        string
		placeholder bool
		shape       shape
		readOnly    bool
		wantTyped   bool
	}{
		{"stack removed, text", "abc", false, stackToPlain, false, true},
		{"stack removed, empty with a placeholder", "", true, stackToPlain, false, true},
		{"stack removed, empty without a placeholder", "", false, stackToPlain, false, true},
		{"key changed, text", "abc", false, keyChanged, false, true},
		{"key changed, empty with a placeholder", "", true, keyChanged, false, true},
		{"stack added, text", "abc", false, plainToStack, false, true},
		{"read only opens nothing", "abc", false, stackToPlain, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := NewTextEditingController(tt.text)
			focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
			field := func(key int) Widget {
				return NewSizedBox(NewEditableText(controller, focus, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888),
					EditableTextOpts{ReadOnly: tt.readOnly, Key: foundation.NewValueKey(key)}), SizedBoxOpts{Width: foundation.Some(200.0)})
			}
			withStack := func(key int) Widget {
				children := []Widget{}
				if tt.placeholder {
					children = append(children, NewText("placeholder"))
				}
				children = append(children, field(key))
				return NewCenter(NewStack(children))
			}
			plain := func(key int) Widget { return NewCenter(field(key)) }

			first, second := withStack(1), plain(1)
			switch tt.shape {
			case keyChanged:
				first, second = plain(1), plain(2)
			case plainToStack:
				first, second = plain(1), withStack(1)
			}
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: first}))
			focus.RequestFocus()
			ts.Pump()
			if !focus.HasPrimaryFocus() {
				t.Fatal("the node has no focus before the field is mounted again")
			}
			selBefore := controller.Selection()
			// The field is made again under the node that kept the focus.
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: second}))
			ts.Pump()
			if !focus.HasPrimaryFocus() {
				t.Fatal("the node lost the focus when the field was mounted again")
			}
			if sel := controller.Selection(); !sel.Equal(selBefore) {
				t.Errorf("selection = %+v after the field was made again, want %+v", sel, selBefore)
			}
			state, ok := lastInputState(ts)
			if got := ok && state.Enabled; got != tt.wantTyped {
				t.Errorf("input method on = %v, want %v (state %+v)", got, tt.wantTyped, state)
			}
			if got := typeIntoField(ts, controller, "x"); got != tt.wantTyped {
				t.Errorf("typed text taken = %v, want %v; text %q", got, tt.wantTyped, controller.Text())
			}
			if errs := ts.Errors(); len(errs) != 0 {
				t.Errorf("the framework reported %d errors: %v", len(errs), errs[0])
			}
		})
	}
}

func TestEditableTextGetsANodeThatHasTheFocus(t *testing.T) {
	tests := []struct {
		name      string
		readOnly  bool
		wantTyped bool
	}{
		{"focused node", false, true},
		{"read only", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := NewTextEditingController("abc")
			unfocused := NewFocusNode(FocusNodeOpts{DebugLabel: "unfocused"})
			focused := NewFocusNode(FocusNodeOpts{DebugLabel: "focused"})
			field := func(node *FocusNode) Widget {
				return NewEditableText(controller, node, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888),
					EditableTextOpts{ReadOnly: tt.readOnly, Key: foundation.NewValueKey("field")})
			}
			// The field is on the first node; the second one has the focus through a
			// Focus widget beside it.
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewColumn([]Widget{
				NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: focused}),
				NewSizedBox(field(unfocused), SizedBoxOpts{Width: foundation.Some(200.0)}),
			}), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
			focused.RequestFocus()
			ts.Pump()
			if !focused.HasPrimaryFocus() {
				t.Fatal("the second node has no focus")
			}
			if s, ok := lastInputState(ts); ok && s.Enabled {
				t.Fatal("the input method is on before the field has the focus")
			}
			// The field gets the node that has the focus, and the Focus widget goes.
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewColumn([]Widget{
				NewSizedBox(field(focused), SizedBoxOpts{Width: foundation.Some(200.0)}),
			}), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
			ts.Pump()
			if !focused.HasPrimaryFocus() {
				t.Fatal("the node lost the focus when the field got it")
			}
			state, ok := lastInputState(ts)
			if got := ok && state.Enabled; got != tt.wantTyped {
				t.Errorf("input method on = %v, want %v", got, tt.wantTyped)
			}
			if got := typeIntoField(ts, controller, "x"); got != tt.wantTyped {
				t.Errorf("typed text taken = %v, want %v; text %q", got, tt.wantTyped, controller.Text())
			}
		})
	}
}

func TestEditableTextWithAConnectionOpensNoSecondOne(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewTextEditingController("abc")
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, plainEditableStyle(),
		ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
	state.RequestKeyboard()
	ts.Pump()
	before := len(ts.TextInputStates())
	// A frame later nothing is attached again.
	ts.Pump()
	ts.Pump()
	if after := len(ts.TextInputStates()); after != before {
		t.Errorf("the input method got %d states after the first frames, want %d", after, before)
	}
	if !typeIntoField(ts, controller, "x") {
		t.Error("the field with the focus does not take keys")
	}
}

func TestEditableTextWithoutTheFocusOpensNothing(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewTextEditingController("abc")
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, plainEditableStyle(),
		ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	ts.Pump()
	if s, ok := lastInputState(ts); ok && s.Enabled {
		t.Errorf("the input method is on for a field without the focus: %+v", s)
	}
	if typeIntoField(ts, controller, "x") {
		t.Error("a field without the focus took keys")
	}
}

func TestEditableTextTapMovesTheCaretAndTakesKeys(t *testing.T) {
	tests := []struct {
		name     string
		hasFocus bool
	}{
		{"without the focus", false},
		{"with the focus", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := NewTextEditingController("hello world")
			focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(NewEditableText(controller, focus, plainEditableStyle(),
				ui.NewColor(0xFF000000), ui.NewColor(0xFF888888)), SizedBoxOpts{Width: foundation.Some(200.0)}))}))
			if tt.hasFocus {
				focus.RequestFocus()
				ts.Pump()
			}
			ts.TapAt(textOffsetToPosition(ts, 5))
			ts.Pump()
			if !focus.HasPrimaryFocus() {
				t.Error("the tap did not give the focus to the field")
			}
			sel := controller.Selection()
			if !sel.IsCollapsed() || sel.BaseOffset != 5 {
				t.Errorf("selection = %+v, want a caret at 5", sel)
			}
			state, ok := lastInputState(ts)
			if !ok || !state.Enabled {
				t.Errorf("the input method is off after the tap: %+v", state)
			}
			if !typeIntoField(ts, controller, "X") || controller.Text() != "helloX world" {
				t.Errorf("text = %q, want the key typed at the caret", controller.Text())
			}
		})
	}
}

func TestEditableTextTapOnAFieldMountedAgainWithTheFocus(t *testing.T) {
	tests := []struct {
		name      string
		tapAt     int
		wantCaret int
	}{
		{"in the middle of the text", 5, 5},
		{"where the caret is", 11, 11},
		{"where the caret is, not at the end", 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := NewTextEditingController("hello world")
			focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
			home := func(key int) Widget {
				return NewCenter(NewSizedBox(NewEditableText(controller, focus, plainEditableStyle(),
					ui.NewColor(0xFF000000), ui.NewColor(0xFF888888), EditableTextOpts{Key: foundation.NewValueKey(key)}),
					SizedBoxOpts{Width: foundation.Some(200.0)}))
			}
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: home(1)}))
			focus.RequestFocus()
			ts.Pump()
			if tt.tapAt == 5 && tt.name != "in the middle of the text" {
				controller.SetSelection(services.NewTextSelectionCollapsed(5))
			}
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: home(2)}))
			ts.Pump()
			ts.TapAt(textOffsetToPosition(ts, tt.tapAt))
			ts.Pump()
			if sel := controller.Selection(); !sel.IsCollapsed() || sel.BaseOffset != tt.wantCaret {
				t.Errorf("selection = %+v, want a caret at %d", sel, tt.wantCaret)
			}
			if state, ok := lastInputState(ts); !ok || !state.Enabled {
				t.Errorf("the input method is off after the tap: %+v", state)
			}
			if !typeIntoField(ts, controller, "X") {
				t.Errorf("the field did not take keys after the tap; text %q", controller.Text())
			}
		})
	}
}
