package soft

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/proof"
)

// renderProof renders a registered proof scene.
func renderProof(t *testing.T, name string) *image.RGBA {
	t.Helper()
	s, err := proof.Lookup(name)
	if err != nil {
		t.Fatal(err)
	}
	img, err := Render(s.Build(), s.Width, s.Height)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// TestGoldenProofScenes keeps the scenes of the proof images from changing
// unnoticed: the blend-mode grid and the gradients.
func TestGoldenProofScenes(t *testing.T) {
	for _, name := range []string{"blend-modes", "gradients"} {
		t.Run(name, func(t *testing.T) {
			goldens.Check(t, "proof-"+name, renderProof(t, name))
		})
	}
}
