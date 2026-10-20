package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestTextThemeBasics(t *testing.T) {
	empty := NewTextTheme()
	typography := NewTypography()
	tests := []struct {
		name      string
		a, b      *TextTheme
		wantEqual bool
	}{
		{"empty copyWith", empty, empty.CopyWith(TextThemeOpts{}), true},
		{"empty apply", empty, empty.Apply(TextThemeApply{}), true},
		{"empty merge nil", empty, empty.Merge(nil), true},
		{"black copyWith", typography.Black, typography.Black.CopyWith(TextThemeOpts{}), true},
		{"black apply", typography.Black, typography.Black.Apply(TextThemeApply{}), true},
		{"black merge nil", typography.Black, typography.Black.Merge(nil), true},
		{"empty merge black", typography.Black, empty.Merge(typography.Black), true},
		{"black merge black", typography.Black, typography.Black.Merge(typography.Black), true},
		{"black merge white", typography.White, typography.Black.Merge(typography.White), true},
		{"black is not white", typography.Black, typography.White, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.wantEqual {
				t.Errorf("Equal = %v, want %v", got, tt.wantEqual)
			}
		})
	}
}

func TestTextThemeLerpSpecialCases(t *testing.T) {
	if got := TextThemeLerp(nil, nil, 0); !got.Equal(NewTextTheme()) {
		t.Errorf("lerp of nil and nil = %v, want an empty theme", got)
	}
	theme := NewTextTheme()
	if TextThemeLerp(theme, theme, 0.5) != theme {
		t.Error("lerp of identical themes is not the theme itself")
	}
}

func TestTextThemeCopyWith(t *testing.T) {
	typography := NewTypography()
	white := typography.White
	copied := typography.Black.CopyWith(TextThemeOpts(*white))
	if !white.Equal(copied) {
		t.Error("copying every style of white onto black does not give white")
	}
}

func TestTextThemeMergeWithNilStyles(t *testing.T) {
	colored := func(c uint32) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(ui.NewColor(c))})
	}
	partial := NewTextTheme(TextThemeOpts{TitleLarge: colored(0xcafefeed)})
	full := NewThemeDataFallback().TextTheme.Merge(partial)
	if got, want := full.TitleLarge.Color.Value(), ui.NewColor(0xcafefeed); got != want {
		t.Errorf("merged titleLarge color = %v, want %v", got, want)
	}
	onlyHeadlineSmallAndTitleLarge := NewTextTheme(TextThemeOpts{HeadlineSmall: colored(0xcafefeed), TitleLarge: colored(0xbeefcafe)})
	onlyBodyMediumAndTitleLarge := NewTextTheme(TextThemeOpts{BodyMedium: colored(0xfeedfeed), TitleLarge: colored(0xdeadcafe)})
	merged := onlyHeadlineSmallAndTitleLarge.Merge(onlyBodyMediumAndTitleLarge)
	if merged.BodyLarge != nil {
		t.Error("bodyLarge is not nil")
	}
	checks := []struct {
		name      string
		got, want ui.Color
	}{
		{"bodyMedium", merged.BodyMedium.Color.Value(), ui.NewColor(0xfeedfeed)},
		{"headlineSmall", merged.HeadlineSmall.Color.Value(), ui.NewColor(0xcafefeed)},
		{"titleLarge", merged.TitleLarge.Color.Value(), ui.NewColor(0xdeadcafe)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s color = %v, want %v", c.name, c.got, c.want)
		}
	}
	if !onlyHeadlineSmallAndTitleLarge.Merge(nil).Equal(onlyHeadlineSmallAndTitleLarge) {
		t.Error("merging nil changed the theme")
	}
}

func TestTextThemeApply(t *testing.T) {
	displayColor, bodyColor, decorationColor := ui.NewColor(1), ui.NewColor(2), ui.NewColor(3)
	fallback := []string{"font", "family", "fallback"}
	decoration := ui.CombineTextDecorations(ui.TextDecorationUnderline, ui.TextDecorationLineThrough)
	theme := NewTypography().Black.Apply(TextThemeApply{
		FontFamily: "fontFamily", FontFamilyFallback: fallback,
		DisplayColor: foundation.Some(displayColor), BodyColor: foundation.Some(bodyColor),
		Decoration: foundation.Some(decoration), DecorationColor: foundation.Some(decorationColor),
		DecorationStyle: ui.TextDecorationStyleDashed,
	})
	// The display color goes to displayLarge, displayMedium, displaySmall,
	// headlineLarge, headlineMedium and bodySmall; the body color to the rest.
	wantColors := []ui.Color{displayColor, displayColor, displayColor, displayColor, displayColor, bodyColor,
		bodyColor, bodyColor, bodyColor, bodyColor, bodyColor, displayColor, bodyColor, bodyColor, bodyColor}
	for i, s := range styles(theme) {
		if got := s.Color.Value(); got != wantColors[i] {
			t.Errorf("style %d: color = %v, want %v", i, got, wantColors[i])
		}
		if s.FontFamily != "fontFamily" {
			t.Errorf("style %d: font family = %q", i, s.FontFamily)
		}
		if got := s.FontFamilyFallback(); len(got) != 3 || got[0] != "font" || got[2] != "fallback" {
			t.Errorf("style %d: font family fallback = %v", i, got)
		}
		if got := s.DecorationColor.Value(); got != decorationColor {
			t.Errorf("style %d: decoration color = %v", i, got)
		}
		if s.DecorationStyle != ui.TextDecorationStyleDashed {
			t.Errorf("style %d: decoration style = %v", i, s.DecorationStyle)
		}
		if got := s.Decoration.Value(); got != decoration {
			t.Errorf("style %d: decoration = %v, want %v", i, got, decoration)
		}
	}
}

func TestTextThemeApplyFactorsAndDeltas(t *testing.T) {
	typography := NewTypography()
	base := TypographyEnglishLike2018.Merge(typography.Black)
	withWordSpacing := func(theme *TextTheme) *TextTheme {
		out := *theme
		for i, s := range styles(&out) {
			styles := []**painting.TextStyle{&out.DisplayLarge, &out.DisplayMedium, &out.DisplaySmall, &out.HeadlineLarge, &out.HeadlineMedium, &out.HeadlineSmall,
				&out.TitleLarge, &out.TitleMedium, &out.TitleSmall, &out.BodyLarge, &out.BodyMedium, &out.BodySmall, &out.LabelLarge, &out.LabelMedium, &out.LabelSmall}
			*styles[i] = s.CopyWith(painting.TextStyleCopy{WordSpacing: foundation.Some(1.0)})
		}
		return &out
	}
	base2021 := TypographyEnglishLike2021.Merge(NewTypographyMaterial2021().Black)
	tests := []struct {
		name  string
		base  *TextTheme
		apply TextThemeApply
		get   func(*painting.TextStyle) float64
	}{
		{"font size", base, TextThemeApply{FontSizeFactor: foundation.Some(2.0), FontSizeDelta: 5}, func(s *painting.TextStyle) float64 { return s.FontSize.Value() }},
		{"letter spacing", base, TextThemeApply{LetterSpacingFactor: foundation.Some(2.0), LetterSpacingDelta: 5}, func(s *painting.TextStyle) float64 { return s.LetterSpacing.Value() }},
		{"word spacing", withWordSpacing(base), TextThemeApply{WordSpacingFactor: foundation.Some(2.0), WordSpacingDelta: 5}, func(s *painting.TextStyle) float64 { return s.WordSpacing.Value() }},
		{"height", base2021, TextThemeApply{HeightFactor: foundation.Some(2.0), HeightDelta: 5}, func(s *painting.TextStyle) float64 { return s.Height.Value() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applied := tt.base.Apply(tt.apply)
			before, after := styles(tt.base), styles(applied)
			for i := range before {
				if got, want := tt.get(after[i]), tt.get(before[i])*2+5; got != want {
					t.Errorf("style %d = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestTextThemeLerpWithNullParameters(t *testing.T) {
	theme := NewTypography().Black
	tests := []struct {
		name string
		a, b *TextTheme
	}{
		{"second null", theme, nil},
		{"first null", nil, theme},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lerped := TextThemeLerp(tt.a, tt.b, 0.25)
			for i, got := range styles(lerped) {
				var sa, sb *painting.TextStyle
				if tt.a != nil {
					sa = styles(tt.a)[i]
				}
				if tt.b != nil {
					sb = styles(tt.b)[i]
				}
				if want := painting.TextStyleLerp(sa, sb, 0.25); !got.Equal(want) {
					t.Errorf("style %d = %v, want %v", i, got, want)
				}
			}
		})
	}
	for i, s := range styles(TextThemeLerp(nil, nil, 0.25)) {
		if s != nil {
			t.Errorf("style %d of the lerp of two nil themes is not nil", i)
		}
	}
}

func TestVisualDensityLerp(t *testing.T) {
	a := NewVisualDensity(VisualDensityOpts{Horizontal: foundation.Some(1.0), Vertical: foundation.Some(0.5)})
	b := NewVisualDensity(VisualDensityOpts{Horizontal: foundation.Some(2.0), Vertical: foundation.Some(1.0)})
	tests := []struct {
		t    float64
		want VisualDensity
	}{
		{0, VisualDensity{1, 0.5}},
		{0.25, VisualDensity{1.25, 0.625}},
		{1, VisualDensity{2, 1}},
	}
	for _, tt := range tests {
		if got := VisualDensityLerp(a, b, tt.t); got != tt.want {
			t.Errorf("lerp at %v = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestTextThemeOfAndPrimaryOf(t *testing.T) {
	tests := []struct {
		name string
		opts ThemeDataOpts
		get  func(bc widgets.BuildContext) *TextTheme
		want func(*ThemeData) *TextTheme
	}{
		{"of", ThemeDataOpts{TextTheme: NewTextTheme(TextThemeOpts{DisplayLarge: painting.NewTextStyle(painting.TextStyleOpts{
			Color: foundation.Some(Colors.Blue.Primary), FontSize: foundation.Some(30.0)})})},
			TextThemeOf, func(d *ThemeData) *TextTheme { return d.TextTheme }},
		{"primaryOf", ThemeDataOpts{PrimaryTextTheme: NewTextTheme(TextThemeOpts{DisplayLarge: painting.NewTextStyle(painting.TextStyleOpts{
			BackgroundColor: foundation.Some(Colors.Green.Primary), FontStyle: ui.FontStyleItalic})})},
			TextThemePrimaryOf, func(d *ThemeData) *TextTheme { return d.PrimaryTextTheme }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var got, want *TextTheme
			ts.PumpWidget(NewTheme(NewThemeData(tt.opts), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				got, want = tt.get(bc), tt.want(ThemeOf(bc))
				return widgets.NewSizedBox(nil)
			})))
			if got == nil || !got.Equal(want) {
				t.Errorf("text theme from the context = %v, want %v", got, want)
			}
		})
	}
}
