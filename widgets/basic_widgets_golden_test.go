package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func swatchOf(c uint32, w, h float64) Widget {
	return NewSizedBox(NewColoredBox(ui.NewColor(c), nil), SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)})
}

func at(l, tp float64, child Widget) Widget {
	return NewPositioned(child, PositionedOpts{Left: foundation.Some(l), Top: foundation.Some(tp)})
}

// TestBasicWidgetsGolden draws the layout, painting and clipping widgets of
// step 7.6 in one scene: a container with a decoration, a row, a wrap, a table,
// a rotated box, clips, an opacity, a physical model and a placeholder.
func TestBasicWidgetsGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	deco := painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFE8DEF8)),
		BorderRadius: painting.NewBorderRadiusCircular(12), Border: painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFF6750A4)), Width: foundation.Some(2.0)})})
	scene := NewDirectionality(ui.TextDirectionLTR, NewStack([]Widget{
		NewPositionedFill(NewColoredBox(ui.NewColor(0xFFFFFBFE), nil)),
		at(10, 10, NewContainer(NewRow([]Widget{swatchOf(0xFF6750A4, 30, 30), NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(8.0)}), swatchOf(0xFF625B71, 30, 30), NewSpacer(), swatchOf(0xFF7D5260, 30, 30)}),
			ContainerOpts{Width: foundation.Some(180.0), Padding: painting.NewEdgeInsetsAll(10), Decoration: deco, ClipBehavior: ui.ClipAntiAlias})),
		at(210, 10, NewSizedBox(NewWrap([]Widget{swatchOf(0xFFB3261E, 24, 24), swatchOf(0xFF386A20, 24, 24), swatchOf(0xFF0061A4, 24, 24),
			swatchOf(0xFFFFB74D, 24, 24), swatchOf(0xFF6750A4, 24, 24), swatchOf(0xFF7D5260, 24, 24), swatchOf(0xFF386A20, 24, 24)}, WrapOpts{Spacing: 6, RunSpacing: 6}),
			SizedBoxOpts{Width: foundation.Some(130.0)})),
		at(10, 90, NewTable([]TableRow{
			NewTableRow([]Widget{swatchOf(0xFFB3261E, 20, 20), swatchOf(0xFF386A20, 20, 20)}),
			NewTableRow([]Widget{NewTableCell(swatchOf(0xFF0061A4, 20, 10), TableCellOpts{VerticalAlignment: rendering.TableCellVerticalAlignmentBottom}), swatchOf(0xFFFFB74D, 20, 20)}),
		}, TableOpts{DefaultColumnWidth: rendering.NewFixedColumnWidth(40), Border: &rendering.TableBorder{}})),
		at(110, 100, NewTransformRotate(0.4, swatchOf(0xFF6750A4, 50, 30))),
		at(190, 90, NewClipOval(NewSizedBox(NewColoredBox(ui.NewColor(0xFF386A20), nil), SizedBoxOpts{Width: foundation.Some(60.0), Height: foundation.Some(40.0)}))),
		at(270, 90, NewClipRRect(swatchOf(0xFF0061A4, 60, 40), ClipRRectOpts{BorderRadius: painting.NewBorderRadiusCircular(14)})),
		at(10, 170, NewOpacity(0.4, swatchOf(0xFFB3261E, 60, 40))),
		at(40, 190, NewOpacity(0.4, swatchOf(0xFF0061A4, 60, 40))),
		at(120, 170, NewPhysicalModel(ui.NewColor(0xFFFFFFFF), swatchOf(0xFF6750A4, 60, 40), PhysicalModelOpts{Elevation: 6,
			BorderRadius: foundation.Some(painting.NewBorderRadiusCircular(8))})),
		at(200, 170, NewSizedBox(NewPlaceholder(nil), SizedBoxOpts{Width: foundation.Some(70.0), Height: foundation.Some(50.0)})),
		at(290, 170, NewSizedBox(NewIndexedStack([]Widget{swatchOf(0xFFB3261E, 50, 50), swatchOf(0xFF386A20, 50, 50)}, IndexedStackOpts{Index: foundation.Some(1)}), SizedBoxOpts{Width: foundation.Some(50.0)})),
		at(10, 250, NewFractionalTranslation(ui.Offset{Dx: 0.5, Dy: 0}, swatchOf(0xFF7D5260, 40, 30))),
		at(120, 250, NewFittedBox(swatchOf(0xFF0061A4, 40, 20), FittedBoxOpts{Fit: painting.BoxFitFill})),
	}))
	ts.PumpWidget(scene)
	ts.MatchesGolden("basic_widgets_part1")
}
