package painting

import (
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func ts(text string, children ...InlineSpan) *TextSpan {
	return NewTextSpan(TextSpanOpts{Text: foundation.Some(text), Children: children})
}

func tsStyle(text string, style *TextStyle, children ...InlineSpan) *TextSpan {
	return NewTextSpan(TextSpanOpts{Text: foundation.Some(text), Style: style, Children: children})
}

// testPlaceholder is a PlaceholderSpan like WidgetSpan.
type testPlaceholder struct{ PlaceholderSpanBase }

func newTestPlaceholder() *testPlaceholder {
	return &testPlaceholder{NewPlaceholderSpanBase(PlaceholderSpanOpts{Alignment: ui.PlaceholderAlignmentMiddle})}
}
func (p *testPlaceholder) Build(builder *ui.ParagraphBuilder, opts ...SpanBuildOpts) {
	_, dims := spanBuildScaler(opts)
	d := dims[builder.PlaceholderCount()]
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, p.Alignment())
}
func (p *testPlaceholder) VisitChildren(v InlineSpanVisitor) bool     { return v(p) }
func (p *testPlaceholder) VisitDirectChildren(InlineSpanVisitor) bool { return true }
func (p *testPlaceholder) DebugAssertIsValid() bool                   { return true }
func (p *testPlaceholder) CompareTo(o InlineSpan) RenderComparison {
	if InlineSpan(p) == o {
		return RenderComparisonIdentical
	}
	return RenderComparisonLayout
}
func (p *testPlaceholder) Equal(o InlineSpan) bool { return InlineSpan(p) == o }
func (p *testPlaceholder) GetSpanForPositionVisitor(pos ui.TextPosition, off *Accumulator) InlineSpan {
	if pos.Offset == off.Value() {
		return p
	}
	off.Increment(len(placeholderText))
	return nil
}
func (p *testPlaceholder) CodeUnitAtVisitor(index int, off *Accumulator) (byte, bool) {
	local := index - off.Value()
	off.Increment(len(placeholderText))
	_ = local
	return 0, false
}
func (p *testPlaceholder) ToStringShort() string { return "testPlaceholder" }
func (p *testPlaceholder) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	p.DebugFillBase(b)
}
func (p *testPlaceholder) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

func TestInlineSpanToPlainText(t *testing.T) {
	labelled := NewTextSpan(TextSpanOpts{Text: foundation.Some("abc"), SemanticsLabel: foundation.Some("label")})
	tests := []struct {
		name string
		span InlineSpan
		opts ToPlainTextOpts
		want string
	}{
		{"nested", ts("a", ts("b", ts("c")), ts("d")), ToPlainTextOpts{}, "abcd"},
		{"no text", NewTextSpan(TextSpanOpts{Children: []InlineSpan{ts("x")}}), ToPlainTextOpts{}, "x"},
		{"labels", ts("a", labelled), ToPlainTextOpts{}, "alabel"},
		{"labels excluded", ts("a", labelled), ToPlainTextOpts{ExcludeSemanticsLabels: true}, "aabc"},
		{"placeholder", ts("a", newTestPlaceholder(), ts("b")), ToPlainTextOpts{}, "a￼b"},
		{"placeholder excluded", ts("a", newTestPlaceholder(), ts("b")), ToPlainTextOpts{ExcludePlaceholders: true}, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InlineSpanToPlainText(tt.span, tt.opts); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInlineSpanGetSpanForPosition(t *testing.T) {
	a, b, c := ts("abc"), ts("de"), ts("f")
	root := ts("", a, ts("", b), c)
	root.Text = foundation.None[string]()
	tests := []struct {
		name string
		pos  ui.TextPosition
		want InlineSpan
	}{
		{"start", ui.NewTextPosition(0), a},
		{"inside", ui.NewTextPosition(2), a},
		{"boundary downstream goes to the next", ui.NewTextPosition(3), b},
		{"boundary upstream stays", ui.NewTextPosition(3, ui.TextPositionOpts{Affinity: ui.TextAffinityUpstream}), a},
		{"last span", ui.NewTextPosition(5), c},
		{"end downstream", ui.NewTextPosition(6), nil},
		{"end upstream", ui.NewTextPosition(6, ui.TextPositionOpts{Affinity: ui.TextAffinityUpstream}), c},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InlineSpanGetSpanForPosition(root, tt.pos); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInlineSpanCodeUnitAt(t *testing.T) {
	root := ts("ab", ts("cd"), ts("é"))
	tests := []struct {
		index int
		want  byte
		ok    bool
	}{
		{-1, 0, false}, {0, 'a', true}, {1, 'b', true}, {2, 'c', true}, {3, 'd', true}, {4, 0xC3, true}, {5, 0xA9, true}, {6, 0, false},
	}
	for _, tt := range tests {
		got, ok := InlineSpanCodeUnitAt(root, tt.index)
		if got != tt.want || ok != tt.ok {
			t.Errorf("CodeUnitAt(%d) = %v, %v, want %v, %v", tt.index, got, ok, tt.want, tt.ok)
		}
	}
}

func TestTextSpanCompareTo(t *testing.T) {
	small := NewTextStyle(TextStyleOpts{FontSize: f64(10)})
	big := NewTextStyle(TextStyleOpts{FontSize: f64(20)})
	red := NewTextStyle(TextStyleOpts{FontSize: f64(10), Color: col(0xFFFF0000)})
	rec1, rec2 := newFakeRecognizer(), newFakeRecognizer()
	tests := []struct {
		name string
		a, b InlineSpan
		want RenderComparison
	}{
		{"same", ts("a"), ts("a"), RenderComparisonIdentical},
		{"text", ts("a"), ts("b"), RenderComparisonLayout},
		{"child count", ts("a", ts("b")), ts("a"), RenderComparisonLayout},
		{"style presence", tsStyle("a", small), ts("a"), RenderComparisonLayout},
		{"style layout", tsStyle("a", small), tsStyle("a", big), RenderComparisonLayout},
		{"style paint", tsStyle("a", small), tsStyle("a", red), RenderComparisonPaint},
		{"recognizer", NewTextSpan(TextSpanOpts{Text: foundation.Some("a"), Recognizer: rec1}), NewTextSpan(TextSpanOpts{Text: foundation.Some("a"), Recognizer: rec2}), RenderComparisonMetadata},
		{"recognizer then paint", NewTextSpan(TextSpanOpts{Text: foundation.Some("a"), Recognizer: rec1, Style: small}),
			NewTextSpan(TextSpanOpts{Text: foundation.Some("a"), Recognizer: rec2, Style: red}), RenderComparisonPaint},
		{"child paint", ts("a", tsStyle("b", small)), ts("a", tsStyle("b", red)), RenderComparisonPaint},
		{"child layout", ts("a", tsStyle("b", small)), ts("a", tsStyle("b", big)), RenderComparisonLayout},
		{"other kind", ts("a"), newTestPlaceholder(), RenderComparisonLayout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.CompareTo(tt.b); got != tt.want {
				t.Errorf("CompareTo = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTextSpanEqual(t *testing.T) {
	small := NewTextStyle(TextStyleOpts{FontSize: f64(10)})
	tests := []struct {
		name string
		a, b InlineSpan
		want bool
	}{
		{"equal", tsStyle("a", small, ts("b")), tsStyle("a", NewTextStyle(TextStyleOpts{FontSize: f64(10)}), ts("b")), true},
		{"text", ts("a"), ts("b"), false},
		{"child", ts("a", ts("b")), ts("a", ts("c")), false},
		{"style", tsStyle("a", small), ts("a"), false},
		{"semantics label", NewTextSpan(TextSpanOpts{Text: foundation.Some("a"), SemanticsLabel: foundation.Some("x")}), ts("a"), false},
		{"other kind", ts("a"), newTestPlaceholder(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTextSpanSemanticsInformation(t *testing.T) {
	loc := ui.NewLocale("fr")
	root := NewTextSpan(TextSpanOpts{
		Text: foundation.Some("ab"), Locale: foundation.Some(loc),
		Children: []InlineSpan{
			NewTextSpan(TextSpanOpts{Text: foundation.Some("cd"), SpellOut: foundation.Some(true)}),
			newTestPlaceholder(),
			NewTextSpan(TextSpanOpts{Text: foundation.Some("ef"), SemanticsLabel: foundation.Some("EF"), SemanticsIdentifier: foundation.Some("id")}),
		},
	})
	infos := InlineSpanGetSemanticsInformation(root)
	if len(infos) != 4 {
		t.Fatalf("got %d infos: %v", len(infos), infos)
	}
	if infos[0].Text != "ab" || len(infos[0].StringAttributes) != 1 {
		t.Errorf("first = %v (%d attributes)", infos[0], len(infos[0].StringAttributes))
	}
	// The child inherits the locale and adds spell out.
	if infos[1].Text != "cd" || len(infos[1].StringAttributes) != 2 {
		t.Errorf("second = %v (%d attributes)", infos[1], len(infos[1].StringAttributes))
	}
	if !infos[2].IsPlaceholder || !infos[2].RequiresOwnNode {
		t.Errorf("third = %v", infos[2])
	}
	if !infos[3].RequiresOwnNode || infos[3].SemanticsLabel != "EF" {
		t.Errorf("fourth = %v", infos[3])
	}

	combined := CombineSemanticsInfo(infos)
	// ab+cd, then the placeholder, then the node with the identifier, with an
	// empty group between and after.
	var texts []string
	for _, c := range combined {
		texts = append(texts, c.Text)
	}
	if got, want := strings.Join(texts, "|"), "abcd|￼||ef|"; got != want {
		t.Errorf("combined texts = %q, want %q", got, want)
	}
	// The attributes of the second span shift past the label of the first.
	attrs := combined[0].StringAttributes
	if len(attrs) != 3 {
		t.Fatalf("combined attributes = %d", len(attrs))
	}
	if r := attrs[2].Range(); r.Start != 2 || r.End != 4 {
		t.Errorf("shifted range = %v, want 2..4", r)
	}
}

func TestTextSpanDiagnostics(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds print only short names")
	}
	tests := []struct {
		name string
		span *TextSpan
		want string
	}{
		{"empty", NewTextSpan(), "TextSpan:\n  (empty)"},
		{"text", ts("hello"), `TextSpan:\n  "hello"`},
	}
	for _, tt := range tests {
		got := foundation.ToDiagnosticsNode(tt.span).ToStringDeep()
		want := strings.ReplaceAll(tt.want, `\n`, "\n")
		if !strings.HasPrefix(got, want) {
			t.Errorf("%s: got %q, want prefix %q", tt.name, got, want)
		}
	}
}

func TestTextSpanVisit(t *testing.T) {
	root := ts("a", ts("b", ts("c")), ts("d"))
	var seen []string
	root.VisitChildren(func(s InlineSpan) bool {
		txt, _ := s.(*TextSpan).Text.Get()
		seen = append(seen, txt)
		return txt != "c"
	})
	if got := strings.Join(seen, ""); got != "abc" {
		t.Errorf("visit stopped at %q, want abc", got)
	}
	seen = nil
	root.VisitDirectChildren(func(s InlineSpan) bool {
		txt, _ := s.(*TextSpan).Text.Get()
		seen = append(seen, txt)
		return true
	})
	if got := strings.Join(seen, ""); got != "bd" {
		t.Errorf("direct children = %q, want bd", got)
	}
}

// newFakeRecognizer makes a recognizer that is only compared by identity.
func newFakeRecognizer() gestures.GestureRecognizer { return gestures.NewTapGestureRecognizer(nil) }

func TestTextSpanEvents(t *testing.T) {
	pt := newPaintingTester(t)
	gb := gestures.NewGestureBinding(pt.d, pt.sched)
	rec := gestures.NewTapGestureRecognizer(gb)
	rec.OnTap = func() {}
	span := NewTextSpan(TextSpanOpts{Text: foundation.Some("tap"), Recognizer: rec})
	span.HandleEvent(gestures.NewPointerMoveEvent(gestures.PointerEventOpts{Pointer: 1}), nil)
	if rec.IsTrackingPointer(1) {
		t.Error("only a pointer down is given to the recognizer")
	}
	span.HandleEvent(gestures.NewPointerDownEvent(gestures.PointerEventOpts{Pointer: 1, Buttons: foundation.Some(gestures.PrimaryButton)}), nil)
	if !rec.IsTrackingPointer(1) {
		t.Error("a pointer down starts the recognizer")
	}
	rec.Dispose()
	// Without a recognizer the event is ignored.
	ts("x").HandleEvent(gestures.NewPointerDownEvent(gestures.PointerEventOpts{Pointer: 2}), nil)
}

func TestTextSpanMouseAnnotation(t *testing.T) {
	entered := 0
	span := NewTextSpan(TextSpanOpts{Text: foundation.Some("x"), OnEnter: func(*gestures.PointerEnterEvent) { entered++ }})
	if span.OnEnter() == nil || span.OnExit() != nil || !span.ValidForMouseTracker() {
		t.Error("annotation callbacks")
	}
	span.OnEnter()(nil)
	if entered != 1 {
		t.Error("the callback is the one given")
	}
	if span.Cursor() != services.MouseCursorDefer {
		t.Errorf("cursor = %v", span.Cursor())
	}
	if span.Equal(NewTextSpan(TextSpanOpts{Text: foundation.Some("x"), OnEnter: func(*gestures.PointerEnterEvent) {}})) {
		t.Error("spans with callbacks are not equal to others")
	}
	if got := foundation.DiagnosticableString(span); foundation.DebugMode && !strings.Contains(got, "callbacks: [enter]") {
		t.Errorf("String = %q", got)
	}
}

func TestTextSpanBuildAppliesStyles(t *testing.T) {
	engine := bundledEngine(t)
	width := func(span InlineSpan) float64 {
		p := NewTextPainter(TextPainterOpts{TextEngine: engine, Text: span, TextDirection: ui.TextDirectionLTR})
		p.Layout(0, math.Inf(1))
		return p.Width()
	}
	base := robotoStyle(10)
	big := robotoStyle(20)
	small := width(tsStyle("abcd", base))
	mixed := width(tsStyle("ab", base, tsStyle("cd", big)))
	large := width(tsStyle("abcd", big))
	if !(small < mixed && mixed < large) {
		t.Errorf("widths small %v mixed %v large %v", small, mixed, large)
	}
	// The nested style is merged onto the style of the parent by the paragraph builder.
	inherit := width(tsStyle("ab", big, ts("cd")))
	if !approx(inherit, large, 0.01) {
		t.Errorf("a child without a style inherits the parent's: %v vs %v", inherit, large)
	}
	// Invalid UTF-8 is replaced instead of failing.
	if w := width(tsStyle("a\xffb", base)); w <= 0 {
		t.Errorf("width of a text with invalid UTF-8 = %v", w)
	}
}
