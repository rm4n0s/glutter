// Command custompaint shows CustomPaint with a custom painter: an analog clock
// with a gradient face, a shadow, minute and hour ticks and three hands. A
// StatefulWidget gets the time from a goroutine once a second and calls
// SetState; the painter repaints only when ShouldRepaint says that the second
// changed. It uses only the public packages.
//
// Run it with: go run ./examples/custompaint
//
// On Windows: go run ./cmd/glutter-windows run ./examples/custompaint (under Wine on Linux, headless with
// -headless; GLUTTER_BACKEND is win32 there). Nothing of this example is missing on Windows.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_SCREENSHOT=<file.png>.
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	background = ui.NewColor(0xFFFFFBFE)
	faceInner  = ui.NewColor(0xFFFFFFFF)
	faceOuter  = ui.NewColor(0xFFE8DEF8)
	rim        = ui.NewColor(0xFF6750A4)
	ink        = ui.NewColor(0xFF1C1B1F)
	accent     = ui.NewColor(0xFFB3261E)
)

// clockPainter draws the clock of a time on a square box.
type clockPainter struct {
	rendering.CustomPainterBase
	time    time.Time
	onPaint func()
}

func newClockPainter(t time.Time, onPaint func()) *clockPainter {
	return &clockPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), time: t, onPaint: onPaint}
}

// Paint draws the shadow, the face, the ticks and the hands.
func (p *clockPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	if p.onPaint != nil {
		p.onPaint()
	}
	center := ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2}
	radius := math.Min(size.Width, size.Height)/2 - 14

	// The shadow of the face.
	disc := ui.NewPath()
	disc.AddOval(ui.NewRectFromCircle(center, radius))
	canvas.DrawShadow(disc, ui.NewColor(0xFF000000), 8, false)

	// The face: a radial gradient and a rim.
	face := ui.NewPaint()
	face.SetIsAntiAlias(true)
	face.SetShader(ui.NewGradientRadial(center.Sub(ui.Offset{Dx: radius * 0.25, Dy: radius * 0.25}), radius*1.3, []ui.Color{faceInner, faceOuter}))
	canvas.DrawCircle(center, radius, face)
	stroke := ui.NewPaint()
	stroke.SetIsAntiAlias(true)
	stroke.SetStyle(ui.PaintingStyleStroke)
	stroke.SetStrokeWidth(4)
	stroke.SetColor(rim)
	canvas.DrawCircle(center, radius, stroke)

	// The ticks: 60 of them, longer and thicker every five.
	canvas.Save()
	canvas.Translate(center.Dx, center.Dy)
	tick := ui.NewPaint()
	tick.SetIsAntiAlias(true)
	tick.SetStyle(ui.PaintingStyleStroke)
	tick.SetStrokeCap(ui.StrokeCapRound)
	tick.SetColor(ink)
	for i := range 60 {
		length, width := radius*0.05, 1.0
		if i%5 == 0 {
			length, width = radius*0.12, 3.0
		}
		tick.SetStrokeWidth(width)
		canvas.DrawLine(ui.Offset{Dy: -(radius - 8)}, ui.Offset{Dy: -(radius - 8 - length)}, tick)
		canvas.Rotate(2 * math.Pi / 60)
	}
	canvas.Restore()

	// The hands, from the angles of the time.
	h, m, s := p.time.Hour()%12, p.time.Minute(), float64(p.time.Second())
	minute := (float64(m) + s/60) / 60
	hour := (float64(h) + minute) / 12
	hand := func(fraction, length, width float64, color ui.Color) {
		paint := ui.NewPaint()
		paint.SetIsAntiAlias(true)
		paint.SetStyle(ui.PaintingStyleStroke)
		paint.SetStrokeCap(ui.StrokeCapRound)
		paint.SetStrokeWidth(width)
		paint.SetColor(color)
		angle := fraction * 2 * math.Pi
		tip := ui.Offset{Dx: center.Dx + math.Sin(angle)*radius*length, Dy: center.Dy - math.Cos(angle)*radius*length}
		tail := ui.Offset{Dx: center.Dx - math.Sin(angle)*radius*0.12, Dy: center.Dy + math.Cos(angle)*radius*0.12}
		canvas.DrawLine(tail, tip, paint)
	}
	hand(hour, 0.5, 7, ink)
	hand(minute, 0.75, 5, ink)
	hand(s/60, 0.85, 2, accent)
	cap := ui.NewPaint()
	cap.SetIsAntiAlias(true)
	cap.SetColor(accent)
	canvas.DrawCircle(center, 7, cap)
}

// ShouldRepaint is true when the new painter shows another second.
func (p *clockPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*clockPainter)
	return !ok || o.time.Unix() != p.time.Unix()
}

// HitTest leaves the default.
func (p *clockPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }

// clockApp shows the clock and updates it every second. The clock function
// gives the time: the app passes time.Now, a test passes a fixed time.
type clockApp struct {
	widgets.StatefulWidget
	now     func() time.Time
	onPaint func() // called by the painter each time it paints; tests count with it
}

func newClockApp(now func() time.Time) *clockApp {
	return &clockApp{StatefulWidget: widgets.NewStatefulWidgetBase(), now: now}
}

func (w *clockApp) CreateState() widgets.State { return &clockState{} }

type clockState struct {
	widgets.StateOf[*clockApp]
	time time.Time
}

func (s *clockState) InitState() {
	now := s.Widget().now
	s.time = now()
	// The goroutine hands the time over with SetState, which is safe to call from
	// any goroutine; it stops when the state is disposed. It keeps the clock
	// function that the widget had when the state was made.
	s.Go(func(ctx context.Context) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t := now()
				s.SetState(func() { s.time = t })
			}
		}
	})
}

func (s *clockState) Build(widgets.BuildContext) widgets.Widget {
	caption := widgets.NewText(s.time.Format("15:04:05"), widgets.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(20.0), Color: foundation.Some(ink), FontWeight: ui.FontWeightW500,
	})})
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColoredBox(background, widgets.NewColumn([]widgets.Widget{
		widgets.NewExpanded(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: newClockPainter(s.time, s.Widget().onPaint)})),
		widgets.NewPadding(painting.NewEdgeInsetsSymmetric(14, 0), widgets.NewCenter(caption)),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})))
}

func main() {
	err := widgets.RunApp(newClockApp(time.Now), widgets.RunAppOpts{
		AppID:  "dev.glutter.custompaint",
		Window: widgets.WindowConfig{Title: "Glutter – custom paint", Size: ui.Size{Width: 420, Height: 480}},
	})
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
