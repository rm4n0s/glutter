//go:build darwin

package coreaudio

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

const tagFakeSession = "FakeSession"

// fakeSession is a Session that records what it was asked and fails when told to.
type fakeSession struct {
	kinds []platform.AudioKind
	err   error
}

func (s *fakeSession) Activate(kind platform.AudioKind) error {
	s.kinds = append(s.kinds, kind)
	return s.err
}

// routeService is a service over a system that has only the route (iOS), with a session.
func routeService(t *testing.T, sess Session) (*fakeSystem, *Service) {
	t.Helper()
	f := newFakeSystem(t)
	f.lib.routeOnly = true
	s := f.service(nil)
	s.session = sess
	return f, s
}

func TestRouteOnlySystemHasOneDeviceEachWay(t *testing.T) {
	f, s := routeService(t, nil)
	out, err := s.Devices(platform.AudioPlayback)
	if err != nil || len(out) != 1 || out[0].ID != RouteOutputID || !out[0].Default || out[0].Kind != platform.AudioPlayback {
		t.Errorf("playback devices = %+v, %v", out, err)
	}
	in, err := s.Devices(platform.AudioCapture)
	if err != nil || len(in) != 1 || in[0].ID != RouteInputID || !in[0].Default || in[0].Kind != platform.AudioCapture {
		t.Errorf("capture devices = %+v, %v", in, err)
	}
	// There is nothing to listen to: the AudioObject functions are not there.
	if len(f.listeners) != 0 {
		t.Errorf("listeners were added: %v", f.listeners)
	}
}

func TestRouteOnlyDeviceOpensWithoutNamingTheQueue(t *testing.T) {
	f, s := routeService(t, nil)
	// The route's own ID names no AudioObject, and the queue is not told a device: the system routes it.
	d := openPlayback(t, s, platform.AudioOpenOptions{ID: RouteOutputID}, &recorder{})
	if len(f.queues) != 1 || len(f.queues[0].props) != 0 {
		t.Errorf("queues %d, props %v", len(f.queues), f.queues[0].props)
	}
	if d.object != 0 || d.listening {
		t.Errorf("the device has an object (%d) or a listener (%v)", d.object, d.listening)
	}
	// A device that is not the route's is refused as on macOS.
	if _, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: "UsbHeadset"}, &recorder{}); err == nil {
		t.Error("a device that the route does not have was opened")
	}
}

func TestSessionIsActivatedBeforeAStreamOpens(t *testing.T) {
	sess := &fakeSession{}
	f, s := routeService(t, sess)
	openPlayback(t, s, platform.AudioOpenOptions{}, &recorder{})
	if _, err := s.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, &recorder{}); err != nil {
		t.Fatal(err)
	}
	if len(sess.kinds) != 2 || sess.kinds[0] != platform.AudioPlayback || sess.kinds[1] != platform.AudioCapture {
		t.Errorf("the session was activated for %v", sess.kinds)
	}
	if len(f.queues) != 2 {
		t.Errorf("queues = %d", len(f.queues))
	}
}

func TestSessionThatFailsStopsTheOpen(t *testing.T) {
	sess := &fakeSession{err: errors.New(tagFakeSession, "the session refused")}
	f, s := routeService(t, sess)
	_, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagFakeSession {
		t.Errorf("error %v, want the session's own", err)
	}
	if len(f.queues) != 0 {
		t.Errorf("a queue was made after the session failed")
	}
}

func TestInterruptionPausesAndResumes(t *testing.T) {
	sess := &fakeSession{}
	f, s := routeService(t, sess)
	h := &recorder{}
	d := openPlayback(t, s, platform.AudioOpenOptions{}, h)
	q := f.queues[0]
	if q.started != 1 {
		t.Fatalf("started = %d", q.started)
	}

	// The interruption begins: the queue is paused, the handler gets silence, the app is told.
	s.Interrupt(true, false)
	if q.paused != 1 || !d.silent.Load() {
		t.Errorf("paused %d, silent %v", q.paused, d.silent.Load())
	}
	if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioPaused {
		t.Errorf("states = %v", got)
	}
	// A second begin does nothing: the device is not running.
	s.Interrupt(true, false)
	if q.paused != 1 || len(h.stateList()) != 1 {
		t.Errorf("a second interruption paused again: %d, %v", q.paused, h.stateList())
	}

	// It ends, and the system says to go on: the session is active again and the queue runs.
	before := len(sess.kinds)
	s.Interrupt(false, true)
	if q.started != 2 || len(sess.kinds) != before+1 || d.silent.Load() {
		t.Errorf("started %d, activations %d, silent %v", q.started, len(sess.kinds)-before, d.silent.Load())
	}
	if got := h.stateList(); len(got) != 2 || got[1] != platform.AudioRunning {
		t.Errorf("states = %v", got)
	}
	// An end without a begin does nothing.
	s.Interrupt(false, true)
	if q.started != 2 {
		t.Errorf("started = %d", q.started)
	}
}

func TestInterruptionThatDoesNotSayToResumeLeavesTheDevicePaused(t *testing.T) {
	f, s := routeService(t, nil)
	h := &recorder{}
	d := openPlayback(t, s, platform.AudioOpenOptions{}, h)
	s.Interrupt(true, false)
	s.Interrupt(false, false)
	if f.queues[0].started != 1 || !d.silent.Load() {
		t.Errorf("started %d, silent %v", f.queues[0].started, d.silent.Load())
	}
	// The app can resume it when it wants.
	if err := d.Resume(); err != nil {
		t.Fatal(err)
	}
	if f.queues[0].started != 2 {
		t.Errorf("started = %d", f.queues[0].started)
	}
}

func TestInterruptionLeavesWhatTheAppPausedAlone(t *testing.T) {
	f, s := routeService(t, nil)
	h := &recorder{}
	d := openPlayback(t, s, platform.AudioOpenOptions{}, h)
	if err := d.Pause(); err != nil {
		t.Fatal(err)
	}
	s.Interrupt(true, false)
	s.Interrupt(false, true)
	// The app paused it, so the interruption neither paused it again nor resumed it.
	if f.queues[0].paused != 1 || f.queues[0].started != 1 {
		t.Errorf("paused %d, started %d", f.queues[0].paused, f.queues[0].started)
	}
	if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioPaused {
		t.Errorf("states = %v", got)
	}
}

func TestInterruptionIgnoresClosedDevices(t *testing.T) {
	f, s := routeService(t, nil)
	d := openPlayback(t, s, platform.AudioOpenOptions{}, &recorder{})
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	s.Interrupt(true, false)
	s.Interrupt(false, true)
	if f.queues[0].paused != 0 {
		t.Errorf("a closed device was paused")
	}
}

func TestRouteLostPausesPlaybackOnly(t *testing.T) {
	f, s := routeService(t, nil)
	hp, hc := &recorder{}, &recorder{}
	openPlayback(t, s, platform.AudioOpenOptions{}, hp)
	if _, err := s.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, hc); err != nil {
		t.Fatal(err)
	}
	s.RouteLost()
	if f.queues[0].paused != 1 || f.queues[1].paused != 0 {
		t.Errorf("paused: playback %d, capture %d", f.queues[0].paused, f.queues[1].paused)
	}
	if got := hp.stateList(); len(got) != 1 || got[0] != platform.AudioPaused {
		t.Errorf("states = %v", got)
	}
	// A device that is paused already is not paused again.
	s.RouteLost()
	if f.queues[0].paused != 1 {
		t.Errorf("paused = %d", f.queues[0].paused)
	}
}
