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
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func TestPreferredSize(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 100}})
	child := NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(20.0)})
	w := NewPreferredSize(ui.Size{Width: 50, Height: 60}, child)
	var asInterface PreferredSizeWidget = w
	if got := asInterface.PreferredSize(); got != (ui.Size{Width: 50, Height: 60}) {
		t.Errorf("PreferredSize = %v", got)
	}
	if w.Child() != Widget(child) {
		t.Error("Child is not the child")
	}
	// It does not constrain the child: the child has the size its own widget gives.
	ts.PumpWidget(NewAlign(w))
	if got := ts.Size(gluttertest.ByWidget(child)); got != (ui.Size{Width: 30, Height: 20}) {
		t.Errorf("child size = %v, want 30 x 20", got)
	}
}
