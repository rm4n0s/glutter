package offscreen_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// imageScene is a scene that draws images in every way an operation can: DrawImage,
// DrawImageRect, DrawImageNine, DrawAtlas, an image shader in a paint, an image inside a
// picture inside a picture, and an image shader in a shader mask layer.
func imageScene(img *ui.Image) *ui.Scene {
	inner := gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawImage(img, ui.Offset{Dx: 70, Dy: 70}, nil)
	})
	pic := gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawColor(ui.NewColor(0xFFF0F0F0), ui.BlendModeSrcOver)
		c.DrawImage(img, ui.Offset{Dx: 4, Dy: 4}, nil)
		c.DrawImageRect(img, ui.NewRectFromLTWH(4, 4, 16, 16), ui.NewRectFromLTWH(44, 4, 40, 30), nil)
		c.DrawImageNine(img, ui.NewRectFromLTWH(10, 10, 12, 12), ui.NewRectFromLTWH(4, 44, 60, 50), nil)
		c.DrawAtlas(img, []ui.RSTransform{{Scos: 1, Ssin: 0, Tx: 90, Ty: 4}, {Scos: 0.5, Ssin: 0.5, Tx: 100, Ty: 30}},
			[]ui.Rect{ui.NewRectFromLTWH(0, 0, 16, 16), ui.NewRectFromLTWH(16, 16, 16, 16)}, nil, ui.BlendModeSrcOver, nil, ui.NewPaint())
		shaded := ui.NewPaint()
		shaded.SetShader(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeMirror, nil))
		c.DrawRect(ui.NewRectFromLTWH(70, 44, 50, 20), shaded)
		c.DrawPicture(inner)
	})
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, pic)
	b.PushShaderMask(ui.NewImageShader(img, ui.TileModeClamp, ui.TileModeClamp, nil), ui.NewRectFromLTWH(0, 100, 128, 28), ui.BlendModeSrcIn)
	b.AddPicture(ui.Offset{}, gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 100, 128, 28), gputest.FillPaint(0xFF2060C0))
	}))
	b.Pop()
	return b.Build()
}

// richBody is text in the ways a paragraph draws: plain, with a background, a shadow, a
// blurred shadow, wavy and dotted decorations, wrapped lines, and rotated.
func richBody() widgets.Widget {
	style := func(o painting.TextStyleOpts) *painting.TextStyle { return painting.NewTextStyle(o) }
	red := ui.NewColor(0xFFC62828)
	line := func(s string, st *painting.TextStyle) widgets.Widget {
		return widgets.NewText(s, widgets.TextOpts{Style: st})
	}
	return widgets.NewColumn([]widgets.Widget{
		line("Plain text, 14 pixels", style(painting.TextStyleOpts{FontSize: foundation.Some(14.0)})),
		line("Background color", style(painting.TextStyleOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFEB3B))})),
		line("A shadow", style(painting.TextStyleOpts{FontSize: foundation.Some(22.0), Shadows: []ui.Shadow{
			ui.NewShadow(ui.ShadowOpts{Color: &red, Offset: ui.Offset{Dx: 2, Dy: 2}, BlurRadius: 3})}})),
		line("Wavy underline", style(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline), DecorationStyle: ui.TextDecorationStyleWavy})),
		line("Dotted line through", style(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationLineThrough), DecorationStyle: ui.TextDecorationStyleDotted})),
		widgets.NewSizedBox(line("Wrapped text that is long enough to need more than one line to show", nil), widgets.SizedBoxOpts{Width: foundation.Some(180.0)}),
		widgets.NewTransformRotate(0.3, line("Rotated text", nil)),
	})
}

// paragraphsOf lists the paragraphs a scene draws, in the pictures and the pictures inside them.
func paragraphsOf(l ui.Layer) []*ui.Paragraph {
	var out []*ui.Paragraph
	var inPicture func(p *ui.Picture)
	inPicture = func(p *ui.Picture) {
		for _, op := range p.DisplayList().Ops() {
			switch v := op.(type) {
			case ui.OpDrawParagraph:
				out = append(out, v.Paragraph)
			case ui.OpDrawPicture:
				inPicture(v.Picture)
			}
		}
	}
	if v, ok := l.(*ui.PictureLayer); ok {
		inPicture(v.Picture)
	}
	for _, c := range l.Children() {
		out = append(out, paragraphsOf(c)...)
	}
	return out
}

func TestDetachDrawsTheSamePixels(t *testing.T) {
	img := gputest.Pattern(32)
	tests := []struct {
		name  string
		scene func(t *testing.T) *ui.Scene
		w, h  int
	}{
		{"images in every operation", func(*testing.T) *ui.Scene { return imageScene(img) }, 128, 128},
		{"text in every style", func(t *testing.T) *ui.Scene {
			_, s := pumpFrame(t, scaffoldApp(richBody()))
			return s
		}, frameW, frameH},
		{"a screen of text", func(t *testing.T) *ui.Scene { _, s := pumpFrame(t, scaffoldApp(textBody())); return s }, frameW, frameH},
		{"a scene with no paragraph and no image", func(*testing.T) *ui.Scene { return gputest.CanvasScene(gputest.StripesAndDisc) }, 128, 128},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := tt.scene(t)
			want, err := soft.Render(scene, tt.w, tt.h)
			if err != nil {
				t.Fatal(err)
			}
			detached := offscreen.NewDetacher().Detach(scene)
			if detached == scene {
				t.Fatal("Detach returned the scene itself")
			}
			got, err := soft.Render(detached, tt.w, tt.h)
			if err != nil {
				t.Fatal(err)
			}
			samePixels(t, tt.name, got, want)
		})
	}
}

// TestDetachedCopyKeepsTheOldPixels lays the paragraphs of the source out again with another
// width and disposes its images: the source then draws something else (or fails), and the copy
// draws what the source drew when it was detached.
func TestDetachedCopyKeepsTheOldPixels(t *testing.T) {
	t.Run("paragraphs laid out again", func(t *testing.T) {
		_, scene := pumpFrame(t, scaffoldApp(richBody()))
		paragraphs := paragraphsOf(scene.Root())
		if len(paragraphs) == 0 {
			t.Fatal("the scene has no paragraph")
		}
		want, err := soft.Render(scene, frameW, frameH)
		if err != nil {
			t.Fatal(err)
		}
		detached := offscreen.NewDetacher().Detach(scene)
		for _, p := range paragraphs {
			p.Layout(ui.ParagraphConstraints{Width: 40})
		}
		changed, err := soft.Render(scene, frameW, frameH)
		if err != nil {
			t.Fatal(err)
		}
		if string(changed.Pix) == string(want.Pix) {
			t.Fatal("laying the paragraphs out again did not change the source, so this test checks nothing")
		}
		got, err := soft.Render(detached, frameW, frameH)
		if err != nil {
			t.Fatal(err)
		}
		samePixels(t, "the copy", got, want)
	})
	t.Run("images disposed", func(t *testing.T) {
		img := gputest.Pattern(32).Clone()
		scene := imageScene(img)
		want, err := soft.Render(scene, 128, 128)
		if err != nil {
			t.Fatal(err)
		}
		detached := offscreen.NewDetacher().Detach(scene)
		img.Dispose()
		_, err = soft.Render(scene, 128, 128)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "SoftImageDisposed" {
			t.Fatalf("the source with a disposed image: %v, want SoftImageDisposed, so this test checks nothing", err)
		}
		got, err := soft.Render(detached, 128, 128)
		if err != nil {
			t.Fatal(err)
		}
		samePixels(t, "the copy", got, want)
	})
	t.Run("an image that was disposed before", func(t *testing.T) {
		img := gputest.Pattern(32).Clone()
		img.Dispose()
		_, err := soft.Render(offscreen.NewDetacher().Detach(imageScene(img)), 128, 128)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "SoftImageDisposed" {
			t.Errorf("the copy of a disposed image draws it: %v", err)
		}
	})
}

// TestDetachKeepsWhatItCan checks that what holds nothing to detach is shared, and the rest is
// not.
func TestDetachKeepsWhatItCan(t *testing.T) {
	plain := gputest.PictureOf(gputest.StripesAndDisc)
	_, live := pumpFrame(t, scaffoldApp(textBody()))
	b := ui.NewSceneBuilder()
	b.PushClipRect(ui.NewRectFromLTWH(0, 0, 100, 100))
	b.AddPicture(ui.Offset{Dx: 5, Dy: 5}, plain)
	b.Pop()
	scene := b.Build()

	detached := offscreen.NewDetacher().Detach(scene)
	if detached.Root() != scene.Root() {
		t.Error("a scene without paragraphs and images has a root of its own")
	}
	if got := offscreen.NewDetacher().Detach(live); got.Root() == live.Root() {
		t.Error("a scene with paragraphs shares its root")
	}
	if offscreen.NewDetacher().Detach(nil) != nil {
		t.Error("Detach(nil) is not nil")
	}
}

// TestDetachReusesCopies checks that the copy of a picture that the next scene has again is the
// same picture, so that the damage of a frame that changed nothing is empty, and that the
// detacher keeps the copies of the last scene only.
func TestDetachReusesCopies(t *testing.T) {
	_, live := pumpFrame(t, scaffoldApp(textBody()))
	var textPic *ui.Picture
	var walk func(l ui.Layer)
	walk = func(l ui.Layer) {
		if v, ok := l.(*ui.PictureLayer); ok && textPic == nil && len(paragraphsOf(v)) > 0 {
			textPic = v.Picture
		}
		for _, c := range l.Children() {
			walk(c)
		}
	}
	walk(live.Root())
	if textPic == nil {
		t.Fatal("no picture with text")
	}
	other := gputest.PictureOf(gputest.StripesAndDisc)
	scene := func(pics ...*ui.Picture) *ui.Scene {
		b := ui.NewSceneBuilder()
		for _, p := range pics {
			b.AddPicture(ui.Offset{}, p)
		}
		return b.Build()
	}
	picOf := func(s *ui.Scene, i int) *ui.Picture { return s.Root().Children()[i].(*ui.PictureLayer).Picture }

	d := offscreen.NewDetacher()
	d1 := d.Detach(scene(textPic, other))
	d2 := d.Detach(scene(textPic, other))
	if picOf(d1, 0) == textPic {
		t.Fatal("the picture with text was not copied")
	}
	if picOf(d2, 0) != picOf(d1, 0) || picOf(d2, 1) != other {
		t.Error("the next scene did not get the same copies")
	}
	surf := offscreen.NewSurface(frameW, frameH)
	if _, err := surf.Draw(d1); err != nil {
		t.Fatal(err)
	}
	if damage, err := surf.Draw(d2); err != nil || len(damage) != 0 {
		t.Errorf("the damage of a frame that changed nothing is %v, %v", damage, err)
	}
	// A scene without the picture makes the detacher forget its copy.
	d.Detach(scene(other))
	if d3 := d.Detach(scene(textPic)); picOf(d3, 0) == picOf(d1, 0) {
		t.Error("the detacher kept a copy of a picture that the last scene did not have")
	}
}

// TestDetachedScenesDrawWhileTheFrameworkGoesOn is the race test of the step: the test
// goroutine builds frames of a widget tree whose text changes and is laid out again, while
// another goroutine draws the detached scenes through a Surface. Under -race, drawing the scenes
// themselves instead of the copies is reported as a race (the text engine's caches).
func TestDetachedScenesDrawWhileTheFrameworkGoesOn(t *testing.T) {
	const frames = 40
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	det := offscreen.NewDetacher()
	surf := offscreen.NewSurface(400, 300)
	scenes := make(chan *ui.Scene, 1)
	var wg sync.WaitGroup
	var drawErr error
	var drawn int
	wg.Add(1)
	go func() {
		defer wg.Done()
		for s := range scenes {
			if _, err := surf.Draw(s); err != nil && drawErr == nil {
				drawErr = err
			}
			drawn++
		}
	}()
	var last *ui.Scene
	for i := 0; i < frames; i++ {
		// The text stays the same for four frames at a time and the width changes in each, so
		// that the framework lays the same paragraph out again.
		width := 120.0 + float64(i%7)*40
		text := fmt.Sprintf("Block %d: the quick brown fox jumps over the lazy dog, again and again, to fill the lines", i/4)
		ts.PumpWidget(scaffoldApp(widgets.NewSizedBox(widgets.NewText(text), widgets.SizedBoxOpts{Width: foundation.Some(width)})))
		ts.Pump(16 * time.Millisecond)
		last = lastScene(ts)
		scenes <- det.Detach(last)
	}
	close(scenes)
	wg.Wait()
	if drawErr != nil {
		t.Fatal(drawErr)
	}
	if drawn == 0 {
		t.Fatal("nothing was drawn")
	}
	// The last frame the goroutine drew is the last one the framework made.
	want, err := soft.Render(last, 400, 300)
	if err != nil {
		t.Fatal(err)
	}
	samePixels(t, "the last frame", surf.Image(), want)
}

func TestDetachedLongTextShowsTheSamePixelsAsTheTextItself(t *testing.T) {
	e := proof.BundledTextEngine()
	text := ""
	for i := 0; i < 300; i++ {
		text += fmt.Sprintf("Line %d of a long text, 0123456789\n", i)
	}
	tests := []struct {
		name   string
		style  ui.TextStyleOpts
		offset float64
	}{
		{"the start", ui.TextStyleOpts{}, 0},
		{"the middle", ui.TextStyleOpts{}, -2500},
		{"the end", ui.TextStyleOpts{}, -5400},
		{"a blurred shadow that reaches in from above", ui.TextStyleOpts{Shadows: []ui.Shadow{{Color: ui.NewColor(0xFF000000), Offset: ui.Offset{Dx: 3, Dy: 40}, BlurRadius: 8}}}, -2540},
		{"a decoration", ui.TextStyleOpts{Decoration: ui.Ptr(ui.TextDecorationUnderline)}, -3000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := tt.style
			style.FontFamily = fonts.RobotoFamily
			style.FontSize = ui.Ptr(15.0)
			b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{}))
			b.PushStyle(ui.NewTextStyle(style))
			b.AddText(text)
			p := b.Build()
			p.Layout(ui.NewParagraphConstraints(300))
			rec := ui.NewPictureRecorder()
			c := ui.NewCanvas(rec)
			c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
			c.DrawParagraph(p, ui.Offset{Dx: 4, Dy: tt.offset})
			sb := ui.NewSceneBuilder()
			sb.AddPicture(ui.Offset{}, rec.EndRecording())
			scene := sb.Build()

			want, err := soft.Render(scene, 320, 200)
			if err != nil {
				t.Fatal(err)
			}
			s := offscreen.NewSurface(320, 200)
			if _, err := s.Draw(offscreen.NewDetacher().Detach(scene)); err != nil {
				t.Fatal(err)
			}
			samePixels(t, "detached", s.Image(), want)
		})
	}
}
