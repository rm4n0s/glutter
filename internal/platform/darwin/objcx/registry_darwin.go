//go:build darwin

package objcx

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// Handle names a Go object that an Objective-C instance stands for. The instance keeps it in an ivar
// and the methods of its class find the object by it (rule 2 of SPEC §8.1, §29.2): Objective-C never
// holds a Go pointer. 0 is never a handle.
//
// Flutter: no counterpart
type Handle uintptr

// handles is the table of the Go objects that Objective-C instances stand for. It is per process, not
// per platform: purego never frees a callback and a class cannot be registered twice, so the methods
// of a class are made once and every instance of it, whichever platform made it, finds its object
// here. An entry is deleted by DropHandle when the instance goes away, so the table only holds what is
// alive (a memory test checks HandleCount).
var handles struct {
	mu   sync.RWMutex
	next atomic.Uintptr
	m    map[Handle]any
}

// NewHandle stores v and returns its handle.
//
// Flutter: no counterpart
func NewHandle(v any) Handle {
	h := Handle(handles.next.Add(1))
	handles.mu.Lock()
	if handles.m == nil {
		handles.m = map[Handle]any{}
	}
	handles.m[h] = v
	handles.mu.Unlock()
	return h
}

// Lookup returns the object of a handle, or false when the handle is 0 or was dropped.
//
// Flutter: no counterpart
func Lookup(h Handle) (any, bool) {
	handles.mu.RLock()
	defer handles.mu.RUnlock()
	v, ok := handles.m[h]
	return v, ok
}

// LookupAs is Lookup for an object of type T: false when the handle is gone or holds another type.
//
// Flutter: no counterpart
func LookupAs[T any](h Handle) (T, bool) {
	v, ok := Lookup(h)
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

// DropHandle deletes the handle. Dropping a handle that is gone does nothing.
//
// Flutter: no counterpart
func DropHandle(h Handle) {
	handles.mu.Lock()
	delete(handles.m, h)
	handles.mu.Unlock()
}

// HandleCount is the number of handles that are alive, for the memory tests.
//
// Flutter: no counterpart
func HandleCount() int {
	handles.mu.RLock()
	defer handles.mu.RUnlock()
	return len(handles.m)
}

// handleIvar is the name of the ivar that holds the Handle of an instance. Every class that Go
// registers has it.
const handleIvar = "goHandle"

// Method is a method that Go implements: Name is the selector ("mouseDown:"), and Fn a function whose
// first two arguments are the object (objc.ID) and the selector (objc.SEL), then the arguments of the
// method, and whose result, if any, is the return value. Fn cannot take or return a structure by value
// (a purego callback cannot, SPEC §22).
//
// Flutter: no counterpart
type Method struct {
	Name string
	Fn   any
}

// ClassDef describes a class that Go implements.
//
// Flutter: no counterpart
type ClassDef struct {
	// Name is the name without the prefix, such as "View"; the class is called Glutter<Name>_<pid>
	// (SPEC §29.2), so that two copies of the module in one process cannot clash.
	Name string
	// Super is the name of the superclass, such as "NSView".
	Super string
	// Protocols are the names of the protocols the class adopts, such as "NSTextInputClient". A name
	// that the runtime does not know is an error.
	Protocols []string
	// Methods are the methods that Go implements.
	Methods []Method
}

// RegisteredClass is a class that RegisterClass made.
//
// Flutter: no counterpart
type RegisteredClass struct {
	// Name is the name that the runtime knows the class by (Glutter<Name>_<pid>).
	Name string
	// Class is the class.
	Class objc.Class
}

// className is the name of the class that a ClassDef makes in this process.
func className(def ClassDef) string { return fmt.Sprintf("Glutter%s_%d", def.Name, os.Getpid()) }

// RegisterClass registers the class with the Objective-C runtime. It reserves the callbacks of its
// methods in package dl first (CallbackLimit panics when there are too many). It fails with
// MacOSClassMissing when the superclass is not known, and with MacOSClassRegistrationFailed, and the
// metadata "class", when the runtime refuses: the name is taken (a class is made once per process), a
// protocol is unknown, or a method's function has the wrong shape.
//
// Flutter: no counterpart
func RegisterClass(def ClassDef) (*RegisteredClass, error) {
	name := className(def)
	super, err := Class(def.Super)
	if err != nil {
		return nil, err // MacOSClassMissing names the superclass
	}
	var protocols []*objc.Protocol
	for _, p := range def.Protocols {
		proto := objc.GetProtocol(p)
		if proto == nil {
			return nil, errors.New(tagMacOSClassRegistrationFailed, "a protocol is not known to the runtime",
				"class", name, "protocol", p)
		}
		protocols = append(protocols, proto)
	}
	methods := make([]objc.MethodDef, len(def.Methods))
	for i, m := range def.Methods {
		methods[i] = objc.MethodDef{Cmd: Sel(m.Name), Fn: m.Fn}
	}
	ivars := []objc.FieldDef{{Name: handleIvar, Type: reflect.TypeFor[uintptr](), Attribute: objc.ReadWrite}}
	// One callback for each method, and one for each of the getter and setter of the handle ivar.
	dl.ReserveCallbacks(len(methods) + 2)
	class, err := objc.RegisterClass(name, super, protocols, ivars, methods)
	if err != nil {
		return nil, errors.NewErr(tagMacOSClassRegistrationFailed, err).SetMetadata("class", name)
	}
	return &RegisteredClass{Name: name, Class: class}, nil
}

// New makes an instance of the class with [[cls alloc] init] (+1: release it) and gives it the handle.
//
// Flutter: no counterpart
func (c *RegisteredClass) New(h Handle) objc.ID {
	obj := objc.ID(c.Class).Send(Sel("alloc")).Send(Sel("init"))
	SetHandle(obj, h)
	return obj
}

// SetHandle stores the handle in an instance of a class that RegisterClass made.
//
// Flutter: no counterpart
func SetHandle(obj objc.ID, h Handle) {
	obj.SetIvar(obj.Class().InstanceVariable(handleIvar), objc.ID(h))
}

// HandleOf reads the handle of an instance of a class that RegisterClass made: the first thing that
// a method of such a class does. It is 0 for an instance that has none.
//
// Flutter: no counterpart
func HandleOf(obj objc.ID) Handle {
	return Handle(obj.GetIvar(obj.Class().InstanceVariable(handleIvar)))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSClassRegistrationFailed = "MacOSClassRegistrationFailed"
)
