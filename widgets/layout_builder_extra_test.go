// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestLayoutBuilderExtra checks that the builder runs again when the extra value
// changes at layout time, though the constraints do not change (the replacement
// of the BoxConstraints subclass of Flutter's Scaffold).
func TestLayoutBuilderExtra(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 100}})
	extra := 1
	var seen []int
	// The same widget stays in the tree: only the layout can tell that the extra
	// value is not the one of the last build.
	builder := NewLayoutBuilder(func(_ BuildContext, _ rendering.BoxConstraints) Widget {
		seen = append(seen, extra)
		return NewSizedBox(nil)
	}, LayoutBuilderOpts{Extra: func() any { return extra }})
	ts.PumpWidget(builder)
	tests := []struct {
		name       string
		change     func()
		wantBuilds int
		wantLast   int
	}{
		{"layout again with the same value", func() {}, 1, 1},
		{"the extra value changes", func() { extra = 2 }, 2, 2},
		{"the same value again", func() {}, 2, 2},
		{"another change", func() { extra = 3 }, 3, 3},
	}
	// Not subtests: the render objects belong to the goroutine of the test.
	for _, tc := range tests {
		tc.change()
		ts.RenderObject(gluttertest.ByType[*LayoutBuilder]()).MarkNeedsLayout()
		ts.Pump()
		if len(seen) != tc.wantBuilds || seen[len(seen)-1] != tc.wantLast {
			t.Errorf("%s: builds = %d (last saw %d), want %d (%d)", tc.name, len(seen), seen[len(seen)-1], tc.wantBuilds, tc.wantLast)
		}
	}
}
