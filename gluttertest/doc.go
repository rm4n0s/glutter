// Package gluttertest is the widget-testing package: WidgetTester, finders
// and golden comparison, driven by a fake clock and a fake platform. A test makes
// a Tester with New, pumps a widget into it, finds widgets, sends them pointer
// and key events, and checks what it finds. No display, no GPU and no real time
// are involved (SPEC §17).
//
//	func TestCounter(t *testing.T) {
//		tester := gluttertest.New(t)
//		tester.PumpWidget(NewCounter())
//		tester.Tap(gluttertest.ByType[*Button]())
//		tester.Pump()
//		if err := gluttertest.FindsOneWidget(tester, gluttertest.ByText("1")); err != nil {
//			t.Error(err)
//		}
//	}
//
// The text of a test is laid out with the fonts bundled with Glutter (Roboto and
// Material Icons) and no system fonts, so that it is the same on every machine.
// Opts.Fonts adds more, for example a monospace font for the golden of a code view.
//
// An error that the framework reports (a panic in Build, layout or a callback,
// SPEC §14.2) does not stop the test where it happens; it is kept, and the test
// fails at its end unless it took the error with TakeError.
//
// At the end of a test the tree is unmounted, as flutter_test does after the body
// of a test: New registers a cleanup that pumps an empty widget, so every
// State.Dispose runs and every State.Lifetime is cancelled. It does nothing when
// the test failed already or when the framework reported an error that the test
// did not take. Cleanups run last in, first out, so a cleanup that the test
// registers after New still sees the tree; an object that the tree uses and that
// the test disposes (a controller, a delegate) is registered with t.Cleanup before
// New, to be disposed after the unmounting.
//
// Flutter: package:flutter_test/flutter_test.dart.
package gluttertest
