// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	indeterminateLinearDuration   = 1800
	indeterminateCircularDuration = 1333 * 2222
	trackGapRampDownThreshold     = 0.01
)

// LinearProgressIndicatorDefaultAnimationDuration is the duration of one cycle of
// the indeterminate linear animation.
//
// Flutter: material/progress_indicator.dart LinearProgressIndicator.defaultAnimationDuration
const LinearProgressIndicatorDefaultAnimationDuration = indeterminateLinearDuration * time.Millisecond

// CircularProgressIndicatorDefaultAnimationDuration is the duration of the whole
// indeterminate circular animation, which is the least common multiple of the
// cycles of the arc and of the rotation.
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator.defaultAnimationDuration
const CircularProgressIndicatorDefaultAnimationDuration = indeterminateCircularDuration * time.Millisecond

// The places of the stroke of a CircularProgressIndicator relative to its box.
const (
	// CircularProgressIndicatorStrokeAlignInside is Flutter's strokeAlignInside.
	//
	// Flutter: material/progress_indicator.dart CircularProgressIndicator.strokeAlignInside
	CircularProgressIndicatorStrokeAlignInside = -1.0
	// CircularProgressIndicatorStrokeAlignCenter is Flutter's strokeAlignCenter.
	//
	// Flutter: material/progress_indicator.dart CircularProgressIndicator.strokeAlignCenter
	CircularProgressIndicatorStrokeAlignCenter = 0.0
	// CircularProgressIndicatorStrokeAlignOutside is Flutter's strokeAlignOutside.
	//
	// Flutter: material/progress_indicator.dart CircularProgressIndicator.strokeAlignOutside
	CircularProgressIndicatorStrokeAlignOutside = 1.0
)

// RefreshProgressIndicatorDefaultStrokeWidth is the default stroke width of a
// RefreshProgressIndicator.
//
// Flutter: material/progress_indicator.dart RefreshProgressIndicator.defaultStrokeWidth
const RefreshProgressIndicatorDefaultStrokeWidth = 2.5

// ProgressIndicator is what the progress indicators have in common: the value,
// the colors and the semantics.
//
// Flutter: material/progress_indicator.dart ProgressIndicator
type ProgressIndicator struct {
	widgets.StatefulWidget
	value           foundation.Opt[float64]
	backgroundColor foundation.Opt[ui.Color]
	color           foundation.Opt[ui.Color]
	valueColor      animation.Animation[foundation.Opt[ui.Color]]
	semanticsLabel  string
	semanticsValue  string
}

// Value is the progress of a determinate indicator between 0 and 1, or none for
// an indeterminate one.
//
// Flutter: material/progress_indicator.dart ProgressIndicator.value
func (w *ProgressIndicator) Value() foundation.Opt[float64] { return w.value }

// BackgroundColor is the color of the track.
//
// Flutter: material/progress_indicator.dart ProgressIndicator.backgroundColor
func (w *ProgressIndicator) BackgroundColor() foundation.Opt[ui.Color] { return w.backgroundColor }

// Color is the color of the active part.
//
// Flutter: material/progress_indicator.dart ProgressIndicator.color
func (w *ProgressIndicator) Color() foundation.Opt[ui.Color] { return w.color }

// SemanticsLabel is the label that tells what the indicator is for.
//
// Flutter: material/progress_indicator.dart ProgressIndicator.semanticsLabel
func (w *ProgressIndicator) SemanticsLabel() string { return w.semanticsLabel }

// SemanticsValue is the value that the semantics tell: the percentage of a
// determinate indicator by default.
//
// Flutter: material/progress_indicator.dart ProgressIndicator.semanticsValue
func (w *ProgressIndicator) SemanticsValue() string { return w.semanticsValue }

func (w *ProgressIndicator) effectiveValue() foundation.Opt[float64] {
	if v, ok := w.value.Get(); ok {
		return foundation.Some(ui.ClampDouble(v, 0, 1))
	}
	return w.value
}

func (w *ProgressIndicator) getValueColor(bc widgets.BuildContext, defaultColor foundation.Opt[ui.Color]) ui.Color {
	if w.valueColor != nil {
		if c, ok := w.valueColor.Value().Get(); ok {
			return c
		}
	}
	return firstOpt(w.color, ProgressIndicatorThemeOf(bc).Color, defaultColor, foundation.Some(ThemeOf(bc).ColorScheme.Primary())).Value()
}

func (w *ProgressIndicator) buildSemanticsWrapper(child widgets.Widget) widgets.Widget {
	isProgressBar := false
	value := w.semanticsValue
	if v, ok := w.effectiveValue().Get(); ok {
		if value == "" {
			value = fmt.Sprint(int(math.Round(v * 100)))
		}
		isProgressBar = true
	}
	o := widgets.SemanticsOpts{Value: optString(value)}
	if w.semanticsLabel != "" {
		o.Label = foundation.Some(w.semanticsLabel)
	}
	if isProgressBar {
		o.Role = foundation.Some(ui.SemanticsRoleProgressBar)
		o.MinValue, o.MaxValue = foundation.Some("0"), foundation.Some("100")
	} else {
		o.Role = foundation.Some(ui.SemanticsRoleLoadingSpinner)
	}
	return widgets.NewSemantics(child, o)
}

// progressControllers picks the controller of the indeterminate animation: the
// one of the widget, of the theme, or the one of the state.
type progressControllers struct {
	internal *animation.AnimationController
}

func (c *progressControllers) init(vsync scheduler.TickerProvider, duration time.Duration) {
	c.internal = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{Duration: foundation.Some(duration)})
}

func (c *progressControllers) controller(bc widgets.BuildContext, widgetController *animation.AnimationController) *animation.AnimationController {
	if widgetController != nil {
		return widgetController
	}
	if themed := ProgressIndicatorThemeOf(bc).Controller; themed != nil {
		return themed
	}
	return c.internal
}

// update repeats the internal controller for an indeterminate indicator and stops
// it for a determinate one.
func (c *progressControllers) update(value foundation.Opt[float64]) {
	if value.IsNone() && !c.internal.IsAnimating() {
		c.internal.Repeat()
	} else if value.IsSome() && c.internal.IsAnimating() {
		c.internal.Stop()
	}
}

// LinearProgressIndicatorOpts are the optional parameters of
// NewLinearProgressIndicator.
//
// Flutter: material/progress_indicator.dart LinearProgressIndicator
type LinearProgressIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Value is the progress between 0 and 1; none is an indeterminate indicator.
	Value foundation.Opt[float64]
	// BackgroundColor is the color of the track.
	BackgroundColor foundation.Opt[ui.Color]
	// Color is the color of the active part.
	Color foundation.Opt[ui.Color]
	// ValueColor is an animated color of the active part; it wins over Color.
	ValueColor animation.Animation[foundation.Opt[ui.Color]]
	// MinHeight is the minimum height of the line (it panics in debug builds with
	// the tag LinearProgressIndicatorMinHeightNotPositive when it is not
	// positive).
	MinHeight foundation.Opt[float64]
	// SemanticsLabel is the label of the semantics.
	SemanticsLabel string
	// SemanticsValue is the value of the semantics.
	SemanticsValue string
	// BorderRadius is the border radius of both the track and the indicator.
	BorderRadius painting.BorderRadiusGeometry
	// StopIndicatorColor is the color of the stop indicator of a determinate
	// indicator (not in the 2023 look).
	StopIndicatorColor foundation.Opt[ui.Color]
	// StopIndicatorRadius is the radius of the stop indicator.
	StopIndicatorRadius foundation.Opt[float64]
	// TrackGap is the gap between the active part and the track.
	TrackGap foundation.Opt[float64]
	// Year2023 selects the 2023 look, which is the default.
	Year2023 foundation.Opt[bool]
	// Controller is the controller of the indeterminate animation. It cannot be
	// given with a Value (the tag ProgressIndicatorValueAndController).
	Controller *animation.AnimationController
}

// LinearProgressIndicator is a line that shows the progress of a task, or that a
// task is running when its progress is not known.
//
// Flutter: material/progress_indicator.dart LinearProgressIndicator
type LinearProgressIndicator struct {
	ProgressIndicator
	o LinearProgressIndicatorOpts
}

// NewLinearProgressIndicator creates a linear progress indicator.
//
// Flutter: material/progress_indicator.dart LinearProgressIndicator.new
func NewLinearProgressIndicator(opts ...LinearProgressIndicatorOpts) *LinearProgressIndicator {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if v, ok := o.MinHeight.Get(); ok && !(v > 0) {
			panic(errors.New(tagLinearProgressIndicatorMinHeightNotPositive, "minHeight must be greater than zero", "minHeight", v))
		}
		if o.Value.IsSome() && o.Controller != nil {
			panic(errors.New(tagLinearProgressIndicatorValueAndController, "A progress indicator cannot have both a value and a controller. "+
				"The \"value\" property is for a determinate indicator with a specific progress, "+
				"while the \"controller\" is for controlling the animation of an indeterminate indicator. "+
				"To resolve this, provide only one of the two properties."))
		}
	}
	return &LinearProgressIndicator{
		ProgressIndicator: ProgressIndicator{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: o.Value,
			backgroundColor: o.BackgroundColor, color: o.Color, valueColor: o.ValueColor, semanticsLabel: o.SemanticsLabel, semanticsValue: o.SemanticsValue},
		o: o,
	}
}

// CreateState creates the state.
//
// Flutter: material/progress_indicator.dart LinearProgressIndicator.createState
func (w *LinearProgressIndicator) CreateState() widgets.State { return &linearProgressIndicatorState{} }

type linearProgressIndicatorState struct {
	widgets.StateOf[*LinearProgressIndicator]
	widgets.SingleTickerProvider
	controllers progressControllers
}

func (s *linearProgressIndicatorState) InitState() {
	s.controllers.init(s, LinearProgressIndicatorDefaultAnimationDuration)
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *linearProgressIndicatorState) DidUpdateWidget(*LinearProgressIndicator) {
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *linearProgressIndicatorState) Dispose() { s.controllers.internal.Dispose() }

func linearDefaults(bc widgets.BuildContext, year2023 bool) ProgressIndicatorThemeData {
	colors := ThemeOf(bc).ColorScheme
	switch {
	case !ThemeOf(bc).UseMaterial3:
		return ProgressIndicatorThemeData{Color: foundation.Some(colors.Primary()), LinearTrackColor: foundation.Some(colors.Background()), LinearMinHeight: foundation.Some(4.0)}
	case year2023:
		return ProgressIndicatorThemeData{Color: foundation.Some(colors.Primary()), LinearTrackColor: foundation.Some(colors.SecondaryContainer()), LinearMinHeight: foundation.Some(4.0)}
	}
	return ProgressIndicatorThemeData{
		Color: foundation.Some(colors.Primary()), LinearTrackColor: foundation.Some(colors.SecondaryContainer()), LinearMinHeight: foundation.Some(4.0),
		BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0 / 2)), StopIndicatorColor: foundation.Some(colors.Primary()),
		StopIndicatorRadius: foundation.Some(4.0 / 2), TrackGap: foundation.Some(4.0)}
}

func (s *linearProgressIndicatorState) buildIndicator(bc widgets.BuildContext, animationValue float64, textDirection ui.TextDirection) widgets.Widget {
	w := s.Widget()
	indicatorTheme := ProgressIndicatorThemeOf(bc)
	year2023 := firstOpt(w.o.Year2023, indicatorTheme.Year2023, foundation.Some(true)).Value()
	defaults := linearDefaults(bc, year2023)
	trackColor := firstOpt(w.o.BackgroundColor, indicatorTheme.LinearTrackColor, defaults.LinearTrackColor).Value()
	minHeight := firstOpt(w.o.MinHeight, indicatorTheme.LinearMinHeight, defaults.LinearMinHeight).Value()
	borderRadius := firstNonNil(w.o.BorderRadius, indicatorTheme.BorderRadius, defaults.BorderRadius)
	var stopColor foundation.Opt[ui.Color]
	var stopRadius, trackGap foundation.Opt[float64]
	if !year2023 {
		stopColor = firstOpt(w.o.StopIndicatorColor, indicatorTheme.StopIndicatorColor, defaults.StopIndicatorColor)
		stopRadius = firstOpt(w.o.StopIndicatorRadius, indicatorTheme.StopIndicatorRadius, defaults.StopIndicatorRadius)
		trackGap = firstOpt(w.o.TrackGap, indicatorTheme.TrackGap, defaults.TrackGap)
	}
	var result widgets.Widget = widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
		MinWidth: foundation.Some(math.Inf(1)), MinHeight: foundation.Some(minHeight)}),
		widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: &linearProgressIndicatorPainter{
			trackColor: trackColor, valueColor: w.getValueColor(bc, defaults.Color), value: w.effectiveValue(), animationValue: animationValue,
			textDirection: textDirection, indicatorBorderRadius: borderRadius, stopIndicatorColor: stopColor, stopIndicatorRadius: stopRadius, trackGap: trackGap}}))
	if borderRadius != nil && w.effectiveValue().IsNone() {
		result = widgets.NewClipRRect(result, widgets.ClipRRectOpts{BorderRadius: borderRadius})
	}
	return w.buildSemanticsWrapper(result)
}

func (s *linearProgressIndicatorState) Build(bc widgets.BuildContext) widgets.Widget {
	textDirection := widgets.DirectionalityOf(bc)
	controller := s.controllers.controller(bc, s.Widget().o.Controller)
	if s.Widget().effectiveValue().IsSome() {
		return s.buildIndicator(bc, controller.Value(), textDirection)
	}
	return widgets.NewAnimatedBuilder(controller, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
		return s.buildIndicator(bc, controller.Value(), textDirection)
	}, nil)
}

var (
	linearLine1Head = animation.NewInterval(0, 750.0/indeterminateLinearDuration, animation.NewCubic(0.2, 0.0, 0.8, 1.0))
	linearLine1Tail = animation.NewInterval(333.0/indeterminateLinearDuration, (333.0+750.0)/indeterminateLinearDuration, animation.NewCubic(0.4, 0.0, 1.0, 1.0))
	linearLine2Head = animation.NewInterval(1000.0/indeterminateLinearDuration, (1000.0+567.0)/indeterminateLinearDuration, animation.NewCubic(0.0, 0.0, 0.65, 1.0))
	linearLine2Tail = animation.NewInterval(1267.0/indeterminateLinearDuration, (1267.0+533.0)/indeterminateLinearDuration, animation.NewCubic(0.10, 0.0, 0.45, 1.0))
)

// linearProgressIndicatorPainter paints the line.
//
// Flutter: material/progress_indicator.dart _LinearProgressIndicatorPainter
type linearProgressIndicatorPainter struct {
	rendering.CustomPainterBase
	trackColor            ui.Color
	valueColor            ui.Color
	value                 foundation.Opt[float64]
	animationValue        float64
	textDirection         ui.TextDirection
	indicatorBorderRadius painting.BorderRadiusGeometry
	stopIndicatorColor    foundation.Opt[ui.Color]
	stopIndicatorRadius   foundation.Opt[float64]
	trackGap              foundation.Opt[float64]
}

func (p *linearProgressIndicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	drawLinear := func(start, end float64, color ui.Color) {
		if end-start <= 0 {
			return
		}
		ltr := p.textDirection == ui.TextDirectionLTR
		left, right := start, end
		if !ltr {
			left, right = 1-end, 1-start
		}
		rect := ui.NewRectFromLTRB(left*size.Width, 0, right*size.Width, size.Height)
		paint := ui.NewPaint()
		paint.SetColor(color)
		if p.indicatorBorderRadius != nil {
			canvas.DrawRRect(p.indicatorBorderRadius.Resolve(p.textDirection).ToRRect(rect), paint)
		} else {
			canvas.DrawRect(rect, paint)
		}
	}
	drawStop := func() {
		maxRadius := size.Height / 2
		radius := math.Min(p.stopIndicatorRadius.Value(), maxRadius)
		paint := ui.NewPaint()
		paint.SetColor(p.stopIndicatorColor.Value())
		position := ui.Offset{Dx: size.Width - maxRadius, Dy: maxRadius}
		if p.textDirection == ui.TextDirectionRTL {
			position = ui.Offset{Dx: maxRadius, Dy: maxRadius}
		}
		canvas.DrawCircle(position, radius, paint)
	}
	gapFraction := func(current, trackGapFraction float64) float64 {
		return trackGapFraction * ui.ClampDouble(current, 0, trackGapRampDownThreshold) / trackGapRampDownThreshold
	}
	trackGapFraction := p.trackGap.OrElse(0) / size.Width
	if v, ok := p.value.Get(); ok {
		v = ui.ClampDouble(v, 0, 1)
		trackStart := 0.0
		if trackGapFraction > 0 {
			trackStart = v + gapFraction(v, trackGapFraction)
		}
		if trackStart < 1 {
			drawLinear(trackStart, 1, p.trackColor)
		}
		if r, ok := p.stopIndicatorRadius.Get(); ok && r > 0 {
			drawStop()
		}
		if v > 0 {
			drawLinear(0, v, p.valueColor)
		}
		return
	}
	firstHead := linearLine1Head.Transform(p.animationValue)
	firstTail := linearLine1Tail.Transform(p.animationValue)
	secondHead := linearLine2Head.Transform(p.animationValue)
	secondTail := linearLine2Tail.Transform(p.animationValue)
	if firstHead < 1-trackGapFraction {
		start := 0.0
		if firstHead > 0 {
			start = firstHead + gapFraction(firstHead, trackGapFraction)
		}
		drawLinear(start, 1, p.trackColor)
	}
	if firstHead-firstTail > 0 {
		drawLinear(firstTail, firstHead, p.valueColor)
	}
	if firstTail > trackGapFraction {
		start := 0.0
		if secondHead > 0 {
			start = secondHead + gapFraction(secondHead, trackGapFraction)
		}
		end := 1.0
		if firstTail < 1 {
			end = firstTail - gapFraction(1-firstTail, trackGapFraction)
		}
		drawLinear(start, end, p.trackColor)
	}
	if secondHead-secondTail > 0 {
		drawLinear(secondTail, secondHead, p.valueColor)
	}
	if secondTail > trackGapFraction {
		end := 1.0
		if secondTail < 1 {
			end = secondTail - gapFraction(1-secondTail, trackGapFraction)
		}
		drawLinear(0, end, p.trackColor)
	}
}

func (p *linearProgressIndicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*linearProgressIndicatorPainter)
	return !ok || o.trackColor != p.trackColor || o.valueColor != p.valueColor || o.value != p.value || o.animationValue != p.animationValue ||
		o.textDirection != p.textDirection || !foundation.ValuesEqual(o.indicatorBorderRadius, p.indicatorBorderRadius) ||
		o.stopIndicatorColor != p.stopIndicatorColor || o.stopIndicatorRadius != p.stopIndicatorRadius || o.trackGap != p.trackGap
}

// CircularProgressIndicatorOpts are the optional parameters of
// NewCircularProgressIndicator.
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator
type CircularProgressIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Value is the progress between 0 and 1; none is an indeterminate indicator.
	Value foundation.Opt[float64]
	// BackgroundColor is the color of the track.
	BackgroundColor foundation.Opt[ui.Color]
	// Color is the color of the active part.
	Color foundation.Opt[ui.Color]
	// ValueColor is an animated color of the active part; it wins over Color.
	ValueColor animation.Animation[foundation.Opt[ui.Color]]
	// StrokeWidth is the width of the line.
	StrokeWidth foundation.Opt[float64]
	// StrokeAlign is where the stroke is placed: -1 inside the box, 0 on its
	// edge and 1 outside.
	StrokeAlign foundation.Opt[float64]
	// SemanticsLabel is the label of the semantics.
	SemanticsLabel string
	// SemanticsValue is the value of the semantics.
	SemanticsValue string
	// StrokeCap is the shape of the ends of the line.
	StrokeCap foundation.Opt[ui.StrokeCap]
	// Constraints are the size constraints of the indicator.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// TrackGap is the gap between the active part and the track (not in the 2023
	// look).
	TrackGap foundation.Opt[float64]
	// Year2023 selects the 2023 look, which is the default.
	Year2023 foundation.Opt[bool]
	// Padding is the padding around the track.
	Padding painting.EdgeInsetsGeometry
	// Controller is the controller of the indeterminate animation. It cannot be
	// given with a Value (the tag ProgressIndicatorValueAndController).
	Controller *animation.AnimationController
}

// CircularProgressIndicator is a circle that spins to show that a task is
// running, or that fills to show its progress.
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator
type CircularProgressIndicator struct {
	ProgressIndicator
	o        CircularProgressIndicatorOpts
	adaptive bool
}

// NewCircularProgressIndicator creates a circular progress indicator.
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator.new
func NewCircularProgressIndicator(opts ...CircularProgressIndicatorOpts) *CircularProgressIndicator {
	return newCircularProgressIndicator(firstOpts(opts), false)
}

// NewCircularProgressIndicatorAdaptive creates a circular progress indicator
// that is a Cupertino one on iOS and macOS in Flutter.
//
// Deviation: Glutter has no Cupertino library, so the indicator is the Material
// one on every platform (SPEC §12.10).
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator.adaptive
func NewCircularProgressIndicatorAdaptive(opts ...CircularProgressIndicatorOpts) *CircularProgressIndicator {
	return newCircularProgressIndicator(firstOpts(opts), true)
}

func newCircularProgressIndicator(o CircularProgressIndicatorOpts, adaptive bool) *CircularProgressIndicator {
	if foundation.DebugMode && o.Value.IsSome() && o.Controller != nil {
		panic(errors.New(tagCircularProgressIndicatorValueAndController, "A progress indicator cannot have both a value and a controller. "+
			"The \"value\" property is for a determinate indicator with a specific progress, "+
			"while the \"controller\" is for controlling the animation of an indeterminate indicator. "+
			"To resolve this, provide only one of the two properties."))
	}
	return &CircularProgressIndicator{
		ProgressIndicator: ProgressIndicator{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: o.Value,
			backgroundColor: o.BackgroundColor, color: o.Color, valueColor: o.ValueColor, semanticsLabel: o.SemanticsLabel, semanticsValue: o.SemanticsValue},
		o: o, adaptive: adaptive,
	}
}

// CreateState creates the state.
//
// Flutter: material/progress_indicator.dart CircularProgressIndicator.createState
func (w *CircularProgressIndicator) CreateState() widgets.State {
	return &circularProgressIndicatorState{}
}

const (
	circularPathCount     = indeterminateCircularDuration / 1333
	circularRotationCount = indeterminateCircularDuration / 2222
)

var (
	strokeHeadTween = animation.NewCurveTween(animation.NewInterval(0.0, 0.5, animation.Curves.FastOutSlowIn)).
			Chain(animation.NewCurveTween(animation.NewSawTooth(circularPathCount)))
	strokeTailTween = animation.NewCurveTween(animation.NewInterval(0.5, 1.0, animation.Curves.FastOutSlowIn)).
			Chain(animation.NewCurveTween(animation.NewSawTooth(circularPathCount)))
	offsetTween   = animation.NewCurveTween(animation.NewSawTooth(circularPathCount))
	rotationTween = animation.NewCurveTween(animation.NewSawTooth(circularRotationCount))
)

type circularProgressIndicatorState struct {
	widgets.StateOf[*CircularProgressIndicator]
	widgets.SingleTickerProvider
	controllers progressControllers
}

func (s *circularProgressIndicatorState) InitState() {
	s.controllers.init(s, CircularProgressIndicatorDefaultAnimationDuration)
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *circularProgressIndicatorState) DidUpdateWidget(*CircularProgressIndicator) {
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *circularProgressIndicatorState) Dispose() { s.controllers.internal.Dispose() }

func circularDefaults(bc widgets.BuildContext, year2023, indeterminate bool) ProgressIndicatorThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	switch {
	case !theme.UseMaterial3:
		return ProgressIndicatorThemeData{Color: foundation.Some(colors.Primary()), StrokeWidth: foundation.Some(4.0),
			StrokeAlign: foundation.Some(CircularProgressIndicatorStrokeAlignCenter),
			Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(36.0), MinHeight: foundation.Some(36.0)}))}
	case year2023:
		return ProgressIndicatorThemeData{Color: foundation.Some(colors.Primary()), StrokeWidth: foundation.Some(4.0),
			StrokeAlign: foundation.Some(CircularProgressIndicatorStrokeAlignCenter),
			Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(36.0), MinHeight: foundation.Some(36.0)}))}
	}
	d := ProgressIndicatorThemeData{Color: foundation.Some(colors.Primary()), StrokeWidth: foundation.Some(4.0),
		StrokeAlign: foundation.Some(CircularProgressIndicatorStrokeAlignInside),
		Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(40.0), MinHeight: foundation.Some(40.0)})),
		TrackGap:    foundation.Some(4.0), CircularTrackPadding: painting.NewEdgeInsetsAll(4.0)}
	if !indeterminate {
		d.CircularTrackColor = foundation.Some(colors.SecondaryContainer())
	}
	return d
}

func (s *circularProgressIndicatorState) buildMaterialIndicator(bc widgets.BuildContext, headValue, tailValue, offsetValue, rotationValue float64) widgets.Widget {
	w := s.Widget()
	indicatorTheme := ProgressIndicatorThemeOf(bc)
	year2023 := firstOpt(w.o.Year2023, indicatorTheme.Year2023, foundation.Some(true)).Value()
	defaults := circularDefaults(bc, year2023, w.effectiveValue().IsNone())
	trackColor := firstOpt(w.o.BackgroundColor, indicatorTheme.CircularTrackColor, defaults.CircularTrackColor)
	strokeWidth := firstOpt(w.o.StrokeWidth, indicatorTheme.StrokeWidth, defaults.StrokeWidth).Value()
	strokeAlign := firstOpt(w.o.StrokeAlign, indicatorTheme.StrokeAlign, defaults.StrokeAlign).Value()
	strokeCap := firstOpt(w.o.StrokeCap, indicatorTheme.StrokeCap)
	constraints := firstOpt(w.o.Constraints, indicatorTheme.Constraints, defaults.Constraints).Value()
	var trackGap foundation.Opt[float64]
	if !year2023 {
		trackGap = firstOpt(w.o.TrackGap, indicatorTheme.TrackGap, defaults.TrackGap)
	}
	padding := firstNonNil(w.o.Padding, indicatorTheme.CircularTrackPadding, defaults.CircularTrackPadding)
	var result widgets.Widget = widgets.NewConstrainedBox(constraints, widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: newCircularProgressIndicatorPainter(
		circularPainterOpts{trackColor: trackColor, valueColor: w.getValueColor(bc, defaults.Color), value: w.effectiveValue(), headValue: headValue,
			tailValue: tailValue, offsetValue: offsetValue, rotationValue: rotationValue, strokeWidth: strokeWidth, strokeAlign: strokeAlign,
			strokeCap: strokeCap, trackGap: trackGap, year2023: year2023})}))
	if padding != nil {
		result = widgets.NewPadding(padding, result)
	}
	return w.buildSemanticsWrapper(result)
}

func (s *circularProgressIndicatorState) buildAnimation(bc widgets.BuildContext) widgets.Widget {
	controller := s.controllers.controller(bc, s.Widget().o.Controller)
	return widgets.NewAnimatedBuilder(controller, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
		v := controller.Value()
		return s.buildMaterialIndicator(bc, strokeHeadTween.Transform(v), strokeTailTween.Transform(v), offsetTween.Transform(v), rotationTween.Transform(v))
	}, nil)
}

func (s *circularProgressIndicatorState) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		if s.Widget().effectiveValue().IsSome() {
			return s.buildMaterialIndicator(bc, 0, 0, 0, 0)
		}
		return s.buildAnimation(bc)
	})
}

const (
	circularEpsilon    = .001
	circularSweep      = math.Pi*2.0 - circularEpsilon
	circularStartAngle = -math.Pi / 2.0
)

type circularPainterOpts struct {
	trackColor                                       foundation.Opt[ui.Color]
	valueColor                                       ui.Color
	value                                            foundation.Opt[float64]
	headValue, tailValue, offsetValue, rotationValue float64
	strokeWidth, strokeAlign                         float64
	strokeCap                                        foundation.Opt[ui.StrokeCap]
	trackGap                                         foundation.Opt[float64]
	year2023                                         bool
}

// circularProgressIndicatorPainter paints the arc.
//
// Flutter: material/progress_indicator.dart _CircularProgressIndicatorPainter
type circularProgressIndicatorPainter struct {
	rendering.CustomPainterBase
	circularPainterOpts
	arcStart, arcSweep float64
}

func newCircularProgressIndicatorPainter(o circularPainterOpts) *circularProgressIndicatorPainter {
	p := &circularProgressIndicatorPainter{circularPainterOpts: o}
	if v, ok := o.value.Get(); ok {
		p.arcStart = circularStartAngle
		p.arcSweep = ui.ClampDouble(v, 0, 1) * circularSweep
	} else {
		p.arcStart = circularStartAngle + o.tailValue*3/2*math.Pi + o.rotationValue*math.Pi*2.0 + o.offsetValue*0.5*math.Pi
		p.arcSweep = math.Max(o.headValue*3/2*math.Pi-o.tailValue*3/2*math.Pi, circularEpsilon)
	}
	return p
}

func (p *circularProgressIndicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetColor(p.valueColor)
	paint.SetStrokeWidth(p.strokeWidth)
	paint.SetStyle(ui.PaintingStyleStroke)
	strokeOffset := p.strokeWidth / 2 * -p.strokeAlign
	arcBase := ui.Offset{Dx: strokeOffset, Dy: strokeOffset}
	arcSize := ui.Size{Width: size.Width - strokeOffset*2, Height: size.Height - strokeOffset*2}
	hasGap := p.trackGap.OrElse(0) > 0
	if trackColor, ok := p.trackColor.Get(); ok {
		background := ui.NewPaint()
		background.SetColor(trackColor)
		background.SetStrokeWidth(p.strokeWidth)
		background.SetStrokeCap(p.strokeCap.OrElse(ui.StrokeCapRound))
		background.SetStyle(ui.PaintingStyleStroke)
		if v, hasValue := p.value.Get(); hasGap && hasValue && v > circularEpsilon {
			arcRadius := arcSize.ShortestSide() / 2
			strokeRadius := p.strokeWidth / arcRadius
			gapRadius := p.trackGap.Value() / arcRadius
			startGap := strokeRadius + gapRadius
			endGap := startGap * 2
			if v < circularEpsilon {
				endGap = startGap
			}
			startSweep := -math.Pi/2.0 + startGap
			endSweep := math.Max(0.0, math.Pi*2.0-ui.ClampDouble(v, 0, 1)*math.Pi*2.0-endGap)
			canvas.Save()
			canvas.Scale(-1, 1)
			canvas.Translate(-size.Width, 0)
			canvas.DrawArc(arcBase.And(arcSize), startSweep, endSweep, false, background)
			canvas.Restore()
		} else {
			canvas.DrawArc(arcBase.And(arcSize), 0, circularSweep, false, background)
		}
	}
	switch {
	case p.year2023 && p.value.IsNone() && p.strokeCap.IsNone():
		paint.SetStrokeCap(ui.StrokeCapSquare)
	case p.year2023:
		paint.SetStrokeCap(p.strokeCap.OrElse(ui.StrokeCapButt))
	default:
		paint.SetStrokeCap(p.strokeCap.OrElse(ui.StrokeCapRound))
	}
	canvas.DrawArc(arcBase.And(arcSize), p.arcStart, p.arcSweep, false, paint)
}

func (p *circularProgressIndicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*circularProgressIndicatorPainter)
	return !ok || o.circularPainterOpts != p.circularPainterOpts
}

// RefreshProgressIndicatorOpts are the optional parameters of
// NewRefreshProgressIndicator.
//
// Flutter: material/progress_indicator.dart RefreshProgressIndicator
type RefreshProgressIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Value is the progress between 0 and 1; none is an indeterminate indicator.
	Value foundation.Opt[float64]
	// BackgroundColor is the color of the disc behind the arc.
	BackgroundColor foundation.Opt[ui.Color]
	// Color is the color of the arc.
	Color foundation.Opt[ui.Color]
	// ValueColor is an animated color of the arc; it wins over Color.
	ValueColor animation.Animation[foundation.Opt[ui.Color]]
	// StrokeWidth is the width of the line; the default is
	// RefreshProgressIndicatorDefaultStrokeWidth.
	StrokeWidth foundation.Opt[float64]
	// StrokeAlign is where the stroke is placed.
	StrokeAlign foundation.Opt[float64]
	// SemanticsLabel is the label of the semantics.
	SemanticsLabel string
	// SemanticsValue is the value of the semantics.
	SemanticsValue string
	// StrokeCap is the shape of the ends of the line.
	StrokeCap foundation.Opt[ui.StrokeCap]
	// Elevation is the elevation of the disc; the default is 2.
	Elevation foundation.Opt[float64]
	// IndicatorMargin is the empty space around the disc; the default is 4.
	IndicatorMargin painting.EdgeInsetsGeometry
	// IndicatorPadding is the padding between the disc and the arc; the default
	// is 12.
	IndicatorPadding painting.EdgeInsetsGeometry
}

// RefreshProgressIndicator is the circular indicator of a RefreshIndicator: an arc
// with an arrow head on a disc.
//
// Flutter: material/progress_indicator.dart RefreshProgressIndicator
type RefreshProgressIndicator struct {
	ProgressIndicator
	o RefreshProgressIndicatorOpts
}

// NewRefreshProgressIndicator creates the indicator.
//
// Flutter: material/progress_indicator.dart RefreshProgressIndicator.new
func NewRefreshProgressIndicator(opts ...RefreshProgressIndicatorOpts) *RefreshProgressIndicator {
	o := firstOpts(opts)
	return &RefreshProgressIndicator{
		ProgressIndicator: ProgressIndicator{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: o.Value,
			backgroundColor: o.BackgroundColor, color: o.Color, valueColor: o.ValueColor, semanticsLabel: o.SemanticsLabel, semanticsValue: o.SemanticsValue},
		o: o,
	}
}

// CreateState creates the state.
//
// Flutter: material/progress_indicator.dart RefreshProgressIndicator.createState
func (w *RefreshProgressIndicator) CreateState() widgets.State {
	return &refreshProgressIndicatorState{}
}

const (
	refreshIndicatorSize  = 41.0
	refreshStrokeHeadSpan = 0.33
)

var (
	refreshConvertTween            = animation.NewCurveTween(animation.NewInterval(0.1, refreshStrokeHeadSpan))
	refreshArrowheadInterval       = animation.NewInterval(0.1, refreshStrokeHeadSpan)
	refreshAdditionalRotationTween = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
		{Tween: animation.NewTween(-0.1, -0.2), Weight: refreshStrokeHeadSpan},
		{Tween: animation.NewTween(-0.2, 1.35), Weight: 1 - refreshStrokeHeadSpan},
	})
)

type refreshProgressIndicatorState struct {
	widgets.StateOf[*RefreshProgressIndicator]
	widgets.SingleTickerProvider
	controllers progressControllers
	lastValue   foundation.Opt[float64]
}

func (s *refreshProgressIndicatorState) InitState() {
	s.controllers.init(s, CircularProgressIndicatorDefaultAnimationDuration)
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *refreshProgressIndicatorState) DidUpdateWidget(*RefreshProgressIndicator) {
	s.controllers.update(s.Widget().effectiveValue())
}

func (s *refreshProgressIndicatorState) Dispose() { s.controllers.internal.Dispose() }

func (s *refreshProgressIndicatorState) Build(bc widgets.BuildContext) widgets.Widget {
	controller := s.controllers.controller(bc, nil)
	if v, ok := s.Widget().effectiveValue().Get(); ok {
		s.lastValue = foundation.Some(v)
		controller.SetValue(refreshConvertTween.Transform(v) * (1333 / 2.0 / indeterminateCircularDuration))
	}
	return widgets.NewAnimatedBuilder(controller, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
		v := controller.Value()
		return s.buildMaterialIndicator(bc, 1.05*strokeHeadTween.Transform(v), strokeTailTween.Transform(v), offsetTween.Transform(v), rotationTween.Transform(v))
	}, nil)
}

func (s *refreshProgressIndicatorState) buildMaterialIndicator(bc widgets.BuildContext, headValue, tailValue, offsetValue, rotationValue float64) widgets.Widget {
	w := s.Widget()
	value := w.effectiveValue()
	arrowheadScale := 0.0
	if v, ok := value.Get(); ok {
		arrowheadScale = refreshArrowheadInterval.Transform(v)
	}
	rotation := 0.0
	if v, ok := firstOpt(value, s.lastValue).Get(); ok {
		rotation = math.Pi * refreshAdditionalRotationTween.Transform(v)
	}
	valueColor := w.getValueColor(bc, foundation.None[ui.Color]())
	opacity := valueColor.Opacity()
	valueColor = valueColor.WithOpacity(1.0)
	defaults := circularDefaults(bc, true, value.IsNone())
	indicatorTheme := ProgressIndicatorThemeOf(bc)
	backgroundColor := firstOpt(w.o.BackgroundColor, indicatorTheme.RefreshBackgroundColor, foundation.Some(ThemeOf(bc).CanvasColor)).Value()
	strokeWidth := firstOpt(w.o.StrokeWidth, indicatorTheme.StrokeWidth, defaults.StrokeWidth).Value()
	if w.o.StrokeWidth.IsNone() && indicatorTheme.StrokeWidth.IsNone() {
		strokeWidth = RefreshProgressIndicatorDefaultStrokeWidth
	}
	strokeAlign := firstOpt(w.o.StrokeAlign, indicatorTheme.StrokeAlign, defaults.StrokeAlign).Value()
	strokeCap := firstOpt(w.o.StrokeCap, indicatorTheme.StrokeCap)
	margin := firstNonNil[painting.EdgeInsetsGeometry](w.o.IndicatorMargin, painting.NewEdgeInsetsAll(4.0))
	padding := firstNonNil[painting.EdgeInsetsGeometry](w.o.IndicatorPadding, painting.NewEdgeInsetsAll(12.0))
	painter := &refreshProgressIndicatorPainter{
		circularProgressIndicatorPainter: newCircularProgressIndicatorPainter(circularPainterOpts{valueColor: valueColor, headValue: headValue, tailValue: tailValue,
			offsetValue: offsetValue, rotationValue: rotationValue, strokeWidth: strokeWidth, strokeAlign: strokeAlign, strokeCap: strokeCap, year2023: true}),
		arrowheadScale: arrowheadScale}
	return w.buildSemanticsWrapper(widgets.NewPadding(margin,
		widgets.NewSizedBoxFromSize(ui.Size{Width: refreshIndicatorSize, Height: refreshIndicatorSize},
			NewMaterial(widgets.NewPadding(padding,
				widgets.NewOpacity(opacity, widgets.NewTransformRotate(rotation, widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: painter})))),
				MaterialOpts{Type: MaterialTypeCircle, Color: foundation.Some(backgroundColor), Elevation: w.o.Elevation.OrElse(2.0)}))))
}

// refreshProgressIndicatorPainter paints the arc and its arrow head.
//
// Flutter: material/progress_indicator.dart _RefreshProgressIndicatorPainter
type refreshProgressIndicatorPainter struct {
	*circularProgressIndicatorPainter
	arrowheadScale float64
}

func (p *refreshProgressIndicatorPainter) paintArrowhead(canvas *ui.Canvas, size ui.Size) {
	arcEnd := p.arcStart + p.arcSweep
	ux, uy := math.Cos(arcEnd), math.Sin(arcEnd)
	radius := size.Width / 2.0
	pointX := radius + ux*radius + -uy*p.strokeWidth*2.0*p.arrowheadScale
	pointY := radius + uy*radius + ux*p.strokeWidth*2.0*p.arrowheadScale
	arrowheadRadius := p.strokeWidth * 2.0 * p.arrowheadScale
	innerRadius, outerRadius := radius-arrowheadRadius, radius+arrowheadRadius
	path := ui.NewPath()
	path.MoveTo(radius+ux*innerRadius, radius+uy*innerRadius)
	path.LineTo(radius+ux*outerRadius, radius+uy*outerRadius)
	path.LineTo(pointX, pointY)
	path.Close()
	paint := ui.NewPaint()
	paint.SetColor(p.valueColor)
	paint.SetStrokeWidth(p.strokeWidth)
	paint.SetStyle(ui.PaintingStyleFill)
	canvas.DrawPath(path, paint)
}

func (p *refreshProgressIndicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.circularProgressIndicatorPainter.Paint(canvas, size)
	if p.arrowheadScale > 0.0 {
		p.paintArrowhead(canvas, size)
	}
}

func (p *refreshProgressIndicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*refreshProgressIndicatorPainter)
	return !ok || o.circularPainterOpts != p.circularPainterOpts || o.arrowheadScale != p.arrowheadScale
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCircularProgressIndicatorValueAndController = "CircularProgressIndicatorValueAndController"
	tagLinearProgressIndicatorMinHeightNotPositive = "LinearProgressIndicatorMinHeightNotPositive"
	tagLinearProgressIndicatorValueAndController   = "LinearProgressIndicatorValueAndController"
)
