// Command glutter-windows builds, runs and takes screenshots of Glutter apps for Windows (SPEC
// §25.1):
//
//	glutter-windows build [-o app.exe] [-arch amd64,arm64] [-release] [-console] [-name Name] [-app-id id] [-icon icon.png] [-version 1.2.3.4] <package>
//	glutter-windows run [-headless] [-env GLUTTER_X=v]... [-timeout d] [build flags] <package>
//	glutter-windows shot [-env GLUTTER_X=v]... [-screen] [-settle d] [-timeout d] -o file.png [build flags] <package>
//
// build makes an .exe for each architecture, with the application manifest, the icon and the version
// information that internal/wintool writes in pure Go, and with the GUI subsystem unless -console is
// given. run builds the .exe for this machine and starts it, under Wine on Linux (on the user's
// display, or on a headless X server with -headless) and by itself on Windows, and shows its output.
// shot is proof method F: it builds the app, runs it with GLUTTER_SCREENSHOT under Wine on a
// headless X server, and saves the frame the app saved (with -screen, the whole screen of the X
// server).
//
// Go 1.27 or later is all the tool needs; run and shot on Linux also need Wine and Xvfb (and
// ImageMagick for -screen) and must be started in a checkout of Glutter, whose scripts they use.
//
// Flutter: no counterpart (flutter build windows, flutter run -d windows)
package main
