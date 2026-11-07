// Command sensortest is the program of the browser tests of the motion sensors and the haptics of the web (step 31.7, SPEC §28.4,
// §28.8). It is a small Material app whose driver opens sensors of package sensors, sends the page DeviceMotionEvents of its own
// making, and posts what the app saw as a result of kind "sensors". GLUTTER_SENSOR_MODE says what it does:
//
//   - "motion": opens the accelerometer and the gyroscope, sends events (one, three at once, one with no rotation), closes the
//     sensors one after the other, and then asks for each kind of haptic feedback, with navigator.vibrate replaced by a recorder;
//   - "permission": gives DeviceMotionEvent a requestPermission, as Safari on iOS has, and shows that Open waits for a gesture,
//     asks once, and takes the answer (GLUTTER_SENSOR_ANSWER is "granted" or "denied");
//   - "none": takes DeviceMotionEvent away, and shows that there are no sensors.
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with the web build tool
// (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
