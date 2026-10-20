package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// PerformanceOverlayOpts are the optional parameters of NewPerformanceOverlay.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.new
type PerformanceOverlayOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OptionsMask has the bits of the rendering.PerformanceOverlayOption values to show;
	// the default is 0, which shows nothing.
	OptionsMask int
}

// PerformanceOverlayAllEnabledOpts are the optional parameters of
// NewPerformanceOverlayAllEnabled.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.allEnabled
type PerformanceOverlayAllEnabledOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// PerformanceOverlay displays the performance of the application: a graph and the
// statistics of the time to build each frame (UI) and of the time to draw it (Raster),
// against the frame budget of the display (SPEC §14.3). The engine measures the frames
// only while an overlay is in the tree.
//
// Deviation: Flutter's overlay is drawn by the engine; this one draws itself from the
// timings that the engine reports, and does not keep the app drawing frames (see
// rendering.RenderPerformanceOverlay, SPEC §12.10).
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay
type PerformanceOverlay struct {
	LeafRenderObjectWidget
	optionsMask int
}

// NewPerformanceOverlay creates the overlay with the options of the mask.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.new
func NewPerformanceOverlay(opts ...PerformanceOverlayOpts) *PerformanceOverlay {
	o := firstOpts(opts)
	return &PerformanceOverlay{LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), optionsMask: o.OptionsMask}
}

// NewPerformanceOverlayAllEnabled creates the overlay with every option: both graphs
// and both texts.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.allEnabled
func NewPerformanceOverlayAllEnabled(opts ...PerformanceOverlayAllEnabledOpts) *PerformanceOverlay {
	return NewPerformanceOverlay(PerformanceOverlayOpts{Key: firstOpts(opts).Key, OptionsMask: rendering.PerformanceOverlayAllMask()})
}

// OptionsMask is the bits of the options that are shown.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.optionsMask
func (w *PerformanceOverlay) OptionsMask() int { return w.optionsMask }

// frameBudget is how long a frame may take on the display of the view of the context.
func (w *PerformanceOverlay) frameBudget(bc BuildContext) time.Duration {
	if v := MaybeViewOf(bc); v != nil {
		if rate := v.Display().RefreshRate; rate > 0 {
			return time.Duration(float64(time.Second) / rate)
		}
	}
	return time.Second / 60
}

// CreateRenderObject creates the render object, listening to the frame timings of the
// binding.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.createRenderObject
func (w *PerformanceOverlay) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	opts := rendering.RenderPerformanceOverlayOpts{OptionsMask: w.optionsMask, FrameBudget: w.frameBudget(bc)}
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		opts.Scheduler = b.Scheduler()
		opts.TextEngine = b.PlatformDispatcher().TextEngine()
	}
	return rendering.NewRenderPerformanceOverlay(opts)
}

// UpdateRenderObject sets the options.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.updateRenderObject
func (w *PerformanceOverlay) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPerformanceOverlay)
	r.SetOptionsMask(w.optionsMask)
	r.SetFrameBudget(w.frameBudget(bc))
}
