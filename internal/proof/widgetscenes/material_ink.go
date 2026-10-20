// Package widgetscenes holds the proof scenes that are made by pumping a widget
// tree through gluttertest. They live apart from package proof because gluttertest
// depends on the software renderer, whose tests use package proof.
//
// Flutter: no counterpart
package widgetscenes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func init() {
	proof.Register(proof.Scene{
		Name: "material-ink",
		Description: "Material surfaces at the elevations 0, 1, 3, 6, 8 and 12, in the light theme (top) and the dark theme (bottom), " +
			"and below each row an InkWell whose ripple is frozen in the middle of its animation",
		Width:  760,
		Height: 560,
		Build:  func() *ui.Scene { return MaterialInkScene() },
	})
}

// testingTB is the TB of a Tester that nobody runs as a test: a failure panics.
type testingTB struct{ cleanups []func() }

func (*testingTB) Helper() {}
func (*testingTB) Errorf(format string, args ...any) {
	panic(fmt.Sprintf("proof scene: "+format, args...))
}
func (*testingTB) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf("proof scene: "+format, args...))
}
func (*testingTB) Fatal(args ...any)   { panic(fmt.Sprint(append([]any{"proof scene: "}, args...)...)) }
func (*testingTB) Logf(string, ...any) {}
func (tb *testingTB) Cleanup(f func()) { tb.cleanups = append(tb.cleanups, f) }
func (tb *testingTB) close() {
	for i := len(tb.cleanups) - 1; i >= 0; i-- {
		tb.cleanups[i]()
	}
}

// MaterialInkScene renders the Material surfaces and the frozen ripple of the
// step 12.3 proof through gluttertest: a widget tree pumped with a fake clock.
//
// Flutter: no counterpart
func MaterialInkScene() *ui.Scene {
	tb := &testingTB{}
	defer tb.close()
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 760, Height: 560}})

	panel := func(theme *material.ThemeData, name string) widgets.Widget {
		scheme := theme.ColorScheme
		var surfaces []widgets.Widget
		for _, e := range []int{0, 1, 3, 6, 8, 12} {
			surfaces = append(surfaces, widgets.NewSizedBox(material.NewMaterial(
				widgets.NewCenter(widgets.NewText(strconv.Itoa(e))),
				material.MaterialOpts{
					Elevation: float64(e), Color: foundation.Some(scheme.Surface()), SurfaceTintColor: foundation.Some(scheme.SurfaceTint()),
					Shape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusCircular(12)}),
				}), widgets.SizedBoxOpts{Width: foundation.Some(88.0), Height: foundation.Some(88.0)}))
		}
		ripple := widgets.NewSizedBox(material.NewMaterial(
			material.NewInkWell(widgets.NewCenter(widgets.NewText(name+": press and hold")), material.InkWellOpts{
				Key: foundation.NewValueKey(name), OnTap: func() {}, BorderRadius: foundation.Some(painting.NewBorderRadiusCircular(12)),
			}),
			material.MaterialOpts{Elevation: 1, Color: foundation.Some(scheme.SurfaceContainerHighest()),
				Shape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusCircular(12)})}),
			widgets.SizedBoxOpts{Width: foundation.Some(320.0), Height: foundation.Some(64.0)})
		return material.NewTheme(theme, widgets.NewColoredBox(theme.ScaffoldBackgroundColor,
			widgets.NewSizedBox(widgets.NewColumn([]widgets.Widget{
				widgets.NewRow(surfaces, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceEvenly}),
				ripple,
			}, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceEvenly}),
				widgets.SizedBoxOpts{Width: foundation.Some(760.0), Height: foundation.Some(280.0)})))
	}
	light := material.NewThemeData(material.ThemeDataOpts{FontFamily: "Roboto"})
	dark := material.NewThemeData(material.ThemeDataOpts{Brightness: ui.BrightnessDark, FontFamily: "Roboto"})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		panel(light, "light"), panel(dark, "dark"),
	})))
	ts.PumpAndSettle()
	for _, name := range []string{"light", "dark"} {
		ts.StartGesture(ts.Center(gluttertest.ByKey(foundation.NewValueKey(name))))
	}
	ts.Pump(150 * time.Millisecond) // the taps are recognized and the ripples start
	ts.Pump(250 * time.Millisecond) // frozen in the middle of the animation
	scenes := ts.Scenes()
	return scenes[len(scenes)-1]
}
