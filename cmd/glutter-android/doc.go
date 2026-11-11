// Command glutter-android builds, installs, runs and takes screenshots of Glutter apps on
// Android (SPEC §24.1):
//
//	glutter-android build [-o app.apk] [-abi arm64,amd64] [-debug] [-app-id id] [-label name] [-icon icon.png] [-permission NAME]... <package>
//	glutter-android install [-device serial] <package|app.apk>
//	glutter-android run [-device serial] [-env GLUTTER_X=v]... <package>
//	glutter-android shot [-avd glutter-37 | -device serial] [-renderer r] [-env GLUTTER_X=v]... [-screen] -o file.png <package>
//	glutter-android perf [-avd glutter-37 | -device serial] [-debug] [-abi arm64,amd64] [-renderer r] [-env GLUTTER_X=v]... [-starts 5] [-settle 8s] [-seconds 10] [-frames=true] [-format text|markdown] [-o report] <package>
//
// build, install, run and perf make release builds (stripped, foundation.ReleaseMode) unless -debug is
// given; shot makes a debug build (-debug=false for a release), since the hooks it drives
// (GLUTTER_DEV_*) are read only by debug builds. Without -key and -cert the APK is signed with the
// debug key and is debuggable (adb run-as works); with them it is a store APK, not debuggable.
//
// perf builds the package (each ABI by default), installs it and prints a
// report of the numbers of SPEC §15: the size of the APK per ABI and of its libapp.so, the cold
// start (am start -W and the engine's first-frame time), the idle CPU, the frame times of a
// scenario (GLUTTER_FRAME_STATS=1 with the dev events of -env GLUTTER_DEV_EVENTS=..., which
// only debug builds read) and the memory. scripts/perf-android.sh runs it for the examples.
//
// It needs the Android SDK and NDK (ANDROID_HOME) and a Java runtime for apksigner.
//
// Flutter: no counterpart (flutter build apk, flutter run)
package main
