// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func sizedBox(w, h float64) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)})
}

func TestDivider(t *testing.T) {
	tests := []struct {
		name       string
		material3  bool
		divider    widgets.Widget
		wantHeight float64
		wantLine   float64
	}{
		{"M3 default", true, NewDivider(), 16, 1},
		{"M2 default", false, NewDivider(), 16, 0},
		{"height", true, NewDivider(DividerOpts{Height: foundation.Some(30.0), Thickness: foundation.Some(4.0)}), 30, 4},
		{"zero height", true, NewDivider(DividerOpts{Height: foundation.Some(0.0)}), 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(tc.material3), widgets.NewSizedBox(tc.divider, widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
			ts.PumpAndSettle()
			if got := ts.Size(gluttertest.ByType[*Divider]()); got != (ui.Size{Width: 200, Height: tc.wantHeight}) {
				t.Errorf("size %v, want 200x%v", got, tc.wantHeight)
			}
			line := gluttertest.Descendant(gluttertest.ByType[*Divider](), gluttertest.ByType[*widgets.DecoratedBox]())
			if got := ts.Size(line).Height; got != tc.wantLine {
				t.Errorf("line thickness %v, want %v", got, tc.wantLine)
			}
		})
	}
}

func TestDividerIndentAndVertical(t *testing.T) {
	ts := pumpButton(t, standardTheme(true), widgets.NewSizedBox(NewDivider(DividerOpts{Indent: foundation.Some(20.0), EndIndent: foundation.Some(30.0)}),
		widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
	ts.PumpAndSettle()
	line := gluttertest.Descendant(gluttertest.ByType[*Divider](), gluttertest.ByType[*widgets.DecoratedBox]())
	if got := ts.Size(line).Width; got != 150 {
		t.Errorf("line width %v, want 150", got)
	}
	ts = pumpButton(t, standardTheme(true), widgets.NewSizedBox(NewVerticalDivider(VerticalDividerOpts{Width: foundation.Some(10.0)}),
		widgets.SizedBoxOpts{Height: foundation.Some(100.0)}))
	ts.PumpAndSettle()
	if got := ts.Size(gluttertest.ByType[*VerticalDivider]()); got != (ui.Size{Width: 10, Height: 100}) {
		t.Errorf("vertical divider size %v", got)
	}
}

func TestDividerTheme(t *testing.T) {
	theme := DividerThemeData{Space: foundation.Some(40.0), Thickness: foundation.Some(3.0), Indent: foundation.Some(5.0)}
	tests := []struct {
		name  string
		build func() widgets.Widget
	}{
		{"theme widget", func() widgets.Widget { return NewDividerTheme(theme, NewDivider()) }},
		{"theme data", func() widgets.Widget { return NewDivider() }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			th := standardTheme(true)
			if tc.name == "theme data" {
				th = th.CopyWith(ThemeDataCopy{DividerTheme: foundation.Some(theme)})
			}
			ts := pumpButton(t, th, widgets.NewSizedBox(tc.build(), widgets.SizedBoxOpts{Width: foundation.Some(100.0)}))
			ts.PumpAndSettle()
			if got := ts.Size(gluttertest.ByType[*Divider]()); got.Height != 40 {
				t.Errorf("height %v, want 40", got.Height)
			}
			line := gluttertest.Descendant(gluttertest.ByType[*Divider](), gluttertest.ByType[*widgets.DecoratedBox]())
			if got := ts.Size(line); got.Height != 3 || got.Width != 95 {
				t.Errorf("line %v, want 95x3", got)
			}
		})
	}
}

func TestDividerCreateBorderSide(t *testing.T) {
	if got := DividerCreateBorderSide(nil); got.Width != 0 || got.Color != Colors.Black {
		t.Errorf("nil context: %v", got)
	}
	var m3, m2 painting.BorderSide
	capture := func(dst *painting.BorderSide) widgets.Widget {
		return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			*dst = DividerCreateBorderSide(bc)
			return widgets.NewSizedBox(nil)
		})
	}
	pumpButton(t, standardTheme(true), capture(&m3))
	pumpButton(t, standardTheme(false), capture(&m2))
	if m3.Width != 1 {
		t.Errorf("M3 width %v, want 1", m3.Width)
	}
	if m2.Width != 0 {
		t.Errorf("M2 width %v, want 0", m2.Width)
	}
	if m3.Color == m2.Color {
		t.Errorf("both have the color %v", m3.Color)
	}
}

func TestCard(t *testing.T) {
	tests := []struct {
		name      string
		material3 bool
		card      func(child widgets.Widget) widgets.Widget
		wantSize  ui.Size
	}{
		{"elevated M3", true, func(c widgets.Widget) widgets.Widget { return NewCard(c) }, ui.Size{Width: 108, Height: 58}},
		{"filled", true, func(c widgets.Widget) widgets.Widget { return NewCardFilled(c) }, ui.Size{Width: 108, Height: 58}},
		{"outlined", true, func(c widgets.Widget) widgets.Widget { return NewCardOutlined(c) }, ui.Size{Width: 108, Height: 58}},
		{"M2", false, func(c widgets.Widget) widgets.Widget { return NewCard(c) }, ui.Size{Width: 108, Height: 58}},
		{"margin", true, func(c widgets.Widget) widgets.Widget {
			return NewCard(c, CardOpts{Margin: painting.NewEdgeInsetsAll(10)})
		}, ui.Size{Width: 120, Height: 70}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(tc.material3), tc.card(sizedBox(100, 50)))
			if got := ts.Size(gluttertest.ByType[*Card]()); got != tc.wantSize {
				t.Errorf("size %v, want %v", got, tc.wantSize)
			}
		})
	}
}

func TestCardTheme(t *testing.T) {
	data := CardThemeData{Margin: painting.NewEdgeInsetsAll(20), Elevation: foundation.Some(8.0)}
	for _, withWidget := range []bool{true, false} {
		th := standardTheme(true)
		var card widgets.Widget = NewCard(sizedBox(100, 50))
		if withWidget {
			card = NewCardTheme(data, card)
		} else {
			th = th.CopyWith(ThemeDataCopy{CardTheme: foundation.Some(data)})
		}
		ts := pumpButton(t, th, card)
		if got := ts.Size(gluttertest.ByType[*Card]()); got != (ui.Size{Width: 140, Height: 90}) {
			t.Errorf("widget=%v: size %v, want 140x90", withWidget, got)
		}
	}
}

func TestBadge(t *testing.T) {
	tests := []struct {
		name  string
		badge widgets.Widget
		want  ui.Size
	}{
		{"dot", NewBadge(nil), ui.Size{Width: 6, Height: 6}},
		{"label", NewBadge(nil, BadgeOpts{Label: widgets.NewText("3")}), ui.Size{Width: 16, Height: 16}},
		{"count", NewBadgeCount(1234, nil), ui.Size{Width: 16, Height: 16}},
		{"hidden with a child", NewBadge(sizedBox(30, 20), BadgeOpts{Label: widgets.NewText("3"), NoLabelVisible: true}), ui.Size{Width: 30, Height: 20}},
		{"over a child", NewBadge(sizedBox(30, 20), BadgeOpts{Label: widgets.NewText("3")}), ui.Size{Width: 30, Height: 20}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), tc.badge)
			size := ts.Size(gluttertest.ByType[*Badge]())
			if tc.name == "count" {
				if size.Height != tc.want.Height || size.Width <= 16 {
					t.Errorf("size %v: the label of 999+ must widen the badge", size)
				}
				return
			}
			if tc.name == "label" && size.Height != 16 {
				t.Errorf("size %v, want a height of 16", size)
				return
			}
			if tc.name != "label" && size != tc.want {
				t.Errorf("size %v, want %v", size, tc.want)
			}
		})
	}
	ts := pumpButton(t, standardTheme(true), NewBadgeCount(1234, nil))
	if !ts.Exists(gluttertest.ByText("999+")) {
		t.Error("a count above the maximum shows 999+")
	}
	ts = pumpButton(t, standardTheme(true), NewBadgeCount(7, nil, BadgeCountOpts{MaxCount: foundation.Some(5)}))
	if !ts.Exists(gluttertest.ByText("5+")) {
		t.Error("the max count is the one given")
	}
}

func TestBadgePlacement(t *testing.T) {
	ts := pumpButton(t, standardTheme(true), NewBadge(sizedBox(40, 40), BadgeOpts{Label: widgets.NewText("9")}))
	child := ts.TopLeft(gluttertest.ByType[*widgets.SizedBox]())
	label := ts.Center(gluttertest.ByText("9"))
	// The badge is over the top end corner of the child.
	if label.Dx < child.Dx+40-10 || label.Dy > child.Dy+8 {
		t.Errorf("the label is at %v, the child at %v", label, child)
	}
}

func TestTooltip(t *testing.T) {
	ts := pumpButton(t, standardTheme(true), NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "Hello tip"}))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Hello tip")) {
		t.Fatal("the tooltip shows before a hover")
	}
	ts.Hover(ts.Center(gluttertest.ByType[*Tooltip]()))
	ts.Pump(50 * time.Millisecond)
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Hello tip")) {
		t.Fatal("the tooltip is not shown on a hover")
	}
	ts.Hover(ui.Offset{Dx: 5, Dy: 5})
	ts.Pump(time.Second)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Hello tip")) {
		t.Error("the tooltip stays after the pointer left")
	}
}

func TestTooltipOptions(t *testing.T) {
	tests := []struct {
		name  string
		child widgets.Widget
		theme func(*ThemeData) *ThemeData
		wait  time.Duration
		want  bool
	}{
		{"visibility off", NewTooltipVisibility(false, NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip"})), nil, 0, false},
		{"visibility on", NewTooltipVisibility(true, NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip"})), nil, 0, true},
		{"empty message", NewTooltip(sizedBox(40, 40), TooltipOpts{}), nil, 0, false},
		{"wait duration", NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip", WaitDuration: foundation.Some(time.Second)}), nil, 100 * time.Millisecond, false},
		{"waited", NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip", WaitDuration: foundation.Some(time.Second)}), nil, 1100 * time.Millisecond, true},
		{"rich message", NewTooltip(sizedBox(40, 40), TooltipOpts{RichMessage: painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("tip")})}), nil, 0, true},
		{"theme wait", NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip"}), func(th *ThemeData) *ThemeData {
			return th.CopyWith(ThemeDataCopy{TooltipTheme: foundation.Some(TooltipThemeData{WaitDuration: foundation.Some(2 * time.Second)})})
		}, time.Second, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			th := standardTheme(true)
			if tc.theme != nil {
				th = tc.theme(th)
			}
			ts := pumpButton(t, th, tc.child)
			ts.PumpAndSettle()
			ts.Hover(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
			ts.Pump(tc.wait + 50*time.Millisecond)
			ts.Pump(300 * time.Millisecond)
			if got := ts.Exists(gluttertest.ByText("tip")); got != tc.want {
				t.Errorf("shown = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTooltipPlacement(t *testing.T) {
	for _, below := range []bool{true, false} {
		ts := pumpButton(t, standardTheme(true), NewTooltip(sizedBox(40, 40), TooltipOpts{Message: "tip", PreferBelow: foundation.Some(below)}))
		target := ts.Center(gluttertest.ByType[*widgets.SizedBox]())
		ts.Hover(target)
		ts.Pump(400 * time.Millisecond)
		y := ts.Center(gluttertest.ByText("tip")).Dy
		if below && y <= target.Dy || !below && y >= target.Dy {
			t.Errorf("preferBelow=%v: the tooltip is at y=%v, the target at %v", below, y, target.Dy)
		}
	}
}

func TestDisplayThemeData(t *testing.T) {
	color := ui.NewColor(0xFF112233)
	t.Run("divider", func(t *testing.T) {
		a := DividerThemeData{Color: foundation.Some(color), Space: foundation.Some(10.0)}
		b := DividerThemeData{Color: foundation.Some(ui.NewColor(0xFFFFFFFF)), Space: foundation.Some(20.0)}
		if got := DividerThemeDataLerp(a, b, 0.5); got.Space != foundation.Some(15.0) {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(DividerThemeDataCopy{})) || a.Equal(a.CopyWith(DividerThemeDataCopy{Space: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
		if !NewThemeData().DividerTheme.Equal(DividerThemeData{}) {
			t.Error("the default theme has a divider theme")
		}
	})
	t.Run("card", func(t *testing.T) {
		a := CardThemeData{Elevation: foundation.Some(2.0), ClipBehavior: ui.ClipNone}
		b := CardThemeData{Elevation: foundation.Some(6.0), ClipBehavior: ui.ClipHardEdge}
		if got := CardThemeDataLerp(a, b, 0.5); got.Elevation != foundation.Some(4.0) || got.ClipBehavior != ui.ClipHardEdge {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(CardThemeDataCopy{})) || a.Equal(a.CopyWith(CardThemeDataCopy{Elevation: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
	})
	t.Run("badge", func(t *testing.T) {
		a := BadgeThemeData{Offset: foundation.Some(ui.Offset{Dx: 0, Dy: 0}), LargeSize: foundation.Some(10.0)}
		b := BadgeThemeData{Offset: foundation.Some(ui.Offset{Dx: 10, Dy: 20}), LargeSize: foundation.Some(20.0)}
		got := BadgeThemeDataLerp(a, b, 0.5)
		if got.Offset != foundation.Some(ui.Offset{Dx: 5, Dy: 10}) || got.LargeSize != foundation.Some(15.0) {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(BadgeThemeDataCopy{})) || a.Equal(a.CopyWith(BadgeThemeDataCopy{LargeSize: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
	})
	t.Run("tooltip", func(t *testing.T) {
		a := TooltipThemeData{VerticalOffset: foundation.Some(10.0), WaitDuration: foundation.Some(time.Second), PreferBelow: foundation.Some(true)}
		b := TooltipThemeData{VerticalOffset: foundation.Some(20.0), PreferBelow: foundation.Some(false)}
		got := TooltipThemeDataLerp(a, b, 0.25)
		// Flutter's lerp leaves the durations out, and takes the first of the
		// flags before the middle.
		if got.VerticalOffset != foundation.Some(12.5) || got.WaitDuration.IsSome() || got.PreferBelow != foundation.Some(true) {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(TooltipThemeDataCopy{})) || a.Equal(a.CopyWith(TooltipThemeDataCopy{VerticalOffset: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
	})
	t.Run("expansion tile", func(t *testing.T) {
		a := ExpansionTileThemeData{BackgroundColor: foundation.Some(color), ClipBehavior: ui.ClipNone}
		b := ExpansionTileThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), ClipBehavior: ui.ClipHardEdge}
		if got := ExpansionTileThemeDataLerp(a, b, 0.5); got.ClipBehavior != ui.ClipHardEdge || got.BackgroundColor == a.BackgroundColor {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(ExpansionTileThemeDataCopy{})) || a.Equal(a.CopyWith(ExpansionTileThemeDataCopy{IconColor: foundation.Some(color)})) {
			t.Error("Equal / CopyWith")
		}
	})
	t.Run("list tile", func(t *testing.T) {
		a := ListTileThemeData{Dense: foundation.Some(true), MinTileHeight: foundation.Some(40.0), Style: ListTileStyleList}
		b := ListTileThemeData{Dense: foundation.Some(false), MinTileHeight: foundation.Some(60.0), Style: ListTileStyleDrawer}
		got := ListTileThemeDataLerp(a, b, 0.75)
		if got.MinTileHeight != foundation.Some(55.0) || got.Style != ListTileStyleDrawer || got.Dense != foundation.Some(false) {
			t.Errorf("lerp %v", got)
		}
		if !a.Equal(a.CopyWith(ListTileThemeDataCopy{})) || a.Equal(a.CopyWith(ListTileThemeDataCopy{MinTileHeight: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
	})
}

func TestDisplayErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		f     func()
		route string
	}{
		{"divider height", func() { NewDivider(DividerOpts{Height: foundation.Some(-1.0)}) }, "DividerSizeNegative"},
		{"divider thickness", func() { NewDivider(DividerOpts{Thickness: foundation.Some(-1.0)}) }, "DividerThicknessNegative"},
		{"divider indent", func() { NewDivider(DividerOpts{Indent: foundation.Some(-1.0)}) }, "DividerIndentNegative"},
		{"divider end indent", func() { NewDivider(DividerOpts{EndIndent: foundation.Some(-1.0)}) }, "DividerEndIndentNegative"},
		{"vertical divider width", func() { NewVerticalDivider(VerticalDividerOpts{Width: foundation.Some(-1.0)}) }, "VerticalDividerSizeNegative"},
		{"vertical divider thickness", func() { NewVerticalDivider(VerticalDividerOpts{Thickness: foundation.Some(-1.0)}) }, "VerticalDividerThicknessNegative"},
		{"card elevation", func() { NewCard(nil, CardOpts{Elevation: foundation.Some(-1.0)}) }, "CardElevationNegative"},
		{"badge count", func() { NewBadgeCount(-1, nil) }, "BadgeCountNegative"},
		{"badge max count", func() { NewBadgeCount(1, nil, BadgeCountOpts{MaxCount: foundation.Some(0)}) }, "BadgeMaxCountNotPositive"},
		{"tooltip both messages", func() {
			NewTooltip(nil, TooltipOpts{Message: "a", RichMessage: painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("b")})})
		}, "TooltipMessageAndRichMessage"},
		{"list tile three line", func() { NewListTile(ListTileOpts{IsThreeLine: foundation.Some(true)}) }, "ListTileThreeLineNoSubtitle"},
		{"divide tiles", func() { ListTileDivideTiles(nil, nil, foundation.None[ui.Color]()) }, "ListTileDivideTilesNoColor"},
		{"checkbox tile value", func() { NewCheckboxListTile(foundation.None[bool](), nil) }, "CheckboxListTileValueNull"},
		{"checkbox tile three line", func() {
			NewCheckboxListTile(foundation.Some(true), nil, CheckboxListTileOpts{IsThreeLine: foundation.Some(true)})
		}, "CheckboxListTileThreeLineNoSubtitle"},
		{"switch tile three line", func() { NewSwitchListTile(true, nil, SwitchListTileOpts{IsThreeLine: foundation.Some(true)}) }, "SwitchListTileThreeLineNoSubtitle"},
		{"switch tile image error", func() {
			NewSwitchListTile(true, nil, SwitchListTileOpts{OnActiveThumbImageError: func(error) {}})
		}, "SwitchListTileThumbImageErrorWithoutImage"},
		{"radio tile three line", func() {
			NewRadioListTile(1, RadioListTileOpts[int]{IsThreeLine: foundation.Some(true)})
		}, "RadioListTileThreeLineNoSubtitle"},
		{"expansion tile baseline", func() {
			NewExpansionTile(nil, ExpansionTileOpts{ExpandedCrossAxisAlignment: rendering.CrossAxisAlignmentBaseline})
		}, "ExpansionTileCrossAxisAlignmentBaseline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.f), tt.route)
		})
	}
}
