// Package webtest is the harness that runs a Glutter web app in a headless browser (SPEC §17
// "Browser tests", §23.8): an httptest server that serves a build made by internal/webtool and
// collects the PNG and the results the page posts, launchers for Chromium and Firefox with
// temporary profiles that run under timeout in their own process group and are killed when
// the test ends, and Shot, which glutter-web's shot command and the browser tests use. No
// WebDriver or DevTools library is used: the page reports to the server.
//
// Flutter: no counterpart (flutter test --platform chrome)
package webtest
