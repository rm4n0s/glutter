package rendering

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ProfilePaintCallback is called with each render object that paints, when
// DebugOptions.OnProfilePaint is set.
//
// Flutter: rendering/debug.dart ProfilePaintCallback
type ProfilePaintCallback func(renderObject RenderObject)

// DebugPaintCallback is called after a RenderView has painted, to paint on
// top of it.
//
// Flutter: rendering/view.dart DebugPaintCallback
type DebugPaintCallback func(context *PaintingContext, offset ui.Offset, renderView *RenderView)

// DefaultRepaintColor is the value of DebugOptions.CurrentRepaintColor before
// the rainbow changes it: a yellow with 40% opacity.
//
// Flutter: rendering/debug.dart _kDebugDefaultRepaintColor
var DefaultRepaintColor = painting.NewHSVColorFromAHSV(0.4, 60, 1, 1)

// DebugOptions are the debugging flags of the rendering library. The renderer
// binding holds one and shares it with its pipeline owners; layers and render
// objects read it through their owner. A flag has an effect only in debug
// builds.
//
// Deviation: Flutter's flags are global variables; these are fields of a value
// that the binding holds (SPEC §12.10).
//
// Flutter: rendering/debug.dart
type DebugOptions struct {
	// ReportError is called with the errors that the render objects report and
	// that do not stop the frame, such as an overflow; nil prints them with
	// DebugPrint. The binding sets it from RendererBindingOpts.ReportError.
	//
	// Flutter: foundation/assertions.dart FlutterError.reportError
	ReportError func(err error)
	// UIThread says which goroutine is the UI thread. Debug builds check it in
	// MarkNeedsLayout and panic with the tag WrongThread; nil
	// checks nothing (SPEC §13).
	//
	// Flutter: no counterpart
	UIThread *foundation.UIThread
	// PaintSizeEnabled paints a box around every render box
	// (debugPaintSizeEnabled).
	PaintSizeEnabled bool
	// PaintBaselinesEnabled paints the alphabetic and ideographic baselines
	// (debugPaintBaselinesEnabled).
	PaintBaselinesEnabled bool
	// PaintTextLayoutBoxes paints a box around every text box
	// (debugPaintTextLayoutBoxes).
	PaintTextLayoutBoxes bool
	// PaintLayerBordersEnabled paints a border around every layer
	// (debugPaintLayerBordersEnabled).
	PaintLayerBordersEnabled bool
	// PaintPointersEnabled paints a color over a box that has pointers on it
	// (debugPaintPointersEnabled).
	PaintPointersEnabled bool
	// RepaintRainbowEnabled paints a changing color border around each
	// repainted layer (debugRepaintRainbowEnabled).
	RepaintRainbowEnabled bool
	// RepaintTextRainbowEnabled paints a changing color over repainted text
	// (debugRepaintTextRainbowEnabled).
	RepaintTextRainbowEnabled bool
	// CurrentRepaintColor is the color of the next repaint rainbow
	// (debugCurrentRepaintColor); the zero value means DefaultRepaintColor.
	CurrentRepaintColor painting.HSVColor
	// PrintMarkNeedsLayoutStacks prints the stack when a render object is marked
	// as needing layout (debugPrintMarkNeedsLayoutStacks).
	PrintMarkNeedsLayoutStacks bool
	// PrintMarkNeedsPaintStacks prints the stack when a render object is marked
	// as needing paint (debugPrintMarkNeedsPaintStacks).
	PrintMarkNeedsPaintStacks bool
	// PrintLayouts prints a line for every render object that lays out
	// (debugPrintLayouts).
	PrintLayouts bool
	// CheckIntrinsicSizes checks that the intrinsic sizes and the dry layout of
	// a box agree with its layout (debugCheckIntrinsicSizes).
	CheckIntrinsicSizes bool
	// ProfileLayoutsEnabled is kept for compatibility: layout profiling is not
	// ported (debugProfileLayoutsEnabled).
	ProfileLayoutsEnabled bool
	// ProfilePaintsEnabled is kept for compatibility: paint profiling is not
	// ported (debugProfilePaintsEnabled).
	ProfilePaintsEnabled bool
	// EnhanceLayoutTimelineArguments is kept for compatibility
	// (debugEnhanceLayoutTimelineArguments).
	EnhanceLayoutTimelineArguments bool
	// EnhancePaintTimelineArguments is kept for compatibility
	// (debugEnhancePaintTimelineArguments).
	EnhancePaintTimelineArguments bool
	// OnProfilePaint is called with each render object that paints
	// (debugOnProfilePaint).
	OnProfilePaint ProfilePaintCallback
	// DisableClipLayers makes the clip layers not clip (debugDisableClipLayers).
	DisableClipLayers bool
	// DisablePhysicalShapeLayers makes the physical shape layers not draw
	// elevation (debugDisablePhysicalShapeLayers).
	DisablePhysicalShapeLayers bool
	// DisableOpacityLayers makes the opacity layers not fade
	// (debugDisableOpacityLayers).
	DisableOpacityLayers bool
	// Print writes a line of debug output; the default is fmt.Println
	// (debugPrint).
	Print func(string)

	paintCallbacks []DebugPaintCallback
}

// NewDebugOptions creates options with every flag off.
//
// Flutter: rendering/debug.dart
func NewDebugOptions() *DebugOptions {
	return &DebugOptions{CurrentRepaintColor: DefaultRepaintColor}
}

func (o *DebugOptions) repaintColor() painting.HSVColor {
	if o.CurrentRepaintColor == (painting.HSVColor{}) {
		return DefaultRepaintColor
	}
	return o.CurrentRepaintColor
}

// DebugPrint writes a line through Print.
//
// Flutter: foundation/print.dart debugPrint
func (o *DebugOptions) DebugPrint(s string) {
	if o.Print != nil {
		o.Print(s)
		return
	}
	fmt.Println(s)
}

// AddPaintCallback registers a callback that runs after every RenderView
// paints (debug builds only).
//
// Flutter: rendering/view.dart RenderView.debugAddPaintCallback
func (o *DebugOptions) AddPaintCallback(cb DebugPaintCallback) {
	if foundation.DebugMode {
		o.paintCallbacks = append(o.paintCallbacks, cb)
	}
}

// RemovePaintCallback removes the callback registered last with the same
// function value; because functions are not comparable it takes the index that
// AddPaintCallback's order gives.
//
// Flutter: rendering/view.dart RenderView.debugRemovePaintCallback
func (o *DebugOptions) RemovePaintCallback(index int) {
	if foundation.DebugMode && index >= 0 && index < len(o.paintCallbacks) {
		o.paintCallbacks = append(o.paintCallbacks[:index], o.paintCallbacks[index+1:]...)
	}
}

// PaintCallbackCount is the number of registered paint callbacks.
func (o *DebugOptions) PaintCallbackCount() int { return len(o.paintCallbacks) }

// DebugAssertAllRenderVarsUnset panics with the tag RenderDebugVarsSet and the
// reason if any flag is set, in debug builds. CheckIntrinsicSizesOverride is the
// value that CheckIntrinsicSizes is expected to have.
//
// Flutter: rendering/debug.dart debugAssertAllRenderVarsUnset
func (o *DebugOptions) DebugAssertAllRenderVarsUnset(reason string, checkIntrinsicSizesOverride bool) bool {
	if foundation.DebugMode {
		if o.PaintSizeEnabled || o.PaintBaselinesEnabled || o.PaintLayerBordersEnabled || o.PaintTextLayoutBoxes ||
			o.PaintPointersEnabled || o.RepaintRainbowEnabled || o.RepaintTextRainbowEnabled ||
			o.repaintColor() != DefaultRepaintColor || o.PrintMarkNeedsLayoutStacks || o.PrintMarkNeedsPaintStacks ||
			o.PrintLayouts || o.CheckIntrinsicSizes != checkIntrinsicSizesOverride || o.ProfileLayoutsEnabled ||
			o.ProfilePaintsEnabled || o.OnProfilePaint != nil || o.DisableClipLayers ||
			o.DisablePhysicalShapeLayers || o.DisableOpacityLayers {
			panic(errors.New("RenderDebugVarsSet", reason))
		}
	}
	return true
}

func debugDrawDoubleRect(canvas *ui.Canvas, outer, inner ui.Rect, color ui.Color) {
	path := ui.NewPath()
	path.SetFillType(ui.PathFillTypeEvenOdd)
	path.AddRect(outer)
	path.AddRect(inner)
	paint := ui.NewPaint()
	paint.SetColor(color)
	canvas.DrawPath(path, paint)
}

// DebugPaintPadding paints the padding between the outer and the inner
// rectangle in a debug color; without an inner rectangle it paints the outer one
// in gray. Debug builds only.
//
// Flutter: rendering/debug.dart debugPaintPadding
func DebugPaintPadding(canvas *ui.Canvas, outerRect ui.Rect, innerRect foundation.Opt[ui.Rect], outlineWidth float64) {
	if !foundation.DebugMode {
		return
	}
	if inner, ok := innerRect.Get(); ok && !inner.IsEmpty() {
		debugDrawDoubleRect(canvas, outerRect, inner, ui.NewColorFromARGB(0x90, 0x00, 0x90, 0xFF))
		debugDrawDoubleRect(canvas, inner.Inflate(outlineWidth).Intersect(outerRect), inner, ui.NewColorFromARGB(0xFF, 0x00, 0x90, 0xFF))
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColorFromARGB(0x90, 0x90, 0x90, 0x90))
	canvas.DrawRect(outerRect, paint)
}

// DebugCheckHasBoundedAxis panics in debug builds, with the tag
// ViewportUnboundedMainAxis or ViewportUnboundedCrossAxis, if a viewport that
// scrolls along the axis is given unbounded constraints.
//
// Flutter: rendering/debug.dart debugCheckHasBoundedAxis
func DebugCheckHasBoundedAxis(axis painting.Axis, constraints BoxConstraints) bool {
	if !foundation.DebugMode || (constraints.HasBoundedHeight() && constraints.HasBoundedWidth()) {
		return true
	}
	switch axis {
	case painting.AxisVertical:
		if !constraints.HasBoundedHeight() {
			panic(errors.New("ViewportUnboundedMainAxis", "Vertical viewport was given unbounded height. "+
				"Viewports expand in the scrolling direction to fill their container. In this case, a vertical viewport "+
				"was given an unlimited amount of vertical space in which to expand. This situation typically happens "+
				"when a scrollable widget is nested inside another scrollable widget. If this widget is always nested "+
				"in a scrollable widget there is no need to use a viewport because there will always be enough "+
				"vertical space for the children. In this case, consider using a Column or Wrap instead. Otherwise, "+
				"consider using a CustomScrollView to concatenate arbitrary slivers into a single scrollable.",
				"axis", axis.String()))
		}
		if !constraints.HasBoundedWidth() {
			panic(errors.New("ViewportUnboundedCrossAxis", "Vertical viewport was given unbounded width. "+
				"Viewports expand in the cross axis to fill their container and constrain their children to match "+
				"their extent in the cross axis. In this case, a vertical viewport was given an unlimited amount of "+
				"horizontal space in which to expand.", "axis", axis.String()))
		}
	case painting.AxisHorizontal:
		if !constraints.HasBoundedWidth() {
			panic(errors.New("ViewportUnboundedMainAxis", "Horizontal viewport was given unbounded width. "+
				"Viewports expand in the scrolling direction to fill their container. In this case, a horizontal "+
				"viewport was given an unlimited amount of horizontal space in which to expand. This situation "+
				"typically happens when a scrollable widget is nested inside another scrollable widget. If this "+
				"widget is always nested in a scrollable widget there is no need to use a viewport because there "+
				"will always be enough horizontal space for the children. In this case, consider using a Row or "+
				"Wrap instead. Otherwise, consider using a CustomScrollView to concatenate arbitrary slivers into "+
				"a single scrollable.", "axis", axis.String()))
		}
		if !constraints.HasBoundedHeight() {
			panic(errors.New("ViewportUnboundedCrossAxis", "Horizontal viewport was given unbounded height. "+
				"Viewports expand in the cross axis to fill their container and constrain their children to match "+
				"their extent in the cross axis. In this case, a horizontal viewport was given an unlimited amount of "+
				"vertical space in which to expand.", "axis", axis.String()))
		}
	}
	return true
}

var _ = math.Inf
