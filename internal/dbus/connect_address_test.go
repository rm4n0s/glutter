//go:build !js

package dbus_test

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	rmerrors "github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
)

// godbus has no unix transport on js/wasm, so this file is not built there.

// TestConnectAddressStopsWaitingWhenItsContextEnds: ctx still bounds the wait for the
// connection, the authentication included, now that the connection is not made with it: a
// bus that accepts the connection and never answers ends in DBusConnectFailed when ctx
// ends.
func TestConnectAddressStopsWaitingWhenItsContextEnds(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "silent")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() }) // the test is over
	var accepted []net.Conn
	var mu sync.Mutex
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return // closed at the end of the test
			}
			mu.Lock()
			accepted = append(accepted, c) // kept open, never answered
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range accepted {
			_ = c.Close() // the test is over
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = dbus.ConnectAddress(ctx, "unix:path="+socket)
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("ConnectAddress returned after %v, its context ended after 200ms", took)
	}
	assertTag(t, err, "ConnectAddress->dialWithoutContext.DBusConnectFailed")
	appErr, _ := rmerrors.FromError(err)
	if appErr != nil && appErr.Metadata["address"] != "unix:path="+socket {
		t.Errorf("metadata = %v, want the address", appErr.Metadata)
	}
}
