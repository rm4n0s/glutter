package android

/*
#include <android/api-level.h>
*/
import "C"

// SDKInt is the API level of the device that runs the app (Build.VERSION.SDK_INT), from the NDK's
// android_get_device_api_level. The engine decides with it whether Vulkan is used (SPEC §24.4).
//
// Flutter: embedding/android FlutterJNI (Build.VERSION.SDK_INT)
func SDKInt() int { return int(C.android_get_device_api_level()) }
