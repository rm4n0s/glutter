package text

import (
	"bytes"
	"context"
	"testing"

	"github.com/rm4n0s/glutter/material/fonts"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// noSystem is a collection with no system fonts.
func noSystem(t *testing.T) *Collection {
	t.Helper()
	return NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
}

// loadedMono loads the four styles of Go Mono under the family "Go Mono" and
// returns the Path of each, in the order regular, bold, italic, bold italic.
func loadedMono(t *testing.T, c *Collection) (paths [4]string) {
	t.Helper()
	for i, data := range [][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF} {
		if err := c.AddFont("Go Mono", data); err != nil {
			t.Fatalf("AddFont %d: %v", i, err)
		}
	}
	files, err := c.Match(context.Background(), "Go Mono", 400, false, "")
	if err != nil || len(files) != 4 {
		t.Fatalf("Match after loading four fonts: %v, %v", files, err)
	}
	// The fonts are numbered in the order they were loaded.
	for i := range paths {
		paths[i] = LoadedPrefix + string(rune('1'+i))
	}
	return paths
}

func TestAddFontMatchOrder(t *testing.T) {
	c := noSystem(t)
	p := loadedMono(t, c)
	tests := []struct {
		name   string
		weight int
		italic bool
		want   [4]string
	}{
		{"regular", 400, false, [4]string{p[0], p[1], p[2], p[3]}},
		{"bold", 700, false, [4]string{p[1], p[0], p[3], p[2]}},
		{"italic", 400, true, [4]string{p[2], p[3], p[0], p[1]}},
		{"bold italic", 700, true, [4]string{p[3], p[2], p[1], p[0]}},
		{"w500 is closer to regular", 500, false, [4]string{p[0], p[1], p[2], p[3]}},
		{"w600 is closer to bold", 600, false, [4]string{p[1], p[0], p[3], p[2]}},
		{"w300 prefers regular and italic comes last", 300, false, [4]string{p[0], p[1], p[2], p[3]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := c.Match(context.Background(), "go mono", tt.weight, tt.italic, "")
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			var got [4]string
			for i, f := range files {
				if i < 4 {
					got[i] = f.Path
				}
				if !f.IsLoaded() || f.Family != "Go Mono" {
					t.Errorf("file %d = %+v, want a loaded Go Mono", i, f)
				}
			}
			if got != tt.want {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddFontFamilyAndDuplicates(t *testing.T) {
	c := noSystem(t)
	g0 := c.Generation()
	if err := c.AddFont("", gomono.TTF); err != nil {
		t.Fatalf("AddFont with the family of the font: %v", err)
	}
	g1 := c.Generation()
	if g1 == g0 {
		t.Error("loading a font did not change the generation")
	}
	files, err := c.Match(context.Background(), "Go Mono", 400, false, "")
	if err != nil || len(files) != 1 {
		t.Fatalf("the font is under the name inside it: %v, %v", files, err)
	}
	if err := c.AddFont("Go Mono", bytes.Clone(gomono.TTF)); err != nil {
		t.Fatalf("AddFont again: %v", err)
	}
	if c.Generation() != g1 {
		t.Error("the same bytes under the same family changed the generation")
	}
	if files, _ = c.Match(context.Background(), "Go Mono", 400, false, ""); len(files) != 1 {
		t.Errorf("the same font twice gives %d files", len(files))
	}
	// The same bytes under another family are another font.
	if err := c.AddFont("Terminal", gomono.TTF); err != nil {
		t.Fatalf("AddFont under another family: %v", err)
	}
	if files, _ = c.Match(context.Background(), "Terminal", 400, false, ""); len(files) != 1 || files[0].Path == "loaded:1" {
		t.Errorf("Terminal = %+v", files)
	}
}

func TestAddFontCopiesTheData(t *testing.T) {
	c := noSystem(t)
	data := bytes.Clone(gomono.TTF)
	if err := c.AddFont("Mono", data); err != nil {
		t.Fatal(err)
	}
	clear(data)
	files, _ := c.Match(context.Background(), "Mono", 400, false, "")
	if _, err := c.Face(context.Background(), files[0]); err != nil {
		t.Errorf("Face after the caller overwrote its bytes: %v", err)
	}
}

func TestAddFontBeatsBundled(t *testing.T) {
	c := bundledCollection(t, CollectionOpts{Matcher: &fakeMatcher{}})
	if err := c.AddFont("Roboto", gomono.TTF); err != nil {
		t.Fatal(err)
	}
	files, err := c.Match(context.Background(), fonts.RobotoFamily, 400, false, "")
	if err != nil || len(files) != 2 || !files[0].IsLoaded() || !files[1].IsBundled() {
		t.Errorf("Roboto = %+v, %v; want the loaded font, then the bundled one", files, err)
	}
}

func TestAddFontFace(t *testing.T) {
	c := noSystem(t)
	p := loadedMono(t, c)
	for i, path := range p {
		face, err := c.Face(context.Background(), FontFile{Path: path})
		if err != nil {
			t.Fatalf("Face %s: %v", path, err)
		}
		if _, ok := face.NominalGlyph('a'); !ok {
			t.Errorf("font %d has no 'a'", i)
		}
	}
}

func TestAddFontErrors(t *testing.T) {
	// A font whose name table is renamed to a tag nobody uses has no family name.
	nameless := bytes.Clone(gomono.TTF)
	if i := bytes.Index(nameless[:512], []byte("name")); i < 0 {
		t.Fatal("the name table is not in the first 512 bytes")
	} else {
		copy(nameless[i:], "nam_")
	}
	tests := []struct {
		name   string
		family string
		data   []byte
		route  string
	}{
		{"data is not a font", "X", []byte("this is not a font"), "Collection.AddFont->describeFont.FontDataInvalid"},
		{"no data", "X", nil, "Collection.AddFont->describeFont.FontDataInvalid"},
		{"no family and none in the font", "", nameless, "Collection.AddFont.FontFamilyMissing"},
		{"blank family and none in the font", "  ", nameless, "Collection.AddFont.FontFamilyMissing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := noSystem(t)
			g := c.Generation()
			err := c.AddFont(tt.family, tt.data)
			if err == nil {
				t.Fatal("AddFont succeeded")
			}
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
			}
			if c.Generation() != g {
				t.Error("a failed AddFont changed the generation")
			}
		})
	}
}
