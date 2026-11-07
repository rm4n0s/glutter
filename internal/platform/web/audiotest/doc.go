// Command audiotest is the program of the browser tests of the audio of the web (step 31.6, SPEC §28.3, §28.8). It is a small
// Material app whose driver opens devices of the platform's audio service, and posts what happened to the test server as a result of
// kind "audio". GLUTTER_AUDIO_MODE says what it does:
//
//   - "play": plays a tone of 440 Hz, reads it back through an AnalyserNode that the driver puts on the worklet node, pauses and
//     resumes the device, and closes it;
//   - "gesture": opens a playback in a browser that wants a gesture first, and shows that the device waits (DeviceStateSuspended) and
//     that a gesture calls resume on the context;
//   - "capture": opens a recording and counts the frames (the browser of the test has a fake microphone, or none at all).
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with the web build tool
// (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
