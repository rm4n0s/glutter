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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// pumpTile shows the widget in a Material app 800 wide.
func pumpTile(t *testing.T, theme *ThemeData, child widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(child,
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	ts.PumpAndSettle()
	return ts
}

func TestListTileHeights(t *testing.T) {
	text := func(s string) widgets.Widget { return widgets.NewText(s) }
	tests := []struct {
		name      string
		material3 bool
		tile      ListTileOpts
		want      float64
	}{
		{"one line", true, ListTileOpts{Title: text("A")}, 56},
		{"two lines", true, ListTileOpts{Title: text("A"), Subtitle: text("B")}, 72},
		{"three lines", true, ListTileOpts{Title: text("A"), Subtitle: text("B"), IsThreeLine: foundation.Some(true)}, 88},
		{"dense one line", true, ListTileOpts{Title: text("A"), Dense: foundation.Some(true)}, 48},
		{"dense two lines", true, ListTileOpts{Title: text("A"), Subtitle: text("B"), Dense: foundation.Some(true)}, 64},
		{"dense three lines M2", false, ListTileOpts{Title: text("A"), Subtitle: text("B"), Dense: foundation.Some(true), IsThreeLine: foundation.Some(true)}, 76},
		{"M2 one line", false, ListTileOpts{Title: text("A")}, 56},
		{"min tile height", true, ListTileOpts{Title: text("A"), MinTileHeight: foundation.Some(100.0)}, 100},
		{"tall title", true, ListTileOpts{Title: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(80.0)})}, 96},
		{"compact density", true, ListTileOpts{Title: text("A"), VisualDensity: foundation.Some(VisualDensityCompact())}, 56 - 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpTile(t, standardTheme(tc.material3), widgets.NewSizedBox(NewListTile(tc.tile), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			if got := ts.Size(gluttertest.ByType[*ListTile]()); got != (ui.Size{Width: 400, Height: tc.want}) {
				t.Errorf("size %v, want 400x%v", got, tc.want)
			}
		})
	}
}

func TestListTileLayout(t *testing.T) {
	lead := sizedBox(24, 24)
	trail := sizedBox(30, 30)
	tile := NewListTile(ListTileOpts{Leading: lead, Title: widgets.NewText("Title"), Trailing: trail})
	for _, dir := range []ui.TextDirection{ui.TextDirectionLTR, ui.TextDirectionRTL} {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: widgets.NewDirectionality(dir, NewScaffold(ScaffoldOpts{
			Body: widgets.NewAlign(widgets.NewSizedBox(tile, widgets.SizedBoxOpts{Width: foundation.Some(400.0)}), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}))}))
		ts.PumpAndSettle()
		leading := ts.TopLeft(gluttertest.ByWidget(lead))
		trailing := ts.TopLeft(gluttertest.ByWidget(trail))
		title := ts.TopLeft(gluttertest.ByText("Title"))
		root := ts.TopLeft(gluttertest.ByType[*ListTile]())
		if dir == ui.TextDirectionLTR {
			// Material 3: the content padding is 16 at the start and 24 at the end.
			if leading.Dx != root.Dx+16 || trailing.Dx != root.Dx+400-24-30 || title.Dx <= leading.Dx+24 {
				t.Errorf("LTR: leading %v, title %v, trailing %v", leading, title, trailing)
			}
		} else if leading.Dx <= title.Dx || trailing.Dx >= title.Dx {
			t.Errorf("RTL: leading %v, title %v, trailing %v", leading, title, trailing)
		}
	}
}

func TestListTileTitleAlignment(t *testing.T) {
	tests := []struct {
		name  string
		align ListTileTitleAlignment
		three bool
		want  float64 // top of the leading widget, under the top of the tile
	}{
		{"top", ListTileTitleAlignmentTop, false, 8},
		{"center", ListTileTitleAlignmentCenter, false, 16},
		{"bottom", ListTileTitleAlignmentBottom, false, 72 - 24 - 8},
		{"threeLine two lines", ListTileTitleAlignmentThreeLine, false, 24},
		{"threeLine three lines", ListTileTitleAlignmentThreeLine, true, 8},
		{"titleHeight", ListTileTitleAlignmentTitleHeight, false, 16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lead := sizedBox(24, 24)
			opts := ListTileOpts{Leading: lead, Title: widgets.NewText("A"), Subtitle: widgets.NewText("B"), TitleAlignment: tc.align}
			if tc.three {
				opts.IsThreeLine = foundation.Some(true)
			}
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewListTile(opts), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			root := ts.TopLeft(gluttertest.ByType[*ListTile]())
			got := ts.TopLeft(gluttertest.ByWidget(lead)).Dy - root.Dy
			height := ts.Size(gluttertest.ByType[*ListTile]()).Height
			want := tc.want
			if tc.three {
				// A tile of three lines is 88 high.
				want = tc.want
			}
			if tc.name == "bottom" || tc.name == "center" || tc.name == "threeLine two lines" || tc.name == "titleHeight" {
				want = map[string]float64{
					"bottom":              height - 24 - 8,
					"center":              (height - 24) / 2,
					"threeLine two lines": (height - 24) / 2,
					"titleHeight":         min((height-24)/2, 16),
				}[tc.name]
			}
			if got != want {
				t.Errorf("leading is %v under the top of a tile %v high, want %v", got, height, want)
			}
		})
	}
}

func TestListTileTapAndEnabled(t *testing.T) {
	taps, longs, focus := 0, 0, 0
	tests := []struct {
		name     string
		tile     ListTileOpts
		wantTaps int
		wantLong int
	}{
		{"enabled", ListTileOpts{OnTap: func() { taps++ }, OnLongPress: func() { longs++ }}, 1, 1},
		{"disabled", ListTileOpts{OnTap: func() { taps++ }, OnLongPress: func() { longs++ }, Disabled: true}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			taps, longs = 0, 0
			tc.tile.Title = widgets.NewText("Tap me")
			tc.tile.OnFocusChange = func(bool) { focus++ }
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewListTile(tc.tile), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			ts.Tap(gluttertest.ByText("Tap me"))
			ts.LongPress(gluttertest.ByText("Tap me"))
			ts.PumpAndSettle()
			if taps != tc.wantTaps || longs != tc.wantLong {
				t.Errorf("taps %d, long presses %d; want %d and %d", taps, longs, tc.wantTaps, tc.wantLong)
			}
		})
	}
}

func TestListTileColorsAndStyles(t *testing.T) {
	var title, subtitle, leadingStyle *painting.TextStyle
	var textTheme *TextTheme
	capture := func(dst **painting.TextStyle, text string) widgets.Widget {
		return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			*dst = widgets.DefaultTextStyleOf(bc).Style()
			textTheme = ThemeOf(bc).TextTheme
			return widgets.NewText(text)
		})
	}
	selected := ui.NewColor(0xFF00AA00)
	tests := []struct {
		name      string
		material3 bool
		tile      ListTileOpts
		check     func(t *testing.T)
	}{
		{"M3 colors", true, ListTileOpts{}, func(t *testing.T) {
			if title.Color.Value() != standardTheme(true).ColorScheme.OnSurface() || subtitle.Color.Value() != standardTheme(true).ColorScheme.OnSurfaceVariant() {
				t.Errorf("title %v, subtitle %v", title.Color, subtitle.Color)
			}
		}},
		{"selected", true, ListTileOpts{Selected: true, SelectedColor: foundation.Some(selected)}, func(t *testing.T) {
			if title.Color.Value() != selected || subtitle.Color.Value() != selected {
				t.Errorf("title %v, subtitle %v, want %v", title.Color, subtitle.Color, selected)
			}
		}},
		{"text color", true, ListTileOpts{TextColor: foundation.Some(selected)}, func(t *testing.T) {
			if title.Color.Value() != selected {
				t.Errorf("title %v", title.Color)
			}
		}},
		{"disabled", true, ListTileOpts{Disabled: true}, func(t *testing.T) {
			if title.Color.Value() != standardTheme(true).DisabledColor {
				t.Errorf("title %v, want the disabled color", title.Color)
			}
		}},
		{"dense", true, ListTileOpts{Dense: foundation.Some(true)}, func(t *testing.T) {
			if title.FontSize.Value() != 13 || subtitle.FontSize.Value() != 12 {
				t.Errorf("sizes %v and %v, want 13 and 12", title.FontSize, subtitle.FontSize)
			}
		}},
		{"title style", true, ListTileOpts{TitleTextStyle: &painting.TextStyle{FontSize: foundation.Some(30.0)}}, func(t *testing.T) {
			if title.FontSize.Value() != 30 {
				t.Errorf("size %v", title.FontSize)
			}
		}},
		{"M2 drawer style", false, ListTileOpts{Style: ListTileStyleDrawer}, func(t *testing.T) {
			if title.FontSize.Value() != textTheme.BodyLarge.FontSize.Value() {
				t.Errorf("title size %v, want the body large one", title.FontSize)
			}
		}},
		{"M2 list style", false, ListTileOpts{Style: ListTileStyleList}, func(t *testing.T) {
			if title.FontSize.Value() != textTheme.TitleMedium.FontSize.Value() {
				t.Errorf("title size %v, want the title medium one", title.FontSize)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.tile
			o.Title, o.Subtitle, o.Leading = capture(&title, "T"), capture(&subtitle, "S"), capture(&leadingStyle, "L")
			pumpTile(t, standardTheme(tc.material3), widgets.NewSizedBox(NewListTile(o), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			tc.check(t)
		})
	}
}

func TestListTileTheme(t *testing.T) {
	data := ListTileThemeData{Dense: foundation.Some(true), HorizontalTitleGap: foundation.Some(40.0), TitleTextStyle: &painting.TextStyle{FontSize: foundation.Some(21.0)}}
	for _, name := range []string{"widget", "merge", "theme data"} {
		t.Run(name, func(t *testing.T) {
			th := standardTheme(true)
			tile := widgets.Widget(NewListTile(ListTileOpts{Leading: sizedBox(24, 24), Title: widgets.NewText("Title")}))
			switch name {
			case "widget":
				tile = NewListTileTheme(data, tile)
			case "merge":
				tile = ListTileThemeMerge(data, tile)
			default:
				th = th.CopyWith(ThemeDataCopy{ListTileTheme: foundation.Some(data)})
			}
			ts := pumpTile(t, th, widgets.NewSizedBox(tile, widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			if got := ts.Size(gluttertest.ByType[*ListTile]()).Height; got != 48 {
				t.Errorf("height %v, want 48 for a dense tile", got)
			}
			// title starts after 16 + max(24, 24) + 40.
			if got := ts.TopLeft(gluttertest.ByText("Title")).Dx - ts.TopLeft(gluttertest.ByType[*ListTile]()).Dx; got != 16+24+40 {
				t.Errorf("the title starts at %v, want 80", got)
			}
		})
	}
	t.Run("merge keeps the parent", func(t *testing.T) {
		parent := ListTileThemeData{Dense: foundation.Some(true)}
		tile := NewListTileTheme(parent, ListTileThemeMerge(ListTileThemeData{HorizontalTitleGap: foundation.Some(40.0)},
			NewListTile(ListTileOpts{Title: widgets.NewText("Title")})))
		ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(tile, widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		if got := ts.Size(gluttertest.ByType[*ListTile]()).Height; got != 48 {
			t.Errorf("height %v, want 48", got)
		}
	})
}

func TestListTileDivideTiles(t *testing.T) {
	tiles := []widgets.Widget{NewListTile(ListTileOpts{Title: widgets.NewText("A")}), NewListTile(ListTileOpts{Title: widgets.NewText("B")}),
		NewListTile(ListTileOpts{Title: widgets.NewText("C")})}
	if got := ListTileDivideTiles(nil, tiles[:1], foundation.Some(Colors.Red.Primary)); len(got) != 1 {
		t.Errorf("a single tile is left alone, got %d", len(got))
	}
	got := ListTileDivideTiles(nil, tiles, foundation.Some(Colors.Red.Primary))
	if len(got) != 3 {
		t.Fatalf("got %d widgets", len(got))
	}
	for i, w := range got {
		_, isBox := w.(*widgets.DecoratedBox)
		if want := i < 2; isBox != want {
			t.Errorf("widget %d: decorated = %v, want %v", i, isBox, want)
		}
	}
	ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(widgets.NewColumn(got, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
	if h := ts.Size(gluttertest.ByType[*widgets.Column]()).Height; h != 3*56 {
		t.Errorf("the dividers take no space: height %v", h)
	}
}

func TestListTileSemantics(t *testing.T) {
	skipUnlessDebug(t) // the semantics finders need a debug build
	ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(
		NewListTile(ListTileOpts{Title: widgets.NewText("Tappable"), OnTap: func() {}}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
	handle := ts.EnsureSemantics()
	defer handle.Dispose()
	ts.Pump()
	if !ts.Exists(gluttertest.BySemanticsLabel("Tappable")) {
		t.Error("the tile has no semantics label")
	}
}

func TestListTileLeadingConsumesWidth(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts ListTileOpts
		tag  string
	}{
		{"leading", ListTileOpts{Leading: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(1000.0)}), Title: widgets.NewText("x")}, "ListTileLeadingConsumesTileWidth"},
		{"trailing", ListTileOpts{Trailing: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(1000.0)}), Title: widgets.NewText("x")}, "ListTileTrailingConsumesTileWidth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := recoverError(func() {
				ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewListTile(tc.opts), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
				takeAndPanic(ts)
			})
			wantRoute(t, err, tc.tag)
		})
	}
}

func TestListTileBackgroundHidden(t *testing.T) {
	skipUnlessDebug(t)
	err := recoverError(func() {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewColoredBox(Colors.Red.Primary,
			NewListTile(ListTileOpts{Title: widgets.NewText("x"), OnTap: func() {}}))})}))
		takeAndPanic(ts)
	})
	wantRoute(t, err, "ListTileBackgroundHidden")
}
