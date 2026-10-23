// Package androidtool builds, installs and signs Android apps for glutter-android (SPEC
// §24.1): it finds the SDK and the NDK, generates the manifest, runs the Go toolchain once
// per ABI, links the resources with aapt2, packs the APK with the embedding's classes.dex,
// aligns and signs it. It runs on the developer's machine, never inside an app, and needs
// no Gradle, Android Studio, AndroidX or Kotlin.
//
// The Java classes of the embedding are in java/ and their compiled form is the checked-in
// classes.dex, which `glutter-gen android-dex` makes (D22).
//
// Flutter: no counterpart (flutter build apk)
package androidtool
