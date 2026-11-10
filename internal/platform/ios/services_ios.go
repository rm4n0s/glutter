//go:build ios

package ios

import (
	"context"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/coreaudio"
	"github.com/rm4n0s/glutter/internal/platform/darwin/notify"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/darwin/openuri"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// desktop is the services of package desktop on iOS (SPEC §30.6): open a URI with UIApplication and notifications through
// UserNotifications (the rules of package notify, as on macOS). The tray, the file dialogs and a menu bar do not exist on iOS: package
// desktop answers those with their Unavailable tags (DesktopName says iOS).
//
// Flutter: no counterpart (Flutter apps use the url_launcher and flutter_local_notifications plugins)
type desktop struct {
	p     *Platform
	notes *notify.Notifier
}

var (
	_ mobile.Services          = (*desktop)(nil)
	_ platform.DesktopProvider = (*Platform)(nil)
)

// Desktop returns the services; they are made at the first call, which may come from any goroutine, and live as long as the platform.
//
// Flutter: no counterpart
func (p *Platform) Desktop() mobile.Services {
	p.deskOnce.Do(func() {
		d := &desktop{p: p}
		d.notes = notify.New(notify.NewCenter(func() *notify.Notifier { return d.notes }))
		p.desk = d
	})
	return p.desk
}

// deskState are the fields of the platform for the services.
type deskState struct {
	deskOnce sync.Once
	desk     *desktop
}

// DesktopName says which desktop the services are for: package desktop uses it in the error that says that D-Bus is not tried.
//
// Flutter: no counterpart
func (d *desktop) DesktopName() string { return "iOS" }

// OpenURI opens the URI with UIApplication.openURL:options:completionHandler: on the UI thread and waits for the system's answer, which
// is the completion handler's. The checks of package openuri come first (a script URI is not opened). A URI that no app handles, or that
// the system refuses (an https: or custom scheme that needs the Info.plist's LSApplicationQueriesSchemes does not: opening needs none),
// is OpenURIUnavailable.
//
// Flutter: no counterpart (url_launcher_ios: launchUrl)
func (d *desktop) OpenURI(ctx context.Context, uri string) error {
	if err := openuri.Check(uri); err != nil {
		return err
	}
	done := make(chan bool, 1)
	d.p.Post(func() { d.open(uri, done) })
	select {
	case ok := <-done:
		if !ok {
			return errors.New(tagOpenURIUnavailable, "iOS cannot open the URI", "uri", uri, "reason", "no app opens the URL, or the system refused it")
		}
		return nil
	case <-ctx.Done():
		return errors.NewErr(tagOpenURIUnavailable, ctx.Err()).SetMetadata("uri", uri, "reason", "the context ended before iOS answered")
	}
}

// open asks the application to open the URL; its answer goes to done. The block is kept until the completion handler has run.
func (d *desktop) open(uri string, done chan<- bool) {
	objcx.WithPool(func() {
		url := objcx.NewURL(uri)
		if url == 0 {
			done <- false
			return
		}
		defer objcx.Release(url)
		var kept objc.Block
		blk := objcx.NewBlock(func(_ objc.Block, ok bool) {
			done <- ok
			// The block is released after the handler has returned: a block that ends itself would free the code that is running.
			d.p.Post(kept.Release)
		})
		kept = blk.Copy()
		blk.Release()
		opts := class("NSDictionary").Send(sel("dictionary"))
		class("UIApplication").Send(sel("sharedApplication")).Send(sel("openURL:options:completionHandler:"), url, opts, kept)
	})
}

// ShowNotification shows the notification through UserNotifications: the permission is asked at the first one (the prompt of iOS), and a
// user who refused is NotificationsUnavailable.
//
// Flutter: no counterpart (flutter_local_notifications_darwin)
func (d *desktop) ShowNotification(ctx context.Context, spec mobile.NotificationSpec) (uint32, error) {
	return d.notes.Show(ctx, spec)
}

// CloseNotification takes the notification off the screen. A notification that is gone is not an error.
//
// Flutter: no counterpart
func (d *desktop) CloseNotification(_ context.Context, id uint32) error {
	d.notes.Close(id)
	return nil
}

// iosServices is platform.Services of iOS: the accessibility bridge (SPEC §30.7), the gamepads, the audio and the motion sensors (§30.9).
type iosServices struct {
	acc   platform.Accessibility
	pads  platform.Gamepads
	audio platform.Audio
	sens  platform.Sensors
}

var _ platform.Services = iosServices{}

func (s iosServices) Accessibility() platform.Accessibility { return s.acc }
func (s iosServices) Gamepads() platform.Gamepads           { return s.pads }
func (s iosServices) Audio() platform.Audio                 { return s.audio }
func (s iosServices) Sensors() platform.Sensors             { return s.sens }

// servicesState are the fields of the platform for the audio service (the other services have their own).
type servicesState struct {
	audioOnce     sync.Once
	audio         *coreaudio.Service
	audioPolicy   *sessionPolicy
	audioObserver objc.ID
	// motion is the motion sensors service (coremotion_ios.go), made empty and loaded at the first use.
	motion coreMotion
}

// Services returns the services of the platform: the accessibility bridge, the gamepads, the audio and the sensors. They are made at the
// first call, which is the engine's; the audio and the sensors start their work when an app opens a stream or a sensor.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services {
	p.accOnce.Do(func() { p.acc = newA11yBridge(p) })
	return iosServices{acc: p.acc, audio: p.audioService(), sens: &p.motion, pads: p.pads()}
}

// closeServices ends the services with the platform: the timers of the notifications stop.
func (d *desktop) closeServices() { d.notes.Shutdown() }

// The tag of the error that this file makes (rm4n0s/errors, R5); package desktop matches it by value.
const tagOpenURIUnavailable = "OpenURIUnavailable"
