package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from the group 'magnifier' of packages/flutter/test/widgets/selectable_region_test.dart.
func TestSelectableRegionMagnifier(t *testing.T) {
	t.Run("Can drag handles to show, unshow, and update magnifier", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		const text = "Monkeys and rabbits in my soup"
		fakeMagnifier := foundation.NewUniqueKey()
		var magnifierInfo *foundation.ValueNotifier[MagnifierInfo]

		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: selectionTextStyle(), home: NewSelectableRegion(testTextSelectionControls{}, ltrText(text), SelectableRegionOpts{
			ContextMenuBuilder: emptyContextMenu,
			MagnifierConfiguration: NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{
				MagnifierBuilder: func(_ BuildContext, _ *MagnifierController, localMagnifierInfo *foundation.ValueNotifier[MagnifierInfo]) Widget {
					magnifierInfo = localMagnifierInfo
					return NewContainer(nil, ContainerOpts{Key: fakeMagnifier})
				},
			}),
		})}))
		ts.PumpAndSettle()

		paragraph := paragraphOf(ts, text)

		// Show the selection handles.
		activateSelectionGesture := ts.StartGesture(paragraphOffsetToPosition(paragraph, len(text)/2))
		ts.Pump(500 * time.Millisecond)
		activateSelectionGesture.Up()
		ts.Pump(500 * time.Millisecond)

		// Drag the handle around so that the magnifier shows.
		selectionBox := paragraph.GetBoxesForSelection(ui.TextRange{Start: paragraph.Selections()[0].Start(), End: paragraph.Selections()[0].End()})[0]
		leftHandlePos := paragraph.LocalToGlobal(ui.Offset{Dx: selectionBox.Left, Dy: selectionBox.Bottom}, nil)
		gesture := ts.StartGesture(leftHandlePos)
		gesture.MoveTo(paragraphOffsetToPosition(paragraph, len(text)-2))
		ts.Pump()

		// Expect the magnifier to show and then store it's position.
		if ts.Count(gluttertest.ByKey(fakeMagnifier)) != 1 {
			t.Fatal("the magnifier is not shown while a handle is dragged")
		}
		firstDragGesturePosition := magnifierInfo.Value().GlobalGesturePosition

		gesture.MoveTo(paragraphOffsetToPosition(paragraph, len(text)))
		ts.Pump()

		// Expect the position the magnifier gets to have moved.
		if firstDragGesturePosition == magnifierInfo.Value().GlobalGesturePosition {
			t.Error("the position the magnifier gets did not move")
		}

		// Lift the pointer and expect the magnifier to disappear.
		gesture.Up()
		ts.Pump()

		if ts.Count(gluttertest.ByKey(fakeMagnifier)) != 0 {
			t.Error("the magnifier is still shown after the drag")
		}
	})

	t.Run("a region without a configuration has the disabled one", func(t *testing.T) {
		r := NewSelectableRegion(testTextSelectionControls{}, ltrText("a"))
		if r.MagnifierConfiguration() != TextMagnifierConfigurationDisabled {
			t.Error("the default configuration is not the disabled one")
		}
	})
}
