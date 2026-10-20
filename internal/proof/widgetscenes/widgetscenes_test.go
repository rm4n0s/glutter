package widgetscenes

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
)

func TestSceneIsRegisteredAndRenders(t *testing.T) {
	s, err := proof.Lookup("material-ink")
	if err != nil {
		t.Fatal(err)
	}
	img, err := soft.Render(s.Build(), s.Width, s.Height)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != s.Width || b.Dy() != s.Height {
		t.Errorf("image is %v, want %dx%d", b, s.Width, s.Height)
	}
	// The light panel (top) and the dark panel (bottom) have different backgrounds.
	if top, bottom := img.RGBAAt(5, 5), img.RGBAAt(5, s.Height-5); top == bottom {
		t.Errorf("both panels have the background %v", top)
	}
	// The ripple of the light panel is frozen in the middle of its animation: the
	// center of the ink well differs from its edge, which the ripple has not reached.
	if center, edge := img.RGBAAt(380, 205), img.RGBAAt(230, 205); center == edge {
		t.Errorf("the ripple is not painted: the center and the edge are both %v", center)
	}
}

func TestSceneIsDeterministic(t *testing.T) {
	s, err := proof.Lookup("material-ink")
	if err != nil {
		t.Fatal(err)
	}
	a, err := soft.Render(s.Build(), s.Width, s.Height)
	if err != nil {
		t.Fatal(err)
	}
	b, err := soft.Render(s.Build(), s.Width, s.Height)
	if err != nil {
		t.Fatal(err)
	}
	if string(a.Pix) != string(b.Pix) {
		t.Error("two renders of the scene differ")
	}
}
