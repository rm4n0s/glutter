//go:build glutter_browser

package examples

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// TestBrowserEveryWebExampleDrawsItsFirstFrame opens each web example in Chromium (WebGPU) and in
// Firefox (WebGL 2) and waits for its first idle frame (proof method D, SPEC §23.8). The program must
// not end with an error (the harness reports that as WebAppFailed), and the frame must have the size
// of the window and show more than one color: a blank page is a failure. The look of the frames is
// checked by the proofs and by the goldens of each example, not here.
func TestBrowserEveryWebExampleDrawsItsFirstFrame(t *testing.T) {
	const width, height = 900, 640
	root := webtest.ModuleRoot(t)
	for _, name := range webExamples(t) {
		t.Run(name, func(t *testing.T) {
			out, err := os.MkdirTemp("", "glutter-example-web-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(out) }) // a temporary directory: a leftover is harmless
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if _, err := webtool.Build(ctx, webtool.BuildOptions{Package: "./examples/" + name, Dir: root, OutDir: out}); err != nil {
				t.Fatalf("building %s for the web: %v", name, err)
			}
			webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				data, err := webtest.Shot(ctx, webtest.ShotOptions{Dir: out, Browser: b, Width: width, Height: height})
				if err != nil {
					t.Fatalf("%s in %s: %v", name, b, err)
				}
				img, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("the screenshot is not a PNG: %v", err)
				}
				if got := img.Bounds().Size(); got != (image.Point{X: width, Y: height}) {
					t.Errorf("the frame is %v, want %dx%d", got, width, height)
				}
				if !hasTwoColors(img) {
					t.Error("the first frame is blank")
				}
			})
		})
	}
}

// hasTwoColors says whether the image holds at least two different pixels.
func hasTwoColors(img image.Image) bool {
	b := img.Bounds()
	first := image.NewRGBA(image.Rect(0, 0, 1, 1))
	first.Set(0, 0, img.At(b.Min.X, b.Min.Y))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.At(x, y) != first.At(0, 0) {
				return true
			}
		}
	}
	return false
}
