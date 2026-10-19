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
