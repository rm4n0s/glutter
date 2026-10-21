//go:build linux

package wayland

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

const testTimeout = 10 * time.Second

type nopHandler = platform.NopHandler

// connect opens a connection to a fake compositor and builds a Platform on it,
// in shared-memory mode, which needs no GPU.
func connect(t *testing.T, cfg fakecomp.Config) (*Platform, *fakecomp.Server, error) {
	t.Helper()
	return connectWith(t, cfg, Options{ShmFill: true, NoLibdecor: true})
}

// connectWith is connect with the options given.
func connectWith(t *testing.T, cfg fakecomp.Config, opts Options) (*Platform, *fakecomp.Server, error) {
	t.Helper()
	if _, err := wl.Connect("glutter-probe-for-libwayland"); err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == "LibraryLoadFailed" {
			t.Skipf("libwayland-client is not available: %v", err)
		}
	}
	// A scale of 1 unless the test says otherwise: the fake compositor's own default is 2,
	// for the tests of the bindings.
	if cfg.OutputScale == 0 {
		cfg.OutputScale = 1
	}
	if cfg.PreferredBufferScale == 0 {
		cfg.PreferredBufferScale = 1
	}
	srv, fd := fakecomp.New(t, cfg)
	conn, err := wl.ConnectFD(fd)
	if err != nil {
		t.Fatalf("ConnectFD: %v", err)
	}
	p, err := newPlatform(conn, opts)
	if err != nil {
		srv.Wait() // newPlatform closed the connection
		return nil, srv, err
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		srv.Wait()
	})
	return p, srv, nil
}

func TestRequiredGlobalsAreBound(t *testing.T) {
	cfg := fakecomp.Config{Globals: append(fakecomp.DefaultGlobals(),
		fakecomp.Global{Name: 40, Interface: "wl_output", Version: 3}, // a second output, older
		fakecomp.Global{Name: 8, Interface: "wp_viewporter", Version: 1},
		fakecomp.Global{Name: 9, Interface: "zxdg_decoration_manager_v1", Version: 1},
		fakecomp.Global{Name: 10, Interface: "something_unknown_v1", Version: 1},
	)}
	p, srv, err := connect(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := p.sess

	binds := strings.Join(srv.Requests(), "\n")
	for _, want := range []string{
		"wl_registry.bind name=1 interface=wl_compositor version=6",
		"wl_registry.bind name=3 interface=wl_seat version=7", // offered 7, we use up to 9
		"wl_registry.bind name=2 interface=wl_output version=4",
		"wl_registry.bind name=40 interface=wl_output version=3", // never above what is offered
		"wl_registry.bind name=4 interface=xdg_wm_base version=6",
		"wl_registry.bind name=6 interface=wl_shm version=1",
	} {
		if !strings.Contains(binds, want) {
			t.Errorf("missing %q in\n%s", want, binds)
		}
	}
	if s.compositor == nil || s.shm == nil || s.seat == nil || s.wmBase == nil || len(s.outputs) != 2 {
		t.Errorf("not everything is bound: compositor=%v shm=%v seat=%v wmBase=%v outputs=%d",
			s.compositor, s.shm, s.seat, s.wmBase, len(s.outputs))
	}
}

func TestOptionalGlobalsAreRecorded(t *testing.T) {
	globals := append(fakecomp.DefaultGlobals(),
		fakecomp.Global{Name: 20, Interface: "wp_presentation", Version: 1},
		fakecomp.Global{Name: 21, Interface: "zxdg_decoration_manager_v1", Version: 2},
		fakecomp.Global{Name: 22, Interface: "something_unknown_v1", Version: 1},
	)
	p, srv, err := connect(t, fakecomp.Config{Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Global{
		"wp_presentation":            {20, 1},
		"zxdg_decoration_manager_v1": {21, 2},
		"wl_data_device_manager":     {5, 3}, // part of the default globals
	}
	if !reflect.DeepEqual(p.sess.optionalPresent, want) {
		t.Errorf("optional globals = %v, want %v", p.sess.optionalPresent, want)
	}
	if g, ok := p.Globals("something_unknown_v1"); !ok || len(g) != 1 {
		t.Errorf("Globals(unknown) = %v, %v; every announced global is listed", g, ok)
	}
	if g, ok := p.Globals("zwp_tablet_manager_v2"); ok || len(g) != 0 {
		t.Errorf("Globals(absent) = %v, %v, want none", g, ok)
	}
	// Optional globals that no code uses yet are only recorded, not bound.
	if strings.Contains(strings.Join(srv.Requests(), "\n"), "wp_presentation") {
		t.Error("an optional global was bound")
	}
}

func TestScaleGlobalsAreBoundTogether(t *testing.T) {
	both := append(fakecomp.DefaultGlobals(),
		fakecomp.Global{Name: 20, Interface: "wp_viewporter", Version: 1},
		fakecomp.Global{Name: 21, Interface: "wp_fractional_scale_manager_v1", Version: 1})
	tests := []struct {
		name               string
		globals            []fakecomp.Global
		viewporter, scaler bool
	}{
		{"both", both, true, true},
		{"a viewporter only", both[:len(both)-1], true, false},
		// Fractional scale is of no use without a viewport.
		{"a fractional scale manager only", append(fakecomp.DefaultGlobals(), both[len(both)-1]), false, false},
		{"neither", fakecomp.DefaultGlobals(), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _, err := connect(t, fakecomp.Config{Globals: tt.globals})
			if err != nil {
				t.Fatal(err)
			}
			if (p.sess.viewporter != nil) != tt.viewporter || (p.sess.fractionalScale != nil) != tt.scaler {
				t.Errorf("viewporter %v, fractional scale manager %v", p.sess.viewporter != nil, p.sess.fractionalScale != nil)
			}
		})
	}
}

func TestMissingRequiredGlobals(t *testing.T) {
	complete := fakecomp.DefaultGlobals()
	required := []string{"wl_compositor", "wl_subcompositor", "wl_shm", "wl_seat", "wl_output", "xdg_wm_base"}
	for _, missing := range required {
		t.Run(missing, func(t *testing.T) {
			var globals []fakecomp.Global
			for _, g := range complete {
				if g.Interface != missing {
					globals = append(globals, g)
				}
			}
			_, _, err := connect(t, fakecomp.Config{Globals: globals})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != "WaylandProtocolMissing" || !appErr.HasRoute("newPlatform->newSession->session.bindRequired.WaylandProtocolMissing") {
				t.Errorf("Tag = %q, route = %s", appErr.Tag, appErr.Route())
			}
			// The globals are checked in a fixed order, so with one missing it is that one.
			if appErr.Metadata["interface"] != missing {
				t.Errorf("Metadata[interface] = %v, want %s", appErr.Metadata["interface"], missing)
			}
		})
	}

	t.Run("the complete set works", func(t *testing.T) {
		if _, _, err := connect(t, fakecomp.Config{Globals: complete}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestConnectFailure(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "")
	_, err := New("glutter-no-such-display")
	appErr, ok := errors.FromError(err)
	if !ok {
		if err == nil {
			t.Fatal("New succeeded without a compositor")
		}
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	switch appErr.Tag {
	case "LibraryLoadFailed":
		t.Skipf("libwayland-client is not available: %v", err)
	case "WaylandConnectFailed":
		if !appErr.HasRoute("New->Connect.WaylandConnectFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	default:
		t.Errorf("Tag = %q, want WaylandConnectFailed", appErr.Tag)
	}
}

// runPlatform starts p.Run on a goroutine and returns its result channel and a
// cancel function.
func runPlatform(t *testing.T, p *Platform) (<-chan error, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, nopHandler{}) }()
	return done, cancel
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(testTimeout):
		t.Fatal("Run did not return in time")
		return nil
	}
}

func newRunnablePlatform(t *testing.T) (*Platform, *fakecomp.Server) {
	t.Helper()
	p, srv, err := connect(t, fakecomp.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return p, srv
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	p, _ := newRunnablePlatform(t)
	done, cancel := runPlatform(t, p)
	time.Sleep(30 * time.Millisecond) // the loop is waiting
	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("Run returned %v, want nil", err)
	}
}

func TestRunTwiceFails(t *testing.T) {
	p, _ := newRunnablePlatform(t)
	done, cancel := runPlatform(t, p)
	started := make(chan struct{})
	p.Post(func() { close(started) })
	<-started

	err := p.Run(context.Background(), nopHandler{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "PlatformAlreadyRunning" || !appErr.HasRoute("Platform.Run.PlatformAlreadyRunning") {
		t.Errorf("second Run returned %v, want PlatformAlreadyRunning via Platform.Run", err)
	}
	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("first Run returned %v", err)
	}
}

func TestPostFromOtherGoroutinesRunsInOrderOnTheLoop(t *testing.T) {
	const goroutines, per = 20, 20
	p, _ := newRunnablePlatform(t)
	done, cancel := runPlatform(t, p)

	var mu sync.Mutex
	got := make([][]int, goroutines)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range per {
				p.Post(func() {
					mu.Lock()
					got[g] = append(got[g], i)
					mu.Unlock()
				})
			}
		}()
	}
	wg.Wait()
	// Every goroutine has finished posting, so this function is queued after
	// all of theirs and runs after them.
	finished := make(chan struct{})
	p.Post(func() { close(finished) })
	<-finished
	cancel()
	if err := wait(t, done); err != nil {
		t.Fatal(err)
	}
	for g, seq := range got {
		if len(seq) != per {
			t.Errorf("goroutine %d: %d functions ran, want %d", g, len(seq), per)
			continue
		}
		for i, v := range seq {
			if v != i {
				t.Errorf("goroutine %d: position %d ran post %d", g, i, v)
				break
			}
		}
	}
}

func TestPingIsAnsweredWithPong(t *testing.T) {
	p, srv := newRunnablePlatform(t)
	done, cancel := runPlatform(t, p)

	srv.Send(srv.ID("xdg_wm_base"), 0, uint32(1234)) // xdg_wm_base.ping(serial)
	waitUntil(t, func() bool {
		return strings.Contains(srv.Joined(), "xdg_wm_base.pong serial=1234")
	}, "the compositor never received the pong")
	cancel()
	if err := wait(t, done); err != nil {
		t.Fatal(err)
	}
}

func waitUntil(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestProtocolErrorEndsRun(t *testing.T) {
	p, srv := newRunnablePlatform(t)
	done, _ := runPlatform(t, p)
	time.Sleep(30 * time.Millisecond)

	// wl_display.error(object_id, code, message) about the shm object.
	srv.Send(1, 0, srv.ID("wl_shm"), uint32(1), "invalid format")
	err := wait(t, done)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "WaylandProtocolError" {
		t.Fatalf("Tag = %q, want WaylandProtocolError (%s)", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["interface"] != "wl_shm" || appErr.Metadata["id"] != srv.ID("wl_shm") || appErr.Metadata["code"] != uint32(1) {
		t.Errorf("Metadata = %v, want interface wl_shm, id %d, code 1", appErr.Metadata, srv.ID("wl_shm"))
	}
}

func TestConnectionLossEndsRun(t *testing.T) {
	p, srv := newRunnablePlatform(t)
	done, _ := runPlatform(t, p)
	time.Sleep(30 * time.Millisecond)

	srv.Hangup()
	err := wait(t, done)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WaylandConnectionLost" {
		t.Errorf("Run returned %v, want WaylandConnectionLost", err)
	}
}
