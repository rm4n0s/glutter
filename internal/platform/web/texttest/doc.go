// Command texttest is the program of the browser tests of the web text input and clipboard (step
// 21.3, SPEC §17, §23.5). It is a small Material app with three text fields (a single line, a
// password and a multiline field). In the browser it dispatches synthetic DOM events at the hidden text
// field that the platform makes, as a user and an input method would, and at the host, with a
// navigator.clipboard that it replaces by a stub whose promises it settles itself, and posts what it saw to
// the test server as a result of kind "text": what the hidden field is, the text the framework holds after
// typing, composing and pasting, and whether keys sent while a clipboard read is pending were handled.
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with the web
// build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
