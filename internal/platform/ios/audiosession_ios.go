//go:build ios

package ios

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/coreaudio"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// uiSessionAPI is sessionAPI over AVAudioSession's shared instance (SPEC §30.9).
//
// Flutter: no counterpart
type uiSessionAPI struct{}

var _ sessionAPI = uiSessionAPI{}

// reasonOf is the localized description of an NSError, "" for none.
func reasonOf(nserror objc.ID) string {
	if nserror == 0 {
		return ""
	}
	return objcx.String(send[objc.ID](nserror, "localizedDescription"))
}

// session is AVAudioSession.sharedInstance.
func (uiSessionAPI) session() objc.ID { return class("AVAudioSession").Send(sel("sharedInstance")) }

// SetCategory sets the category and its options.
func (a uiSessionAPI) SetCategory(category string, options uint64) (reason string, ok bool) {
	objcx.WithPool(func() {
		c := objcx.NewString(category)
		defer objcx.Release(c)
		var nserr objc.ID
		ok = send[bool](a.session(), "setCategory:withOptions:error:", c, options, unsafe.Pointer(&nserr))
		if !ok {
			reason = reasonOf(nserr)
		}
	})
	return reason, ok
}

// SetActive makes the session active or not.
func (a uiSessionAPI) SetActive(active bool) (reason string, ok bool) {
	objcx.WithPool(func() {
		var nserr objc.ID
		ok = send[bool](a.session(), "setActive:error:", active, unsafe.Pointer(&nserr))
		if !ok {
			reason = reasonOf(nserr)
		}
	})
	return reason, ok
}

// The notifications of AVAudioSession that the platform listens to, and the keys of what they carry (the strings are the names of the
// constants).
const (
	interruptionNotification = "AVAudioSessionInterruptionNotification"
	routeChangeNotification  = "AVAudioSessionRouteChangeNotification"
	interruptionTypeKey      = "AVAudioSessionInterruptionTypeKey"
	interruptionOptionKey    = "AVAudioSessionInterruptionOptionKey"
	routeChangeReasonKey     = "AVAudioSessionRouteChangeReasonKey"
)

// registerAudioObserver registers the class of the observer that the notification centre calls for the session's notifications.
func registerAudioObserver() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "AudioObserver", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "audioInterrupted:", Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed && p.audio != nil {
					typ, _ := noteNumber(note, interruptionTypeKey)
					opts, _ := noteNumber(note, interruptionOptionKey)
					began, resume := interruptionOf(typ, opts)
					p.audio.Interrupt(began, resume)
				}
			}},
			{Name: "audioRouteChanged:", Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed && p.audio != nil {
					if reason, _ := noteNumber(note, routeChangeReasonKey); routeLostOf(reason) {
						p.audio.RouteLost()
					}
				}
			}},
		},
	})
}

// noteNumber is the unsigned number that a notification carries under the key.
func noteNumber(note objc.ID, key string) (uint64, bool) {
	info := send[objc.ID](note, "userInfo")
	if info == 0 {
		return 0, false
	}
	k := objcx.NewString(key)
	defer objcx.Release(k)
	v := info.Send(sel("objectForKey:"), k)
	if v == 0 {
		return 0, false
	}
	return uint64(objcx.Int(v)), true
}

// audioService returns the audio service, made at the first call: Core Audio over the route, with the session of the app and the
// permission of the microphone. The notifications of the session are listened to from then on.
func (p *Platform) audioService() *coreaudio.Service {
	p.audioOnce.Do(func() {
		if _, err := dl.OpenFramework("AVFoundation"); err != nil {
			p.log.Error("AVFoundation could not be loaded: audio is not available", "err", err)
		}
		p.audioPolicy = &sessionPolicy{api: uiSessionAPI{}}
		p.audio = coreaudio.New(coreaudio.Options{Permissions: coreaudio.NewPermissions(), Log: p.log, Session: p.audioPolicy})
		objcx.WithPool(func() {
			p.audioObserver = p.s.audioObserver.New(p.handle)
			center := class("NSNotificationCenter").Send(sel("defaultCenter"))
			for name, selector := range map[string]string{interruptionNotification: "audioInterrupted:", routeChangeNotification: "audioRouteChanged:"} {
				n := objcx.NewString(name)
				center.Send(sel("addObserver:selector:name:object:"), p.audioObserver, sel(selector), n, objc.ID(0))
				objcx.Release(n)
			}
		})
	})
	return p.audio
}

// stopAudio stops listening to the session and releases the observer.
func (p *Platform) stopAudio() {
	if p.audioObserver == 0 {
		return
	}
	class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("removeObserver:"), p.audioObserver)
	objcx.Release(p.audioObserver)
	p.audioObserver = 0
}

// SendAudioInterruption posts AVAudioSession's own interruption notification, as the system does when a call comes (began) or ends (the
// system says to resume), so that a Simulator test, which has no call, drives the platform's handling of interruptions.
//
// Flutter: no counterpart
func (p *Platform) SendAudioInterruption(began bool) {
	objcx.WithPool(func() {
		typ, opts := int64(0), int64(1) // ended, should resume
		if began {
			typ, opts = 1, 0
		}
		tk, ok := objcx.NewString(interruptionTypeKey), objcx.NewString(interruptionOptionKey)
		tn, on := objcx.NewNumberInt(typ), objcx.NewNumberInt(opts)
		dict := objcx.NewDictionary([]objc.ID{tk, ok}, []objc.ID{tn, on})
		name := objcx.NewString(interruptionNotification)
		class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("postNotificationName:object:userInfo:"), name, objc.ID(0), dict)
		for _, o := range []objc.ID{tk, ok, tn, on, dict, name} {
			objcx.Release(o)
		}
	})
}
