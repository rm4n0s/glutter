package fontconfig

import (
	"context"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
)

func TestCollectionOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := Open(ctx)
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	defer m.Close()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})

	tests := []struct {
		name   string
		family string
		weight int
		italic bool
		lang   string
	}{
		{"sans-serif", "sans-serif", 400, false, ""},
		{"bold serif", "serif", 700, false, ""},
		{"japanese", "sans-serif", 400, false, "ja"},
		{"monospace italic", "monospace", 400, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := c.Match(ctx, tt.family, tt.weight, tt.italic, tt.lang)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("Match returned no font file")
			}
			// The first files load, including faces of collections (.ttc).
			for _, f := range files[:min(len(files), 5)] {
				if _, err := c.Face(ctx, f); err != nil {
					t.Errorf("Face(%+v): %v", f, err)
				}
			}
		})
	}
}
