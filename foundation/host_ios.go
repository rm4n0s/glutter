//go:build ios

package foundation

// IsWeb is whether the program runs on the web: false on iOS.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on: iOS (SPEC §30.10). The framework then takes
// Flutter's iOS branches (the Cupertino text editing shortcuts, the bouncing scroll physics, the typography of
// the platform and the iOS text selection).
func hostTargetPlatform() TargetPlatform { return TargetPlatformIOS }
