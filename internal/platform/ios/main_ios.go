//go:build ios

package ios

/*
#cgo LDFLAGS: -framework UIKit -framework Foundation -framework QuartzCore -framework IOSurface
*/
import "C"

import (
	"runtime"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// init locks the main goroutine, which runs the init functions of the program, to the main thread of the process (the thread that the
// runtime started on): UIKit works only there, and a program that links this package calls Main or Platform.Run from main.
func init() { runtime.LockOSThread() }

// Main runs UIApplicationMain on the calling thread, which must be the main thread of the process (a program's main function, whose
// thread package mainthread locked), with a delegate class and a scene delegate class that Go implements. It does not return while
// the app runs. It fails with IOSNotMainThread when the thread is another, and with the loader's errors when a framework or a class is missing.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterAppDelegate.mm FlutterAppDelegate
func Main(o MainOptions) error {
	return launch(o.DidFinishLaunching, func() (*objcx.RegisteredClass, error) { return registerSceneDelegate(o) })
}

// launch is Main with the scene delegate class made by the caller: package ios's own platform has one of its own (scene_ios.go).
func launch(didFinish func(), registerScene func() (*objcx.RegisteredClass, error)) error {
	uikit, err := dl.OpenFramework("UIKit")
	if err != nil {
		return err
	}
	libc, err := dl.Open("/usr/lib/libSystem.B.dylib")
	if err != nil {
		return err
	}
	var pthreadMainNP func() int32
	if err := libc.Bind(&pthreadMainNP, "pthread_main_np"); err != nil {
		return err
	}
	if pthreadMainNP() == 0 {
		return errors.New(TagIOSNotMainThread, "UIKit works only on the main thread of the process: call Main from main")
	}
	var uiApplicationMain func(argc int32, argv uintptr, principal, delegate objc.ID) int32
	if err := uikit.Bind(&uiApplicationMain, "UIApplicationMain"); err != nil {
		return err
	}
	scene, err := registerScene()
	if err != nil {
		return err
	}
	app, err := registerAppDelegate(didFinish, scene)
	if err != nil {
		return err
	}
	name := objcx.NewString(app.Name)
	defer objcx.Release(name)
	// UIApplicationMain runs until the app ends and does not return in practice; if it does, the app is over.
	uiApplicationMain(0, 0, 0, name)
	return nil
}

// registerAppDelegate registers the app delegate (it does not adopt UIApplicationDelegate formally: the runtime does not know that
// protocol, which no UIKit class adopts, and UIKit asks the object for each method with respondsToSelector:): it tells UIKit that the app launched, and answers for the scene configuration
// with the scene delegate (the manifest of the Info.plist has no configurations, so UIKit asks).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterAppDelegate.mm FlutterAppDelegate
func registerAppDelegate(didFinish func(), scene *objcx.RegisteredClass) (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "AppDelegate", Super: "UIResponder",
		Methods: []objcx.Method{
			{Name: "application:didFinishLaunchingWithOptions:", Fn: func(objc.ID, objc.SEL, objc.ID, objc.ID) bool {
				if didFinish != nil {
					didFinish()
				}
				return true
			}},
			{Name: "application:configurationForConnectingSceneSession:options:", Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, session objc.ID, _ objc.ID) objc.ID {
				class, err := objcx.Class("UISceneConfiguration")
				if err != nil {
					return 0
				}
				name := objcx.NewString("Default Configuration")
				defer objcx.Release(name)
				config := objc.ID(class).Send(objcx.Sel("configurationWithName:sessionRole:"), name, session.Send(objcx.Sel("role")))
				config.Send(objcx.Sel("setDelegateClass:"), scene.Class)
				return config
			}},
		},
	})
}

// registerSceneDelegate registers the scene delegate, which adopts UIWindowSceneDelegate (UIKit raises an exception for a delegate
// class that does not conform to UISceneDelegate): when the scene connects it makes a window in it with an empty view controller
// and shows it. The window is kept by the delegate (UIKit holds the delegate as long as the scene lives).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSceneDelegate.mm scene:willConnectToSession:options:
func registerSceneDelegate(o MainOptions) (*objcx.RegisteredClass, error) {
	var window objc.ID
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "SceneDelegate", Super: "UIResponder", Protocols: []string{"UIWindowSceneDelegate"},
		Methods: []objcx.Method{
			{Name: "scene:willConnectToSession:options:", Fn: func(_ objc.ID, _ objc.SEL, scene, _, _ objc.ID) {
				windowClass, err := objcx.Class("UIWindow")
				if err != nil {
					return
				}
				controllerClass, err := objcx.Class("UIViewController")
				if err != nil {
					return
				}
				window = objc.ID(windowClass).Send(objcx.Sel("alloc")).Send(objcx.Sel("initWithWindowScene:"), scene)
				controller := objc.ID(controllerClass).Send(objcx.Sel("alloc")).Send(objcx.Sel("init"))
				window.Send(objcx.Sel("setRootViewController:"), controller)
				objcx.Release(controller)
				window.Send(objcx.Sel("makeKeyAndVisible"))
				if o.SceneDidConnect != nil {
					o.SceneDidConnect()
				}
			}},
		},
	})
}
