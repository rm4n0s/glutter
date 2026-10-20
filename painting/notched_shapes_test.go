// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestNotchedShapes(t *testing.T) {
	host := ui.NewRectFromLTWH(0, 100, 400, 60)
	tests := []struct {
		name      string
		shape     NotchedShape
		guest     foundation.Opt[ui.Rect]
		inside    ui.Offset // a point in the bar, under the guest
		wantInner bool
	}{
		{"circular without a guest", CircularNotchedRectangle{}, foundation.None[ui.Rect](), ui.Offset{Dx: 200, Dy: 110}, true},
		{"circular with a guest that is away", CircularNotchedRectangle{}, foundation.Some(ui.NewRectFromLTWH(0, 0, 40, 40)), ui.Offset{Dx: 200, Dy: 110}, true},
		{"circular with a docked guest", CircularNotchedRectangle{}, foundation.Some(ui.NewRectFromLTWH(172, 72, 56, 56)), ui.Offset{Dx: 200, Dy: 110}, false},
		{"circular, outside the notch", CircularNotchedRectangle{}, foundation.Some(ui.NewRectFromLTWH(172, 72, 56, 56)), ui.Offset{Dx: 20, Dy: 110}, true},
		{"automatic shape ignores the guest", AutomaticNotchedShape{Host: NewRoundedRectangleBorder()}, foundation.Some(ui.NewRectFromLTWH(172, 72, 56, 56)), ui.Offset{Dx: 200, Dy: 110}, true},
		{"automatic shape rounds the corners", AutomaticNotchedShape{Host: NewRoundedRectangleBorder(RoundedRectangleBorderOpts{BorderRadius: NewBorderRadiusCircular(30)})},
			foundation.None[ui.Rect](), ui.Offset{Dx: 1, Dy: 101}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.shape.GetOuterPath(host, tc.guest)
			if got := path.Contains(tc.inside); got != tc.wantInner {
				t.Errorf("Contains(%v) = %v, want %v", tc.inside, got, tc.wantInner)
			}
		})
	}
}
