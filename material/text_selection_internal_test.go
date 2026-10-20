// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestMaterialTextSelectionHandleColor(t *testing.T) {
	handleColor := ui.NewColor(0xFFAA3300)
	tests := []struct {
		name  string
		theme TextSelectionThemeData
		want  func(s *ColorScheme) ui.Color
	}{
		{"primary by default", TextSelectionThemeData{}, func(s *ColorScheme) ui.Color { return s.Primary() }},
		{"the handle color of the theme", NewTextSelectionThemeData(TextSelectionThemeDataOpts{SelectionHandleColor: foundation.Some(handleColor)}),
			func(*ColorScheme) ui.Color { return handleColor }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
				Theme: NewThemeData(ThemeDataOpts{TextSelectionTheme: tc.theme}),
				Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					return widgets.NewAlign(MaterialTextSelectionHandleControls.BuildHandle(bc, rendering.TextSelectionHandleTypeLeft, 14, nil),
						widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
				})}),
			}))
			scheme := ThemeOf(ts.Element(gluttertest.ByType[*Scaffold]())).ColorScheme
			var painter *textSelectionHandlePainter
			for _, w := range ts.Widgets(gluttertest.ByType[*widgets.CustomPaint]()) {
				if p, ok := w.(*widgets.CustomPaint).Painter().(*textSelectionHandlePainter); ok {
					painter = p
				}
			}
			if painter == nil {
				t.Fatal("no handle painter")
			}
			if painter.color != tc.want(scheme) {
				t.Errorf("handle color = %v, want %v", painter.color, tc.want(scheme))
			}
			if painter.ShouldRepaint(&textSelectionHandlePainter{color: ui.NewColor(1)}) != true || painter.ShouldRepaint(&textSelectionHandlePainter{color: painter.color}) {
				t.Error("ShouldRepaint is wrong")
			}
		})
	}
}
