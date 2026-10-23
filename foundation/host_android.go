//go:build android

package foundation

// IsWeb is whether the program runs on the web: false on Android.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on: Android (SPEC §24.12). The framework
// then takes Flutter's Android branches (clamping scroll physics, the Android typography, the tap
// target sizes).
func hostTargetPlatform() TargetPlatform { return TargetPlatformAndroid }
