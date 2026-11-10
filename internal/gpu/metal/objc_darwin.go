//go:build darwin

package metal

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The glue to the Objective-C runtime that the device uses: selectors by name, messages with a typed
// result, the loading of the frameworks and the two C functions that find the devices.

func sel(name string) objc.SEL { return objcx.Sel(name) }

// send sends the message and returns its result as T.
func send[T any](id objc.ID, name string, args ...any) T { return objc.Send[T](id, sel(name), args...) }

// call sends the message and returns the object it answers.
func call(id objc.ID, name string, args ...any) objc.ID { return id.Send(sel(name), args...) }

// sendBool sends a message that answers a BOOL.
func sendBool(id objc.ID, name string, args ...any) bool {
	return objc.Send[int8](id, sel(name), args...) != 0
}

// class returns the class of that name.
func class(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

// newObject makes an instance of the class with `new`: an object that the caller owns.
func newObject(className string) objc.ID { return call(class(className), "new") }

// retain and release are the reference counting of an object that the device keeps.
func retain(id objc.ID) objc.ID {
	if id != 0 {
		objcx.Retain(id)
	}
	return id
}

func release(id objc.ID) {
	if id != 0 {
		objcx.Release(id)
	}
}

// description is the localized description of an NSError, or the description of any object.
func description(e objc.ID) string {
	if e == 0 {
		return ""
	}
	if d := call(e, "localizedDescription"); d != 0 {
		return objcx.String(d)
	}
	return objcx.String(call(e, "description"))
}

// metalFuncs are the functions of the Metal framework.
type metalFuncs struct {
	createSystemDefault func() objc.ID
	copyAllDevices      func() objc.ID
}

// metalFramework loads Metal and QuartzCore (which has CAMetalLayer) once.
var metalFramework = sync.OnceValues(func() (*metalFuncs, error) {
	if _, err := dl.OpenFramework("QuartzCore"); err != nil {
		return nil, err
	}
	lib, err := dl.OpenFramework("Metal")
	if err != nil {
		return nil, err
	}
	f := new(metalFuncs)
	if err := lib.Bind(&f.createSystemDefault, "MTLCreateSystemDefaultDevice"); err != nil {
		return nil, err
	}
	if lib.Has("MTLCopyAllDevices") {
		if err := lib.Bind(&f.copyAllDevices, "MTLCopyAllDevices"); err != nil {
			f.copyAllDevices = nil
		}
	}
	return f, nil
})
