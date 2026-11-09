//go:build darwin

package dl

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
)

// MaxCallbacks is how many C→Go callbacks Callback creates before it panics with CallbackLimit.
// purego allows about 2,000 per process and never frees them, so this leaves a safety margin (SPEC §8.1
// rule 2, §29.2). The methods of the classes that Go registers for AppKit count as callbacks too.
//
// Flutter: no counterpart
const MaxCallbacks = 1500

// callbackRegistry counts the callbacks created and enforces the limit.
type callbackRegistry struct {
	mu       sync.Mutex
	limit    int
	count    int
	register func(fn any) uintptr
}

// processCallbacks is the registry of this process. The purego limit is per process, so there is one
// registry; it is the only package-level state, and it only counts.
var processCallbacks = &callbackRegistry{limit: MaxCallbacks, register: purego.NewCallback}

func (r *callbackRegistry) add(fn any) uintptr {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count >= r.limit {
		panic(errors.New(tagCallbackLimit, "too many C to Go callbacks; create them once at startup and dispatch by user data",
			"limit", r.limit))
	}
	ptr := r.register(fn)
	r.count++
	return ptr
}

// Callback converts a Go function into a C function pointer. Create callbacks once, at startup, and
// dispatch by user data: they are never freed, and a process can have only about 2,000. After
// MaxCallbacks it panics with CallbackLimit. A callback cannot take or return a structure by value
// (SPEC §29.2, §22).
//
// Flutter: no counterpart
func Callback(fn any) uintptr { return processCallbacks.add(fn) }

func (r *callbackRegistry) reserve(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count+n > r.limit {
		panic(errors.New(tagCallbackLimit, "too many C to Go callbacks; create them once at startup and dispatch by user data",
			"limit", r.limit, "have", r.count, "wanted", n))
	}
	r.count += n
}

// ReserveCallbacks counts n callbacks that are made outside Callback, by purego itself: the method
// implementations of a class that Go registers with the Objective-C runtime and the function types of
// blocks each cost one, and are never freed. It panics with CallbackLimit, and counts nothing, when
// that would pass MaxCallbacks.
//
// Flutter: no counterpart
func ReserveCallbacks(n int) { processCallbacks.reserve(n) }

// CallbackCount returns how many callbacks Callback and ReserveCallbacks have counted.
//
// Flutter: no counterpart
func CallbackCount() int {
	processCallbacks.mu.Lock()
	defer processCallbacks.mu.Unlock()
	return processCallbacks.count
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCallbackLimit = "CallbackLimit"
)
