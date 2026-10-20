// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/banner_test.dart. The test canvas is
// a recorder: its transforms stand in for the translate and rotate calls, and
// the memory events case is not ported.

type bannerOps struct {
	tx, ty, angle float64
	rects         []ui.Rect
	paints        []*ui.Paint
	paragraphs    []ui.Offset
}

// recordBanner paints the painter on a recording canvas of the size.
func recordBanner(painter *BannerPainter, size ui.Size) bannerOps {
	recorder := ui.NewPictureRecorder()
	painter.Paint(ui.NewCanvas(recorder), size)
	var out bannerOps
	total := *ui.NewMatrix4Identity()
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		switch op := op.(type) {
		case ui.OpTransform:
			// A translation by zero is not recorded, so the transforms are multiplied.
			m := op.Matrix
			total = *total.Multiply(&m)
		case ui.OpDrawRect:
			out.rects = append(out.rects, op.Rect)
			paint := op.Paint
			out.paints = append(out.paints, &paint)
		case ui.OpDrawParagraph:
			out.paragraphs = append(out.paragraphs, op.Offset)
		}
	}
	out.tx, out.ty = float64(total.Entry(0, 3)), float64(total.Entry(1, 3))
	out.angle = math.Atan2(float64(total.Entry(1, 0)), float64(total.Entry(0, 0)))
	return out
}

func TestBannerPainterLocations(t *testing.T) {
	ts := gluttertest.New(t)
	engine := ts.Dispatcher().TextEngine()
	tests := []struct {
		name     string
		location BannerLocation
		layout   ui.TextDirection
		wantX    string // "low" is less than 100, "high" is more than 900
		wantY    string
		wantTurn float64 // the angle, in quarters of a turn of pi
	}{
		{"topStart LTR is in the top left", BannerLocationTopStart, ui.TextDirectionLTR, "low", "low", -1},
		{"topStart RTL is in the top right", BannerLocationTopStart, ui.TextDirectionRTL, "high", "low", 1},
		{"topEnd LTR is in the top right", BannerLocationTopEnd, ui.TextDirectionLTR, "high", "low", 1},
		{"topEnd RTL is in the top left", BannerLocationTopEnd, ui.TextDirectionRTL, "low", "low", -1},
		{"bottomStart LTR is in the bottom left", BannerLocationBottomStart, ui.TextDirectionLTR, "low", "high", 1},
		{"bottomStart RTL is in the bottom right", BannerLocationBottomStart, ui.TextDirectionRTL, "high", "high", -1},
		{"bottomEnd LTR is in the bottom right", BannerLocationBottomEnd, ui.TextDirectionLTR, "high", "high", -1},
		{"bottomEnd RTL is in the bottom left", BannerLocationBottomEnd, ui.TextDirectionRTL, "low", "high", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The text direction is the other one, to see that it does not change the layout.
			textDirection := ui.TextDirectionRTL
			if tt.layout == ui.TextDirectionRTL {
				textDirection = ui.TextDirectionLTR
			}
			painter := NewBannerPainter("foo", textDirection, tt.location, tt.layout, BannerPainterOpts{TextEngine: engine})
			ops := recordBanner(painter, ui.Size{Width: 1000, Height: 1000})
			for axis, got := range map[string][2]any{"x": {tt.wantX, ops.tx}, "y": {tt.wantY, ops.ty}} {
				want, v := got[0].(string), got[1].(float64)
				if (want == "low" && v >= 100) || (want == "high" && v <= 900) {
					t.Errorf("translate %s = %v, want %s", axis, v, want)
				}
			}
			if want := tt.wantTurn * math.Pi / 4; math.Abs(ops.angle-want) > 1e-6 {
				t.Errorf("angle = %v, want %v", ops.angle, want)
			}
			painter.Dispose()
		})
	}
}

func TestBannerPainterDraws(t *testing.T) {
	ts := gluttertest.New(t)
	painter := NewBannerPainter("foo", ui.TextDirectionLTR, BannerLocationTopEnd, ui.TextDirectionLTR,
		BannerPainterOpts{TextEngine: ts.Dispatcher().TextEngine()})
	defer painter.Dispose()
	ops := recordBanner(painter, ui.Size{Width: 800, Height: 600})
	wantRect := ui.NewRectFromLTRB(-40, 28, 40, 40)
	if len(ops.rects) != 2 || ops.rects[0] != wantRect || ops.rects[1] != wantRect {
		t.Fatalf("rects = %v", ops.rects)
	}
	if ops.paints[0].MaskFilter() == nil || ops.paints[0].Color() != ui.NewColor(0x7F000000) {
		t.Errorf("the shadow paint is %v with the mask filter %v", ops.paints[0].Color(), ops.paints[0].MaskFilter())
	}
	if ops.paints[1].MaskFilter() != nil || ops.paints[1].Color() != ui.NewColor(0xA0B71C1C) {
		t.Errorf("the banner paint is %v with the mask filter %v", ops.paints[1].Color(), ops.paints[1].MaskFilter())
	}
	if ops.tx != 800 || ops.ty != 0 || math.Abs(ops.angle-math.Pi/4) > 1e-6 {
		t.Errorf("translate (%v, %v), angle %v", ops.tx, ops.ty, ops.angle)
	}
	if len(ops.paragraphs) != 1 || ops.paragraphs[0] != (ui.Offset{Dx: -40, Dy: 29}) {
		t.Errorf("paragraphs = %v", ops.paragraphs)
	}
}

func TestBannerPainterShouldRepaint(t *testing.T) {
	base := func(opts ...func(*BannerPainterOpts)) *BannerPainter {
		o := BannerPainterOpts{}
		for _, f := range opts {
			f(&o)
		}
		return NewBannerPainter("foo", ui.TextDirectionLTR, BannerLocationTopEnd, ui.TextDirectionLTR, o)
	}
	a := base()
	tests := []struct {
		name  string
		other *BannerPainter
		want  bool
	}{
		{"the same", base(), false},
		{"another message", NewBannerPainter("bar", ui.TextDirectionLTR, BannerLocationTopEnd, ui.TextDirectionLTR), true},
		{"another location", NewBannerPainter("foo", ui.TextDirectionLTR, BannerLocationTopStart, ui.TextDirectionLTR), true},
		{"another color", base(func(o *BannerPainterOpts) { o.Color = foundation.Some(ui.NewColor(0xFF00FF00)) }), true},
		{"another text style", base(func(o *BannerPainterOpts) {
			o.TextStyle = painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(30.0)})
		}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.other.ShouldRepaint(a); got != tt.want {
				t.Errorf("ShouldRepaint = %v, want %v", got, tt.want)
			}
		})
	}
	if hit := a.HitTest(ui.Offset{}); hit.Value() {
		t.Error("the banner takes no hits")
	}
}

func TestBannerWidget(t *testing.T) {
	t.Run("paints in the top end corner of the view", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewBanner("Hello", BannerLocationTopEnd, nil)))
		painter := ts.Widget(gluttertest.ByType[*CustomPaint]()).(*CustomPaint).ForegroundPainter().(*BannerPainter)
		ops := recordBanner(painter, ui.Size{Width: 800, Height: 600})
		if ops.tx != 800 || ops.ty != 0 || len(ops.rects) != 2 {
			t.Errorf("translate (%v, %v), rects %v", ops.tx, ops.ty, ops.rects)
		}
	})

	t.Run("the shadow can be configured", func(t *testing.T) {
		ts := gluttertest.New(t)
		shadow := painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ui.NewColor(0xFF008000)), BlurRadius: 8.0})
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR,
			NewBanner("Shadow banner", BannerLocationTopEnd, nil, BannerOpts{Shadow: foundation.Some(shadow)})))
		painter := ts.Widget(gluttertest.ByType[*CustomPaint]()).(*CustomPaint).ForegroundPainter().(*BannerPainter)
		ops := recordBanner(painter, ui.Size{Width: 800, Height: 600})
		if ops.paints[0].Color() != ui.NewColor(0xFF008000) {
			t.Errorf("the shadow color is %v", ops.paints[0].Color())
		}
	})

	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBoxShrink(
			NewBanner("X", BannerLocationBottomEnd, nil, BannerOpts{TextDirection: ui.TextDirectionLTR, LayoutDirection: ui.TextDirectionLTR})))))
		if size := ts.Size(gluttertest.ByType[*Banner]()); size != (ui.Size{}) {
			t.Errorf("size = %v, want zero", size)
		}
	})

	t.Run("without a Directionality it is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(NewBanner("X", BannerLocationTopEnd, nil))
		err := ts.TakeError()
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("DirectionalityMissing") {
			t.Errorf("unexpected error %v", err)
		}
		ts.TakeErrors()
	})

	t.Run("it has the location names of Flutter", func(t *testing.T) {
		for l, want := range map[BannerLocation]string{
			BannerLocationTopStart: "topStart", BannerLocationTopEnd: "topEnd", BannerLocationBottomStart: "bottomStart", BannerLocationBottomEnd: "bottomEnd",
		} {
			if l.Name() != want {
				t.Errorf("name = %q, want %q", l.Name(), want)
			}
		}
	})
}

func TestCheckedModeBanner(t *testing.T) {
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBoxShrink(NewCheckedModeBanner(NewText("X"))))))
		if size := ts.Size(gluttertest.ByType[*CheckedModeBanner]()); size != (ui.Size{}) {
			t.Errorf("size = %v, want zero", size)
		}
	})

	t.Run("shows DEBUG in a debug build", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCheckedModeBanner(NewText("X"))))
		banner := ts.Widget(gluttertest.ByType[*Banner]()).(*Banner)
		if banner.Message() != "DEBUG" || banner.Location() != BannerLocationTopEnd {
			t.Errorf("banner %q at %v", banner.Message(), banner.Location())
		}
	})

	t.Run("a WidgetsApp has it", func(t *testing.T) {
		skipUnlessDebug(t)
		tests := []struct {
			name string
			show foundation.Opt[bool]
			want int
		}{
			{"by default", foundation.None[bool](), 1},
			{"turned on", foundation.Some(true), 1},
			{"turned off", foundation.Some(false), 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				ts.PumpWidget(NewWidgetsApp(ui.NewColor(0xFF123456), WidgetsAppOpts{Home: NewPlaceholder(nil), DebugShowCheckedModeBanner: tt.show,
					PageRouteBuilder: defaultTestPageRoute}))
				if got := ts.Count(gluttertest.ByType[*Banner]()); got != tt.want {
					t.Errorf("%d banners, want %d", got, tt.want)
				}
				found := 0
				for _, w := range ts.Widgets(gluttertest.ByType[*CustomPaint]()) {
					if _, ok := w.(*CustomPaint).ForegroundPainter().(*BannerPainter); ok {
						found++
					}
				}
				if found != tt.want {
					t.Errorf("%d CustomPaints have the banner painter, want %d", found, tt.want)
				}
			})
		}
	})
}
