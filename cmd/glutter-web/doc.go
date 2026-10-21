// Command glutter-web builds, serves and screenshots Glutter apps for the web (SPEC §23.1,
// §23.8):
//
//	go run ./cmd/glutter-web build [-o dir] [-release] <package>
//	go run ./cmd/glutter-web serve [-addr :8080] [-release] <package>
//	go run ./cmd/glutter-web shot -browser chromium|firefox [-renderer r] [-events script] [-after d] -o file.png <package>
//	go run ./cmd/glutter-web perf -browser chromium|firefox [-renderer r] [-release] [-query k=v] <package>
//
// perf prints "key value" lines (sizes of times in milliseconds) that scripts/perf-web.sh reads.
//
// Flutter: no counterpart (flutter build web, flutter run -d web-server).
package main
