package rendering_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

var (
	blue = ui.NewColor(0xFF0000FF)
	grey = ui.NewColor(0xFF9E9E9E)
	red  = ui.NewColor(0xFFFF0000)
)

// fakeEditableTextState is _FakeEditableTextState of editable_test.dart.
type fakeEditableTextState struct {
	services.TextSelectionDelegateDefaults
	value     services.TextEditingValue
	selection foundation.Opt[services.TextSelection]
	causes    []services.SelectionChangedCause
}

func newFakeState() *fakeEditableTextState {
	return &fakeEditableTextState{value: services.TextEditingValueEmpty}
}

func (f *fakeEditableTextState) TextEditingValue() services.TextEditingValue { return f.value }
func (f *fakeEditableTextState) UserUpdateTextEditingValue(v services.TextEditingValue, cause services.SelectionChangedCause) {
	f.selection = foundation.Some(v.Selection)
	f.causes = append(f.causes, cause)
}
func (f *fakeEditableTextState) HideToolbar(bool)                             {}
func (f *fakeEditableTextState) BringIntoView(ui.TextPosition)                {}
func (f *fakeEditableTextState) CutSelection(services.SelectionChangedCause)  {}
func (f *fakeEditableTextState) PasteText(services.SelectionChangedCause)     {}
func (f *fakeEditableTextState) SelectAll(services.SelectionChangedCause)     {}
func (f *fakeEditableTextState) CopySelection(services.SelectionChangedCause) {}

func textStyle(size float64) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Height: foundation.Some(1.0)})
}

func editableSpan(text string, size float64) *painting.TextSpan {
	return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(text), Style: textStyle(size)})
}

// newEditable is RenderEditable with the arguments that the tests of Flutter
// always pass.
func newEditable(span painting.InlineSpan, opts ...rendering.RenderEditableOpts) *rendering.RenderEditable {
	var o rendering.RenderEditableOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.TextEngine = renderingtest.BundledTextEngine()
	o.Text = span
	return rendering.NewRenderEditable(ui.TextDirectionLTR, rendering.NewLayerLink(), rendering.NewLayerLink(),
		rendering.NewViewportOffsetZero(), newFakeState(), o)
}

func newEditableOffset(span painting.InlineSpan, offset rendering.ViewportOffset, delegate services.TextSelectionDelegate, o rendering.RenderEditableOpts) *rendering.RenderEditable {
	o.TextEngine = renderingtest.BundledTextEngine()
	o.Text = span
	return rendering.NewRenderEditable(ui.TextDirectionLTR, rendering.NewLayerLink(), rendering.NewLayerLink(), offset, delegate, o)
}

func collapsedAt(offset int) foundation.Opt[services.TextSelection] {
	return foundation.Some(services.NewTextSelectionCollapsed(offset))
}

// paintedOps paints the box, which was laid out with its compositing bits
// updated, into a fresh layer, and returns the drawing operations of every
// picture, in paint order.
func paintedOps(t *testing.T, box rendering.Box) []ui.DisplayOp {
	t.Helper()
	ops, _ := paintedOpsAndLayers(t, box)
	return ops
}

// clipLayers are the layers of the clips that the editable pushed.
func clipLayers(layers []rendering.Layer) []*rendering.ClipRectLayer {
	var out []*rendering.ClipRectLayer
	for _, l := range layers {
		if c, ok := l.(*rendering.ClipRectLayer); ok {
			out = append(out, c)
		}
	}
	return out
}

func paintedOpsAndLayers(t *testing.T, box rendering.Box) ([]ui.DisplayOp, []rendering.Layer) {
	t.Helper()
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
	box.Paint(ctx, ui.OffsetZero)
	ctx.StopRecordingIfNeeded()
	var ops []ui.DisplayOp
	for _, l := range root.DepthFirstIterateChildren() {
		if pl, ok := l.(*rendering.PictureLayer); ok && pl.Picture() != nil {
			ops = append(ops, pl.Picture().DisplayList().Ops()...)
		}
	}
	return ops, root.DepthFirstIterateChildren()
}

func rectOps(ops []ui.DisplayOp) []ui.OpDrawRect {
	var out []ui.OpDrawRect
	for _, op := range ops {
		if r, ok := op.(ui.OpDrawRect); ok {
			out = append(out, r)
		}
	}
	return out
}

func rrectOps(ops []ui.DisplayOp) []ui.OpDrawRRect {
	var out []ui.OpDrawRRect
	for _, op := range ops {
		if r, ok := op.(ui.OpDrawRRect); ok {
			out = append(out, r)
		}
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// lineHeight is the height of a line of Roboto at the size, with a line height
// of 1.
func lineHeight(t *testing.T, size float64) float64 {
	t.Helper()
	p := painting.NewTextPainter(painting.TextPainterOpts{TextEngine: renderingtest.BundledTextEngine(), Text: editableSpan("A", size), TextDirection: ui.TextDirectionLTR})
	p.Layout(0, math.Inf(1))
	return p.Height()
}

func textWidth(size float64, text string) float64 {
	p := painting.NewTextPainter(painting.TextPainterOpts{TextEngine: renderingtest.BundledTextEngine(), Text: editableSpan(text, size), TextDirection: ui.TextDirectionLTR})
	p.Layout(0, math.Inf(1))
	return p.Width()
}

func TestEditableRespectsClipBehavior(t *testing.T) {
	tests := []struct {
		name     string
		clip     ui.Clip
		wantClip bool
	}{
		{"default is hard edge", ui.ClipUnspecified, true},
		{"none", ui.ClipNone, false},
		{"hard edge", ui.ClipHardEdge, true},
		{"anti alias", ui.ClipAntiAlias, true},
		{"anti alias with save layer", ui.ClipAntiAliasWithSaveLayer, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			editable := newEditable(editableSpan(strings.Repeat("a", 10000), 14),
				rendering.RenderEditableOpts{Selection: collapsedAt(0), ClipBehavior: tc.clip})
			h := newHarness(t)
			layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 1, MaxHeight: 100}), phaseCompositingBits))
			// The painters are repaint boundaries, so the editable needs compositing and
			// clips with a layer.
			_, layers := paintedOpsAndLayers(t, editable)
			clips := clipLayers(layers)
			if (len(clips) > 0) != tc.wantClip {
				t.Fatalf("clip layers = %d, want clipping %v", len(clips), tc.wantClip)
			}
			want := tc.clip
			if want == ui.ClipUnspecified {
				want = ui.ClipHardEdge // by default
			}
			if tc.wantClip && clips[0].ClipBehavior() != want {
				t.Errorf("clip behavior = %v, want %v", clips[0].ClipBehavior(), want)
			}
		})
	}
}

func TestEditableReportsTheRealHeightWhenMaxLinesIsOne(t *testing.T) {
	tall := painting.NewTextSpan(painting.TextSpanOpts{
		Style: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(10.0), Height: foundation.Some(1.0)}),
		Children: []painting.InlineSpan{painting.NewTextSpan(painting.TextSpanOpts{
			Text: foundation.Some("TALL"), Style: textStyle(100)})},
	})
	editable := newEditable(tall)
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 600, MaxHeight: 600}))
	if got, want := editable.Size().Height, lineHeight(t, 100); !approx(got, want) {
		t.Errorf("height = %v, want the height of the tall line %v", got, want)
	}
}

func TestEditableReportsTheHeightOfTheFirstLineWhenMaxLinesIsOne(t *testing.T) {
	editable := newEditable(editableSpan(strings.Repeat("liiiiines\n", 10), 10))
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 600, MaxHeight: 600}))
	if got, want := editable.Size().Height, lineHeight(t, 10); !approx(got, want) {
		t.Errorf("height = %v, want the height of one line %v", got, want)
	}
}

func TestEditableDefaultSemanticsInputType(t *testing.T) {
	editable := newEditable(editableSpan("text", 14))
	config := semantics.NewSemanticsConfiguration()
	editable.DescribeSemanticsConfiguration(config)
	if config.InputType() != ui.SemanticsInputTypeText {
		t.Errorf("input type = %v, want text", config.InputType())
	}
}

func TestEditableDescribeApproximatePaintClipRespectsClipBehavior(t *testing.T) {
	editable := newEditable(editableSpan(strings.Repeat("a", 10000), 14),
		rendering.RenderEditableOpts{Selection: collapsedAt(0), ClipBehavior: ui.ClipNone})
	h := newHarness(t)
	layout(t, h, editable)
	visited := false
	editable.VisitChildren(func(child rendering.RenderObject) {
		visited = true
		if clip := editable.DescribeApproximatePaintClip(child); clip.IsSome() {
			t.Errorf("clip = %v, want none", clip)
		}
	})
	if !visited {
		t.Error("the editable has no children")
	}
	editable.SetClipBehavior(ui.ClipHardEdge)
	pumpFrame(h, phaseLayout)
	editable.VisitChildren(func(child rendering.RenderObject) {
		if clip := editable.DescribeApproximatePaintClip(child); !clip.IsSome() {
			t.Error("a text that overflows has no clip")
		}
	})
}

func TestEditablePaintRespectsTheOffsetArgument(t *testing.T) {
	const fontSize = 20.0
	editable := newEditable(editableSpan("text", fontSize), rendering.RenderEditableOpts{Selection: collapsedAt(0)})
	layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 1000, MaxHeight: 1000}), phaseCompositingBits))

	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 1000, 1000))
	paintOffset := ui.Offset{Dx: 100, Dy: 200}
	editable.Paint(ctx, paintOffset)
	ctx.StopRecordingIfNeeded()
	var leaders []*rendering.LeaderLayer
	for _, l := range root.DepthFirstIterateChildren() {
		if leader, ok := l.(*rendering.LeaderLayer); ok {
			leaders = append(leaders, leader)
		}
	}
	if len(leaders) != 2 {
		t.Fatalf("%d leader layers, want 2", len(leaders))
	}
	endpoint := editable.GetEndpointsForSelection(services.NewTextSelectionCollapsed(0))[0].Point.Add(paintOffset)
	for i, l := range leaders {
		if l.Offset() != endpoint {
			t.Errorf("leader %d offset = %v, want %v", i, l.Offset(), endpoint)
		}
	}
}

func TestEditableClipsWhenTheStartOfTheTextIsOffset(t *testing.T) {
	// The text fits, but its start is scrolled by 10 pixels.
	editable := newEditableOffset(editableSpan("A", 10), rendering.NewViewportOffsetFixed(10), newFakeState(),
		rendering.RenderEditableOpts{Selection: collapsedAt(0), Locale: foundation.Some(ui.NewLocale("en", "US"))})
	layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}), phaseCompositingBits))
	_, layers := paintedOpsAndLayers(t, editable)
	clips := clipLayers(layers)
	if len(clips) != 1 {
		t.Fatalf("%d clip layers, want 1", len(clips))
	}
	got := clips[0].ClipRect().Value()
	if got.Left != 0 || got.Top != 0 || got.Width() != editable.Size().Width || got.Height() != editable.Size().Height {
		t.Errorf("clip = %v, want the box of the editable %v", got, editable.Size())
	}
}

func TestEditableCanChangeCursorColorRadiusAndVisibility(t *testing.T) {
	showCursor := foundation.NewValueNotifier(true)
	editable := newEditable(editableSpan("test", 10), rendering.RenderEditableOpts{
		BackgroundCursorColor: foundation.Some(grey),
		CursorColor:           foundation.Some(red),
		Selection:             foundation.Some(services.NewTextSelectionCollapsed(4, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream})),
	})
	h := newHarness(t)
	layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseCompositingBits))
	if got := len(rectOps(paintedOps(t, editable))); got != 0 {
		t.Fatalf("%d rects painted by default, want no caret", got)
	}

	editable.SetShowCursor(showCursor)
	pumpFrame(h, phaseCompositingBits)
	width := textWidth(10, "test")
	rects := rectOps(paintedOps(t, editable))
	// The caret is on a physical pixel.
	if len(rects) != 1 || rects[0].Paint.Color() != red || math.Abs(rects[0].Rect.Left-width) > 0.5 || rects[0].Rect.Width() != 1 || rects[0].Rect.Height() != 10 {
		t.Fatalf("caret = %+v, want a red rect at x=%v of 1x10", rects, width)
	}

	// Now a rounded caret.
	editable.SetCursorColor(foundation.Some(blue))
	editable.SetCursorWidth(4)
	editable.SetCursorRadius(foundation.Some(ui.NewRadiusCircular(3)))
	pumpFrame(h, phaseCompositingBits)
	rrects := rrectOps(paintedOps(t, editable))
	if len(rrects) != 1 || rrects[0].Paint.Color() != blue {
		t.Fatalf("rounded caret = %+v", rrects)
	}
	if got := rrects[0].RRect.OuterRect(); math.Abs(got.Left-width) > 0.5 || got.Width() != 4 || got.Height() != 10 {
		t.Errorf("rounded caret = %v, want 4x10 at x=%v", got, width)
	}

	// The caret is much taller with a bigger text scale.
	editable.SetTextScaler(painting.NewTextScalerLinear(2))
	pumpFrame(h, phaseCompositingBits)
	rrects = rrectOps(paintedOps(t, editable))
	if len(rrects) != 1 || !approx(rrects[0].RRect.OuterRect().Height(), 20) {
		t.Errorf("caret at 2x = %+v, want a height of 20", rrects)
	}

	// The caret can be turned off.
	showCursor.SetValue(false)
	pumpFrame(h, phaseCompositingBits)
	if got := len(rrectOps(paintedOps(t, editable))); got != 0 {
		t.Errorf("%d rrects painted with the cursor off", got)
	}
}

func TestEditableCanChangeTextAlign(t *testing.T) {
	skipUnlessDebug(t)
	editable := newEditable(editableSpan("test", 14))
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	if editable.TextAlign() != ui.TextAlignStart {
		t.Errorf("text align = %v, want start", editable.TextAlign())
	}
	if editable.DebugNeedsLayout() {
		t.Error("needs layout after layout")
	}
	editable.SetTextAlign(ui.TextAlignCenter)
	if editable.TextAlign() != ui.TextAlignCenter || !editable.DebugNeedsLayout() {
		t.Errorf("text align = %v, needs layout = %v, want center and true", editable.TextAlign(), editable.DebugNeedsLayout())
	}
}

func TestEditableCanReadPlainText(t *testing.T) {
	editable := newEditable(painting.NewTextSpan(painting.TextSpanOpts{
		Text: foundation.Some("hello "), Style: textStyle(10),
		Children: []painting.InlineSpan{painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("world")})},
	}))
	if got := editable.PlainText(); got != "hello world" {
		t.Errorf("plain text = %q", got)
	}
}

func TestEditableCaretWithIdeographicScript(t *testing.T) {
	// Chinese characters are 3 bytes each, so offsets are bytes of UTF-8.
	text := "中文测试"
	editable := newEditable(editableSpan(text, 10), rendering.RenderEditableOpts{
		CursorColor: foundation.Some(red), ShowCursor: foundation.NewValueNotifier(true),
		Selection: collapsedAt(0),
	})
	h := newHarness(t)
	layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 100}), phaseCompositingBits))
	first := editable.GetLocalRectForCaret(ui.NewTextPosition(0))
	second := editable.GetLocalRectForCaret(ui.NewTextPosition(3))
	last := editable.GetLocalRectForCaret(ui.NewTextPosition(len(text)))
	if !(first.Left < second.Left && second.Left < last.Left) {
		t.Errorf("caret positions %v, %v, %v do not grow", first.Left, second.Left, last.Left)
	}
	if editable.GetLocalRectForCaret(ui.NewTextPosition(len(text))).Height() <= 0 {
		t.Error("the caret has no height")
	}
}

func TestEditableTextIsPaintedAboveSelection(t *testing.T) {
	editable := newEditable(editableSpan("text", 10), rendering.RenderEditableOpts{
		SelectionColor: foundation.Some(blue),
		Selection:      foundation.Some(services.NewTextSelection(0, 4)),
	})
	layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseCompositingBits))
	var order []string
	for _, op := range paintedOps(t, editable) {
		switch op.(type) {
		case ui.OpDrawRect:
			order = append(order, "rect")
		case ui.OpDrawParagraph:
			order = append(order, "text")
		}
	}
	if !reflect.DeepEqual(order, []string{"rect", "text"}) {
		t.Errorf("paint order = %v, want the selection below the text", order)
	}
}

func TestEditableCursorCanPaintAboveOrBelowTheText(t *testing.T) {
	tests := []struct {
		name  string
		above bool
		want  []string
	}{
		{"below the text", false, []string{"rect", "text"}},
		{"above the text", true, []string{"text", "rect"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			editable := newEditable(editableSpan("test", 10), rendering.RenderEditableOpts{
				CursorColor: foundation.Some(red), ShowCursor: foundation.NewValueNotifier(true),
				Selection: collapsedAt(4), PaintCursorAboveText: tc.above,
			})
			layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseCompositingBits))
			var order []string
			for _, op := range paintedOps(t, editable) {
				switch op.(type) {
				case ui.OpDrawRect:
					order = append(order, "rect")
				case ui.OpDrawParagraph:
					order = append(order, "text")
				}
			}
			if !reflect.DeepEqual(order, tc.want) {
				t.Errorf("paint order = %v, want %v", order, tc.want)
			}
		})
	}

	t.Run("the setter rebuilds the painters", func(t *testing.T) {
		editable := newEditable(editableSpan("test", 10), rendering.RenderEditableOpts{
			CursorColor: foundation.Some(red), ShowCursor: foundation.NewValueNotifier(true), Selection: collapsedAt(4),
		})
		h := newHarness(t)
		layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseCompositingBits))
		editable.SetPaintCursorAboveText(true)
		pumpFrame(h, phaseCompositingBits)
		var order []string
		for _, op := range paintedOps(t, editable) {
			switch op.(type) {
			case ui.OpDrawRect:
				order = append(order, "rect")
			case ui.OpDrawParagraph:
				order = append(order, "text")
			}
		}
		if !reflect.DeepEqual(order, []string{"text", "rect"}) {
			t.Errorf("paint order = %v", order)
		}
	})
}

func TestEditableDoesNotPaintTheCaretWithoutAValidSelection(t *testing.T) {
	tests := []struct {
		name      string
		selection foundation.Opt[services.TextSelection]
		wantRects int
	}{
		{"no selection", foundation.None[services.TextSelection](), 0},
		{"invalid selection", foundation.Some(services.NewTextSelectionCollapsed(-1)), 0},
		{"valid selection", collapsedAt(2), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			editable := newEditable(editableSpan("test", 10), rendering.RenderEditableOpts{
				CursorColor: foundation.Some(red), ShowCursor: foundation.NewValueNotifier(true), Selection: tc.selection,
			})
			layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseCompositingBits))
			if got := len(rectOps(paintedOps(t, editable))); got != tc.wantRects {
				t.Errorf("%d rects painted, want %d", got, tc.wantRects)
			}
		})
	}
}

// globalOffsetOf is the global position that is at the local offset of the box.
func globalOffsetOf(box rendering.Box, local ui.Offset) ui.Offset {
	return box.LocalToGlobal(local, nil)
}

func TestEditableSelectsTheCorrectPlaceWithOffsets(t *testing.T) {
	const text = "abc def ghi\njkl mno pqr"
	delegate := newFakeState()
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: text})
	editable := newEditableOffset(editableSpan(text, 20), rendering.NewViewportOffsetZero(), delegate, rendering.RenderEditableOpts{
		MaxLines: foundation.Some(rendering.MaxLinesUnlimited), Selection: collapsedAt(0),
	})
	h := newHarness(t)
	layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}), phaseCompositingBits))

	lineH := editable.PreferredLineHeight()
	tapAt := func(local ui.Offset) {
		editable.HandleTapDown(gestures.TapDownDetails{GlobalPosition: globalOffsetOf(editable, local)})
		editable.HandleTap()
	}
	tapAt(ui.Offset{Dx: 1, Dy: 1})
	if sel := delegate.selection.Value(); sel.BaseOffset != 0 || sel.ExtentOffset != 0 {
		t.Errorf("tap at the start: %v", sel)
	}
	tapAt(ui.Offset{Dx: 1, Dy: lineH + 1})
	if sel := delegate.selection.Value(); sel.BaseOffset != 12 || sel.ExtentOffset != 12 {
		t.Errorf("tap at the start of line 2: %v, want 12", sel)
	}
	tapAt(ui.Offset{Dx: 490, Dy: 1})
	if sel := delegate.selection.Value(); sel.BaseOffset != 11 {
		t.Errorf("tap after the end of line 1: %v, want 11", sel)
	}
	if delegate.causes[len(delegate.causes)-1] != services.SelectionChangedCauseTap {
		t.Errorf("cause = %v, want tap", delegate.causes[len(delegate.causes)-1])
	}

	// A long press selects the word.
	editable.HandleTapDown(gestures.TapDownDetails{GlobalPosition: globalOffsetOf(editable, ui.Offset{Dx: 1, Dy: 1})})
	editable.HandleLongPress()
	if sel := delegate.selection.Value(); sel.Start() != 0 || sel.End() != 3 {
		t.Errorf("long press: %v, want the word abc", sel)
	}
	if delegate.causes[len(delegate.causes)-1] != services.SelectionChangedCauseLongPress {
		t.Error("the cause of a long press is not longPress")
	}
	// A double tap selects the word too.
	editable.HandleDoubleTap()
	if delegate.causes[len(delegate.causes)-1] != services.SelectionChangedCauseDoubleTap {
		t.Error("the cause of a double tap is not doubleTap")
	}

	// The selection is cut to the length of the text of the field.
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: "ab"})
	tapAt(ui.Offset{Dx: 490, Dy: 1})
	if sel := delegate.selection.Value(); sel.BaseOffset != 2 {
		t.Errorf("tap with a shorter value: %v, want the offset cut to 2", sel)
	}
}

func TestEditableSelectsCorrectPlaceWhenOffsetsAreFlipped(t *testing.T) {
	const text = "abc def ghi"
	delegate := newFakeState()
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: text})
	editable := newEditableOffset(editableSpan(text, 20), rendering.NewViewportOffsetZero(), delegate, rendering.RenderEditableOpts{Selection: collapsedAt(0)})
	layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}), phaseCompositingBits))

	at := func(dx float64) ui.Offset { return globalOffsetOf(editable, ui.Offset{Dx: dx, Dy: 1}) }
	// Drag from the second word to the first.
	editable.SelectPositionAt(at(textWidth(20, "abc def")), foundation.Some(at(1)), services.SelectionChangedCauseDrag)
	sel := delegate.selection.Value()
	if sel.BaseOffset != 7 || sel.ExtentOffset != 0 {
		t.Errorf("drag backwards: %v, want base 7 and extent 0", sel)
	}
	// Words: from the third word back to the first.
	editable.SelectWordsInRange(at(textWidth(20, "abc def gh")), foundation.Some(at(1)), services.SelectionChangedCauseDrag)
	sel = delegate.selection.Value()
	if sel.BaseOffset != 11 || sel.ExtentOffset != 0 {
		t.Errorf("select words backwards: %v, want base 11 and extent 0", sel)
	}
	editable.SelectWordsInRange(at(1), foundation.Some(at(textWidth(20, "abc def gh"))), services.SelectionChangedCauseDrag)
	sel = delegate.selection.Value()
	if sel.BaseOffset != 0 || sel.ExtentOffset != 11 {
		t.Errorf("select words forwards: %v, want base 0 and extent 11", sel)
	}
}

func TestEditableSelectWordEdge(t *testing.T) {
	const text = "abc def"
	delegate := newFakeState()
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: text})
	editable := newEditableOffset(editableSpan(text, 20), rendering.NewViewportOffsetZero(), delegate, rendering.RenderEditableOpts{Selection: collapsedAt(0)})
	layout(t, newHarness(t), editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}), phaseCompositingBits))

	// Near the start of the word, the caret goes to its start; near the end, to its end.
	editable.HandleTapDown(gestures.TapDownDetails{GlobalPosition: globalOffsetOf(editable, ui.Offset{Dx: textWidth(20, "abc d") + 1, Dy: 1})})
	editable.SelectWordEdge(services.SelectionChangedCauseTap)
	if sel := delegate.selection.Value(); sel.BaseOffset != 7 || sel.ExtentOffset != 7 {
		t.Errorf("selection = %v, want a caret at the end of def", sel)
	}
	editable.HandleTapDown(gestures.TapDownDetails{GlobalPosition: globalOffsetOf(editable, ui.Offset{Dx: 1, Dy: 1})})
	editable.SelectWordEdge(services.SelectionChangedCauseTap)
	if sel := delegate.selection.Value(); sel.BaseOffset != 0 || sel.ExtentOffset != 0 {
		t.Errorf("selection = %v, want a caret at the start of abc", sel)
	}
}

func TestEditableGetWordAtOffset(t *testing.T) {
	editable := newEditable(editableSpan("abc def", 20))
	layout(t, newHarness(t), editable)
	tests := []struct {
		name     string
		position ui.TextPosition
		want     [2]int
	}{
		{"in a word", ui.NewTextPosition(5), [2]int{4, 7}},
		{"at the end of the text", ui.NewTextPosition(7), [2]int{7, 7}},
		{"past the end", ui.NewTextPosition(30), [2]int{7, 7}},
		{"on the space", ui.NewTextPosition(3), [2]int{3, 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sel := editable.GetWordAtOffset(tc.position)
			if got := [2]int{sel.BaseOffset, sel.ExtentOffset}; got != tc.want {
				t.Errorf("word = %v, want %v", got, tc.want)
			}
		})
	}
	editable.SetObscureText(true)
	if sel := editable.GetWordAtOffset(ui.NewTextPosition(2)); sel.Start() != 0 || sel.End() != 7 {
		t.Errorf("obscured word = %v, want all the text", sel)
	}

	t.Run("a negative effective offset is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		requireTag(t, "EditableWordOffsetNegative", func() {
			editable.SetObscureText(false)
			editable.GetWordAtOffset(ui.TextPosition{Offset: 0, Affinity: ui.TextAffinityUpstream})
		})
	})
}

func TestEditableMaxScrollExtent(t *testing.T) {
	tests := []struct {
		name      string
		maxLines  foundation.Opt[int]
		text      string
		width     float64
		wantExtra func(e *rendering.RenderEditable) float64
	}{
		{"one line scrolls horizontally", foundation.Some(1), "hello hello hello hello hello", 50,
			func(e *rendering.RenderEditable) float64 {
				return textWidth(14, "hello hello hello hello hello") + 2 - 50
			}},
		{"a short line does not scroll", foundation.Some(1), "hi", 500, func(*rendering.RenderEditable) float64 { return 0 }},
		{"several lines scroll vertically (the last line break starts a line)", foundation.Some(2), strings.Repeat("hello hello\n", 6), 500,
			func(e *rendering.RenderEditable) float64 {
				return 7*e.PreferredLineHeight() - 2*e.PreferredLineHeight()
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			editable := newEditable(editableSpan(tc.text, 14), rendering.RenderEditableOpts{MaxLines: tc.maxLines})
			layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: tc.width, MaxHeight: 1000}))
			if got, want := editable.MaxScrollExtent(), tc.wantExtra(editable); !approx(got, want) {
				t.Errorf("max scroll extent = %v, want %v", got, want)
			}
		})
	}
}

func TestEditableHasFocusIsInitialized(t *testing.T) {
	if newEditable(editableSpan("a", 10)).HasFocus() {
		t.Error("an editable has the focus at first")
	}
	editable := newEditable(editableSpan("a", 10), rendering.RenderEditableOpts{HasFocus: true})
	if !editable.HasFocus() {
		t.Error("HasFocus: true was ignored")
	}
	config := semantics.NewSemanticsConfiguration()
	editable.DescribeSemanticsConfiguration(config)
	if got, ok := config.IsFocused().Get(); !ok || !got {
		t.Error("the semantics say the editable has no focus")
	}
}

func TestEditableGetEndpointsForSelection(t *testing.T) {
	editable := newEditable(editableSpan("Hello world", 20), rendering.RenderEditableOpts{Selection: collapsedAt(0)})
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	h := editable.PreferredLineHeight()

	t.Run("a caret has one point", func(t *testing.T) {
		points := editable.GetEndpointsForSelection(services.NewTextSelectionCollapsed(0))
		if len(points) != 1 || !points[0].Direction.IsNone() || points[0].Point.Dx != 0 || math.Abs(points[0].Point.Dy-h) > 5 {
			t.Errorf("points = %v, want one at the bottom of the line (0, ~%v)", points, h)
		}
	})
	t.Run("a selection has two", func(t *testing.T) {
		editable.SetSelectionWidthStyle(ui.BoxWidthStyleTight)
		defer editable.SetSelectionWidthStyle(ui.BoxWidthStyleMax)
		points := editable.GetEndpointsForSelection(services.NewTextSelection(0, 5))
		if len(points) != 2 {
			t.Fatalf("points = %v", points)
		}
		if points[0].Point.Dx != 0 || !approx(points[1].Point.Dx, textWidth(20, "Hello")) || !approx(points[1].Point.Dy, h) {
			t.Errorf("points = %v", points)
		}
		if d, _ := points[0].Direction.Get(); d != ui.TextDirectionLTR {
			t.Errorf("direction = %v", d)
		}
	})
	t.Run("empty characters have a point", func(t *testing.T) {
		empty := newEditable(editableSpan("\n", 20))
		layout(t, newHarness(t), empty, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
		if points := empty.GetEndpointsForSelection(services.NewTextSelection(0, 1)); len(points) == 0 {
			t.Error("no points for a selection of a line break")
		}
	})
}

func TestTextSelectionPointCompares(t *testing.T) {
	a := rendering.TextSelectionPoint{Point: ui.Offset{Dx: 1, Dy: 2}, Direction: foundation.Some(ui.TextDirectionLTR)}
	b := rendering.TextSelectionPoint{Point: ui.Offset{Dx: 1, Dy: 2}, Direction: foundation.Some(ui.TextDirectionLTR)}
	if a != b {
		t.Error("equal points differ")
	}
	if a == (rendering.TextSelectionPoint{Point: ui.Offset{Dx: 1, Dy: 2}, Direction: foundation.Some(ui.TextDirectionRTL)}) {
		t.Error("different directions are equal")
	}
	for point, want := range map[rendering.TextSelectionPoint]string{
		a: "Offset(1.0, 2.0)-ltr",
		{Point: ui.Offset{}, Direction: foundation.Some(ui.TextDirectionRTL)}: "Offset(0.0, 0.0)-rtl",
		{Point: ui.Offset{}}: "Offset(0.0, 0.0)",
	} {
		if got := point.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	}
}

func TestEditableGetRectForComposingRange(t *testing.T) {
	editable := newEditable(editableSpan("abc def", 20))
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	if _, ok := editable.GetRectForComposingRange(ui.TextRangeEmpty); ok {
		t.Error("a rect for no composing range")
	}
	if _, ok := editable.GetRectForComposingRange(ui.NewTextRange(2, 2)); ok {
		t.Error("a rect for a collapsed range")
	}
	editable.SetSelectionWidthStyle(ui.BoxWidthStyleTight)
	rect, ok := editable.GetRectForComposingRange(ui.NewTextRange(0, 3))
	if !ok || !approx(rect.Left, 0) || !approx(rect.Right, textWidth(20, "abc")) || rect.Height() <= 0 {
		t.Errorf("rect = %v, %v", rect, ok)
	}
	// The rectangle moves with the scroll.
	scrolled := newEditableOffset(editableSpan("abc def", 20), rendering.NewViewportOffsetFixed(5), newFakeState(), rendering.RenderEditableOpts{})
	layout(t, newHarness(t), scrolled, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	moved, _ := scrolled.GetRectForComposingRange(ui.NewTextRange(0, 3))
	if !approx(moved.Left, -5) {
		t.Errorf("scrolled rect = %v, want it moved by -5", moved)
	}

	t.Run("more than one run on a line", func(t *testing.T) {
		mixed := newEditable(painting.NewTextSpan(painting.TextSpanOpts{
			Text: foundation.Some("abc"), Style: textStyle(20),
			Children: []painting.InlineSpan{painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("def"), Style: textStyle(40)})},
		}))
		layout(t, newHarness(t), mixed, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
		r, ok := mixed.GetRectForComposingRange(ui.NewTextRange(0, 6))
		if !ok || r.Height() < lineHeight(t, 40)-1 {
			t.Errorf("rect = %v, want one that holds both runs", r)
		}
	})
}

// recordingPainter is a RenderEditablePainter that records its calls.
type recordingPainter struct {
	foundation.ChangeNotifier
	name    string
	calls   *[]string
	repaint bool
	canvas  *ui.Canvas
	size    ui.Size
}

func (p *recordingPainter) ShouldRepaint(old rendering.RenderEditablePainter) bool { return p.repaint }
func (p *recordingPainter) Paint(canvas *ui.Canvas, size ui.Size, _ *rendering.RenderEditable) {
	*p.calls = append(*p.calls, p.name)
	p.canvas, p.size = canvas, size
}

func TestEditableCustomPainters(t *testing.T) {
	var calls []string
	background := &recordingPainter{name: "background", calls: &calls, repaint: true}
	foreground := &recordingPainter{name: "foreground", calls: &calls, repaint: true}
	editable := newEditable(editableSpan("text", 20), rendering.RenderEditableOpts{Painter: background, ForegroundPainter: foreground})
	h := newHarness(t)
	layout(t, h, editable, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}), phaseCompositingBits))
	paintedOps(t, editable)
	if !reflect.DeepEqual(calls, []string{"background", "foreground"}) {
		t.Errorf("painters were called in the order %v", calls)
	}
	// The painters paint in the viewport of the text, which is as wide as the
	// editable (the line is forced) and as tall as the text.
	if want := (ui.Size{Width: 500, Height: lineHeight(t, 20)}); !approx(background.size.Width, want.Width) || !approx(background.size.Height, want.Height) {
		t.Errorf("painter size = %v, want %v", background.size, want)
	}

	t.Run("changing the painters", func(t *testing.T) {
		calls = nil
		other := &recordingPainter{name: "other", calls: &calls, repaint: true}
		editable.SetPainter(other)
		editable.SetForegroundPainter(nil)
		pumpFrame(h, phaseCompositingBits)
		paintedOps(t, editable)
		if !reflect.DeepEqual(calls, []string{"other"}) {
			t.Errorf("painters were called in the order %v, want only the new background painter", calls)
		}
		if editable.Painter() != rendering.RenderEditablePainter(other) || editable.ForegroundPainter() != nil {
			t.Error("the getters do not give the new painters")
		}
	})

	t.Run("a painter that notifies repaints only its layer", func(t *testing.T) {
		calls = nil
		again := &recordingPainter{name: "again", calls: &calls, repaint: true}
		editable.SetPainter(again)
		pumpFrame(h, phasePaint)
		editable.Painter().(*recordingPainter).NotifyListeners()
		if !editable.Attached() {
			t.Fatal("not attached")
		}
	})

	t.Run("a painter that does not need a repaint leaves the layer clean", func(t *testing.T) {
		quiet := &recordingPainter{name: "quiet", calls: &calls, repaint: false}
		editable.SetPainter(quiet)
		if quiet.HasListeners() == false {
			t.Error("the editable does not listen to the painter")
		}
	})
}

func TestEditableLastTapDownRequiredBeforeSelecting(t *testing.T) {
	skipUnlessDebug(t)
	editable := newEditable(editableSpan("text", 20))
	layout(t, newHarness(t), editable)
	requireTag(t, "EditableNoTapDown", func() { editable.HandleTap() })
}

func TestEditableConstructorAsserts(t *testing.T) {
	skipUnlessDebug(t)
	span := editableSpan("a", 10)
	tests := []struct {
		tag  string
		opts rendering.RenderEditableOpts
	}{
		{"EditableMaxLinesInvalid", rendering.RenderEditableOpts{MaxLines: foundation.Some(-1)}},
		{"EditableMinLinesInvalid", rendering.RenderEditableOpts{MinLines: foundation.Some(-1), MaxLines: foundation.Some(rendering.MaxLinesUnlimited)}},
		{"EditableMinLinesGreaterThanMaxLines", rendering.RenderEditableOpts{MinLines: foundation.Some(3), MaxLines: foundation.Some(2)}},
		{"EditableExpandsWithLines", rendering.RenderEditableOpts{Expands: true}},
		{"EditableObscuringCharacterInvalid", rendering.RenderEditableOpts{ObscuringCharacter: "ab"}},
		{"EditableCursorWidthInvalid", rendering.RenderEditableOpts{CursorWidth: foundation.Some(-1.0)}},
		{"EditableCursorHeightInvalid", rendering.RenderEditableOpts{CursorHeight: foundation.Some(-1.0)}},
		{"EditableCursorColorMissing", rendering.RenderEditableOpts{ShowCursor: foundation.NewValueNotifier(true)}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			requireTag(t, tc.tag, func() { newEditable(span, tc.opts) })
		})
	}
	t.Run("setters", func(t *testing.T) {
		editable := newEditable(span)
		requireTag(t, "EditableMaxLinesInvalid", func() { editable.SetMaxLines(foundation.Some(0)) })
		requireTag(t, "EditableMinLinesInvalid", func() { editable.SetMinLines(foundation.Some(0)) })
		requireTag(t, "EditableObscuringCharacterInvalid", func() { editable.SetObscuringCharacter("") })
	})
	// The tags are in the errors of the panics.
	var tagged bool
	if v := panicOf(func() { newEditable(span, rendering.RenderEditableOpts{CursorWidth: foundation.Some(-1.0)}) }); v != nil {
		if err, ok := v.(error); ok {
			if appErr, ok := errors.FromError(err); ok && appErr.Tag == "EditableCursorWidthInvalid" {
				tagged = true
			}
		}
	}
	if !tagged {
		t.Error("the panic is not an error with the tag")
	}
}

func TestEditableMultilineHeights(t *testing.T) {
	text := "one\ntwo\nthree\nfour\nfive"
	lineH := lineHeight(t, 14)
	tests := []struct {
		name  string
		opts  rendering.RenderEditableOpts
		lines float64
	}{
		{"unlimited lines fit the text", rendering.RenderEditableOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited)}, 5},
		{"max lines cap the height", rendering.RenderEditableOpts{MaxLines: foundation.Some(3)}, 3},
		{"min lines raise it", rendering.RenderEditableOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited), MinLines: foundation.Some(8)}, 8},
		{"min and max lines", rendering.RenderEditableOpts{MaxLines: foundation.Some(7), MinLines: foundation.Some(6)}, 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			editable := newEditable(editableSpan(text, 14), tc.opts)
			layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 300, MaxHeight: 1000}))
			if got, want := editable.Size().Height, tc.lines*lineH; !approx(got, want) {
				t.Errorf("height = %v, want %v lines (%v)", got, tc.lines, want)
			}
		})
	}
	t.Run("expands fills the constraints", func(t *testing.T) {
		editable := newEditable(editableSpan(text, 14), rendering.RenderEditableOpts{Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited)})
		_ = editable
	})
}

func TestEditableIntrinsics(t *testing.T) {
	const text = "Hello World"
	editable := newEditable(editableSpan(text, 14))
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	caret := editable.CursorWidth() + 1
	if got, want := editable.GetMaxIntrinsicWidth(math.Inf(1)), textWidth(14, text)+caret; !approx(got, want) {
		t.Errorf("max intrinsic width = %v, want the text and the caret %v", got, want)
	}
	if got := editable.GetMinIntrinsicWidth(math.Inf(1)); got < textWidth(14, "World")-1 || got > textWidth(14, text) {
		t.Errorf("min intrinsic width = %v", got)
	}
	if got, want := editable.GetMinIntrinsicHeight(100), lineHeight(t, 14); !approx(got, want) {
		t.Errorf("min intrinsic height = %v, want %v", got, want)
	}
	multi := newEditable(editableSpan(text, 14), rendering.RenderEditableOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited)})
	layout(t, newHarness(t), multi, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	if got, want := multi.GetMaxIntrinsicHeight(textWidth(14, "Hello Wor")), 2*lineHeight(t, 14); !approx(got, want) {
		t.Errorf("height at a narrow width = %v, want two lines %v", got, want)
	}
	if got, want := multi.GetMaxIntrinsicHeight(math.Inf(1)), lineHeight(t, 14); !approx(got, want) {
		t.Errorf("height at infinite width = %v, want one line %v", got, want)
	}
}

func TestEditableDryLayoutAndBaseline(t *testing.T) {
	editable := newEditable(editableSpan("Hello", 14), rendering.RenderEditableOpts{DisableForceLine: true})
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	dry := editable.GetDryLayout(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500})
	if dry != editable.Size() {
		t.Errorf("dry layout = %v, layout = %v", dry, editable.Size())
	}
	if want := textWidth(14, "Hello") + 2; !approx(dry.Width, want) {
		t.Errorf("width without force line = %v, want the text and the caret %v", dry.Width, want)
	}
	dryBaseline := editable.GetDryBaseline(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}, ui.TextBaselineAlphabetic)
	if !dryBaseline.IsSome() || dryBaseline.Value() <= 0 || dryBaseline.Value() >= editable.Size().Height {
		t.Errorf("dry baseline = %v, want one inside the box of height %v", dryBaseline, editable.Size().Height)
	}
}

func TestEditableVerticalCaretMovement(t *testing.T) {
	editable := newEditable(editableSpan("one two three\nfour\nfive six", 14), rendering.RenderEditableOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited)})
	harness := newHarness(t)
	layout(t, harness, editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	run := editable.StartVerticalCaretMovement(ui.NewTextPosition(8)) // "thr|ee"
	if !run.IsValid() || run.Current().Offset != 8 {
		t.Fatalf("run = %v at %v", run.IsValid(), run.Current())
	}
	if !run.MoveNext() {
		t.Fatal("no next line")
	}
	// The second line, "four", is shorter than the column: the caret goes to its end.
	if got := run.Current().Offset; got != 18 {
		t.Errorf("caret on line 2 = %d, want the end of 'four' (18)", got)
	}
	if !run.MoveNext() || run.MoveNext() {
		t.Error("expected exactly one more line")
	}
	// Coming back remembers the column of the start.
	if !run.MovePrevious() {
		t.Fatal("cannot go back up")
	}
	if !run.MovePrevious() {
		t.Fatal("cannot go back up to the first line")
	}
	if got := run.Current().Offset; got != 8 {
		t.Errorf("caret back on line 1 = %d, want 8", got)
	}
	if run.MovePrevious() {
		t.Error("moved above the first line")
	}
	if !run.MoveByOffset(2 * editable.PreferredLineHeight()) {
		t.Error("MoveByOffset did not move")
	}
	if run.MoveByOffset(1000) && run.Current().Offset < 19 {
		t.Errorf("MoveByOffset(1000) = %d, want the last line", run.Current().Offset)
	}
	if run.MoveByOffset(1000) {
		t.Error("moved past the last line")
	}
	if !run.MoveByOffset(-1000) {
		t.Error("did not move up")
	}

	t.Run("a run is invalid after the layout changes", func(t *testing.T) {
		run := editable.StartVerticalCaretMovement(ui.NewTextPosition(0))
		editable.SetText(editableSpan("different\ntext", 14))
		pumpFrame(harness, phaseLayout)
		if run.IsValid() {
			t.Error("the run is valid after the text changed")
		}
		skipUnlessDebug(t)
		requireTag(t, "VerticalCaretMovementRunInvalid", func() { run.MoveNext() })
	})
}

func TestEditableTextLayoutMetrics(t *testing.T) {
	editable := newEditable(editableSpan("one two\nthree four", 14), rendering.RenderEditableOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited)})
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	var metrics services.TextLayoutMetrics = editable
	if line := metrics.GetLineAtOffset(ui.NewTextPosition(2)); line.Start() != 0 || line.End() != 7 {
		t.Errorf("line at 2 = %v, want the first line without its line break", line)
	}
	if word := metrics.GetWordBoundary(ui.NewTextPosition(9)); word.Start != 8 || word.End != 13 {
		t.Errorf("word at 9 = %v", word)
	}
	if below := metrics.GetTextPositionBelow(ui.NewTextPosition(1)); below.Offset < 8 {
		t.Errorf("below 1 = %v, want a position on the second line", below)
	}
	if above := metrics.GetTextPositionAbove(ui.NewTextPosition(10)); above.Offset > 7 {
		t.Errorf("above 10 = %v, want a position on the first line", above)
	}
	editable.SetObscureText(true)
	if line := metrics.GetLineAtOffset(ui.NewTextPosition(2)); line.Start() != 0 || line.End() != len(editable.PlainText()) {
		t.Errorf("obscured line = %v, want all the text", line)
	}
}

func TestEditableSemantics(t *testing.T) {
	delegate := newFakeState()
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: "one two three"})
	editable := newEditableOffset(editableSpan("one two three", 14), rendering.NewViewportOffsetZero(), delegate, rendering.RenderEditableOpts{
		HasFocus: true, Selection: collapsedAt(4),
	})
	layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
	config := semantics.NewSemanticsConfiguration()
	editable.DescribeSemanticsConfiguration(config)
	if got := config.AttributedValue().String; got != "one two three" {
		t.Errorf("value = %q", got)
	}
	if !config.IsTextField() || !config.IsFocusable() || config.IsReadOnly() || config.IsObscured() || config.IsMultiline() {
		t.Errorf("flags: textField %v focusable %v readOnly %v obscured %v multiline %v",
			config.IsTextField(), config.IsFocusable(), config.IsReadOnly(), config.IsObscured(), config.IsMultiline())
	}
	if sel := config.TextSelection().Value(); sel.BaseOffset != 4 || sel.ExtentOffset != 4 {
		t.Errorf("selection = %v", sel)
	}
	if config.OnSetSelection() == nil || config.OnSetText() == nil {
		t.Fatal("a focused field handles setting the selection and the text")
	}
	if config.OnMoveCursorForwardByWord() == nil || config.OnMoveCursorBackwardByCharacter() == nil {
		t.Fatal("the cursor can move")
	}

	config.OnSetText()("hello")
	if sel := delegate.selection.Value(); sel.BaseOffset != 5 || sel.ExtentOffset != 5 {
		t.Errorf("selection after SetText = %v", sel)
	}
	config.OnSetSelection()(services.NewTextSelection(1, 3))
	if sel := delegate.selection.Value(); sel.BaseOffset != 1 || sel.ExtentOffset != 3 {
		t.Errorf("selection after SetSelection = %v", sel)
	}
	if delegate.causes[len(delegate.causes)-1] != services.SelectionChangedCauseKeyboard {
		t.Error("the cause of a semantics action is keyboard")
	}
	config.OnMoveCursorForwardByCharacter()(false)
	if sel := delegate.selection.Value(); sel.BaseOffset != 5 || sel.ExtentOffset != 5 {
		t.Errorf("after forward by character = %v, want a caret at 5", sel)
	}
	config.OnMoveCursorBackwardByCharacter()(true)
	if sel := delegate.selection.Value(); sel.BaseOffset != 4 || sel.ExtentOffset != 3 {
		t.Errorf("after backward by character extending = %v, want base 4 extent 3", sel)
	}
	config.OnMoveCursorForwardByWord()(false)
	if sel := delegate.selection.Value(); sel.BaseOffset != 8 || sel.ExtentOffset != 8 {
		t.Errorf("after forward by word = %v, want a caret at the start of 'three' (8)", sel)
	}
	config.OnMoveCursorBackwardByWord()(false)
	if sel := delegate.selection.Value(); sel.BaseOffset != 0 || sel.ExtentOffset != 0 {
		t.Errorf("after backward by word = %v, want a caret at the start of 'one' (the editable's selection is still the caret at 4)", sel)
	}

	t.Run("obscured text is replaced", func(t *testing.T) {
		secret := newEditable(editableSpan("abc", 14), rendering.RenderEditableOpts{ObscureText: true, ObscuringCharacter: "*"})
		cfg := semantics.NewSemanticsConfiguration()
		secret.DescribeSemanticsConfiguration(cfg)
		if got := cfg.AttributedValue().String; got != "***" {
			t.Errorf("value = %q, want ***", got)
		}
		if !cfg.IsObscured() {
			t.Error("not obscured")
		}
	})
	t.Run("a field without the focus handles no setters", func(t *testing.T) {
		unfocused := newEditable(editableSpan("abc", 14), rendering.RenderEditableOpts{Selection: collapsedAt(1)})
		cfg := semantics.NewSemanticsConfiguration()
		unfocused.DescribeSemanticsConfiguration(cfg)
		if cfg.OnSetSelection() != nil || cfg.OnSetText() != nil {
			t.Error("an unfocused field handles setting the selection or the text")
		}
	})
	t.Run("a read only field handles no SetText", func(t *testing.T) {
		readOnly := newEditable(editableSpan("abc", 14), rendering.RenderEditableOpts{HasFocus: true, ReadOnly: true})
		cfg := semantics.NewSemanticsConfiguration()
		readOnly.DescribeSemanticsConfiguration(cfg)
		if cfg.OnSetText() != nil || !cfg.IsReadOnly() {
			t.Error("a read only field handles SetText")
		}
	})
}

func TestEditableDisposeReleasesTheChildren(t *testing.T) {
	editable := newEditable(editableSpan("abc", 14), rendering.RenderEditableOpts{})
	layout(t, newHarness(t), editable)
	editable.Dispose()
}

func TestEditableSemanticsRecognizerFailures(t *testing.T) {
	skipUnlessDebug(t)
	spanWith := func(rec gestures.GestureRecognizer) *painting.TextSpan {
		return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("link"), Style: textStyle(14), Recognizer: rec})
	}
	describe := func(editable *rendering.RenderEditable) func() {
		return func() { editable.DescribeSemanticsConfiguration(semantics.NewSemanticsConfiguration()) }
	}
	tap := gestures.NewTapGestureRecognizer(nil)
	defer tap.Dispose()
	t.Run("EditableRecognizerNotReadOnly", func(t *testing.T) {
		requireTag(t, "EditableRecognizerNotReadOnly", describe(newEditable(spanWith(tap), rendering.RenderEditableOpts{})))
		requireTag(t, "EditableRecognizerNotReadOnly", describe(newEditable(spanWith(tap), rendering.RenderEditableOpts{ReadOnly: true, ObscureText: true})))
	})
	t.Run("EditableRecognizerUnsupported", func(t *testing.T) {
		drag := gestures.NewVerticalDragGestureRecognizer(nil)
		defer drag.Dispose()
		editable := newEditable(spanWith(drag), rendering.RenderEditableOpts{ReadOnly: true})
		layout(t, newHarness(t), editable, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 500}))
		describe(editable)()
		node := semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{})
		requireTag(t, "EditableRecognizerUnsupported", func() { editable.AssembleSemanticsNode(node, semantics.NewSemanticsConfiguration(), nil) })
	})
	t.Run("a tap recognizer in a read only editable", func(t *testing.T) {
		describe(newEditable(spanWith(tap), rendering.RenderEditableOpts{ReadOnly: true}))()
	})
	t.Run("EditableSemanticsInfoMissing", func(t *testing.T) {
		editable := newEditable(editableSpan("text", 14), rendering.RenderEditableOpts{})
		requireTag(t, "EditableSemanticsInfoMissing", func() {
			editable.AssembleSemanticsNode(nil, semantics.NewSemanticsConfiguration(), nil)
		})
	})
}
