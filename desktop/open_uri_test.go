package desktop_test

import (
	"context"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/widgets"
)

// inGoroutine runs fn on a goroutine, as an app calls package desktop, and runs the
// functions it posts to the UI thread until it returns.
func inGoroutine(t *testing.T, ts *gluttertest.Tester, fn func() error) error {
	t.Helper()
	res := make(chan error, 1)
	go func() { res <- fn() }()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case err := <-res:
			return err
		case <-deadline:
			t.Fatal("the desktop call never returned")
		default:
		}
		ts.FlushPosted()
		time.Sleep(time.Millisecond)
	}
}

// tagged checks that err is a tagged error with the tag on its route, and returns it.
func tagged(t *testing.T, err error, tag string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(tag) {
		t.Fatalf("got %s via %s, want %s", appErr.Tag, appErr.Route(), tag)
	}
	return appErr
}

// assertTag is tagged for the tests that do not look at the error.
func assertTag(t *testing.T, err error, tag string) {
	t.Helper()
	_ = tagged(t, err, tag)
}

const (
	openURI = "org.freedesktop.portal.OpenURI.OpenURI"
	unique  = ":1.77"
)

func tokenIn(t *testing.T, opts map[string]godbus.Variant) string {
	t.Helper()
	token, _ := opts["handle_token"].Value().(string)
	if token == "" {
		t.Fatalf("no handle_token in %#v", opts)
	}
	return token
}

// portalAnswers makes the fake bus answer OpenURI with the response code.
func portalAnswers(t *testing.T, b *fakebus.Bus, code uint32) {
	t.Helper()
	b.Handle(dbus.PortalDest, dbus.PortalPath, openURI, func(c fakebus.Call) ([]any, error) {
		path := dbus.PortalRequestPath(unique, tokenIn(t, c.Args[2].(map[string]godbus.Variant)))
		b.Emit(&godbus.Signal{Path: path, Name: "org.freedesktop.portal.Request.Response",
			Body: []any{code, map[string]godbus.Variant{}}})
		return []any{path}, nil
	})
}

// programs records the programs that the app starts.
type programs struct {
	mu   sync.Mutex
	runs []run
	err  error
}

type run struct {
	env  []string
	name string
	args []string
}

func (p *programs) run(_ context.Context, env []string, name string, args ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.runs = append(p.runs, run{env, name, args})
	return p.err
}

func (p *programs) all() []run {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]run(nil), p.runs...)
}

func TestOpenURIThroughThePortal(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	portalAnswers(t, b, 0)
	desktop.UseBus(bc, b)
	progs := &programs{}
	desktop.UseProgram(bc, progs.run)

	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org/a?b=c") }); err != nil {
		t.Fatal(err)
	}
	calls := b.Calls()
	if len(calls) != 1 || calls[0].Method != openURI {
		t.Fatalf("calls = %+v", calls)
	}
	args := calls[0].Args
	if args[0] != "test:0" || args[1] != "https://example.org/a?b=c" {
		t.Errorf("parent %v, uri %v", args[0], args[1])
	}
	opts := args[2].(map[string]godbus.Variant)
	if got, _ := opts["activation_token"].Value().(string); got != "test-token" {
		t.Errorf("activation_token = %q, want the token of the platform", got)
	}
	if runs := progs.all(); len(runs) != 0 {
		t.Errorf("xdg-open ran after the portal worked: %+v", runs)
	}
}

func TestOpenURIWhenTheUserCancelsIsNotAnError(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	portalAnswers(t, b, 1)
	desktop.UseBus(bc, b)
	progs := &programs{}
	desktop.UseProgram(bc, progs.run)
	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "mailto:a@example.org") }); err != nil {
		t.Fatal(err)
	}
	if len(progs.all()) != 0 {
		t.Error("a cancelled chooser started xdg-open")
	}
}

func TestOpenURIFallsBackToXdgOpen(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, bc widgets.BuildContext)
	}{
		{"the portal answers that it could not", func(t *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			portalAnswers(t, b, 2)
			desktop.UseBus(bc, b)
		}},
		{"there is no portal on the bus", func(_ *testing.T, bc widgets.BuildContext) {
			desktop.UseBus(bc, fakebus.New(unique)) // every call fails with ServiceUnknown
		}},
		{"the portal fails the call", func(_ *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle(dbus.PortalDest, dbus.PortalPath, openURI, func(fakebus.Call) ([]any, error) {
				return nil, &godbus.Error{Name: "org.freedesktop.portal.Error.NotAllowed"}
			})
			desktop.UseBus(bc, b)
		}},
		{"there is no session bus", func(_ *testing.T, bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
				return nil, errors.New(tagDBusConnectFailed, "no bus")
			})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(t, bc)
			progs := &programs{}
			desktop.UseProgram(bc, progs.run)
			if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org") }); err != nil {
				t.Fatal(err)
			}
			runs := progs.all()
			if len(runs) != 1 || runs[0].name != "xdg-open" || len(runs[0].args) != 1 || runs[0].args[0] != "https://example.org" {
				t.Fatalf("runs = %+v", runs)
			}
			if len(runs[0].env) != 1 || runs[0].env[0] != "XDG_ACTIVATION_TOKEN=test-token" {
				t.Errorf("env = %v, want the activation token", runs[0].env)
			}
		})
	}
}

func TestOpenURIUnavailable(t *testing.T) {
	ts, bc := app(t)
	desktop.UseBus(bc, fakebus.New(unique))
	desktop.UseProgram(bc, (&programs{err: errors.New(tagProgramFailed, "xdg-open is not installed")}).run)
	err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org") })
	appErr := tagged(t, err, "OpenURIUnavailable")
	if appErr.Metadata["uri"] != "https://example.org" || appErr.Metadata["portalTag"] != "DBusCallFailed" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestOpenURIInvalid(t *testing.T) {
	ts, bc := app(t)
	for _, uri := range []string{"", "no-scheme", "/just/a/path", "https://a\nb", "ht tp://x", "://nothing"} {
		err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, uri) })
		assertTag(t, err, "OpenURIInvalid")
	}
	// Nothing went to the bus or to programs.
	b := fakebus.New(unique)
	desktop.UseBus(bc, b)
	_ = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "") })
	if len(b.Calls()) != 0 {
		t.Error("an invalid URI reached the bus")
	}
}

func TestOpenURIWhileTheUserHasNotAnswered(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	b.Handle(dbus.PortalDest, dbus.PortalPath, openURI, func(c fakebus.Call) ([]any, error) {
		return []any{dbus.PortalRequestPath(unique, tokenIn(t, c.Args[2].(map[string]godbus.Variant)))}, nil
	})
	var closed sync.WaitGroup
	closed.Add(1)
	b.HandleAnyPath(dbus.PortalDest, "org.freedesktop.portal.Request.Close", func(fakebus.Call) ([]any, error) {
		closed.Done()
		return nil, nil
	})
	desktop.UseBus(bc, b)
	progs := &programs{}
	desktop.UseProgram(bc, progs.run)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := inGoroutine(t, ts, func() error { return desktop.OpenURI(ctx, bc, "https://example.org") })
	assertTag(t, err, "PortalRequestAborted")
	closed.Wait() // the portal request was closed
	if len(progs.all()) != 0 {
		t.Error("xdg-open ran after the call was cancelled")
	}
}

func TestADesktopCallOnTheUIThreadFails(t *testing.T) {
	_, bc := app(t)
	err := desktop.OpenURI(context.Background(), bc, "https://example.org")
	assertTag(t, err, "DesktopCallOnUIThread")
}

func TestADesktopCallWithoutABinding(t *testing.T) {
	err := desktop.OpenURI(context.Background(), nil, "https://example.org")
	assertTag(t, err, "DesktopWithoutBinding")
}

func TestOpenURIGivesUpWhenThePlatformNeverAnswers(t *testing.T) {
	_, bc := app(t)
	// Nobody runs the posted functions, as when the UI thread is stuck.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	res := make(chan error, 1)
	go func() { res <- desktop.OpenURI(ctx, bc, "https://example.org") }()
	select {
	case err := <-res:
		assertTag(t, err, "DesktopCallCancelled")
	case <-time.After(5 * time.Second):
		t.Fatal("the call did not give up")
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagProgramFailed = "ProgramFailed"
)
