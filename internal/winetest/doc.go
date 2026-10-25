// Package winetest is the harness of the tests that run a Windows program under Wine (SPEC §25.13):
// it makes the Wine prefix in .wine-test/prefix at the module root (with the workaround for the
// first boot, under a lock), starts a headless X server (scripts/headless-x11.sh) and the program
// under `timeout -k`, in a process group of its own and with the environment of §25.13, stops
// everything with `wineserver -k`, and finds what is left (Leftovers reads /proc).
//
// The harness runs on Linux: the tests of the packages that need Windows run inside Wine with
// scripts/wine-exec.sh (go test -exec), and the tests of this package, of cmd/glutter-windows and of
// the examples start Wine themselves. On Windows the package has only UnderWine, which tells a
// test whether it runs under Wine.
//
// Flutter: no counterpart (flutter drive)
package winetest
