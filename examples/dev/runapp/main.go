// Command runapp is a development example (step 7.4): it runs the widgets
// binding with widgets.RunApp and shows a render tree made of widgets, to check
// the whole path from RunApp to the window. It uses the internal packages,
// before the basic widgets exist.
//
// Run it with: go run ./examples/dev/runapp
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>.
package main

import (
	"fmt"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// swatch is a leaf render object widget: a colored box of a fixed size.
type swatch struct {
	widgets.LeafRenderObjectWidget
	color         ui.Color
	width, height float64
}

func newSwatch(c ui.Color, w, h float64) *swatch {
	return &swatch{LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(), color: c, width: w, height: h}
}

func (s *swatch) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(s.color)}),
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: s.width, Height: s.height}), nil))
}

func (s *swatch) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	ro.(*rendering.RenderDecoratedBox).Child().(*rendering.RenderConstrainedBox).SetAdditionalConstraints(
		rendering.NewBoxConstraintsTight(ui.Size{Width: s.width, Height: s.height}))
}

// row is a horizontal flex.
type row struct {
	widgets.MultiChildRenderObjectWidget
}

func (r *row) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: painting.AxisHorizontal,
		TextDirection: ui.TextDirectionLTR, MainAxisAlignment: rendering.MainAxisAlignmentCenter,
		CrossAxisAlignment: rendering.CrossAxisAlignmentCenter})
}

// app builds three swatches whose sizes follow the size of the window.
type app struct{ widgets.StatelessWidget }

func (app) Build(bc widgets.BuildContext) widgets.Widget {
	scale := widgets.MediaQuerySizeOf(bc).Height / 400
	return &row{widgets.NewMultiChildRenderObjectWidgetBase([]widgets.Widget{
		newSwatch(ui.NewColor(0xFF6750A4), 120*scale, 120*scale),
		newSwatch(ui.NewColor(0xFFEADDFF), 80*scale, 200*scale),
		newSwatch(ui.NewColor(0xFF21005D), 160*scale, 60*scale),
	})}
}

func main() {
	err := widgets.RunApp(&app{widgets.NewStatelessWidgetBase()}, widgets.RunAppOpts{
		AppID:  "dev.glutter.runapp",
		Window: widgets.WindowConfig{Title: "Glutter RunApp", Size: ui.Size{Width: 640, Height: 400}},
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
