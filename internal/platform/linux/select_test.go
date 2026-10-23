//go:build linux

package linux

import (
	"bytes"
	"context"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// fakeConn is a Connection that only knows its name.
type fakeConn struct {
	platform.Platform
	name   string
	closed bool
}

func (c *fakeConn) Close() error { c.closed = true; return nil }

// attempt is a fake backend: it records that it was called and returns the
// connection or the error.
type attempt struct {
	err   error
	calls int
	conn  *fakeConn
}

func (a *attempt) fn() func() (Connection, error) {
	return func() (Connection, error) {
		a.calls++
		if a.err != nil {
			return nil, a.err
		}
		return a.conn, nil
	}
}

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestSelect(t *testing.T) {
	waylandErr := errors.New(tagWaylandConnectFailed, "no compositor", "display", "wayland-0")
	libErr := errors.New(tagLibraryLoadFailed, "no libwayland-client")
	x11Err := errors.New(tagX11OpenDisplayFailed, "no server")

	tests := []struct {
		name        string
		option      string
		env         map[string]string
		wayland     error
		x11         error
		want        Backend
		wantErrTag  string
		wantWayland int // calls
		wantX11     int
		wantInLog   []string
	}{
		{
			name: "Wayland first when the session has it",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"},
			want: BackendWayland, wantWayland: 1,
			wantInLog: []string{"backend chosen", "backend=wayland", "WAYLAND_DISPLAY=wayland-0"},
		},
		{
			name: "X11 when only DISPLAY is set",
			env:  map[string]string{"DISPLAY": ":0"},
			want: BackendX11, wantX11: 1,
			wantInLog: []string{"backend=x11", "DISPLAY=:0"},
		},
		{
			name: "X11 after Wayland cannot connect",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, wayland: waylandErr,
			want: BackendX11, wantWayland: 1, wantX11: 1,
			wantInLog: []string{"Wayland is not usable", "WaylandConnectFailed", "after Wayland failed"},
		},
		{
			name: "X11 after the Wayland libraries do not load",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":1"}, wayland: libErr,
			want: BackendX11, wantWayland: 1, wantX11: 1,
		},
		{
			name: "Wayland alone when only WAYLAND_DISPLAY is set",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-1"},
			want: BackendWayland, wantWayland: 1,
		},
		{
			name:       "NoDisplay when nothing is set",
			env:        map[string]string{},
			wantErrTag: "NoDisplay",
		},
		{
			name: "NoDisplay when both fail", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"},
			wayland: waylandErr, x11: x11Err, wantErrTag: "NoDisplay", wantWayland: 1, wantX11: 1,
		},
		{
			name: "NoDisplay when the only one fails", env: map[string]string{"DISPLAY": ":0"},
			x11: x11Err, wantErrTag: "NoDisplay", wantX11: 1,
		},
		{
			name: "GLUTTER_BACKEND=x11 wins over a Wayland session",
			env:  map[string]string{BackendEnv: "x11", "WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"},
			want: BackendX11, wantX11: 1,
			wantInLog: []string{"GLUTTER_BACKEND=x11"},
		},
		{
			name: "GLUTTER_BACKEND=wayland ignores DISPLAY",
			env:  map[string]string{BackendEnv: "wayland", "DISPLAY": ":0"},
			want: BackendWayland, wantWayland: 1,
		},
		{
			name:   "the option wins over the environment variable",
			option: "wayland",
			env:    map[string]string{BackendEnv: "x11", "WAYLAND_DISPLAY": "w"},
			want:   BackendWayland, wantWayland: 1,
			wantInLog: []string{"the Backend option=wayland"},
		},
		{
			name: "an explicit backend that fails is not replaced",
			env:  map[string]string{BackendEnv: "wayland", "WAYLAND_DISPLAY": "w", "DISPLAY": ":0"}, wayland: waylandErr,
			wantErrTag: "WaylandConnectFailed", wantWayland: 1,
		},
		{
			name: "an explicit X11 that fails is not replaced",
			env:  map[string]string{BackendEnv: "x11", "WAYLAND_DISPLAY": "w", "DISPLAY": ":0"}, x11: x11Err,
			wantErrTag: "X11OpenDisplayFailed", wantX11: 1,
		},
		{
			name:       "an unknown backend",
			env:        map[string]string{BackendEnv: "quartz", "DISPLAY": ":0"},
			wantErrTag: "BackendUnknown",
		},
		{
			name: "an unknown backend in the option", option: "mir", env: map[string]string{"DISPLAY": ":0"},
			wantErrTag: "BackendUnknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			way := &attempt{err: tt.wayland, conn: &fakeConn{name: "wayland"}}
			x := &attempt{err: tt.x11, conn: &fakeConn{name: "x11"}}
			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, nil))
			conn, got, err := Select(Selection{
				Backend: tt.option, Getenv: env(tt.env), Wayland: way.fn(), X11: x.fn(), Logger: log,
			})
			if way.calls != tt.wantWayland || x.calls != tt.wantX11 {
				t.Errorf("calls: wayland %d, x11 %d; want %d and %d", way.calls, x.calls, tt.wantWayland, tt.wantX11)
			}
			if tt.wantErrTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantErrTag {
					t.Fatalf("error = %v, want tag %s", err, tt.wantErrTag)
				}
				if conn != nil || got != "" {
					t.Errorf("a connection came with the error: %v %q", conn, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want || conn.(*fakeConn).name != string(tt.want) {
				t.Errorf("backend %q (connection %q), want %q", got, conn.(*fakeConn).name, tt.want)
			}
			for _, s := range tt.wantInLog {
				if !strings.Contains(logs.String(), s) {
					t.Errorf("the log lacks %q:\n%s", s, logs.String())
				}
			}
		})
	}
}

func TestSelectRoutes(t *testing.T) {
	tests := []struct {
		name  string
		sel   Selection
		route string
	}{
		{"nothing to connect to", Selection{Getenv: env(nil)}, "Select.NoDisplay"},
		{"an unknown backend", Selection{Getenv: env(map[string]string{BackendEnv: "quartz"})}, "Select.BackendUnknown"},
		{"a backend this build lacks", Selection{Getenv: env(map[string]string{BackendEnv: "x11"})}, "Select->connect.BackendUnavailable"},
		{"a build without Wayland picked by the option", Selection{Backend: "wayland", Getenv: env(nil)}, "Select->connect.BackendUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sel.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
			_, _, err := Select(tt.sel)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

func TestNoDisplayCarriesWhatWasTried(t *testing.T) {
	way := &attempt{err: errors.New(tagWaylandProtocolMissing, "no xdg_wm_base", "interface", "xdg_wm_base")}
	x := &attempt{err: errors.New(tagX11OpenDisplayFailed, "no server", "display", ":0")}
	_, _, err := Select(Selection{
		Getenv:  env(map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}),
		Wayland: way.fn(), X11: x.fn(),
		Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	})
	appErr, _ := errors.FromError(err)
	wantTried := []string{
		"wayland: WaylandProtocolMissing: no xdg_wm_base",
		"x11: X11OpenDisplayFailed: no server",
	}
	if got, _ := appErr.Metadata["tried"].([]string); !reflect.DeepEqual(got, wantTried) {
		t.Errorf("tried = %v, want %v", appErr.Metadata["tried"], wantTried)
	}
	if appErr.Metadata["WAYLAND_DISPLAY"] != "wayland-0" || appErr.Metadata["DISPLAY"] != ":0" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestDescribePlainErrors(t *testing.T) {
	if got := describe(context.DeadlineExceeded); got != context.DeadlineExceeded.Error() {
		t.Errorf("describe = %q", got)
	}
}

func TestSelectUsesTheProcessEnvironmentByDefault(t *testing.T) {
	t.Setenv(BackendEnv, "x11")
	x := &attempt{conn: &fakeConn{name: "x11"}}
	_, b, err := Select(Selection{X11: x.fn(), Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))})
	if err != nil || b != BackendX11 {
		t.Errorf("Select = %q, %v", b, err)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLibraryLoadFailed      = "LibraryLoadFailed"
	tagWaylandConnectFailed   = "WaylandConnectFailed"
	tagWaylandProtocolMissing = "WaylandProtocolMissing"
	tagX11OpenDisplayFailed   = "X11OpenDisplayFailed"
)
