// Command glutter-windows builds, runs and takes screenshots of Glutter apps for Windows (SPEC
// §25.1):
//
//	glutter-windows build [-o app.exe] [-arch amd64,arm64] [-release] [-console] [-name Name] [-app-id id] [-icon icon.png] [-version 1.2.3.4] <package>
//	glutter-windows run [-headless] [-env GLUTTER_X=v]... [-timeout d] [build flags] <package>
//	glutter-windows shot [-env GLUTTER_X=v]... [-screen] [-settle d] [-timeout d] -o file.png [build flags] <package>
//	glutter-windows perf [-release=true] [-arch amd64,arm64] [-renderer r] [-env GLUTTER_X=v]... [-starts N] [-settle d] [-seconds N] [-frames=true] [-format text|markdown] [-o report] <package>
//
// build makes an .exe for each architecture, with the application manifest, the icon and the version
// information that internal/wintool writes in pure Go, and with the GUI subsystem unless -console is
// given. run builds the .exe for this machine and starts it, under Wine on Linux (on the user's
// display, or on a headless X server with -headless) and by itself on Windows, and shows its output.
// shot is proof method F: it builds the app, runs it with GLUTTER_SCREENSHOT under Wine on a
// headless X server, and saves the frame the app saved (with -screen, the whole screen of the X
// server). perf (Linux, under Wine) builds the package and prints the size of the .exe files, the cold
// start with an empty and with a warm shader cache, the idle CPU and the memory of the Wine processes
// (read from /proc), and the engine's frame statistics of a scenario (SPEC §15, §25.13); its numbers
// are Wine's, not a GPU's on Windows.
//
// Go 1.27 or later is all the tool needs; run and shot on Linux also need Wine and Xvfb (and
// ImageMagick for -screen) and must be started in a checkout of Glutter, whose scripts they use.
//
// Flutter: no counterpart (flutter build windows, flutter run -d windows)
package main
