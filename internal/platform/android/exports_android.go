package android

/*
#include <stdint.h>
*/
import "C"

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

//export goFocus
func goFocus(focused C.int) { process.focusChanged(focused != 0) }
