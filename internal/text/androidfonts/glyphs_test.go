package androidfonts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
)

// robotoPath is the Roboto that Material bundles; the tests use it as a real font file.
const robotoPath = "../../../material/fonts/Roboto-Variable.ttf"

func TestGlyphChecker(t *testing.T) {
	if _, err := os.Stat(robotoPath); err != nil {
		t.Skipf("no font file to test with: %v", err)
	}
	dir := t.TempDir()
	notFont := filepath.Join(dir, "notfont.ttf")
	if err := os.WriteFile(notFont, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := newGlyphChecker()
	tests := []struct {
		name  string
		path  string
		index int
		r     rune
		want  bool
	}{
		{"Latin", robotoPath, 0, 'A', true},
		{"Latin again, from the cache", robotoPath, 0, 'z', true},
		{"Greek", robotoPath, 0, 'Ω', true},
		{"Hiragana is not in Roboto", robotoPath, 0, 'あ', false},
		{"emoji is not in Roboto", robotoPath, 0, '\U0001F600', false},
		{"no face at the index", robotoPath, 1, 'A', false},
		{"a negative index", robotoPath, -1, 'A', false},
		{"a file that is not there", filepath.Join(dir, "missing.ttf"), 0, 'A', false},
		{"a file that is not a font", notFont, 0, 'A', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.hasGlyph(tt.path, tt.index, tt.r); got != tt.want {
				t.Errorf("hasGlyph(%q, %d, %q) = %t, want %t", tt.path, tt.index, tt.r, got, tt.want)
			}
		})
	}
}

// TestDefaultsWithRealFiles runs the matcher with the default file functions on a folder that has a
// fonts.xml and a real font, and plugs it into a text.Collection as the engine does.
func TestDefaultsWithRealFiles(t *testing.T) {
	data, err := os.ReadFile(robotoPath)
	if err != nil {
		t.Skipf("no font file to test with: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Roboto-Regular.ttf"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	xml := `<familyset version="23">
		<family name="sans-serif"><font weight="400" style="normal">Roboto-Regular.ttf</font></family>
		<alias name="arial" to="sans-serif"/>
		<family lang="ja"><font weight="400" style="normal">Missing-Japanese.ttc</font></family>
	</familyset>`
	if err := os.WriteFile(filepath.Join(dir, "fonts.xml"), []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := New(Options{FontsXML: []string{filepath.Join(dir, "fonts.xml")}, FontDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})

	roboto := filepath.Join(dir, "Roboto-Regular.ttf")
	tests := []struct {
		family string
		want   []string
	}{
		{"sans-serif", []string{roboto}},
		{"Arial", []string{roboto}},
		{"Comic Sans", nil},
	}
	for _, tt := range tests {
		files, err := c.Match(ctx, tt.family, 400, false, "")
		if err != nil {
			t.Fatalf("Match(%q): %v", tt.family, err)
		}
		var got []string
		for _, f := range files {
			got = append(got, f.Path)
		}
		if len(got) != len(tt.want) || (len(got) > 0 && got[0] != tt.want[0]) {
			t.Errorf("Collection.Match(%q) = %v, want %v", tt.family, got, tt.want)
		}
	}
	f, ok, err := c.Fallback(ctx, "sans-serif", 400, false, "ja", 'A')
	if err != nil || !ok || f.Path != roboto {
		t.Errorf("Fallback(A) = %v, %t, %v", f, ok, err)
	}
	if f, ok, err := c.Fallback(ctx, "sans-serif", 400, false, "ja", 'あ'); err != nil || ok {
		t.Errorf("Fallback(あ) = %v, %t, %v; want none", f, ok, err)
	}
	face, err := c.Face(ctx, text.FontFile{Path: roboto})
	if err != nil || face == nil {
		t.Errorf("Face of the matched file: %v", err)
	}
	families, err := c.Families(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, f := range families {
		found[f] = true
	}
	if !found["sans-serif"] || !found["arial"] || len(families) != 2 {
		t.Errorf("Families = %v", families)
	}
}

func TestCoversWithTheDefaultCheck(t *testing.T) {
	if _, err := os.Stat(robotoPath); err != nil {
		t.Skipf("no font file to test with: %v", err)
	}
	dir := t.TempDir()
	notFont := filepath.Join(dir, "notfont.ttf")
	if err := os.WriteFile(notFont, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := newGlyphChecker()
	tests := []struct {
		name      string
		path      string
		r         rune
		has, know bool
	}{
		{"a letter", robotoPath, 'A', true, true},
		{"hiragana is not in Roboto", robotoPath, 'あ', false, true},
		{"a file that is not a font", notFont, 'A', false, false},
		{"a file that is not there", filepath.Join(dir, "missing.ttf"), 'A', false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, known := c.covers(tt.path, 0, tt.r)
			if has != tt.has || known != tt.know {
				t.Errorf("covers(%q) = %t, %t; want %t, %t", tt.r, has, known, tt.has, tt.know)
			}
		})
	}
}

func TestCovers(t *testing.T) {
	d := device(t, fontsAPI26)
	d.glyphs = map[string]string{
		"/system/fonts/Roboto-Regular.ttf#0":      "abc",
		"/system/fonts/NotoSansCJK-Regular.ttc#2": "あ骨",
		"/system/fonts/NotoColorEmoji.ttf#0":      "\U0001F600",
	}
	m := mustNew(t, d)
	tests := []struct {
		name      string
		file      text.FontFile
		r         rune
		has, know bool
	}{
		{"a letter in Roboto", text.FontFile{Path: "/system/fonts/Roboto-Regular.ttf"}, 'a', true, true},
		{"a tab Roboto does not have", text.FontFile{Path: "/system/fonts/Roboto-Regular.ttf"}, '\t', false, true},
		{"Han in the CJK face of the configuration", text.FontFile{Path: "/system/fonts/NotoSansCJK-Regular.ttc", Index: 2}, '骨', true, true},
		{"Han in another face of the collection", text.FontFile{Path: "/system/fonts/NotoSansCJK-Regular.ttc", Index: 1}, '骨', false, true},
		{"the high bits of the index are the caller's to strip", text.FontFile{Path: "/system/fonts/Roboto-Regular.ttf", Index: 0x10000}, 'a', false, false},
		{"a font the configuration does not list", text.FontFile{Path: "/system/fonts/Other.ttf"}, 'a', false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, known := m.Covers(context.Background(), tt.file, tt.r)
			if has != tt.has || known != tt.know {
				t.Errorf("Covers(%+v, %q) = %t, %t; want %t, %t", tt.file, tt.r, has, known, tt.has, tt.know)
			}
		})
	}
}
