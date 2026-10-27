// Package editorscene makes the scenes of a frame in which only the caret of a text editor blinks,
// for the allocation tests and benchmarks of the renderers (SPEC §15, step 29.15). The tree is
// the one of examples/editor (an EditableText in a bordered box, a read-only panel and a status
// bar) built from the public widgets, so that the renderers see what a real editor draws.
//
// It imports gluttertest, so only external test packages (package foo_test) may import it.
//
// Flutter: no counterpart
package editorscene

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The size of the window, in logical pixels, and the length of a caret blink.
const (
	Width  = 820
	Height = 460
	Blink  = 500 * time.Millisecond
)

// Text is a text of lines lines of about 60 characters.
//
// Flutter: no counterpart
func Text(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "Line %d: the quick brown fox jumps over the lazy dog, 0123456789\n", i)
	}
	return b.String()
}

func style(size float64, color ui.Color) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color),
	})
}

func box(color ui.Color, child widgets.Widget) widgets.Widget {
	return widgets.NewContainer(child, widgets.ContainerOpts{
		Padding: painting.NewEdgeInsetsAll(12),
		Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(color),
			BorderRadius: painting.NewBorderRadiusCircular(8),
			Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFFCAC4D0))}),
		}),
	})
}

// App is the editor app with the given text in the editor and in the saved panel.
//
// Flutter: no counterpart
func App(text string) widgets.Widget {
	ink := ui.NewColor(0xFF1C1B1F)
	controller := widgets.NewTextEditingController()
	controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text))),
	}))
	field := widgets.NewEditableText(controller, widgets.NewFocusNode(), style(16, ink), ui.NewColor(0xFF6750A4), ui.NewColor(0xFFCAC4D0),
		widgets.EditableTextOpts{
			Autofocus: true, Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited),
			SelectionColor:    foundation.Some(ui.NewColor(0x666750A4)),
			SelectionControls: widgets.NewEmptyTextSelectionControls(),
			KeyboardType:      foundation.Some(services.TextInputTypeMultiline),
			TextInputAction:   services.TextInputActionNewline,
		})
	panels := widgets.NewRow([]widgets.Widget{
		widgets.NewExpanded(box(ui.NewColor(0xFFFFFFFF), field), widgets.ExpandedOpts{Flex: foundation.Some(3)}),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
		widgets.NewExpanded(box(ui.NewColor(0xFFF3EDF7), widgets.NewSingleChildScrollView(
			widgets.NewText(text, widgets.TextOpts{Style: style(16, ink)}))), widgets.ExpandedOpts{Flex: foundation.Some(2)}),
	}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	status := widgets.NewColoredBox(ui.NewColor(0xFFE8DEF8), widgets.NewPadding(painting.NewEdgeInsetsSymmetric(8, 16),
		widgets.NewText("Ln 1, Col 1   ·   lines, words, characters", widgets.TextOpts{Style: style(13, ink)})))
	page := widgets.NewColoredBox(ui.NewColor(0xFFFFFBFE), widgets.NewColumn([]widgets.Widget{
		widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsAll(16), panels)),
		status,
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))
	return widgets.NewWidgetsApp(ui.NewColor(0xFFFFFBFE), widgets.WidgetsAppOpts{
		Title:                      "editor",
		DebugShowCheckedModeBanner: foundation.Some(false),
		TextStyle:                  style(16, ink),
		OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
			return widgets.NewPageRouteBuilder(
				func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
					return page
				},
				widgets.PageRouteBuilderOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
		},
	})
}

// Tester pumps the editor with the given number of lines until the caret blinks, as on the Linux
// desktop. discard drops the scenes the frames make; the caller keeps the ones it wants through
// Scenes.
//
// Flutter: no counterpart
func Tester(tb testing.TB, lines int, discard bool) *gluttertest.Tester {
	tb.Helper()
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	tb.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: Width, Height: Height}, DiscardScenes: discard})
	ts.PumpWidget(App(Text(lines)))
	ts.Pump(50 * time.Millisecond)
	// Two blinks, so that every cache the caret frame fills is full.
	ts.Pump(Blink)
	ts.Pump(Blink)
	return ts
}

// CaretScenes are two scenes of consecutive frames that differ in the caret only: the caret on,
// then off.
//
// Flutter: no counterpart
func CaretScenes(tb testing.TB, lines int) (on, off *ui.Scene) {
	tb.Helper()
	ts := Tester(tb, lines, false)
	last := func() *ui.Scene { s := ts.Scenes(); return s[len(s)-1] }
	n := len(ts.Scenes())
	ts.Pump(Blink)
	on = last()
	ts.Pump(Blink)
	off = last()
	if len(ts.Scenes()) != n+2 {
		tb.Fatalf("the caret blinks made %d frames, want 2", len(ts.Scenes())-n)
	}
	return on, off
}

// BytesPerFrame runs frame the given number of times and returns the bytes the process allocated
// for each (runtime.MemStats.TotalAlloc), for the allocation tests: nothing else may run while it
// measures. It collects garbage first, so that the count starts clean.
//
// Flutter: no counterpart
func BytesPerFrame(frames int, frame func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < frames; i++ {
		frame()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / uint64(frames)
}
