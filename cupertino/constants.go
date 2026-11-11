package cupertino

import (
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// MinInteractiveDimensionCupertino is the minimum dimension of any interactive region according to
// the iOS Human Interface Guidelines: 44 pixels.
//
// Flutter: cupertino/constants.dart kMinInteractiveDimensionCupertino
const MinInteractiveDimensionCupertino = 44.0

// CupertinoFocusColorOpacity is the opacity of the outline of a focused Cupertino widget: the system blue
// with this opacity, CupertinoFocusColorBrightness and CupertinoFocusColorSaturation.
//
// Flutter: cupertino/constants.dart kCupertinoFocusColorOpacity
const CupertinoFocusColorOpacity = 0.80

// CupertinoFocusColorBrightness is the lightness of the outline of a focused Cupertino widget.
//
// Flutter: cupertino/constants.dart kCupertinoFocusColorBrightness
const CupertinoFocusColorBrightness = 0.69

// CupertinoFocusColorSaturation is the saturation of the outline of a focused Cupertino widget.
//
// Flutter: cupertino/constants.dart kCupertinoFocusColorSaturation
const CupertinoFocusColorSaturation = 0.835

// CupertinoButtonTintedOpacityLight is the opacity of the background of a tinted button in the light
// brightness.
//
// Flutter: cupertino/constants.dart kCupertinoButtonTintedOpacityLight
const CupertinoButtonTintedOpacityLight = 0.12

// CupertinoButtonTintedOpacityDark is the opacity of the background of a tinted button in the dark brightness.
//
// Flutter: cupertino/constants.dart kCupertinoButtonTintedOpacityDark
const CupertinoButtonTintedOpacityDark = 0.26

// CupertinoButtonDefaultIconSize is the size of the icon of a button that has no text size.
//
// Flutter: cupertino/constants.dart kCupertinoButtonDefaultIconSize
const CupertinoButtonDefaultIconSize = 20.0

// CupertinoButtonTapMoveSlop is how far from a button of a touch device a finger can move and still tap it.
//
// Flutter: cupertino/constants.dart kCupertinoButtonTapMoveSlop
const CupertinoButtonTapMoveSlop = 70.0

// CupertinoButtonPadding is the padding of a button of the size: the unspecified size is the large one.
//
// Flutter: cupertino/constants.dart kCupertinoButtonPadding
func CupertinoButtonPadding(size CupertinoButtonSize) painting.EdgeInsets {
	switch size {
	case CupertinoButtonSizeSmall:
		return painting.NewEdgeInsetsSymmetric(6, 12)
	case CupertinoButtonSizeMedium:
		return painting.NewEdgeInsetsSymmetric(10, 15)
	}
	return painting.NewEdgeInsetsSymmetric(16, 20)
}

// CupertinoButtonSizeBorderRadius is the border radius of a button of the size: the unspecified size is the
// large one.
//
// Flutter: cupertino/constants.dart kCupertinoButtonSizeBorderRadius
func CupertinoButtonSizeBorderRadius(size CupertinoButtonSize) painting.BorderRadius {
	switch size {
	case CupertinoButtonSizeSmall, CupertinoButtonSizeMedium:
		return painting.NewBorderRadiusAll(ui.NewRadiusCircular(40))
	}
	return painting.NewBorderRadiusAll(ui.NewRadiusCircular(12))
}

// CupertinoButtonMinSize is the minimum width and height of a button of the size: the unspecified size is the
// large one.
//
// Flutter: cupertino/constants.dart kCupertinoButtonMinSize
func CupertinoButtonMinSize(size CupertinoButtonSize) float64 {
	switch size {
	case CupertinoButtonSizeSmall:
		return 28
	case CupertinoButtonSizeMedium:
		return 32
	}
	return 44
}
