//go:build ios

package ios

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// registerView registers the class of the view of a window: a UIView that the window's display link calls (displayLinkFired:), that is
// told when it is laid out (layoutSubviews: a rotation, a split of the screen, the safe area) and that is opaque. Each method finds
// its window by the handle in the instance's ivar and runs under the platform's guard (SPEC §30.4).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterView.mm FlutterView
func registerView() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "View", Super: "UIView",
		Methods: []objcx.Method{
			{Name: "isOpaque", Fn: func(objc.ID, objc.SEL) bool { return true }},
			{Name: "layoutSubviews", Fn: func(self objc.ID, _ objc.SEL) {
				if w := windowOf(self); w != nil {
					w.p.guard(w.layout)
				}
			}},
			{Name: "displayLinkFired:", Fn: func(self objc.ID, _ objc.SEL, link objc.ID) {
				if w := windowOf(self); w != nil {
					w.p.guard(func() { w.linkFired(link) })
				}
			}},
		},
	})
}

// registerPlatformSceneDelegate registers the scene delegate of the platform, which adopts UIWindowSceneDelegate (UIKit raises an
// exception for a delegate class that does not conform to UISceneDelegate): the scene's connection gives the windows their UIWindow
// and shows them, and the scene's activation states are the lifecycle of the app (SPEC §30.3, §30.7).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterSceneDelegate.mm FlutterSceneDelegate
func registerPlatformSceneDelegate(p *Platform) (*objcx.RegisteredClass, error) {
	state := func(name string, a activation) objcx.Method {
		return objcx.Method{Name: name, Fn: func(_ objc.ID, _ objc.SEL, scene objc.ID) {
			p.guard(func() { p.setActivation(scene, a) })
		}}
	}
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "SceneDelegate", Super: "UIResponder", Protocols: []string{"UIWindowSceneDelegate"},
		Methods: []objcx.Method{
			{Name: "scene:willConnectToSession:options:", Fn: func(_ objc.ID, _ objc.SEL, scene, _, _ objc.ID) {
				p.guard(func() { p.sceneConnect(scene) })
			}},
			state("sceneDidBecomeActive:", activationForegroundActive),
			state("sceneWillResignActive:", activationForegroundInactive),
			state("sceneWillEnterForeground:", activationForegroundInactive),
			state("sceneDidEnterBackground:", activationBackground),
			{Name: "sceneDidDisconnect:", Fn: func(_ objc.ID, _ objc.SEL, scene objc.ID) {
				p.guard(func() { p.sceneDisconnect(scene) })
			}},
		},
	})
}

// windowOf is the window of an instance of the view.
func windowOf(self objc.ID) *window {
	w, _ := objcx.LookupAs[*window](objcx.HandleOf(self))
	return w
}

// sceneConnect is the connection of a window scene: the first one is the scene that the windows of the app are made in. Each window
// that was made before gets its UIWindow and, now that the handler is there, its first frame.
func (p *Platform) sceneConnect(scene objc.ID) {
	objcx.Retain(scene)
	p.scenes[scene] = activation(send[int64](scene, "activationState"))
	if p.scene == 0 {
		p.scene = scene
	}
	for _, w := range p.sortedWindows() {
		w.attach(scene)
	}
	p.showNewWindows()
	p.updateLifecycle()
}

// sceneDisconnect forgets a scene that UIKit disconnected (it may connect again, as a new session).
func (p *Platform) sceneDisconnect(scene objc.ID) {
	if _, ok := p.scenes[scene]; !ok {
		return
	}
	delete(p.scenes, scene)
	if p.scene == scene {
		p.scene = 0
		for _, w := range p.windows {
			w.detach()
		}
	}
	objcx.Release(scene)
	p.updateLifecycle()
}

// setActivation is a change of the activation state of a scene: the lifecycle, the focus of the windows and the frames follow it.
func (p *Platform) setActivation(scene objc.ID, a activation) {
	if _, ok := p.scenes[scene]; !ok {
		return
	}
	p.scenes[scene] = a
	for _, w := range p.sortedWindows() {
		if w.scene == scene {
			w.activationChanged(a == activationForegroundActive)
		}
	}
	p.updateLifecycle()
}

// showNewWindows gives each window that has a UIWindow and was not shown its first frame and shows it.
func (p *Platform) showNewWindows() {
	for _, w := range p.sortedWindows() {
		w.firstShow()
	}
}

// RequestOrientations asks the scene to turn to one of the orientations (iOS 16 and later: UIWindowScene's geometry preferences; the
// system turns the scene when the device allows it and the Info.plist lists the orientation). The metrics follow when the scene has
// turned. Before a scene has connected, and on an iOS that has no geometry preferences, it does nothing.
//
// Flutter: services/system_chrome.dart SystemChrome.setPreferredOrientations
func (p *Platform) RequestOrientations(mask Orientation) {
	if p.scene == 0 {
		return
	}
	cls, err := objcx.Class("UIWindowSceneGeometryPreferencesIOS")
	if err != nil {
		return
	}
	objcx.WithPool(func() {
		prefs := objc.ID(cls).Send(sel("alloc")).Send(sel("initWithInterfaceOrientations:"), uint64(mask))
		defer objcx.Release(prefs)
		// The handler is told of a failure only; the new metrics are what the app sees of a success.
		handler := objcx.NewBlock(func(_ objc.Block, _ objc.ID) {})
		defer handler.Release()
		p.scene.Send(sel("requestGeometryUpdateWithPreferences:errorHandler:"), prefs, handler)
	})
}
