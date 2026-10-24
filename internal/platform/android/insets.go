package android

import "github.com/rm4n0s/glutter/ui"

// insets are the areas of the window that the system covers, in physical pixels, as the Java
// view worked them out from the WindowInsets (SPEC §24.7):
//
//	padding  the system bars and the display cutout, ViewPadding
//	keyboard the input method's, ViewInsets
//	gesture  the system gestures', SystemGestureInsets
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterView.java FlutterView.onApplyWindowInsets
type insets struct {
	padding, keyboard, gesture edgeInsets
}

// edgeInsets are the four distances from the edges of the window.
type edgeInsets struct{ left, top, right, bottom int32 }

func (e edgeInsets) viewPadding() ui.ViewPadding {
	return ui.ViewPadding{Left: float64(e.left), Top: float64(e.top), Right: float64(e.right), Bottom: float64(e.bottom)}
}

// insetsValues is how many numbers Native.insets passes: left, top, right and bottom of padding,
// keyboard and gesture.
const insetsValues = 12

// decodeInsets reads the numbers of Native.insets; it reports false when there are not twelve or
// one is negative (the view never makes a negative inset).
func decodeInsets(v []int32) (insets, bool) {
	if len(v) != insetsValues {
		return insets{}, false
	}
	for _, n := range v {
		if n < 0 {
			return insets{}, false
		}
	}
	edge := func(i int) edgeInsets { return edgeInsets{v[i], v[i+1], v[i+2], v[i+3]} }
	return insets{padding: edge(0), keyboard: edge(4), gesture: edge(8)}, true
}
