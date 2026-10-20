// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func sizedBoxOf(w, h float64) *widgets.SizedBox {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)})
}

func rectOf(ts *gluttertest.Tester, f gluttertest.Finder) ui.Rect {
	tl, size := ts.TopLeft(f), ts.Size(f)
	return ui.NewRectFromLTWH(tl.Dx, tl.Dy, size.Width, size.Height)
}

func TestAppBarBottomAndFlexibleSpace(t *testing.T) {
	bottom := &fixedPreferredSize{size: ui.Size{Width: 400, Height: 40}}
	flexible := widgets.NewSizedBoxExpand(nil, widgets.WidgetOpts{Key: foundation.NewValueKey("flexible")})
	ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
		AppBar: NewAppBar(AppBarOpts{Title: widgets.NewText("T"), Bottom: bottom, FlexibleSpace: flexible}),
	}))
	if got := ts.Size(gluttertest.ByType[*AppBar]()).Height; got != 56+40 {
		t.Errorf("app bar height = %v, want the toolbar plus the bottom", got)
	}
	if got := rectOf(ts, gluttertest.ByWidget(bottom)); got != ui.NewRectFromLTWH(0, 56, 400, 40) {
		t.Errorf("bottom = %v, want it under the toolbar", got)
	}
	if got := rectOf(ts, gluttertest.ByKey(foundation.NewValueKey("flexible"))); got != ui.NewRectFromLTWH(0, 0, 400, 96) {
		t.Errorf("flexible space = %v, want the whole app bar", got)
	}
	if got := NewAppBar(AppBarOpts{Bottom: bottom}).PreferredSize().Height; got != 96 {
		t.Errorf("preferred height = %v, want 96", got)
	}
}

// fixedPreferredSize is a box with a preferred size, for AppBar.bottom.
type fixedPreferredSize struct {
	widgets.StatelessWidget
	size ui.Size
}

func (w *fixedPreferredSize) PreferredSize() ui.Size { return w.size }

func (w *fixedPreferredSize) Build(widgets.BuildContext) widgets.Widget {
	return sizedBoxOf(w.size.Width, w.size.Height)
}

func TestAppBarLeadingAndTitleLayout(t *testing.T) {
	const w = 400.0
	title := sizedBoxOf(80, 20)
	tests := []struct {
		name      string
		dir       ui.TextDirection
		o         AppBarOpts
		wantTitle ui.Rect
	}{
		{"leading width of the app bar", ui.TextDirectionLTR, AppBarOpts{Leading: sizedBoxOf(10, 10), LeadingWidth: foundation.Some(100.0)}, ui.NewRectFromLTWH(100+16, 18, 80, 20)},
		{"title spacing", ui.TextDirectionLTR, AppBarOpts{TitleSpacing: foundation.Some(0.0)}, ui.NewRectFromLTWH(0, 18, 80, 20)},
		{"right to left", ui.TextDirectionRTL, AppBarOpts{}, ui.NewRectFromLTWH(w-16-80, 18, 80, 20)},
		{"centered with actions", ui.TextDirectionLTR, AppBarOpts{CenterTitle: foundation.Some(true), Actions: []widgets.Widget{sizedBoxOf(48, 48)}}, ui.NewRectFromLTWH((w-80)/2, 18, 80, 20)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.Title = title
			ts := pumpScaffold(t, tc.dir, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{AppBar: NewAppBar(o)}))
			if got := rectOf(ts, gluttertest.ByWidget(title)); got != tc.wantTitle {
				t.Errorf("title = %v, want %v", got, tc.wantTitle)
			}
		})
	}
}

func TestAppBarForegroundColor(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	var titleStyle *painting.TextStyle
	var iconTheme widgets.IconThemeData
	var actionIconTheme widgets.IconThemeData
	title := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		titleStyle = widgets.DefaultTextStyleOf(bc).Style()
		return widgets.NewText("T")
	})
	action := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		actionIconTheme = widgets.IconThemeOf(bc)
		return widgets.NewSizedBox(nil)
	})
	leading := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		iconTheme = widgets.IconThemeOf(bc)
		return widgets.NewSizedBox(nil)
	})
	pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
		AppBar: NewAppBar(AppBarOpts{Title: title, Leading: leading, Actions: []widgets.Widget{action}, ForegroundColor: foundation.Some(red)}),
	}))
	if got := titleStyle.Color.OrElse(ui.Color{}); got != red {
		t.Errorf("title color = %v, want red", got)
	}
	if got := iconTheme.Color().OrElse(ui.Color{}); got != red {
		t.Errorf("leading icon color = %v, want red", got)
	}
	if got := actionIconTheme.Color().OrElse(ui.Color{}); got != red {
		t.Errorf("action icon color = %v, want red", got)
	}
	if got := iconTheme.Size().OrElse(0); got != 24 {
		t.Errorf("icon size = %v, want 24", got)
	}
}

func TestAppBarScrolledUnderElevation(t *testing.T) {
	tests := []struct {
		name      string
		o         AppBarOpts
		scroll    float64
		wantElev  float64
		wantTinct bool
	}{
		{"at rest", AppBarOpts{}, 0, 0, false},
		{"scrolled under", AppBarOpts{}, 100, 3, true},
		{"custom scrolled under elevation", AppBarOpts{ScrolledUnderElevation: foundation.Some(9.0)}, 100, 9, true},
		{"elevation at rest", AppBarOpts{Elevation: foundation.Some(2.0)}, 0, 2, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.Title = widgets.NewText("T")
			controller := widgets.NewScrollController()
			t.Cleanup(controller.Dispose)
			ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(o),
				Body:   widgets.NewListView(manyBoxesOf(50, 100), widgets.ListViewOpts{Controller: controller}),
			}))
			controller.JumpTo(tc.scroll)
			ts.PumpAndSettle()
			material := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*AppBar](), gluttertest.ByType[*Material]()))
			if got := gluttertest.WidgetAs[*Material](ts, material).Elevation(); got != tc.wantElev {
				t.Errorf("elevation = %v, want %v", got, tc.wantElev)
			}
		})
	}
}

func manyBoxesOf(n int, height float64) []widgets.Widget {
	out := make([]widgets.Widget, n)
	for i := range out {
		out[i] = sizedBoxOf(100, height)
	}
	return out
}

func TestAppBarSystemOverlayStyle(t *testing.T) {
	tests := []struct {
		name  string
		theme *ThemeData
		o     AppBarOpts
		want  ui.Brightness
	}{
		// The status bar icons contrast the background: dark icons on the light
		// Material 3 surface, light icons on the dark one.
		{"light theme", NewThemeData(), AppBarOpts{}, ui.BrightnessDark},
		{"dark theme", NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark}), AppBarOpts{}, ui.BrightnessLight},
		{"given", NewThemeData(), AppBarOpts{SystemOverlayStyle: foundation.Some(services.SystemUiOverlayStyleDark)}, services.SystemUiOverlayStyleDark.StatusBarIconBrightness},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(tc.theme, widgets.NewMediaQuery(widgets.NewMediaQueryData(),
				NewScaffold(ScaffoldOpts{AppBar: NewAppBar(tc.o)})))))
			regions := ts.Widgets(gluttertest.ByType[*widgets.AnnotatedRegion]())
			found := false
			for _, r := range regions {
				if style, ok := r.(*widgets.AnnotatedRegion).Value().(services.SystemUiOverlayStyle); ok {
					found = true
					if style.StatusBarIconBrightness != tc.want {
						t.Errorf("status bar icon brightness = %v, want %v", style.StatusBarIconBrightness, tc.want)
					}
				}
			}
			if !found {
				t.Error("no AnnotatedRegion of a SystemUiOverlayStyle")
			}
		})
	}
}

func TestAppBarNegativeElevation(t *testing.T) {
	skipUnlessDebug(t)
	appErr, ok := errors.FromError(recoverError(func() { NewAppBar(AppBarOpts{Elevation: foundation.Some(-1.0)}) }))
	if !ok || !appErr.HasRoute("AppBarElevationNegative") {
		t.Errorf("error = %v", appErr)
	}
}

func TestAppBarIconButtonThemes(t *testing.T) {
	red, green := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00)
	theme := NewThemeData(ThemeDataOpts{VisualDensity: foundation.Some(VisualDensityStandard()), MaterialTapTargetSize: MaterialTapTargetSizeShrinkWrap})
	icon := func(ts *gluttertest.Tester, button gluttertest.Finder) widgets.IconThemeData {
		return widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(button, gluttertest.ByType[*widgets.Icon]()))))
	}
	pump := func(bar *AppBar, drawer widgets.Widget) *gluttertest.Tester {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{AppBar: bar, Drawer: drawer})}))
		return ts
	}

	t.Run("the leading drawer button is an icon button, centered in 56 pixels", func(t *testing.T) {
		ts := pump(NewAppBar(), widgets.NewSizedBox(nil))
		button := gluttertest.ByType[*DrawerButton]()
		if !ts.Exists(gluttertest.Descendant(button, gluttertest.ByType[*IconButton]())) {
			t.Fatal("the drawer button has no icon button")
		}
		if got := ts.TopLeft(gluttertest.ByType[*IconButton]()).Dx; got != 8 {
			t.Errorf("left of the icon button = %v, want 8 (centered in 56)", got)
		}
		if got := ts.Size(gluttertest.ByType[*IconButton]()); got != (ui.Size{Width: 40, Height: 40}) {
			t.Errorf("size of the icon button = %v, want 40x40", got)
		}
	})
	t.Run("the icon theme of the app bar colors the leading button", func(t *testing.T) {
		ts := pump(NewAppBar(AppBarOpts{IconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(red), Size: foundation.Some(30.0)}))}), widgets.NewSizedBox(nil))
		it := icon(ts, gluttertest.ByType[*DrawerButton]())
		if it.Color() != foundation.Some(red) || it.Size() != foundation.Some(30.0) {
			t.Errorf("icon theme = %v, want red and 30", it)
		}
	})
	t.Run("the actions icon theme colors the action buttons", func(t *testing.T) {
		ts := pump(NewAppBar(AppBarOpts{
			ActionsIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(green)})),
			Actions:          []widgets.Widget{NewIconButton(func() {}, NewIcon(Icons.Search()))}}), nil)
		it := icon(ts, gluttertest.ByType[*IconButton]())
		if it.Color() != foundation.Some(green) {
			t.Errorf("icon color = %v, want green", it.Color())
		}
	})
	t.Run("the end drawer button takes the size of the icon theme", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{
			AppBar:    NewAppBar(AppBarOpts{IconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(36.0)}))}),
			EndDrawer: widgets.NewSizedBox(nil)})}))
		if !ts.Exists(gluttertest.ByType[*EndDrawerButton]()) {
			t.Fatal("there is no end drawer button")
		}
		if got := icon(ts, gluttertest.ByType[*EndDrawerButton]()).Size(); got != foundation.Some(36.0) {
			t.Errorf("icon size = %v, want 36", got)
		}
	})
}
