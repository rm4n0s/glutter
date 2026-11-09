package darwinfonts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"golang.org/x/image/font/gofont/goregular"
)

// TestCoversWithTheDefaultCheck reads real files: a font it can read is known, a file it cannot read is
// not, so that text.Collection.Covers reads it and reports the failure.
func TestCoversWithTheDefaultCheck(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good.ttf"), filepath.Join(dir, "bad.ttf")
	if err := os.WriteFile(good, goregular.TTF, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := New(Options{Faces: []Face{
		{Family: "Good", Path: good, Weight: 400}, {Family: "Bad", Path: bad, Weight: 400}, {Family: "Gone", Path: filepath.Join(dir, "gone.ttf"), Weight: 400},
	}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		path      string
		r         rune
		has, know bool
	}{
		{"a letter", good, 'a', true, true},
		{"Han is not in Go Regular", good, '日', false, true},
		{"a file that is not a font", bad, 'a', false, false},
		{"a file that is not there", filepath.Join(dir, "gone.ttf"), 'a', false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, known := m.Covers(context.Background(), text.FontFile{Path: tt.path}, tt.r)
			if has != tt.has || known != tt.know {
				t.Errorf("Covers(%q) = %t, %t; want %t, %t", tt.r, has, known, tt.has, tt.know)
			}
		})
	}
}
