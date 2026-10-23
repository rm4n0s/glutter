//go:build !(js && wasm) && !android

package foundation

// IsWeb is whether the program runs on the web: true under GOOS=js GOARCH=wasm.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = false

// hostTargetPlatform is the platform the framework runs on. Linux is the only
// platform with a desktop backend (SPEC §3); the web asks the browser (web_js.go).
func hostTargetPlatform() TargetPlatform { return TargetPlatformLinux }
