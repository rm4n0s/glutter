//go:build !(js && wasm) && !android && !windows && !(darwin && !ios)

package foundation

// IsWeb is whether the program runs on the web: true under GOOS=js GOARCH=wasm.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on: Linux on the desktop. The web asks the
// browser (web_js.go); Android, Windows and macOS have a file of their own (host_android.go, host_windows.go,
// host_darwin.go). iOS has none until step 35.1, so a GOOS=ios build compiles with this default.
func hostTargetPlatform() TargetPlatform { return TargetPlatformLinux }
