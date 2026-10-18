package soft

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/proof"
)

// TestGoldenProofScenes keeps the scenes of the proof images from changing
// unnoticed: the blend-mode grid and the gradients.
func TestGoldenProofScenes(t *testing.T) {
	for _, name := range []string{"blend-modes", "gradients"} {
		t.Run(name, func(t *testing.T) {
			s, err := proof.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			img, err := Render(s.Build(), s.Width, s.Height)
			if err != nil {
				t.Fatal(err)
			}
			goldens.Check(t, "proof-"+name, img)
		})
	}
}
