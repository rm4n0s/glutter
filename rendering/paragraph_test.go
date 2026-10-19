package rendering_test

import (
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// placeholderLength is the length in bytes of the placeholder character.
const placeholderLength = 3

const kText = "I polished up that handle so carefullee\nThat now I am the Ruler of the Queen's Navee!"

// roboto is the text style of the paragraph tests: Roboto at a size. The tests
// of Flutter use a font whose glyphs are one em wide; the bundled fonts are not
// like that, so the tests compare measured values.
func roboto(size float64) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size)})
}

func textSpan(text string, size float64, children ...painting.InlineSpan) *painting.TextSpan {
	return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(text), Style: roboto(size), Children: children})
}

// widthLimit is the constraints of a box of at most the width and any height.
func widthLimit(w float64) rendering.BoxConstraints {
	return rendering.BoxConstraints{MaxWidth: w, MaxHeight: inf}
}

func newParagraph(text painting.InlineSpan, opts ...rendering.RenderParagraphOpts) *rendering.RenderParagraph {
	var o rendering.RenderParagraphOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.TextEngine = renderingtest.BundledTextEngine()
	return rendering.NewRenderParagraph(text, ui.TextDirectionLTR, o)
}

// testPlaceholder is a PlaceholderSpan like WidgetSpan.
type testPlaceholder struct{ painting.PlaceholderSpanBase }

func newPlaceholder() *testPlaceholder {
	return &testPlaceholder{painting.NewPlaceholderSpanBase(painting.PlaceholderSpanOpts{Alignment: ui.PlaceholderAlignmentBottom})}
}

func (p *testPlaceholder) Build(builder *ui.ParagraphBuilder, opts ...painting.SpanBuildOpts) {
	var o painting.SpanBuildOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	d := o.Dimensions[builder.PlaceholderCount()]
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, p.Alignment())
}
func (p *testPlaceholder) VisitChildren(v painting.InlineSpanVisitor) bool     { return v(p) }
func (p *testPlaceholder) VisitDirectChildren(painting.InlineSpanVisitor) bool { return true }
func (p *testPlaceholder) DebugAssertIsValid() bool                            { return true }
func (p *testPlaceholder) CompareTo(o painting.InlineSpan) painting.RenderComparison {
	if painting.InlineSpan(p) == o {
		return painting.RenderComparisonIdentical
	}
	return painting.RenderComparisonLayout
}
func (p *testPlaceholder) Equal(o painting.InlineSpan) bool { return painting.InlineSpan(p) == o }
func (p *testPlaceholder) GetSpanForPositionVisitor(pos ui.TextPosition, off *painting.Accumulator) painting.InlineSpan {
	if pos.Offset == off.Value() {
		return p
	}
	off.Increment(placeholderLength)
	return nil
}
func (p *testPlaceholder) CodeUnitAtVisitor(_ int, off *painting.Accumulator) (byte, bool) {
	off.Increment(placeholderLength)
	return 0, false
}
func (p *testPlaceholder) ToStringShort() string { return "testPlaceholder" }
func (p *testPlaceholder) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	p.DebugFillBase(b)
}
func (p *testPlaceholder) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// applyParentData gives the children the spans that they stand for, like
// _applyParentData of paragraph_test.dart.
func applyParentData(children []rendering.Box, span painting.InlineSpan) {
	i := 0
	span.VisitChildren(func(s painting.InlineSpan) bool {
		if p, ok := s.(painting.PlaceholderSpan); ok {
			children[i].ParentData().(*rendering.TextParentData).Span = p
			i++
		}
		return true
	})
}

func TestParagraphCaretAndPositionQueries(t *testing.T) {
	p := newParagraph(textSpan(kText, 14))
	layout(t, newHarness(t), p)
	caret := ui.NewRectFromLTWH(0, 0, 2, 20)
	offset5 := p.GetOffsetForCaret(ui.NewTextPosition(5), caret)
	if offset5.Dx <= 0 {
		t.Errorf("caret at 5 = %v, want dx > 0", offset5)
	}
	offset25 := p.GetOffsetForCaret(ui.NewTextPosition(25), caret)
	if offset25.Dx <= offset5.Dx {
		t.Errorf("caret at 25 = %v, want to the right of %v", offset25, offset5)
	}
	offset50 := p.GetOffsetForCaret(ui.NewTextPosition(50), caret)
	if offset50.Dy <= offset5.Dy {
		t.Errorf("caret at 50 = %v, want on a lower line than %v", offset50, offset5)
	}
	lineHeight := p.GetFullHeightForCaret(ui.NewTextPosition(5))
	if lineHeight <= 0 || lineHeight > 20 {
		t.Errorf("full caret height = %v", lineHeight)
	}
	position20 := p.GetPositionForOffset(ui.Offset{Dx: 20, Dy: 5})
	position40 := p.GetPositionForOffset(ui.Offset{Dx: 40, Dy: 5})
	positionBelow := p.GetPositionForOffset(ui.Offset{Dx: 5, Dy: lineHeight + 5})
	if position20.Offset <= 0 || position40.Offset <= position20.Offset || positionBelow.Offset <= position40.Offset {
		t.Errorf("positions = %v, %v, %v, want growing", position20, position40, positionBelow)
	}
}

func TestParagraphGetBoxesForSelection(t *testing.T) {
	p := newParagraph(textSpan(kText, 10))
	layout(t, newHarness(t), p)
	if boxes := p.GetBoxesForSelection(ui.TextRange{Start: 5, End: 25}); len(boxes) != 1 {
		t.Errorf("boxes for a range inside the first line = %d, want 1", len(boxes))
	}
	boxes := p.GetBoxesForSelection(ui.TextRange{Start: 25, End: 50})
	if len(boxes) < 2 {
		t.Fatalf("boxes for a range over two lines = %d, want at least 2", len(boxes))
	}
	if boxes[0].Top != 0 || boxes[len(boxes)-1].Top < boxes[0].Bottom {
		t.Errorf("the boxes are not on two lines: %v", boxes)
	}

	t.Run("several spans and lines", func(t *testing.T) {
		first := textSpan("First ", 10, textSpan("smallsecond ", 5), textSpan("third fourth fifth", 10))
		wide := newParagraph(first)
		layout(t, newHarness(t), wide, withConstraints(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}))
		narrow := newParagraph(first)
		layout(t, newHarness(t), narrow, withConstraints(widthLimit(wide.Size().Width/2)))
		boxes := narrow.GetBoxesForSelection(ui.TextRange{Start: 0, End: 36})
		if len(boxes) < 4 {
			t.Fatalf("boxes = %d, want at least 4: %v", len(boxes), boxes)
		}
		// The boxes of a line share the baseline: the small text is aligned lower than the top of the line.
		if boxes[1].Top <= boxes[0].Top || boxes[1].Bottom >= boxes[0].Bottom {
			t.Errorf("the small box %v is not inside the line of %v", boxes[1], boxes[0])
		}
		for i := 1; i < len(boxes); i++ {
			if boxes[i].Top < boxes[i-1].Top {
				t.Errorf("box %d is above box %d", i, i-1)
			}
		}
		maxBoxes := narrow.GetBoxesForSelection(ui.TextRange{Start: 0, End: 36},
			ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleMax, WidthStyle: ui.BoxWidthStyleMax})
		for i := 1; i < len(maxBoxes); i++ {
			if maxBoxes[i].Top == maxBoxes[i-1].Top && maxBoxes[i].Bottom != maxBoxes[i-1].Bottom {
				t.Errorf("the max style boxes %v do not share the height of the line", maxBoxes)
			}
		}
	})
}

func TestParagraphWordBoundary(t *testing.T) {
	p := newParagraph(textSpan(kText, 14))
	layout(t, newHarness(t), p)
	tests := []struct {
		offset int
		want   string
	}{{5, "polished"}, {50, " "}, {75, "Queen's"}}
	for _, tc := range tests {
		r := p.GetWordBoundary(ui.NewTextPosition(tc.offset))
		if got := kText[r.Start:r.End]; got != tc.want {
			t.Errorf("word at %d = %q, want %q", tc.offset, got, tc.want)
		}
	}
}

func TestParagraphOverflow(t *testing.T) {
	skipUnlessDebug(t)
	text := "This\nis a wrapping test. It should wrap at manual newlines, and if softWrap is true, also at spaces."
	p := newParagraph(textSpan(text, 10), rendering.RenderParagraphOpts{MaxLines: foundation.Some(1)})
	h := newHarness(t)
	relayout := func(maxLines foundation.Opt[int], softWrap bool, overflow painting.TextOverflow) {
		p.SetMaxLines(maxLines)
		p.SetSoftWrap(softWrap)
		p.SetOverflow(overflow)
		pumpFrame(h, phaseLayout)
	}
	layout(t, h, p, withConstraints(widthLimit(30)))
	lineHeight := p.Size().Height
	n := foundation.Some[int]
	none := foundation.None[int]()
	tests := []struct {
		name      string
		maxLines  foundation.Opt[int]
		softWrap  bool
		overflow  painting.TextOverflow
		wantLines float64 // 0 means "more than 5"
	}{
		{"3 lines, clip", n(3), true, painting.TextOverflowClip, 3},
		{"unlimited, clip", none, true, painting.TextOverflowClip, 0},
		{"1 line, ellipsis", n(1), true, painting.TextOverflowEllipsis, 1},
		{"3 lines, ellipsis", n(3), true, painting.TextOverflowEllipsis, 3},
		{"unlimited, ellipsis", none, true, painting.TextOverflowEllipsis, 2},
		{"1 line, no wrap, clip", n(1), false, painting.TextOverflowClip, 1},
		{"3 lines, no wrap, clip", n(3), false, painting.TextOverflowClip, 2},
		{"unlimited, no wrap, clip", none, false, painting.TextOverflowClip, 2},
		{"1 line, no wrap, ellipsis", n(1), false, painting.TextOverflowEllipsis, 1},
		{"3 lines, no wrap, ellipsis", n(3), false, painting.TextOverflowEllipsis, 3},
		{"unlimited, no wrap, ellipsis", none, false, painting.TextOverflowEllipsis, 2},
	}
	for _, tc := range tests {
		relayout(tc.maxLines, tc.softWrap, tc.overflow)
		got := p.Size().Height
		if tc.wantLines == 0 {
			if got <= 5*lineHeight {
				t.Errorf("%s: height = %v, want more than %v", tc.name, got, 5*lineHeight)
			}
		} else if math.Abs(got-tc.wantLines*lineHeight) > 1e-9 {
			t.Errorf("%s: height = %v, want %v lines of %v", tc.name, got, tc.wantLines, lineHeight)
		}
	}
	relayout(n(3), true, painting.TextOverflowFade)
	if !p.DebugHasOverflowShader() {
		t.Error("the fade has no shader")
	}
	relayout(n(3), true, painting.TextOverflowEllipsis)
	if p.DebugHasOverflowShader() {
		t.Error("the ellipsis has a shader")
	}
	relayout(n(100), true, painting.TextOverflowFade)
	if p.DebugHasOverflowShader() {
		t.Error("a fade without overflow has a shader")
	}
}

func TestParagraphMaxLines(t *testing.T) {
	text := "How do you write like you're running out of time? Write day and night like you're running out of time?"
	p := newParagraph(textSpan(text, 10))
	h := newHarness(t)
	layout(t, h, p, withConstraints(widthLimit(100)))
	p.SetMaxLines(foundation.Some(1))
	pumpFrame(h, phaseLayout)
	lineHeight := p.Size().Height
	for _, lines := range []int{2, 3} {
		p.SetMaxLines(foundation.Some(lines))
		pumpFrame(h, phaseLayout)
		if got, want := p.Size().Height, float64(lines)*lineHeight; math.Abs(got-want) > 1e-9 {
			t.Errorf("maxLines %d: height = %v, want %v", lines, got, want)
		}
	}
	p.SetMaxLines(foundation.None[int]())
	pumpFrame(h, phaseLayout)
	if p.Size().Height <= 3*lineHeight {
		t.Errorf("unlimited lines: height = %v, want more than %v", p.Size().Height, 3*lineHeight)
	}
	skipUnlessDebug(t)
	requireTag(t, "ParagraphMaxLinesInvalid", func() { p.SetMaxLines(foundation.Some(0)) })
}

func TestParagraphTextAlignRepaintsAndRelayoutsInPaint(t *testing.T) {
	skipUnlessDebug(t)
	p := newParagraph(textSpan("A", 10), rendering.RenderParagraphOpts{TextAlign: ui.TextAlignLeft})
	layout(t, newHarness(t), p, withConstraints(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(100.0)})))
	rectForA := func() ui.Rect {
		boxes := p.GetBoxesForSelection(ui.TextRange{Start: 0, End: 1})
		if len(boxes) != 1 {
			t.Fatalf("boxes = %v", boxes)
		}
		return boxes[0].ToRect()
	}
	left := rectForA()
	if left.Left != 0 {
		t.Errorf("A at %v, want left 0", left)
	}
	p.SetTextAlign(ui.TextAlignRight)
	if p.AsRenderObject().DebugNeedsLayout() || !p.AsRenderObject().DebugNeedsPaint() {
		t.Error("a new alignment must need paint, not layout")
	}
	root := rendering.NewContainerLayer()
	p.Paint(rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600)), ui.OffsetZero)
	if right := rectForA(); math.Abs(right.Right-100) > 1e-9 || right.Width() != left.Width() {
		t.Errorf("A at %v, want it to end at 100", right)
	}
}

func TestParagraphDevicePixelRatio(t *testing.T) {
	skipUnlessDebug(t)
	p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("Hello")}))
	if p.DevicePixelRatio() != 1 {
		t.Errorf("default ratio = %v", p.DevicePixelRatio())
	}
	h := newHarness(t)
	layout(t, h, p, atPhase(layoutOpts{}, phasePaint))
	p.SetDevicePixelRatio(2)
	if p.DevicePixelRatio() != 2 {
		t.Errorf("ratio = %v, want 2", p.DevicePixelRatio())
	}
	if foundation.DebugMode && (p.AsRenderObject().DebugNeedsLayout() || p.AsRenderObject().DebugNeedsPaint()) {
		t.Error("changing the ratio needs layout or paint")
	}
	c := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("Hello")}),
		rendering.RenderParagraphOpts{DevicePixelRatio: foundation.Some(2.5)})
	b := foundation.NewDiagnosticPropertiesBuilder()
	c.DebugFillProperties(b)
	found := false
	for _, n := range b.Properties {
		if n.Name().OrElse("") == "devicePixelRatio" && n.Value() == any(2.5) {
			found = true
		}
	}
	if !found {
		t.Error("devicePixelRatio is not among the properties")
	}
}

func TestParagraphDidExceedMaxLines(t *testing.T) {
	tests := []struct {
		name     string
		maxLines foundation.Opt[int]
		overflow painting.TextOverflow
		want     bool
	}{
		{"none limited", foundation.None[int](), painting.TextOverflowClip, false},
		{"limited by maxLines", foundation.Some(1), painting.TextOverflowClip, true},
		{"limited by ellipsis", foundation.None[int](), painting.TextOverflowEllipsis, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newParagraph(textSpan("Here is a long text, maybe exceed maxlines", 10),
				rendering.RenderParagraphOpts{MaxLines: tc.maxLines, Overflow: tc.overflow})
			layout(t, newHarness(t), p, withConstraints(widthLimit(100)))
			if got := p.DidExceedMaxLines(); got != tc.want {
				t.Errorf("DidExceedMaxLines() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParagraphChangingColorDoesNotLayout(t *testing.T) {
	skipUnlessDebug(t)
	style := func(c uint32) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, Color: foundation.Some(ui.NewColor(c))})
	}
	span := func(text string, c uint32) painting.InlineSpan {
		return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(text), Style: style(c)})
	}
	p := newParagraph(span("Hello", 0xFF000000))
	h := newHarness(t)
	layout(t, h, p, atPhase(withConstraints(widthLimit(100)), phasePaint))
	ro := p.AsRenderObject()
	if ro.DebugNeedsLayout() || ro.DebugNeedsPaint() {
		t.Fatal("the paragraph needs layout or paint after the frame")
	}
	steps := []struct {
		text          painting.InlineSpan
		layout, paint bool
		description   string
	}{
		{span("Hello World", 0xFF000000), true, false, "other text"},
		{span("Hello World", 0xFFFFFFFF), false, true, "other color"},
	}
	for _, s := range steps {
		p.SetText(s.text)
		if ro.DebugNeedsLayout() != s.layout || ro.DebugNeedsPaint() != s.paint {
			t.Errorf("%s: needs layout %v, paint %v, want %v and %v", s.description, ro.DebugNeedsLayout(), ro.DebugNeedsPaint(), s.layout, s.paint)
		}
		pumpFrame(h, phasePaint)
		if ro.DebugNeedsLayout() || ro.DebugNeedsPaint() {
			t.Errorf("%s: the paragraph needs layout or paint after the frame", s.description)
		}
	}
}

func TestParagraphLinearTextScaler(t *testing.T) {
	span := textSpan("a", 10, textSpan("b", 20, textSpan("c", 20)), textSpan("d", 10))
	one := newParagraph(span)
	one.Layout(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}, rendering.LayoutOpts{})
	scaled := newParagraph(span, rendering.RenderParagraphOpts{TextScaler: painting.NewTextScalerLinear(1.3)})
	scaled.Layout(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}, rendering.LayoutOpts{})
	if got, want := scaled.Size().Width, one.Size().Width*1.3; math.Abs(got-want) > 0.01*want {
		t.Errorf("scaled width = %v, want about %v", got, want)
	}
	boxes := func(p *rendering.RenderParagraph) []ui.TextBox {
		var out []ui.TextBox
		for i := 0; i < 4; i++ {
			out = append(out, p.GetBoxesForSelection(ui.TextRange{Start: i, End: i + 1})...)
		}
		return out
	}
	b1, b2 := boxes(one), boxes(scaled)
	if len(b1) != 4 || len(b2) != 4 {
		t.Fatalf("boxes: %d and %d, want 4", len(b1), len(b2))
	}
	for i := range b1 {
		if got, want := b2[i].ToRect().Height(), b1[i].ToRect().Height()*1.3; math.Abs(got-want) > 0.02*want {
			t.Errorf("box %d scaled height = %v, want about %v", i, got, want)
		}
	}
	if !(b1[1].ToRect().Height() > b1[0].ToRect().Height()) {
		t.Errorf("the nested spans have the sizes %v and %v", b1[0].ToRect(), b1[1].ToRect())
	}
}

func TestParagraphToStringDeep(t *testing.T) {
	skipUnlessDebug(t)
	p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(kText)}),
		rendering.RenderParagraphOpts{Locale: foundation.Some(ui.NewLocale("ja", "JP"))})
	want := "RenderParagraph#00000 NEEDS-LAYOUT NEEDS-PAINT DETACHED\n" +
		" │ parentData: MISSING\n" +
		" │ constraints: MISSING\n" +
		" │ size: MISSING\n" +
		" │ textAlign: start\n" +
		" │ textDirection: ltr\n" +
		" │ softWrap: wrapping at box width\n" +
		" │ overflow: clip\n" +
		" │ locale: ja_JP\n" +
		" │ maxLines: unlimited\n" +
		" ╘═╦══ text ═══\n" +
		"   ║ TextSpan:\n" +
		"   ║   \"I polished up that handle so carefullee\n" +
		"   ║   That now I am the Ruler of the Queen's Navee!\"\n" +
		"   ╚═══════════\n"
	if got := deepInfo(p); got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestParagraphLocaleSetter(t *testing.T) {
	p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(kText)}),
		rendering.RenderParagraphOpts{Locale: foundation.Some(ui.NewLocale("zh", "HK"))})
	if p.Locale() != foundation.Some(ui.NewLocale("zh", "HK")) {
		t.Errorf("locale = %v", p.Locale())
	}
	p.SetLocale(foundation.Some(ui.NewLocale("ja", "JP")))
	if p.Locale() != foundation.Some(ui.NewLocale("ja", "JP")) {
		t.Errorf("locale = %v", p.Locale())
	}
}

// inlineParagraph is a paragraph of "a", placeholders of 14x14 and text, with a
// box of 14x14 for each placeholder.
func inlineParagraph(t *testing.T, spec string, maxWidth float64, phase enginePhase) (*rendering.RenderParagraph, []rendering.Box) {
	t.Helper()
	var spans []painting.InlineSpan
	var boxes []rendering.Box
	for _, c := range spec {
		if c == '_' {
			spans = append(spans, newPlaceholder())
			b := newTBox(14, 14)
			b.hitSelf = true
			boxes = append(boxes, b)
		} else {
			spans = append(spans, painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(string(c))}))
		}
	}
	text := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("a"), Style: roboto(10), Children: spans})
	p := newParagraph(text, rendering.RenderParagraphOpts{Children: boxes})
	applyParentData(boxes, text)
	layout(t, newHarness(t), p, atPhase(withConstraints(widthLimit(maxWidth)), phase))
	return p, boxes
}

func TestParagraphInlineWidgets(t *testing.T) {
	p, boxes := inlineParagraph(t, "__a_", 200, phaseLayout)
	// The text is "a", two placeholders, "a" and a placeholder; the engine joins
	// the boxes of the two placeholders that follow each other.
	textBoxes := p.GetBoxesForSelection(ui.TextRange{Start: 0, End: 100})
	if len(textBoxes) != 4 {
		t.Fatalf("boxes = %d, want 4: %v", len(textBoxes), textBoxes)
	}
	for i, wantWidth := range map[int]float64{1: 28, 3: 14} {
		if w, h := textBoxes[i].Right-textBoxes[i].Left, textBoxes[i].Bottom-textBoxes[i].Top; w != wantWidth || h != 14 {
			t.Errorf("placeholder box %d is %vx%v, want %vx14", i, w, h, wantWidth)
		}
	}
	for i := 1; i < len(textBoxes); i++ {
		if textBoxes[i].Left != textBoxes[i-1].Right {
			t.Errorf("box %d does not follow box %d: %v", i, i-1, textBoxes)
		}
	}
	wantOffsets := []ui.Offset{
		{Dx: textBoxes[1].Left, Dy: textBoxes[1].Top},
		{Dx: textBoxes[1].Left + 14, Dy: textBoxes[1].Top},
		{Dx: textBoxes[3].Left, Dy: textBoxes[3].Top},
	}
	for i, want := range wantOffsets {
		if got := offsetOfText(boxes[i]); got != want {
			t.Errorf("child %d at %v, want %v", i, got, want)
		}
	}
	t.Run("height style max aligns the text boxes to the line", func(t *testing.T) {
		maxBoxes := p.GetBoxesForSelection(ui.TextRange{Start: 0, End: 100}, ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleMax})
		for i, b := range maxBoxes {
			if b.Top != 0 || b.Bottom != 14 {
				t.Errorf("box %d = %v, want the line from 0 to 14", i, b)
			}
		}
	})
	t.Run("wraps", func(t *testing.T) {
		p, boxes := inlineParagraph(t, "__a_____", 50, phaseLayout)
		if got := p.Size().Height; got < 42 {
			t.Errorf("height = %v, want three lines of at least 14", got)
		}
		last := offsetOfText(boxes[len(boxes)-1])
		if last.Dy < 28 {
			t.Errorf("the last child is at %v, want on the third line", last)
		}
	})
}

func offsetOfText(b rendering.Box) ui.Offset {
	return b.ParentData().(*rendering.TextParentData).Offset.Value()
}

func TestParagraphInlineChildrenArePaintedAndHit(t *testing.T) {
	p, boxes := inlineParagraph(t, "_", 200, phasePaint)
	if boxes[0].(*tBox).paints == 0 {
		t.Error("the inline child was not painted")
	}
	at := offsetOfText(boxes[0])
	res := rendering.NewBoxHitTestResult()
	if !p.HitTest(res, at.Add(ui.Offset{Dx: 3, Dy: 3})) {
		t.Fatal("the paragraph was not hit")
	}
	var hitChild bool
	for _, e := range res.Path() {
		hitChild = hitChild || e.Target() == any(boxes[0])
	}
	if !hitChild {
		t.Error("the inline child was not hit")
	}
	m := ui.NewMatrix4Identity()
	p.ApplyPaintTransform(boxes[0], m)
	if got, ok := m.AsTranslation(); !ok || got != at {
		t.Errorf("paint transform = %v, want a translation by %v", got, at)
	}
	boxes[0].ParentData().(*rendering.TextParentData).Offset = foundation.None[ui.Offset]()
	z := ui.NewMatrix4Identity()
	p.ApplyPaintTransform(boxes[0], z)
	if z.IsIdentity() {
		t.Errorf("the transform of a child that was not laid out must be zero, it is the identity")
	}
}

func TestParagraphTextSpanHitTesting(t *testing.T) {
	a := textSpan(strings.Repeat("A", 10), 10)
	bc := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("BC"), Style: painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(10.0), LetterSpacing: foundation.Some(26.0)})})
	text := painting.NewTextSpan(painting.TextSpanOpts{Style: roboto(10), Children: []painting.InlineSpan{a, bc}})
	p := newParagraph(text)
	// Wide enough for all the As, not for the Bs.
	aWidth := newParagraph(a)
	layout(t, newHarness(t), aWidth, withConstraints(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}))
	width := math.Ceil(aWidth.Size().Width) + 1
	layout(t, newHarness(t), p, withConstraints(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(width)})))
	lineHeight := aWidth.Size().Height
	spansAt := func(x, y float64) ([]painting.InlineSpan, bool) {
		res := rendering.NewBoxHitTestResult()
		hit := p.HitTest(res, ui.Offset{Dx: x, Dy: y})
		var spans []painting.InlineSpan
		for _, e := range res.Path() {
			if s, ok := e.Target().(*painting.TextSpan); ok {
				spans = append(spans, s)
			}
		}
		return spans, hit
	}
	if spans, hit := spansAt(2, lineHeight/2); !hit || len(spans) != 1 || spans[0] != painting.InlineSpan(a) {
		t.Errorf("first A: hit %v, spans %v", hit, spans)
	}
	if spans, hit := spansAt(width-2, lineHeight/2); !hit || len(spans) != 1 || spans[0] != painting.InlineSpan(a) {
		t.Errorf("last A: hit %v, spans %v", hit, spans)
	}
	if spans, hit := spansAt(width+100, lineHeight/2); hit || len(spans) != 0 {
		t.Errorf("far away from the line: hit %v, spans %v", hit, spans)
	}
	if spans, hit := spansAt(13+3, lineHeight*1.5); !hit || len(spans) != 1 || spans[0] != painting.InlineSpan(bc) {
		t.Errorf("on B: hit %v, spans %v", hit, spans)
	}
	if spans, hit := spansAt(9999, 9999); hit || len(spans) != 0 {
		t.Errorf("not even close: hit %v, spans %v", hit, spans)
	}
	var _ gestures.HitTestTarget = a
}

// testRegistrar is TestSelectionRegistrar of paragraph_test.dart.
type testRegistrar struct{ selectables []rendering.Selectable }

func (r *testRegistrar) Add(s rendering.Selectable) { r.selectables = append(r.selectables, s) }
func (r *testRegistrar) Remove(s rendering.Selectable) {
	for i, x := range r.selectables {
		if x == s {
			r.selectables = append(r.selectables[:i], r.selectables[i+1:]...)
			return
		}
	}
	panic("removed a selectable that was not registered")
}

func TestParagraphSelectionRegistrar(t *testing.T) {
	registrar := &testRegistrar{}
	p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("1234567")}), rendering.RenderParagraphOpts{Registrar: registrar})
	if len(registrar.selectables) != 1 {
		t.Fatalf("selectables = %d, want 1", len(registrar.selectables))
	}
	f := registrar.selectables[0]
	if !p.SelectableBelongsToParagraph(f) || !p.AlwaysNeedsCompositing() {
		t.Error("the fragment is not the paragraph's, or the paragraph does not need compositing")
	}
	if f.ContentLength() != 7 || !f.Value().HasContent || f.Value().HasSelection() {
		t.Errorf("fragment: length %d, geometry %+v", f.ContentLength(), f.Value())
	}
	layout(t, newHarness(t), p)
	if len(f.BoundingBoxes()) != 1 {
		t.Errorf("bounding boxes = %v", f.BoundingBoxes())
	}
	if f.Size() != p.TextSize() {
		t.Errorf("fragment size = %v, want %v", f.Size(), p.TextSize())
	}
	p.SetText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("")}))
	if len(registrar.selectables) != 0 {
		t.Errorf("selectables after an empty text = %d, want 0", len(registrar.selectables))
	}
	p.SetText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("abc")}))
	if len(registrar.selectables) != 1 {
		t.Errorf("selectables after new text = %d, want 1", len(registrar.selectables))
	}
	p.SetRegistrar(nil)
	if len(registrar.selectables) != 0 {
		t.Errorf("selectables after removing the registrar = %d, want 0", len(registrar.selectables))
	}
	p.Dispose()
}

func TestParagraphSelectionFragmentsSplitAtPlaceholders(t *testing.T) {
	registrar := &testRegistrar{}
	text := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("ab"), Style: roboto(10), Children: []painting.InlineSpan{
		newPlaceholder(), painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("cde")}), newPlaceholder(), newPlaceholder()}})
	boxes := []rendering.Box{newTBox(1, 1), newTBox(1, 1), newTBox(1, 1)}
	newParagraph(text, rendering.RenderParagraphOpts{Registrar: registrar, Children: boxes})
	if len(registrar.selectables) != 2 {
		t.Fatalf("fragments = %d, want 2", len(registrar.selectables))
	}
	if registrar.selectables[0].ContentLength() != 2 || registrar.selectables[1].ContentLength() != 3 {
		t.Errorf("lengths = %d, %d, want 2 and 3", registrar.selectables[0].ContentLength(), registrar.selectables[1].ContentLength())
	}
}

func TestParagraphSelectionColorRepaintsOnlyWithSelection(t *testing.T) {
	skipUnlessDebug(t)
	p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("abc")}), rendering.RenderParagraphOpts{Registrar: &testRegistrar{}})
	h := newHarness(t)
	layout(t, h, p, atPhase(layoutOpts{}, phasePaint))
	p.SetSelectionColor(foundation.Some(ui.NewColor(0xFF112233)))
	if p.AsRenderObject().DebugNeedsPaint() {
		t.Error("a selection color without a selection needs paint")
	}
	if p.SelectionColor() != foundation.Some(ui.NewColor(0xFF112233)) {
		t.Errorf("selection color = %v", p.SelectionColor())
	}
}

func TestPositionInlineChildren(t *testing.T) {
	skipUnlessDebug(t)
	mk := func(n int) *rendering.RenderParagraph {
		p := newParagraph(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("How are you \n")}))
		for i := 0; i < n; i++ {
			p.Add(rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}), nil))
		}
		return p
	}
	boxes := func(n int) []ui.TextBox {
		var out []ui.TextBox
		for i := 0; i < n; i++ {
			out = append(out, ui.TextBox{Left: float64(i) * 10, Top: 0, Right: float64(i+1) * 10, Bottom: 10, Direction: ui.TextDirectionLTR})
		}
		return out
	}
	p := mk(2)
	requireTag(t, "InlineBoxesMismatch", func() { rendering.PositionInlineChildren(&p.ContainerRenderObject, boxes(3)) })
	q := mk(3)
	rendering.PositionInlineChildren(&q.ContainerRenderObject, boxes(2))
	first := q.FirstChild().ParentData().(*rendering.TextParentData)
	last := q.LastChild().ParentData().(*rendering.TextParentData)
	if first.Offset != foundation.Some(ui.Offset{}) || last.Offset.IsSome() {
		t.Errorf("first offset %v, last offset %v, want (0, 0) and none", first.Offset, last.Offset)
	}
	if got := first.String(); !strings.Contains(got, "offset: Offset(0.0, 0.0)") {
		t.Errorf("String() = %q", got)
	}
	if got := last.String(); !strings.Contains(got, "not laid out") {
		t.Errorf("String() = %q", got)
	}
}

func TestParagraphDebugAsserts(t *testing.T) {
	skipUnlessDebug(t)
	spanOf := func() painting.InlineSpan {
		return painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("a"), Style: roboto(10), Children: []painting.InlineSpan{newPlaceholder()}})
	}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"query before layout", "ParagraphNeedsLayout", func() {
			newParagraph(spanOf()).GetOffsetForCaret(ui.NewTextPosition(0), ui.NewRectFromLTWH(0, 0, 1, 1))
		}},
		{"baseline before layout", "BaselineNeedsLayout", func() {
			newParagraph(spanOf()).ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic)
		}},
		{"inline child without a span", "InlineChildWithoutSpan", func() {
			p := newParagraph(spanOf(), rendering.RenderParagraphOpts{Children: []rendering.Box{newTBox(1, 1)}})
			layout(t, newHarness(t), p, withConstraints(widthLimit(100)))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, tc.call) })
	}
}

// TestFlexCrossAxisIntrinsicsWithText is "cross axis intrinsics, with ascending
// flex flow layout" and "... descending ..." of flex_test.dart. The reference
// uses a font whose glyphs are as wide as the font size; here the expected
// height of each paragraph is measured with a paragraph of its own at the width
// that the flex gives it.
func TestFlexCrossAxisIntrinsicsWithText(t *testing.T) {
	text := func() painting.InlineSpan { return textSpan("A\u200bA\u200bA", 10) }
	probe := newParagraph(text())
	one := probe.GetMaxIntrinsicWidth(inf) / 3
	for _, tc := range []struct {
		name         string
		flex2, flex3 int
	}{{"ascending", 1, 2}, {"descending", 2, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			box1 := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
				Width: foundation.Some(5.0), Height: foundation.Some(5.0)}), nil)
			box2, box3 := newParagraph(text()), newParagraph(text())
			flex := rendering.NewRenderFlex([]rendering.Box{box1, box2, box3}, rendering.RenderFlexOpts{TextDirection: ui.TextDirectionLTR})
			for _, b := range []struct {
				box  *rendering.RenderParagraph
				flex int
			}{{box2, tc.flex2}, {box3, tc.flex3}} {
				pd := b.box.ParentData().(*rendering.FlexParentData)
				pd.Flex, pd.Fit = foundation.Some(b.flex), rendering.FlexFitTight // the fit has no effect on intrinsics
			}
			heightAt := func(width float64, flex int) float64 {
				return newParagraph(text()).GetMinIntrinsicHeight(width * float64(flex) / float64(tc.flex2+tc.flex3))
			}
			if got, want := flex.GetMinIntrinsicHeight(inf), probe.GetMinIntrinsicHeight(inf); got != want {
				t.Errorf("height at infinity = %v, want %v", got, want)
			}
			// The widths that the paragraphs get: they hold 3, 2 or 1 'A's per line.
			for _, width := range []float64{5 + 9*one + 1, 5 + 9*one - 1, 5 + 6*one, 5 + 3*one, 5 + 2*one} {
				free := width - 5
				want := max(5, heightAt(free, tc.flex2), heightAt(free, tc.flex3))
				if got := flex.GetMinIntrinsicHeight(width); math.Abs(got-want) > 1e-9 {
					t.Errorf("min intrinsic height at %.1f = %v, want %v", width, got, want)
				}
				if got := flex.GetMaxIntrinsicHeight(width); math.Abs(got-want) > 1e-9 {
					t.Errorf("max intrinsic height at %.1f = %v, want %v", width, got, want)
				}
			}
		})
	}
}

// TestWrapIntrinsicsWithText are the two wrap_test.dart cases that have
// children whose height depends on their width (or the reverse).
func TestWrapIntrinsicsWithText(t *testing.T) {
	const lineHeight = 15.0
	longText := func() painting.InlineSpan {
		return textSpan("A very very very very very very very very long text", lineHeight)
	}
	probe := newParagraph(longText())
	natural := probe.GetMaxIntrinsicWidth(inf)
	widths := []float64{inf, natural + 10, natural / 2, natural / 3}

	t.Run("width in height out", func(t *testing.T) {
		wrap := rendering.NewRenderWrap([]rendering.Box{newParagraph(longText())})
		for _, w := range widths {
			if got, want := wrap.ComputeMaxIntrinsicHeight(w), newParagraph(longText()).GetMaxIntrinsicHeight(w); got != want {
				t.Errorf("max intrinsic height at %v = %v, want %v", w, got, want)
			}
		}
		if got := wrap.ComputeMaxIntrinsicHeight(inf); math.Abs(got-newParagraph(longText()).GetMaxIntrinsicHeight(inf)) > 1e-9 || got <= 0 {
			t.Errorf("max intrinsic height at infinity = %v", got)
		}
		if got, want := wrap.ComputeMaxIntrinsicHeight(natural/3), 3*wrap.ComputeMaxIntrinsicHeight(inf); got < want {
			t.Errorf("a third of the width needs at least three lines: %v, want at least %v", got, want)
		}
	})
	t.Run("height in width out", func(t *testing.T) {
		wrap := rendering.NewRenderWrap([]rendering.Box{rendering.NewRenderRotatedBox(1, newParagraph(longText()))},
			rendering.RenderWrapOpts{Direction: painting.AxisVertical})
		for _, w := range widths {
			if got, want := wrap.ComputeMaxIntrinsicWidth(w), newParagraph(longText()).GetMaxIntrinsicHeight(w); got != want {
				t.Errorf("max intrinsic width at %v = %v, want %v", w, got, want)
			}
		}
	})
}
