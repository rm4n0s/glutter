package foundation

import (
	"fmt"
	"maps"
	"runtime/debug"
	"slices"

	"github.com/rm4n0s/errors"
)

// ErrorReporter receives the errors that the framework recovers from instead of
// letting them end the program: a panic in a Build method, in layout or paint,
// in a callback or a listener. The binding passes one to every part of the
// framework. The default of RunApp logs the error with its ToJson through the
// app's logger (SPEC §14.2).
//
// Flutter: foundation/assertions.dart FlutterError.onError
type ErrorReporter func(err error)

// The tags of the errors made from recovered panics.
const (
	// BuildPanicked is a panic in a Build method (or a lifecycle hook) of a
	// widget.
	//
	// Flutter: widgets/framework.dart ComponentElement.performRebuild
	BuildPanicked = "BuildPanicked"
	// LayoutPanicked is a panic while a render object lays itself out.
	//
	// Flutter: rendering/object.dart RenderObject._reportException
	LayoutPanicked = "LayoutPanicked"
	// PaintPanicked is a panic while a render object paints.
	//
	// Flutter: rendering/object.dart RenderObject._reportException
	PaintPanicked = "PaintPanicked"
	// CallbackPanicked is a panic in any other callback that the framework
	// calls: a gesture callback, a listener, a frame callback, a task.
	//
	// Flutter: foundation/assertions.dart FlutterError.reportError
	CallbackPanicked = "CallbackPanicked"
)

// PanicError makes the error to report for a recovered panic. The tag is one of
// BuildPanicked, LayoutPanicked, PaintPanicked and CallbackPanicked; context
// says what the framework was doing ("while building Foo"); the metadata has
// the panic value (panic), the stack of the goroutine (stack), the context and
// the pairs given in kv, such as the element path.
//
// Deviation: a panic that is already a tagged *errors.Error, such as a failed
// debug check (SetStateAfterDispose), is reported as it is, with the context,
// the stack and kv added to its metadata, so that its own tag stays the route
// of the error (SPEC §14.2, §12.10).
//
// Flutter: foundation/assertions.dart FlutterErrorDetails
func PanicError(tag, context string, recovered any, kv ...any) error {
	meta := append([]any{"context", context, "stack", string(debug.Stack())}, kv...)
	if appErr, ok := recovered.(*errors.Error); ok {
		// SetMetadata replaces the metadata, so the error's own pairs go first.
		own := make([]any, 0, 2*len(appErr.Metadata)+len(meta))
		for _, k := range slices.Sorted(maps.Keys(appErr.Metadata)) {
			own = append(own, k, appErr.Metadata[k])
		}
		return appErr.SetMetadata(append(own, meta...)...)
	}
	if err, ok := recovered.(error); ok {
		return errors.NewErr(tag, err).SetMetadata(meta...)
	}
	return errors.New(tag, fmt.Sprint(recovered), append([]any{"panic", recovered}, meta...)...)
}

// Guard runs fn. If fn panics and report is not nil, the panic is recovered, made
// into an error with PanicError and given to report, and Guard returns true. If
// report is nil nothing is recovered: the panic goes on, so that a program that
// has no reporter does not lose its errors.
//
// kv is a function so that the pairs, such as the element path, are made only
// when there is something to report; it may be nil.
//
// Flutter: foundation/assertions.dart FlutterError.reportError
func Guard(report ErrorReporter, tag, context string, kv func() []any, fn func()) (panicked bool) {
	if report == nil {
		fn()
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			var extra []any
			if kv != nil {
				extra = kv()
			}
			report(PanicError(tag, context, r, extra...))
			panicked = true
		}
	}()
	fn()
	return false
}

// IsRecoveredPanic reports whether err was made by PanicError, that is, whether
// it is the report of a panic that the framework recovered.
//
// Flutter: foundation/assertions.dart FlutterErrorDetails
func IsRecoveredPanic(err error) bool {
	appErr, ok := errors.FromError(err)
	if !ok {
		return false
	}
	_, has := appErr.Metadata["stack"]
	return has
}
