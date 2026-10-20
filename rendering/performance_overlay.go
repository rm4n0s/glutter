package rendering

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// PerformanceOverlayOption is a part of the performance overlay.
//
// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption
type PerformanceOverlayOption int

// The parts of the overlay. Flutter's constants are the bits of the options mask, in
// the order of the enum's indexes (Index); Mask is the bit.
const (
	// PerformanceOverlayOptionUnspecified is the zero value: not an option.
	//
	// Flutter: no counterpart
	PerformanceOverlayOptionUnspecified PerformanceOverlayOption = iota
	// PerformanceOverlayOptionDisplayRasterizerStatistics writes the statistics of the
	// raster thread (the time to draw a frame) as text.
	//
	// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption.displayRasterizerStatistics
	PerformanceOverlayOptionDisplayRasterizerStatistics
	// PerformanceOverlayOptionVisualizeRasterizerStatistics draws a graph of the time to
	// draw each frame.
	//
	// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption.visualizeRasterizerStatistics
	PerformanceOverlayOptionVisualizeRasterizerStatistics
	// PerformanceOverlayOptionDisplayEngineStatistics writes the statistics of the UI
	// thread (the time to build a frame) as text.
	//
	// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption.displayEngineStatistics
	PerformanceOverlayOptionDisplayEngineStatistics
	// PerformanceOverlayOptionVisualizeEngineStatistics draws a graph of the time to
	// build each frame.
	//
	// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption.visualizeEngineStatistics
	PerformanceOverlayOptionVisualizeEngineStatistics
)

// Index is the position of the option in Flutter's enum, from 0.
//
// Flutter: rendering/performance_overlay.dart PerformanceOverlayOption.index
func (o PerformanceOverlayOption) Index() int { return int(o) - 1 }

// Mask is the bit of the option in an options mask.
//
// Flutter: rendering/performance_overlay.dart (1 << option.index)
func (o PerformanceOverlayOption) Mask() int {
	if o <= PerformanceOverlayOptionUnspecified {
		return 0
	}
	return 1 << o.Index()
}

var (
	rasterizerMask = PerformanceOverlayOptionDisplayRasterizerStatistics.Mask() | PerformanceOverlayOptionVisualizeRasterizerStatistics.Mask()
	engineMask     = PerformanceOverlayOptionDisplayEngineStatistics.Mask() | PerformanceOverlayOptionVisualizeEngineStatistics.Mask()
)

// PerformanceOverlayAllMask has the bits of every option.
//
// Flutter: widgets/performance_overlay.dart PerformanceOverlay.allEnabled
func PerformanceOverlayAllMask() int { return rasterizerMask | engineMask }

const (
	// performanceGraphHeight is the height of a graph, in logical pixels.
	performanceGraphHeight = 80.0
	// performanceBarWidth is the width of the bar of a frame.
	performanceBarWidth = 2.0
	// DefaultFrameTimingCapacity is how many frames a FrameTimingHistory keeps.
	DefaultFrameTimingCapacity = 240
)

// FrameTimingHistory keeps the last frames' timings, for the performance overlay.
//
// Flutter: no counterpart
type FrameTimingHistory struct {
	capacity int
	timings  []ui.FrameTiming
}

// NewFrameTimingHistory creates a history of the last capacity frames; a value below 1 keeps
// DefaultFrameTimingCapacity.
//
// Flutter: no counterpart
func NewFrameTimingHistory(capacity int) *FrameTimingHistory {
	if capacity < 1 {
		capacity = DefaultFrameTimingCapacity
	}
	return &FrameTimingHistory{capacity: capacity}
}

// Add appends the timings, dropping the oldest frames over the capacity.
//
// Flutter: no counterpart
func (h *FrameTimingHistory) Add(timings ...ui.FrameTiming) {
	h.timings = append(h.timings, timings...)
	if over := len(h.timings) - h.capacity; over > 0 {
		h.timings = append(h.timings[:0], h.timings[over:]...)
	}
}

// Len is the number of frames kept.
//
// Flutter: no counterpart
func (h *FrameTimingHistory) Len() int { return len(h.timings) }

// Last returns the n newest frames, oldest first (all of them if there are fewer).
//
// Flutter: no counterpart
func (h *FrameTimingHistory) Last(n int) []ui.FrameTiming {
	if n > len(h.timings) {
		n = len(h.timings)
	}
	return h.timings[len(h.timings)-n:]
}

// RenderPerformanceOverlayOpts are the optional parameters of NewRenderPerformanceOverlay.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.new
type RenderPerformanceOverlayOpts struct {
	// OptionsMask has the bits of the PerformanceOverlayOption values to show.
	OptionsMask int
	// Scheduler is where the frame timings come from: the overlay listens while it is
	// attached. Without one the overlay shows what History is given with Add.
	//
	// Deviation: Flutter's engine draws the overlay from its own measurements; this one
	// draws it from the timings the engine reports (SPEC §12.10).
	Scheduler *scheduler.SchedulerBinding
	// TextEngine lays out the labels; without one the graphs have no text.
	TextEngine ui.ParagraphFactory
	// FrameBudget is how long a frame may take: the bars of longer frames are red. The
	// default is a 60th of a second.
	FrameBudget time.Duration
	// History is the timings to show; the default is a new history of
	// DefaultFrameTimingCapacity frames.
	History *FrameTimingHistory
}

// RenderPerformanceOverlay displays the frame times of the app: a graph and the
// statistics of the time to build each frame and of the time to draw it.
//
// Deviation: Flutter paints a PerformanceOverlayLayer that the engine fills in; this
// paints the graphs with the canvas, from the timings that the engine reports
// (ui.PlatformDispatcher.OnReportTimings), and is a repaint boundary so that a new
// frame's timing repaints only the overlay. The timings are added without asking for
// a frame (MarkNeedsPaintQuietly): the overlay shows them with the next frame that
// something else asks for, and does not keep the app drawing (SPEC §12.10).
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay
type RenderPerformanceOverlay struct {
	RenderBox
	optionsMask int
	history     *FrameTimingHistory
	sched       *scheduler.SchedulerBinding
	handle      scheduler.TimingsHandle
	listening   bool
	engine      ui.ParagraphFactory
	budget      time.Duration
}

// NewRenderPerformanceOverlay creates the overlay.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.new
func NewRenderPerformanceOverlay(opts ...RenderPerformanceOverlayOpts) *RenderPerformanceOverlay {
	o := firstOpts(opts)
	r := &RenderPerformanceOverlay{optionsMask: o.OptionsMask, sched: o.Scheduler, engine: o.TextEngine, budget: o.FrameBudget, history: o.History}
	if r.history == nil {
		r.history = NewFrameTimingHistory(0)
	}
	if r.budget <= 0 {
		r.budget = time.Second / 60
	}
	InitRenderBox(&r.RenderBox, r)
	return r
}

// OptionsMask is the bits of the options that are shown.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.optionsMask
func (r *RenderPerformanceOverlay) OptionsMask() int { return r.optionsMask }

// SetOptionsMask sets the options and repaints if they changed.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.optionsMask
func (r *RenderPerformanceOverlay) SetOptionsMask(value int) {
	if value == r.optionsMask {
		return
	}
	r.optionsMask = value
	r.MarkNeedsLayout()
	r.MarkNeedsPaint()
}

// History is the timings the overlay shows.
//
// Flutter: no counterpart
func (r *RenderPerformanceOverlay) History() *FrameTimingHistory { return r.history }

// SetFrameBudget sets how long a frame may take, and repaints if it changed.
//
// Flutter: no counterpart
func (r *RenderPerformanceOverlay) SetFrameBudget(value time.Duration) {
	if value <= 0 || value == r.budget {
		return
	}
	r.budget = value
	r.MarkNeedsPaint()
}

// SetTextEngine sets the engine that lays out the labels.
//
// Flutter: no counterpart
func (r *RenderPerformanceOverlay) SetTextEngine(engine ui.ParagraphFactory) {
	if engine == r.engine {
		return
	}
	r.engine = engine
	r.MarkNeedsPaint()
}

// Attach starts listening to the frame timings.
//
// Flutter: rendering/object.dart RenderObject.attach
func (r *RenderPerformanceOverlay) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	if r.sched != nil && !r.listening {
		r.handle = r.sched.AddTimingsCallback(r.onTimings)
		r.listening = true
	}
}

// Detach stops listening.
//
// Flutter: rendering/object.dart RenderObject.detach
func (r *RenderPerformanceOverlay) Detach() {
	if r.listening {
		r.sched.RemoveTimingsCallback(r.handle)
		r.listening = false
	}
	r.RenderBox.Detach()
}

func (r *RenderPerformanceOverlay) onTimings(timings []ui.FrameTiming) {
	r.history.Add(timings...)
	r.MarkNeedsPaintQuietly()
}

// SizedByParent is true.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.sizedByParent
func (r *RenderPerformanceOverlay) SizedByParent() bool { return true }

// IsRepaintBoundary is true.
func (r *RenderPerformanceOverlay) IsRepaintBoundary() bool { return true }

// AlwaysNeedsCompositing is true.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.alwaysNeedsCompositing
func (r *RenderPerformanceOverlay) AlwaysNeedsCompositing() bool { return true }

// ComputeMinIntrinsicWidth is 0.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.computeMinIntrinsicWidth
func (r *RenderPerformanceOverlay) ComputeMinIntrinsicWidth(float64) float64 { return 0 }

// ComputeMaxIntrinsicWidth is 0.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.computeMaxIntrinsicWidth
func (r *RenderPerformanceOverlay) ComputeMaxIntrinsicWidth(float64) float64 { return 0 }

func (r *RenderPerformanceOverlay) intrinsicHeight() float64 {
	result := 0.0
	if r.optionsMask&rasterizerMask != 0 {
		result += performanceGraphHeight
	}
	if r.optionsMask&engineMask != 0 {
		result += performanceGraphHeight
	}
	return result
}

// ComputeMinIntrinsicHeight is the height of the graphs.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.computeMinIntrinsicHeight
func (r *RenderPerformanceOverlay) ComputeMinIntrinsicHeight(float64) float64 {
	return r.intrinsicHeight()
}

// ComputeMaxIntrinsicHeight is the height of the graphs.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.computeMaxIntrinsicHeight
func (r *RenderPerformanceOverlay) ComputeMaxIntrinsicHeight(float64) float64 {
	return r.intrinsicHeight()
}

// ComputeDryLayout is as wide as the constraints allow and as high as the graphs.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.computeDryLayout
func (r *RenderPerformanceOverlay) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return constraints.Constrain(ui.Size{Width: math.Inf(1), Height: r.intrinsicHeight()})
}

// Paint draws the graph of the raster times and then the one of the build times.
//
// Flutter: rendering/performance_overlay.dart RenderPerformanceOverlay.paint
func (r *RenderPerformanceOverlay) Paint(context *PaintingContext, offset ui.Offset) {
	canvas := context.Canvas()
	size := r.Size()
	y := offset.Dy
	if r.optionsMask&rasterizerMask != 0 {
		r.paintGraph(canvas, ui.NewRectFromLTWH(offset.Dx, y, size.Width, performanceGraphHeight), "Raster",
			PerformanceOverlayOptionDisplayRasterizerStatistics.Mask()&r.optionsMask != 0,
			PerformanceOverlayOptionVisualizeRasterizerStatistics.Mask()&r.optionsMask != 0,
			ui.FrameTiming.RasterDuration)
		y += performanceGraphHeight
	}
	if r.optionsMask&engineMask != 0 {
		r.paintGraph(canvas, ui.NewRectFromLTWH(offset.Dx, y, size.Width, performanceGraphHeight), "UI",
			PerformanceOverlayOptionDisplayEngineStatistics.Mask()&r.optionsMask != 0,
			PerformanceOverlayOptionVisualizeEngineStatistics.Mask()&r.optionsMask != 0,
			ui.FrameTiming.BuildDuration)
	}
}

var (
	overlayBackground = ui.NewColor(0xCC000000)
	overlayGood       = ui.NewColor(0xFF4CAF50)
	overlayBad        = ui.NewColor(0xFFF44336)
	overlayLine       = ui.NewColor(0x99FFFFFF)
	overlayDivider    = ui.NewColor(0x66FFFFFF)
	overlayText       = ui.NewColor(0xFFFFFFFF)
)

// paintGraph draws one panel: the bars of the last frames (the newest at the right
// edge) against a line at the frame budget, then the label with the statistics.
func (r *RenderPerformanceOverlay) paintGraph(canvas *ui.Canvas, rect ui.Rect, label string, text, graph bool, duration func(ui.FrameTiming) time.Duration) {
	fill := ui.NewPaint()
	fill.SetColor(overlayBackground)
	canvas.DrawRect(rect, fill)
	bars := int(rect.Width() / performanceBarWidth)
	frames := r.history.Last(bars)
	if graph {
		scale := rect.Height() / (2 * float64(r.budget))
		x := rect.Right - performanceBarWidth*float64(len(frames))
		for _, f := range frames {
			d := duration(f)
			h := math.Min(math.Max(float64(d)*scale, 1), rect.Height())
			if d > r.budget {
				fill.SetColor(overlayBad)
			} else {
				fill.SetColor(overlayGood)
			}
			canvas.DrawRect(ui.NewRectFromLTWH(x, rect.Bottom-h, performanceBarWidth, h), fill)
			x += performanceBarWidth
		}
		fill.SetColor(overlayLine)
		canvas.DrawRect(ui.NewRectFromLTWH(rect.Left, rect.Bottom-rect.Height()/2, rect.Width(), 1), fill)
	}
	// A line at the bottom separates the graph from the one below.
	fill.SetColor(overlayDivider)
	canvas.DrawRect(ui.NewRectFromLTWH(rect.Left, rect.Bottom-1, rect.Width(), 1), fill)
	if text {
		r.paintLabel(canvas, rect, label, frames, duration)
	}
}

func milliseconds(d time.Duration) string {
	return strconv.FormatFloat(float64(d.Microseconds())/1000, 'f', 1, 64)
}

// statistics returns the text of the label: the average and the worst of the frames and
// the last one.
func overlayStatistics(label string, frames []ui.FrameTiming, duration func(ui.FrameTiming) time.Duration) string {
	if len(frames) == 0 {
		return label + ": no frames yet"
	}
	var sum, worst time.Duration
	for _, f := range frames {
		d := duration(f)
		sum += d
		worst = max(worst, d)
	}
	avg := sum / time.Duration(len(frames))
	return label + ": avg " + milliseconds(avg) + " ms, worst " + milliseconds(worst) + " ms, last " + milliseconds(duration(frames[len(frames)-1])) + " ms"
}

func (r *RenderPerformanceOverlay) paintLabel(canvas *ui.Canvas, rect ui.Rect, label string, frames []ui.FrameTiming, duration func(ui.FrameTiming) time.Duration) {
	if r.engine == nil {
		return
	}
	tp := painting.NewTextPainter(painting.TextPainterOpts{
		TextEngine:    r.engine,
		TextDirection: ui.TextDirectionLTR,
		Text: painting.NewTextSpan(painting.TextSpanOpts{
			Text: foundation.Some(overlayStatistics(label, frames, duration)),
			Style: painting.NewTextStyle(painting.TextStyleOpts{
				FontFamily: "Roboto", FontSize: foundation.Some(11.0), Color: foundation.Some(overlayText),
			}),
		}),
	})
	defer tp.Dispose()
	tp.Layout(0, rect.Width()-8)
	tp.Paint(canvas, ui.Offset{Dx: rect.Left + 4, Dy: rect.Top + 3})
}
