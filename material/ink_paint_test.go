package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestInkPaintsOnTheMaterial(t *testing.T) {
	tests := []struct {
		name   string
		ink    func(child widgets.Widget) widgets.Widget
		center ui.Color
		corner bool // whether the corner of the box is painted too
	}{
		{"color", func(c widgets.Widget) widgets.Widget {
			return NewInk(c, InkOpts{Color: foundation.Some(blue), Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
		}, blue, true},
		{"decoration with a shape", func(c widgets.Widget) widgets.Widget {
			return NewInk(c, InkOpts{Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(blue), Shape: painting.BoxShapeCircle}),
				Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
		}, blue, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(box(NewMaterial(tt.ink(widgets.NewSizedBox(nil)), MaterialOpts{Color: foundation.Some(white)})))
			if got := pixel(ts, 400, 300); got != rgba(tt.center) {
				t.Errorf("center = %v, want %v", got, rgba(tt.center))
			}
			if got := pixel(ts, 352, 252) == rgba(tt.center); got != tt.corner {
				t.Errorf("corner painted = %v, want %v", got, tt.corner)
			}
			if got := pixel(ts, 100, 100); got != rgba(white) {
				t.Errorf("outside = %v", got)
			}
		})
	}
}

func TestInkIsPaintedBelowTheSplash(t *testing.T) {
	// The ink is a feature of the material, and the splash is too: the splash is
	// added later, so it is painted over the ink and the child is above both.
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(m2, NewInk(square(NewInkWell(nil, InkWellOpts{OnTap: func() {}, SplashColor: foundation.Some(ui.NewColor(0x80FF0000)), HighlightColor: foundation.Some(ui.NewColor(0x00000000))})),
		InkOpts{Color: foundation.Some(blue)})))
	if got := pixel(ts, 400, 300); got != rgba(blue) {
		t.Fatalf("ink = %v", got)
	}
	ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(150 * time.Millisecond)
	ts.Pump(150 * time.Millisecond)
	got := pixel(ts, 400, 300)
	if got == rgba(blue) || got.R == 0 || got.B == 0 {
		t.Errorf("splash over the ink = %v, expected a mix of red and blue", got)
	}
}

func TestInkUpdatesItsDecoration(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	build := func(c ui.Color, padded bool) widgets.Widget {
		o := InkOpts{Color: foundation.Some(c), Width: foundation.Some(100.0), Height: foundation.Some(100.0)}
		if padded {
			o.Padding = painting.NewEdgeInsetsAll(10)
		}
		return box(NewMaterial(NewInk(widgets.NewSizedBox(nil), o), MaterialOpts{Color: foundation.Some(white)}))
	}
	ts.PumpWidget(build(blue, false))
	ts.PumpWidget(build(red, false))
	if got := pixel(ts, 400, 300); got != rgba(red) {
		t.Errorf("after the update = %v, want red", got)
	}
	ts.PumpWidget(build(red, true))
	if got := pixel(ts, 400, 300); got != rgba(red) {
		t.Errorf("after the padding = %v, want red", got)
	}
	// Removing the Ink takes its decoration off the material.
	ts.PumpWidget(box(NewMaterial(widgets.NewSizedBox(nil), MaterialOpts{Color: foundation.Some(white)})))
	if got := pixel(ts, 400, 300); got != rgba(white) {
		t.Errorf("after removing the ink = %v, want white", got)
	}
}

func TestInkPanics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"color and decoration", func() {
			NewInk(nil, InkOpts{Color: foundation.Some(red), Decoration: painting.NewBoxDecoration()})
		}, "NewInk.InkColorAndDecoration"},
		{"negative padding", func() { NewInk(nil, InkOpts{Padding: painting.NewEdgeInsetsAll(-1)}) }, "NewInk.InkPaddingNegative"},
		{"negative padding of an image", func() { NewInkImage(nil, nil, InkImageOpts{Padding: painting.NewEdgeInsetsAll(-1)}) }, "NewInkImage.InkPaddingNegative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.build), tt.route)
		})
	}
}

func TestInkResponseNotContained(t *testing.T) {
	// A response that is not contained paints its splash beyond its box, in a circle.
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(m2, widgets.NewSizedBox(NewInkResponse(nil, InkResponseOpts{
		OnTap: func() {}, SplashColor: foundation.Some(ui.NewColor(0xFFFF0000)), HighlightColor: foundation.Some(ui.NewColor(0x00000000)),
		Radius: foundation.Some(60.0),
	}), widgets.SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(20.0)})))
	base := pixel(ts, 400, 300)
	g := ts.StartGesture(ui.Offset{Dx: 400, Dy: 300})
	ts.Pump(150 * time.Millisecond)
	ts.Pump(700 * time.Millisecond)
	if got := pixel(ts, 400, 300); got == base {
		t.Error("the splash was not painted at the center")
	}
	if got := pixel(ts, 400, 340); got == base {
		t.Errorf("the splash did not reach beyond the response: %v", got)
	}
	g.Up()
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 340); got != base {
		t.Errorf("the splash did not go away: %v", got)
	}
}

func TestInkWellRoundedClip(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	tests := []struct {
		name string
		o    InkWellOpts
	}{
		{"border radius", InkWellOpts{BorderRadius: foundation.Some(painting.NewBorderRadiusCircular(50))}},
		{"custom border", InkWellOpts{CustomBorder: painting.NewCircleBorder()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.o
			o.OnTap, o.HoverColor = func() {}, foundation.Some(green)
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, o))))
			base := pixel(ts, 355, 255)
			ts.Hover(ui.Offset{Dx: 400, Dy: 300})
			ts.PumpAndSettle()
			if got := pixel(ts, 400, 300); got != rgba(green) {
				t.Errorf("center = %v, want green", got)
			}
			if got := pixel(ts, 355, 255); got != base {
				t.Errorf("clipped corner = %v, want %v", got, base)
			}
		})
	}
}

func TestInkWellHighlightShapeOfInkResponse(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, square(NewInkResponse(nil, InkResponseOpts{OnTap: func() {}, HoverColor: foundation.Some(green), HighlightShape: painting.BoxShapeCircle}))))
	base := pixel(ts, 355, 255)
	ts.Hover(ui.Offset{Dx: 400, Dy: 300})
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != rgba(green) {
		t.Errorf("center = %v", got)
	}
	if got := pixel(ts, 355, 255); got != base {
		t.Errorf("corner of a circle highlight = %v, want %v", got, base)
	}
}

func TestInkResponseRebuildUpdatesHighlightShape(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	build := func(shape painting.BoxShape) widgets.Widget {
		return inkApp(nil, square(NewInkResponse(nil, InkResponseOpts{OnTap: func() {}, HoverColor: foundation.Some(green), HighlightShape: shape})))
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(build(painting.BoxShapeCircle))
	ts.Hover(ui.Offset{Dx: 400, Dy: 300})
	ts.PumpAndSettle()
	if got := pixel(ts, 355, 255); got == rgba(green) {
		t.Fatal("the corner of a circle highlight is painted")
	}
	ts.PumpWidget(build(painting.BoxShapeRectangle))
	ts.PumpAndSettle()
	if got := pixel(ts, 355, 255); got != rgba(green) {
		t.Errorf("corner after the shape became a rectangle = %v, want green", got)
	}
}

func TestInkWellDisabledHoverTakesHighlightDown(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	build := func(enabled bool) widgets.Widget {
		var tap func()
		if enabled {
			tap = func() {}
		}
		return inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnTap: tap, HoverColor: foundation.Some(green)})))
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(build(true))
	base := pixel(ts, 400, 300)
	ts.Hover(ui.Offset{Dx: 400, Dy: 300})
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != rgba(green) {
		t.Fatalf("hovered = %v", got)
	}
	ts.PumpWidget(build(false))
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != base {
		t.Errorf("hover highlight stayed after the ink well was disabled: %v", got)
	}
	ts.Hover(ui.Offset{Dx: 10, Dy: 10})
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestInkHoverDurationFades(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnTap: func() {}, HoverColor: foundation.Some(green), HoverDuration: foundation.Some(400 * time.Millisecond)}))))
	base := pixel(ts, 400, 300)
	ts.Hover(ui.Offset{Dx: 400, Dy: 300})
	ts.Pump()
	ts.Pump(200 * time.Millisecond)
	mid := pixel(ts, 400, 300)
	if mid == base || mid == rgba(green) {
		t.Errorf("mid fade = %v, should be between %v and green", mid, base)
	}
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != rgba(green) {
		t.Errorf("end = %v", got)
	}
}
