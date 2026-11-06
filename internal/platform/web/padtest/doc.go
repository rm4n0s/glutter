// Command padtest is the program of the browser test of the gamepads of the web (step 31.2, SPEC §28.2, §28.8). It is a small
// Material app with two buttons that count their activations, and in the browser it replaces navigator.getGamepads with a script of
// its own: it plugs in a pad that the browser maps to the standard layout, presses the D-pad and the A button, rumbles, and unplugs
// it, and posts what happened to the test server as a result of kind "pads": the pads the framework saw, the counts of the
// buttons, the effects that the pad's vibration actuator was asked to play, and how many times the page read the pads.
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with the web build tool
// (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
