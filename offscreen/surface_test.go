package offscreen_test

import (
	"bytes"
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	viewW = 480
	viewH = 360
)

// damageKind is what a change must damage.
type damageKind int

const (
	damageAny  damageKind = iota // some rectangles; the tracker may merge them into the whole surface
	damagePart                   // some of the surface, not all
	damageAll                    // the whole surface, as one rectangle
)

func fieldBody(controller *widgets.TextEditingController) widgets.Widget {
	return scaffoldApp(widgets.NewColumn([]widgets.Widget{
		widgets.NewRepaintBoundary(widgets.NewSizedBox(material.NewTextField(material.TextFieldOpts{Controller: controller}),
			widgets.SizedBoxOpts{Height: foundation.Some(64.0)})),
		widgets.NewExpanded(widgets.NewRepaintBoundary(textBody())),
	}))
}

func TestSurfaceDrawMatchesAWholeRedraw(t *testing.T) {
	var controller *widgets.TextEditingController
	field := gluttertest.ByType[*material.TextField]()
	color := func(c uint32) widgets.Widget {
		return scaffoldApp(widgets.NewStack([]widgets.Widget{
			widgets.NewListView([]widgets.Widget{widgets.NewText("under the opacity layer"), widgets.NewText("more text below")}),
			widgets.NewPositioned(widgets.NewOpacity(0.5, widgets.NewColoredBox(ui.NewColor(c), widgets.NewSizedBoxExpand(nil))),
				widgets.PositionedOpts{Left: foundation.Some(40.0), Top: foundation.Some(30.0), Width: foundation.Some(200.0), Height: foundation.Some(120.0)}),
		}))
	}
	blur := func(c uint32) widgets.Widget {
		return scaffoldApp(widgets.NewStack([]widgets.Widget{
			widgets.NewColoredBox(ui.NewColor(c), widgets.NewSizedBoxExpand(nil)),
			widgets.NewPositioned(widgets.NewBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 3, SigmaY: 3}), widgets.NewSizedBoxExpand(nil)),
				widgets.PositionedOpts{Left: foundation.Some(20.0), Top: foundation.Some(20.0), Width: foundation.Some(150.0), Height: foundation.Some(100.0)}),
		}))
	}
	tests := []struct {
		name  string
		build func() widgets.Widget
		// setup runs before the first frame is drawn, change after it.
		setup  func(t *testing.T, ts *gluttertest.Tester)
		change func(t *testing.T, ts *gluttertest.Tester)
		want   damageKind
	}{
		{
			name: "text typed into a field",
			build: func() widgets.Widget {
				controller = widgets.NewTextEditingController("abc")
				return fieldBody(controller)
			},
			setup: func(t *testing.T, ts *gluttertest.Tester) {
				ts.Tap(field)
				for i := 0; i < 3; i++ {
					ts.Pump(300 * time.Millisecond)
				}
			},
			change: func(t *testing.T, ts *gluttertest.Tester) { ts.EnterText(field, "abc and some more") },
			want:   damageAny,
		},
		{
			name: "a caret that moved",
			build: func() widgets.Widget {
				controller = widgets.NewTextEditingController("hello caret")
				return fieldBody(controller)
			},
			setup: func(t *testing.T, ts *gluttertest.Tester) {
				ts.Tap(field)
				for i := 0; i < 3; i++ {
					ts.Pump(300 * time.Millisecond)
				}
			},
			change: func(t *testing.T, ts *gluttertest.Tester) {
				ts.TapAt(ui.Offset{Dx: ts.TopLeft(field).Dx + 30, Dy: ts.Center(field).Dy})
				ts.Pump()
			},
			want: damageAny,
		},
		{
			name: "a caret that blinks",
			build: func() widgets.Widget {
				controller = widgets.NewTextEditingController("hello caret")
				return fieldBody(controller)
			},
			setup: func(t *testing.T, ts *gluttertest.Tester) {
				ts.Tap(field)
				for i := 0; i < 3; i++ {
					ts.Pump(300 * time.Millisecond)
				}
			},
			change: func(t *testing.T, ts *gluttertest.Tester) { ts.Pump(500 * time.Millisecond) },
			want:   damagePart,
		},
		{
			name:  "a list that scrolled",
			build: func() widgets.Widget { return scaffoldApp(widgets.NewRepaintBoundary(textBody())) },
			change: func(t *testing.T, ts *gluttertest.Tester) {
				ts.Drag(gluttertest.ByType[*widgets.ListView](), ui.Offset{Dy: -120})
				ts.PumpAndSettle()
			},
			want: damagePart,
		},
		{
			name:   "a color under an opacity layer",
			build:  func() widgets.Widget { return color(0xFFD32F2F) },
			change: func(t *testing.T, ts *gluttertest.Tester) { ts.PumpWidget(color(0xFF1976D2)) },
			want:   damagePart,
		},
		{
			name:   "a backdrop blur",
			build:  func() widgets.Widget { return blur(0xFF388E3C) },
			change: func(t *testing.T, ts *gluttertest.Tester) { ts.PumpWidget(blur(0xFFF57C00)) },
			want:   damageAll,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: viewW, Height: viewH}})
			ts.PumpWidget(tt.build())
			ts.Pump(50 * time.Millisecond)
			if tt.setup != nil {
				tt.setup(t, ts)
			}
			det := offscreen.NewDetacher()
			surf := offscreen.NewSurface(viewW, viewH)

			first := lastScene(ts)
			damage, err := surf.Draw(det.Detach(first))
			if err != nil {
				t.Fatal(err)
			}
			if len(damage) != 1 || damage[0] != image.Rect(0, 0, viewW, viewH) {
				t.Errorf("the first frame's damage is %v, want the whole surface", damage)
			}
			want, err := soft.Render(first, viewW, viewH)
			if err != nil {
				t.Fatal(err)
			}
			samePixels(t, "first frame", surf.Image(), want)

			// Draw every frame the change made, one after the other, as a view does.
			frames := len(ts.Scenes())
			tt.change(t, ts)
			if len(ts.Scenes()) == frames {
				t.Fatal("the change made no frame")
			}
			whole := viewW * viewH
			smallest, drawn := whole, 0
			for i, next := range ts.Scenes()[frames:] {
				before := append([]byte(nil), surf.Image().Pix...)
				damage, err := surf.Draw(det.Detach(next))
				if err != nil {
					t.Fatal(err)
				}
				outsideUnchanged(t, before, surf.Image(), damage)
				if want, err = soft.Render(next, viewW, viewH); err != nil {
					t.Fatal(err)
				}
				samePixels(t, fmt.Sprintf("frame %d after the change", i+1), surf.Image(), want)
				if len(damage) == 0 {
					continue
				}
				drawn++
				smallest = min(smallest, area(damage))
				if tt.want == damageAll && (len(damage) != 1 || damage[0] != image.Rect(0, 0, viewW, viewH)) {
					t.Errorf("frame %d: the damage is %v, want the whole surface", i+1, damage)
				}
			}
			if drawn == 0 {
				t.Fatal("no frame of the change has damage")
			}
			if tt.want == damagePart && smallest >= whole {
				t.Errorf("every frame damaged the whole surface, want one that damaged part of it")
			}
		})
	}
}

func TestSurfaceResize(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: viewW, Height: viewH}})
	ts.PumpWidget(scaffoldApp(textBody()))
	scene := lastScene(ts)
	det := offscreen.NewDetacher()
	surf := offscreen.NewSurface(viewW, viewH)
	if _, err := surf.Draw(det.Detach(scene)); err != nil {
		t.Fatal(err)
	}
	tests := []struct{ w, h int }{{viewW / 2, viewH}, {viewW, viewH}, {viewW + 100, viewH + 50}}
	for _, tt := range tests {
		surf.Resize(tt.w, tt.h)
		if surf.Width() != tt.w || surf.Height() != tt.h || surf.Image().Rect != image.Rect(0, 0, tt.w, tt.h) {
			t.Fatalf("after Resize(%d, %d) the surface is %dx%d, image %v", tt.w, tt.h, surf.Width(), surf.Height(), surf.Image().Rect)
		}
		damage, err := surf.Draw(det.Detach(scene))
		if err != nil {
			t.Fatal(err)
		}
		if len(damage) != 1 || damage[0] != image.Rect(0, 0, tt.w, tt.h) {
			t.Errorf("%dx%d: the damage is %v, want the whole surface", tt.w, tt.h, damage)
		}
		want, err := soft.Render(scene, tt.w, tt.h)
		if err != nil {
			t.Fatal(err)
		}
		samePixels(t, "resized", surf.Image(), want)
	}
}

// TestSurfaceUnchangedScene checks that a scene that changes nothing has no damage and draws
// nothing, whether it is the scene again or another with the same pictures.
func TestSurfaceUnchangedScene(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFF3366CC))
	c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 80, 40), 8, 8), p)
	pic := rec.EndRecording()
	build := func() *ui.Scene {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, pic)
		return b.Build()
	}
	tests := []struct {
		name string
		next func(first *ui.Scene) *ui.Scene
	}{
		{"the same scene", func(first *ui.Scene) *ui.Scene { return first }},
		{"another scene with the same picture", func(*ui.Scene) *ui.Scene { return build() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			surf := offscreen.NewSurface(100, 60)
			first := build()
			if _, err := surf.Draw(first); err != nil {
				t.Fatal(err)
			}
			// Mark the pixels: a redraw would clear them.
			surf.Image().Pix[0] = 77
			damage, err := surf.Draw(tt.next(first))
			if err != nil {
				t.Fatal(err)
			}
			if len(damage) != 0 {
				t.Errorf("the damage is %v, want none", damage)
			}
			if surf.Image().Pix[0] != 77 {
				t.Error("something was drawn")
			}
		})
	}
}

func TestSurfaceErrors(t *testing.T) {
	invalid := func(w, h int) func() { return func() { offscreen.NewSurface(w, h) } }
	resize := func(w, h int) func() { return func() { offscreen.NewSurface(10, 10).Resize(w, h) } }
	panics := []struct {
		name  string
		do    func()
		route string
	}{
		{"zero width", invalid(0, 10), "NewSurface.SurfaceSizeInvalid"},
		{"negative height", invalid(10, -1), "NewSurface.SurfaceSizeInvalid"},
		{"too large", invalid(1<<20, 10), "NewSurface.SurfaceSizeInvalid"},
		{"resize to zero", resize(0, 0), "Surface.Resize.SurfaceSizeInvalid"},
	}
	for _, tt := range panics {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, _ := recover().(error)
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != "SurfaceSizeInvalid" || !appErr.HasRoute(tt.route) {
					t.Errorf("recovered %v, want SurfaceSizeInvalid via %s", err, tt.route)
				}
			}()
			tt.do()
		})
	}
	t.Run("nil scene", func(t *testing.T) {
		damage, err := offscreen.NewSurface(10, 10).Draw(nil)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "SurfaceSceneNil" || !appErr.HasRoute("Surface.Draw.SurfaceSceneNil") || damage != nil {
			t.Errorf("got %v, %v", damage, err)
		}
	})
	t.Run("a scene the renderer cannot draw", func(t *testing.T) {
		img := ui.NewImageFromRGBA(make([]byte, 16), 2, 2)
		b := ui.NewSceneBuilder()
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawImage(img, ui.Offset{}, ui.NewPaint())
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		img.Dispose()
		surf := offscreen.NewSurface(10, 10)
		_, err := surf.Draw(b.Build())
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "SoftImageDisposed" || !appErr.HasRoute("Surface.Draw->renderer.checkImage") && !appErr.HasRoute("Surface.Draw") {
			t.Errorf("got %v", err)
		}
		// The next frame redraws everything.
		ok2 := ui.NewSceneBuilder()
		damage, err := surf.Draw(ok2.Build())
		if err != nil || len(damage) != 1 || damage[0] != image.Rect(0, 0, 10, 10) {
			t.Errorf("after the error: damage %v, err %v", damage, err)
		}
	})
}

// outsideUnchanged fails the test when a pixel outside the rectangles is not what it was.
func outsideUnchanged(t *testing.T, before []byte, img *image.RGBA, rects []image.Rectangle) {
	t.Helper()
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			inside := false
			for _, r := range rects {
				inside = inside || (image.Point{X: x, Y: y}).In(r)
			}
			i := img.PixOffset(x, y)
			if !inside && !bytes.Equal(before[i:i+4], img.Pix[i:i+4]) {
				t.Fatalf("pixel %d,%d is outside the damage %v and changed", x, y, rects)
			}
		}
	}
}
