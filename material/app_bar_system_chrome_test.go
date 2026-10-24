// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// useAndroid makes Android the target platform of the test, as flutter_test's default is: the
// navigation bar of the style is only for Android.
func useAndroid(t *testing.T) {
	t.Helper()
	skipUnlessDebug(t) // the platform override is a debug-only feature
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

// Ported from packages/flutter/test/material/app_bar_test.dart: the cases that read
// SystemChrome.latestStyle after the app bar's AnnotatedRegion was painted.
func TestAppBarSetsTheSystemChromeStyle(t *testing.T) {
	red := ui.NewColor(0xFFF44336)
	green := ui.NewColor(0xFF4CAF50)
	transparent := Colors.Transparent
	lightBlue := Colors.LightBlue
	tests := []struct {
		name  string
		theme *ThemeData
		o     AppBarOpts
		want  services.SystemUiOverlayStyle
	}{
		{
			"Material3 - AppBar draws a light system bar for a dark background",
			NewThemeDataDark(true), AppBarOpts{},
			services.SystemUiOverlayStyle{
				StatusBarColor:          foundation.Some(transparent),
				StatusBarBrightness:     ui.BrightnessDark,
				StatusBarIconBrightness: ui.BrightnessLight,
			},
		},
		{
			"Material3 - AppBar draws a dark system bar for a light background",
			NewThemeData(), AppBarOpts{},
			services.SystemUiOverlayStyle{
				StatusBarColor:          foundation.Some(transparent),
				StatusBarBrightness:     ui.BrightnessLight,
				StatusBarIconBrightness: ui.BrightnessDark,
			},
		},
		{
			"Material2 - AppBar draws a light system bar for a dark background",
			NewThemeDataDark(false), AppBarOpts{},
			services.SystemUiOverlayStyle{
				StatusBarBrightness:     ui.BrightnessDark,
				StatusBarIconBrightness: ui.BrightnessLight,
			},
		},
		{
			"Material2 - AppBar draws a dark system bar for a light background",
			NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false), PrimarySwatch: &lightBlue}), AppBarOpts{},
			services.SystemUiOverlayStyle{
				StatusBarBrightness:     ui.BrightnessLight,
				StatusBarIconBrightness: ui.BrightnessDark,
			},
		},
		{
			"AppBar systemOverlayStyle is use to style status bar and navigation bar",
			NewThemeData(),
			AppBarOpts{SystemOverlayStyle: foundation.Some(services.SystemUiOverlayStyleLight.CopyWith(services.SystemUiOverlayStyleCopy{
				StatusBarColor:           foundation.Some(red),
				SystemNavigationBarColor: foundation.Some(green),
			}))},
			services.SystemUiOverlayStyleLight.CopyWith(services.SystemUiOverlayStyleCopy{
				StatusBarColor:           foundation.Some(red),
				SystemNavigationBarColor: foundation.Some(green),
			}),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(tc.theme, widgets.NewMediaQuery(widgets.NewMediaQueryData(),
				NewScaffold(ScaffoldOpts{AppBar: NewAppBar(tc.o), Body: widgets.NewSizedBoxExpand(nil)})))))
			ts.FlushPosted()
			got, ok := ts.Binding.SystemChrome().LatestStyle()
			if !ok {
				t.Fatal("the app bar's style did not reach SystemChrome")
			}
			if got != tc.want {
				t.Errorf("SystemChrome.LatestStyle = %v, want %v", got, tc.want)
			}
		})
	}
}

// Ported from app_bar_test.dart ('Material3 - Default system bar brightness based on AppBar
// background color brightness.'): the style follows the theme when it changes.
func TestAppBarSystemChromeStyleFollowsTheTheme(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	pump := func(theme *ThemeData) {
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(theme, widgets.NewMediaQuery(widgets.NewMediaQueryData(),
			NewScaffold(ScaffoldOpts{AppBar: NewAppBar(AppBarOpts{Title: widgets.NewText("Title")})})))))
		ts.FlushPosted()
	}
	tests := []struct {
		name  string
		theme *ThemeData
		want  ui.Brightness
	}{
		{"light theme", NewThemeData(), ui.BrightnessLight},
		{"dark theme", NewThemeDataDark(), ui.BrightnessDark},
	}
	for _, tc := range tests {
		pump(tc.theme)
		got, ok := ts.Binding.SystemChrome().LatestStyle()
		if !ok || got.StatusBarBrightness != tc.want {
			t.Errorf("%s: status bar brightness = %v (sent %v), want %v", tc.name, got.StatusBarBrightness, ok, tc.want)
		}
	}
}
