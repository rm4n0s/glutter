package soft

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestPictureAndSceneToImage(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawRect(rectAt(0, 0, 6, 4), paint(0xFFFF0000))
	c.DrawRect(rectAt(6, 0, 6, 4), paint(0x8000FF00))
	pic := rec.EndRecording()
	opts := ui.ToImageOpts{Rasterizer: ImageRasterizer{}}

	img, err := pic.ToImage(12, 8, opts)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width() != 12 || img.Height() != 8 {
		t.Fatalf("size = %dx%d", img.Width(), img.Height())
	}
	px := func(x, y int) [4]byte { i := 4 * (y*12 + x); return [4]byte(img.Pixels()[i : i+4]) }
	if px(2, 2) != [4]byte{255, 0, 0, 255} || px(8, 2) != [4]byte{0, 128, 0, 128} || px(2, 6) != [4]byte{} {
		t.Errorf("pixels: %v %v %v", px(2, 2), px(8, 2), px(2, 6))
	}

	// The same through a scene, and the picture again at another size, which crops.
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, pic)
	scene := b.Build()
	again, err := scene.ToImage(12, 8, opts)
	if err != nil || !bytes.Equal(again.Pixels(), img.Pixels()) {
		t.Errorf("scene.ToImage differs from picture.ToImage: %v", err)
	}
	small, err := pic.ToImageSync(4, 2, opts)
	if err != nil || small.Width() != 4 || small.Height() != 2 || small.Pixels()[0] != 255 {
		t.Errorf("cropped image: %v, %v", small, err)
	}

	// The image made by ToImage can be drawn, and is the same as the original.
	redrawn := renderOf(t, 12, 8, func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, withQuality(ui.FilterQualityNone)) })
	if !bytes.Equal(redrawn.Pix, img.Pixels()) {
		t.Error("drawing the rasterized image does not give the image back")
	}

	// A picture of text is rasterized like any other (a paragraph that cannot draw draws nothing).
	if _, err := rec2(t).ToImage(1, 1, opts); err != nil {
		t.Errorf("a picture with a paragraph failed: %v", err)
	}
	var rasterizer ImageRasterizer
	if _, err := rasterizer.RasterizeScene(scene, 0, 5); err == nil {
		t.Error("an invalid size should fail")
	}
}

// rec2 is a picture the software renderer cannot draw yet: text.
func rec2(t *testing.T) *ui.Picture {
	t.Helper()
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawParagraph(&ui.Paragraph{}, ui.Offset{})
	return rec.EndRecording()
}
