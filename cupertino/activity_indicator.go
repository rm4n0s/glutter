package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// activityIndicatorDefaultRadius is the default radius of the indicator.
const activityIndicatorDefaultRadius = 10.0

// activityIndicatorActiveTickColor is the default color of the ticks.
var activityIndicatorActiveTickColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF3C3C44), ui.NewColor(0xFFEBEBF5))

// CupertinoActivityIndicatorOpts are the optional parameters of NewCupertinoActivityIndicator.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.new
type CupertinoActivityIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color of the ticks; the default is a dynamic gray.
	Color ui.ColorValue
	// NotAnimating says that the indicator does not spin (Flutter's animating is true by default).
	NotAnimating bool
	// Radius is the radius of the indicator; the default is 10 and it must be greater than zero.
	Radius foundation.Opt[float64]
}

// CupertinoActivityIndicatorPartiallyRevealedOpts are the optional parameters of
// NewCupertinoActivityIndicatorPartiallyRevealed.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.partiallyRevealed
type CupertinoActivityIndicatorPartiallyRevealedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color of the ticks.
	Color ui.ColorValue
	// Radius is the radius of the indicator; the default is 10.
	Radius foundation.Opt[float64]
	// Progress is how much of the ticks show, from 0 to 1; the default is 1.
	Progress foundation.Opt[float64]
}

// CupertinoActivityIndicator is an iOS styled activity indicator: eight ticks, with a bright one that goes round.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator
type CupertinoActivityIndicator struct {
	widgets.StatefulWidget
	color     ui.ColorValue
	animating bool
	radius    float64
	progress  float64
}

// NewCupertinoActivityIndicator creates a spinning indicator. In debug builds it panics with the tag
// CupertinoActivityIndicatorRadiusInvalid when the radius is not greater than zero.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.new
func NewCupertinoActivityIndicator(opts ...CupertinoActivityIndicatorOpts) *CupertinoActivityIndicator {
	o := firstOf(opts)
	radius := o.Radius.OrElse(activityIndicatorDefaultRadius)
	if foundation.DebugMode && !(radius > 0.0) {
		panic(errors.New(tagCupertinoActivityIndicatorRadiusInvalid, "the radius must be greater than zero", "radius", radius))
	}
	return &CupertinoActivityIndicator{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o.Color, !o.NotAnimating, radius, 1.0}
}

// NewCupertinoActivityIndicatorPartiallyRevealed creates an indicator that does not spin and shows only a part of
// its ticks, for the pull to refresh. In debug builds it panics with the tags
// CupertinoActivityIndicatorRadiusInvalid and CupertinoActivityIndicatorProgressInvalid.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.partiallyRevealed
func NewCupertinoActivityIndicatorPartiallyRevealed(opts ...CupertinoActivityIndicatorPartiallyRevealedOpts) *CupertinoActivityIndicator {
	o := firstOf(opts)
	radius := o.Radius.OrElse(activityIndicatorDefaultRadius)
	progress := o.Progress.OrElse(1.0)
	if foundation.DebugMode {
		if !(radius > 0.0) {
			panic(errors.New(tagCupertinoActivityIndicatorRadiusInvalid, "the radius must be greater than zero", "radius", radius))
		}
		if !(progress >= 0.0 && progress <= 1.0) {
			panic(errors.New(tagCupertinoActivityIndicatorProgressInvalid, "the progress must be from 0 to 1", "progress", progress))
		}
	}
	return &CupertinoActivityIndicator{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o.Color, false, radius, progress}
}

// Color is the color of the ticks, or nil.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.color
func (w *CupertinoActivityIndicator) Color() ui.ColorValue { return w.color }

// Animating is whether the indicator spins.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.animating
func (w *CupertinoActivityIndicator) Animating() bool { return w.animating }

// Radius is the radius of the indicator.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.radius
func (w *CupertinoActivityIndicator) Radius() float64 { return w.radius }

// Progress is how much of the ticks show.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.progress
func (w *CupertinoActivityIndicator) Progress() float64 { return w.progress }

// CreateState creates the state.
//
// Flutter: cupertino/activity_indicator.dart CupertinoActivityIndicator.createState
func (w *CupertinoActivityIndicator) CreateState() widgets.State {
	return &cupertinoActivityIndicatorState{}
}

type cupertinoActivityIndicatorState struct {
	widgets.StateOf[*CupertinoActivityIndicator]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
}

func (s *cupertinoActivityIndicatorState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(time.Second)})
	if s.Widget().animating {
		s.controller.Repeat()
	}
}

func (s *cupertinoActivityIndicatorState) DidUpdateWidget(old *CupertinoActivityIndicator) {
	if s.Widget().animating != old.animating {
		if s.Widget().animating {
			s.controller.Repeat()
		} else {
			s.controller.Stop()
		}
	}
}

func (s *cupertinoActivityIndicatorState) Dispose() { s.controller.Dispose() }

func (s *cupertinoActivityIndicatorState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var color ui.Color
	if w.color != nil {
		color = w.color.EffectiveColor()
	} else {
		color = CupertinoDynamicColorResolve(activityIndicatorActiveTickColor, bc)
	}
	return widgets.NewSizedBoxSquare(foundation.Some(w.radius*2), widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{
		Painter: newCupertinoActivityIndicatorPainter(s.controller, color, w.radius, w.progress)}))
}

var activityIndicatorAlphaValues = []int{47, 47, 47, 47, 72, 97, 122, 147}

const activityIndicatorPartiallyRevealedAlpha = 147

// cupertinoActivityIndicatorPainter paints the ticks.
//
// Flutter: cupertino/activity_indicator.dart _CupertinoActivityIndicatorPainter
type cupertinoActivityIndicatorPainter struct {
	rendering.CustomPainterBase
	position             animation.Animation[float64]
	activeColor          ui.Color
	radius, progress     float64
	tickFundamentalShape ui.RRect
}

func newCupertinoActivityIndicatorPainter(position animation.Animation[float64], activeColor ui.Color, radius, progress float64) *cupertinoActivityIndicatorPainter {
	return &cupertinoActivityIndicatorPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(position), position: position, activeColor: activeColor, radius: radius, progress: progress,
		tickFundamentalShape: ui.NewRRectFromLTRBXY(-radius/activityIndicatorDefaultRadius, -radius/3.0, radius/activityIndicatorDefaultRadius, -radius,
			radius/activityIndicatorDefaultRadius, radius/activityIndicatorDefaultRadius),
	}
}

func (p *cupertinoActivityIndicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	tickCount := len(activityIndicatorAlphaValues)
	canvas.Save()
	canvas.Translate(size.Width/2.0, size.Height/2.0)
	activeTick := int(math.Floor(float64(tickCount) * p.position.Value()))
	for i := 0; float64(i) < float64(tickCount)*p.progress; i++ {
		t := ((i-activeTick)%tickCount + tickCount) % tickCount
		alpha := activityIndicatorAlphaValues[t]
		if p.progress < 1 {
			alpha = activityIndicatorPartiallyRevealedAlpha
		}
		paint.SetColor(p.activeColor.WithAlpha(alpha))
		canvas.DrawRRect(p.tickFundamentalShape, paint)
		canvas.Rotate(2 * math.Pi / float64(tickCount))
	}
	canvas.Restore()
}

func (p *cupertinoActivityIndicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*cupertinoActivityIndicatorPainter)
	return !ok || o.position != p.position || o.activeColor != p.activeColor || o.progress != p.progress
}

// CupertinoLinearActivityIndicatorOpts are the optional parameters of NewCupertinoLinearActivityIndicator.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.new
type CupertinoLinearActivityIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Height is the height of the bar; the default is 4.5 and it must be greater than zero.
	Height foundation.Opt[float64]
	// Color is the color of the progress; the default is the active blue.
	Color ui.ColorValue
}

// CupertinoLinearActivityIndicator is an iOS styled linear progress bar: a rounded track with the part that is done.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator
type CupertinoLinearActivityIndicator struct {
	widgets.StatelessWidget
	progress float64
	height   float64
	color    ui.ColorValue
}

// NewCupertinoLinearActivityIndicator creates a bar with the progress (from 0 to 1). In debug builds it panics with
// the tags CupertinoActivityIndicatorHeightInvalid and CupertinoActivityIndicatorProgressInvalid.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.new
func NewCupertinoLinearActivityIndicator(progress float64, opts ...CupertinoLinearActivityIndicatorOpts) *CupertinoLinearActivityIndicator {
	o := firstOf(opts)
	height := o.Height.OrElse(4.5)
	if foundation.DebugMode {
		if !(height > 0) {
			panic(errors.New(tagCupertinoActivityIndicatorHeightInvalid, "the height must be greater than zero", "height", height))
		}
		if !(progress >= 0.0 && progress <= 1.0) {
			panic(errors.New(tagCupertinoActivityIndicatorProgressInvalid, "the progress must be from 0 to 1", "progress", progress))
		}
	}
	return &CupertinoLinearActivityIndicator{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), progress, height, o.Color}
}

// Progress is how much of the bar is done.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.progress
func (w *CupertinoLinearActivityIndicator) Progress() float64 { return w.progress }

// Height is the height of the bar.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.height
func (w *CupertinoLinearActivityIndicator) Height() float64 { return w.height }

// Color is the color of the progress, or nil.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.color
func (w *CupertinoLinearActivityIndicator) Color() ui.ColorValue { return w.color }

// Build builds the bar.
//
// Flutter: cupertino/activity_indicator.dart CupertinoLinearActivityIndicator.build
func (w *CupertinoLinearActivityIndicator) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewConstrainedBox(rendering.BoxConstraints{MinHeight: w.height, MinWidth: infinity, MaxWidth: infinity, MaxHeight: infinity},
		widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: &linearActivityIndicatorPainter{progress: w.progress, color: w.color}}))
}

// linearActivityIndicatorPainter paints the track and the progress of the bar.
//
// Flutter: cupertino/activity_indicator.dart _CupertinoLinearActivityIndicator
type linearActivityIndicatorPainter struct {
	rendering.CustomPainterBase
	progress float64
	color    ui.ColorValue
}

func (p *linearActivityIndicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	background := ui.NewPaint()
	background.SetColor(CupertinoColors.SystemFill.EffectiveColor())
	radius := ui.NewRadiusCircular(size.Height / 2)
	canvas.DrawRRect(painting.NewBorderRadiusAll(radius).ToRRect(ui.NewRectFromLTWH(0, 0, size.Width, size.Height)), background)
	if p.progress > 0 {
		progressPaint := ui.NewPaint()
		if p.color != nil {
			progressPaint.SetColor(p.color.EffectiveColor())
		} else {
			progressPaint.SetColor(CupertinoColors.ActiveBlue.EffectiveColor())
		}
		canvas.DrawRRect(painting.NewBorderRadiusAll(radius).ToRRect(ui.NewRectFromLTWH(0, 0, clampDouble(p.progress, 0.0, 1.0)*size.Width, size.Height)), progressPaint)
	}
}

func (p *linearActivityIndicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*linearActivityIndicatorPainter)
	return !ok || o.progress != p.progress || o.color != p.color
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoActivityIndicatorRadiusInvalid   = "CupertinoActivityIndicatorRadiusInvalid"
	tagCupertinoActivityIndicatorProgressInvalid = "CupertinoActivityIndicatorProgressInvalid"
	tagCupertinoActivityIndicatorHeightInvalid   = "CupertinoActivityIndicatorHeightInvalid"
)
