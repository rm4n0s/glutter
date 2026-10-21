// Command inputtest is the program of the browser tests of the web input (step 21.2, SPEC §17,
// §23.4). It is a small Material app: a button that counts its taps, a Shortcuts widget that
// counts the presses of the K key, and a long ListView. In the browser it dispatches synthetic DOM
// events at its own canvas, as a user would make them, and posts what it saw to the test server
// as a result of kind "inputs": the count, the scroll offset, the cursor of the canvas, and whether
// the default action of each event was prevented.
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with
// the web build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
