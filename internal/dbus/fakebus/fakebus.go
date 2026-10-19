// Package fakebus is an in-process D-Bus for tests (SPEC §17). It records
// every method call, answers calls with handlers the test registers, and
// delivers the signals the test emits. It implements dbus.Bus.
//
// Flutter: no counterpart
package fakebus

import (
	"context"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
)

// Call is one recorded method call.
//
// Flutter: no counterpart
type Call struct {
	// Dest is the name the call went to.
	Dest string
	// Path is the object path of the call.
	Path godbus.ObjectPath
	// Method is the full method name, such as "org.freedesktop.portal.Settings.ReadAll".
	Method string
	// Args are the arguments.
	Args []any
}

// Handler answers a call: it returns the values of the reply, or an error.
// A *godbus.Error makes the call fail the way a bus error does.
//
// Flutter: no counterpart
type Handler func(call Call) ([]any, error)

// Bus is the fake bus. It implements dbus.Bus.
//
// Flutter: no counterpart
type Bus struct {
	unique string

	mu       sync.Mutex
	handlers map[string]Handler
	calls    []Call
	subs     map[*subscription]struct{}
	closed   bool
}

type subscription struct {
	match dbus.SignalMatch
	ch    chan *godbus.Signal
}

// New creates a fake bus whose connection has the unique name unique, such as
// ":1.42".
//
// Flutter: no counterpart
func New(unique string) *Bus {
	return &Bus{unique: unique, handlers: map[string]Handler{}, subs: map[*subscription]struct{}{}}
}

func handlerKey(dest string, path godbus.ObjectPath, method string) string {
	return dest + "|" + string(path) + "|" + method
}

// Handle makes h answer the calls to dest, path and method. A later Handle for
// the same triple replaces the earlier one.
//
// Flutter: no counterpart
func (b *Bus) Handle(dest string, path godbus.ObjectPath, method string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[handlerKey(dest, path, method)] = h
}

// HandleAnyPath makes h answer the calls to dest and method on any object path
// that has no handler of its own.
//
// Flutter: no counterpart
func (b *Bus) HandleAnyPath(dest, method string, h Handler) {
	b.Handle(dest, anyPath, method, h)
}

// anyPath is the key under which HandleAnyPath stores a handler. It is not a
// valid object path, so no real call can use it.
const anyPath godbus.ObjectPath = "*"

// Calls returns the calls made so far, in order.
//
// Flutter: no counterpart
func (b *Bus) Calls() []Call {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Call(nil), b.calls...)
}

// UniqueName is the unique name given to New.
func (b *Bus) UniqueName() string { return b.unique }

// Call records the call and runs its handler. A call nobody handles fails with
// org.freedesktop.DBus.Error.ServiceUnknown, as if the service were missing.
func (b *Bus) Call(ctx context.Context, dest string, path godbus.ObjectPath, method string, args ...any) ([]any, error) {
	c := Call{Dest: dest, Path: path, Method: method, Args: args}
	b.mu.Lock()
	b.calls = append(b.calls, c)
	h := b.handlers[handlerKey(dest, path, method)]
	if h == nil {
		h = b.handlers[handlerKey(dest, anyPath, method)]
	}
	b.mu.Unlock()
	if h == nil {
		return nil, errors.NewErr("DBusCallFailed", &godbus.Error{
			Name: "org.freedesktop.DBus.Error.ServiceUnknown",
			Body: []any{"The name " + dest + " was not provided by any .service files"},
		}).SetMetadata("dest", dest)
	}
	return h(c)
}

// Subscribe delivers the emitted signals that match until ctx is done.
func (b *Bus) Subscribe(ctx context.Context, m dbus.SignalMatch) (<-chan *godbus.Signal, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("FakeBusClosed", "the fake bus is closed")
	}
	s := &subscription{match: m, ch: make(chan *godbus.Signal, 64)}
	b.subs[s] = struct{}{}
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[s]; ok {
			delete(b.subs, s)
			close(s.ch)
		}
	}()
	return s.ch, nil
}

// Emit delivers sig to every matching subscription.
//
// Flutter: no counterpart
func (b *Bus) Emit(sig *godbus.Signal) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.subs {
		if s.match.Matches(sig) {
			select {
			case s.ch <- sig:
			default:
			}
		}
	}
}

// Subscribers is the number of open subscriptions.
//
// Flutter: no counterpart
func (b *Bus) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// Close closes every subscription.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for s := range b.subs {
		delete(b.subs, s)
		close(s.ch)
	}
	return nil
}

var _ dbus.Bus = (*Bus)(nil)
