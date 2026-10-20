package rendering_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// These tests are ported from the Selection group of
// packages/flutter/test/rendering/paragraph_test.dart. The font of Flutter's
// tests has glyphs one em wide; the tests compare with the boxes the paragraph
// reports instead of constants. 'getPositionForOffset works' is not ported: it
// only measures the font.

var selectionColor = ui.NewColor(0xAF6694E8)

// selectionParagraph selects from one position to the other in every selectable
// of the paragraph's registrar.
func selectionParagraph(p *rendering.RenderParagraph, registrar *testRegistrar, start, end ui.TextPosition) {
	for _, s := range registrar.selectables {
		s.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForStart(p.GetOffsetForCaret(start, ui.Rect{}).Add(ui.Offset{Dy: 5}), rendering.TextGranularityUnspecified))
		s.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForEnd(p.GetOffsetForCaret(end, ui.Rect{}).Add(ui.Offset{Dy: 5}), rendering.TextGranularityUnspecified))
	}
}

// paintedOps paints the paragraph and lists what was drawn, in order: the names
// of the canvas operations, with "leader" for the leader layers.
func paragraphPaintOps(p *rendering.RenderParagraph) (names []string, rects []ui.OpDrawRect) {
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
	p.Paint(ctx, ui.OffsetZero)
	ctx.StopRecordingIfNeeded()
	for _, l := range root.DepthFirstIterateChildren() {
		switch layer := l.(type) {
		case *rendering.PictureLayer:
			if layer.Picture() == nil {
				continue
			}
			for _, op := range layer.Picture().DisplayList().Ops() {
				names = append(names, reflect.TypeOf(op).Name())
				if r, ok := op.(ui.OpDrawRect); ok {
					rects = append(rects, r)
				}
			}
		case *rendering.LeaderLayer:
			names = append(names, "leader")
		}
	}
	return names, rects
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}

func newSelectableParagraph(text string, opts ...rendering.RenderParagraphOpts) (*rendering.RenderParagraph, *testRegistrar) {
	registrar := &testRegistrar{}
	var o rendering.RenderParagraphOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Registrar = registrar
	return newParagraph(textSpan(text, 14), o), registrar
}

func TestParagraphPaintsTheSelectionHighlight(t *testing.T) {
	p, registrar := newSelectableParagraph("1234567", rendering.RenderParagraphOpts{SelectionColor: foundation.Some(selectionColor)})
	layout(t, newHarness(t), p)
	if _, rects := paragraphPaintOps(p); len(rects) != 0 {
		t.Fatalf("drawn rects without a selection = %v", rects)
	}
	for _, sel := range [][2]int{{1, 5}, {2, 4}} {
		selectionParagraph(p, registrar, ui.TextPosition{Offset: sel[0]}, ui.TextPosition{Offset: sel[1]})
		names, rects := paragraphPaintOps(p)
		boxes := p.GetBoxesForSelection(ui.TextRange{Start: sel[0], End: sel[1]})
		if len(rects) != 1 || len(boxes) != 1 || rects[0].Rect != boxes[0].ToRect() {
			t.Fatalf("selection %v: drawn %v, want the box %v", sel, rects, boxes)
		}
		if rects[0].Paint.Style() != ui.PaintingStyleFill || rects[0].Paint.Color() != selectionColor {
			t.Errorf("selection %v: paint %v %v", sel, rects[0].Paint.Style(), rects[0].Paint.Color())
		}
		// The highlight is painted before the text.
		if r, d := indexOf(names, "OpDrawRect"), indexOf(names, "OpDrawParagraph"); r < 0 || d < 0 || r > d {
			t.Errorf("selection %v: operations %v, want the rect before the paragraph", sel, names)
		}
	}
}

// Regression test for https://github.com/flutter/flutter/issues/182776.
func TestParagraphPaintsTheSelectionOutsideTheFadeLayerAndHandlesAfterTheText(t *testing.T) {
	p, registrar := newSelectableParagraph("a\na\na\na", rendering.RenderParagraphOpts{
		SelectionColor: foundation.Some(selectionColor), MaxLines: foundation.Some(3), Overflow: painting.TextOverflowFade})
	layout(t, newHarness(t), p, withConstraints(rendering.BoxConstraints{MaxWidth: 100}))
	if !p.DebugHasOverflowShader() {
		t.Fatal("the paragraph has no overflow shader")
	}
	for _, s := range registrar.selectables {
		s.DispatchSelectionEvent(rendering.SelectAllSelectionEvent{})
		s.PushHandleLayers(rendering.NewLayerLink(), rendering.NewLayerLink())
	}
	names, _ := paragraphPaintOps(p)
	saveLayer, paragraph, leader := indexOf(names, "OpSaveLayer"), indexOf(names, "OpDrawParagraph"), indexOf(names, "leader")
	if saveLayer < 0 || paragraph <= saveLayer {
		t.Fatalf("operations %v: want the text in the layer", names)
	}
	if leader <= paragraph {
		t.Errorf("operations %v: want the handles after the text", names)
	}
	if indexOf(names[:saveLayer], "OpDrawRect") < 0 {
		t.Errorf("operations %v: want the selection before the layer", names)
	}
	if indexOf(names[saveLayer:], "OpDrawRect") >= 0 && indexOf(names[saveLayer:], "OpDrawRect") < paragraph-saveLayer {
		t.Errorf("operations %v: the selection is inside the layer", names)
	}
}

// Regression test for https://github.com/flutter/flutter/issues/126652.
func TestParagraphPaintsTheSelectionWhenTappingAChineseCharacter(t *testing.T) {
	p, registrar := newSelectableParagraph("你好", rendering.RenderParagraphOpts{SelectionColor: foundation.Some(selectionColor)})
	layout(t, newHarness(t), p)
	if _, rects := paragraphPaintOps(p); len(rects) != 0 {
		t.Fatalf("drawn rects without a selection = %v", rects)
	}
	for _, s := range registrar.selectables {
		s.DispatchSelectionEvent(rendering.SelectWordSelectionEvent{GlobalPosition: ui.Offset{Dx: 7}})
	}
	_, rects := paragraphPaintOps(p)
	if len(rects) == 0 || rects[0].Rect.IsEmpty() || rects[0].Paint.Style() != ui.PaintingStyleFill || rects[0].Paint.Color() != selectionColor {
		t.Errorf("drawn rects = %+v", rects)
	}
}

func TestParagraphSelectAllWithAWidgetSpan(t *testing.T) {
	registrar := &testRegistrar{}
	text := painting.NewTextSpan(painting.TextSpanOpts{Style: roboto(14), Children: []painting.InlineSpan{
		painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("before the span")}), newPlaceholder(),
		painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("after the span")})}})
	boxes := []rendering.Box{newTBox(10, 10)}
	p := newParagraph(text, rendering.RenderParagraphOpts{Registrar: registrar, Children: boxes})
	applyParentData(boxes, text)
	layout(t, newHarness(t), p)
	// The widget span registers with the selection container without going
	// through the paragraph.
	if len(registrar.selectables) != 2 {
		t.Fatalf("selectables = %d, want 2", len(registrar.selectables))
	}
	for i, s := range registrar.selectables {
		s.DispatchSelectionEvent(rendering.SelectAllSelectionEvent{})
		if g := s.Value(); !g.HasContent || g.Status != rendering.SelectionStatusUncollapsed {
			t.Errorf("segment %d: geometry %+v", i, g)
		}
	}
	if sel := p.Selections(); len(sel) != 2 {
		t.Errorf("selections = %v, want one per segment", sel)
	}
}

func selectionOf(t *testing.T, p *rendering.RenderParagraph) services.TextSelection {
	t.Helper()
	sel := p.Selections()
	if len(sel) != 1 {
		t.Fatalf("selections = %v, want one", sel)
	}
	return sel[0]
}

func extend(s rendering.Selectable, forward bool, granularity rendering.TextGranularity) {
	s.DispatchSelectionEvent(rendering.GranularlyExtendSelectionEvent{Forward: forward, IsEnd: true, Granularity: granularity})
}

func TestParagraphCanGranularlyExtendTheSelection(t *testing.T) {
	const text = "how are you\nI am fine\nThank you"
	type step struct {
		forward     bool
		granularity rendering.TextGranularity
		start, end  int
	}
	tests := []struct {
		name     string
		from, to int
		steps    []step
	}{
		{"character", 4, 5, []step{
			{true, rendering.TextGranularityCharacter, 4, 6}, {false, rendering.TextGranularityCharacter, 4, 5}}},
		{"word", 4, 5, []step{
			{true, rendering.TextGranularityWord, 4, 7}, {false, rendering.TextGranularityWord, 4, 4}, {false, rendering.TextGranularityWord, 0, 4}}},
		{"line", 4, 5, []step{
			{true, rendering.TextGranularityLine, 4, 11}, {false, rendering.TextGranularityLine, 0, 4}}},
		{"document", 14, 15, []step{
			{true, rendering.TextGranularityDocument, 14, 31}, {false, rendering.TextGranularityDocument, 0, 14}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, registrar := newSelectableParagraph(text)
			layout(t, newHarness(t), p)
			if len(registrar.selectables) != 1 {
				t.Fatalf("selectables = %d", len(registrar.selectables))
			}
			selectionParagraph(p, registrar, ui.TextPosition{Offset: tc.from}, ui.TextPosition{Offset: tc.to})
			if sel := selectionOf(t, p); sel.Start() != tc.from || sel.End() != tc.to {
				t.Fatalf("selection = %v, want %d to %d", sel, tc.from, tc.to)
			}
			for i, s := range tc.steps {
				extend(registrar.selectables[0], s.forward, s.granularity)
				if sel := selectionOf(t, p); sel.Start() != s.start || sel.End() != s.end {
					t.Errorf("step %d: selection = %v, want %d to %d", i, sel, s.start, s.end)
				}
			}
		})
	}
}

func TestParagraphCanGranularlyExtendTheSelectionWithoutAnActiveSelection(t *testing.T) {
	p, registrar := newSelectableParagraph("how are you\nI am fine\nThank you")
	layout(t, newHarness(t), p)
	if len(p.Selections()) != 0 {
		t.Fatal("the paragraph has a selection")
	}
	extend(registrar.selectables[0], true, rendering.TextGranularityWord)
	if sel := selectionOf(t, p); sel.Start() != 0 || sel.End() != 3 {
		t.Errorf("selection = %v, want [how]", sel)
	}
	registrar.selectables[0].DispatchSelectionEvent(rendering.ClearSelectionEvent{})
	if len(p.Selections()) != 0 {
		t.Fatal("the selection was not cleared")
	}
	extend(registrar.selectables[0], false, rendering.TextGranularityWord)
	if sel := selectionOf(t, p); sel.Start() != 28 || sel.End() != 31 {
		t.Errorf("selection = %v, want [you] of the last line", sel)
	}
}

func TestParagraphCanDirectionallyExtendTheSelection(t *testing.T) {
	p, registrar := newSelectableParagraph("how are you\nI am fine\nThank you")
	layout(t, newHarness(t), p)
	s := registrar.selectables[0]
	selectionParagraph(p, registrar, ui.TextPosition{Offset: 14}, ui.TextPosition{Offset: 15})
	if sel := selectionOf(t, p); sel.Start() != 14 || sel.End() != 15 {
		t.Fatalf("selection = %v", sel)
	}
	baseline := s.GetTransformTo(nil).TransformPoint(s.Value().EndSelectionPoint.LocalPosition).Dx
	s.DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{IsEnd: true, Dx: baseline, Direction: rendering.SelectionExtendDirectionNextLine})
	// how are you
	// I [am fine
	// Tha]nk you: the third line, near the same horizontal position.
	if sel := selectionOf(t, p); sel.Start() != 14 || sel.End() < 22 || sel.End() > 31 {
		t.Errorf("after the next line: selection = %v, want it to end on the third line", sel)
	}
	s.DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{IsEnd: true, Dx: baseline, Direction: rendering.SelectionExtendDirectionPreviousLine})
	if sel := selectionOf(t, p); sel.Start() != 14 || sel.End() != 15 {
		t.Errorf("after the previous line: selection = %v, want [a]", sel)
	}
}

func TestParagraphCanDirectionallyExtendTheSelectionWithoutOne(t *testing.T) {
	p, registrar := newSelectableParagraph("how are you\nI am fine\nThank you")
	layout(t, newHarness(t), p)
	s := registrar.selectables[0]
	baseline := s.GetTransformTo(nil).TransformPoint(ui.Offset{Dx: s.Size().Width / 2}).Dx
	s.DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{IsEnd: true, Dx: baseline, Direction: rendering.SelectionExtendDirectionForward})
	// The start of the text to about the middle of the first line.
	if sel := selectionOf(t, p); sel.Start() != 0 || sel.End() < 1 || sel.End() > 11 {
		t.Errorf("selection = %v, want it from the start to the middle of the first line", sel)
	}
	s.DispatchSelectionEvent(rendering.ClearSelectionEvent{})
	if len(p.Selections()) != 0 {
		t.Fatal("the selection was not cleared")
	}
	s.DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{IsEnd: true, Dx: baseline, Direction: rendering.SelectionExtendDirectionBackward})
	// The end of the text to about the middle of the last line.
	if sel := selectionOf(t, p); sel.End() != 31 || sel.Start() < 22 || sel.Start() > 30 {
		t.Errorf("selection = %v, want it from the middle of the last line to the end", sel)
	}
}
