// Package android is the platform backend for Android (GOOS=android, SPEC §24): the JNI entry
// points the Java embedding calls, the thread that runs the program's main, the log lines sent
// to logcat, and, from step 25.3, the looper, the surface, the input and the services.
//
// It is the only package with cgo (D21), and only in the *_android.go files; the files without
// that suffix are plain Go that builds and is tested everywhere (SPEC §24.13). No C type, C
// pointer or JNI reference leaves the package.
//
// Flutter: shell/platform/android (the embedding of the engine)
package android
