// Command glutter-web builds, serves and screenshots Glutter apps for the web (SPEC §23.1,
// §23.8):
//
//	go run ./cmd/glutter-web build [-o dir] [-debug] <package>
//	go run ./cmd/glutter-web serve [-addr :8080] [-debug] <package>
//	go run ./cmd/glutter-web shot -browser chromium|firefox [-renderer r] [-debug=false] [-events script] [-after d] -o file.png <package>
//	go run ./cmd/glutter-web perf -browser chromium|firefox [-renderer r] [-debug] [-query k=v] <package>
//
// build, serve and perf make release builds unless -debug is given; shot makes a debug build, since
// the hooks of its events and queries (GLUTTER_DEV_*) are read only by debug builds.
//
// perf prints "key value" lines (sizes of times in milliseconds) that scripts/perf-web.sh reads.
//
// Flutter: no counterpart (flutter build web, flutter run -d web-server).
package main
