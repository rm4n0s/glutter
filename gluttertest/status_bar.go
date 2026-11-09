package gluttertest

// SimulateStatusBarTap acts as if the user tapped the status bar, the gesture that scrolls a page of iOS to
// its top: the binding tells its observers.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.simulateStatusBarTap
func (t *Tester) SimulateStatusBarTap() {
	t.tb.Helper()
	t.Binding.HandleStatusBarTap()
}
