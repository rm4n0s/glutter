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

// Tests AnnotatedRegion (Flutter's tests of it are layer-search tests, which
// rendering's layer tests cover): the widget configures its render object.
func TestAnnotatedRegionConfiguresTheRenderObject(t *testing.T) {
	tests := []struct {
		name  string
		w     func() *AnnotatedRegion
		value any
		sized bool
	}{
		{"sized by default", func() *AnnotatedRegion { return NewAnnotatedRegion("a", newBox(10, 10)) }, "a", true},
		{"not sized", func() *AnnotatedRegion {
			return NewAnnotatedRegion(3, newBox(10, 10), AnnotatedRegionOpts{NotSized: true})
		}, 3, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tc.w(), ui.TextDirectionLTR)
			r := ts.RenderObject(gluttertest.ByType[*AnnotatedRegion]()).(*rendering.RenderAnnotatedRegion)
			if r.Value() != tc.value || r.Sized() != tc.sized {
				t.Errorf("render object = (%v, %v), want (%v, %v)", r.Value(), r.Sized(), tc.value, tc.sized)
			}
			if w := gluttertest.WidgetAs[*AnnotatedRegion](ts, gluttertest.ByType[*AnnotatedRegion]()); w.Value() != tc.value || w.Sized() != tc.sized {
				t.Errorf("widget = (%v, %v)", w.Value(), w.Sized())
			}
		})
	}
}

func TestAnnotatedRegionUpdates(t *testing.T) {
	ts := gluttertest.New(t)
	pumpLoose(ts, NewAnnotatedRegion("a", newBox(10, 10)), ui.TextDirectionLTR)
	pumpLoose(ts, NewAnnotatedRegion("b", newBox(10, 10), AnnotatedRegionOpts{NotSized: true}), ui.TextDirectionLTR)
	r := ts.RenderObject(gluttertest.ByType[*AnnotatedRegion]()).(*rendering.RenderAnnotatedRegion)
	if r.Value() != "b" || r.Sized() {
		t.Errorf("render object = (%v, %v), want (b, false)", r.Value(), r.Sized())
	}
}
