//go:build ios

package ios

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the system UI of iOS (SPEC §30.3): the view controller of a window answers UIKit's questions about the status bar, the
// home indicator and the supported orientations from what the app asked, and a change of the answer is told to UIKit.

var _ ui.SystemChromeHost = (*window)(nil)

// registerViewController registers the class of the view controller of a window. UIKit asks it for the status bar's style and
// visibility, for the home indicator's, and for the orientations it supports.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm FlutterViewController
func registerViewController() (*objcx.RegisteredClass, error) {
	// chromeOf answers with what the window's chrome says; a controller whose window is gone answers the defaults.
	chromeOf := func(self objc.ID) chrome {
		if w := windowOf(self); w != nil {
			return w.chrome
		}
		return chrome{}
	}
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "ViewController", Super: "UIViewController",
		Methods: []objcx.Method{
			{Name: "preferredStatusBarStyle", Fn: func(self objc.ID, _ objc.SEL) int64 {
				if c := chromeOf(self); c.styleSet {
					return c.style
				}
				return statusBarDefault
			}},
			{Name: "prefersStatusBarHidden", Fn: func(self objc.ID, _ objc.SEL) bool { return chromeOf(self).hideStatusBar }},
			{Name: "prefersHomeIndicatorAutoHidden", Fn: func(self objc.ID, _ objc.SEL) bool { return chromeOf(self).hideHomeIndicator }},
			{Name: "supportedInterfaceOrientations", Fn: func(self objc.ID, _ objc.SEL) uint64 { return uint64(chromeOf(self).supported()) }},
			{Name: "shouldAutorotate", Fn: func(objc.ID, objc.SEL) bool { return true }},
		},
	})
}

// updateChrome tells UIKit that what the view controller answers has changed, so that it asks again.
func (w *window) updateChrome() {
	if w.vc == 0 {
		return
	}
	w.vc.Send(sel("setNeedsStatusBarAppearanceUpdate"))
	w.vc.Send(sel("setNeedsUpdateOfHomeIndicatorAutoHidden"))
	if send[bool](w.vc, "respondsToSelector:", sel("setNeedsUpdateOfSupportedInterfaceOrientations")) {
		w.vc.Send(sel("setNeedsUpdateOfSupportedInterfaceOrientations"))
	}
}

// SetPreferredOrientations limits the orientations of the app: the view controller supports the ones listed (every one for an empty
// list), and the scene turns to one of them if it is not in one now. A list with no orientation that iOS knows does nothing. The
// Info.plist must list the orientations too (iOS 16 and later).
//
// Flutter: services/system_chrome.dart SystemChrome.setPreferredOrientations
func (w *window) SetPreferredOrientations(orientations []ui.DeviceOrientation) {
	c, ok := w.chrome.withOrientations(orientations)
	if !ok {
		return
	}
	w.chrome = c
	objcx.WithPool(func() {
		w.updateChrome()
		w.p.RequestOrientations(c.supported())
	})
}

// SetEnabledSystemUIMode shows or hides the status bar and the home indicator (SPEC §30.3): the edge-to-edge mode shows both, the
// modes that hide the bars hide the status bar and let the home indicator hide itself, and the manual mode shows the ones listed.
//
// Flutter: services/system_chrome.dart SystemChrome.setEnabledSystemUIMode
func (w *window) SetEnabledSystemUIMode(mode ui.SystemUIMode, overlays []ui.SystemUIOverlay) {
	w.chrome = w.chrome.withMode(mode, overlays)
	objcx.WithPool(w.updateChrome)
}

// SetSystemUIOverlayStyle sets the status bar's style from the brightness of the status bar (a dark one has light content). The
// other parts of the style are Android's: iOS has no navigation bar and colors the status bar itself.
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIOverlayStyle
func (w *window) SetSystemUIOverlayStyle(style ui.SystemUIOverlayStyle) {
	w.chrome = w.chrome.withStyle(style)
	objcx.WithPool(w.updateChrome)
}

// SetSystemUIChangeCallback keeps the function. iOS never shows or hides the bars because of the user (the status bar and the home
// indicator come back with a touch or a swipe, which the system does without telling the app), so it is never called.
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIChangeCallback
func (w *window) SetSystemUIChangeCallback(fn func(visible bool)) { w.chromeCallback = fn }

// ChromeReport says what UIKit has of the system UI of the window: the style and visibility of the status bar as the scene's status
// bar manager has them, the home indicator and the supported orientations of the view controller. The dev events step "chrome" logs it,
// so that a Simulator test sees what UIKit has, not what Go meant.
func (w *window) chromeReport() (style int64, statusHidden, homeHidden bool, orientations uint64) {
	if w.scene != 0 {
		if mgr := send[objc.ID](w.scene, "statusBarManager"); mgr != 0 {
			style = send[int64](mgr, "statusBarStyle")
			statusHidden = send[bool](mgr, "isStatusBarHidden")
		}
	}
	homeHidden = send[bool](w.vc, "prefersHomeIndicatorAutoHidden")
	orientations = send[uint64](w.vc, "supportedInterfaceOrientations")
	return
}

// logChrome logs what chromeReport says as the line "ios chrome" (the dev events step "chrome").
func (w *window) logChrome() {
	style, statusHidden, homeHidden, orientations := w.chromeReport()
	w.p.log.Info(ChromeLogLine, "statusBarStyle", style, "statusBarHidden", statusHidden, "homeIndicatorHidden", homeHidden, "orientations", orientations)
}

// ReportChrome logs the line "ios chrome" for the first window: what UIKit has of its system UI (see ChromeLogLine). The Simulator
// tests use it to see the state that UIKit has after the app's calls.
//
// Flutter: no counterpart
func (p *Platform) ReportChrome() {
	if wins := p.sortedWindows(); len(wins) > 0 {
		wins[0].logChrome()
	}
}
