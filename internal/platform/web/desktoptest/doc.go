// Command desktoptest is the program of the browser tests of the web desktop services (step 21.6, SPEC
// §17, §23.7). It is a small Material app. In the browser it replaces window.open and the Notification
// constructor of the page by stubs that record what they are asked and that it settles by hand (the
// browser's permission prompt cannot be answered in a headless test), calls desktop.OpenURI and
// desktop.ShowNotification from a goroutine as an app does, and posts what it saw to the test server as a
// result of kind "desktop": the arguments window.open got, the tags of the errors, whether a refused
// permission was asked for again, the options the Notification was made with, and the callbacks that
// came back on the UI thread. The code under test is the real one; only the browser's objects are stubs.
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with the web
// build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
