//go:build glutter_ibus && linux

package dbus_test

import (
	"context"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/internal/dbus"
)

// The tests in this file run the real IBus: ibus-daemon on a private bus of its own and
// ibus-portal on a private session bus (dbus-daemon), with a temporary home, so that
// neither the user's IBus nor the user's session bus is touched. They need the build tag
// glutter_ibus and skip when IBus is not installed:
//
//	go test -tags glutter_ibus -run RealIBus -v ./internal/dbus

// realIBus is an IBus started for a test.
type realIBus struct {
	// session is the address of the private session bus, where the portal is.
	session string
	// address is the address of the private bus of ibus-daemon.
	address string
}

// startRealIBus starts a session bus, ibus-daemon and ibus-portal, and stops them, with
// everything they started, when the test ends.
func startRealIBus(t *testing.T) realIBus {
	t.Helper()
	daemon, err := exec.LookPath("ibus-daemon")
	if err != nil {
		t.Skip("ibus-daemon is not installed")
	}
	portal := ""
	for _, p := range []string{"/usr/libexec/ibus-portal", "/usr/lib/ibus/ibus-portal", "/usr/lib/ibus-portal"} {
		if _, err := os.Stat(p); err == nil {
			portal = p
			break
		}
	}
	if portal == "" {
		t.Skip("ibus-portal is not installed")
	}
	session := startPrivateBus(t)
	dir := t.TempDir()
	address := "unix:path=" + filepath.Join(dir, "ibus")
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + dir,
		"XDG_CONFIG_HOME=" + filepath.Join(dir, "config"),
		"XDG_CACHE_HOME=" + filepath.Join(dir, "cache"),
		"XDG_DATA_HOME=" + filepath.Join(dir, "data"),
		"XDG_RUNTIME_DIR=" + dir,
		"DBUS_SESSION_BUS_ADDRESS=" + session,
		"IBUS_ADDRESS=" + address,
		"GSETTINGS_BACKEND=memory",
	}
	start := func(path string, args ...string) {
		t.Helper()
		cmd := exec.Command(path, args...)
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // so that its children are stopped with it
		if err := cmd.Start(); err != nil {
			t.Fatalf("starting %s: %v", path, err)
		}
		t.Cleanup(func() {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) // the group may have ended already
			done := make(chan struct{})
			go func() {
				_ = cmd.Wait() // killed: Wait reports the signal
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // it did not stop on SIGTERM
				<-done
			}
		})
	}
	start(daemon, "--single", "--panel=disable", "--emoji-extension=disable", "--config=disable",
		"--cache=none", "--address="+address)
	waitFor(t, "ibus-daemon", func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		bus, err := dbus.ConnectAddress(ctx, address)
		if err != nil {
			return false
		}
		_ = bus.Close() // a probe
		return true
	})
	start(portal)
	probe := dial(t, session)
	waitFor(t, "ibus-portal", func() bool {
		var owned bool
		err := probe.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, "org.freedesktop.portal.IBus").Store(&owned)
		return err == nil && owned
	})
	return realIBus{session: session, address: address}
}

func waitFor(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatalf("%s did not start", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestRealIBusClosesTheInputContextAfterItsContextEnds is the quit of an app with IBus
// (Emetos's issues of 2026-11-17): the input method bridge cancels its context, and then
// closes the input method. Closing destroys the input context without an error, on the
// private bus of ibus-daemon (whose connection must not end with the context it was opened
// with) and on the portal (whose input contexts have Destroy on org.freedesktop.IBus.Service
// only), and the portal's context is gone afterwards.
func TestRealIBusClosesTheInputContextAfterItsContextEnds(t *testing.T) {
	ibus := startRealIBus(t)
	tests := []struct {
		name string
		// direct is whether the input method goes to the private bus of ibus-daemon.
		direct bool
	}{
		{"private bus", true},
		{"portal", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionConn := dial(t, ibus.session)
			session := dbus.NewBus(sessionConn)
			opts := dbus.IMEOptions{Hint: "ibus", Program: "glutter-test"}
			if tt.direct {
				opts.IBusBus = func(ctx context.Context) (dbus.Bus, error) { return dbus.ConnectAddress(ctx, ibus.address) }
			}
			// The bridge's context: it lives until the app quits.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			im, err := dbus.OpenInputMethod(ctx, session, opts, func(dbus.IMEEvent) {})
			if err != nil {
				t.Fatal(err)
			}
			callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
			defer callCancel()
			if err := im.FocusIn(callCtx); err != nil {
				t.Fatalf("FocusIn: %v", err)
			}
			path := portalContexts(t, sessionConn, tt.direct)

			cancel() // the app quits
			if err := im.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if tt.direct {
				return // the context ended with the connection, which Close closed
			}
			// The session bus is the app's and stays open: the portal's context must be gone.
			err = sessionConn.Object("org.freedesktop.portal.IBus", path).
				Call("org.freedesktop.IBus.InputContext.FocusIn", 0).Err
			if err == nil {
				t.Errorf("the portal's input context %s is still there after Close", path)
			}
		})
	}
}

// portalContexts returns the path of the input context that the portal made for conn: the
// one object under /org/freedesktop/IBus that introspection shows. It returns "" for the
// private bus.
func portalContexts(t *testing.T, conn *godbus.Conn, direct bool) godbus.ObjectPath {
	t.Helper()
	if direct {
		return ""
	}
	var data string
	err := conn.Object("org.freedesktop.portal.IBus", "/org/freedesktop/IBus").
		Call("org.freedesktop.DBus.Introspectable.Introspect", 0).Store(&data)
	if err != nil {
		t.Fatalf("introspecting the portal: %v", err)
	}
	var node struct {
		Children []struct {
			Name string `xml:"name,attr"`
		} `xml:"node"`
	}
	if err := xml.Unmarshal([]byte(data), &node); err != nil {
		t.Fatalf("reading the introspection of the portal: %v", err)
	}
	if len(node.Children) == 0 {
		t.Fatalf("the portal shows no input context:\n%s", data)
	}
	return godbus.ObjectPath("/org/freedesktop/IBus/" + node.Children[len(node.Children)-1].Name)
}
