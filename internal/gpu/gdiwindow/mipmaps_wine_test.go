//go:build windows && glutter_wine

package gdiwindow

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestWineKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1
// (internal/issues, soft-mipmaps) with the software renderer of Windows into a real window under
// Wine: a spinner turns over a picture that is drawn at half its size. The picture's mipmap chain
// is made once and kept across the frames, and the window shows each frame (a pixel of the picture,
// one of the spinner and one of the background, read back with GetPixel). Before the step every
// frame made the chain again.
func TestWineKeepsTheMipmapsOfAPicture(t *testing.T) {
	hwnd := makeWindow(t, issues.SoftMipmapsWidth, issues.SoftMipmapsHeight)
	r := NewRenderer(NewPresenter(), Options{Background: ui.NewColor(0xFFFFFFFF)})
	defer func() { _ = r.Close() }()
	s, err := r.Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: hwnd})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Destroy()
	rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		img, err := r.RenderCapture(s, scene, ui.Size{Width: float64(w), Height: float64(h)}, 1)
		if err != nil {
			return nil, issues.SoftMipmapsStats{}, err
		}
		for _, pt := range [][2]int32{{4, 4}, {256, 192}, {600, 400}} {
			want := img.RGBAAt(int(pt[0]), int(pt[1]))
			want.A = 255
			if got := pixel(t, hwnd, pt[0], pt[1]); got != (color.RGBA{R: want.R, G: want.G, B: want.B, A: 255}) {
				t.Errorf("pixel %v of the window = %v, the frame has %v", pt, got, want)
			}
		}
		st := r.MipmapStats()
		return img, issues.SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range rep.Lines() {
		t.Log(l)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
}
