//go:build darwin

package objcx

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// selectors caches the selectors that Sel made: purego's RegisterName takes the global lock of the
// Objective-C runtime, and a selector never changes. It only holds what the runtime would answer again.
var selectors sync.Map // string -> objc.SEL

// Sel returns the selector of that name, registering it the first time.
//
// Flutter: no counterpart
func Sel(name string) objc.SEL {
	if v, ok := selectors.Load(name); ok {
		return v.(objc.SEL)
	}
	sel := objc.RegisterName(name)
	selectors.Store(name, sel)
	return sel
}

// foundation loads Foundation once: the classes of NSString and the like exist in the process only
// after it. A system without it cannot run Glutter, so a failure panics with the tagged error.
var foundation = sync.OnceValue(func() *dl.Lib {
	lib, err := dl.OpenFramework("Foundation")
	if err != nil {
		panic(err)
	}
	return lib
})

// Init loads Foundation, so that its classes can be found. The functions of the package call it on
// first use; a program that wants the failure as a value calls Init at start.
//
// Flutter: no counterpart
func Init() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	foundation()
	return nil
}

// Class returns the Objective-C class of that name, such as "NSString". It fails with
// MacOSClassMissing when the process has no such class (a framework that is not loaded, or an AppKit
// class on iOS).
//
// Flutter: no counterpart
func Class(name string) (objc.Class, error) {
	foundation()
	c := objc.GetClass(name)
	if c == 0 {
		return 0, errors.New(tagMacOSClassMissing, "the Objective-C runtime has no such class", "class", name)
	}
	return c, nil
}

// MustClass is Class for a class that the system always has: it panics with the tagged error when
// the class is missing.
//
// Flutter: no counterpart
func MustClass(name string) objc.Class {
	c, err := Class(name)
	if err != nil {
		panic(err)
	}
	return c
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSClassMissing = "MacOSClassMissing"
)
