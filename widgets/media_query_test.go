package widgets

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/media_query_test.dart: the tests of
// MediaQuery.of and maybeOf, MediaQueryData.fromView (with and without the
// platform data), MediaQuery.fromView (the notifications, with and without
// parent data), copyWith, removePadding, removeViewInsets, removeViewPadding,
// the accessors of the aspects, the partial dependencies and the independent
// width and height. The display features and corner radii, supportsAnnounce and
// the text style overrides are not in Glutter (SPEC §12.10), nor are the
// WidgetsApp based tests and the hash code ones (Go has no hashCode).

// fullData is the data of the removePadding tests of Flutter.
func fullData() MediaQueryData {
	return NewMediaQueryData(MediaQueryDataOpts{
		Size:                  ui.Size{Width: 2, Height: 4},
		DevicePixelRatio:      2,
		TextScaler:            painting.NewTextScalerLinear(1.2),
		Padding:               painting.NewEdgeInsetsFromLTRB(3, 1, 2, 4),
		ViewPadding:           painting.NewEdgeInsetsFromLTRB(10, 6, 8, 12),
		ViewInsets:            painting.NewEdgeInsetsFromLTRB(7, 5, 6, 8),
		AlwaysUse24HourFormat: true, AccessibleNavigation: true, InvertColors: true, DisableAnimations: true,
		BoldText: true, HighContrast: true, OnOffSwitchLabels: true, NavigationMode: NavigationModeDirectional,
	})
}

// dataSeenBelow pumps a MediaQuery with the data above a function that
// transforms it with a child MediaQuery, and returns what the child reads.
func dataSeenBelow(t *testing.T, data MediaQueryData, wrap func(bc BuildContext, child Widget) Widget) MediaQueryData {
	t.Helper()
	tr := newTree(t)
	var got MediaQueryData
	inner := builderOf(func(bc BuildContext) Widget { got = MediaQueryOf(bc); return newLeaf() })
	tr.pump(NewMediaQuery(data, builderOf(func(bc BuildContext) Widget { return wrap(bc, inner) })))
	return got
}

func TestMediaQueryDoesNotHaveADefault(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.collectPanics = true
	var maybe bool
	tr.pump(builderOf(func(bc BuildContext) Widget {
		_, maybe = MediaQueryMaybeOf(bc)
		MediaQueryOf(bc)
		return newLeaf()
	}))
	if maybe {
		t.Error("MediaQueryMaybeOf found a MediaQuery in a tree that has none")
	}
	appErr := requireErrorTag(t, tr.errs[0], "MediaQueryMissing")
	if !strings.Contains(appErr.Message, "MediaQuery") {
		t.Errorf("message = %q", appErr.Message)
	}
}

func TestMediaQueryOfFindsTheDataWhenThereIsOne(t *testing.T) {
	data := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 100, Height: 200}})
	tr := newTree(t)
	var of, maybe MediaQueryData
	var ok bool
	tr.pump(NewMediaQuery(data, builderOf(func(bc BuildContext) Widget {
		of = MediaQueryOf(bc)
		maybe, ok = MediaQueryMaybeOf(bc)
		return newLeaf()
	})))
	if !of.Equal(data) || !ok || !maybe.Equal(data) {
		t.Errorf("of = %v, maybe = %v (%v), want %v", of, maybe, ok, data)
	}
}

func TestMediaQueryDataDefaults(t *testing.T) {
	d := NewMediaQueryData()
	if d.DevicePixelRatio != 1 || d.PlatformBrightness != ui.BrightnessLight || d.NavigationMode != NavigationModeTraditional ||
		!painting.TextScalerEqual(d.TextScaler, painting.TextScalerNoScaling) {
		t.Errorf("defaults = %v", d)
	}
	if slop, _ := d.GestureSettings.TouchSlop.Get(); slop != gestures.TouchSlop {
		t.Errorf("touch slop = %v, want kTouchSlop", slop)
	}
	for _, tc := range []struct {
		size ui.Size
		want Orientation
	}{
		{ui.Size{Width: 100, Height: 200}, OrientationPortrait},
		{ui.Size{Width: 200, Height: 100}, OrientationLandscape},
		{ui.Size{Width: 100, Height: 100}, OrientationPortrait},
	} {
		if got := NewMediaQueryData(MediaQueryDataOpts{Size: tc.size}).Orientation(); got != tc.want {
			t.Errorf("orientation of %v = %v, want %v", tc.size, got, tc.want)
		}
	}
}

func TestMediaQueryDataFromViewIsSane(t *testing.T) {
	bt := newBindingTree(t, bindingTreeOpts{size: ui.Size{Width: 1600, Height: 1200}, ratio: 2})
	data := NewMediaQueryDataFromView(bt.view, nil)
	if data.Size != (ui.Size{Width: 800, Height: 600}) || data.DevicePixelRatio != 2 {
		t.Errorf("size = %v, ratio = %v; want 800x600 at 2", data.Size, data.DevicePixelRatio)
	}
	if data.AccessibleNavigation || data.InvertColors || data.DisableAnimations || data.BoldText || data.HighContrast || data.OnOffSwitchLabels {
		t.Errorf("accessibility features are on: %v", data)
	}
	if data.PlatformBrightness != ui.BrightnessLight || !data.GestureSettings.TouchSlop.IsNone() {
		t.Errorf("brightness = %v, touch slop = %v", data.PlatformBrightness, data.GestureSettings)
	}
	if !data.Equal(data.CopyWith(MediaQueryDataCopyOpts{})) {
		t.Error("CopyWith with no options changed the data")
	}
	if s := data.String(); strings.Contains(s, "\n") || !strings.HasPrefix(s, "MediaQueryData(size: ") {
		t.Errorf("String = %q, want a one line description", s)
	}
}

func TestMediaQueryDataFromViewUsesThePlatformData(t *testing.T) {
	bt := newBindingTree(t, bindingTreeOpts{ratio: 2})
	bt.setView(func(c *ui.ViewConfiguration) {
		c.Padding = ui.ViewPadding{Left: 2, Top: 4, Right: 6, Bottom: 8}
		c.ViewPadding = ui.ViewPadding{Left: 4, Top: 8, Right: 12, Bottom: 16}
		c.ViewInsets = ui.ViewPadding{Left: 2, Top: 4, Right: 6, Bottom: 8}
		c.SystemGestureInsets = ui.ViewPadding{Left: 10, Top: 10, Right: 10, Bottom: 10}
		c.GestureSettings = ui.GestureSettings{PhysicalTouchSlop: 20}
	})
	platform := NewMediaQueryData(MediaQueryDataOpts{
		TextScaler: painting.NewTextScalerLinear(1234), PlatformBrightness: ui.BrightnessDark, AccessibleNavigation: true,
		InvertColors: true, DisableAnimations: true, BoldText: true, HighContrast: true, OnOffSwitchLabels: true,
		AlwaysUse24HourFormat: true, NavigationMode: NavigationModeDirectional, SupportsShowingSystemContextMenu: true,
	})
	data := NewMediaQueryDataFromView(bt.view, &platform)
	if data.Size != (ui.Size{Width: 400, Height: 300}) || data.DevicePixelRatio != 2 {
		t.Errorf("size = %v, ratio = %v; want 400x300 at 2", data.Size, data.DevicePixelRatio)
	}
	tests := []struct {
		name      string
		got, want any
	}{
		{"padding", data.Padding, painting.NewEdgeInsetsFromViewPadding(bt.view.Padding(), 2)},
		{"viewPadding", data.ViewPadding, painting.NewEdgeInsetsFromViewPadding(bt.view.ViewPadding(), 2)},
		{"viewInsets", data.ViewInsets, painting.NewEdgeInsetsFromViewPadding(bt.view.ViewInsets(), 2)},
		{"systemGestureInsets", data.SystemGestureInsets, painting.NewEdgeInsetsFromViewPadding(bt.view.SystemGestureInsets(), 2)},
		{"brightness", data.PlatformBrightness, platform.PlatformBrightness},
		{"accessibleNavigation", data.AccessibleNavigation, true},
		{"invertColors", data.InvertColors, true},
		{"disableAnimations", data.DisableAnimations, true},
		{"boldText", data.BoldText, true},
		{"highContrast", data.HighContrast, true},
		{"onOffSwitchLabels", data.OnOffSwitchLabels, true},
		{"alwaysUse24HourFormat", data.AlwaysUse24HourFormat, true},
		{"navigationMode", data.NavigationMode, NavigationModeDirectional},
		{"supportsShowingSystemContextMenu", data.SupportsShowingSystemContextMenu, true},
		{"gestureSettings", data.GestureSettings, gestures.NewDeviceGestureSettingsFromView(bt.view)},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if !painting.TextScalerEqual(data.TextScaler, platform.TextScaler) {
		t.Errorf("text scaler = %v, want %v", data.TextScaler, platform.TextScaler)
	}
	if slop, _ := data.GestureSettings.TouchSlop.Get(); slop != 10 {
		t.Errorf("touch slop = %v, want 20 physical pixels = 10 logical", slop)
	}
}

func TestMediaQueryDataFromViewUsesTheDispatcherWithoutPlatformData(t *testing.T) {
	bt := newBindingTree(t)
	bt.setPlatform(func(c *ui.PlatformConfiguration) {
		c.TextScaleFactor = 123
		c.PlatformBrightness = ui.BrightnessDark
		c.AccessibilityFeatures = ui.AccessibilityFeatures{
			AccessibleNavigation: true, InvertColors: true, DisableAnimations: true, BoldText: true, HighContrast: true, OnOffSwitchLabels: true,
		}
	})
	data := NewMediaQueryDataFromView(bt.view, nil)
	if !painting.TextScalerEqual(data.TextScaler, painting.NewTextScalerLinear(123)) || data.TextScaler.TextScaleFactor() != 123 {
		t.Errorf("text scaler = %v, want 123", data.TextScaler)
	}
	if data.PlatformBrightness != ui.BrightnessDark || !data.AccessibleNavigation || !data.InvertColors || !data.DisableAnimations ||
		!data.BoldText || !data.HighContrast || !data.OnOffSwitchLabels {
		t.Errorf("the settings of the dispatcher were not used: %v", data)
	}
}

func TestMediaQueryCopyWith(t *testing.T) {
	base := fullData()
	changed := base.CopyWith(MediaQueryDataCopyOpts{
		Size:               foundation.Some(ui.Size{Width: 1, Height: 2}),
		DevicePixelRatio:   foundation.Some(3.0),
		TextScaler:         foundation.Some(painting.NewTextScalerLinear(2)),
		PlatformBrightness: foundation.Some(ui.BrightnessDark), Padding: foundation.Some(painting.EdgeInsetsZero),
		ViewPadding: foundation.Some(painting.EdgeInsetsZero), ViewInsets: foundation.Some(painting.EdgeInsetsZero),
		SystemGestureInsets: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1)), AlwaysUse24HourFormat: foundation.Some(false),
		HighContrast: foundation.Some(false), OnOffSwitchLabels: foundation.Some(false), DisableAnimations: foundation.Some(false),
		InvertColors: foundation.Some(false), AccessibleNavigation: foundation.Some(false), BoldText: foundation.Some(false),
		NavigationMode:  foundation.Some(NavigationModeTraditional),
		GestureSettings: foundation.Some(gestures.NewDeviceGestureSettings(foundation.Some(7.0))),
	})
	want := NewMediaQueryData(MediaQueryDataOpts{
		Size: ui.Size{Width: 1, Height: 2}, DevicePixelRatio: 3, TextScaler: painting.NewTextScalerLinear(2),
		PlatformBrightness: ui.BrightnessDark, SystemGestureInsets: painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1),
		GestureSettings: foundation.Some(gestures.NewDeviceGestureSettings(foundation.Some(7.0))),
	})
	if !changed.Equal(want) {
		t.Errorf("CopyWith = %v\nwant     %v", changed, want)
	}
	if !base.Equal(fullData()) {
		t.Error("CopyWith changed the original")
	}
}

func TestMediaQueryEqual(t *testing.T) {
	a := fullData()
	if !a.Equal(fullData()) {
		t.Error("equal data are not equal")
	}
	for name, change := range map[string]MediaQueryDataCopyOpts{
		"size":    {Size: foundation.Some(ui.Size{Width: 9, Height: 9})},
		"scaler":  {TextScaler: foundation.Some(painting.NewTextScalerLinear(3))},
		"padding": {Padding: foundation.Some(painting.EdgeInsetsZero)},
		"nav":     {NavigationMode: foundation.Some(NavigationModeTraditional)},
		"bold":    {BoldText: foundation.Some(false)},
	} {
		if a.Equal(a.CopyWith(change)) {
			t.Errorf("data that differ in %s are equal", name)
		}
	}
}

func TestMediaQueryRemoveSides(t *testing.T) {
	all := RemoveSides{Left: true, Top: true, Right: true, Bottom: true}
	base := fullData()
	pad, vp, vi := base.Padding, base.ViewPadding, base.ViewInsets
	zero := painting.EdgeInsetsZero
	tests := []struct {
		name        string
		remove      func(bc BuildContext, child Widget) Widget
		wantPadding painting.EdgeInsets
		wantVP      painting.EdgeInsets
		wantInsets  painting.EdgeInsets
	}{
		{"removePadding all", func(bc BuildContext, c Widget) Widget { return NewMediaQueryRemovePadding(bc, all, c) },
			zero, vi, vi},
		{"removePadding top only", func(bc BuildContext, c Widget) Widget {
			return NewMediaQueryRemovePadding(bc, RemoveSides{Top: true}, c)
		}, withTop(pad, 0), withTop(vp, vi.Top), vi},
		{"removeViewInsets all", func(bc BuildContext, c Widget) Widget { return NewMediaQueryRemoveViewInsets(bc, all, c) },
			pad, painting.NewEdgeInsetsFromLTRB(3, 1, 2, 4), zero},
		{"removeViewInsets left only", func(bc BuildContext, c Widget) Widget {
			return NewMediaQueryRemoveViewInsets(bc, RemoveSides{Left: true}, c)
		}, pad, painting.NewEdgeInsetsFromLTRB(3, vp.Top, vp.Right, vp.Bottom), painting.NewEdgeInsetsFromLTRB(0, vi.Top, vi.Right, vi.Bottom)},
		{"removeViewPadding all", func(bc BuildContext, c Widget) Widget { return NewMediaQueryRemoveViewPadding(bc, all, c) },
			zero, zero, vi},
		{"removeViewPadding bottom only", func(bc BuildContext, c Widget) Widget {
			return NewMediaQueryRemoveViewPadding(bc, RemoveSides{Bottom: true}, c)
		}, painting.NewEdgeInsetsFromLTRB(pad.Left, pad.Top, pad.Right, 0), painting.NewEdgeInsetsFromLTRB(vp.Left, vp.Top, vp.Right, 0), vi},
		{"nothing removed", func(bc BuildContext, c Widget) Widget { return NewMediaQueryRemovePadding(bc, RemoveSides{}, c) },
			pad, vp, vi},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dataSeenBelow(t, base, tc.remove)
			if got.Padding != tc.wantPadding || got.ViewPadding != tc.wantVP || got.ViewInsets != tc.wantInsets {
				t.Errorf("padding = %v, viewPadding = %v, viewInsets = %v\nwant      %v, %v, %v",
					got.Padding, got.ViewPadding, got.ViewInsets, tc.wantPadding, tc.wantVP, tc.wantInsets)
			}
			rest := got.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(base.Padding), ViewPadding: foundation.Some(base.ViewPadding),
				ViewInsets: foundation.Some(base.ViewInsets)})
			if !rest.Equal(base) {
				t.Errorf("the other fields changed: %v", got)
			}
		})
	}
}

func withTop(e painting.EdgeInsets, top float64) painting.EdgeInsets {
	e.Top = top
	return e
}

func TestMediaQueryTextScalingWrappers(t *testing.T) {
	base := NewMediaQueryData(MediaQueryDataOpts{TextScaler: painting.NewTextScalerLinear(3)})
	tests := []struct {
		name string
		wrap func(child Widget) Widget
		want float64
	}{
		{"no scaling", func(c Widget) Widget { return NewMediaQueryWithNoTextScaling(c) }, 1},
		{"clamped to a maximum", func(c Widget) Widget {
			return NewMediaQueryWithClampedTextScaling(c, MediaQueryClampedTextScalingOpts{MaxScaleFactor: 2})
		}, 2},
		{"clamped to a minimum", func(c Widget) Widget {
			return NewMediaQueryWithClampedTextScaling(c, MediaQueryClampedTextScalingOpts{MinScaleFactor: 4})
		}, 4},
		{"clamp that does not apply", func(c Widget) Widget {
			return NewMediaQueryWithClampedTextScaling(c, MediaQueryClampedTextScalingOpts{MinScaleFactor: 1, MaxScaleFactor: 5})
		}, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dataSeenBelow(t, base, func(_ BuildContext, child Widget) Widget { return tc.wrap(child) })
			if f := got.TextScaler.TextScaleFactor(); f != tc.want {
				t.Errorf("text scale factor = %v, want %v", f, tc.want)
			}
		})
	}
}

// TestMediaQueryAccessorsDefaults checks the defaults of the accessors that do
// not panic without a MediaQuery, and that every Maybe accessor says so.
func TestMediaQueryAccessorsDefaults(t *testing.T) {
	tr := newTree(t)
	tr.pump(builderOf(func(bc BuildContext) Widget {
		if !painting.TextScalerEqual(MediaQueryTextScalerOf(bc), painting.TextScalerNoScaling) {
			t.Error("TextScalerOf without a MediaQuery should be no scaling")
		}
		if MediaQueryPlatformBrightnessOf(bc) != ui.BrightnessLight || MediaQueryHighContrastOf(bc) || MediaQueryOnOffSwitchLabelsOf(bc) || MediaQueryBoldTextOf(bc) {
			t.Error("the defaults without a MediaQuery are light, and false")
		}
		maybes := []struct {
			name string
			ok   bool
		}{
			{"size", second(MediaQueryMaybeSizeOf(bc))},
			{"width", second(MediaQueryMaybeWidthOf(bc))},
			{"height", second(MediaQueryMaybeHeightOf(bc))},
			{"orientation", second(MediaQueryMaybeOrientationOf(bc))},
			{"devicePixelRatio", second(MediaQueryMaybeDevicePixelRatioOf(bc))},
			{"textScaler", second(MediaQueryMaybeTextScalerOf(bc))},
			{"platformBrightness", second(MediaQueryMaybePlatformBrightnessOf(bc))},
			{"padding", second(MediaQueryMaybePaddingOf(bc))},
			{"viewInsets", second(MediaQueryMaybeViewInsetsOf(bc))},
			{"systemGestureInsets", second(MediaQueryMaybeSystemGestureInsetsOf(bc))},
			{"viewPadding", second(MediaQueryMaybeViewPaddingOf(bc))},
			{"alwaysUse24HourFormat", second2(MediaQueryMaybeAlwaysUse24HourFormatOf(bc))},
			{"accessibleNavigation", second2(MediaQueryMaybeAccessibleNavigationOf(bc))},
			{"invertColors", second2(MediaQueryMaybeInvertColorsOf(bc))},
			{"highContrast", second2(MediaQueryMaybeHighContrastOf(bc))},
			{"onOffSwitchLabels", second2(MediaQueryMaybeOnOffSwitchLabelsOf(bc))},
			{"disableAnimations", second2(MediaQueryMaybeDisableAnimationsOf(bc))},
			{"boldText", second2(MediaQueryMaybeBoldTextOf(bc))},
			{"navigationMode", second(MediaQueryMaybeNavigationModeOf(bc))},
			{"gestureSettings", second(MediaQueryMaybeGestureSettingsOf(bc))},
			{"systemContextMenu", second2(MediaQueryMaybeSupportsShowingSystemContextMenu(bc))},
		}
		for _, m := range maybes {
			if m.ok {
				t.Errorf("MediaQueryMaybe%sOf found a MediaQuery in a tree that has none", m.name)
			}
		}
		return newLeaf()
	}))
}

func second[T any](_ T, ok bool) bool { return ok }
func second2(_, ok bool) bool         { return ok }

// TestMediaQueryAccessors checks that each accessor reads its field.
func TestMediaQueryAccessors(t *testing.T) {
	data := fullData().CopyWith(MediaQueryDataCopyOpts{
		SystemGestureInsets: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 2, 3, 4)),
		GestureSettings:     foundation.Some(gestures.NewDeviceGestureSettings(foundation.Some(5.0))),
		PlatformBrightness:  foundation.Some(ui.BrightnessDark),
	})
	tr := newTree(t)
	tr.pump(NewMediaQuery(data, builderOf(func(bc BuildContext) Widget {
		checks := []struct {
			name      string
			got, want any
		}{
			{"size", MediaQuerySizeOf(bc), data.Size},
			{"width", MediaQueryWidthOf(bc), data.Size.Width},
			{"height", MediaQueryHeightOf(bc), data.Size.Height},
			{"orientation", MediaQueryOrientationOf(bc), OrientationPortrait},
			{"devicePixelRatio", MediaQueryDevicePixelRatioOf(bc), 2.0},
			{"textScaleFactor", MediaQueryTextScalerOf(bc).TextScaleFactor(), 1.2},
			{"platformBrightness", MediaQueryPlatformBrightnessOf(bc), ui.BrightnessDark},
			{"padding", MediaQueryPaddingOf(bc), data.Padding},
			{"viewInsets", MediaQueryViewInsetsOf(bc), data.ViewInsets},
			{"systemGestureInsets", MediaQuerySystemGestureInsetsOf(bc), data.SystemGestureInsets},
			{"viewPadding", MediaQueryViewPaddingOf(bc), data.ViewPadding},
			{"alwaysUse24HourFormat", MediaQueryAlwaysUse24HourFormatOf(bc), true},
			{"accessibleNavigation", MediaQueryAccessibleNavigationOf(bc), true},
			{"invertColors", MediaQueryInvertColorsOf(bc), true},
			{"highContrast", MediaQueryHighContrastOf(bc), true},
			{"onOffSwitchLabels", MediaQueryOnOffSwitchLabelsOf(bc), true},
			{"disableAnimations", MediaQueryDisableAnimationsOf(bc), true},
			{"boldText", MediaQueryBoldTextOf(bc), true},
			{"navigationMode", MediaQueryNavigationModeOf(bc), NavigationModeDirectional},
			{"gestureSettings", MediaQueryGestureSettingsOf(bc), data.GestureSettings},
			{"systemContextMenu", MediaQuerySupportsShowingSystemContextMenu(bc), false},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
			}
		}
		return newLeaf()
	})))
}

// aspectCase is a variant of 'MediaQuery partial dependencies': a way to read
// one aspect, and a change of the data that touches only that aspect.
type aspectCase struct {
	name   string
	read   func(bc BuildContext)
	change MediaQueryDataCopyOpts
	// also are the other readers that the change touches, from the size of the
	// base data of 800x600 (landscape).
	also []string
}

var aspectCases = []aspectCase{
	{"size", func(bc BuildContext) { MediaQuerySizeOf(bc) }, MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 5, Height: 6})}, []string{"width", "height", "orientation"}},
	{"width", func(bc BuildContext) { MediaQueryWidthOf(bc) }, MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 5, Height: 600})}, []string{"size", "orientation"}},
	{"height", func(bc BuildContext) { MediaQueryHeightOf(bc) }, MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 800, Height: 6})}, []string{"size"}},
	{"devicePixelRatio", func(bc BuildContext) { MediaQueryDevicePixelRatioOf(bc) }, MediaQueryDataCopyOpts{DevicePixelRatio: foundation.Some(2.0)}, nil},
	{"textScaler", func(bc BuildContext) { MediaQueryTextScalerOf(bc) }, MediaQueryDataCopyOpts{TextScaler: foundation.Some(painting.NewTextScalerLinear(2))}, nil},
	{"platformBrightness", func(bc BuildContext) { MediaQueryPlatformBrightnessOf(bc) }, MediaQueryDataCopyOpts{PlatformBrightness: foundation.Some(ui.BrightnessDark)}, nil},
	{"padding", func(bc BuildContext) { MediaQueryPaddingOf(bc) }, MediaQueryDataCopyOpts{Padding: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1))}, nil},
	{"viewInsets", func(bc BuildContext) { MediaQueryViewInsetsOf(bc) }, MediaQueryDataCopyOpts{ViewInsets: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1))}, nil},
	{"systemGestureInsets", func(bc BuildContext) { MediaQuerySystemGestureInsetsOf(bc) }, MediaQueryDataCopyOpts{SystemGestureInsets: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1))}, nil},
	{"viewPadding", func(bc BuildContext) { MediaQueryViewPaddingOf(bc) }, MediaQueryDataCopyOpts{ViewPadding: foundation.Some(painting.NewEdgeInsetsFromLTRB(1, 1, 1, 1))}, nil},
	{"alwaysUse24HourFormat", func(bc BuildContext) { MediaQueryAlwaysUse24HourFormatOf(bc) }, MediaQueryDataCopyOpts{AlwaysUse24HourFormat: foundation.Some(true)}, nil},
	{"accessibleNavigation", func(bc BuildContext) { MediaQueryAccessibleNavigationOf(bc) }, MediaQueryDataCopyOpts{AccessibleNavigation: foundation.Some(true)}, nil},
	{"invertColors", func(bc BuildContext) { MediaQueryInvertColorsOf(bc) }, MediaQueryDataCopyOpts{InvertColors: foundation.Some(true)}, nil},
	{"highContrast", func(bc BuildContext) { MediaQueryHighContrastOf(bc) }, MediaQueryDataCopyOpts{HighContrast: foundation.Some(true)}, nil},
	{"onOffSwitchLabels", func(bc BuildContext) { MediaQueryOnOffSwitchLabelsOf(bc) }, MediaQueryDataCopyOpts{OnOffSwitchLabels: foundation.Some(true)}, nil},
	{"disableAnimations", func(bc BuildContext) { MediaQueryDisableAnimationsOf(bc) }, MediaQueryDataCopyOpts{DisableAnimations: foundation.Some(true)}, nil},
	{"boldText", func(bc BuildContext) { MediaQueryBoldTextOf(bc) }, MediaQueryDataCopyOpts{BoldText: foundation.Some(true)}, nil},
	{"navigationMode", func(bc BuildContext) { MediaQueryNavigationModeOf(bc) }, MediaQueryDataCopyOpts{NavigationMode: foundation.Some(NavigationModeDirectional)}, nil},
	{"gestureSettings", func(bc BuildContext) { MediaQueryGestureSettingsOf(bc) }, MediaQueryDataCopyOpts{GestureSettings: foundation.Some(gestures.NewDeviceGestureSettings(foundation.Some(1.0)))}, nil},
	{"systemContextMenu", func(bc BuildContext) { MediaQuerySupportsShowingSystemContextMenu(bc) }, MediaQueryDataCopyOpts{SupportsShowingSystemContextMenu: foundation.Some(true)}, nil},
	{"orientation", func(bc BuildContext) { MediaQueryOrientationOf(bc) }, MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 100, Height: 200})}, []string{"size", "width", "height"}},
}

// TestMediaQueryPartialDependencies is 'MediaQuery partial dependencies': a
// widget that reads one aspect is rebuilt when that aspect changes, and when
// another aspect changes only if it is the same one.
func TestMediaQueryPartialDependencies(t *testing.T) {
	base := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}})
	for _, reader := range aspectCases {
		for _, change := range aspectCases {
			affects := reader.name == change.name || slices.Contains(change.also, reader.name)
			t.Run(reader.name+" reader, "+change.name+" change", func(t *testing.T) {
				tr := newTree(t)
				builds := 0
				child := builderOf(func(bc BuildContext) Widget { builds++; reader.read(bc); return newLeaf() })
				tr.pump(NewMediaQuery(base, child))
				if builds != 1 {
					t.Fatalf("built %d times, want 1", builds)
				}
				tr.pump(NewMediaQuery(base.CopyWith(change.change), child))
				want := 1
				if affects {
					want = 2
				}
				if builds != want {
					t.Errorf("built %d times after the change, want %d", builds, want)
				}
			})
		}
	}
}

// TestMediaQueryWidthAndHeightAreListenedToIndependently is 'MediaQuery width
// and height can be listened to independently'.
func TestMediaQueryWidthAndHeightAreListenedToIndependently(t *testing.T) {
	data := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}})
	tr := newTree(t)
	widthBuilds, heightBuilds := 0, 0
	var width, height float64
	column := newMulti(
		builderOf(func(bc BuildContext) Widget { widthBuilds++; width = MediaQueryWidthOf(bc); return newLeaf() }),
		builderOf(func(bc BuildContext) Widget { heightBuilds++; height = MediaQueryHeightOf(bc); return newLeaf() }),
	)
	tr.pump(NewMediaQuery(data, column))
	data = data.CopyWith(MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 900, Height: 600})})
	tr.pump(NewMediaQuery(data, column))
	if width != 900 || widthBuilds != 2 || heightBuilds != 1 {
		t.Errorf("after a width change: width = %v, builds = %d/%d; want 900, 2/1", width, widthBuilds, heightBuilds)
	}
	data = data.CopyWith(MediaQueryDataCopyOpts{Size: foundation.Some(ui.Size{Width: 900, Height: 700})})
	tr.pump(NewMediaQuery(data, column))
	if height != 700 || widthBuilds != 2 || heightBuilds != 2 {
		t.Errorf("after a height change: height = %v, builds = %d/%d; want 700, 2/2", height, widthBuilds, heightBuilds)
	}
}

// TestMediaQueryFromViewCreatesAMediaQuery is 'MediaQuery.fromView creates a
// MediaQuery', and 'creates a SystemTextScaler'.
func TestMediaQueryFromViewCreatesAMediaQuery(t *testing.T) {
	bt := newBindingTree(t, bindingTreeOpts{ratio: 2})
	bt.setPlatform(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 2 })
	var got MediaQueryData
	var found bool
	bt.attachUnwrapped(NewMediaQueryFromView(bt.view, builderOf(func(bc BuildContext) Widget {
		got, found = MediaQueryMaybeOf(bc)
		return newLeaf()
	})))
	bt.frame()
	if !found || got.Size != (ui.Size{Width: 400, Height: 300}) || got.DevicePixelRatio != 2 {
		t.Errorf("found = %v, data = %v; want 400x300 at 2", found, got)
	}
	if got.TextScaler.TextScaleFactor() != 2 {
		t.Errorf("text scale factor = %v, want the system's 2", got.TextScaler.TextScaleFactor())
	}
}

// TestMediaQueryFromViewUpdatesOnNotifications is 'MediaQuery.fromView updates
// on notifications (no parent data)' and '(with parent data)'.
func TestMediaQueryFromViewUpdatesOnNotifications(t *testing.T) {
	tests := []struct {
		name string
		// parent is the data of a MediaQuery above, or nil for none.
		parent *MediaQueryData
		// touch changes the platform; platformWins is whether the data below
		// follows the change (it does only without parent data).
		touch        func(bt *bindingTree)
		read         func(d MediaQueryData) any
		platformWins any
	}{
		{"metrics", nil, func(bt *bindingTree) {
			bt.setView(func(c *ui.ViewConfiguration) { c.PhysicalSize = ui.Size{Width: 200, Height: 100} })
		}, func(d MediaQueryData) any { return d.Size }, ui.Size{Width: 200, Height: 100}},
		{"text scale", nil, func(bt *bindingTree) {
			bt.setPlatform(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 3 })
		}, func(d MediaQueryData) any { return d.TextScaler.TextScaleFactor() }, 3.0},
		{"brightness", nil, func(bt *bindingTree) {
			bt.setPlatform(func(c *ui.PlatformConfiguration) { c.PlatformBrightness = ui.BrightnessDark })
		}, func(d MediaQueryData) any { return d.PlatformBrightness }, ui.BrightnessDark},
		{"accessibility", nil, func(bt *bindingTree) {
			bt.setPlatform(func(c *ui.PlatformConfiguration) { c.AccessibilityFeatures.BoldText = true })
		}, func(d MediaQueryData) any { return d.BoldText }, true},
	}
	for _, tc := range tests {
		for _, withParent := range []bool{false, true} {
			name := tc.name + " without parent data"
			if withParent {
				name = tc.name + " with parent data"
			}
			t.Run(name, func(t *testing.T) {
				bt := newBindingTree(t, bindingTreeOpts{})
				parent := NewMediaQueryData(MediaQueryDataOpts{
					TextScaler: painting.NewTextScalerLinear(1.5), PlatformBrightness: ui.BrightnessLight, BoldText: false,
				})
				var seen []MediaQueryData
				child := builderOf(func(bc BuildContext) Widget { seen = append(seen, MediaQueryOf(bc)); return newLeaf() })
				root := NewMediaQueryFromView(bt.view, child)
				if withParent {
					root = NewMediaQuery(parent, root)
				}
				bt.attachUnwrapped(root)
				bt.frame()
				before := tc.read(seen[len(seen)-1])
				tc.touch(bt)
				bt.frame()
				after := tc.read(seen[len(seen)-1])
				followsPlatform := !withParent || tc.name == "metrics"
				switch {
				case followsPlatform && after != tc.platformWins:
					t.Errorf("after the change the data has %v (before %v), want %v", after, before, tc.platformWins)
				case !followsPlatform && after != before:
					t.Errorf("with parent data the platform setting leaked: %v became %v", before, after)
				}
			})
		}
	}
}

// TestMediaQueryFromViewUpdatesWhenTheParentDataChanges is 'MediaQuery.fromView
// updates when parent data changes'.
func TestMediaQueryFromViewUpdatesWhenTheParentDataChanges(t *testing.T) {
	bt := newBindingTree(t)
	var scalers []float64
	child := builderOf(func(bc BuildContext) Widget {
		scalers = append(scalers, MediaQueryOf(bc).TextScaler.TextScaleFactor())
		return newLeaf()
	})
	mq := NewMediaQueryFromView(bt.view, child)
	set := func(f float64) {
		bt.b.AttachRootWidget(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{TextScaler: painting.NewTextScalerLinear(f)}), mq))
		bt.frame()
	}
	set(1.5)
	set(2.5)
	if len(scalers) == 0 || scalers[len(scalers)-1] != 2.5 {
		t.Errorf("scalers = %v, want the last to be 2.5", scalers)
	}
}

// TestMediaQueryFromViewBrightnessOverride checks debugBrightnessOverride.
func TestMediaQueryFromViewBrightnessOverride(t *testing.T) {
	skipUnlessDebug(t)
	debug := NewDebugOptions()
	debug.BrightnessOverride = ui.BrightnessDark
	bt := newBindingTree(t, bindingTreeOpts{debug: debug})
	var got ui.Brightness
	bt.attachUnwrapped(NewMediaQueryFromView(bt.view, builderOf(func(bc BuildContext) Widget {
		got = MediaQueryOf(bc).PlatformBrightness
		return newLeaf()
	})))
	bt.frame()
	if got != ui.BrightnessDark {
		t.Errorf("brightness = %v, want the override", got)
	}
}

// TestMediaQueryFromViewStopsListeningWhenRemoved checks that the state removes
// its observer in Dispose.
func TestMediaQueryFromViewStopsListeningWhenRemoved(t *testing.T) {
	bt := newBindingTree(t)
	bt.attachUnwrapped(NewMediaQueryFromView(bt.view, newLeaf()))
	bt.frame()
	before := len(bt.b.observers)
	if before != 1 {
		t.Fatalf("%d observers, want 1", before)
	}
	bt.b.AttachRootWidget(newLeaf())
	bt.frame()
	if n := len(bt.b.observers); n != 0 {
		t.Errorf("%d observers after the MediaQuery left the tree, want 0", n)
	}
}

func TestMediaQueryDataStringsAndEnums(t *testing.T) {
	d := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}})
	want := fmt.Sprintf("MediaQueryData(size: %s, devicePixelRatio: 1.0, textScaler: %v, platformBrightness: light", d.Size, d.TextScaler)
	if !strings.HasPrefix(d.String(), want) {
		t.Errorf("String = %q, want the prefix %q", d.String(), want)
	}
	if OrientationPortrait.String() != "portrait" || OrientationLandscape.String() != "landscape" ||
		NavigationModeDirectional.String() != "directional" || NavigationModeUnspecified.String() != "traditional" {
		t.Error("the enum names are not Flutter's")
	}
}

func TestMediaQueryClampedTextScalingChecksItsLimits(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts MediaQueryClampedTextScalingOpts
	}{
		{"the maximum is below the minimum", MediaQueryClampedTextScalingOpts{MinScaleFactor: 3, MaxScaleFactor: 2}},
		{"a negative minimum", MediaQueryClampedTextScalingOpts{MinScaleFactor: -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requireTag(t, "MediaQueryClampInvalid", func() { NewMediaQueryWithClampedTextScaling(newLeaf(), tc.opts) })
		})
	}
}
