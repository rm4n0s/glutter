// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/rendering/view_chrome_style_test.dart: the RenderView takes
// the overlay style of the annotated regions at the centers of the status bar and the navigation
// bar, and passes it to SystemChrome.
func TestSystemChromeStyleOfTheView(t *testing.T) {
	const (
		statusBarHeight     = 25.0
		navigationBarHeight = 54.0
		deviceHeight        = 960.0
		deviceWidth         = 480.0
		devicePixelRatio    = 2.0
	)
	blue := ui.NewColor(0xFF2196F3)
	green := ui.NewColor(0xFF4CAF50)
	mobile := []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia}
	android := []foundation.TargetPlatform{foundation.TargetPlatformAndroid}
	box := func(w, h float64) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)})
	}
	region := func(s services.SystemUiOverlayStyle, child Widget) Widget { return NewAnnotatedRegion(s, child) }
	style := func(nav, status foundation.Opt[ui.Color]) services.SystemUiOverlayStyle {
		return services.SystemUiOverlayStyle{SystemNavigationBarColor: nav, StatusBarColor: status}
	}
	someColor := foundation.Some[ui.Color]
	none := foundation.None[ui.Color]()

	tests := []struct {
		name      string
		platforms []foundation.TargetPlatform
		device    bool // sets the padding and the size of the view
		build     func() Widget
		// wantStatus and wantNav are the colors of the latest style; none means unset.
		wantStatus, wantNav foundation.Opt[ui.Color]
		// wantSent is false when no style may reach SystemChrome.
		wantSent bool
	}{
		{
			name: "statusBarColor isn't set for unannotated view", platforms: mobile,
			build: func() Widget { return NewSizedBoxExpand(nil) },
		},
		{
			name: "statusBarColor is set for annotated view", platforms: mobile, device: true,
			build: func() Widget {
				return region(style(none, someColor(blue)), NewSizedBoxExpand(nil))
			},
			wantSent: true, wantStatus: someColor(blue),
		},
		{
			name: "statusBarColor isn't set when view covers less than half of the system status bar", platforms: mobile, device: true,
			build: func() Widget {
				return NewAlign(region(style(none, someColor(blue)), box(100, statusBarHeight/2.0-1)), AlignOpts{Alignment: painting.AlignmentTopCenter})
			},
		},
		{
			name: "statusBarColor is set when view covers more than half of the system status bar", platforms: mobile, device: true,
			build: func() Widget {
				return NewAlign(region(style(none, someColor(blue)), box(100, statusBarHeight/2.0+1)), AlignOpts{Alignment: painting.AlignmentTopCenter})
			},
			wantSent: true, wantStatus: someColor(blue),
		},
		{
			name: "systemNavigationBarColor isn't set for non Android device", platforms: []foundation.TargetPlatform{foundation.TargetPlatformIOS}, device: true,
			build: func() Widget {
				return region(style(someColor(blue), none), NewSizedBoxExpand(nil))
			},
			wantSent: true,
		},
		{
			name: "systemNavigationBarColor isn't set for unannotated view", platforms: android,
			build: func() Widget { return NewSizedBoxExpand(nil) },
		},
		{
			name: "systemNavigationBarColor is set for annotated view", platforms: android, device: true,
			build: func() Widget {
				return region(style(someColor(blue), none), NewSizedBoxExpand(nil))
			},
			wantSent: true, wantNav: someColor(blue),
		},
		{
			name: "systemNavigationBarColor isn't set when view covers less than half of navigation bar", platforms: android, device: true,
			build: func() Widget {
				return NewAlign(region(style(someColor(blue), none), box(100, navigationBarHeight/2.0-1)), AlignOpts{Alignment: painting.AlignmentBottomCenter})
			},
		},
		{
			name: "systemNavigationBarColor is set when view covers more than half of navigation bar", platforms: android, device: true,
			build: func() Widget {
				return NewAlign(region(style(someColor(blue), none), box(100, navigationBarHeight/2.0+1)), AlignOpts{Alignment: painting.AlignmentBottomCenter})
			},
			wantSent: true, wantNav: someColor(blue),
		},
		{
			name:      "Top AnnotatedRegion provides status bar overlay style and bottom AnnotatedRegion provides navigation bar overlay style",
			platforms: android, device: true,
			build: func() Widget {
				return NewColumn([]Widget{
					NewExpanded(region(style(someColor(blue), someColor(blue)), NewSizedBoxExpand(nil))),
					NewExpanded(region(style(someColor(green), someColor(green)), NewSizedBoxExpand(nil))),
				})
			},
			wantSent: true, wantStatus: someColor(blue), wantNav: someColor(green),
		},
		{
			name:      "Top only AnnotatedRegion provides status bar and navigation bar style properties",
			platforms: android, device: true,
			build: func() Widget {
				return NewColumn([]Widget{
					NewExpanded(region(style(someColor(blue), someColor(blue)), NewSizedBoxExpand(nil))),
					NewExpanded(NewSizedBoxExpand(nil)),
				})
			},
			wantSent: true, wantStatus: someColor(blue), wantNav: someColor(blue),
		},
		{
			name:      "Bottom only AnnotatedRegion provides status bar and navigation bar style properties",
			platforms: android, device: true,
			build: func() Widget {
				return NewColumn([]Widget{
					NewExpanded(NewSizedBoxExpand(nil)),
					NewExpanded(region(style(someColor(green), someColor(green)), NewSizedBoxExpand(nil))),
				})
			},
			wantSent: true, wantStatus: someColor(green), wantNav: someColor(green),
		},
	}
	for _, tc := range tests {
		for _, platform := range tc.platforms {
			t.Run(tc.name+"/"+platform.String(), func(t *testing.T) {
				skipUnlessDebug(t) // the platform override is a debug-only feature
				foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
				t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
				ts := gluttertest.New(t)
				if tc.device {
					ts.SetViewConfiguration(func(c *ui.ViewConfiguration) {
						padding := ui.ViewPadding{Top: statusBarHeight * devicePixelRatio, Bottom: navigationBarHeight * devicePixelRatio}
						c.ViewPadding, c.Padding = padding, padding
						c.DevicePixelRatio = devicePixelRatio
						c.PhysicalSize = ui.Size{Width: deviceWidth * devicePixelRatio, Height: deviceHeight * devicePixelRatio}
					})
				}
				ts.PumpWidget(tc.build())
				ts.PumpAndSettle()
				ts.FlushPosted()
				got, sent := ts.Binding.SystemChrome().LatestStyle()
				if sent != tc.wantSent {
					t.Fatalf("style sent = %v (%v), want %v", sent, got, tc.wantSent)
				}
				if !sent {
					return
				}
				if got.StatusBarColor != tc.wantStatus || got.SystemNavigationBarColor != tc.wantNav {
					t.Errorf("status bar color = %v, navigation bar color = %v; want %v and %v",
						got.StatusBarColor, got.SystemNavigationBarColor, tc.wantStatus, tc.wantNav)
				}
			})
		}
	}
}
