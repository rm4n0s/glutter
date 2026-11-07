//go:build linux

package aaudio

import (
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The callbacks of AAudio are made once, with purego, and find their device by the number that was given as the user data (SPEC
// §8.1 rule 2). The table is the one piece of state of the process here: callbacks are never freed, so they cannot belong to a
// device.
var handleTable = struct {
	sync.RWMutex
	next    uintptr
	devices map[uintptr]*device
}{next: 1, devices: map[uintptr]*device{}}

// register gives a device its number.
func register(d *device) uintptr {
	handleTable.Lock()
	defer handleTable.Unlock()
	h := handleTable.next
	handleTable.next++
	handleTable.devices[h] = d
	return h
}

// unregister forgets a device; a callback that comes later for its number finds nothing.
func unregister(h uintptr) {
	handleTable.Lock()
	delete(handleTable.devices, h)
	handleTable.Unlock()
}

// deviceOf finds the device of a number.
func deviceOf(h uintptr) *device {
	handleTable.RLock()
	defer handleTable.RUnlock()
	return handleTable.devices[h]
}

// callbackSet holds the two C function pointers.
type callbackSet struct {
	data, err uintptr
}

var (
	callbackOnce sync.Once
	callbackPtrs callbackSet
)

// callbacks makes the callbacks, once (a function and not a variable that holds the result, because the callbacks refer to the
// functions that use them).
func callbacks() callbackSet {
	callbackOnce.Do(func() {
		callbackPtrs = callbackSet{
			data: dl.Callback(func(stream, user, audio uintptr, frames int32) int32 { return dispatchData(user, audio, frames) }),
			err:  dl.Callback(func(stream, user uintptr, result int32) { dispatchError(stream, user, result) }),
		}
	})
	return callbackPtrs
}

// dispatchData is the data callback: it runs on the audio thread of AAudio, and a panic of the handler is logged, makes the device
// lost and stops the stream (it must not unwind through C).
func dispatchData(user, audio uintptr, frames int32) (result int32) {
	d := deviceOf(user)
	if d == nil {
		return callbackStop
	}
	defer func() {
		if r := recover(); r != nil {
			d.log().Error("audio: the handler of an AAudio stream panicked", "panic", r, "stack", string(debug.Stack()))
			go d.lose()
			result = callbackStop
		}
	}()
	return d.onData(audio, frames)
}

// dispatchError is the error callback: it runs on a thread of AAudio, and only hands a disconnect to a goroutine (a stream cannot be
// closed from its own callbacks).
func dispatchError(stream, user uintptr, result int32) {
	d := deviceOf(user)
	if d == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error("audio: the error callback of an AAudio stream panicked", "panic", r)
		}
	}()
	d.onError(stream, result)
}
