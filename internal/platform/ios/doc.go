// Package ios is the platform of iOS (SPEC §30): UIKit through purego's objc (SPEC §29.2, the helpers of internal/platform/darwin)
// and, in the *_ios.go files only, the few lines of cgo that Go's iOS port needs (D35: the port links externally and purego needs
// cgo there, so the files that name the frameworks to link carry `import "C"`; nothing else is in C).
//
// Step 35.1 has the entry: Main calls UIApplicationMain on the main thread with a delegate class and a scene delegate class that Go
// implements, and calls back when the app has launched and a scene has connected. The view, the frames, the input and the rest
// come with the later steps of M35. On every other operating system the package has Main too, which fails with IOSUnavailable, so
// that a program that calls it builds everywhere.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterAppDelegate.mm FlutterAppDelegate
package ios

const (
	// TagIOSNotMainThread is the tag of the error of a call that UIKit allows on the main thread only, made on another thread.
	//
	// Flutter: no counterpart
	TagIOSNotMainThread = "IOSNotMainThread"
	// TagIOSUnavailable is the tag of the error of a call that needs iOS, made on another operating system.
	//
	// Flutter: no counterpart
	TagIOSUnavailable = "IOSUnavailable"
)

// MainOptions are the parameters of Main.
//
// Flutter: no counterpart
type MainOptions struct {
	// DidFinishLaunching is called on the main thread when the app has finished launching
	// (application:didFinishLaunchingWithOptions:); nil for nothing.
	//
	// Flutter: shell/platform/darwin/ios/framework/Source/FlutterAppDelegate.mm application:didFinishLaunchingWithOptions:
	DidFinishLaunching func()
	// SceneDidConnect is called on the main thread when the window scene has connected and its window is visible
	// (scene:willConnectToSession:options:); nil for nothing.
	//
	// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSceneDelegate.mm scene:willConnectToSession:options:
	SceneDidConnect func()
}
