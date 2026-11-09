// Command glutter-macos builds, runs and takes screenshots of Glutter apps on macOS (SPEC §29.1). In this
// step it has `shot`: it builds the app and runs it with GLUTTER_SCREENSHOT, so that the app saves its own
// first settled frame (proof method G, SPEC §29.13) with no screen recording permission. `build`, `run`
// and `perf` come with step 33.15.
//
// Flutter: no counterpart
package main
