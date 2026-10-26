// Package wintool builds, runs and takes screenshots of Glutter apps for Windows, for
// glutter-windows (SPEC §25.1): it generates the application manifest, writes the resources of the
// .exe (the manifest, the icon with its sizes, the version information) as a COFF object with a
// .rsrc section in pure Go, runs the Go toolchain with that object added through `go build
// -overlay`, and starts the .exe under Wine through internal/winetest. It runs on the developer's
// machine, never inside an app, and needs no rc.exe, windres, Windows SDK or C compiler.
//
// Flutter: no counterpart (flutter build windows)
package wintool
