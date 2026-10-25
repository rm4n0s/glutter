//go:build windows

package foundation

// IsWeb is whether the program runs on the web: false on Windows.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on: Windows (SPEC §25.12). The framework
// then takes Flutter's Windows branches (the Ctrl-based shortcuts, the Windows typography, the scroll
// physics and the text selection behavior).
func hostTargetPlatform() TargetPlatform { return TargetPlatformWindows }
