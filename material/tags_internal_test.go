// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestPrivateWidgetErrors checks the route of failures that only the private
// widgets of the package can reach.
func TestPrivateWidgetErrors(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the checks are for debug builds")
	}
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"drag animation while the index changes", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			var controller *TabController
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(3, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				controller = DefaultTabControllerOf(bc)
				return widgets.NewSizedBox(nil)
			}))}))
			controller.AnimateTo(2)
			_ = (&dragAnimation{changeAnimation: changeAnimation{controller: controller}, index: 0}).Value()
		}, "TabBarDragWhileChanging"},
		{"flexible space without settings", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: &scrollUnderFlexibleSpace{
				StatelessWidget: widgets.NewStatelessWidgetBase(),
				configBuilder:   func(widgets.BuildContext) scrollUnderFlexibleConfig { return scrollUnderFlexibleConfig{} },
			}})}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "ScrollUnderFlexibleSpaceNoSettings"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, _ = r.(error)
					}
				}()
				tc.run(t)
			}()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tc.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
