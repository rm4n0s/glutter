package viewport3d_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/viewport3d"
)

func TestRenderFailsWhenThePixelsDoNotArrive(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	c := scenetest.Cases()[0]
	// A wait that runs nothing leaves the read of the picture unfinished.
	img, err := scenetest.Render(rig.frame.Device, func(func() bool) {}, c)
	if img != nil {
		t.Errorf("got a picture of %v", img.Bounds())
	}
	wantTag(t, err, scenetest.TagReadbackFailed, "Render->RenderAsking.SceneReadbackFailed")
}

func TestEveryCaseHasANameASizeAndABuilder(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range scenetest.Cases() {
		if c.Name == "" || c.Width <= 0 || c.Height <= 0 || c.Build == nil || seen[c.Name] {
			t.Errorf("case %+v is not one that the display tests can use, or its name is used twice", c.Name)
		}
		seen[c.Name] = true
	}
	if len(seen) < 13 {
		t.Errorf("%d cases", len(seen))
	}
}
