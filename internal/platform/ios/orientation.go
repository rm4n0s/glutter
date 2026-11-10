package ios

// Orientation is a set of interface orientations, the bits of UIInterfaceOrientationMask.
//
// Flutter: services/system_chrome.dart DeviceOrientation
type Orientation uint64

// The interface orientations, with UIKit's bits.
const (
	// OrientationPortrait is UIInterfaceOrientationMaskPortrait.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.portraitUp
	OrientationPortrait Orientation = 1 << 1
	// OrientationPortraitUpsideDown is UIInterfaceOrientationMaskPortraitUpsideDown.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.portraitDown
	OrientationPortraitUpsideDown Orientation = 1 << 2
	// OrientationLandscapeLeft is UIInterfaceOrientationMaskLandscapeLeft.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.landscapeLeft
	OrientationLandscapeLeft Orientation = 1 << 4
	// OrientationLandscapeRight is UIInterfaceOrientationMaskLandscapeRight.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.landscapeRight
	OrientationLandscapeRight Orientation = 1 << 3
	// OrientationLandscape is both landscape orientations.
	//
	// Flutter: no counterpart
	OrientationLandscape = OrientationLandscapeLeft | OrientationLandscapeRight
)
