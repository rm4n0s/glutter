// Package webtool packages a Glutter app for the web (SPEC §23.1): it builds the program for
// js/wasm, writes the directory that a static web server can serve (index.html, wasm_exec.js,
// app.wasm and app.wasm.gz), and serves such a directory with the right types and with the
// gzip file for a browser that accepts it. cmd/glutter-web is its command line, and
// internal/webtest uses it to run apps in browsers.
//
// Flutter: no counterpart (flutter build web, flutter run -d web-server).
package webtool
