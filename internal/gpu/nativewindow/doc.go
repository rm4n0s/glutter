// Package nativewindow is the last-resort renderer of Android (SPEC §24.4): the software
// renderer (package soft) draws the frame, and it is copied into the CPU buffers of the
// window's ANativeWindow (setBuffersGeometry, lock, unlockAndPost, loaded from libandroid.so
// through purego). It is to Android what canvas2d is to the web, and it also draws the first
// frames before the GPU renderers exist.
//
// Flutter: no counterpart (Flutter's Android embedding has the Skia software surface,
// shell/platform/android/android_surface_software.cc)
package nativewindow
