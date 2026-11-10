package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/cupertino/text_theme_test.dart: the default text
// theme matches the Apple design resources (size, family, letter spacing and weight of the
// nine styles).
func TestCupertinoTextThemeMatchesAppleDesignResources(t *testing.T) {
	theme := cupertino.NewCupertinoTextThemeData()
	const (
		none     ui.FontWeight = 0
		regular                = ui.FontWeightW400
		medium                 = ui.FontWeightW500
		semiBold               = ui.FontWeightW600
		bold                   = ui.FontWeightW700
	)
	tests := []struct {
		name          string
		style         *painting.TextStyle
		size          float64
		family        string
		letterSpacing float64
		weight        ui.FontWeight
	}{
		{"textStyle", theme.TextStyle(), 17, "CupertinoSystemText", -0.41, none},
		{"actionTextStyle", theme.ActionTextStyle(), 17, "CupertinoSystemText", -0.41, none},
		{"actionSmallTextStyle", theme.ActionSmallTextStyle(), 15, "CupertinoSystemText", -0.23, none},
		{"tabLabelTextStyle", theme.TabLabelTextStyle(), 10, "CupertinoSystemText", -0.24, medium},
		{"navTitleTextStyle", theme.NavTitleTextStyle(), 17, "CupertinoSystemText", -0.41, semiBold},
		{"navLargeTitleTextStyle", theme.NavLargeTitleTextStyle(), 34, "CupertinoSystemDisplay", 0.38, bold},
		{"navActionTextStyle", theme.NavActionTextStyle(), 17, "CupertinoSystemText", -0.41, none},
		{"pickerTextStyle", theme.PickerTextStyle(), 21, "CupertinoSystemDisplay", -0.6, regular},
		{"dateTimePickerTextStyle", theme.DateTimePickerTextStyle(), 21, "CupertinoSystemDisplay", 0.4, regular},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := tt.style.FontSize.Get(); got != tt.size {
				t.Errorf("font size = %v, want %v", got, tt.size)
			}
			if tt.style.FontFamily != tt.family {
				t.Errorf("font family = %q, want %q", tt.style.FontFamily, tt.family)
			}
			if got, _ := tt.style.LetterSpacing.Get(); got != tt.letterSpacing {
				t.Errorf("letter spacing = %v, want %v", got, tt.letterSpacing)
			}
			if tt.style.FontWeight != tt.weight {
				t.Errorf("font weight = %v, want %v", tt.style.FontWeight, tt.weight)
			}
		})
	}
}

func TestCupertinoTextThemeCopyWithKeepsWhatItIsNotGiven(t *testing.T) {
	custom := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Custom"})
	base := cupertino.NewCupertinoTextThemeData(cupertino.CupertinoTextThemeDataOpts{TextStyle: custom})
	copied := base.CopyWith(cupertino.CupertinoTextThemeDataOpts{PickerTextStyle: custom})
	if copied.TextStyle().FontFamily != "Custom" || copied.PickerTextStyle().FontFamily != "Custom" {
		t.Error("a style that was given is lost")
	}
	if copied.TabLabelTextStyle().FontFamily != "CupertinoSystemText" {
		t.Error("a default style changed")
	}
	if base.Equal(copied) || !base.Equal(base.CopyWith(cupertino.CupertinoTextThemeDataOpts{})) {
		t.Error("equality is wrong")
	}
}

// TestCupertinoTextStylesFallBackToRobotoOnTheWeb: the web has no system fonts, so the text styles of the library name
// the bundled Roboto as their fallback there (SPEC §31.2); the other platforms have the machine's font.
func TestCupertinoTextStylesFallBackToRobotoOnTheWeb(t *testing.T) {
	theme := cupertino.NewCupertinoThemeData().TextTheme()
	for name, style := range map[string]*painting.TextStyle{
		"text": theme.TextStyle(), "nav title": theme.NavTitleTextStyle(), "large title": theme.NavLargeTitleTextStyle(),
		"action": theme.ActionTextStyle(), "picker": theme.PickerTextStyle(),
	} {
		got := style.FontFamilyFallback()
		if foundation.IsWeb {
			if len(got) != 1 || got[0] != "Roboto" {
				t.Errorf("%s: fallback = %v, want [Roboto]", name, got)
			}
		} else if len(got) != 0 {
			t.Errorf("%s: fallback = %v, want none off the web", name, got)
		}
	}
}
