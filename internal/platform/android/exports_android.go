package android

/*
#include <stdint.h>
*/
import "C"

import "unsafe"

// The entry points the JNI stubs of jni_android.c call, on a thread of Android (the main
// thread). Each copies its values out and hands them to the bridge, which queues them for the UI
// thread (SPEC §24.3).

//export goSurfaceCreated
func goSurfaceCreated(window C.uintptr_t) { process.surfaceCreated(uintptr(window)) }

//export goSurfaceChanged
func goSurfaceChanged(width, height C.int, density, refresh C.float, displayWidth, displayHeight C.int) {
	process.surfaceChanged(int(width), int(height), float64(density), float64(refresh), int(displayWidth), int(displayHeight))
}

//export goSurfaceDestroyed
func goSurfaceDestroyed() { process.surfaceDestroyed() }

//export goLifecycle
func goLifecycle(state C.int) { process.lifecycleChanged(lifecycleState(state)) }

//export goMotionEvent
func goMotionEvent(meta *C.longlong, nmeta C.int, ids *C.int, nids C.int, vals *C.float, nvals C.int, times *C.longlong, ntimes C.int,
	pressureMin, pressureMax, hScroll, vScroll C.float) {
	ev, generic, ok := decodeMotion(
		unsafe.Slice((*int64)(unsafe.Pointer(meta)), int(nmeta)), unsafe.Slice((*int32)(unsafe.Pointer(ids)), int(nids)),
		unsafe.Slice((*float32)(unsafe.Pointer(vals)), int(nvals)), unsafe.Slice((*int64)(unsafe.Pointer(times)), int(ntimes)),
		float32(pressureMin), float32(pressureMax))
	if ok {
		process.motion(ev, generic, float64(hScroll), float64(vScroll))
	}
}

//export goKeyEvent
func goKeyEvent(id, action, keyCode, scanCode, metaState, repeatCount C.int, eventTime C.longlong, unicodeChar, source, deviceID C.int) {
	process.key(int(id), keyEvent{
		Action: int32(action), KeyCode: int32(keyCode), ScanCode: int32(scanCode), MetaState: int32(metaState),
		RepeatCount: int32(repeatCount), EventTime: int64(eventTime), UnicodeChar: int32(unicodeChar),
		Source: int32(source), DeviceID: int32(deviceID),
	})
}

//export goFocus
func goFocus(focused C.int) { process.focusChanged(focused != 0) }

//export goInsets
func goInsets(values *C.int, n C.int) {
	if in, ok := decodeInsets(unsafe.Slice((*int32)(unsafe.Pointer(values)), int(n))); ok {
		process.insetsChanged(in)
	}
}

//export goSettings
func goSettings(text *C.char) { process.settingsChanged(C.GoString(text)) }

//export goSystemUI
func goSystemUI(visible C.int) { process.systemUIChanged(visible != 0) }

//export goDisplayChanged
func goDisplayChanged(density, refresh C.float, displayWidth, displayHeight C.int) {
	process.displayChanged(float64(density), float64(refresh), int(displayWidth), int(displayHeight))
}
