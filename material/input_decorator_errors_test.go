// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestInputDecoratorErrors checks the route of the failures of the input
// decoration, the input borders and the input decorator.
func TestInputDecoratorErrors(t *testing.T) {
	skipUnlessDebug(t)
	paintGap := func(border OutlineInputBorder, percentage float64) {
		recorder := ui.NewPictureRecorder()
		border.PaintGap(ui.NewCanvas(recorder), ui.NewRectFromLTWH(0, 0, 200, 56),
			InputBorderGap{Start: foundation.Some(10.0), Extent: 30, Percentage: percentage}, ui.TextDirectionLTR)
	}
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"label and labelText", func() {
			NewInputDecoration(InputDecorationOpts{Label: widgets.NewText("a"), LabelText: str("b")})
		}, "InputDecorationBothGiven"},
		{"error and errorText", func() {
			NewInputDecoration(InputDecorationOpts{Error: widgets.NewText("a"), ErrorText: str("b")})
		}, "InputDecorationBothGiven"},
		{"negative gap padding", func() {
			NewOutlineInputBorder(OutlineInputBorderOpts{GapPadding: foundation.Some(-1.0)})
		}, "OutlineInputBorderGapPadding"},
		{"gap percentage above one", func() { paintGap(NewOutlineInputBorder(), 1.5) }, "InputBorderGapPercentage"},
		{"gap percentage below zero", func() { paintGap(NewOutlineInputBorder(), -0.5) }, "InputBorderGapPercentage"},
		{"corners that are not circular", func() {
			paintGap(NewOutlineInputBorder(OutlineInputBorderOpts{BorderRadius: foundation.Some(painting.NewBorderRadiusAll(ui.NewRadiusElliptical(4, 8)))}), 0.5)
		}, "OutlineInputBorderCorners"},
		{"unbounded width", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 200}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewRow([]widgets.Widget{
				NewInputDecorator(InputDecoration{LabelText: str("x")}, InputDecoratorOpts{Child: widgets.NewText("y")})})})}))
			takeAndPanic(ts)
		}, "InputDecoratorUnboundedWidth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
}
