// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestCheckboxRadioSwitchSliderThemeData(t *testing.T) {
	states := widgets.NewWidgetStates()
	selected := states.With(widgets.WidgetStateSelected)
	t.Run("checkbox", func(t *testing.T) {
		a := NewCheckboxThemeData(CheckboxThemeDataOpts{FillColor: allOpt(colorA), SplashRadius: foundation.Some(10.0)})
		b := NewCheckboxThemeData(CheckboxThemeDataOpts{FillColor: allOpt(colorB), SplashRadius: foundation.Some(30.0), MaterialTapTargetSize: MaterialTapTargetSizePadded})
		if !a.Equal(a.CopyWith(CheckboxThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality is wrong")
		}
		got := CheckboxThemeDataLerp(a, b, 0.5)
		if got.SplashRadius != foundation.Some(20.0) {
			t.Errorf("splash radius = %v, want 20", got.SplashRadius)
		}
		if got.FillColor == nil || got.FillColor.Resolve(states).IsNone() {
			t.Errorf("fill color = %v", got.FillColor)
		}
	})
	t.Run("checkbox side", func(t *testing.T) {
		side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colorA), Width: foundation.Some(2.0)})
		a := NewCheckboxThemeData(CheckboxThemeDataOpts{Side: foundation.Some(side)})
		b := NewCheckboxThemeData(CheckboxThemeDataOpts{StateSide: widgets.NewWidgetStatePropertyAll(foundation.Some(
			painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colorB), Width: foundation.Some(4.0)})))})
		got := CheckboxThemeDataLerp(a, b, 0.5)
		if v, ok := got.Side.Get(); !ok || v.Width != 3.0 {
			t.Errorf("side = %v, want a width of 3", got.Side)
		}
		if got := CheckboxThemeDataLerp(NewCheckboxThemeData(), NewCheckboxThemeData(), 0.5); got.Side.IsSome() {
			t.Error("the side of nothing is set")
		}
	})
	t.Run("radio", func(t *testing.T) {
		a := NewRadioThemeData(RadioThemeDataOpts{FillColor: allOpt(colorA), SplashRadius: foundation.Some(10.0)})
		b := NewRadioThemeData(RadioThemeDataOpts{FillColor: allOpt(colorB), SplashRadius: foundation.Some(30.0)})
		if a.Equal(b) || !a.Equal(a.CopyWith(RadioThemeDataCopy{})) {
			t.Error("equality is wrong")
		}
		if got := RadioThemeDataLerp(a, b, 0.25); got.SplashRadius != foundation.Some(15.0) {
			t.Errorf("splash radius = %v", got.SplashRadius)
		}
	})
	t.Run("switch", func(t *testing.T) {
		a := NewSwitchThemeData(SwitchThemeDataOpts{ThumbColor: allOpt(colorA), SplashRadius: foundation.Some(10.0)})
		b := NewSwitchThemeData(SwitchThemeDataOpts{ThumbColor: allOpt(colorB), SplashRadius: foundation.Some(30.0), TrackOutlineWidth: allOpt(2.0)})
		if a.Equal(b) || !a.Equal(a.CopyWith(SwitchThemeDataCopy{})) {
			t.Error("equality is wrong")
		}
		got := SwitchThemeDataLerp(a, b, 0.5)
		if got.SplashRadius != foundation.Some(20.0) {
			t.Errorf("splash radius = %v", got.SplashRadius)
		}
		if v := got.ThumbColor.Resolve(selected); v.IsNone() {
			t.Error("the thumb color of the lerp is unset")
		}
	})
	t.Run("slider", func(t *testing.T) {
		a := NewSliderThemeData(SliderThemeDataOpts{TrackHeight: foundation.Some(2.0), ActiveTrackColor: foundation.Some(colorA)})
		b := NewSliderThemeData(SliderThemeDataOpts{TrackHeight: foundation.Some(6.0), ActiveTrackColor: foundation.Some(colorB), ShowValueIndicator: ShowValueIndicatorAlways})
		if a.Equal(b) || !a.Equal(a.CopyWith(SliderThemeDataCopy{})) {
			t.Error("equality is wrong")
		}
		got := SliderThemeDataLerp(a, b, 0.5)
		if got.TrackHeight != foundation.Some(4.0) {
			t.Errorf("track height = %v, want 4", got.TrackHeight)
		}
		if got.ShowValueIndicator != ShowValueIndicatorAlways {
			t.Errorf("show value indicator at 0.5 = %v, want always", got.ShowValueIndicator)
		}
	})
	t.Run("diagnostics", func(t *testing.T) {
		skipUnlessDebug(t)
		for name, s := range map[string]string{
			"checkbox": NewCheckboxThemeData(CheckboxThemeDataOpts{SplashRadius: foundation.Some(10.0)}).String(),
			"radio":    NewRadioThemeData(RadioThemeDataOpts{SplashRadius: foundation.Some(10.0)}).String(),
			"switch":   NewSwitchThemeData(SwitchThemeDataOpts{SplashRadius: foundation.Some(10.0)}).String(),
			"slider":   NewSliderThemeData(SliderThemeDataOpts{TrackHeight: foundation.Some(10.0)}).String(),
		} {
			if !strings.Contains(s, "10") {
				t.Errorf("%s: the description lacks the value: %s", name, s)
			}
		}
	})
}

// TestComponentThemesAreUsed checks that the themes above a checkbox, a radio, a
// switch and a slider give them their values.
func TestSelectionThemesAreUsed(t *testing.T) {
	t.Run("checkbox fill", func(t *testing.T) {
		theme := NewCheckboxTheme(NewCheckboxThemeData(CheckboxThemeDataOpts{FillColor: allOpt(ui.NewColor(0xFFFF0000))}),
			NewCheckbox(foundation.Some(true), func(foundation.Opt[bool]) {}))
		ts := pumpButton(t, standardTheme(true), theme)
		ts.PumpAndSettle()
		c := ts.Center(gluttertest.ByType[*Checkbox]())
		if got := pixel(ts, int(c.Dx-7), int(c.Dy-7)); got.R != 0xFF || got.G != 0 || got.B != 0 {
			t.Errorf("fill = %v, want red", got)
		}
	})
	t.Run("theme data checkbox", func(t *testing.T) {
		data := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), CheckboxTheme: NewCheckboxThemeData(CheckboxThemeDataOpts{SplashRadius: foundation.Some(7.0)})})
		if got := data.CheckboxTheme.SplashRadius; got != foundation.Some(7.0) {
			t.Errorf("splash radius of the theme data = %v", got)
		}
	})
	t.Run("slider track height", func(t *testing.T) {
		theme := NewSliderTheme(NewSliderThemeData(SliderThemeDataOpts{TrackHeight: foundation.Some(20.0), ActiveTrackColor: foundation.Some(ui.NewColor(0xFF00FF00))}),
			widgets.NewSizedBox(NewSlider(1.0, func(float64) {}), widgets.SizedBoxOpts{Height: foundation.Some(48.0)}))
		ts := pumpButton(t, standardTheme(true), theme)
		ts.PumpAndSettle()
		slider := gluttertest.ByType[*Slider]()
		c := ts.Center(slider)
		if got := pixel(ts, int(c.Dx), int(c.Dy-8)); got.G != 0xFF || got.R != 0 {
			t.Errorf("the track 8 pixels above the middle = %v, want green: the track is 20 high", got)
		}
	})
}

func TestChipThemeData(t *testing.T) {
	a := NewChipThemeData(ChipThemeDataOpts{BackgroundColor: foundation.Some(colorA), Elevation: foundation.Some(2.0),
		Padding: painting.NewEdgeInsetsAll(4), Brightness: foundation.Some(ui.BrightnessDark)})
	b := NewChipThemeData(ChipThemeDataOpts{BackgroundColor: foundation.Some(colorB), Elevation: foundation.Some(6.0),
		Padding: painting.NewEdgeInsetsAll(8), ShowCheckmark: foundation.Some(false),
		AvatarBoxConstraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}))})
	t.Run("equality and copy", func(t *testing.T) {
		if !a.Equal(a.CopyWith(ChipThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality is wrong")
		}
		c := a.CopyWith(ChipThemeDataCopy{Elevation: foundation.Some(9.0)})
		if c.Elevation != foundation.Some(9.0) || c.BackgroundColor != a.BackgroundColor {
			t.Errorf("copy = %v", c)
		}
	})
	t.Run("lerp", func(t *testing.T) {
		got := ChipThemeDataLerp(a, b, 0.5)
		if got.Elevation != foundation.Some(4.0) {
			t.Errorf("elevation = %v, want 4", got.Elevation)
		}
		if !got.Padding.Equal(painting.NewEdgeInsetsAll(6)) {
			t.Errorf("padding = %v, want 6 all around", got.Padding)
		}
		// The checkmark and the brightness switch at the half, with their defaults.
		if got.ShowCheckmark != foundation.Some(false) || got.Brightness != foundation.Some(ui.BrightnessLight) {
			t.Errorf("showCheckmark = %v, brightness = %v", got.ShowCheckmark, got.Brightness)
		}
		if got := ChipThemeDataLerp(a, b, 0.25); got.ShowCheckmark != foundation.Some(true) || got.Brightness != foundation.Some(ui.BrightnessDark) {
			t.Errorf("at 0.25: showCheckmark = %v, brightness = %v", got.ShowCheckmark, got.Brightness)
		}
	})
	t.Run("from defaults", func(t *testing.T) {
		tests := []struct {
			name           string
			opts           ChipThemeDataFromDefaultsOpts
			wantBackground ui.Color
			wantBrightness ui.Brightness
		}{
			{"light", ChipThemeDataFromDefaultsOpts{Brightness: ui.BrightnessLight}, ui.NewColor(0x1f000000), ui.BrightnessLight},
			{"dark", ChipThemeDataFromDefaultsOpts{Brightness: ui.BrightnessDark}, ui.NewColor(0x1fffffff), ui.BrightnessDark},
			{"from a primary color", ChipThemeDataFromDefaultsOpts{PrimaryColor: foundation.Some(ui.NewColor(0xFFFFFFFF))}, ui.NewColor(0x1fffffff), ui.BrightnessLight},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				d := ChipThemeDataFromDefaults(ui.NewColor(0xFF00FF00), NewThemeData().TextTheme.BodyLarge, tc.opts)
				if d.BackgroundColor != foundation.Some(tc.wantBackground) {
					t.Errorf("background = %v, want %v", d.BackgroundColor, tc.wantBackground)
				}
				if d.Brightness != foundation.Some(tc.wantBrightness) {
					t.Errorf("brightness = %v, want %v", d.Brightness, tc.wantBrightness)
				}
				if d.SecondarySelectedColor != foundation.Some(ui.NewColor(0x3d00FF00)) {
					t.Errorf("secondary selected color = %v", d.SecondarySelectedColor)
				}
				if d.PressElevation != foundation.Some(8.0) || d.ShowCheckmark != foundation.Some(true) {
					t.Errorf("defaults = %v", d)
				}
			})
		}
	})
	t.Run("a theme above a chip is used", func(t *testing.T) {
		theme := NewChipTheme(NewChipThemeData(ChipThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000)),
			Side: foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFF0000FF))}))}),
			NewChip(chipLabel("themed")))
		ts := pumpButton(t, standardTheme(true), theme)
		ts.PumpAndSettle()
		ink := gluttertest.WidgetAs[*Ink](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*RawChip](), gluttertest.ByType[*Ink]())))
		shape := ink.Decoration().(*painting.ShapeDecoration)
		if shape.Color != foundation.Some(ui.NewColor(0xFFFF0000)) {
			t.Errorf("color = %v, want red", shape.Color)
		}
		if side := shape.Shape.(painting.OutlinedBorder).Side(); side.Color != ui.NewColor(0xFF0000FF) {
			t.Errorf("side = %v, want blue", side)
		}
	})
}

func TestCircleAvatar(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name   string
		avatar widgets.Widget
		want   ui.Size
	}{
		{"default", NewCircleAvatar(), ui.Size{Width: 40, Height: 40}},
		{"radius", NewCircleAvatar(CircleAvatarOpts{Radius: foundation.Some(25.0)}), ui.Size{Width: 50, Height: 50}},
		{"min radius grows to the room", NewCircleAvatar(CircleAvatarOpts{MinRadius: foundation.Some(30.0)}), ui.Size{Width: 400, Height: 600}},
		{"max radius", NewCircleAvatar(CircleAvatarOpts{MaxRadius: foundation.Some(10.0)}), ui.Size{Width: 20, Height: 20}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, theme, tc.avatar)
			ts.PumpAndSettle()
			if got := ts.Size(gluttertest.ByType[*CircleAvatar]()); got != tc.want {
				t.Errorf("size = %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("colors of the theme", func(t *testing.T) {
		ts := pumpButton(t, theme, NewCircleAvatar(CircleAvatarOpts{Child: widgets.NewText("AB")}))
		ts.PumpAndSettle()
		c := ts.Center(gluttertest.ByType[*CircleAvatar]())
		got := pixel(ts, int(c.Dx), int(c.Dy-17))
		want := cs.PrimaryContainer()
		if int(got.R) != want.Red() || int(got.G) != want.Green() || int(got.B) != want.Blue() {
			t.Errorf("background = %v, want %v", got, want)
		}
		text := gluttertest.WidgetAs[*widgets.DefaultTextStyle](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*CircleAvatar](), gluttertest.ByType[*widgets.DefaultTextStyle]())))
		if text.Style().Color != foundation.Some(cs.OnPrimaryContainer()) {
			t.Errorf("text color = %v, want on primary container", text.Style().Color)
		}
	})
	t.Run("foreground follows the brightness of the background", func(t *testing.T) {
		m2 := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)})
		tests := []struct {
			name       string
			background ui.Color
			want       ui.Color
		}{
			{"dark background", ui.NewColor(0xFF000000), m2.PrimaryColorLight},
			{"light background", ui.NewColor(0xFFFFFFFF), m2.PrimaryColorDark},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ts := pumpButton(t, m2, NewCircleAvatar(CircleAvatarOpts{BackgroundColor: foundation.Some(tc.background), Child: widgets.NewText("A")}))
				ts.PumpAndSettle()
				text := gluttertest.WidgetAs[*widgets.DefaultTextStyle](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*CircleAvatar](), gluttertest.ByType[*widgets.DefaultTextStyle]())))
				if text.Style().Color != foundation.Some(tc.want) {
					t.Errorf("text color = %v, want %v", text.Style().Color, tc.want)
				}
			})
		}
	})
}

func TestSelectionAndAvatarErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"checkbox null without tristate", func() { NewCheckbox(foundation.None[bool](), nil) }, "CheckboxValueNull"},
		{"switch image error without image", func() {
			NewSwitch(true, nil, SwitchOpts{OnActiveThumbImageError: func(error) {}})
		}, "SwitchThumbImageErrorWithoutImage"},
		{"switch inactive image error without image", func() {
			NewSwitch(true, nil, SwitchOpts{OnInactiveThumbImageError: func(error) {}})
		}, "SwitchThumbImageErrorWithoutImage"},
		{"circle avatar radius with min", func() {
			NewCircleAvatar(CircleAvatarOpts{Radius: foundation.Some(1.0), MinRadius: foundation.Some(1.0)})
		}, "CircleAvatarRadiusConflict"},
		{"circle avatar radius with max", func() {
			NewCircleAvatar(CircleAvatarOpts{Radius: foundation.Some(1.0), MaxRadius: foundation.Some(1.0)})
		}, "CircleAvatarRadiusConflict"},
		{"circle avatar background error", func() {
			NewCircleAvatar(CircleAvatarOpts{OnBackgroundImageError: func(error) {}})
		}, "CircleAvatarImageErrorWithoutImage"},
		{"circle avatar foreground error", func() {
			NewCircleAvatar(CircleAvatarOpts{OnForegroundImageError: func(error) {}})
		}, "CircleAvatarImageErrorWithoutImage"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
	t.Run("radio without a group", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(materialApp(nil, NewRadio(1, RadioOpts[int]{Enabled: foundation.Some(true)})))
		wantRoute(t, recoverError(func() { takeAndPanic(ts) }), "RadioNoRegistry")
	})
	t.Run("raw radio without a registry", func(t *testing.T) {
		wantRoute(t, recoverError(func() {
			widgets.NewRawRadio(1, nil, false, nil, false, nil, true, func(widgets.BuildContext, *widgets.ToggleableStateMixin) widgets.Widget { return nil })
		}), "RawRadioNoRegistry")
	})
}

func TestRadioGroupSingleSelection(t *testing.T) {
	skipUnlessDebug(t)
	ts := pumpButton(t, standardTheme(true), widgets.NewRadioGroup(func(foundation.Opt[int]) {},
		widgets.NewColumn([]widgets.Widget{NewRadio(1), NewRadio(1)}), widgets.RadioGroupOpts[int]{GroupValue: foundation.Some(1)}))
	ts.Pump()
	wantRoute(t, recoverError(func() { takeAndPanic(ts) }), "RadioGroupMultipleSelection")
}
