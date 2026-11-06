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

//export goActivityCreated
func goActivityCreated() { process.activityCreated() }

//export goBack
func goBack(kind C.int, x, y, progress C.float, edge C.int) {
	process.back(int(kind), float64(x), float64(y), float64(progress), int(edge))
}

//export goTrimMemory
func goTrimMemory(level C.int) { process.trimMemory(int(level)) }

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

//export goPointerCapture
func goPointerCapture(captured C.int) { process.pointerCapture(captured != 0) }

//export goCapturedMotion
func goCapturedMotion(action C.int, dx, dy C.float, buttons C.int, eventTime C.longlong) {
	process.capturedMotion(int(action), float64(dx), float64(dy), int(buttons), int64(eventTime))
}

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

//export goTextEdit
func goTextEdit(seq C.int, utf8 *C.char, n C.int, base, extent, composeBase, composeExtent C.int) {
	process.textEdit(int(seq), C.GoStringN(utf8, n), int(base), int(extent), int(composeBase), int(composeExtent))
}

//export goClipboardText
func goClipboardText(id, ok C.int, utf8 *C.char, n C.int) {
	process.clipboardText(int(id), ok != 0, C.GoStringN(utf8, n))
}

//export goA11yState
func goA11yState(enabled, touch, navigation C.int) {
	process.a11yChanged(a11yState{enabled: enabled != 0, touch: touch != 0, navigation: navigation != 0})
}

//export goA11yAction
func goA11yAction(nodeID, action C.int, args *C.char, n C.int) {
	process.a11yAction(int(nodeID), int(action), C.GoBytes(unsafe.Pointer(args), n))
}

//export goDesktopResult
func goDesktopResult(id, code C.int, utf8 *C.char, n C.int) {
	process.desktopResult(int(id), int(code), C.GoStringN(utf8, n))
}

//export goSpellCheckResult
func goSpellCheckResult(id, code C.int, utf8 *C.char, n C.int) {
	process.spellCheckResult(int(id), int(code), C.GoStringN(utf8, n))
}

//export goSpellCheckSupport
func goSpellCheckSupport(supported C.int) { process.spellCheckSupport(supported != 0) }

//export goNotificationEvent
func goNotificationEvent(serial, kind C.int, utf8 *C.char, n C.int) {
	process.notificationEvent(int(serial), int(kind), C.GoStringN(utf8, n))
}

//export goPadAdded
func goPadAdded(device C.int, name *C.char, nname C.int, desc *C.char, ndesc C.int, vendor, product, flags C.int, axes *C.int, naxes C.int,
	ranges *C.float, nranges C.int) {
	d := padDescriptor{
		DeviceID: int(device), Name: C.GoStringN(name, nname), Descriptor: C.GoStringN(desc, ndesc),
		Vendor: int(vendor), Product: int(product), Flags: int(flags),
	}
	for _, a := range unsafe.Slice((*int32)(unsafe.Pointer(axes)), int(naxes)) {
		d.Axes = append(d.Axes, int(a))
	}
	d.Ranges = append(d.Ranges, unsafe.Slice((*float32)(unsafe.Pointer(ranges)), int(nranges))...)
	process.padAdded(d)
}

//export goPadRemoved
func goPadRemoved(device C.int) { process.padRemoved(int(device)) }

//export goPadSync
func goPadSync(devices *C.int, n C.int) {
	var ids []int
	for _, d := range unsafe.Slice((*int32)(unsafe.Pointer(devices)), int(n)) {
		ids = append(ids, int(d))
	}
	process.padSync(ids)
}

//export goPadKey
func goPadKey(device, down, keyCode C.int, eventTime C.longlong) {
	process.padKey(int(device), down != 0, int(keyCode), int64(eventTime))
}

//export goPadMotion
func goPadMotion(device C.int, vals *C.float, n C.int, eventTime C.longlong) {
	process.padMotion(int(device), unsafe.Slice((*float32)(unsafe.Pointer(vals)), int(n)), int64(eventTime))
}
