// Package fakebus is an in-process D-Bus for tests (SPEC §17). It records
// every method call, answers calls with handlers the test registers, and
// delivers the signals the test emits. It implements dbus.Bus.
//
// Flutter: no counterpart
package fakebus

import (
	"context"
	"reflect"
	"sort"
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

	exported map[exportKey]any
	signals  []godbus.Signal
	names    map[string]bool
	// taken are the names that RequestName finds owned by another connection.
	taken map[string]bool
}

type exportKey struct {
	path  godbus.ObjectPath
	iface string
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
	return &Bus{unique: unique, handlers: map[string]Handler{}, subs: map[*subscription]struct{}{},
		exported: map[exportKey]any{}, names: map[string]bool{}, taken: map[string]bool{}}
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
		return nil, errors.NewErr(tagDBusCallFailed, &godbus.Error{
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
		return nil, errors.New(tagFakeBusClosed, "the fake bus is closed")
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

// Export records the receiver; Invoke calls its methods.
func (b *Bus) Export(receiver any, path godbus.ObjectPath, iface string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New(tagFakeBusClosed, "the fake bus is closed")
	}
	b.exported[exportKey{path, iface}] = receiver
	return nil
}

// Unexport forgets the receiver.
func (b *Bus) Unexport(path godbus.ObjectPath, iface string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.exported, exportKey{path, iface})
}

// Exported says whether something is exported as iface on path.
//
// Flutter: no counterpart
func (b *Bus) Exported(path godbus.ObjectPath, iface string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.exported[exportKey{path, iface}]
	return ok
}

// EmitSignal records the signal and delivers it to the matching subscriptions,
// with this connection as the sender.
func (b *Bus) EmitSignal(path godbus.ObjectPath, name string, values ...any) error {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		return errors.New(tagFakeBusClosed, "the fake bus is closed")
	}
	sig := &godbus.Signal{Sender: b.unique, Path: path, Name: name, Body: values}
	b.mu.Lock()
	b.signals = append(b.signals, *sig)
	b.mu.Unlock()
	b.Emit(sig)
	return nil
}

// Signals are the signals that were sent with EmitSignal, oldest first.
//
// Flutter: no counterpart
func (b *Bus) Signals() []godbus.Signal {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]godbus.Signal(nil), b.signals...)
}

// RequestName takes the name, unless the test made it Taken.
func (b *Bus) RequestName(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.taken[name] {
		return errors.New(tagDBusNameTaken, "another connection owns the name", "name", name)
	}
	b.names[name] = true
	return nil
}

// ReleaseName gives the name up.
func (b *Bus) ReleaseName(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.names, name)
	return nil
}

// Names are the well-known names this connection owns.
//
// Flutter: no counterpart
func (b *Bus) Names() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	names := make([]string, 0, len(b.names))
	for n := range b.names {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Take makes RequestName fail for the name, as if another connection owned it.
//
// Flutter: no counterpart
func (b *Bus) Take(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.taken[name] = true
}

// Invoke calls the method of what is exported as iface on path, the way the bus
// would for a call from another connection: the arguments are converted to the
// types of the method, and a *godbus.Error that the method returns is the error.
// It returns the other values of the reply.
//
// Flutter: no counterpart
func (b *Bus) Invoke(path godbus.ObjectPath, iface, method string, args ...any) ([]any, error) {
	b.mu.Lock()
	recv, ok := b.exported[exportKey{path, iface}]
	b.mu.Unlock()
	if !ok {
		return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownObject", Body: []any{string(path) + " " + iface}}
	}
	m := reflect.ValueOf(recv).MethodByName(method)
	if !m.IsValid() {
		return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownMethod", Body: []any{method}}
	}
	mt := m.Type()
	in := make([]reflect.Value, 0, len(args)+1)
	if mt.NumIn() == len(args)+1 && mt.In(0) == reflect.TypeOf(godbus.Sender("")) {
		in = append(in, reflect.ValueOf(godbus.Sender(b.unique)))
	}
	if mt.NumIn() != len(in)+len(args) {
		return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.InvalidArgs", Body: []any{method}}
	}
	for i, a := range args {
		want := mt.In(len(in))
		v, ok := convert(reflect.ValueOf(a), want)
		if !ok {
			return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.InvalidArgs", Body: []any{method, i}}
		}
		in = append(in, v)
	}
	out := m.Call(in)
	var reply []any
	for _, o := range out {
		if e, isErr := o.Interface().(*godbus.Error); isErr {
			if e != nil {
				return nil, e
			}
			continue
		}
		reply = append(reply, o.Interface())
	}
	return reply, nil
}

var _ dbus.Bus = (*Bus)(nil)

// sameClass says whether a value of kind a can be converted to kind b the way the
// bus would: numbers to numbers (a literal 1 for an int32), or the same kind.
func sameClass(a, b reflect.Kind) bool {
	isNumber := func(k reflect.Kind) bool { return k >= reflect.Int && k <= reflect.Float64 }
	return a == b || (isNumber(a) && isNumber(b))
}

// convert makes v a value of type want the way decoding a message would: numbers
// to numbers, slices element by element, structs with the same fields.
func convert(v reflect.Value, want reflect.Type) (reflect.Value, bool) {
	switch {
	case !v.IsValid():
		return reflect.Zero(want), true
	case v.Type().AssignableTo(want):
		return v, true
	case v.Kind() == reflect.Slice && want.Kind() == reflect.Slice:
		out := reflect.MakeSlice(want, v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			e, ok := convert(v.Index(i), want.Elem())
			if !ok {
				return reflect.Value{}, false
			}
			out.Index(i).Set(e)
		}
		return out, true
	case sameClass(v.Kind(), want.Kind()) && v.Type().ConvertibleTo(want):
		return v.Convert(want), true
	}
	return reflect.Value{}, false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDBusCallFailed = "DBusCallFailed"
	tagDBusNameTaken  = "DBusNameTaken"
	tagFakeBusClosed  = "FakeBusClosed"
)
