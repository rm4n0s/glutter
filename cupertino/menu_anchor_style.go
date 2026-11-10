package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// menuIsCupertino is whether the platform is one of Apple's.
//
// Flutter: cupertino/menu_anchor.dart _isCupertino
func menuIsCupertino() bool {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return true
	}
	return false
}

const (
	menuBodyFont                   = CupertinoSystemTextFamily
	menuDisplayFont                = CupertinoSystemDisplayFamily
	menuMobileBaseFontSize         = 17.0
	menuMinimumNormalizedLargeText = 11.0
	menuMinimumTextScaleFactor     = 1 - 3/menuMobileBaseFontSize
	menuMaximumTextScaleFactor     = 1 + 36/menuMobileBaseFontSize
	menuTabletWidthThreshold       = 768.0
)

// menuNormalizeTextScale is how many points the text scaler adds to the base font size of iOS.
//
// Flutter: cupertino/menu_anchor.dart _normalizeTextScale
func menuNormalizeTextScale(textScaler painting.TextScaler) float64 {
	return textScaler.Scale(menuMobileBaseFontSize) - menuMobileBaseFontSize
}

// menuLargeTextModeEnabled is whether the text is as large as the accessibility sizes of iOS.
//
// Flutter: cupertino/menu_anchor.dart _largeTextModeEnabled
func menuLargeTextModeEnabled(bc widgets.BuildContext) bool {
	textScaler, ok := widgets.MediaQueryMaybeTextScalerOf(bc)
	return ok && menuNormalizeTextScale(textScaler) >= menuMinimumNormalizedLargeText
}

// menuWidthOf is the width of the menu for the width of the screen.
//
// Flutter: cupertino/menu_anchor.dart _CupertinoMenuWidth.fromScreenWidth
func menuWidthOf(screenWidth float64, largeText bool) float64 {
	isMobile := screenWidth < menuTabletWidthThreshold
	switch {
	case !isMobile && !largeText:
		return 262
	case !isMobile && largeText:
		return 343
	case isMobile && !largeText:
		return 250
	}
	return 370
}

type dynamicTypeEntry struct {
	size, height, letterSpacing float64
	family                      string
}

// dynamicTypeStyle is a text style of iOS for each text size of the system.
//
// Flutter: cupertino/menu_anchor.dart _DynamicTypeStyle
type dynamicTypeStyle struct {
	entries []dynamicTypeEntry
}

var (
	dynamicTypeBody = dynamicTypeStyle{[]dynamicTypeEntry{
		{14, 19, -0.15, menuBodyFont}, {15, 20, -0.23, menuBodyFont}, {16, 21, -0.31, menuBodyFont}, {17, 22, -0.43, menuBodyFont},
		{19, 24, -0.44, menuBodyFont}, {21, 26, -0.36, menuBodyFont}, {23, 29, -0.10, menuDisplayFont}, {28, 34, 0.38, menuDisplayFont},
		{33, 40, 0.40, menuDisplayFont}, {40, 48, 0.37, menuDisplayFont}, {47, 56, 0.37, menuDisplayFont}, {53, 62, 0.31, menuDisplayFont}}}
	dynamicTypeSubhead = dynamicTypeStyle{[]dynamicTypeEntry{
		{12, 16, 0, menuBodyFont}, {13, 18, -0.08, menuBodyFont}, {14, 19, -0.15, menuBodyFont}, {15, 20, -0.23, menuBodyFont},
		{17, 22, -0.43, menuBodyFont}, {19, 24, -0.45, menuBodyFont}, {21, 28, -0.36, menuBodyFont}, {25, 31, 0.15, menuDisplayFont},
		{30, 37, 0.40, menuDisplayFont}, {36, 43, 0.37, menuDisplayFont}, {42, 50, 0.37, menuDisplayFont}, {49, 58, 0.33, menuDisplayFont}}}
)

func (e dynamicTypeEntry) style() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(e.size), Height: foundation.Some(e.height / e.size),
		LetterSpacing: foundation.Some(e.letterSpacing), FontFamily: e.family})
}

// normalizedBodyScales are the text scales of the sizes of the body, as whole points above the base size.
func menuNormalizedBodyScales() []int {
	scales := make([]int, len(dynamicTypeBody.entries))
	for i, e := range dynamicTypeBody.entries {
		scales[i] = int(e.size - menuMobileBaseFontSize)
	}
	return scales
}

// resolveTextStyle is the style for the text scaler: the one of its size, or the interpolation of the two nearest.
//
// Flutter: cupertino/menu_anchor.dart _DynamicTypeStyle.resolveTextStyle
func (s dynamicTypeStyle) resolveTextStyle(textScaler painting.TextScaler) *painting.TextStyle {
	units := menuNormalizeTextScale(textScaler)
	scales := menuNormalizedBodyScales()
	for i := range s.entries {
		bodyUnits := float64(scales[i])
		if units > bodyUnits {
			continue
		}
		if units == bodyUnits || i == 0 {
			return s.entries[i].style()
		}
		t := (units - float64(scales[i-1])) / (bodyUnits - float64(scales[i-1]))
		return painting.TextStyleLerp(s.entries[i-1].style(), s.entries[i].style(), ui.LerpDouble(0, 1, t))
	}
	return s.entries[len(s.entries)-1].style()
}

// menuSquaredDistanceToRect is the squared distance between the point and the rectangle, zero inside it.
//
// Flutter: cupertino/menu_anchor.dart _computeSquaredDistanceToRect
func menuSquaredDistanceToRect(point ui.Offset, rect ui.Rect) float64 {
	dx := point.Dx - ui.ClampDouble(point.Dx, rect.Left, rect.Right)
	dy := point.Dy - ui.ClampDouble(point.Dy, rect.Top, rect.Bottom)
	return dx*dx + dy*dy
}

// menuRoundToDivisible rounds the value to a multiple of to.
//
// Flutter: cupertino/menu_anchor.dart _roundToDivisible
func menuRoundToDivisible(value, to float64) float64 {
	if to == 0 {
		return value
	}
	return math.Round(value/to) * to
}
