package ui

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestTextEnumStrings(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{FontStyleItalic.String(), "FontStyle.italic"},
		{TextAlignJustify.String(), "TextAlign.justify"},
		{TextBaselineIdeographic.String(), "TextBaseline.ideographic"},
		{TextDecorationStyleWavy.String(), "TextDecorationStyle.wavy"},
		{TextLeadingDistributionEven.String(), "TextLeadingDistribution.even"},
		{TextDirectionRTL.String(), "TextDirection.rtl"},
		{TextDirection(9).String(), "TextDirection(9)"},
		{FontWeightW700.String(), "FontWeight.w700"},
		{FontWeight(650).String(), "FontWeight(650)"},
		{TextDecorationNone.String(), "TextDecoration.none"},
		{TextDecorationUnderline.String(), "TextDecoration.underline"},
		{CombineTextDecorations(TextDecorationUnderline, TextDecorationLineThrough).String(), "TextDecoration.combine([underline, lineThrough])"},
		{NewLocale("en", "US").String(), "en_US"},
		{NewLocale("zh").LanguageTag(), "zh"},
		{Locale{"zh", "Hans", "CN"}.LanguageTag(), "zh-Hans-CN"},
		{NewFontFeature("liga", 1).String(), "FontFeature(liga, 1)"},
		{NewFontVariation("wght", 700).String(), "FontVariation(wght, 700)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestTextDecorationContains(t *testing.T) {
	both := CombineTextDecorations(TextDecorationUnderline, TextDecorationOverline)
	tests := []struct {
		d, other TextDecoration
		want     bool
	}{
		{both, TextDecorationUnderline, true},
		{both, TextDecorationLineThrough, false},
		{TextDecorationUnderline, both, false},
		{TextDecorationNone, TextDecorationNone, true},
		{both, TextDecorationNone, true},
	}
	for _, tt := range tests {
		if got := tt.d.Contains(tt.other); got != tt.want {
			t.Errorf("%v.Contains(%v) = %v, want %v", tt.d, tt.other, got, tt.want)
		}
	}
}

func TestShadow(t *testing.T) {
	if s := NewShadow(); s.Color != NewColor(0xFF000000) || s.BlurRadius != 0 || s.BlurSigma() != 0 {
		t.Errorf("default shadow = %+v", s)
	}
	red := NewColor(0xFFFF0000)
	s := NewShadow(ShadowOpts{Color: &red, Offset: Offset{1, 2}, BlurRadius: 4})
	if s.Color != red || s.Offset != (Offset{1, 2}) || !(s.BlurSigma() > 2.8 && s.BlurSigma() < 2.82) {
		t.Errorf("shadow = %+v sigma %v", s, s.BlurSigma())
	}
}

func TestTextStyleResolve(t *testing.T) {
	red, blue := NewColor(0xFFFF0000), NewColor(0xFF0000FF)
	base := DefaultResolvedTextStyle()
	if base.FontSize != 14 || base.FontWeight != FontWeightW400 || base.Color != NewColor(0xFF000000) || base.HasHeight {
		t.Fatalf("defaults = %+v", base)
	}
	first := NewTextStyle(TextStyleOpts{Color: &red, FontSize: Ptr(20.0), FontFamily: "Roboto", FontFamilyFallback: []string{"Noto"}, Height: Ptr(1.5)}).Resolve(base)
	// A style that sets nothing changes nothing; one that sets a field replaces only it.
	same := NewTextStyle().Resolve(first)
	if !sameStyle(same, first) {
		t.Errorf("an empty style changed the style: %+v", same)
	}
	second := NewTextStyle(TextStyleOpts{FontWeight: FontWeightBold, Color: &blue}).Resolve(first)
	tests := []struct {
		name string
		ok   bool
	}{
		{"color replaced", second.Color == blue},
		{"weight replaced", second.FontWeight == FontWeightBold},
		{"size inherited", second.FontSize == 20},
		{"family inherited", len(second.FontFamilies) == 2 && second.FontFamilies[0] == "Roboto" && second.FontFamilies[1] == "Noto"},
		{"height inherited", second.HasHeight && second.Height == 1.5},
		{"the parent is untouched", first.Color == red && first.FontWeight == FontWeightW400},
	}
	for _, tt := range tests {
		if !tt.ok {
			t.Errorf("%s: %+v", tt.name, second)
		}
	}
	// A foreground replaces the color, and a color replaces a foreground.
	paint := NewPaint()
	fg := NewTextStyle(TextStyleOpts{Foreground: paint}).Resolve(first)
	if fg.Foreground == nil || fg.Foreground == paint {
		t.Errorf("foreground = %v: want a copy of the paint", fg.Foreground)
	}
	if back := NewTextStyle(TextStyleOpts{Color: &blue}).Resolve(fg); back.Foreground != nil || back.Color != blue {
		t.Errorf("a color should replace the foreground: %+v", back)
	}
	// Decoration color defaults to the text color until it is set.
	deco := NewTextStyle(TextStyleOpts{Decoration: Ptr(TextDecorationUnderline)}).Resolve(first)
	if deco.HasDecorationColor || deco.Decoration != TextDecorationUnderline {
		t.Errorf("decoration = %+v", deco)
	}
	decoRed := NewTextStyle(TextStyleOpts{DecorationColor: &red}).Resolve(deco)
	if !decoRed.HasDecorationColor || decoRed.DecorationColor != red {
		t.Errorf("decoration color = %+v", decoRed)
	}
}

func TestTextStyleCopiesItsSlices(t *testing.T) {
	fallback := []string{"a", "b"}
	s := NewTextStyle(TextStyleOpts{FontFamilyFallback: fallback, Shadows: []Shadow{{}}, FontFeatures: []FontFeature{{"liga", 1}}, FontVariations: []FontVariation{{"wght", 1}}})
	fallback[0] = "changed"
	if got := s.Options().FontFamilyFallback[0]; got != "a" {
		t.Errorf("the style shares the caller's slice: %q", got)
	}
	s.Options().FontFamilyFallback[1] = "changed"
	if got := s.Options().FontFamilyFallback[1]; got != "b" {
		t.Errorf("Options shares the style's slice: %q", got)
	}
}

func TestParagraphStyleAccessors(t *testing.T) {
	var none *ParagraphStyle
	if none.Direction() != TextDirectionLTR || none.Align() != TextAlignStart || none.Ellipsis() != "" {
		t.Error("the nil style is not the default")
	}
	if _, ok := none.MaxLines(); ok {
		t.Error("the nil style has a line limit")
	}
	s := NewParagraphStyle(ParagraphStyleOpts{
		TextAlign: TextAlignCenter, TextDirection: TextDirectionRTL, MaxLines: Ptr(3), Ellipsis: "…",
		FontFamily: "Roboto", FontSize: Ptr(18.0), Height: Ptr(2.0), FontWeight: FontWeightBold, FontStyle: FontStyleItalic,
		TextHeightBehavior: &TextHeightBehavior{LeadingDistribution: TextLeadingDistributionEven},
		Locale:             &Locale{LanguageCode: "ar"},
	})
	if s.Direction() != TextDirectionRTL || s.Align() != TextAlignCenter || s.Ellipsis() != "…" {
		t.Error("accessors do not return the options")
	}
	if n, ok := s.MaxLines(); !ok || n != 3 {
		t.Errorf("MaxLines = %d, %v", n, ok)
	}
	base := s.BaseTextStyle()
	if base.FontSize != 18 || base.FontWeight != FontWeightBold || base.FontStyle != FontStyleItalic || !base.HasHeight || base.Height != 2 ||
		len(base.FontFamilies) != 1 || base.FontFamilies[0] != "Roboto" || base.Locale.LanguageCode != "ar" ||
		base.LeadingDistribution != TextLeadingDistributionEven {
		t.Errorf("base style = %+v", base)
	}
	if hb := s.HeightBehavior(); hb.ApplyHeightToFirstAscent || hb.ApplyHeightToLastDescent {
		// A literal behavior with the bools left false turns both off, as written.
		t.Logf("explicit behavior: %v", hb)
	}
	if hb := NewTextHeightBehavior(); !hb.ApplyHeightToFirstAscent || !hb.ApplyHeightToLastDescent || hb.LeadingDistribution != TextLeadingDistributionUnspecified {
		t.Errorf("default behavior = %+v", hb)
	}
	if hb := none.HeightBehavior(); !hb.ApplyHeightToFirstAscent || hb.LeadingDistribution != TextLeadingDistributionProportional {
		t.Errorf("default paragraph behavior = %+v", hb)
	}
	off := NewTextHeightBehavior(TextHeightBehaviorOpts{ApplyHeightToFirstAscent: Ptr(false)})
	if off.ApplyHeightToFirstAscent || !off.ApplyHeightToLastDescent {
		t.Errorf("behavior = %+v", off)
	}
}

func TestStrutResolve(t *testing.T) {
	if got := NewParagraphStyle().Strut(); got.Enabled {
		t.Error("a paragraph without a strut has one")
	}
	if got := NewParagraphStyle(ParagraphStyleOpts{StrutStyle: NewStrutStyle()}).Strut(); got.Enabled {
		t.Error("an empty strut style is enabled")
	}
	s := NewParagraphStyle(ParagraphStyleOpts{StrutStyle: NewStrutStyle(StrutStyleOpts{
		FontFamily: "Roboto", FontFamilyFallback: []string{"X"}, FontSize: Ptr(20.0), Height: Ptr(1.2), Leading: Ptr(0.5),
		LeadingDistribution: TextLeadingDistributionEven, FontWeight: FontWeightW300, ForceStrutHeight: Ptr(true),
	})}).Strut()
	if !s.Enabled || s.FontSize != 20 || !s.HasHeight || s.Height != 1.2 || s.Leading != 0.5 || !s.Force || s.FontWeight != FontWeightW300 ||
		s.LeadingDistribution != TextLeadingDistributionEven || len(s.FontFamilies) != 2 || s.FontStyle != FontStyleNormal {
		t.Errorf("strut = %+v", s)
	}
	d := NewParagraphStyle(ParagraphStyleOpts{StrutStyle: NewStrutStyle(StrutStyleOpts{FontFamily: "Roboto"})}).Strut()
	if !d.Enabled || d.FontSize != 14 || d.FontWeight != FontWeightW400 || d.Leading != -1 || d.Force || d.HasHeight {
		t.Errorf("strut defaults = %+v", d)
	}
}

// recordingFactory is a ParagraphFactory that keeps the content it was given.
type recordingFactory struct{ got *ParagraphContent }

type nopLayout struct{ ParagraphLayout }

func (f *recordingFactory) BuildParagraph(c *ParagraphContent) ParagraphLayout {
	f.got = c
	return nopLayout{}
}

func TestParagraphBuilder(t *testing.T) {
	f := new(recordingFactory)
	red := NewColor(0xFFFF0000)
	b := NewParagraphBuilder(f, NewParagraphStyle(ParagraphStyleOpts{FontSize: Ptr(10.0)}))
	b.AddText("one ")
	b.AddText("")
	b.PushStyle(NewTextStyle(TextStyleOpts{Color: &red}))
	b.AddText("two ")
	b.AddText("three ")
	b.Pop()
	b.AddText("four")
	b.PushStyle(NewTextStyle(TextStyleOpts{}))
	b.AddText("!")
	if b.Build() == nil {
		t.Fatal("no paragraph")
	}
	c := f.got
	if c.Text != "one two three four!" {
		t.Fatalf("text = %q", c.Text)
	}
	wantRanges := [][2]int{{0, 4}, {4, 14}, {14, 19}}
	if len(c.Runs) != len(wantRanges) {
		t.Fatalf("%d runs: %+v", len(c.Runs), c.Runs)
	}
	for i, w := range wantRanges {
		if c.Runs[i].Start != w[0] || c.Runs[i].End != w[1] {
			t.Errorf("run %d = [%d,%d), want %v", i, c.Runs[i].Start, c.Runs[i].End, w)
		}
	}
	if c.Runs[0].Style.FontSize != 10 || c.Runs[1].Style.Color != red || c.Runs[2].Style.Color == red {
		t.Errorf("run styles: %+v", c.Runs)
	}
	// A style pushed with nothing set is the same style: the last two runs would merge, but a
	// pop in between made them different from run 1.
	empty := new(recordingFactory)
	NewParagraphBuilder(empty, nil).Build()
	if empty.got.Text != "" || len(empty.got.Runs) != 0 {
		t.Errorf("empty content = %+v", empty.got)
	}
}

func TestTextPanics(t *testing.T) {
	red := NewColor(0xFFFF0000)
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"weight 0", func() { NewFontWeight(0) }, "FontWeightInvalid", "NewFontWeight.FontWeightInvalid"},
		{"weight 1001", func() { NewFontWeight(1001) }, "FontWeightInvalid", "NewFontWeight.FontWeightInvalid"},
		{"feature tag", func() { NewFontFeature("lig", 1) }, "FontFeatureInvalid", "NewFontFeature.FontFeatureInvalid"},
		{"feature value", func() { NewFontFeature("liga", -1) }, "FontFeatureInvalid", "NewFontFeature.FontFeatureInvalid"},
		{"variation tag", func() { NewFontVariation("weight", 1) }, "FontVariationInvalid", "NewFontVariation.FontVariationInvalid"},
		{"variation value", func() { NewFontVariation("wght", 40000) }, "FontVariationInvalid", "NewFontVariation.FontVariationInvalid"},
		{"shadow blur", func() { NewShadow(ShadowOpts{BlurRadius: -1}) }, "ShadowBlurRadiusNegative", "NewShadow.ShadowBlurRadiusNegative"},
		{"color and foreground", func() { NewTextStyle(TextStyleOpts{Color: &red, Foreground: NewPaint()}) }, "TextStyleColorAndForeground", "NewTextStyle.TextStyleColorAndForeground"},
		{"max lines", func() { NewParagraphStyle(ParagraphStyleOpts{MaxLines: Ptr(-1)}) }, "MaxLinesNegative", "NewParagraphStyle.MaxLinesNegative"},
		{"tab size negative", func() { NewParagraphStyle(ParagraphStyleOpts{TabSize: -1}) }, "TabSizeInvalid", "NewParagraphStyle.TabSizeInvalid"},
		{"tab size NaN", func() { NewParagraphStyle(ParagraphStyleOpts{TabSize: math.NaN()}) }, "TabSizeInvalid", "NewParagraphStyle.TabSizeInvalid"},
		{"tab size infinite", func() { NewParagraphStyle(ParagraphStyleOpts{TabSize: math.Inf(1)}) }, "TabSizeInvalid", "NewParagraphStyle.TabSizeInvalid"},
		{"pop with nothing pushed", func() { NewParagraphBuilder(new(recordingFactory), nil).Pop() }, "ParagraphBuilderPopEmpty", "ParagraphBuilder.Pop.ParagraphBuilderPopEmpty"},
		{"pop once too many", func() {
			b := NewParagraphBuilder(new(recordingFactory), nil)
			b.PushStyle(NewTextStyle())
			b.Pop()
			b.Pop()
		}, "ParagraphBuilderPopEmpty", "ParagraphBuilder.Pop.ParagraphBuilderPopEmpty"},
		{"two text style options", func() { NewTextStyle(TextStyleOpts{}, TextStyleOpts{}) }, "MultipleOptions", "NewTextStyle->oneOpts[...]->First[...].MultipleOptions"},
		{"two paragraph style options", func() { NewParagraphStyle(ParagraphStyleOpts{}, ParagraphStyleOpts{}) }, "MultipleOptions", "NewParagraphStyle->oneOpts[...]->First[...].MultipleOptions"},
		{"two strut style options", func() { NewStrutStyle(StrutStyleOpts{}, StrutStyleOpts{}) }, "MultipleOptions", "NewStrutStyle->oneOpts[...]->First[...].MultipleOptions"},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: no panic", tt.name)
					return
				}
				appErr, ok := errors.FromError(recoverError(t, r))
				if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
					t.Errorf("%s: got %v, want %s via %s", tt.name, appErr, tt.tag, tt.route)
				}
			}()
			tt.do()
		}()
	}
}

func TestTextRange(t *testing.T) {
	r := NewTextRange(2, 5)
	if !r.IsValid() || r.IsCollapsed() || !r.IsNormalized() {
		t.Errorf("range %v", r)
	}
	text := "0123456789"
	if r.TextBefore(text) != "01" || r.TextInside(text) != "234" || r.TextAfter(text) != "56789" {
		t.Errorf("before %q inside %q after %q", r.TextBefore(text), r.TextInside(text), r.TextAfter(text))
	}
	if c := NewTextRangeCollapsed(3); !c.IsCollapsed() || c.TextInside(text) != "" {
		t.Errorf("collapsed = %v", c)
	}
	if TextRangeEmpty.IsValid() || TextRangeEmpty.String() != "TextRange(start: -1, end: -1)" {
		t.Errorf("empty = %v", TextRangeEmpty)
	}
	if NewTextRange(5, 2).IsNormalized() {
		t.Error("a backwards range is normalized")
	}
}

func TestTextPositionAndBox(t *testing.T) {
	if p := NewTextPosition(3); p.Affinity != TextAffinityDownstream || p.String() != "TextPosition(offset: 3, affinity: TextAffinity.downstream)" {
		t.Errorf("position = %v", p)
	}
	// The unspecified affinity is downstream.
	if !(TextPosition{Offset: 3}).Equal(NewTextPosition(3)) || NewTextPosition(3, TextPositionOpts{Affinity: TextAffinityUpstream}).Equal(NewTextPosition(3)) {
		t.Error("Equal does not treat the unspecified affinity as downstream")
	}
	ltr := NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionLTR)
	rtl := NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionRTL)
	if ltr.Start() != 1 || ltr.End() != 3 || rtl.Start() != 3 || rtl.End() != 1 || ltr.ToRect() != (Rect{1, 2, 3, 4}) {
		t.Errorf("boxes %v %v", ltr, rtl)
	}
	for got, want := range map[string]string{
		BoxHeightStyleIncludeLineSpacingTop.String(): "BoxHeightStyle.includeLineSpacingTop",
		BoxWidthStyleMax.String():                    "BoxWidthStyle.max",
		PlaceholderAlignmentAboveBaseline.String():   "PlaceholderAlignment.aboveBaseline",
		TextAffinityUpstream.String():                "TextAffinity.upstream",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestParagraphBuilderPlaceholders(t *testing.T) {
	f := new(recordingFactory)
	b := NewParagraphBuilder(f, nil)
	b.AddText("ab")
	b.AddPlaceholder(10, 20, PlaceholderAlignmentUnspecified)
	b.AddPlaceholder(5, 6, PlaceholderAlignmentTop, AddPlaceholderOpts{Scale: Ptr(2.0), BaselineOffset: Ptr(3.0), Baseline: TextBaselineIdeographic})
	if b.PlaceholderCount() != 2 || b.PlaceholderScales()[0] != 1 || b.PlaceholderScales()[1] != 2 {
		t.Fatalf("count %d scales %v", b.PlaceholderCount(), b.PlaceholderScales())
	}
	b.Build()
	c := f.got
	if c.Text != "ab￼￼" || len(c.Placeholders) != 2 {
		t.Fatalf("content = %+v", c)
	}
	first, second := c.Placeholders[0], c.Placeholders[1]
	if first.Offset != 2 || first.Alignment != PlaceholderAlignmentBaseline || first.BaselineOffset != 20 || first.Baseline != TextBaselineAlphabetic || first.Scale != 1 {
		t.Errorf("first = %+v", first)
	}
	if second.Offset != 5 || second.BaselineOffset != 3 || second.Baseline != TextBaselineIdeographic || second.Scale != 2 {
		t.Errorf("second = %+v", second)
	}
}

func TestTextQueryPanics(t *testing.T) {
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"range below -1", func() { NewTextRange(-2, 3) }, "TextRangeInvalid", "NewTextRange.TextRangeInvalid"},
		{"range not normalized", func() { NewTextRange(5, 2).TextInside("0123456789") }, "TextRangeOutOfBounds", "TextRange.TextInside->TextRange.checked.TextRangeOutOfBounds"},
		{"range outside the text", func() { NewTextRange(2, 50).TextAfter("0123") }, "TextRangeOutOfBounds", "TextRange.TextAfter->TextRange.checked.TextRangeOutOfBounds"},
		{"negative placeholder", func() { NewParagraphBuilder(new(recordingFactory), nil).AddPlaceholder(1, -1, PlaceholderAlignmentTop) }, "PlaceholderSizeInvalid", "ParagraphBuilder.AddPlaceholder.PlaceholderSizeInvalid"},
		{"two position options", func() { NewTextPosition(1, TextPositionOpts{}, TextPositionOpts{}) }, "MultipleOptions", "NewTextPosition->oneOpts[...]->First[...].MultipleOptions"},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: no panic", tt.name)
					return
				}
				appErr, ok := errors.FromError(recoverError(t, r))
				if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
					t.Errorf("%s: got %v, want %s via %s", tt.name, appErr, tt.tag, tt.route)
				}
			}()
			tt.do()
		}()
	}
}
