//go:build linux

// Package asensor is the motion sensors backend of Android (SPEC §28.4, step 31.7): the accelerometer and the gyroscope through the
// sensor manager of the NDK (libandroid.so, Android 8, API 26), called through purego, with the events of every open sensor read on
// a looper that lives on an OS thread of its own. The design follows the Android sensor driver of SDL 3
// (src/sensor/android/SDL_androidsensor.c; zlib license, third_party/sdl/LICENSE), translated to Go and changed: the default rate
// is 60 readings a second or the sensor's slowest allowed, the app may ask for another, the thread starts at the first open and ends
// with the service, the readings of an event queue are handed over in batches with their own timestamps (SDL stamps them with the
// time at which the thread woke up), and a sensor that is closed is never called again.
//
// The package has only the tag of the Linux kernel (Android has it too), so that its tests run on the desktop with a scripted library,
// as the tests of the AAudio backend do; it is used on Android only.
//
// Flutter: no counterpart
package asensor

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of libandroid (SPEC §8.1).
const LibraryEnv = "GLUTTER_ANDROID_LIBRARY"

// The values of the NDK's <android/sensor.h> and <android/looper.h>.
const (
	typeAccelerometer = 1 // ASENSOR_TYPE_ACCELEROMETER
	typeGyroscope     = 4 // ASENSOR_TYPE_GYROSCOPE

	looperPrepareAllowNonCallbacks = 1 // ALOOPER_PREPARE_ALLOW_NON_CALLBACKS

	pollWake     = -1 // ALOOPER_POLL_WAKE
	pollCallback = -2 // ALOOPER_POLL_CALLBACK
	pollTimeout  = -3 // ALOOPER_POLL_TIMEOUT
	pollError    = -4 // ALOOPER_POLL_ERROR

	// looperIdent is the identifier that the event queues give to the looper (SDL's LOOPER_ID_USER).
	looperIdent = 3
)

// The layout of ASensorEvent (<android/sensor.h>): the numbers of the fields that the backend reads.
const (
	eventSize      = 104 // sizeof(ASensorEvent)
	eventTimestamp = 16  // offsetof(ASensorEvent, timestamp)
	eventData      = 24  // offsetof(ASensorEvent, data)
)

// library holds the bound functions of libandroid. The manager, the sensor, the queue and the looper are uintptr; the buffers are C
// memory.
type library struct {
	getInstanceForPackage func(pkg uintptr) uintptr
	getDefaultSensor      func(mgr uintptr, kind int32) uintptr
	createEventQueue      func(mgr, looper uintptr, ident int32, callback, data uintptr) uintptr
	destroyEventQueue     func(mgr, queue uintptr) int32
	enableSensor          func(queue, sensor uintptr) int32
	disableSensor         func(queue, sensor uintptr) int32
	setEventRate          func(queue, sensor uintptr, usec int32) int32
	getEvents             func(queue, events uintptr, count uintptr) int64
	getMinDelay           func(sensor uintptr) int32

	looperPrepare func(opts int32) uintptr
	looperPoll    func(timeoutMillis int32, fd, events, data uintptr) int32
	looperWake    func(looper uintptr)
	looperAcquire func(looper uintptr)
	looperRelease func(looper uintptr)
}

// loadLibrary opens libandroid.so and binds the functions. It fails with SensorUnavailable when the library cannot be loaded (a
// desktop) or lacks a function of API 26.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libandroid.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(platform.TagSensorUnavailable, "could not load libandroid", "names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.getInstanceForPackage, "ASensorManager_getInstanceForPackage"},
		{&l.getDefaultSensor, "ASensorManager_getDefaultSensor"},
		{&l.createEventQueue, "ASensorManager_createEventQueue"},
		{&l.destroyEventQueue, "ASensorManager_destroyEventQueue"},
		{&l.enableSensor, "ASensorEventQueue_enableSensor"},
		{&l.disableSensor, "ASensorEventQueue_disableSensor"},
		{&l.setEventRate, "ASensorEventQueue_setEventRate"},
		{&l.getEvents, "ASensorEventQueue_getEvents"},
		{&l.getMinDelay, "ASensor_getMinDelay"},
		{&l.looperPrepare, "ALooper_prepare"},
		{&l.looperPoll, "ALooper_pollOnce"},
		{&l.looperWake, "ALooper_wake"},
		{&l.looperAcquire, "ALooper_acquire"},
		{&l.looperRelease, "ALooper_release"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			md := map[string]any{}
			if appErr, ok := errors.FromError(err); ok {
				md = appErr.Metadata
			}
			return nil, errors.New(platform.TagSensorUnavailable, "libandroid lacks a function", "symbol", md["symbol"])
		}
	}
	return l, nil
}
