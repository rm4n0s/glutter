package gestures

// DebugOptions are the diagnostics flags of the gestures library. A binding
// and a GestureArenaManager hold one; there are no global flags.
//
// Deviation: Flutter's flags are global variables (SPEC §12.10).
//
// Flutter: gestures/debug.dart debugPrintHitTestResults and the flags beside it
type DebugOptions struct {
	// PrintHitTestResults prints the result of every hit test.
	//
	// Flutter: gestures/debug.dart debugPrintHitTestResults
	PrintHitTestResults bool
	// PrintMouseHoverEvents prints every hover event.
	//
	// Flutter: gestures/debug.dart debugPrintMouseHoverEvents
	PrintMouseHoverEvents bool
	// PrintGestureArenaDiagnostics prints what happens in the gesture arenas.
	//
	// Flutter: gestures/debug.dart debugPrintGestureArenaDiagnostics
	PrintGestureArenaDiagnostics bool
	// PrintRecognizerCallbacksTrace prints a stack trace for every
	// callback of a recognizer.
	//
	// Flutter: gestures/debug.dart debugPrintRecognizerCallbacksTrace
	PrintRecognizerCallbacksTrace bool
	// PrintResamplingMargin prints how long before the sample time the last
	// event came.
	//
	// Flutter: gestures/debug.dart debugPrintResamplingMargin
	PrintResamplingMargin bool
	// Print receives the lines; the default drops them.
	//
	// Flutter: foundation/print.dart debugPrint
	Print func(line string)
}

func (o *DebugOptions) print(line string) {
	if o != nil && o.Print != nil {
		o.Print(line)
	}
}
