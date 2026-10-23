//go:build linux && !android

package wl

import (
	"sync"

	"github.com/rm4n0s/errors"
)

// object is implemented by every generated interface type.
type object interface {
	// base returns the embedded Proxy.
	base() *Proxy
	// dispatch decodes event number opcode from args and calls the handler.
	dispatch(opcode uint32, args eventArgs)
}

// Object is anything that wraps a wl_proxy, such as a *WlSurface. It is the
// type of the arguments that the protocol leaves without an interface.
//
// Flutter: no counterpart
type Object interface {
	// Ptr returns the wl_proxy pointer, or 0 for a nil object.
	Ptr() uintptr
}

// Proxy is the part that every generated interface type shares: the wl_proxy
// of libwayland and the link to the Go object that receives its events.
//
// Flutter: no counterpart
type Proxy struct {
	conn    *Conn
	ptr     uintptr
	handle  uintptr
	version uint32
	dead    bool
}

func (p *Proxy) base() *Proxy { return p }

// handleTable maps the integer user data of a proxy to its Go object. libwayland
// is given the integer, never a Go pointer (SPEC §8.1). There is one table
// because there is one dispatcher callback for the whole process; the table is
// only touched on the UI thread, and the mutex is a guard against misuse.
var handles = struct {
	sync.Mutex
	next uintptr
	m    map[uintptr]object
}{next: 1, m: map[uintptr]object{}}

func addHandle(o object) uintptr {
	handles.Lock()
	defer handles.Unlock()
	h := handles.next
	handles.next++
	handles.m[h] = o
	return h
}

func getHandle(h uintptr) object {
	handles.Lock()
	defer handles.Unlock()
	return handles.m[h]
}

func removeHandle(h uintptr) {
	handles.Lock()
	defer handles.Unlock()
	delete(handles.m, h)
}

// handleCount returns how many objects are registered, for tests.
func handleCount() int {
	handles.Lock()
	defer handles.Unlock()
	return len(handles.m)
}

// register links a Go object to a wl_proxy: it gets a handle as user data and,
// when the interface has events, the shared dispatcher.
func (c *Conn) register(o object, proxy uintptr, hasEvents bool) {
	p := o.base()
	p.conn = c
	p.ptr = proxy
	p.version = c.api.proxyGetVersion(proxy)
	p.handle = addHandle(o)
	c.live[p.handle] = struct{}{}
	c.api.proxySetUserData(proxy, p.handle)
	if hasEvents {
		if c.api.proxyAddDispatcher(proxy, dispatcherPointer(), p.handle, p.handle) != 0 {
			panic(errors.New(tagWaylandDispatcherFailed, "wl_proxy_add_dispatcher failed: the proxy already has a listener",
				"id", c.api.proxyGetID(proxy)))
		}
	}
}

// forget drops the Go side of a proxy that libwayland has destroyed or that is
// about to be.
func (p *Proxy) forget() {
	if p.dead {
		return
	}
	p.dead = true
	removeHandle(p.handle)
	delete(p.conn.live, p.handle)
}

// Conn returns the connection the object belongs to.
//
// Flutter: no counterpart
func (p *Proxy) Conn() *Conn { return p.conn }

// ProxyID returns the object ID on the wire.
//
// Flutter: no counterpart
func (p *Proxy) ProxyID() uint32 { return p.conn.api.proxyGetID(p.ptr) }

// ProxyVersion returns the version of the interface that the object was
// created with.
//
// Flutter: no counterpart
func (p *Proxy) ProxyVersion() uint32 { return p.version }

// ProxyDestroy destroys the client side of the object without sending a
// request. Objects with a destructor request are destroyed by calling it
// instead. Destroying twice does nothing.
//
// Flutter: no counterpart
func (p *Proxy) ProxyDestroy() {
	if p.dead {
		return
	}
	p.forget()
	p.conn.api.proxyDestroy(p.ptr)
}

// marshal sends a request. iface and version describe the object that a
// new_id argument creates (iface is nil when there is none). It returns the
// wl_proxy of the new object, or 0.
func (p *Proxy) marshal(opcode uint32, iface *Interface, version, flags uint32, a *args) uintptr {
	if p.dead {
		panic(errors.New(tagWaylandObjectDestroyed, "request sent to an object that was destroyed",
			"opcode", opcode))
	}
	var ifacePtr uintptr
	if iface != nil {
		ifacePtr = iface.ptr
	}
	ret := p.conn.api.proxyMarshalArrayFlags(p.ptr, opcode, ifacePtr, version, flags, a.ptr)
	if flags&flagDestroy != 0 {
		p.forget() // libwayland has destroyed the proxy
	}
	return ret
}

// dispatchEvent is the one dispatcher of all proxies. libwayland calls it with
// the implementation pointer given to wl_proxy_add_dispatcher, which here is
// the handle of the Go object. A panic in a handler must not unwind through C
// frames, so it is kept and raised again when the call into libwayland returns.
func dispatchEvent(implementation, target, opcode, message, args uintptr) int32 {
	_, _ = target, message
	o := getHandle(implementation)
	if o == nil {
		return 0 // the object was destroyed while the event was queued
	}
	conn := o.base().conn
	defer func() {
		if r := recover(); r != nil && conn.panicValue == nil {
			conn.panicValue = &panicBox{value: r}
		}
	}()
	o.dispatch(uint32(opcode), eventArgs(args))
	return 0
}

// panicBox holds a recovered panic value, which may itself be nil-looking.
type panicBox struct{ value any }

// lookup returns the Go object for a wl_proxy pointer as type T, or the zero
// value when proxy is NULL, is not one of ours, or is not a T.
func lookup[T any](c *Conn, proxy uintptr) T {
	var zero T
	if proxy == 0 {
		return zero
	}
	o := getHandle(c.api.proxyGetUserData(proxy))
	if o == nil {
		return zero
	}
	t, _ := o.(T)
	return t
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandDispatcherFailed = "WaylandDispatcherFailed"
	tagWaylandObjectDestroyed  = "WaylandObjectDestroyed"
)
