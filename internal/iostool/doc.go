// Package iostool holds what `glutter-ios` does that does not need a Mac to be understood: the Info.plist of an iOS app, the layout
// of its bundle, its icons, the command lines of `simctl`, and the build of the program with the compiler of Xcode (SPEC §30.1).
// Everything here but the build and the signing runs on any host, so the tests of the plist, the layout and the command lines run
// everywhere; the build needs Xcode and the tests of it skip without.
//
// Flutter: no counterpart
package iostool
