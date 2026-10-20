package widgets_test

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// testTextSelectionControls is a TextSelectionControls with handles of a fixed
// size: the testTextSelectionHandleControls of Flutter's text_selection_utils.
type testTextSelectionControls struct{}

func (testTextSelectionControls) BuildHandle(bc BuildContext, handleType rendering.TextSelectionHandleType, textLineHeight float64, onTap foundation.VoidCallback) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(22.0), Height: foundation.Some(22.0)})
}

func (testTextSelectionControls) GetHandleAnchor(rendering.TextSelectionHandleType, float64) ui.Offset {
	return ui.Offset{Dx: 11, Dy: 0}
}

func (testTextSelectionControls) GetHandleSize(float64) ui.Size {
	return ui.Size{Width: 22, Height: 22}
}

// textOffsetToPosition is the global position of the caret at the offset of the
// only editable of the tree.
func textOffsetToPosition(ts *gluttertest.Tester, offset int) ui.Offset {
	r := findRenderEditable(ts)
	local := r.GetLocalRectForCaret(ui.NewTextPosition(offset)).Center()
	return r.LocalToGlobal(local, nil)
}

func plainEditableStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(14.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
}
