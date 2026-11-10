// Package iostest is the harness of the iOS Simulator tests (SPEC §30.11): it keeps a device set of its own in `.ios-test/devices` at
// the root of the module (git-ignored, made with `xcrun simctl --set`), makes an iPhone and an iPad of the newest installed runtime on
// first use, boots them, installs and launches apps, reads their console and screenshots, and shuts everything down when the test
// ends. A test that uses it carries the glutter_ios build tag and runs with `-p 1`: two tests in one Simulator at a time. It imports
// `testing`, so no program imports it.
//
// Flutter: no counterpart
package iostest
