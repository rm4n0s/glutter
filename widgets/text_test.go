package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func style(size float64) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(size), FontFamily: "Roboto", Color: foundation.Some(ui.NewColor(0xFF112233))})
}

func richOf(ts *gluttertest.Tester) *rendering.RenderParagraph {
	return ts.RenderObject(gluttertest.ByType[*RichText]()).(*rendering.RenderParagraph)
}

func TestTextSizesFollowTheFontSize(t *testing.T) {
	small, large := style(10), style(20)
	tests := []struct {
		name string
		w    Widget
	}{
		{"small", NewText("Hello", TextOpts{Style: small})},
		{"large", NewText("Hello", TextOpts{Style: large})},
	}
	sizes := make([]ui.Size, len(tests))
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			sizes[i] = ts.Size(gluttertest.ByType[*RichText]())
			if sizes[i].Width <= 0 || sizes[i].Height <= 0 {
				t.Errorf("size = %v, want a positive size", sizes[i])
			}
		})
	}
	if sizes[1].Width < sizes[0].Width*1.9 || sizes[1].Width > sizes[0].Width*2.1 {
		t.Errorf("a font twice as large gave widths %v and %v", sizes[0].Width, sizes[1].Width)
	}
}

func TestTextFindersMatchTheString(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewColumn([]Widget{NewText("alpha"), NewTextRich(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("be"),
		Children: []painting.InlineSpan{painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("ta")})}}))}), ui.TextDirectionLTR)
	for _, s := range []string{"alpha", "beta"} {
		if n := ts.Count(gluttertest.ByText(s)); n != 1 {
			t.Errorf("ByText(%q) found %d, want 1", s, n)
		}
	}
	if ts.Exists(gluttertest.ByText("gamma")) {
		t.Error("ByText(gamma) found something")
	}
	// The RichText inside each Text is only found when it is asked for.
	if n := ts.Count(gluttertest.ByText("alpha", gluttertest.FindOpts{FindRichText: true})); n != 2 {
		t.Errorf("ByText(alpha, FindRichText) found %d, want 2", n)
	}
	if n := ts.Count(gluttertest.ByTextContaining("et", gluttertest.FindOpts{FindRichText: true})); n != 2 {
		t.Errorf("ByTextContaining(et, FindRichText) found %d, want 2", n)
	}
}

func TestDefaultTextStyleIsInherited(t *testing.T) {
	tests := []struct {
		name      string
		w         Widget
		wantSize  float64
		wantWrap  bool
		wantLines foundation.Opt[int]
		wantAlign ui.TextAlign
	}{
		{"fallback", NewText("x"), -1, true, foundation.None[int](), ui.TextAlignStart},
		{"default text style", NewDefaultTextStyle(style(30), NewText("x"), DefaultTextStyleOpts{SoftWrap: foundation.Some(false),
			MaxLines: foundation.Some(2), TextAlign: ui.TextAlignCenter}), 30, false, foundation.Some(2), ui.TextAlignCenter},
		{"own style wins", NewDefaultTextStyle(style(30), NewText("x", TextOpts{Style: style(12)})), 12, true, foundation.None[int](), ui.TextAlignStart},
		{"own options win", NewDefaultTextStyle(style(30), NewText("x", TextOpts{SoftWrap: foundation.Some(true),
			MaxLines: foundation.Some(1), TextAlign: ui.TextAlignEnd}), DefaultTextStyleOpts{SoftWrap: foundation.Some(false), MaxLines: foundation.Some(5)}),
			30, true, foundation.Some(1), ui.TextAlignEnd},
		{"merge", NewDefaultTextStyle(style(30), NewDefaultTextStyleMerge(NewText("x"), DefaultTextStyleMergeOpts{
			MaxLines: foundation.Some(3), Style: painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightW700})})),
			30, true, foundation.Some(3), ui.TextAlignStart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			p := richOf(ts)
			span := p.Text().(*painting.TextSpan)
			if got := span.Style().FontSize.OrElse(-1); got != tt.wantSize {
				t.Errorf("font size = %v, want %v", got, tt.wantSize)
			}
			if p.SoftWrap() != tt.wantWrap {
				t.Errorf("soft wrap = %v, want %v", p.SoftWrap(), tt.wantWrap)
			}
			if p.MaxLines() != tt.wantLines {
				t.Errorf("max lines = %v, want %v", p.MaxLines(), tt.wantLines)
			}
			if p.TextAlign() != tt.wantAlign {
				t.Errorf("text align = %v, want %v", p.TextAlign(), tt.wantAlign)
			}
		})
	}
}

func TestDefaultTextStyleNotifiesDependents(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var builds int
	child := builderOf(func(bc BuildContext) Widget { builds++; DefaultTextStyleOf(bc); return newLeaf() })
	pumpLoose(ts, NewDefaultTextStyle(style(10), child), ui.TextDirectionLTR)
	if builds != 1 {
		t.Fatalf("builds = %d, want 1", builds)
	}
	pumpLoose(ts, NewDefaultTextStyle(style(10), child), ui.TextDirectionLTR)
	if builds != 1 {
		t.Errorf("an equal style rebuilt the dependent: builds = %d", builds)
	}
	pumpLoose(ts, NewDefaultTextStyle(style(11), child), ui.TextDirectionLTR)
	if builds != 2 {
		t.Errorf("a new style did not rebuild the dependent: builds = %d", builds)
	}
}

func TestDefaultTextHeightBehavior(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	b := ui.TextHeightBehavior{ApplyHeightToFirstAscent: false}
	pumpLoose(ts, NewDefaultTextHeightBehavior(b, NewText("x")), ui.TextDirectionLTR)
	if got := richOf(ts).TextHeightBehavior(); got == nil || *got != b {
		t.Errorf("text height behavior = %v, want %v", got, b)
	}
	var got ui.TextHeightBehavior
	ts.PumpWidget(NewDefaultTextHeightBehavior(b, builderOf(func(bc BuildContext) Widget { got = DefaultTextHeightBehaviorOf(bc); return newLeaf() })))
	if got != b {
		t.Errorf("DefaultTextHeightBehaviorOf = %v, want %v", got, b)
	}
	skipUnlessDebug(t)
	ts.PumpWidget(builderOf(func(bc BuildContext) Widget { DefaultTextHeightBehaviorOf(bc); return newLeaf() }))
	_ = requireErrorTag(t, ts.TakeError(), "DefaultTextHeightBehaviorMissing")
}

func TestTextSemantics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		w     Widget
		label string
	}{
		{"label replaces the text", NewText("12 345", TextOpts{SemanticsLabel: foundation.Some("twelve thousand")}), "twelve thousand"},
		{"plain text is its own label", NewText("hello"), "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.EnsureSemantics()
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			if !ts.Exists(gluttertest.BySemanticsLabel(tt.label)) {
				t.Errorf("no node with the label %q", tt.label)
			}
		})
	}
}

func TestTextMaxLinesMustBePositive(t *testing.T) {
	skipUnlessDebug(t)
	err := panicOf(func() { NewText("x", TextOpts{MaxLines: foundation.Some(0)}) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "TextMaxLinesInvalid")
}

func TestRichTextNeedsADirection(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewText("x"))
	_ = requireErrorTag(t, ts.TakeError(), "DirectionalityMissing")
}

func TestWidgetSpanLaysTheWidgetInTheLine(t *testing.T) {
	tests := []struct {
		name string
		span *WidgetSpan
	}{
		{"bottom", NewWidgetSpan(newBox(30, 20))},
		{"middle", NewWidgetSpan(newBox(30, 20), WidgetSpanOpts{Alignment: ui.PlaceholderAlignmentMiddle})},
		{"baseline", NewWidgetSpan(newBox(30, 20), WidgetSpanOpts{Alignment: ui.PlaceholderAlignmentBaseline, Baseline: ui.TextBaselineAlphabetic})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			text := painting.NewTextSpan(painting.TextSpanOpts{Style: style(14), Text: foundation.Some("ab"),
				Children: []painting.InlineSpan{tt.span, painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("cd")})}})
			pumpLoose(ts, NewRichText(text), ui.TextDirectionLTR)
			size := ts.Size(gluttertest.ByType[*RichText]())
			if size.Width < 30 || size.Height < 20 {
				t.Errorf("paragraph size = %v, want at least the 30x20 widget", size)
			}
			b := ts.TopLeft(gluttertest.ByType[*box]())
			p := ts.TopLeft(gluttertest.ByType[*RichText]())
			if b.Dx <= p.Dx {
				t.Errorf("the widget is at x=%v, want it after the first letters (paragraph at %v)", b.Dx, p.Dx)
			}
			if got := ts.Size(gluttertest.ByType[*box]()); got != (ui.Size{Width: 30, Height: 20}) {
				t.Errorf("widget size = %v, want 30x20", got)
			}
		})
	}
}

func TestWidgetSpanScalesWithTheText(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	text := painting.NewTextSpan(painting.TextSpanOpts{Style: style(10), Children: []painting.InlineSpan{NewWidgetSpan(newBox(10, 10))}})
	pumpLoose(ts, NewRichText(text, RichTextOpts{TextScaler: painting.NewTextScalerLinear(2)}), ui.TextDirectionLTR)
	scaler := ts.Elements(gluttertest.ByType[*RichText]())
	if len(scaler) != 1 {
		t.Fatalf("%d paragraphs", len(scaler))
	}
	// The inline widget is drawn at twice its size by the scaling wrapper.
	var wrapper rendering.Box
	scaler[0].VisitChildren(func(e Element) {
		e.VisitChildren(func(e Element) { e.VisitChildren(func(e Element) { wrapper, _ = e.RenderObject().(rendering.Box) }) })
	})
	if wrapper == nil || wrapper.Size() != (ui.Size{Width: 20, Height: 20}) {
		t.Errorf("scaled widget = %v, want a 20x20 box", wrapper)
	}
}

func TestWidgetSpanOrderAndReplace(t *testing.T) {
	a, b := newBox(5, 5, WidgetOpts{Key: foundation.NewValueKey("a")}), newBox(6, 6, WidgetOpts{Key: foundation.NewValueKey("b")})
	build := func(children ...painting.InlineSpan) Widget {
		return NewRichText(painting.NewTextSpan(painting.TextSpanOpts{Children: children}))
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, build(NewWidgetSpan(a), NewWidgetSpan(b)), ui.TextDirectionLTR)
	if !ts.Exists(gluttertest.ByKey(foundation.NewValueKey("a"))) || !ts.Exists(gluttertest.ByKey(foundation.NewValueKey("b"))) {
		t.Fatal("a widget span is missing")
	}
	pumpLoose(ts, build(NewWidgetSpan(b)), ui.TextDirectionLTR)
	if ts.Exists(gluttertest.ByKey(foundation.NewValueKey("a"))) {
		t.Error("the removed span's widget is still there")
	}
	if got := ts.Size(gluttertest.ByKey(foundation.NewValueKey("b"))); got != (ui.Size{Width: 6, Height: 6}) {
		t.Errorf("size of b = %v", got)
	}
}

func TestWidgetSpanBaselineIsRequired(t *testing.T) {
	skipUnlessDebug(t)
	err := panicOf(func() { NewWidgetSpan(newLeaf(), WidgetSpanOpts{Alignment: ui.PlaceholderAlignmentBaseline}) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "WidgetSpanNoBaseline")
}

func TestWidgetSpanComparison(t *testing.T) {
	child := newLeaf()
	tests := []struct {
		name      string
		a, b      painting.InlineSpan
		wantEqual bool
		want      painting.RenderComparison
	}{
		{"same child", NewWidgetSpan(child), NewWidgetSpan(child), true, painting.RenderComparisonIdentical},
		{"other child", NewWidgetSpan(child), NewWidgetSpan(newLeaf()), false, painting.RenderComparisonLayout},
		{"other alignment", NewWidgetSpan(child), NewWidgetSpan(child, WidgetSpanOpts{Alignment: ui.PlaceholderAlignmentTop}), false, painting.RenderComparisonLayout},
		{"text span", NewWidgetSpan(child), painting.NewTextSpan(), false, painting.RenderComparisonLayout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.wantEqual {
				t.Errorf("Equal = %v, want %v", got, tt.wantEqual)
			}
			if got := tt.a.CompareTo(tt.b); got != tt.want {
				t.Errorf("CompareTo = %v, want %v", got, tt.want)
			}
		})
	}
}
