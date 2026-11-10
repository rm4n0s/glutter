// Command inputapp is the app of the input tests of step 35.4: a view that takes the focus and prints a line for each tap, each pan and each
// key event, so that the Simulator test reads what the framework got from the touches and the keys of a script (GLUTTER_DEV_EVENTS).
package main

import (
	"fmt"
	"log"

	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/iostest/inputlines"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	var panX, panY float64
	pan := m.NewGestureDetector(m.NewColoredBox(ui.NewColor(0xFFCCE5FF), m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(1e6), Height: m.Some(1e6)})), m.GestureDetectorOpts{
		OnTapDown: func(d gestures.TapDownDetails) {
			fmt.Printf("%s %d %d %v\n", inputlines.Tap, int(d.GlobalPosition.Dx), int(d.GlobalPosition.Dy), d.Kind.OrElse(ui.PointerDeviceKindUnknown))
		},
		OnPanDown:   func(gestures.DragDownDetails) { panX, panY = 0, 0 },
		OnPanUpdate: func(d gestures.DragUpdateDetails) { panX, panY = panX+d.Delta.Dx, panY+d.Delta.Dy },
		OnPanEnd:    func(gestures.DragEndDetails) { fmt.Printf("%s %d %d\n", inputlines.Pan, int(panX), int(panY)) },
	})
	root := m.NewFocus(pan, m.FocusOpts{
		Autofocus: true,
		OnKeyEvent: func(_ *m.FocusNode, ev services.KeyEvent) m.KeyEventResult {
			kind := "up"
			if _, down := ev.(*services.KeyDownEvent); down {
				kind = "down"
			}
			fmt.Printf("%s %s %s %s %q\n", inputlines.Key, kind, ev.PhysicalKey().DebugName(), ev.LogicalKey().DebugName(), ev.Character())
			return m.KeyEventResultHandled
		},
	})
	app := m.NewMaterialApp(m.MaterialAppOpts{Title: "Input", DebugShowCheckedModeBanner: m.Some(false), Home: m.NewScaffold(m.ScaffoldOpts{Body: root})})
	if err := m.RunApp(app, m.RunAppOpts{AppID: "dev.glutter.inputapp"}); err != nil {
		log.Fatal(err)
	}
}
