//go:build windows

package uiatest

import (
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// EventKind says which handler an Event came through.
//
// Flutter: no counterpart
type EventKind int

// The kinds of Event.
const (
	// EventFocus: the focus changed (IUIAutomationFocusChangedEventHandler).
	//
	// Flutter: no counterpart
	EventFocus EventKind = iota + 1
	// EventProperty: a property changed (IUIAutomationPropertyChangedEventHandler).
	//
	// Flutter: no counterpart
	EventProperty
	// EventAutomation: another event (IUIAutomationEventHandler).
	//
	// Flutter: no counterpart
	EventAutomation
	// EventStructure: the structure changed (IUIAutomationStructureChangedEventHandler).
	//
	// Flutter: no counterpart
	EventStructure
)

// Event is something a handler was told. Sender holds a reference of the element, which the test
// releases; Name is its name when the handler could read it.
//
// Flutter: no counterpart
type Event struct {
	Kind     EventKind
	Sender   *Element
	Name     string
	EventID  uia.EventID
	Property uia.PropertyID
	Value    uia.Value
	Change   int
}

// Listener holds the handlers that the test registered and the events they got.
//
// Flutter: no counterpart
type Listener struct {
	c      *Client
	env    *handlerEnv
	events chan Event

	mu   sync.Mutex
	objs []win32.Object
}

// handlerEnv is the Host and the interface types of the four handlers, made once per process (a callback
// is never freed).
type handlerEnv struct {
	host                                   *win32.Host
	focus, property, automation, structure *win32.InterfaceType
}

var sharedHandlers = sync.OnceValues(newHandlerEnv)

func newHandlerEnv() (*handlerEnv, error) {
	host, err := win32.NewHost(nil)
	if err != nil {
		return nil, err
	}
	e := &handlerEnv{host: host}
	mk := func(iface uia.Interface, args int, fn func(l *Listener, a []uintptr)) (*win32.InterfaceType, error) {
		return host.NewInterface(win32.InterfaceDef{Name: iface.Name, IID: iface.IID, Methods: []win32.MethodDef{{
			Name: iface.Methods[0], Args: args, Fn: func(impl any, a []uintptr) uintptr {
				fn(impl.(*Listener), a)
				return win32.Result(win32.SOK)
			}}}})
	}
	if e.focus, err = mk(IfaceFocusHandler, 1, func(l *Listener, a []uintptr) { l.push(Event{Kind: EventFocus}, a[0]) }); err != nil {
		return nil, err
	}
	if e.property, err = mk(IfacePropertyHandler, 3, func(l *Listener, a []uintptr) {
		ev := Event{Kind: EventProperty, Property: uia.PropertyID(int32(a[1]))}
		if a[2] != 0 {
			v := *win32.View[win32.Variant](a[2])
			switch v.Vt {
			case win32.VTBool:
				ev.Value = uia.Bool(uint16(v.Val) != 0)
			case win32.VTI4:
				ev.Value = uia.Int(int(int32(v.Val)))
			case win32.VTR8:
				ev.Value = uia.Float(math.Float64frombits(v.Val))
			case win32.VTBSTR:
				ev.Value = uia.String(win32.BSTRToString(uintptr(v.Val)))
			}
		}
		l.push(ev, a[0])
	}); err != nil {
		return nil, err
	}
	if e.automation, err = mk(IfaceEventHandler, 2, func(l *Listener, a []uintptr) {
		l.push(Event{Kind: EventAutomation, EventID: uia.EventID(int32(a[1]))}, a[0])
	}); err != nil {
		return nil, err
	}
	if e.structure, err = mk(IfaceStructureHandler, 3, func(l *Listener, a []uintptr) {
		l.push(Event{Kind: EventStructure, Change: int(int32(a[1]))}, a[0])
	}); err != nil {
		return nil, err
	}
	return e, nil
}

// push queues an event with the sender, which it takes a reference of. The handler runs on a thread of
// UI Automation's; the name is read later, by Wait, on the thread of the test: a call into UI Automation
// from inside the handler can wait for the thread that raised the event.
func (l *Listener) push(ev Event, sender uintptr) {
	if sender != 0 {
		o := win32.ObjectFromPtr(sender)
		o.AddRef()
		ev.Sender = &Element{c: l.c, obj: o}
	}
	select {
	case l.events <- ev:
	default: // nobody reads: drop it
		ev.Sender.Release()
	}
}

// NewListener makes a listener for the client.
//
// Flutter: no counterpart
func (c *Client) NewListener() (*Listener, error) {
	env, err := sharedHandlers()
	if err != nil {
		return nil, err
	}
	return &Listener{c: c, env: env, events: make(chan Event, 256)}, nil
}

// handler makes the COM object of a handler type that calls back into the listener.
func (l *Listener) handler(t *win32.InterfaceType) (win32.Object, error) {
	o, err := l.env.host.NewObject(l, t)
	if err != nil {
		return win32.Object{}, err
	}
	l.mu.Lock()
	l.objs = append(l.objs, o)
	l.mu.Unlock()
	return o, nil
}

// OnFocus subscribes to the focus changes of the desktop (AddFocusChangedEventHandler).
//
// Flutter: no counterpart
func (l *Listener) OnFocus() error {
	h, err := l.handler(l.env.focus)
	if err != nil {
		return err
	}
	return call(l.c.auto, IfaceAutomation, "AddFocusChangedEventHandler", 0, h.Ptr())
}

// OnProperty subscribes to the changes of properties of the element and its descendants.
//
// Flutter: no counterpart
func (l *Listener) OnProperty(root *Element, props ...uia.PropertyID) error {
	h, err := l.handler(l.env.property)
	if err != nil {
		return err
	}
	ids := make([]int32, len(props))
	for i, p := range props {
		ids[i] = int32(p)
	}
	sa, err := win32.SafeArrayCreateVector(win32.VTI4, 0, uint32(len(ids)))
	if err != nil {
		return err
	}
	defer win32.SafeArrayDestroy(sa)
	for i := range ids {
		if err := win32.SafeArrayPutElement(sa, int32(i), unsafe.Pointer(&ids[i])); err != nil {
			return err
		}
	}
	return call(l.c.auto, IfaceAutomation, "AddPropertyChangedEventHandler", root.obj.Ptr(), TreeScopeSubtree, 0, h.Ptr(), sa)
}

// OnAutomation subscribes to an event of the element and its descendants.
//
// Flutter: no counterpart
func (l *Listener) OnAutomation(id uia.EventID, root *Element) error {
	h, err := l.handler(l.env.automation)
	if err != nil {
		return err
	}
	return call(l.c.auto, IfaceAutomation, "AddAutomationEventHandler", uintptr(id), root.obj.Ptr(), TreeScopeSubtree, 0, h.Ptr())
}

// OnStructure subscribes to the structure changes of the element and its descendants.
//
// Flutter: no counterpart
func (l *Listener) OnStructure(root *Element) error {
	h, err := l.handler(l.env.structure)
	if err != nil {
		return err
	}
	return call(l.c.auto, IfaceAutomation, "AddStructureChangedEventHandler", root.obj.Ptr(), TreeScopeSubtree, 0, h.Ptr())
}

// Wait returns the first event that match accepts, or false after the timeout. The other events that come
// before it are dropped (and their senders released).
//
// Flutter: no counterpart
func (l *Listener) Wait(match func(Event) bool, timeout time.Duration) (Event, bool) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case ev := <-l.events:
			if ev.Sender != nil {
				if name, err := ev.Sender.Name(); err == nil {
					ev.Name = name
				}
			}
			if match(ev) {
				return ev, true
			}
			ev.Sender.Release()
		case <-deadline.C:
			return Event{}, false
		}
	}
}

// Close removes the handlers and releases them.
//
// Flutter: no counterpart
func (l *Listener) Close() error {
	err := call(l.c.auto, IfaceAutomation, "RemoveAllEventHandlers")
	l.mu.Lock()
	objs := l.objs
	l.objs = nil
	l.mu.Unlock()
	for _, o := range objs {
		o.Release()
	}
	return err
}
