package proof

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

func TestEveryScenePicksAndRenders(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("no proof scenes registered")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("names not sorted: %v", names)
		}
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			s, err := Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			if s.Name != name || s.Description == "" || s.Width <= 0 || s.Height <= 0 {
				t.Errorf("incomplete scene %+v", s)
			}
			img, err := soft.Render(s.Build(), s.Width, s.Height)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if img.Bounds().Dx() != s.Width || img.Bounds().Dy() != s.Height {
				t.Errorf("size = %v", img.Bounds())
			}
		})
	}
}

func TestLookupUnknown(t *testing.T) {
	_, err := Lookup("nope")
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("ProofSceneUnknown") {
		t.Errorf("expected ProofSceneUnknown, got %v", err)
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		err, _ := recover().(error)
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("ProofSceneDuplicate") {
			t.Errorf("expected ProofSceneDuplicate, got %v", err)
		}
	}()
	register(Scene{Name: Names()[0], Build: func() *ui.Scene { return nil }})
}
