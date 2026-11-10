package material_test

import (
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are the iOS and macOS cases of the adaptive widgets of Material: Flutter's adaptive constructors and
// platform branches build the Cupertino widgets there (SPEC §31.3), and the Material ones on the other platforms.

func pumpAdaptive(t *testing.T, platform foundation.TargetPlatform, child widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := pumpAdaptiveNoSettle(t, platform, child)
	ts.PumpAndSettle()
	return ts
}

func pumpAdaptiveNoSettle(t *testing.T, platform foundation.TargetPlatform, child widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform)})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(child)})}))
	ts.Pump()
	return ts
}

func TestAdaptiveWidgetsBuildCupertinoOnApplePlatforms(t *testing.T) {
	noopBool := func(foundation.Opt[bool]) {}
	tests := []struct {
		name  string
		build func() widgets.Widget
		// cupertino is whether the widget is a Cupertino one on the Apple platforms.
		cupertino func(ts *gluttertest.Tester) bool
	}{
		{"Checkbox.adaptive", func() widgets.Widget { return NewCheckboxAdaptive(foundation.Some(true), noopBool) },
			func(ts *gluttertest.Tester) bool {
				return ts.Exists(gluttertest.ByType[*cupertino.CupertinoCheckbox]())
			}},
		{"Radio.adaptive", func() widgets.Widget {
			return NewRadioAdaptive(1, RadioOpts[int]{GroupValue: foundation.Some(1), OnChanged: func(foundation.Opt[int]) {}})
		}, func(ts *gluttertest.Tester) bool {
			return ts.Exists(gluttertest.ByType[*cupertino.CupertinoRadio[int]]())
		}},
		{"Slider.adaptive", func() widgets.Widget { return NewSliderAdaptive(0.5, func(float64) {}) },
			func(ts *gluttertest.Tester) bool { return ts.Exists(gluttertest.ByType[*cupertino.CupertinoSlider]()) }},
		{"CircularProgressIndicator.adaptive", func() widgets.Widget { return NewCircularProgressIndicatorAdaptive() },
			func(ts *gluttertest.Tester) bool {
				return ts.Exists(gluttertest.ByType[*cupertino.CupertinoActivityIndicator]())
			}},
		{"AlertDialog.adaptive", func() widgets.Widget {
			return NewAlertDialogAdaptive(AlertDialogOpts{Content: widgets.NewText("CONTENT")})
		}, func(ts *gluttertest.Tester) bool {
			return ts.Exists(gluttertest.ByType[*cupertino.CupertinoAlertDialog]())
		}},
		{"SwitchListTile.adaptive", func() widgets.Widget {
			return widgets.NewSizedBox(NewSwitchListTileAdaptive(true, func(bool) {}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)})
		}, func(ts *gluttertest.Tester) bool { return ts.Exists(gluttertest.ByType[*Switch]()) }},
	}
	for _, tt := range tests {
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux} {
			t.Run(tt.name+"/"+platform.Name(), func(t *testing.T) {
				withPlatform(t, platform)
				ts := pumpAdaptiveNoSettle(t, platform, tt.build())
				ts.Pump()
				apple := platform == foundation.TargetPlatformIOS || platform == foundation.TargetPlatformMacOS
				if tt.name == "SwitchListTile.adaptive" {
					// The tile holds a Material switch that paints itself the Cupertino way.
					if !tt.cupertino(ts) {
						t.Error("no switch")
					}
					return
				}
				if got := tt.cupertino(ts); got != apple {
					t.Errorf("a Cupertino widget is built = %v, want %v; errors: %v", got, apple, ts.Errors())
				}
			})
		}
	}
}

func TestAdaptiveSwitchGoldens(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("goldens are drawn on the host")
	}
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS, foundation.TargetPlatformAndroid} {
		t.Run(platform.Name(), func(t *testing.T) {
			withPlatform(t, platform)
			row := widgets.NewRow([]widgets.Widget{
				NewSwitchAdaptive(false, func(bool) {}), NewSwitchAdaptive(true, func(bool) {}), NewSwitchAdaptive(true, nil),
				NewSwitch(true, func(bool) {}),
			}, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin})
			ts := pumpAdaptive(t, platform, row)
			ts.MatchesGolden("switch.adaptive." + platform.Name())
		})
	}
}

func TestMaterialBasedCupertinoThemeData(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformIOS)})
	data := MaterialBasedCupertinoThemeData(theme)
	if got, want := data.Brightness(), theme.Brightness(); got != want {
		t.Errorf("brightness = %v, want %v", got, want)
	}
	if got, want := data.PrimaryColor().(ui.Color), theme.ColorScheme.Primary(); got != want {
		t.Errorf("primary color = %v, want %v", got, want)
	}
	over := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformIOS),
		CupertinoOverrideTheme: cupertino.NewNoDefaultCupertinoThemeData(cupertino.NoDefaultCupertinoThemeDataOpts{PrimaryColor: ui.NewColor(0xFF123456)})})
	if got := MaterialBasedCupertinoThemeData(over).PrimaryColor().(ui.Color); got != ui.NewColor(0xFF123456) {
		t.Errorf("primary color with an override = %v", got)
	}
}

func TestTextFieldTakesTheCupertinoThemeOnApplePlatforms(t *testing.T) {
	override := ui.NewColor(0xFF123456)
	tests := []struct {
		platform  foundation.TargetPlatform
		cupertino bool
	}{
		{foundation.TargetPlatformIOS, true},
		{foundation.TargetPlatformMacOS, true},
		{foundation.TargetPlatformAndroid, false},
	}
	for _, tt := range tests {
		t.Run(tt.platform.Name(), func(t *testing.T) {
			withPlatform(t, tt.platform)
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(tt.platform),
				CupertinoOverrideTheme: cupertino.NewNoDefaultCupertinoThemeData(cupertino.NoDefaultCupertinoThemeDataOpts{PrimaryColor: override})})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				widgets.NewDefaultSelectionStyle( // The one of the app has the primary color of the Material theme.
					widgets.NewSizedBox(NewTextField(TextFieldOpts{Autofocus: true}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}),
					widgets.DefaultSelectionStyleOpts{}))})}))
			ts.Pump()
			render := gluttertest.StateOf[*widgets.EditableTextState](ts, gluttertest.ByType[*widgets.EditableText]()).RenderEditable()
			// Without a color in the selection style, the cursor is the primary color of the Cupertino theme on iOS and
			// macOS, and the one of the color scheme elsewhere.
			want := theme.ColorScheme.Primary()
			if tt.cupertino {
				want = override
			}
			if cursor, _ := render.CursorColor().Get(); cursor != want {
				t.Errorf("the cursor color is %v, want %v", cursor, want)
			}
			if _, ok := render.CursorRadius().Get(); ok != tt.cupertino {
				t.Errorf("a cursor radius = %v, want %v", ok, tt.cupertino)
			}
			if got := render.CursorOffset().Dx != 0; got != (tt.platform == foundation.TargetPlatformIOS || tt.platform == foundation.TargetPlatformMacOS) {
				t.Errorf("the cursor offset is %v", render.CursorOffset())
			}
		})
	}
}
