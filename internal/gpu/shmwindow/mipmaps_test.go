package shmwindow

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestRendererKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1
// (internal/issues, soft-mipmaps) through the renderer: a spinner over a picture that is scaled
// down. The picture's mipmap chain is made once and kept across the frames, and every frame shows
// the picture and the spinner. Before the step every frame made the chain again.
func TestRendererKeepsTheMipmapsOfAPicture(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		size := ui.Size{Width: float64(w), Height: float64(h)}
		img, err := r.RenderCapture(s, scene, size, 1, true)
		st := r.MipmapStats()
		return img, issues.SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
}
