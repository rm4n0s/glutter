// Command glutter-ios builds, installs, runs and takes the frames of Glutter apps for iOS (SPEC §30.1): `build` makes the .app bundle
// (for the Simulator with -sim, or for a device with -team and -profile), `run` installs it on a Simulator device and streams its
// console, and `shot` saves a picture of the app (proof method H). Building needs Xcode (the full app, not the Command Line Tools).
// build and run make release builds unless -debug is given; shot makes a debug build (-debug=false for a release), since the
// hooks it drives (GLUTTER_DEV_*) are read only by debug builds.
package main
