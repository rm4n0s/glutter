// Command layout shows the layout widgets of Glutter in five labelled sections:
// a Row with Expanded and Flexible children, a Column for each MainAxisAlignment,
// a Stack with a Positioned badge, a Wrap of twelve colored tags and a Table.
// It uses only the public packages.
//
// Run it with: go run ./examples/layout
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>.
package main

import (
	"fmt"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The Material 3 baseline colors that the example paints with.
var (
	primary          = ui.NewColor(0xFF6750A4)
	primaryContainer = ui.NewColor(0xFFEADDFF)
	secondary        = ui.NewColor(0xFF625B71)
	secondaryCont    = ui.NewColor(0xFFE8DEF8)
	tertiary         = ui.NewColor(0xFF7D5260)
	errorColor       = ui.NewColor(0xFFB3261E)
	surface          = ui.NewColor(0xFFFFFBFE)
	outline          = ui.NewColor(0xFF79747E)
	onSurface        = ui.NewColor(0xFF1C1B1F)
	white            = ui.NewColor(0xFFFFFFFF)
)

func textStyle(size float64, color ui.Color, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color), FontWeight: weight,
	})
}

func label(s string, size float64, color ui.Color, weight ui.FontWeight) widgets.Widget {
	return widgets.NewText(s, widgets.TextOpts{Style: textStyle(size, color, weight)})
}

func gap(width, height float64) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})
}

func outlined() *painting.BoxDecoration {
	return painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(outline), Width: foundation.Some(1.0)}),
		BorderRadius: painting.NewBorderRadiusCircular(8),
	})
}

// section is a heading above a bordered box with the content.
func section(title string, content widgets.Widget) widgets.Widget {
	return widgets.NewColumn([]widgets.Widget{
		label(title, 14, primary, ui.FontWeightW700),
		gap(0, 6),
		widgets.NewDecoratedBox(outlined(), widgets.NewPadding(painting.NewEdgeInsetsAll(10), content)),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

// cell is a colored box with a centered label.
func cell(text string, color, textColor ui.Color) widgets.Widget {
	return widgets.NewContainer(label(text, 12, textColor, ui.FontWeightW500), widgets.ContainerOpts{
		Color: foundation.Some(color), Alignment: painting.AlignmentCenter, Height: foundation.Some(44.0),
	})
}

// rowSection shows Expanded children with flex factors 1 and 2, and a Flexible
// child that takes only the width it needs.
func rowSection() widgets.Widget {
	return section("Row: Expanded and Flexible", widgets.NewRow([]widgets.Widget{
		widgets.NewExpanded(cell("Expanded 1", primary, white)),
		gap(6, 0),
		widgets.NewExpanded(cell("Expanded 2", secondary, white), widgets.ExpandedOpts{Flex: foundation.Some(2)}),
		gap(6, 0),
		widgets.NewFlexible(widgets.NewContainer(label("Flexible", 12, onSurface, ui.FontWeightW500), widgets.ContainerOpts{
			Color: foundation.Some(primaryContainer), Alignment: painting.AlignmentCenter, Width: foundation.Some(64.0), Height: foundation.Some(44.0),
		})),
		gap(6, 0),
		widgets.NewExpanded(cell("Expanded 1", tertiary, white)),
	}))
}

// columnSection shows a Column for each MainAxisAlignment.
func columnSection() widgets.Widget {
	alignments := []struct {
		name string
		a    rendering.MainAxisAlignment
	}{
		{"start", rendering.MainAxisAlignmentStart},
		{"end", rendering.MainAxisAlignmentEnd},
		{"center", rendering.MainAxisAlignmentCenter},
		{"between", rendering.MainAxisAlignmentSpaceBetween},
		{"around", rendering.MainAxisAlignmentSpaceAround},
		{"evenly", rendering.MainAxisAlignmentSpaceEvenly},
	}
	var cols []widgets.Widget
	for i, al := range alignments {
		if i > 0 {
			cols = append(cols, gap(6, 0))
		}
		square := func(c ui.Color) widgets.Widget {
			return widgets.NewPadding(painting.NewEdgeInsetsAll(2),
				widgets.NewSizedBox(widgets.NewColoredBox(c, nil), widgets.SizedBoxOpts{Width: foundation.Some(16.0), Height: foundation.Some(16.0)}))
		}
		cols = append(cols, widgets.NewExpanded(widgets.NewColumn([]widgets.Widget{
			widgets.NewSizedBox(widgets.NewDecoratedBox(outlined(), widgets.NewColumn([]widgets.Widget{
				square(primary), square(secondary), square(tertiary),
			}, widgets.ColumnOpts{MainAxisAlignment: al.a})), widgets.SizedBoxOpts{Height: foundation.Some(100.0)}),
			gap(0, 4),
			label(al.name, 11, onSurface, ui.FontWeightW400),
		}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})))
	}
	return section("Column: every MainAxisAlignment", widgets.NewRow(cols, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}))
}

// stackSection shows a Stack with a Positioned badge in the corner.
func stackSection() widgets.Widget {
	badge := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(errorColor), Shape: painting.BoxShapeCircle,
	}), widgets.NewSizedBox(widgets.NewCenter(label("3", 12, white, ui.FontWeightW700)), widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}))
	return section("Stack: a Positioned badge", widgets.NewSizedBox(widgets.NewStack([]widgets.Widget{
		widgets.NewPositionedFill(widgets.NewColoredBox(secondaryCont, widgets.NewCenter(label("Stack", 16, onSurface, ui.FontWeightW500)))),
		widgets.NewPositioned(badge, widgets.PositionedOpts{Top: foundation.Some(8.0), Right: foundation.Some(8.0)}),
		widgets.NewPositioned(label("bottom left", 11, secondary, ui.FontWeightW400), widgets.PositionedOpts{Left: foundation.Some(8.0), Bottom: foundation.Some(6.0)}),
	}), widgets.SizedBoxOpts{Height: foundation.Some(90.0)}))
}

// wrapSection shows twelve tags that wrap onto new lines.
func wrapSection() widgets.Widget {
	colors := []ui.Color{primary, secondary, tertiary, errorColor}
	var tags []widgets.Widget
	for i := range 12 {
		c := colors[i%len(colors)]
		tags = append(tags, widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(c), BorderRadius: painting.NewBorderRadiusCircular(12)}),
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(4, 10), label(fmt.Sprintf("tag %d", i+1), 12, white, ui.FontWeightW500))))
	}
	return section("Wrap: twelve tags", widgets.NewWrap(tags, widgets.WrapOpts{Spacing: 6, RunSpacing: 6}))
}

// tableSection shows a Table with a header row and a border.
func tableSection() widgets.Widget {
	cellOf := func(s string, bold bool) widgets.Widget {
		w := ui.FontWeightW400
		if bold {
			w = ui.FontWeightW700
		}
		return widgets.NewPadding(painting.NewEdgeInsetsSymmetric(5, 8), label(s, 12, onSurface, w))
	}
	row := func(bold bool, cells ...string) widgets.TableRow {
		var children []widgets.Widget
		for _, c := range cells {
			children = append(children, cellOf(c, bold))
		}
		return widgets.NewTableRow(children)
	}
	header := widgets.NewTableRow([]widgets.Widget{
		widgets.NewColoredBox(primaryContainer, cellOf("Widget", true)),
		widgets.NewColoredBox(primaryContainer, cellOf("Axis", true)),
		widgets.NewColoredBox(primaryContainer, cellOf("Children", true)),
	})
	border := rendering.NewTableBorderAll(rendering.TableBorderAllOpts{Color: foundation.Some(outline), Width: foundation.Some(1.0)})
	return section("Table: rows and columns", widgets.NewTable([]widgets.TableRow{
		header,
		row(false, "Row", "horizontal", "many"),
		row(false, "Column", "vertical", "many"),
		row(false, "Stack", "layers", "many"),
	}, widgets.TableOpts{Border: &border, DefaultColumnWidth: rendering.NewFlexColumnWidth()}))
}

// layoutApp is the root widget of the example.
type layoutApp struct{ widgets.StatelessWidget }

func newLayoutApp() *layoutApp { return &layoutApp{widgets.NewStatelessWidgetBase()} }

func (*layoutApp) Build(widgets.BuildContext) widgets.Widget {
	body := widgets.NewColumn([]widgets.Widget{
		label("Glutter layout widgets", 24, onSurface, ui.FontWeightW700),
		gap(0, 14),
		widgets.NewRow([]widgets.Widget{
			widgets.NewExpanded(widgets.NewColumn([]widgets.Widget{rowSection(), gap(0, 14), columnSection()})),
			gap(16, 0),
			widgets.NewExpanded(widgets.NewColumn([]widgets.Widget{stackSection(), gap(0, 14), wrapSection(), gap(0, 14), tableSection()})),
		}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
	return widgets.NewDirectionality(ui.TextDirectionLTR,
		widgets.NewColoredBox(surface, widgets.NewPadding(painting.NewEdgeInsetsAll(16), body)))
}

func main() {
	err := widgets.RunApp(newLayoutApp(), widgets.RunAppOpts{
		AppID:  "dev.glutter.layout",
		Window: widgets.WindowConfig{Title: "Glutter – layout", Size: ui.Size{Width: 960, Height: 480}},
	})
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
