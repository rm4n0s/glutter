package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// controlsPage lays the controls out as the proof of step 34.4: a row for each control, in its states.
func controlsPage() widgets.Widget {
	pad := func(w widgets.Widget) widgets.Widget { return widgets.NewPadding(painting.NewEdgeInsetsAll(6), w) }
	row := func(children ...widgets.Widget) widgets.Widget {
		return widgets.NewRow(children, widgets.RowOpts{MainAxisSize: 1})
	}
	on, off, mixed := foundation.Some(true), foundation.Some(false), foundation.None[bool]()
	noop := func(foundation.Opt[bool]) {}
	radio := func(selected, enabled bool, check bool) widgets.Widget {
		var value foundation.Opt[int]
		if selected {
			value = foundation.Some(1)
		}
		return widgets.NewRadioGroup(func(foundation.Opt[int]) {}, cupertino.NewCupertinoRadio(1, cupertino.CupertinoRadioOpts[int]{
			Enabled: foundation.Some(enabled), UseCheckmarkStyle: check}), widgets.RadioGroupOpts[int]{GroupValue: value})
	}
	slider := func(v float64, onChanged func(float64), opts cupertino.CupertinoSliderOpts) widgets.Widget {
		return cupertino.NewCupertinoSlider(v, onChanged, opts)
	}
	changed := func(float64) {}
	return widgets.NewColumn([]widgets.Widget{
		row(pad(cupertino.NewCupertinoCheckbox(off, noop)), pad(cupertino.NewCupertinoCheckbox(on, noop)), pad(cupertino.NewCupertinoCheckbox(mixed, noop, cupertino.CupertinoCheckboxOpts{Tristate: true})),
			pad(cupertino.NewCupertinoCheckbox(on, nil)), pad(cupertino.NewCupertinoCheckbox(off, nil))),
		row(pad(radio(false, true, false)), pad(radio(true, true, false)), pad(radio(true, true, true)), pad(radio(true, false, false)), pad(radio(false, false, false))),
		row(pad(slider(0.0, changed, cupertino.CupertinoSliderOpts{})), pad(slider(0.5, changed, cupertino.CupertinoSliderOpts{}))),
		row(pad(slider(0.7, changed, cupertino.CupertinoSliderOpts{Divisions: foundation.Some(4), ActiveColor: cupertino.CupertinoColors.SystemGreen})), pad(slider(0.5, nil, cupertino.CupertinoSliderOpts{}))),
		row(pad(cupertino.NewCupertinoActivityIndicator()), pad(cupertino.NewCupertinoActivityIndicator(cupertino.CupertinoActivityIndicatorOpts{Radius: foundation.Some(16.0), NotAnimating: true})),
			pad(cupertino.NewCupertinoActivityIndicatorPartiallyRevealed(cupertino.CupertinoActivityIndicatorPartiallyRevealedOpts{Radius: foundation.Some(16.0), Progress: foundation.Some(0.6)})),
			pad(widgets.NewSizedBox(cupertino.NewCupertinoLinearActivityIndicator(0.4), widgets.SizedBoxOpts{Width: foundation.Some(100.0)}))),
		pad(cupertino.NewCupertinoSegmentedControl(func(string) {}, segments("Day", "Week", "Month"), cupertino.CupertinoSegmentedControlOpts[string]{GroupValue: foundation.Some("Week"), DisabledChildren: []string{"Month"}})),
		pad(cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("Day", "Week", "Month"), cupertino.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: foundation.Some("Week")})),
		pad(cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("Day", "Week", "Month"), cupertino.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: foundation.Some("Day"), DisabledChildren: []string{"Month"}})),
	}, widgets.ColumnOpts{MainAxisSize: 1})
}

// TestCupertinoControlsGoldens draws the controls of step 34.4 in light and dark: the checkbox, the radio, the
// slider, the activity indicators and the two segmented controls, enabled and disabled.
func TestCupertinoControlsGoldens(t *testing.T) {
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
	}{{"cupertino.controls.light", ui.BrightnessLight}, {"cupertino.controls.dark", ui.BrightnessDark}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 440}})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(controlsPage()))}))
			ts.Pump()
			ts.MatchesGolden(tt.name)
		})
	}
}
