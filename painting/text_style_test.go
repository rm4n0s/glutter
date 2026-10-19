package painting

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func f64(v float64) foundation.Opt[float64] { return foundation.Some(v) }

func col(v uint32) foundation.Opt[ui.Color] { return foundation.Some(ui.NewColor(v)) }

func TestTextStyleCopyWith(t *testing.T) {
	base := NewTextStyle(TextStyleOpts{
		Color: col(0xFF123456), FontSize: f64(12), FontWeight: ui.FontWeightW700, FontFamily: "Roboto",
		FontFamilyFallback: []string{"Noto"}, Package: "pkg",
	})
	tests := []struct {
		name  string
		copy  TextStyleCopy
		check func(t *testing.T, s *TextStyle)
	}{
		{"nothing keeps everything", TextStyleCopy{}, func(t *testing.T, s *TextStyle) {
			if !s.Equal(base) {
				t.Errorf("copy %v != base %v", s, base)
			}
		}},
		{"font size", TextStyleCopy{FontSize: f64(20)}, func(t *testing.T, s *TextStyle) {
			if s.FontSize.Value() != 20 || s.FontWeight != ui.FontWeightW700 {
				t.Errorf("got %v", s)
			}
		}},
		{"family keeps the package prefix", TextStyleCopy{FontFamily: "Other"}, func(t *testing.T, s *TextStyle) {
			if s.FontFamily != "packages/pkg/Other" {
				t.Errorf("family = %q", s.FontFamily)
			}
		}},
		{"fallback keeps the package prefix", TextStyleCopy{}, func(t *testing.T, s *TextStyle) {
			if got := s.FontFamilyFallback(); !reflect.DeepEqual(got, []string{"packages/pkg/Noto"}) {
				t.Errorf("fallback = %v", got)
			}
		}},
		{"foreground drops the color", TextStyleCopy{Foreground: ui.NewPaint()}, func(t *testing.T, s *TextStyle) {
			if s.Color.IsSome() || s.Foreground == nil {
				t.Errorf("color %v foreground %v", s.Color, s.Foreground)
			}
		}},
		{"inherit", TextStyleCopy{Inherit: foundation.Some(false)}, func(t *testing.T, s *TextStyle) {
			if s.Inherit {
				t.Error("inherit should be false")
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.check(t, base.CopyWith(tt.copy)) })
	}
}

func TestTextStyleApply(t *testing.T) {
	base := NewTextStyle(TextStyleOpts{
		Color: col(0xFF000000), FontSize: f64(10), FontWeight: ui.FontWeightW400, LetterSpacing: f64(2), WordSpacing: f64(3),
		Height: f64(1.5), DecorationThickness: f64(2),
	})
	got := base.Apply(TextStyleApply{
		Color: col(0xFFFF0000), FontSizeFactor: f64(2), FontSizeDelta: 1, FontWeightDelta: 2, LetterSpacingFactor: f64(0.5),
		LetterSpacingDelta: 1, WordSpacingDelta: -1, HeightFactor: f64(2), HeightDelta: 0.5,
		DecorationThicknessFactor: f64(3), DecorationThicknessDelta: 1,
	})
	tests := []struct {
		name      string
		got, want any
	}{
		{"color", got.Color.Value(), ui.NewColor(0xFFFF0000)},
		{"fontSize", got.FontSize.Value(), 21.0},
		{"fontWeight", got.FontWeight, ui.FontWeightW600},
		{"letterSpacing", got.LetterSpacing.Value(), 2.0},
		{"wordSpacing", got.WordSpacing.Value(), 2.0},
		{"height", got.Height.Value(), 3.5},
		{"decorationThickness", got.DecorationThickness.Value(), 7.0},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	// The weight stays within w100..w900.
	for _, tt := range []struct {
		delta int
		want  ui.FontWeight
	}{{-10, ui.FontWeightW100}, {10, ui.FontWeightW900}, {0, ui.FontWeightW400}} {
		if w := base.Apply(TextStyleApply{FontWeightDelta: tt.delta}).FontWeight; w != tt.want {
			t.Errorf("delta %d: weight %v, want %v", tt.delta, w, tt.want)
		}
	}
	// A height of kTextHeightNone is left as it is.
	none := NewTextStyle(TextStyleOpts{Height: f64(ui.TextHeightNone)})
	if h := none.Apply(TextStyleApply{HeightFactor: f64(2), HeightDelta: 1}).Height.Value(); h != ui.TextHeightNone {
		t.Errorf("height = %v, want none", h)
	}
	// A foreground keeps the paint and ignores the color.
	fg := ui.NewPaint()
	fgStyle := NewTextStyle(TextStyleOpts{Foreground: fg}).Apply(TextStyleApply{Color: col(0xFF00FF00)})
	if fgStyle.Foreground != fg || fgStyle.Color.IsSome() {
		t.Errorf("foreground style = %v", fgStyle)
	}
}

func TestTextStyleMerge(t *testing.T) {
	a := NewTextStyle(TextStyleOpts{Color: col(0xFF111111), FontSize: f64(10), FontFamily: "A"})
	tests := []struct {
		name  string
		other *TextStyle
		check func(t *testing.T, got *TextStyle)
	}{
		{"nil", nil, func(t *testing.T, got *TextStyle) {
			if got != a {
				t.Error("merging nil must return the receiver")
			}
		}},
		{"does not inherit", NewTextStyle(TextStyleOpts{Inherit: foundation.Some(false), FontSize: f64(99)}), func(t *testing.T, got *TextStyle) {
			if got.FontSize.Value() != 99 || got.Color.IsSome() || got.Inherit {
				t.Errorf("got %v", got)
			}
		}},
		{"overrides what it sets", NewTextStyle(TextStyleOpts{FontSize: f64(20), FontWeight: ui.FontWeightW900}), func(t *testing.T, got *TextStyle) {
			if got.FontSize.Value() != 20 || got.FontWeight != ui.FontWeightW900 || got.Color.Value() != ui.NewColor(0xFF111111) || got.FontFamily != "A" {
				t.Errorf("got %v", got)
			}
		}},
		{"foreground replaces the color", NewTextStyle(TextStyleOpts{Foreground: ui.NewPaint()}), func(t *testing.T, got *TextStyle) {
			if got.Foreground == nil || got.Color.IsSome() {
				t.Errorf("got %v", got)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.check(t, a.Merge(tt.other)) })
	}
}

// lerpedGray is the color between black and white at t.
func lerpedGray(t float64) foundation.Opt[ui.Color] {
	black, white := ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)
	return foundation.Some(*ui.ColorLerp(&black, &white, t))
}

func TestTextStyleLerp(t *testing.T) {
	a := NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), FontWeight: ui.FontWeightW100, FontStyle: ui.FontStyleNormal,
		LetterSpacing: f64(0), Height: f64(1), FontFamily: "A"})
	b := NewTextStyle(TextStyleOpts{Color: col(0xFFFFFFFF), FontSize: f64(30), FontWeight: ui.FontWeightW900, FontStyle: ui.FontStyleItalic,
		LetterSpacing: f64(4), Height: f64(2), FontFamily: "B"})
	tests := []struct {
		name string
		t    float64
		want TextStyleOpts
	}{
		{"start", 0, TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), FontWeight: ui.FontWeightW100, FontStyle: ui.FontStyleNormal,
			LetterSpacing: f64(0), Height: f64(1), FontFamily: "A"}},
		{"quarter", 0.25, TextStyleOpts{Color: lerpedGray(0.25), FontSize: f64(15), FontWeight: ui.FontWeightW300, FontStyle: ui.FontStyleNormal,
			LetterSpacing: f64(1), Height: f64(1.25), FontFamily: "A"}},
		{"half", 0.5, TextStyleOpts{Color: lerpedGray(0.5), FontSize: f64(20), FontWeight: ui.FontWeightW500, FontStyle: ui.FontStyleItalic,
			LetterSpacing: f64(2), Height: f64(1.5), FontFamily: "B"}},
		{"end", 1, TextStyleOpts{Color: col(0xFFFFFFFF), FontSize: f64(30), FontWeight: ui.FontWeightW900, FontStyle: ui.FontStyleItalic,
			LetterSpacing: f64(4), Height: f64(2), FontFamily: "B"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextStyleLerp(a, b, tt.t)
			want := NewTextStyle(tt.want)
			if got.CompareTo(want) != RenderComparisonIdentical {
				t.Errorf("lerp(%v) = %v, want %v", tt.t, got, want)
			}
		})
	}
}

func TestTextStyleLerpNil(t *testing.T) {
	b := NewTextStyle(TextStyleOpts{Color: col(0xFF102030), FontSize: f64(30), FontWeight: ui.FontWeightW900})
	tests := []struct {
		name         string
		a, b         *TextStyle
		t            float64
		wantSize     foundation.Opt[float64]
		wantWeight   ui.FontWeight
		wantColorSet bool
	}{
		{"from null, early", nil, b, 0.25, foundation.None[float64](), ui.FontWeight(525), true},
		{"from null, late", nil, b, 0.75, f64(30), ui.FontWeight(775), true},
		{"to null, early", b, nil, 0.25, f64(30), ui.FontWeight(775), true},
		{"to null, late", b, nil, 0.75, foundation.None[float64](), ui.FontWeight(525), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextStyleLerp(tt.a, tt.b, tt.t)
			if got.FontSize != tt.wantSize {
				t.Errorf("size = %v, want %v", got.FontSize, tt.wantSize)
			}
			if got.FontWeight != tt.wantWeight {
				t.Errorf("weight = %v, want %v", got.FontWeight, tt.wantWeight)
			}
			if got.Color.IsSome() != tt.wantColorSet {
				t.Errorf("color = %v", got.Color)
			}
		})
	}
	if TextStyleLerp(nil, nil, 0.5) != nil {
		t.Error("lerp(nil, nil) must be nil")
	}
	if TextStyleLerp(b, b, 0.5) != b {
		t.Error("lerp(a, a) must be a")
	}
}

func TestTextStyleLerpForeground(t *testing.T) {
	fg := ui.NewPaint()
	fg.SetColor(ui.NewColor(0xFF00FF00))
	a := NewTextStyle(TextStyleOpts{Foreground: fg})
	b := NewTextStyle(TextStyleOpts{Color: col(0xFFFF0000)})
	early := TextStyleLerp(a, b, 0.25)
	if early.Foreground != fg {
		t.Error("early: the foreground of a")
	}
	late := TextStyleLerp(a, b, 0.75)
	if late.Foreground == nil || late.Foreground.Color() != ui.NewColor(0xFFFF0000) {
		t.Errorf("late: foreground %v, want a paint of the color of b", late.Foreground)
	}
	if late.Color.IsSome() {
		t.Error("the color is dropped when a foreground is lerped")
	}
}

func TestLerpFontVariations(t *testing.T) {
	v := func(axis string, value float64) ui.FontVariation { return ui.NewFontVariation(axis, value) }
	tests := []struct {
		name string
		a, b []ui.FontVariation
		t    float64
		want []ui.FontVariation
	}{
		{"t=0", []ui.FontVariation{v("wght", 100)}, []ui.FontVariation{v("wght", 900)}, 0, []ui.FontVariation{v("wght", 100)}},
		{"t=1", []ui.FontVariation{v("wght", 100)}, []ui.FontVariation{v("wght", 900)}, 1, []ui.FontVariation{v("wght", 900)}},
		{"same axis", []ui.FontVariation{v("wght", 100)}, []ui.FontVariation{v("wght", 900)}, 0.5, []ui.FontVariation{v("wght", 500)}},
		{"one empty, early", nil, []ui.FontVariation{v("wght", 900)}, 0.25, nil},
		{"one empty, late", nil, []ui.FontVariation{v("wght", 900)}, 0.75, []ui.FontVariation{v("wght", 900)}},
		{"extra axis", []ui.FontVariation{v("wght", 100)}, []ui.FontVariation{v("wght", 900), v("wdth", 50)}, 0.5,
			[]ui.FontVariation{v("wght", 500), v("wdth", 50)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lerpFontVariations(tt.a, tt.b, tt.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTextStyleCompareTo(t *testing.T) {
	base := NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10)})
	tests := []struct {
		name  string
		other *TextStyle
		want  RenderComparison
	}{
		{"same", base, RenderComparisonIdentical},
		{"equal", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10)}), RenderComparisonIdentical},
		{"color", NewTextStyle(TextStyleOpts{Color: col(0xFFFFFFFF), FontSize: f64(10)}), RenderComparisonPaint},
		{"decoration", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), Decoration: foundation.Some(ui.TextDecorationUnderline)}), RenderComparisonPaint},
		{"size", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(11)}), RenderComparisonLayout},
		{"family", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), FontFamily: "X"}), RenderComparisonLayout},
		{"inherit", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), Inherit: foundation.Some(false)}), RenderComparisonLayout},
		{"overflow", NewTextStyle(TextStyleOpts{Color: col(0xFF000000), FontSize: f64(10), Overflow: TextOverflowEllipsis}), RenderComparisonLayout},
	}
	for _, tt := range tests {
		if got := base.CompareTo(tt.other); got != tt.want {
			t.Errorf("%s: CompareTo = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTextStyleGetTextStyle(t *testing.T) {
	s := NewTextStyle(TextStyleOpts{
		Color: col(0xFF112233), FontSize: f64(10), FontFamily: "F", FontFamilyFallback: []string{"G"}, Package: "p",
		BackgroundColor: col(0xFF445566), Height: f64(1.2), Decoration: foundation.Some(ui.TextDecorationNone),
	})
	o := s.GetTextStyle(GetTextStyleOpts{TextScaler: NewTextScalerLinear(2)}).Options()
	if o.FontSize == nil || *o.FontSize != 20 {
		t.Errorf("font size = %v, want 20", o.FontSize)
	}
	if o.FontFamily != "packages/p/F" || !reflect.DeepEqual(o.FontFamilyFallback, []string{"packages/p/G"}) {
		t.Errorf("families = %q %v", o.FontFamily, o.FontFamilyFallback)
	}
	if o.Background == nil || o.Background.Color() != ui.NewColor(0xFF445566) {
		t.Errorf("background = %v", o.Background)
	}
	if o.Decoration == nil || *o.Decoration != ui.TextDecorationNone {
		t.Errorf("an explicit decoration none must survive: %v", o.Decoration)
	}
	if o.Color == nil || *o.Color != ui.NewColor(0xFF112233) {
		t.Errorf("color = %v", o.Color)
	}
}

func TestTextStyleGetParagraphStyle(t *testing.T) {
	s := NewTextStyle(TextStyleOpts{FontSize: f64(10), FontWeight: ui.FontWeightW300, LeadingDistribution: ui.TextLeadingDistributionEven, Height: f64(2)})
	strut := NewStrutStyle(StrutStyleOpts{FontSize: f64(8), Leading: f64(1)})
	p := s.GetParagraphStyle(GetParagraphStyleOpts{
		TextAlign: ui.TextAlignCenter, TextDirection: ui.TextDirectionRTL, TextScaler: NewTextScalerLinear(2), MaxLines: foundation.Some(3),
		Ellipsis: "…", StrutStyle: strut,
	})
	o := p.Options()
	if o.FontSize == nil || *o.FontSize != 20 {
		t.Errorf("font size = %v, want 20", o.FontSize)
	}
	if o.TextAlign != ui.TextAlignCenter || o.TextDirection != ui.TextDirectionRTL || o.Ellipsis != "…" || o.FontWeight != ui.FontWeightW300 {
		t.Errorf("options = %+v", o)
	}
	if o.MaxLines == nil || *o.MaxLines != 3 {
		t.Errorf("max lines = %v", o.MaxLines)
	}
	if o.TextHeightBehavior == nil || o.TextHeightBehavior.String() != ui.NewTextHeightBehavior(ui.TextHeightBehaviorOpts{LeadingDistribution: ui.TextLeadingDistributionEven}).String() {
		t.Errorf("text height behavior = %v", o.TextHeightBehavior)
	}
	if so := o.StrutStyle.Options(); so.FontSize == nil || *so.FontSize != 16 {
		t.Errorf("strut font size = %v, want 16", so.FontSize)
	}
	// With no font size at all, the default font size is scaled.
	plain := NewTextStyle().GetParagraphStyle(GetParagraphStyleOpts{TextScaler: NewTextScalerLinear(2)}).Options()
	if plain.FontSize == nil || *plain.FontSize != 28 {
		t.Errorf("default font size = %v, want 28", plain.FontSize)
	}
}

func TestTextStyleDiagnostics(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds print only short names")
	}
	tests := []struct {
		name  string
		style *TextStyle
		want  string
	}{
		{"empty", NewTextStyle(), "TextStyle(<all styles inherited>)"},
		{"not inheriting", NewTextStyle(TextStyleOpts{Inherit: foundation.Some(false)}), "TextStyle(inherit: false, <no style specified>)"},
		{"some", NewTextStyle(TextStyleOpts{Color: col(0xFF123456), FontSize: f64(12), FontWeight: ui.FontWeightW700, FontFamily: "Roboto"}),
			"TextStyle(inherit: true, color: " + ui.NewColor(0xFF123456).String() + ", family: Roboto, size: 12.0, weight: 700)"},
		{"decoration", NewTextStyle(TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline), DecorationStyle: ui.TextDecorationStyleDashed, DecorationThickness: f64(2)}),
			"TextStyle(inherit: true, decoration: dashed TextDecoration.underline, decorationThickness: 2.0x)"},
		{"label", NewTextStyle(TextStyleOpts{DebugLabel: "body", FontSize: f64(3)}), "TextStyle(debugLabel: body, inherit: true, size: 3.0)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.String(); got != tt.want {
				t.Errorf("String = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextStylePackagePrefix(t *testing.T) {
	a := NewTextStyle(TextStyleOpts{FontFamily: "F", Package: "p"})
	if a.FontFamily != "packages/p/F" || a.Package() != "p" {
		t.Errorf("family %q package %q", a.FontFamily, a.Package())
	}
	b := a.CopyWith(TextStyleCopy{FontSize: f64(3)})
	if b.FontFamily != "packages/p/F" {
		t.Errorf("a copy keeps the prefix once: %q", b.FontFamily)
	}
	if !a.Equal(NewTextStyle(TextStyleOpts{FontFamily: "F", Package: "p"})) || a.Equal(NewTextStyle(TextStyleOpts{FontFamily: "F"})) {
		t.Error("the package takes part in equality")
	}
}
