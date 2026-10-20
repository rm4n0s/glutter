// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestMaterialBanner(t *testing.T) {
	ts, key := snackApp(t, plainScaffold)
	messenger, _ := key.CurrentState()
	pressed := 0
	controller := messenger.ShowMaterialBanner(NewMaterialBanner(widgets.NewText("banner"), []widgets.Widget{
		widgets.NewGestureDetector(widgets.NewText("DISMISS"), widgets.GestureDetectorOpts{OnTap: func() {
			pressed++
			messenger.HideCurrentMaterialBanner(MaterialBannerClosedReasonDismiss)
		}}),
	}))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("banner")) {
		t.Fatal("the banner is not shown")
	}
	// The banner pushes the body down by its height.
	banner := gluttertest.ByType[*MaterialBanner]()
	bannerHeight := ts.Size(banner).Height
	if bannerHeight < 48 {
		t.Errorf("banner height = %v, want at least the 48 of an action", bannerHeight)
	}
	if got := ts.TopLeft(gluttertest.ByType[*widgets.ColoredBox]()).Dy; got != bannerHeight {
		t.Errorf("body top = %v, want the banner height %v", got, bannerHeight)
	}
	ts.Tap(gluttertest.ByText("DISMISS"))
	ts.PumpAndSettle()
	if pressed != 1 || ts.Exists(gluttertest.ByText("banner")) {
		t.Errorf("pressed %d times, banner still shown: %v", pressed, ts.Exists(gluttertest.ByText("banner")))
	}
	select {
	case reason := <-controller.Closed():
		if reason != MaterialBannerClosedReasonDismiss {
			t.Errorf("reason = %v, want dismiss", reason)
		}
	default:
		t.Error("Closed did not complete")
	}
}

func TestMaterialBannerClosedReasons(t *testing.T) {
	tests := []struct {
		name string
		do   func(m *ScaffoldMessengerState)
		want MaterialBannerClosedReason
	}{
		{"hide", func(m *ScaffoldMessengerState) { m.HideCurrentMaterialBanner() }, MaterialBannerClosedReasonHide},
		{"remove", func(m *ScaffoldMessengerState) { m.RemoveCurrentMaterialBanner() }, MaterialBannerClosedReasonRemove},
		{"clear", func(m *ScaffoldMessengerState) { m.ClearMaterialBanners() }, MaterialBannerClosedReasonHide},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, key := snackApp(t, plainScaffold)
			messenger, _ := key.CurrentState()
			controller := messenger.ShowMaterialBanner(NewMaterialBanner(widgets.NewText("banner"), []widgets.Widget{widgets.NewText("OK")}))
			ts.PumpAndSettle()
			tc.do(messenger)
			ts.PumpAndSettle()
			select {
			case got := <-controller.Closed():
				if got != tc.want {
					t.Errorf("reason = %v, want %v", got, tc.want)
				}
			default:
				t.Error("Closed did not complete")
			}
			if ts.Exists(gluttertest.ByText("banner")) {
				t.Error("the banner is still shown")
			}
		})
	}
}

func TestMaterialBannerQueue(t *testing.T) {
	ts, key := snackApp(t, plainScaffold)
	messenger, _ := key.CurrentState()
	messenger.ShowMaterialBanner(NewMaterialBanner(widgets.NewText("first"), []widgets.Widget{widgets.NewText("OK")}))
	messenger.ShowMaterialBanner(NewMaterialBanner(widgets.NewText("second"), []widgets.Widget{widgets.NewText("OK")}))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("first")) || ts.Exists(gluttertest.ByText("second")) {
		t.Fatal("only the first banner is shown at a time")
	}
	messenger.HideCurrentMaterialBanner()
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	if ts.Exists(gluttertest.ByText("first")) || !ts.Exists(gluttertest.ByText("second")) {
		t.Fatal("the second banner replaces the first")
	}
}

func TestMaterialBannerErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		run  func(t *testing.T)
		want string
	}{
		{"negative elevation", func(*testing.T) {
			NewMaterialBanner(widgets.NewText("x"), []widgets.Widget{widgets.NewText("OK")}, MaterialBannerOpts{Elevation: foundation.Some(-1.0)})
		}, "MaterialBannerElevationNegative"},
		{"no actions", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(widgets.NewMediaQueryData(),
				NewMaterialBanner(widgets.NewText("x"), nil)))))
			err := ts.TakeError()
			if err == nil {
				t.Fatal("no error")
			}
			panic(err)
		}, "MaterialBannerNoActions"},
		{"no scaffold", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			var state *ScaffoldMessengerState
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewScaffoldMessenger(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				state = ScaffoldMessengerOf(bc)
				return widgets.NewSizedBox(nil)
			}))))
			state.ShowMaterialBanner(NewMaterialBanner(widgets.NewText("x"), []widgets.Widget{widgets.NewText("OK")}))
		}, "ShowMaterialBannerNoScaffold"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := recoverError(func() { tc.run(t) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tc.want) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
