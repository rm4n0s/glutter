package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// newPainter makes a painter in Roboto 20 with the text.
func newPainter(t *testing.T, text string, opts ...func(*TextPainterOpts)) *TextPainter {
	t.Helper()
	o := TextPainterOpts{
		TextEngine:    bundledEngine(t),
		Text:          tsStyle(text, robotoStyle(20)),
		TextDirection: ui.TextDirectionLTR,
	}
	for _, f := range opts {
		f(&o)
	}
	return NewTextPainter(o)
}

func approx(a, b, d float64) bool { return math.Abs(a-b) <= d }

func TestTextPainterLayout(t *testing.T) {
	p := newPainter(t, "Hello")
	p.Layout(0, math.Inf(1))
	w := p.Width()
	if w <= 0 || p.Height() <= 0 {
		t.Fatalf("size = %v x %v", w, p.Height())
	}
	if p.MinIntrinsicWidth() != w || p.MaxIntrinsicWidth() != w {
		t.Errorf("intrinsic widths %v %v, width %v", p.MinIntrinsicWidth(), p.MaxIntrinsicWidth(), w)
	}
	if got := p.Size(); got.Width != w || got.Height != p.Height() {
		t.Errorf("size = %v", got)
	}
	// A minimum width stretches the content width, with the parent as basis.
	p.Layout(w+30, math.Inf(1))
	if p.Width() != w+30 {
		t.Errorf("width = %v, want %v", p.Width(), w+30)
	}
	// A maximum width below the text wraps it: the text gets a second line.
	long := newPainter(t, "Hello wide world")
	long.Layout(0, w+1)
	if n := len(long.ComputeLineMetrics()); n < 2 {
		t.Errorf("lines = %d, want at least 2", n)
	}
	if long.Width() > w+1 {
		t.Errorf("width %v exceeds the maximum %v", long.Width(), w+1)
	}
	if !approx(p.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic), p.ComputeLineMetrics()[0].Baseline, 0.01) {
		t.Errorf("alphabetic baseline %v vs line baseline %v", p.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic), p.ComputeLineMetrics()[0].Baseline)
	}
}

func TestTextPainterWidthBasis(t *testing.T) {
	tests := []struct {
		name  string
		basis TextWidthBasis
	}{{"parent", TextWidthBasisParent}, {"longest line", TextWidthBasisLongestLine}}
	var widths []float64
	for _, tt := range tests {
		p := newPainter(t, "aaaa bbbb cccc", func(o *TextPainterOpts) { o.TextWidthBasis = tt.basis })
		p.Layout(0, 90)
		widths = append(widths, p.Width())
	}
	if widths[0] != 90 {
		t.Errorf("parent basis width = %v, want 90", widths[0])
	}
	if widths[1] >= widths[0] {
		t.Errorf("longest line basis width %v should be below %v", widths[1], widths[0])
	}
}

func TestTextPainterEllipsisAndMaxLines(t *testing.T) {
	long := "one two three four five six seven eight nine ten"
	tests := []struct {
		name         string
		opts         func(*TextPainterOpts)
		wantExceeded bool
		wantLines    int
	}{
		{"unlimited", func(*TextPainterOpts) {}, false, -1},
		{"two lines", func(o *TextPainterOpts) { o.MaxLines = foundation.Some(2) }, true, 2},
		{"ellipsis", func(o *TextPainterOpts) { o.MaxLines = foundation.Some(1); o.Ellipsis = "…" }, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPainter(t, long, tt.opts)
			p.Layout(0, 150)
			if got := p.DidExceedMaxLines(); got != tt.wantExceeded {
				t.Errorf("DidExceedMaxLines = %v, want %v", got, tt.wantExceeded)
			}
			if tt.wantLines >= 0 {
				if got := len(p.ComputeLineMetrics()); got != tt.wantLines {
					t.Errorf("lines = %d, want %d", got, tt.wantLines)
				}
			}
		})
	}
}

func TestTextPainterPaintOffset(t *testing.T) {
	tests := []struct {
		name  string
		align ui.TextAlign
		dir   ui.TextDirection
		frac  float64
	}{
		{"left", ui.TextAlignLeft, ui.TextDirectionRTL, 0},
		{"right", ui.TextAlignRight, ui.TextDirectionLTR, 1},
		{"center", ui.TextAlignCenter, ui.TextDirectionLTR, 0.5},
		{"start ltr", ui.TextAlignStart, ui.TextDirectionLTR, 0},
		{"start rtl", ui.TextAlignStart, ui.TextDirectionRTL, 1},
		{"end ltr", ui.TextAlignEnd, ui.TextDirectionLTR, 1},
		{"end rtl", ui.TextAlignEnd, ui.TextDirectionRTL, 0},
		{"justify ltr", ui.TextAlignJustify, ui.TextDirectionLTR, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPainter(t, "Hi", func(o *TextPainterOpts) { o.TextAlign = tt.align; o.TextDirection = tt.dir })
			p.Layout(0, 200)
			rec := ui.NewPictureRecorder()
			canvas := ui.NewCanvas(rec)
			p.Paint(canvas, ui.Offset{Dx: 5, Dy: 7})
			ops := rec.EndRecording().DisplayList().Ops()
			if len(ops) != 1 {
				t.Fatalf("ops = %v", ops)
			}
			op, ok := ops[0].(ui.OpDrawParagraph)
			if !ok {
				t.Fatalf("op = %T", ops[0])
			}
			// The paragraph is laid out 200 wide and aligns its text inside; the painter
			// moves it to the box of the content width.
			wantDx := 5 + tt.frac*(p.Width()-p.layoutCache.paragraph().Width())
			if !approx(op.Offset.Dx, wantDx, 0.01) || op.Offset.Dy != 7 {
				t.Errorf("paint offset = %v, want (%v, 7)", op.Offset, wantDx)
			}
		})
	}
}

func TestTextPainterUnboundedAlignedLayout(t *testing.T) {
	// With an unbounded maximum width and a non-start alignment, the layout uses
	// the intrinsic width so the paint offset stays finite.
	p := newPainter(t, "Hi", func(o *TextPainterOpts) { o.TextAlign = ui.TextAlignCenter })
	p.Layout(0, math.Inf(1))
	w := p.Width()
	if math.IsInf(w, 0) || w <= 0 {
		t.Fatalf("width = %v", w)
	}
	p.Layout(w+20, math.Inf(1))
	if p.Width() != w+20 {
		t.Errorf("width = %v, want %v", p.Width(), w+20)
	}
	if got := p.ComputeLineMetrics()[0].Left; !approx(got, 10, 0.01) {
		t.Errorf("shifted left = %v, want 10", got)
	}
}

func TestTextPainterPositions(t *testing.T) {
	p := newPainter(t, "Hello")
	p.Layout(0, math.Inf(1))
	caret := ui.NewRectFromLTWH(0, 0, 2, 20)
	zero := p.GetOffsetForCaret(ui.NewTextPosition(0), caret)
	end := p.GetOffsetForCaret(ui.NewTextPosition(5), caret)
	mid := p.GetOffsetForCaret(ui.NewTextPosition(2), caret)
	if zero.Dx != 0 {
		t.Errorf("caret at 0 = %v", zero)
	}
	if !approx(end.Dx, p.Width(), 0.01) {
		t.Errorf("caret at the end = %v, want x = %v", end, p.Width())
	}
	if !(zero.Dx < mid.Dx && mid.Dx < end.Dx) {
		t.Errorf("carets not increasing: %v %v %v", zero, mid, end)
	}
	// Hit testing is the inverse of the caret.
	for _, off := range []int{0, 1, 3, 4} {
		o := p.GetOffsetForCaret(ui.NewTextPosition(off), caret)
		got := p.GetPositionForOffset(ui.Offset{Dx: o.Dx + 0.5, Dy: 5})
		if got.Offset != off {
			t.Errorf("position at the caret of %d = %v", off, got)
		}
	}
	boxes := p.GetBoxesForSelection(ui.NewTextRange(1, 3))
	if len(boxes) != 1 {
		t.Fatalf("boxes = %v", boxes)
	}
	if !approx(boxes[0].Left, p.GetOffsetForCaret(ui.NewTextPosition(1), caret).Dx, 0.01) ||
		!approx(boxes[0].Right, p.GetOffsetForCaret(ui.NewTextPosition(3), caret).Dx, 0.01) {
		t.Errorf("box = %v", boxes[0])
	}
	if h := p.GetFullHeightForCaret(ui.NewTextPosition(0), caret); !approx(h, p.Height(), 1.5) {
		t.Errorf("caret height = %v, line height %v", h, p.Height())
	}
	if h := p.PreferredLineHeight(); !approx(h, p.Height(), 1.5) {
		t.Errorf("preferred line height = %v", h)
	}
}

func TestTextPainterEmptyText(t *testing.T) {
	p := newPainter(t, "")
	p.Layout(0, math.Inf(1))
	caret := ui.NewRectFromLTWH(0, 0, 2, 20)
	if got := p.GetOffsetForCaret(ui.NewTextPosition(0), caret); got != (ui.Offset{}) {
		t.Errorf("caret in empty text = %v", got)
	}
	if p.Width() != 0 {
		t.Errorf("width = %v", p.Width())
	}
	if h := p.GetFullHeightForCaret(ui.NewTextPosition(0), caret); h <= 0 {
		t.Errorf("caret height = %v", h)
	}
}

func TestTextPainterRTLCaret(t *testing.T) {
	p := newPainter(t, "שלום", func(o *TextPainterOpts) { o.TextDirection = ui.TextDirectionRTL })
	p.Layout(0, 300)
	caret := ui.NewRectFromLTWH(0, 0, 2, 20)
	start := p.GetOffsetForCaret(ui.NewTextPosition(0), caret)
	end := p.GetOffsetForCaret(ui.NewTextPosition(len("שלום")), caret)
	if !(start.Dx > end.Dx) {
		t.Errorf("rtl caret: start %v should be right of end %v", start, end)
	}
}

func TestTextPainterSetText(t *testing.T) {
	style := robotoStyle(20)
	red := style.CopyWith(TextStyleCopy{Color: col(0xFFFF0000)})
	bigger := robotoStyle(30)
	tests := []struct {
		name         string
		next         InlineSpan
		wantLayout   bool
		wantRebuild  bool
		wantSameSize bool
	}{
		{"same", tsStyle("Hello", style), false, false, true},
		{"paint only", tsStyle("Hello", red), false, true, true},
		{"text", tsStyle("Hello!", style), true, false, false},
		{"layout style", tsStyle("Hello", bigger), true, false, false},
		{"nil", nil, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPainter(t, "")
			p.SetText(tsStyle("Hello", style))
			p.Layout(0, math.Inf(1))
			p.rebuildForPaint = false
			p.SetText(tt.next)
			if got := p.layoutCache == nil; got != tt.wantLayout {
				t.Errorf("needs layout = %v, want %v", got, tt.wantLayout)
			}
			if p.rebuildForPaint != tt.wantRebuild {
				t.Errorf("rebuild for paint = %v, want %v", p.rebuildForPaint, tt.wantRebuild)
			}
		})
	}
}

func TestTextPainterPaintRebuildsParagraph(t *testing.T) {
	style := robotoStyle(20)
	p := newPainter(t, "")
	p.SetText(tsStyle("Hello", style))
	p.Layout(0, math.Inf(1))
	old := p.layoutCache.paragraph()
	p.SetText(tsStyle("Hello", style.CopyWith(TextStyleCopy{Color: col(0xFFFF0000)})))
	rec := ui.NewPictureRecorder()
	p.Paint(ui.NewCanvas(rec), ui.Offset{})
	if p.layoutCache.paragraph() == old {
		t.Error("painting after a paint-only change must paint a rebuilt paragraph")
	}
	if p.rebuildForPaint {
		t.Error("the flag is cleared")
	}
}

func TestTextPainterSettersInvalidateLayout(t *testing.T) {
	tests := []struct {
		name string
		set  func(p *TextPainter)
		want bool
	}{
		{"align", func(p *TextPainter) { p.SetTextAlign(ui.TextAlignCenter) }, true},
		{"same align", func(p *TextPainter) { p.SetTextAlign(ui.TextAlignStart) }, false},
		{"direction", func(p *TextPainter) { p.SetTextDirection(ui.TextDirectionRTL) }, true},
		{"scaler", func(p *TextPainter) { p.SetTextScaler(NewTextScalerLinear(2)) }, true},
		{"same scaler", func(p *TextPainter) { p.SetTextScaler(TextScalerNoScaling) }, false},
		{"ellipsis", func(p *TextPainter) { p.SetEllipsis("…") }, true},
		{"locale", func(p *TextPainter) { p.SetLocale(foundation.Some(ui.NewLocale("fr"))) }, true},
		{"max lines", func(p *TextPainter) { p.SetMaxLines(foundation.Some(2)) }, true},
		{"strut", func(p *TextPainter) { p.SetStrutStyle(NewStrutStyle(StrutStyleOpts{FontSize: f64(5)})) }, true},
		{"height behavior", func(p *TextPainter) {
			b := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOpts{LeadingDistribution: ui.TextLeadingDistributionEven})
			p.SetTextHeightBehavior(&b)
		}, true},
		{"width basis", func(p *TextPainter) { p.SetTextWidthBasis(TextWidthBasisLongestLine) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPainter(t, "Hello")
			p.Layout(0, math.Inf(1))
			tt.set(p)
			if got := p.layoutCache == nil; got != tt.want {
				t.Errorf("layout invalidated = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTextPainterTextScaler(t *testing.T) {
	one := newPainter(t, "Hello")
	one.Layout(0, math.Inf(1))
	two := newPainter(t, "Hello", func(o *TextPainterOpts) { o.TextScaler = NewTextScalerLinear(2) })
	two.Layout(0, math.Inf(1))
	if !approx(two.Width(), one.Width()*2, 0.5) || !approx(two.Height(), one.Height()*2, 1) {
		t.Errorf("scaled size %v x %v vs %v x %v", two.Width(), two.Height(), one.Width(), one.Height())
	}
}

func TestTextPainterPlaceholders(t *testing.T) {
	ph := newTestPlaceholder()
	p := newPainter(t, "", func(o *TextPainterOpts) {
		o.Text = tsStyle("ab", robotoStyle(20), ph, ts("cd"))
	})
	dims := []PlaceholderDimensions{{Size: ui.Size{Width: 30, Height: 15}, Alignment: ui.PlaceholderAlignmentMiddle}}
	p.SetPlaceholderDimensions(dims)
	p.Layout(0, math.Inf(1))
	boxes := p.InlinePlaceholderBoxes()
	if len(boxes) != 1 {
		t.Fatalf("boxes = %v", boxes)
	}
	if !approx(boxes[0].Right-boxes[0].Left, 30, 0.01) || !approx(boxes[0].Bottom-boxes[0].Top, 15, 0.01) {
		t.Errorf("box = %v, want a 30x15 box", boxes[0])
	}
	plain := newPainter(t, "abcd")
	plain.Layout(0, math.Inf(1))
	if p.Width() <= plain.Width() {
		t.Errorf("the placeholder takes space: %v vs %v", p.Width(), plain.Width())
	}
	// Setting the same dimensions again keeps the layout.
	p.SetPlaceholderDimensions(dims)
	if p.layoutCache == nil {
		t.Error("equal dimensions must not invalidate the layout")
	}
	if p.InlinePlaceholderBoxes() == nil {
		t.Error("boxes after layout")
	}
	if (&TextPainter{}).InlinePlaceholderBoxes() != nil {
		t.Error("boxes before layout are nil")
	}
}

func TestTextPainterWordBoundary(t *testing.T) {
	p := newPainter(t, "hello, big world")
	p.Layout(0, math.Inf(1))
	tests := []struct {
		pos  int
		want ui.TextRange
	}{
		{1, ui.NewTextRange(0, 5)},
		{8, ui.NewTextRange(7, 10)},
		{12, ui.NewTextRange(11, 16)},
	}
	for _, tt := range tests {
		if got := p.GetWordBoundary(ui.NewTextPosition(tt.pos)); got != tt.want {
			t.Errorf("GetWordBoundary(%d) = %v, want %v", tt.pos, got, tt.want)
		}
	}
	wb := p.WordBoundaries()
	// Moving by word skips the spaces and the punctuation between words.
	moves := []struct {
		pos         int
		wantLeading int
		wantTrail   int
	}{
		{8, 7, 10},
		{5, 0, 10},
		{11, 11, 16},
	}
	for _, tt := range moves {
		lead, ok := wb.MoveByWordBoundary.GetLeadingTextBoundaryAt(tt.pos)
		if !ok || lead != tt.wantLeading {
			t.Errorf("leading at %d = %d, %v, want %d", tt.pos, lead, ok, tt.wantLeading)
		}
		trail, ok := wb.MoveByWordBoundary.GetTrailingTextBoundaryAt(tt.pos)
		if !ok || trail != tt.wantTrail {
			t.Errorf("trailing at %d = %d, %v, want %d", tt.pos, trail, ok, tt.wantTrail)
		}
	}
	if _, ok := wb.MoveByWordBoundary.GetLeadingTextBoundaryAt(-1); ok {
		t.Error("leading boundary before the text")
	}
	if got := p.GetLineBoundary(ui.NewTextPosition(3)); got != ui.NewTextRange(0, 16) {
		t.Errorf("GetLineBoundary = %v", got)
	}
}

func TestTextPainterOffsetAfterBefore(t *testing.T) {
	p := newPainter(t, "a€😀b")
	tests := []struct {
		offset              int
		after, before       int
		hasAfter, hasBefore bool
	}{
		{0, 1, 0, true, false},
		{1, 4, 0, true, true},
		{4, 8, 1, true, true},
		{8, 9, 4, true, true},
		{9, 0, 8, false, true},
	}
	for _, tt := range tests {
		after, ok := p.GetOffsetAfter(tt.offset)
		if ok != tt.hasAfter || (ok && after != tt.after) {
			t.Errorf("GetOffsetAfter(%d) = %d, %v, want %d, %v", tt.offset, after, ok, tt.after, tt.hasAfter)
		}
		before, ok := p.GetOffsetBefore(tt.offset)
		if ok != tt.hasBefore || (ok && before != tt.before) {
			t.Errorf("GetOffsetBefore(%d) = %d, %v, want %d, %v", tt.offset, before, ok, tt.before, tt.hasBefore)
		}
	}
	for v, want := range map[int][2]bool{0xD800: {true, false}, 0xDBFF: {true, false}, 0xDC00: {false, true}, 0x41: {false, false}} {
		if TextPainterIsHighSurrogate(v) != want[0] || TextPainterIsLowSurrogate(v) != want[1] {
			t.Errorf("surrogates of %#x", v)
		}
	}
}

func TestTextPainterComputeWidth(t *testing.T) {
	engine := bundledEngine(t)
	span := tsStyle("Hello", robotoStyle(20))
	p := newPainter(t, "Hello")
	p.Layout(0, math.Inf(1))
	o := TextPainterComputeWidthOpts{TextEngine: engine}
	if got := TextPainterComputeWidth(span, ui.TextDirectionLTR, o); !approx(got, p.Width(), 0.01) {
		t.Errorf("ComputeWidth = %v, want %v", got, p.Width())
	}
	if got := TextPainterComputeMaxIntrinsicWidth(span, ui.TextDirectionLTR, o); !approx(got, p.MaxIntrinsicWidth(), 0.01) {
		t.Errorf("ComputeMaxIntrinsicWidth = %v, want %v", got, p.MaxIntrinsicWidth())
	}
	o.MinWidth = 500
	if got := TextPainterComputeWidth(span, ui.TextDirectionLTR, o); got != 500 {
		t.Errorf("ComputeWidth with a minimum = %v", got)
	}
}

func TestWordBoundaryHelpers(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{{'\n', true}, {0x85, true}, {0x0B, true}, {0x0C, true}, {0x2028, true}, {0x2029, true}, {'a', false}, {' ', false}, {'\r', false}}
	for _, tt := range tests {
		if got := isNewlineRune(tt.r); got != tt.want {
			t.Errorf("isNewlineRune(%U) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestPlaceholderDimensions(t *testing.T) {
	d := PlaceholderDimensions{Size: ui.Size{Width: 3, Height: 4}, Alignment: ui.PlaceholderAlignmentMiddle}
	if got, want := d.String(), "PlaceholderDimensions(Size(3.0, 4.0), PlaceholderAlignment.middle)"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	b := PlaceholderDimensions{Size: ui.Size{Width: 3, Height: 4}, Alignment: ui.PlaceholderAlignmentBaseline, BaselineOffset: f64(2)}
	if got, want := b.String(), "PlaceholderDimensions(Size(3.0, 4.0), PlaceholderAlignment.baseline(2.0 from top))"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if e := PlaceholderDimensionsEmpty(); e.Size != (ui.Size{}) || e.Alignment != ui.PlaceholderAlignmentBottom {
		t.Errorf("empty = %+v", e)
	}
}
