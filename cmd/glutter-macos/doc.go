// Command glutter-macos builds, runs, takes screenshots of and measures Glutter apps on macOS (SPEC §29.1), as glutter-windows does
// for Windows: `build` makes the signed app bundle (the Info.plist, the .icns icon, a universal executable with -universal, ad-hoc
// signing, the hardened runtime with -hardened), `run` builds it and starts it with the log on the terminal, `shot` runs the app with
// GLUTTER_SCREENSHOT so that it saves its own first settled frame (proof method G, SPEC §29.13: no screen recording permission), and
// `perf` measures the size, the start, the idle CPU, the memory and the frames as SPEC §15 says. Signing with a Developer ID and
// notarization are the owner's (docs/GUIDE.md).
//
// Flutter: no counterpart (flutter build macos)
package main
