package offscreen

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestSurfaceKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1
// (internal/issues, soft-mipmaps) through a Surface: a spinner over a picture that is scaled down,
// drawn where the frames changed. The picture's mipmap chain is made once and kept across the
// frames. Before the step every rectangle of every frame made it again.
func TestSurfaceKeepsTheMipmapsOfAPicture(t *testing.T) {
	s := NewSurface(issues.SoftMipmapsWidth, issues.SoftMipmapsHeight)
	rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, w, h int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		if _, err := s.Draw(scene); err != nil {
			return nil, issues.SoftMipmapsStats{}, err
		}
		img := image.NewRGBA(s.Image().Rect)
		copy(img.Pix, s.Image().Pix)
		st := s.cache.Stats()
		return img, issues.SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
}
