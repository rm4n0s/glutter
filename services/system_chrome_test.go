// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/services/system_chrome_test.dart (the
// SystemUiOverlayStyle value cases).
func TestSystemUiOverlayStyleCopyWith(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	tests := []struct {
		name string
		base SystemUiOverlayStyle
		c    SystemUiOverlayStyleCopy
		want SystemUiOverlayStyle
	}{
		{"nothing set", SystemUiOverlayStyleLight, SystemUiOverlayStyleCopy{}, SystemUiOverlayStyleLight},
		{
			"status bar color",
			SystemUiOverlayStyleDark, SystemUiOverlayStyleCopy{StatusBarColor: foundation.Some(red)},
			SystemUiOverlayStyleDark.CopyWith(SystemUiOverlayStyleCopy{StatusBarColor: foundation.Some(red)}),
		},
		{
			"brightness",
			SystemUiOverlayStyleDark, SystemUiOverlayStyleCopy{StatusBarIconBrightness: ui.BrightnessLight, SystemStatusBarContrastEnforced: foundation.Some(true)},
			SystemUiOverlayStyle{
				SystemNavigationBarColor:          foundation.Some(ui.NewColor(0xFF000000)),
				SystemNavigationBarIconBrightness: ui.BrightnessLight,
				StatusBarIconBrightness:           ui.BrightnessLight,
				StatusBarBrightness:               ui.BrightnessLight,
				SystemStatusBarContrastEnforced:   foundation.Some(true),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.base.CopyWith(tc.c); got != tc.want {
				t.Errorf("CopyWith = %v, want %v", got, tc.want)
			}
		})
	}
	if SystemUiOverlayStyleLight == SystemUiOverlayStyleDark {
		t.Error("light and dark are equal")
	}
	if got := SystemUiOverlayStyleLight.StatusBarIconBrightness; got != ui.BrightnessLight {
		t.Errorf("light icons = %v", got)
	}
}

func TestSystemUiOverlayStyleDiagnostics(t *testing.T) {
	s := SystemUiOverlayStyle{StatusBarColor: foundation.Some(ui.NewColor(0xFF112233))}
	if got := s.String(); got == "" || s.ToStringShort() != "SystemUiOverlayStyle" {
		t.Errorf("String = %q", got)
	}
}
