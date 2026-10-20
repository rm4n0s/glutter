package material_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

var allPlatforms = []foundation.TargetPlatform{
	foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS,
	foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows,
}

func styles(t *TextTheme) []*painting.TextStyle {
	return []*painting.TextStyle{
		t.DisplayLarge, t.DisplayMedium, t.DisplaySmall, t.HeadlineLarge, t.HeadlineMedium, t.HeadlineSmall,
		t.TitleLarge, t.TitleMedium, t.TitleSmall, t.BodyLarge, t.BodyMedium, t.BodySmall,
		t.LabelLarge, t.LabelMedium, t.LabelSmall,
	}
}

func TestTypographyIsDefinedForAllTargetPlatforms(t *testing.T) {
	for _, platform := range allPlatforms {
		typography := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(platform)})
		if typography.Black == nil || typography.White == nil {
			t.Errorf("%s: missing black or white typography", platform.Name())
		}
	}
}

func TestTypographyLerpSpecialCases(t *testing.T) {
	typography := NewTypography()
	if TypographyLerp(typography, typography, 0.5) != typography {
		t.Error("lerp of identical typographies is not the typography itself")
	}
}

func TestTypographyFontFamilyByPlatform(t *testing.T) {
	helsinki := []string{"Ubuntu", "Adwaita Sans", "Cantarell", "DejaVu Sans", "Liberation Sans", "Arial"}
	tests := []struct {
		name         string
		opts         TypographyOpts
		wantFamily   string
		wantFallback []string
	}{
		{"default", TypographyOpts{}, "Roboto", nil},
		{"fuchsia", TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformFuchsia)}, "Roboto", nil},
		{"linux", TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)}, "Roboto", helsinki},
		{"windows", TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformWindows)}, "Segoe UI", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typography := NewTypographyMaterial2018(tt.opts)
			for _, theme := range []*TextTheme{typography.Black, typography.White} {
				if got := theme.TitleLarge.FontFamily; got != tt.wantFamily {
					t.Errorf("font family = %q, want %q", got, tt.wantFamily)
				}
				if got := theme.TitleLarge.FontFamilyFallback(); !slices.Equal(got, tt.wantFallback) {
					t.Errorf("font family fallback = %v, want %v", got, tt.wantFallback)
				}
			}
		})
	}
}

func TestTypographyAppleFonts(t *testing.T) {
	const display, text, meta = "CupertinoSystemDisplay", "CupertinoSystemText", ".AppleSystemUIFont"
	ios := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformIOS)})
	macos := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformMacOS)})
	wantIOS := []string{display, display, display, display, display, display, display, text, text, text, text, text, text, text, text}
	for _, theme := range []*TextTheme{ios.Black, ios.White} {
		for i, s := range styles(theme) {
			if s.FontFamily != wantIOS[i] {
				t.Errorf("iOS style %d font family = %q, want %q", i, s.FontFamily, wantIOS[i])
			}
		}
	}
	for _, theme := range []*TextTheme{macos.Black, macos.White} {
		for i, s := range styles(theme) {
			if s.FontFamily != meta {
				t.Errorf("macOS style %d font family = %q, want %q", i, s.FontFamily, meta)
			}
		}
	}
}

func TestTypographyDebugFillProperties(t *testing.T) {
	skipUnlessDebug(t)
	builder := foundation.NewDiagnosticPropertiesBuilder()
	NewTypographyMaterial2014(TypographyOpts{
		Black: TypographyBlackCupertino, White: TypographyWhiteCupertino,
		EnglishLike: TypographyEnglishLike2018, Dense: TypographyDense2018, Tall: TypographyTall2018,
	}).DebugFillProperties(builder)
	var names []string
	for _, node := range builder.Properties {
		if !node.IsFiltered(foundation.DiagnosticLevelInfo) {
			names = append(names, node.Name().Value())
		}
	}
	if want := []string{"black", "white", "englishLike", "dense", "tall"}; !slices.Equal(names, want) {
		t.Errorf("non-default properties = %v, want %v", names, want)
	}
}

func TestTypographyLerpBetweenAll(t *testing.T) {
	var all []*Typography
	for _, platform := range allPlatforms {
		all = append(all, NewTypographyMaterial2014(TypographyOpts{Platform: foundation.Some(platform)}))
	}
	for _, platform := range allPlatforms {
		all = append(all, NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(platform)}))
	}
	for _, platform := range allPlatforms {
		all = append(all, NewTypographyMaterial2021(Typography2021Opts{TypographyOpts: TypographyOpts{Platform: foundation.Some(platform)}}))
	}
	for _, from := range all {
		for _, to := range all {
			if TypographyLerp(from, to, 0.5) == nil {
				t.Fatal("lerp returned nil")
			}
		}
	}
}

type specStyle struct {
	weight  ui.FontWeight
	size    float64
	spacing float64
}

func checkSpec(t *testing.T, theme *TextTheme, want [15]specStyle) {
	t.Helper()
	names := []string{"displayLarge", "displayMedium", "displaySmall", "headlineLarge", "headlineMedium", "headlineSmall",
		"titleLarge", "titleMedium", "titleSmall", "bodyLarge", "bodyMedium", "bodySmall", "labelLarge", "labelMedium", "labelSmall"}
	for i, s := range styles(theme) {
		if s.FontFamily != "Roboto" {
			t.Errorf("%s: font family = %q", names[i], s.FontFamily)
		}
		if s.FontWeight != want[i].weight {
			t.Errorf("%s: font weight = %v, want %v", names[i], s.FontWeight, want[i].weight)
		}
		if got := s.FontSize.Value(); got != want[i].size {
			t.Errorf("%s: font size = %v, want %v", names[i], got, want[i].size)
		}
		if got := s.LetterSpacing.OrElse(0); got != want[i].spacing {
			t.Errorf("%s: letter spacing = %v, want %v", names[i], got, want[i].spacing)
		}
	}
}

func TestEnglishLike2018MatchesMaterialSpec(t *testing.T) {
	const light, regular, medium = ui.FontWeightW300, ui.FontWeightW400, ui.FontWeightW500
	theme := TypographyEnglishLike2018.Merge(TypographyBlackMountainView)
	checkSpec(t, theme, [15]specStyle{
		{light, 96, -1.5}, {light, 60, -0.5}, {regular, 48, 0}, {regular, 40, 0.25}, {regular, 34, 0.25}, {regular, 24, 0},
		{medium, 20, 0.15}, {regular, 16, 0.15}, {medium, 14, 0.1}, {regular, 16, 0.5}, {regular, 14, 0.25}, {regular, 12, 0.4},
		{medium, 14, 1.25}, {regular, 11, 1.5}, {regular, 10, 1.5},
	})
}

func TestEnglishLike2021MatchesMaterial3Spec(t *testing.T) {
	const regular, medium = ui.FontWeightW400, ui.FontWeightW500
	theme := TypographyEnglishLike2021.Merge(TypographyBlackMountainView)
	checkSpec(t, theme, [15]specStyle{
		{regular, 57, -0.25}, {regular, 45, 0}, {regular, 36, 0}, {regular, 32, 0}, {regular, 28, 0}, {regular, 24, 0},
		{regular, 22, 0}, {medium, 16, 0.15}, {medium, 14, 0.1}, {regular, 16, 0.5}, {regular, 14, 0.25}, {regular, 12, 0.4},
		{medium, 14, 0.1}, {medium, 12, 0.5}, {medium, 11, 0.5},
	})
	heights := [15]float64{1.12, 1.16, 1.22, 1.25, 1.29, 1.33, 1.27, 1.50, 1.43, 1.50, 1.43, 1.33, 1.43, 1.33, 1.45}
	for i, s := range styles(theme) {
		if got := s.Height.Value(); got != heights[i] {
			t.Errorf("style %d: height = %v, want %v", i, got, heights[i])
		}
		if s.TextBaseline != ui.TextBaselineAlphabetic || s.LeadingDistribution != ui.TextLeadingDistributionEven {
			t.Errorf("style %d: baseline %v, leading distribution %v", i, s.TextBaseline, s.LeadingDistribution)
		}
	}
}

func TestDefaultM3TextThemeUsesOnSurface(t *testing.T) {
	for _, brightness := range []ui.Brightness{ui.BrightnessLight, ui.BrightnessDark} {
		t.Run(brightness.String(), func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{Brightness: brightness})
			want := theme.ColorScheme.OnSurface()
			for i, s := range styles(theme.TextTheme) {
				if got := s.Color.Value(); got != want {
					t.Errorf("style %d: color = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestTypographyUnknownPlatform(t *testing.T) {
	route := recoverRoute(t, func() {
		NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(foundation.TargetPlatformUnspecified)})
	})
	if !strings.HasSuffix(route, "TypographyUnknownPlatform") {
		t.Errorf("failure path = %q", route)
	}
}
