// Command glutter-android builds, installs, runs and takes screenshots of Glutter apps on
// Android (SPEC §24.1):
//
//	glutter-android build [-o app.apk] [-abi arm64,amd64] [-release] [-app-id id] [-label name] [-icon icon.png] [-permission NAME]... <package>
//	glutter-android install [-device serial] <package|app.apk>
//	glutter-android run [-device serial] [-env GLUTTER_X=v]... <package>
//	glutter-android shot [-avd glutter-37 | -device serial] [-renderer r] [-env GLUTTER_X=v]... [-screen] -o file.png <package>
//
// It needs the Android SDK and NDK (ANDROID_HOME) and a Java runtime for apksigner.
//
// Flutter: no counterpart (flutter build apk, flutter run)
package main
