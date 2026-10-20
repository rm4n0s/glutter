package dbus

import (
	"context"
	"strings"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
)

// SignalMatch selects the signals a Subscribe delivers. An empty field
// matches every value.
//
// Flutter: no counterpart
type SignalMatch struct {
	// Sender is the well-known or unique name of the sender.
	Sender string
	// Path is the object path the signal comes from.
	Path godbus.ObjectPath
	// Interface is the interface of the signal.
	Interface string
	// Member is the name of the signal.
	Member string
}

// Matches reports whether sig is selected by the path, interface and member
// of the match. The sender is not compared: a real bus filters by sender
// (which may be a well-known name that the signal does not carry), and the
// fake bus has one sender.
//
// Flutter: no counterpart
func (m SignalMatch) Matches(sig *godbus.Signal) bool {
	if m.Path != "" && m.Path != sig.Path {
		return false
	}
	switch {
	case m.Interface != "" && m.Member != "":
		return sig.Name == m.Interface+"."+m.Member
	case m.Interface != "":
		return strings.HasPrefix(sig.Name, m.Interface+".")
	case m.Member != "":
		return strings.HasSuffix(sig.Name, "."+m.Member)
	}
	return true
}

// Bus is the part of a D-Bus connection that the services use. The real
// implementation is Connect; the in-process fake of the tests is
// fakebus.Bus.
//
// Flutter: no counterpart
type Bus interface {
	// UniqueName is the unique name (":1.42") of this connection.
	UniqueName() string
	// Call calls a method and returns the values of the reply. The method is
	// the full name, such as "org.freedesktop.portal.Settings.ReadAll".
	Call(ctx context.Context, dest string, path godbus.ObjectPath, method string, args ...any) ([]any, error)
	// Subscribe delivers the signals that match until ctx is done, then closes
	// the channel. The subscription exists when Subscribe returns, so a signal
	// sent after that is not missed.
	Subscribe(ctx context.Context, match SignalMatch) (<-chan *godbus.Signal, error)
	// Close closes the connection.
	Close() error
}

// Connect opens the session bus. The connection is private to the caller;
// close it with Close.
//
// Flutter: no counterpart
func Connect(ctx context.Context) (Bus, error) {
	conn, err := godbus.ConnectSessionBus(godbus.WithContext(ctx))
	if err != nil {
		return nil, errors.NewErr("DBusConnectFailed", err)
	}
	return &sessionBus{conn: conn}, nil
}

// NewBus wraps an open connection, for example one to a private test bus.
//
// Flutter: no counterpart
func NewBus(conn *godbus.Conn) Bus { return &sessionBus{conn: conn} }

// sessionBus is the Bus on a godbus connection.
type sessionBus struct {
	conn *godbus.Conn
}

func (c *sessionBus) UniqueName() string {
	names := c.conn.Names()
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func (c *sessionBus) Call(ctx context.Context, dest string, path godbus.ObjectPath, method string, args ...any) ([]any, error) {
	call := c.conn.Object(dest, path).CallWithContext(ctx, method, 0, args...)
	if call.Err != nil {
		return nil, errors.NewErr("DBusCallFailed", call.Err).
			SetMetadata("dest", dest).SetMetadata("path", string(path)).SetMetadata("method", method)
	}
	return call.Body, nil
}

func (c *sessionBus) Subscribe(ctx context.Context, match SignalMatch) (<-chan *godbus.Signal, error) {
	opts := []godbus.MatchOption{godbus.WithMatchObjectPath(match.Path)}
	if match.Path == "" {
		opts = nil
	}
	if match.Sender != "" {
		opts = append(opts, godbus.WithMatchSender(match.Sender))
	}
	if match.Interface != "" {
		opts = append(opts, godbus.WithMatchInterface(match.Interface))
	}
	if match.Member != "" {
		opts = append(opts, godbus.WithMatchMember(match.Member))
	}
	raw := make(chan *godbus.Signal, 64)
	c.conn.Signal(raw)
	if err := c.conn.AddMatchSignalContext(ctx, opts...); err != nil {
		c.conn.RemoveSignal(raw)
		return nil, errors.NewErr("DBusSubscribeFailed", err).
			SetMetadata("interface", match.Interface).SetMetadata("member", match.Member)
	}
	out := make(chan *godbus.Signal, 16)
	go c.forward(ctx, match, opts, raw, out)
	return out, nil
}

// forward passes the signals of raw that match to out, and cleans up when ctx
// is done. godbus blocks while a signal channel is full, so raw is drained
// while it is being removed.
func (c *sessionBus) forward(ctx context.Context, match SignalMatch, opts []godbus.MatchOption, raw chan *godbus.Signal, out chan<- *godbus.Signal) {
	defer close(out)
	for {
		select {
		case <-ctx.Done():
			c.unsubscribe(opts, raw)
			return
		case sig, ok := <-raw:
			if !ok {
				// godbus closes the signal channels when the connection is closed.
				return
			}
			if !match.Matches(sig) {
				continue
			}
			select {
			case out <- sig:
			case <-ctx.Done():
				c.unsubscribe(opts, raw)
				return
			}
		}
	}
}

func (c *sessionBus) unsubscribe(opts []godbus.MatchOption, raw chan *godbus.Signal) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.conn.RemoveSignal(raw)
		// The match rule is removed on a best-effort basis: the connection may
		// already be closed, and the bus drops the rule with it.
		_ = c.conn.RemoveMatchSignal(opts...)
	}()
	drain := raw // the goroutine above uses raw
	for {
		select {
		case _, ok := <-drain:
			if !ok {
				drain = nil // closed with the connection: nothing left to drain
			}
		case <-done:
			return
		}
	}
}

func (c *sessionBus) Close() error {
	if err := c.conn.Close(); err != nil {
		return errors.NewErr("DBusCloseFailed", err)
	}
	return nil
}

// ConnectAddress opens a bus at an address, such as the private bus of the IBus
// daemon ("unix:path=/run/user/1000/ibus/dbus-xyz,guid=..."). The connection
// is private to the caller; close it with Close. It fails with
// DBusAddressDialFailed, DBusAddressAuthFailed or DBusAddressHelloFailed.
//
// Flutter: no counterpart
func ConnectAddress(ctx context.Context, address string) (Bus, error) {
	conn, err := godbus.Dial(address, godbus.WithContext(ctx))
	if err != nil {
		return nil, errors.NewErr("DBusAddressDialFailed", err).SetMetadata("address", address)
	}
	if err := conn.Auth(nil); err != nil {
		_ = conn.Close() // the authentication error is the one to report
		return nil, errors.NewErr("DBusAddressAuthFailed", err).SetMetadata("address", address)
	}
	if err := conn.Hello(); err != nil {
		_ = conn.Close() // the hello error is the one to report
		return nil, errors.NewErr("DBusAddressHelloFailed", err).SetMetadata("address", address)
	}
	return &sessionBus{conn: conn}, nil
}
