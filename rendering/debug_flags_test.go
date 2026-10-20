package rendering_test

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// debugHarness makes a harness with the flags on and the debug output collected.
func debugHarness(t *testing.T, set func(o *rendering.DebugOptions)) (*renderingtest.Harness, *[]string) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the debug flags have an effect only in debug builds")
	}
	var lines []string
	o := rendering.NewDebugOptions()
	o.Print = func(s string) { lines = append(lines, s) }
	set(o)
	return renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 100, Height: 100}, Debug: o}), &lines
}

func TestDebugPrintLayoutsPrintsEveryRenderObjectThatLaysOut(t *testing.T) {
	h, lines := debugHarness(t, func(o *rendering.DebugOptions) { o.PrintLayouts = true })
	box := newTBox(10, 10)
	h.SetRoot(box)
	h.Pump()
	found := false
	for _, l := range *lines {
		found = found || (strings.HasPrefix(l, "Laying out") && strings.Contains(l, "RenderView"))
	}
	if !found {
		t.Errorf("no line says that the view was laid out: %q", *lines)
	}
}

// render draws the root in a harness with the flags that set makes, and returns the
// harness's image after each of the repaints that run does.
func renderWith(t *testing.T, set func(o *rendering.DebugOptions), root func(h *renderingtest.Harness) rendering.Box, repaints int, mark func()) []*image.RGBA {
	t.Helper()
	h, _ := debugHarness(t, set)
	h.SetRoot(root(h))
	var out []*image.RGBA
	for i := 0; i < repaints; i++ {
		if mark != nil && i > 0 {
			mark()
		}
		h.Pump()
		img, err := h.Image()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, img)
	}
	return out
}

// changed lists the pixels that differ between two images.
func changed(a, b *image.RGBA) []image.Point {
	var out []image.Point
	for y := 0; y < a.Rect.Dy(); y++ {
		for x := 0; x < a.Rect.Dx(); x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				out = append(out, image.Pt(x, y))
			}
		}
	}
	return out
}

func decorated(argb uint32, child rendering.Box) *rendering.RenderDecoratedBox {
	return rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(argb))}), child)
}

func topLeft(child rendering.Box) rendering.Box {
	return rendering.NewRenderPositionedBox(child, rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopLeft})
}

func TestDebugPaintLayerBordersPaintsABorderAroundTheLayerOfARepaintBoundary(t *testing.T) {
	root := func(*renderingtest.Harness) rendering.Box {
		return topLeft(rendering.NewRenderRepaintBoundary(decorated(0xFF3366CC, newTBox(40, 40))))
	}
	off := renderWith(t, func(*rendering.DebugOptions) {}, root, 1, nil)[0]
	on := renderWith(t, func(o *rendering.DebugOptions) { o.PaintLayerBordersEnabled = true }, root, 1, nil)[0]
	diff := changed(off, on)
	if len(diff) == 0 {
		t.Fatal("the border was not painted")
	}
	// A line of one pixel along the edges of the layer, and nothing inside it.
	for _, p := range diff {
		if p.X > 0 && p.X < 39 && p.Y > 0 && p.Y < 39 || p.X > 40 || p.Y > 40 {
			t.Fatalf("pixel %v changed, which is not on the border of the layer", p)
		}
	}
	if on.RGBAAt(20, 0).R <= off.RGBAAt(20, 0).R {
		t.Errorf("the top edge is %v, expected the orange to add red to %v", on.RGBAAt(20, 0), off.RGBAAt(20, 0))
	}
}

func TestDebugRepaintRainbowChangesColorAtEachRepaint(t *testing.T) {
	var boundary *rendering.RenderRepaintBoundary
	root := func(*renderingtest.Harness) rendering.Box {
		boundary = rendering.NewRenderRepaintBoundary(decorated(0xFFFFFFFF, newTBox(40, 40)))
		return topLeft(boundary)
	}
	off := renderWith(t, func(*rendering.DebugOptions) {}, root, 1, nil)[0]
	on := renderWith(t, func(o *rendering.DebugOptions) { o.RepaintRainbowEnabled = true }, root, 3, func() { boundary.MarkNeedsPaint() })
	// The rainbow is a border of 6 pixels, 3 pixels inside the bounds of the layer: its
	// middle is the row 3.
	at := func(img *image.RGBA) color.RGBA { return img.RGBAAt(20, 3) }
	if at(on[0]) == at(off) {
		t.Fatalf("no rainbow was painted: %v", at(on[0]))
	}
	if at(on[0]) == at(on[1]) || at(on[1]) == at(on[2]) {
		t.Errorf("the rainbow color did not change between repaints: %v %v %v", at(on[0]), at(on[1]), at(on[2]))
	}
	if inner := on[0].RGBAAt(20, 20); inner != off.RGBAAt(20, 20) {
		t.Errorf("the rainbow reached the inside of the layer: %v", inner)
	}
}

func TestDebugPaintBaselinesPaintsTheBaselinesOfText(t *testing.T) {
	var engine ui.ParagraphFactory
	root := func(h *renderingtest.Harness) rendering.Box {
		engine = h.TextEngine
		text := painting.NewTextSpan(painting.TextSpanOpts{
			Text:  foundation.Some("Hxg"),
			Style: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(20.0), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		})
		return topLeft(rendering.NewRenderParagraph(text, ui.TextDirectionLTR, rendering.RenderParagraphOpts{TextEngine: engine}))
	}
	off := renderWith(t, func(*rendering.DebugOptions) {}, root, 1, nil)[0]
	on := renderWith(t, func(o *rendering.DebugOptions) { o.PaintBaselinesEnabled = true }, root, 1, nil)[0]
	diff := changed(off, on)
	if len(diff) < 20 {
		t.Fatalf("%d pixels changed, want a line of baseline across the text", len(diff))
	}
	// The baselines are lines: all the changed pixels are in a few rows.
	rows := map[int]bool{}
	for _, p := range diff {
		rows[p.Y] = true
	}
	if len(rows) > 6 {
		t.Errorf("the changed pixels are in %d rows, want the rows of a line or two", len(rows))
	}
}
