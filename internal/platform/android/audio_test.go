package android

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// permissionFixture is a bridge over a fake host, and the permission over them: the test plays the main thread by answering the
// commands that the host got.
type permissionFixture struct {
	b    *bridge
	host *fakeHost
	perm audioPermission
}

func newPermissionFixture() *permissionFixture {
	b := newBridge()
	h := newFakeHost()
	return &permissionFixture{b: b, host: h, perm: audioPermission{b: b, h: h}}
}

// answer waits for the n-th command and answers it through the bridge, as Native.desktopResult does.
func (f *permissionFixture) answer(t *testing.T, n, code int, text string) command {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if cmds := f.host.sentCommands(); len(cmds) > n {
			f.b.desktopResult(int(cmds[n].A), code, text)
			return cmds[n]
		}
		if time.Now().After(deadline) {
			t.Fatalf("command %d was never sent", n)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestRecordAudioPermission(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		answer      int // the user's answer to the prompt; -1 when no prompt is expected
		text        string
		want        bool
		wantRefused bool
	}{
		{"already granted", answerGranted, -1, "", true, false},
		{"granted at the prompt", answerDenied, answerGranted, "", true, false},
		{"refused at the prompt", answerDenied, answerDenied, "", false, false},
		{"the prompt ended without an answer", answerDenied, answerUndecided, "no activity is shown to ask the user from", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPermissionFixture()
			type result struct {
				ok  bool
				err error
			}
			got := make(chan result, 1)
			go func() {
				ok, err := f.perm.RecordAudio(context.Background())
				got <- result{ok, err}
			}()
			if c := f.answer(t, 0, tt.status, ""); c.Op != opAudioStatus || c.A == 0 {
				t.Errorf("the first command is %+v", c)
			}
			if tt.answer >= 0 {
				if c := f.answer(t, 1, tt.answer, tt.text); c.Op != opAudioPermission || c.A == 0 {
					t.Errorf("the second command is %+v", c)
				}
			}
			r := <-got
			if r.ok != tt.want {
				t.Errorf("granted is %v, want %v", r.ok, tt.want)
			}
			var refused mobile.Refused
			if tt.wantRefused != errors.As(r.err, &refused) {
				t.Errorf("error %v, want a refusal: %v", r.err, tt.wantRefused)
			} else if tt.wantRefused && refused.Detail != tt.text {
				t.Errorf("the reason is %q", refused.Detail)
			}
			if n := len(f.host.sentCommands()); (tt.answer >= 0) != (n == 2) {
				t.Errorf("%d commands were sent", n)
			}
		})
	}
}

func TestRecordAudioPermissionEnds(t *testing.T) {
	t.Run("the activity does not say the status", func(t *testing.T) {
		f := newPermissionFixture()
		f.b.linkOf(f.host).wait = 20 * time.Millisecond
		ok, err := f.perm.RecordAudio(context.Background())
		var refused mobile.Refused
		if ok || !errors.As(err, &refused) {
			t.Errorf("got %v, %v", ok, err)
		}
	})
	t.Run("the context ends while the prompt is shown", func(t *testing.T) {
		f := newPermissionFixture()
		ctx, cancel := context.WithCancel(context.Background())
		got := make(chan error, 1)
		go func() {
			_, err := f.perm.RecordAudio(ctx)
			got <- err
		}()
		f.answer(t, 0, answerDenied, "")
		deadline := time.Now().Add(5 * time.Second)
		for len(f.host.sentCommands()) < 2 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		cancel()
		var refused mobile.Refused
		if err := <-got; !errors.As(err, &refused) {
			t.Errorf("got %v", err)
		}
		// The answer that comes after the wait ended is dropped.
		f.b.desktopResult(int(f.host.sentCommands()[1].A), answerGranted, "")
		if n := len(f.b.linkOf(f.host).pending); n != 0 {
			t.Errorf("%d requests are still waiting", n)
		}
	})
}

func TestTheLinkIsOneForTheDesktopServicesAndThePermission(t *testing.T) {
	b := newBridge()
	h := newFakeHost()
	a := b.linkOf(h)
	b.desktopServices(h)
	if b.linkOf(h) != a || b.desktopLinkFor() != a {
		t.Error("the desktop services made a link of their own")
	}
}

// fakeBackground is an audio backend that records what the platform tells it.
type fakeBackground struct {
	mu    sync.Mutex
	calls []bool
}

func (*fakeBackground) Devices(platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	return nil, nil
}
func (*fakeBackground) SetChangeHandler(func()) {}
func (*fakeBackground) Open(context.Context, platform.AudioKind, platform.AudioOpenOptions, platform.AudioHandler) (platform.AudioDevice, error) {
	return nil, nil
}

func (f *fakeBackground) SetBackground(hidden bool) {
	f.mu.Lock()
	f.calls = append(f.calls, hidden)
	f.mu.Unlock()
}

func (f *fakeBackground) last() (hidden bool, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return false, 0
	}
	return f.calls[len(f.calls)-1], len(f.calls)
}

func TestTheAudioFollowsTheLifecycle(t *testing.T) {
	h := newHarness(t)
	fake := &fakeBackground{}
	h.b.lifecycleChanged(lifecycleStopped)
	h.p.audio.once.Do(func() { h.p.startAudio(fake) })
	if got, n := fake.last(); n != 1 || !got {
		t.Fatalf("the service was started with hidden=%v after %d calls, want hidden (the app was stopped)", got, n)
	}
	if h.p.Services().Audio() == nil {
		t.Fatal("the services have no audio")
	}
	h.run()
	h.b.lifecycleChanged(lifecycleResumed)
	waitFor(t, "the service to be told that the app is shown", func() bool { got, n := fake.last(); return n >= 2 && !got })
	h.b.lifecycleChanged(lifecyclePaused)
	h.sync(func() {})
	if got, _ := fake.last(); got {
		t.Error("a paused activity (visible, not in front) hid the audio")
	}
	h.b.lifecycleChanged(lifecycleStopped)
	waitFor(t, "the service to be told that the app is hidden", func() bool { got, _ := fake.last(); return got })
	if err := h.p.Close(); err != nil {
		t.Fatal(err)
	}
	if h.p.audio.svc.Load() != nil {
		t.Error("the audio service stays after Close")
	}
}

func TestTheServicesHaveTheAudioOfTheSystem(t *testing.T) {
	h := newHarness(t)
	got := h.p.Services().Audio()
	if got == nil {
		t.Skip("this system has no AAudio backend")
	}
	if h.p.Services().Audio() != got {
		t.Error("the audio service was made again")
	}
	h.p.closeAudio()
}
