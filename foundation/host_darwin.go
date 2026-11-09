//go:build darwin && !ios

package foundation

// IsWeb is whether the program runs on the web: false on macOS.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on: macOS (SPEC §29.12). The framework then
// takes Flutter's macOS branches (the Command-based shortcuts, the macOS text editing shortcuts, the
// scroll physics of a trackpad and the typography of the platform).
func hostTargetPlatform() TargetPlatform { return TargetPlatformMacOS }
