//go:build js && wasm

package foundation

import "sync/atomic"

// IsWeb is whether the program runs on the web: true under GOOS=js GOARCH=wasm.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = true

// browserPlatform holds the platform of the browser as a TargetPlatform; 0 means the
// web platform has not said yet, and the framework behaves like Linux until it does.
var browserPlatform atomic.Int32

// hostTargetPlatform is the platform the framework runs on: the one the browser
// reported through SetBrowserTargetPlatform, else Linux.
func hostTargetPlatform() TargetPlatform {
	if p := TargetPlatform(browserPlatform.Load()); p != TargetPlatformUnspecified {
		return p
	}
	return TargetPlatformLinux
}

// SetBrowserTargetPlatform tells the framework which operating system the browser
// runs on, as Flutter web's defaultTargetPlatform finds it from the browser. The web
// platform of the engine calls it once, at startup, before any widget exists. It exists
// only in the js/wasm build.
//
// Flutter: foundation/_platform_web.dart defaultTargetPlatform
func SetBrowserTargetPlatform(p TargetPlatform) { browserPlatform.Store(int32(p)) }
