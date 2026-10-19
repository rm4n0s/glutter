// Command animations shows four animations in four labelled tiles: an
// AnimatedContainer that changes size, color and corner radius when tapped, a
// Pulse (a ScaleTransition driven by a repeating controller), an AnimatedSwitcher
// that cycles a counter, and a TweenAnimationBuilder that turns a plus sign.
// The app also starts the tapped animations by itself, every two seconds. It
// uses only the public packages.
//
// Run it with: go run ./examples/animations
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>.
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The Material 3 baseline colors that the example paints with.
var (
	primary    = ui.NewColor(0xFF6750A4)
	secondary  = ui.NewColor(0xFF625B71)
	tertiary   = ui.NewColor(0xFF7D5260)
	errorColor = ui.NewColor(0xFFB3261E)
	surface    = ui.NewColor(0xFFFFFBFE)
	tile       = ui.NewColor(0xFFF3EDF7)
	outline    = ui.NewColor(0xFF79747E)
	onSurface  = ui.NewColor(0xFF1C1B1F)
	white      = ui.NewColor(0xFFFFFFFF)
)

func textStyle(size float64, color ui.Color, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color), FontWeight: weight,
	})
}

func label(s string, size float64, color ui.Color, weight ui.FontWeight, key ...foundation.Key) widgets.Widget {
	o := widgets.TextOpts{Style: textStyle(size, color, weight)}
	if len(key) > 0 {
		o.Key = key[0]
	}
	return widgets.NewText(s, o)
}

func box(width, height float64, child widgets.Widget) widgets.Widget {
	return widgets.NewSizedBox(child, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})
}

// card is a tile: a title, the animation in a fixed area, and the name of the
// curve it uses.
func card(title string, curve string, content widgets.Widget) widgets.Widget {
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color:        foundation.Some(tile),
		BorderRadius: painting.NewBorderRadiusCircular(16),
		Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(outline), Width: foundation.Some(1.0)}),
	}), widgets.NewPadding(painting.NewEdgeInsetsAll(12), widgets.NewColumn([]widgets.Widget{
		label(title, 14, primary, ui.FontWeightW700),
		widgets.NewExpanded(widgets.NewCenter(content)),
		label("Curves."+curve, 12, onSurface, ui.FontWeightW400),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})))
}

// Pulse is a ScaleTransition driven by a controller that repeats back and
// forth, so the child grows and shrinks for ever.
type Pulse struct {
	widgets.StatefulWidget
	child widgets.Widget
}

// NewPulse creates the widget.
func NewPulse(child widgets.Widget) *Pulse {
	return &Pulse{StatefulWidget: widgets.NewStatefulWidgetBase(), child: child}
}

// CreateState creates the state.
func (p *Pulse) CreateState() widgets.State { return &pulseState{} }

type pulseState struct {
	widgets.StateOf[*Pulse]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
	curved     *animation.CurvedAnimation
	scale      animation.Animation[float64]
}

func (s *pulseState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(time.Second)})
	s.controller.Repeat(animation.RepeatOpts{Reverse: true})
	s.curved = animation.NewCurvedAnimation(s.controller, animation.Curves.EaseInOut)
	s.scale = animation.NewTween(0.7, 1.2).Animate(s.curved)
}

func (s *pulseState) Dispose() {
	s.curved.Dispose()
	s.controller.Dispose()
}

func (s *pulseState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewScaleTransition(s.scale, s.Widget().child)
}

// animationsApp holds the state that the tiles share: whether the container is
// in its second form, and the step of the counter and of the turning plus sign.
type animationsApp struct {
	widgets.StatefulWidget
	// autoAdvance starts the animations by itself: after 100 ms, then every two
	// seconds. The tests turn it off and drive the state themselves.
	autoAdvance bool
}

func newAnimationsApp(autoAdvance bool) *animationsApp {
	return &animationsApp{StatefulWidget: widgets.NewStatefulWidgetBase(), autoAdvance: autoAdvance}
}

func (w *animationsApp) CreateState() widgets.State { return &animationsState{} }

type animationsState struct {
	widgets.StateOf[*animationsApp]
	expanded bool
	step     int
}

func (s *animationsState) InitState() {
	if !s.Widget().autoAdvance {
		return
	}
	s.Go(func(ctx context.Context) {
		wait := 100 * time.Millisecond
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				s.advance()
				wait = 2 * time.Second
			}
		}
	})
}

// advance moves every animation to its next form.
func (s *animationsState) advance() {
	s.SetState(func() {
		s.expanded = !s.expanded
		s.step++
	})
}

func (s *animationsState) toggleContainer() { s.SetState(func() { s.expanded = !s.expanded }) }

func (s *animationsState) containerTile() widgets.Widget {
	width, height, radius, color := 70.0, 70.0, 8.0, primary
	if s.expanded {
		width, height, radius, color = 150, 90, 40, tertiary
	}
	return card("AnimatedContainer (tap it)", "easeInOutCubic", widgets.NewGestureDetector(
		widgets.NewAnimatedContainer(600*time.Millisecond, nil, widgets.AnimatedContainerOpts{
			Curve: animation.Curves.EaseInOutCubic, Width: foundation.Some(width), Height: foundation.Some(height),
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Color: foundation.Some(color), BorderRadius: painting.NewBorderRadiusCircular(radius),
			}),
		}),
		widgets.GestureDetectorOpts{OnTap: s.toggleContainer, Behavior: rendering.HitTestBehaviorOpaque}))
}

func (s *animationsState) pulseTile() widgets.Widget {
	circle := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(secondary), Shape: painting.BoxShapeCircle,
	}), box(90, 90, widgets.NewCenter(label("Pulse", 14, white, ui.FontWeightW500))))
	return card("Pulse", "easeInOut", NewPulse(circle))
}

func (s *animationsState) switcherTile() widgets.Widget {
	counter := label(strconv.Itoa(s.step), 64, errorColor, ui.FontWeightW700, foundation.NewValueKey(s.step))
	return card("AnimatedSwitcher", "easeOut", widgets.NewAnimatedSwitcher(500*time.Millisecond, counter, widgets.AnimatedSwitcherOpts{
		SwitchInCurve:  animation.Curves.EaseOut,
		SwitchOutCurve: animation.Curves.EaseOut,
		TransitionBuilder: func(child widgets.Widget, a animation.Animation[float64]) widgets.Widget {
			return widgets.NewScaleTransition(a, widgets.NewFadeTransition(a, child))
		},
	}))
}

// plus is an icon-like shape: a plus sign with a dot, so that its turn is seen.
func plus() widgets.Widget {
	bar := func(w, h float64) widgets.Widget {
		return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(primary), BorderRadius: painting.NewBorderRadiusCircular(5),
		}), box(w, h, nil))
	}
	dot := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(errorColor), Shape: painting.BoxShapeCircle,
	}), box(14, 14, nil))
	return box(90, 90, widgets.NewStack([]widgets.Widget{
		widgets.NewCenter(bar(90, 22)),
		widgets.NewCenter(bar(22, 90)),
		widgets.NewPositioned(dot, widgets.PositionedOpts{Top: foundation.Some(0.0), Right: foundation.Some(0.0)}),
	}))
}

func (s *animationsState) turnTile() widgets.Widget {
	return card("TweenAnimationBuilder", "easeInOutBack", widgets.NewTweenAnimationBuilder(
		animation.NewTweenEnd(float64(s.step)*0.25), 800*time.Millisecond,
		func(_ widgets.BuildContext, turns float64, child widgets.Widget) widgets.Widget {
			return widgets.NewTransform(ui.NewMatrix4RotationZ(turns*2*math.Pi), child, widgets.TransformOpts{Alignment: painting.AlignmentCenter})
		}, plus(), widgets.TweenAnimationBuilderOpts[float64]{Curve: animation.Curves.EaseInOutBack}))
}

func (s *animationsState) Build(widgets.BuildContext) widgets.Widget {
	gap := box(12, 12, nil)
	row := func(a, b widgets.Widget) widgets.Widget {
		return widgets.NewExpanded(widgets.NewRow([]widgets.Widget{widgets.NewExpanded(a), gap, widgets.NewExpanded(b)},
			widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))
	}
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColoredBox(surface, widgets.NewPadding(painting.NewEdgeInsetsAll(16),
		widgets.NewColumn([]widgets.Widget{
			row(s.containerTile(), s.pulseTile()),
			gap,
			row(s.switcherTile(), s.turnTile()),
		}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))))
}

func main() {
	err := widgets.RunApp(newAnimationsApp(true), widgets.RunAppOpts{
		AppID:  "dev.glutter.animations",
		Window: widgets.WindowConfig{Title: "Glutter – animations", Size: ui.Size{Width: 640, Height: 480}},
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
