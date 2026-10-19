package widgets

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// DebugOptions are the debugging flags of the widgets library, and the ones of
// the rendering library they extend. The app passes one through
// RunAppOpts.Debug and the framework reads it through the binding. A flag has an
// effect only in debug builds.
//
// Deviation: Flutter's flags are global variables; these are fields of a value
// that the binding holds (SPEC §12.10, §14.3).
//
// Flutter: widgets/debug.dart
type DebugOptions struct {
	// Rendering are the flags of the rendering library; nil makes a fresh set
	// when the binding is created.
	Rendering *rendering.DebugOptions
	// PrintRebuildDirtyWidgets prints a line each time an element is built or
	// rebuilt (debugPrintRebuildDirtyWidgets).
	PrintRebuildDirtyWidgets bool
	// PrintBuildScope prints a line when BuildScope starts and finishes
	// (debugPrintBuildScope).
	PrintBuildScope bool
	// PrintScheduleBuildForStacks prints the stack each time an element is
	// added to the dirty list (debugPrintScheduleBuildForStacks).
	PrintScheduleBuildForStacks bool
	// PrintGlobalKeyedWidgetLifecycle prints a line when a widget with a global
	// key is deactivated or unmounted (debugPrintGlobalKeyedWidgetLifecycle).
	PrintGlobalKeyedWidgetLifecycle bool
	// ProfileBuildsEnabled records the builds of every widget in the trace
	// (debugProfileBuildsEnabled).
	ProfileBuildsEnabled bool
	// ProfileBuildsEnabledUserWidgets records the builds of the widgets the app
	// created (debugProfileBuildsEnabledUserWidgets).
	ProfileBuildsEnabledUserWidgets bool
	// EnhanceBuildTimelineArguments adds the widget properties to the trace of a
	// build (debugEnhanceBuildTimelineArguments).
	EnhanceBuildTimelineArguments bool
	// HighlightDeprecatedWidgets draws a box around the deprecated widgets
	// (debugHighlightDeprecatedWidgets).
	HighlightDeprecatedWidgets bool
	// PaintFocusBoxes paints a box around the focused widget
	// (debugPaintFocusBoxes).
	PaintFocusBoxes bool
	// FocusChanges prints a line for each change of the focus tree
	// (debugFocusChanges).
	//
	// Flutter: widgets/focus_manager.dart debugFocusChanges
	FocusChanges bool
	// BrightnessOverride, when it is not BrightnessUnspecified, replaces the
	// platform brightness of the MediaQueryData made from the view
	// (debugBrightnessOverride).
	//
	// Flutter: foundation/debug.dart debugBrightnessOverride
	BrightnessOverride ui.Brightness
	// PrintKeyboardEvents prints what the hardware keyboard does with each key
	// event (debugPrintKeyboardEvents).
	//
	// Flutter: services/debug.dart debugPrintKeyboardEvents
	PrintKeyboardEvents bool
	// Print writes a line of debug output; nil writes to standard error
	// (debugPrint).
	//
	// Flutter: foundation/print.dart debugPrint
	Print func(line string)
}

// NewDebugOptions creates the flags, all off, with a fresh set of rendering
// flags.
//
// Flutter: widgets/debug.dart
func NewDebugOptions() *DebugOptions {
	return &DebugOptions{Rendering: rendering.NewDebugOptions()}
}

// DebugPrint writes the line through Print.
//
// Flutter: foundation/print.dart debugPrint
func (o *DebugOptions) DebugPrint(line string) {
	switch {
	case o == nil:
		return
	case o.Print != nil:
		o.Print(line)
	default:
		fmt.Fprintln(os.Stderr, line)
	}
}

// debugStack is the stack of the caller, for PrintScheduleBuildForStacks.
func debugStack() string { return string(debug.Stack()) }
