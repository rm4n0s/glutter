package ui

import (
	"testing"
)

func TestFontWeightLerp(t *testing.T) {
	tests := []struct {
		name string
		a, b FontWeight
		t    float64
		want FontWeight
	}{
		{"nulls", 0, 0, 0.5, 0},
		{"both", FontWeightW100, FontWeightW900, 0.25, FontWeightW300},
		{"from null is normal", 0, FontWeightW900, 0.5, 650},
		{"to null is normal", FontWeightW100, 0, 0.5, 250},
		{"rounds", FontWeightW100, FontWeightW300, 0.333, 167},
		{"clamps high", FontWeightW900, FontWeightW900, 2, 900},
		{"clamps low", FontWeightW100, FontWeightW100, -1, 100},
	}
	for _, tt := range tests {
		if got := FontWeightLerp(tt.a, tt.b, tt.t); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFontVariationLerp(t *testing.T) {
	a, b, c := NewFontVariation("wght", 100), NewFontVariation("wght", 300), NewFontVariation("wdth", 50)
	tests := []struct {
		name string
		a, b *FontVariation
		t    float64
		want *FontVariation
	}{
		{"same axis", &a, &b, 0.5, &FontVariation{"wght", 200}},
		{"different axes, early", &a, &c, 0.25, &a},
		{"different axes, late", &a, &c, 0.75, &c},
		{"one nil, early", nil, &b, 0.25, nil},
		{"one nil, late", nil, &b, 0.75, &b},
		{"both nil", nil, nil, 0.5, nil},
	}
	for _, tt := range tests {
		got := FontVariationLerp(tt.a, tt.b, tt.t)
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestShadowLerp(t *testing.T) {
	red, blue := NewColor(0xFFFF0000), NewColor(0xFF0000FF)
	a := NewShadow(ShadowOpts{Color: &red, Offset: Offset{Dx: 2, Dy: 2}, BlurRadius: 4})
	b := NewShadow(ShadowOpts{Color: &blue, Offset: Offset{Dx: 6, Dy: 6}, BlurRadius: 8})
	mid := ShadowLerp(&a, &b, 0.5)
	if mid.Offset != (Offset{Dx: 4, Dy: 4}) || mid.BlurRadius != 6 {
		t.Errorf("mid = %+v", mid)
	}
	if got := ShadowLerp(nil, &b, 0.5); got.Offset != (Offset{Dx: 3, Dy: 3}) || got.BlurRadius != 4 {
		t.Errorf("from nil = %+v", got)
	}
	if got := ShadowLerp(&a, nil, 0.5); got.Offset != (Offset{Dx: 1, Dy: 1}) || got.BlurRadius != 2 {
		t.Errorf("to nil = %+v", got)
	}
	if ShadowLerp(nil, nil, 0.5) != nil {
		t.Error("two nils give nil")
	}
	if got := a.Scale(2); got.Offset != (Offset{Dx: 4, Dy: 4}) || got.BlurRadius != 8 || got.Color != a.Color {
		t.Errorf("Scale = %+v", got)
	}
	list := ShadowLerpList([]Shadow{a}, []Shadow{b, b}, 0.5)
	if len(list) != 2 || list[0].BlurRadius != 6 {
		t.Errorf("list = %+v", list)
	}
	if ShadowLerpList(nil, nil, 0.5) != nil {
		t.Error("two empty lists give nil")
	}
}

func TestEnumNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{FontStyleItalic.Name(), "italic"},
		{TextAlignCenter.Name(), "center"},
		{TextBaselineIdeographic.Name(), "ideographic"},
		{TextDecorationStyleWavy.Name(), "wavy"},
		{TextLeadingDistributionEven.Name(), "even"},
		{TextDirectionRTL.Name(), "rtl"},
		{TextAffinityUpstream.Name(), "upstream"},
		{PlaceholderAlignmentAboveBaseline.Name(), "aboveBaseline"},
		{BoxHeightStyleIncludeLineSpacingTop.Name(), "includeLineSpacingTop"},
		{BoxWidthStyleMax.Name(), "max"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("name = %q, want %q", tt.got, tt.want)
		}
	}
}

func TestStringAttributes(t *testing.T) {
	r := NewTextRange(1, 4)
	spell := NewSpellOutStringAttribute(r)
	loc := NewLocaleStringAttribute(r, NewLocale("fr", "CA"))
	if spell.Range() != r || loc.Range() != r || loc.Locale() != NewLocale("fr", "CA") {
		t.Error("accessors")
	}
	moved := NewTextRange(5, 8)
	if got := spell.CopyWithRange(moved); got.Range() != moved {
		t.Errorf("spell copy = %v", got.Range())
	}
	cl := loc.CopyWithRange(moved).(LocaleStringAttribute)
	if cl.Range() != moved || cl.Locale() != loc.Locale() {
		t.Errorf("locale copy = %v", cl)
	}
}

func TestExplicitDecorationNoneOverridesParent(t *testing.T) {
	under := TextDecorationUnderline
	none := TextDecorationNone
	parent := NewTextStyle(TextStyleOpts{Decoration: &under}).Resolve(DefaultResolvedTextStyle())
	if parent.Decoration != TextDecorationUnderline {
		t.Fatalf("parent decoration = %v", parent.Decoration)
	}
	if got := NewTextStyle(TextStyleOpts{Decoration: &none}).Resolve(parent); got.Decoration != TextDecorationNone {
		t.Errorf("an explicit none must clear the underline, got %v", got.Decoration)
	}
	if got := NewTextStyle().Resolve(parent); got.Decoration != TextDecorationUnderline {
		t.Errorf("an unset decoration inherits, got %v", got.Decoration)
	}
}

func TestPlatformDispatcherTextEngine(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	if d.TextEngine() != nil {
		t.Error("there is no engine at first")
	}
	var f ParagraphFactory = nopFactory{}
	d.SetTextEngine(f)
	if d.TextEngine() != f {
		t.Error("the engine is kept")
	}
}

type nopFactory struct{}

func (nopFactory) BuildParagraph(*ParagraphContent) ParagraphLayout { return nil }
