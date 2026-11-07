package android

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// The operations of the activity's command call for the permission to record (SPEC §28.3, step 31.6), done on the main thread by
// AudioPermission.java; the answers come back in Native.desktopResult, like those of the desktop services (desktop.go has 19 to 23,
// gamepads.go 28).
const (
	// opAudioStatus: A is the number of the request. The answer's code is answerGranted when the app may record, answerDenied when it may
	// not (yet).
	opAudioStatus = 29
	// opAudioPermission: A is the number of the request. The answer's code is answerGranted when the user granted the permission,
	// answerDenied when the user refused, and answerUndecided when it ended without a decision (the text is the reason).
	opAudioPermission = 30
)

// audioBackend is the audio service of Android (AAudio, in package aaudio) as the platform uses it: the service itself, and the way to
// tell it that the app is hidden, which pauses the devices that were opened to pause then.
type audioBackend interface {
	platform.Audio
	SetBackground(hidden bool)
}

// audioPermission asks the activity for RECORD_AUDIO. It implements aaudio.Permissions over the link of the desktop services, which
// brings the answers of the main thread back.
type audioPermission struct {
	b *bridge
	h host
}

// RecordAudio checks the permission and, when it is not granted, shows the system's prompt and waits for the user, until ctx ends. The
// prompt belongs to the activity: with none on the screen the answer is a refusal to decide, which is an error.
func (a audioPermission) RecordAudio(ctx context.Context) (bool, error) {
	l := a.b.linkOf(a.h)
	st, err := l.call(ctx, command{Op: opAudioStatus})
	if err != nil {
		return false, err
	}
	if st.code == answerGranted {
		return true, nil
	}
	got := make(chan desktopAnswer, 1)
	id := l.request(command{Op: opAudioPermission}, func(ans desktopAnswer) { got <- ans })
	select {
	case ans := <-got:
		switch ans.code {
		case answerGranted:
			return true, nil
		case answerDenied:
			return false, nil
		}
		return false, mobile.Refused{Detail: ans.text}
	case <-ctx.Done():
		l.forget(id)
		return false, mobile.Refused{Detail: "the context ended: " + ctx.Err().Error()}
	}
}

// audioState is the audio service of the platform, made by the first call of Services, and the worker that tells it that the app was
// hidden or shown.
type audioState struct {
	once   sync.Once
	svc    atomic.Pointer[audioBackend]
	hidden atomic.Bool
	wake   chan struct{}
}

// audioService returns the audio service, made on the first call: nil when there is no backend for the system the program runs on.
func (p *Platform) audioService() platform.Audio {
	p.audio.once.Do(func() {
		if svc := newAudioBackend(p); svc != nil {
			p.startAudio(svc)
		}
	})
	if s := p.audio.svc.Load(); s != nil {
		return *s
	}
	return nil
}

// startAudio makes the backend the audio service of the platform, with the state of the app now (hidden or not) and the worker that
// passes on the changes.
func (p *Platform) startAudio(svc audioBackend) {
	_, lifecycle, _ := p.b.snapshotAll()
	p.audio.hidden.Store(!lifecycle.visible())
	svc.SetBackground(!lifecycle.visible())
	p.audio.wake = make(chan struct{}, 1)
	p.audio.svc.Store(&svc)
	go func(wake <-chan struct{}) {
		for range wake {
			svc.SetBackground(p.audio.hidden.Load())
		}
	}(p.audio.wake)
}

// audioLifecycle tells the audio service, if there is one, that the app is hidden or shown. It is called on the UI thread and does not
// wait: pausing a device waits for its stream, which a worker does.
func (p *Platform) audioLifecycle(hidden bool) {
	if p.audio.svc.Load() == nil {
		return
	}
	p.audio.hidden.Store(hidden)
	select {
	case p.audio.wake <- struct{}{}:
	default:
	}
}

// closeAudio ends the worker of the audio service.
func (p *Platform) closeAudio() {
	if p.audio.svc.Load() != nil {
		close(p.audio.wake)
		p.audio.svc.Store(nil)
	}
}
