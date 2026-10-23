// Package androidtest is the harness of the device tests (SPEC §24.13): it creates the virtual
// devices of the emulator in .android-test/ at the module root, starts one headless in a
// process group of its own under `timeout -k`, with its own adb server and its own HOME so that
// nothing of the user's ~/.android is read or written, waits for it to boot, and stops it
// (and the adb server) afterwards. It also installs and starts apps and reads their results.
//
// glutter-android shot uses it too, to start an emulator when no device is given.
//
// Flutter: no counterpart (flutter drive)
package androidtest
