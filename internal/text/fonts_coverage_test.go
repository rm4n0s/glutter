package text

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-text/typesetting/fontscan"
	"github.com/rm4n0s/glutter/internal/text/fonttest"
	"golang.org/x/image/font/gofont/goregular"
)

// coverageFonts writes test fonts that share the outlines of Go Regular and differ in their character
// maps: Latin (the font itself), Greek (25 letters from alpha, to omega), Han (16 from U+4E00) and one whose cmap
// table cannot be read.
func coverageFonts(t *testing.T) (latin, greek, han, broken FontFile) {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, data []byte) FontFile {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return FontFile{Path: path, Family: name}
	}
	return write("latin.ttf", goregular.TTF),
		write("greek.ttf", fonttest.WithCmap(t, goregular.TTF, fonttest.Range('α', 25))),
		write("han.ttf", fonttest.WithCmap(t, goregular.TTF, fonttest.Range(0x4E00, 16))),
		write("broken.ttf", fonttest.WithBrokenCmap(t, goregular.TTF))
}

func TestCoversFromTheCmap(t *testing.T) {
	latin, greek, han, _ := coverageFonts(t)
	c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
	ctx := context.Background()
	tests := []struct {
		name string
		file FontFile
		r    rune
		want bool
	}{
		{"a letter in the Latin font", latin, 'a', true},
		{"Han is not in the Latin font", latin, '日', false},
		{"a Greek letter in the Greek font", greek, 'ω', true},
		{"the 25th Greek letter ends the range", greek, 'α' + 24, true},
		{"one past the range", greek, 'α' + 25, false},
		{"a Latin letter is not in the Greek font", greek, 'a', false},
		{"Han in the Han font", han, 0x4E0F, true},
		{"a tab is in none of the test fonts", han, '\t', false},
		{"the same answer again, from what was kept", greek, 'ω', true},
		{"the high bits of the index are ignored", FontFile{Path: greek.Path, Index: 0x10000}, 'ω', true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Covers(ctx, tt.file, tt.r)
			if err != nil || got != tt.want {
				t.Errorf("Covers(%q) = %t, %v; want %t", tt.r, got, err, tt.want)
			}
		})
	}
	if n := c.ParsedFonts(); n != 0 {
		t.Errorf("%d fonts were parsed in full to read their character maps, want none", n)
	}
}

func TestCoversOfLoadedAndBundledFonts(t *testing.T) {
	c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
	if err := c.AddFont("Greek", fonttest.WithCmap(t, goregular.TTF, fonttest.Range('α', 25))); err != nil {
		t.Fatal(err)
	}
	if err := c.AddBundled("Han", fonttest.WithCmap(t, goregular.TTF, fonttest.Range(0x4E00, 16))); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		file FontFile
		r    rune
		want bool
	}{
		{"a loaded font has its letters", FontFile{Path: LoadedPrefix + "1"}, 'β', true},
		{"a loaded font lacks others", FontFile{Path: LoadedPrefix + "1"}, 'b', false},
		{"a bundled font has its characters", FontFile{Path: BundledPrefix + "Han"}, 0x4E01, true},
		{"a bundled font lacks others", FontFile{Path: BundledPrefix + "Han"}, 'β', false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Covers(ctx, tt.file, tt.r)
			if err != nil || got != tt.want {
				t.Errorf("Covers(%q) = %t, %v; want %t", tt.r, got, err, tt.want)
			}
		})
	}
	// A replaced bundled font is read again.
	if err := c.AddBundled("Han", fonttest.WithCmap(t, goregular.TTF, fonttest.Range('a', 4))); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Covers(ctx, FontFile{Path: BundledPrefix + "Han"}, 'a'); err != nil || !got {
		t.Errorf("after the font was replaced, Covers('a') = %t, %v; want true", got, err)
	}
	if n := c.ParsedFonts(); n != 0 {
		t.Errorf("%d fonts were parsed in full, want none", n)
	}
}

func TestCoversOfAParsedFontUsesItsMap(t *testing.T) {
	latin, _, _, _ := coverageFonts(t)
	c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
	if _, err := c.Face(context.Background(), latin); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Covers(context.Background(), latin, 'a'); err != nil || !got {
		t.Errorf("Covers('a') = %t, %v; want true", got, err)
	}
	if len(c.cmaps) != 0 {
		t.Errorf("the map of a parsed font was read again: %d kept", len(c.cmaps))
	}
}

// coverMatcher is a Matcher that is a Coverer: it knows its fonts by a table and says nothing of the others.
type coverMatcher struct {
	fakeMatcher
	covers map[string]string // the runes of each path
	calls2 int
}

func (m *coverMatcher) Covers(_ context.Context, f FontFile, r rune) (has, known bool) {
	m.calls2++
	runes, ok := m.covers[f.Path]
	if !ok {
		return false, false
	}
	for _, c := range runes {
		if c == r {
			return true, true
		}
	}
	return false, true
}

func TestCoversPrefersTheMatcher(t *testing.T) {
	latin, greek, han, _ := coverageFonts(t)
	m := &coverMatcher{covers: map[string]string{
		"/nowhere/listed.ttf": "ab", // a file that is not there: the matcher answers, nothing is opened
		greek.Path:            "a",  // the matcher says Latin where the font's own map says Greek
		latin.Path:            "",   // and nothing where the font has Latin
	}}
	c := NewCollection(CollectionOpts{Matcher: m})
	ctx := context.Background()
	tests := []struct {
		name string
		file FontFile
		r    rune
		want bool
	}{
		{"a file the matcher knows is not opened", FontFile{Path: "/nowhere/listed.ttf"}, 'a', true},
		{"and lacks what it does not say", FontFile{Path: "/nowhere/listed.ttf"}, 'c', false},
		{"the matcher's answer wins over the cmap, yes", greek, 'a', true},
		{"the matcher's answer wins over the cmap, no", latin, 'a', false},
		{"a font the matcher does not know is read", han, 0x4E01, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Covers(ctx, tt.file, tt.r)
			if err != nil || got != tt.want {
				t.Errorf("Covers(%q) = %t, %v; want %t", tt.r, got, err, tt.want)
			}
		})
	}
	if n := c.ParsedFonts(); n != 0 {
		t.Errorf("%d fonts were parsed in full, want none", n)
	}
}

func TestCoversNeverAsksTheMatcherAboutLoadedFonts(t *testing.T) {
	m := &coverMatcher{covers: map[string]string{LoadedPrefix + "1": "z"}}
	c := NewCollection(CollectionOpts{Matcher: m})
	if err := c.AddFont("Greek", fonttest.WithCmap(t, goregular.TTF, fonttest.Range('α', 25))); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Covers(context.Background(), FontFile{Path: LoadedPrefix + "1"}, 'α'); err != nil || !got {
		t.Errorf("Covers = %t, %v; want true, from the font's own map", got, err)
	}
	if m.calls2 != 0 {
		t.Errorf("the matcher was asked %d times about a font it did not list", m.calls2)
	}
}

// TestCoversFromTheScan checks the footprints of the font scan against the cmap of each font file.
func TestCoversFromTheScan(t *testing.T) {
	s, err := newScanMatcher("", nil) // the cache directory of TestMain: fontscan scans once per process
	if err != nil {
		t.Skipf("the font scan is not available: %v", err)
	}
	if len(s.footprints) == 0 {
		t.Skip("the system has no fonts")
	}
	other := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}}) // reads the cmap itself
	ctx := context.Background()
	checked := 0
	for _, fp := range s.footprints {
		if checked == 25 {
			break
		}
		file := FontFile{Path: fp.Location.File, Index: int(fp.Location.Index)}
		if !supportedFont(file.Path) {
			continue
		}
		for _, r := range []rune{'a', '\t', 'α', '日', 0x1F600} {
			has, known := s.Covers(ctx, file, r)
			if !known {
				t.Fatalf("Covers(%s, %q) is unknown for a font of the scan", file.Path, r)
			}
			want, err := other.Covers(ctx, file, r)
			if err != nil {
				continue // a font that the scan lists and go-text cannot read
			}
			if has != want {
				t.Errorf("Covers(%s#%d, %q): the footprint says %t, the cmap %t", file.Path, file.Index, r, has, want)
			}
		}
		checked++
	}
	if _, known := s.Covers(ctx, FontFile{Path: "/nowhere/not-a-font.ttf"}, 'a'); known {
		t.Error("Covers knows a file that the scan did not list")
	}
}

// TestScanCoversUsesTheFootprints builds a matcher from a footprint with no runes: the font is known
// and has nothing, and a font of another face of the same file is not known.
func TestScanCoversUsesTheFootprints(t *testing.T) {
	s := &scanMatcher{footprints: []fontscan.Footprint{{Location: fontscan.Location{File: "/f/a.ttf", Index: 1}}}}
	tests := []struct {
		name      string
		file      FontFile
		has, know bool
	}{
		{"a listed face with no runes", FontFile{Path: "/f/a.ttf", Index: 1}, false, true},
		{"the high bits of the index are the instance, not the face", FontFile{Path: "/f/a.ttf", Index: 1 | 1<<16}, false, true},
		{"another face of the file", FontFile{Path: "/f/a.ttf", Index: 0}, false, false},
		{"another file", FontFile{Path: "/f/b.ttf", Index: 1}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, known := s.Covers(context.Background(), tt.file, 'a')
			if has != tt.has || known != tt.know {
				t.Errorf("Covers = %t, %t; want %t, %t", has, known, tt.has, tt.know)
			}
		})
	}
}

func TestCoversFailures(t *testing.T) {
	_, _, _, broken := coverageFonts(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	notFont := filepath.Join(dir, "notes.ttf")
	if err := os.WriteFile(notFont, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		ctx   context.Context
		file  FontFile
		route string
	}{
		{"a cmap that cannot be read", context.Background(), broken, "Collection.Covers->Collection.loadCmap->readCmap.FontLoadFailed"},
		{"a file that is not a font", context.Background(), FontFile{Path: notFont}, "Collection.Covers->Collection.loadCmap->readCmap.FontLoadFailed"},
		{"a file that is not there", context.Background(), FontFile{Path: filepath.Join(dir, "missing.ttf")}, "Collection.Covers->Collection.loadCmap.FontLoadFailed"},
		{"no face at the index", context.Background(), FontFile{Path: broken.Path, Index: 3}, "Collection.Covers->Collection.loadCmap->readCmap.FontLoadFailed"},
		{"an unknown bundled family", context.Background(), FontFile{Path: BundledPrefix + "Nope"}, "Collection.Covers->Collection.loadCmap.FontLoadFailed"},
		{"canceled", canceled, broken, "Collection.Covers.FontLoadCanceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
			got, err := c.Covers(tt.ctx, tt.file, 'a')
			if got {
				t.Error("a font that cannot be read has a glyph")
			}
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

// TestCoversReportsAnUnreadableCmapOnce checks that the failure is reported by the first call only: the
// text layout asks for every character, and would report it for each.
func TestCoversReportsAnUnreadableCmapOnce(t *testing.T) {
	_, _, _, broken := coverageFonts(t)
	c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
	reported := 0
	for _, r := range []rune{'a', 'b', 'c', 'a'} {
		got, err := c.Covers(context.Background(), broken, r)
		if err != nil {
			reported++
		}
		if got {
			t.Errorf("Covers(%q) = true for a font with no readable map", r)
		}
	}
	if reported != 1 {
		t.Errorf("the failure was reported %d times, want once", reported)
	}
}

func TestParsedFontsCountsFaceNotCovers(t *testing.T) {
	latin, greek, _, _ := coverageFonts(t)
	c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{}})
	ctx := context.Background()
	if _, err := c.Covers(ctx, greek, 'a'); err != nil {
		t.Fatal(err)
	}
	if n := c.ParsedFonts(); n != 0 {
		t.Fatalf("ParsedFonts after Covers = %d, want 0", n)
	}
	for i := 0; i < 2; i++ { // the second call finds the font parsed
		if _, err := c.Face(ctx, latin); err != nil {
			t.Fatal(err)
		}
	}
	if n := c.ParsedFonts(); n != 1 {
		t.Errorf("ParsedFonts after two Face calls of one font = %d, want 1", n)
	}
}
