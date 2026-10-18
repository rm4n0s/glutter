package text

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/material/fonts"
)

// TestMain points the cache of the font scan to a directory of its own: fontscan
// scans once per process, so every test shares one index.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "glutter-text-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func routeOf(t *testing.T, err error) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	return appErr
}

// fakeMatcher answers every request with the same files.
type fakeMatcher struct {
	files []FontFile
	err   error
	calls []string
}

func (m *fakeMatcher) Match(_ context.Context, family string, _ int, _ bool, _ string) ([]FontFile, error) {
	m.calls = append(m.calls, family)
	return m.files, m.err
}

func bundledCollection(t *testing.T, opts ...CollectionOpts) *Collection {
	t.Helper()
	c := NewCollection(opts...)
	if err := fonts.Register(c); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return c
}

func TestBundledFonts(t *testing.T) {
	c := bundledCollection(t)
	ctx := context.Background()
	tests := []struct {
		name   string
		family string
		runes  []rune
	}{
		{"Roboto has Latin", fonts.RobotoFamily, []rune("Hello, Ägypten")},
		{"Roboto has Cyrillic and Greek", "roboto", []rune("Привет αβγ")},
		{"Material Icons has the icon code points", fonts.MaterialIconsFamily, []rune{0xe145, 0xe88a}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := c.Match(ctx, tt.family, 400, false, "")
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(files) == 0 || !files[0].IsBundled() {
				t.Fatalf("the first file is not the bundled font: %+v", files)
			}
			face, err := c.Face(ctx, files[0])
			if err != nil {
				t.Fatalf("Face: %v", err)
			}
			for _, r := range tt.runes {
				if _, ok := face.NominalGlyph(r); !ok {
					t.Errorf("no glyph for %U", r)
				}
			}
		})
	}
}

func TestMatchOrder(t *testing.T) {
	sys := []FontFile{{Path: "/fonts/a.ttf", Family: "A"}, {Path: "/fonts/b.ttf", Index: 2, Family: "B"}}
	tests := []struct {
		name    string
		family  string
		bundled bool
		want    []FontFile
	}{
		{"system fonts only", "sans-serif", true, sys},
		{"the bundled font comes first", "Roboto", true, append([]FontFile{{Path: BundledPrefix + "Roboto", Family: "Roboto"}}, sys...)},
		{"family names ignore case and blanks", "ROBOTO", true, append([]FontFile{{Path: BundledPrefix + "ROBOTO", Family: "ROBOTO"}}, sys...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &fakeMatcher{files: sys}
			c := bundledCollection(t, CollectionOpts{Matcher: m})
			got, err := c.Match(context.Background(), tt.family, 400, false, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Match = %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("file %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
			if len(m.calls) != 1 || m.calls[0] != tt.family {
				t.Errorf("matcher calls = %v, want one for %q", m.calls, tt.family)
			}
		})
	}
}

func TestMatchSkipsFormatsGoTextCannotRead(t *testing.T) {
	m := &fakeMatcher{files: []FontFile{
		{Path: "/fonts/a.t1"}, {Path: "/fonts/b.TTF"}, {Path: "/fonts/c.pcf.gz"}, {Path: "/fonts/d.ttc", Index: 1},
		{Path: "/fonts/e.otf"}, {Path: "/fonts/f.pfb"}, {Path: "/fonts/g.bdf"}, {Path: "/fonts/noext"},
	}}
	got, err := NewCollection(CollectionOpts{Matcher: m}).Match(context.Background(), "serif", 400, false, "")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	want := []string{"/fonts/b.TTF", "/fonts/d.ttc", "/fonts/e.otf"}
	if !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestFaceCache(t *testing.T) {
	c := bundledCollection(t)
	ctx := context.Background()
	f := FontFile{Path: BundledPrefix + "Roboto"}
	bold := Variation{Tag: "wght", Value: 700}
	width := Variation{Tag: "wdth", Value: 75}
	get := func(vs ...Variation) *font.Face {
		t.Helper()
		face, err := c.Face(ctx, f, vs...)
		if err != nil {
			t.Fatal(err)
		}
		return face
	}
	plain := get()
	if get() != plain {
		t.Error("the same request gave another face")
	}
	if get(bold) == plain {
		t.Error("a variation gave the face without it")
	}
	if get(bold, width) != get(width, bold) {
		t.Error("the order of the variations changed the face")
	}
	if get(Variation{Tag: "wght", Value: 100}, bold) != get(bold) {
		t.Error("a later value of an axis did not win")
	}
	// The variation moves the glyph advances: bold is wider than regular.
	gid, _ := plain.NominalGlyph('M')
	if regular, heavy := plain.HorizontalAdvance(gid), get(bold).HorizontalAdvance(gid); heavy <= regular {
		t.Errorf("advance of M: bold %v, regular %v; the weight axis had no effect", heavy, regular)
	}
	// Faces of one font share the parsed font.
	if get(bold).Font != plain.Font {
		t.Error("faces of one font do not share the parsed font")
	}
}

func TestFaceFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icons.ttf")
	if err := os.WriteFile(path, fonts.MaterialIcons(), 0o600); err != nil {
		t.Fatal(err)
	}
	c := NewCollection()
	face, err := c.Face(context.Background(), FontFile{Path: path})
	if err != nil {
		t.Fatalf("Face: %v", err)
	}
	if _, ok := face.NominalGlyph(0xe145); !ok {
		t.Error("the icon glyph is missing")
	}
	// The high 16 bits of fontconfig's index (a named instance) do not select a face.
	again, err := c.Face(context.Background(), FontFile{Path: path, Index: 4 << 16})
	if err != nil || again != face {
		t.Errorf("a named instance index gave another face (err %v)", err)
	}
}

func TestFailures(t *testing.T) {
	dir := t.TempDir()
	notFont := filepath.Join(dir, "notes.ttf")
	if err := os.WriteFile(notFont, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	icons := filepath.Join(dir, "icons.ttf")
	if err := os.WriteFile(icons, fonts.MaterialIcons(), 0o600); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	ctx := context.Background()

	tests := []struct {
		name  string
		run   func(c *Collection) error
		route string
	}{
		{"missing file", func(c *Collection) error {
			_, err := c.Face(ctx, FontFile{Path: filepath.Join(dir, "missing.ttf")})
			return err
		}, "Collection.Face->Collection.load.FontLoadFailed"},
		{"file is not a font", func(c *Collection) error {
			_, err := c.Face(ctx, FontFile{Path: notFont})
			return err
		}, "Collection.Face->Collection.load.FontLoadFailed"},
		{"no face at the index", func(c *Collection) error {
			_, err := c.Face(ctx, FontFile{Path: icons, Index: 3})
			return err
		}, "Collection.Face->Collection.load.FontLoadFailed"},
		{"unknown bundled family", func(c *Collection) error {
			_, err := c.Face(ctx, FontFile{Path: BundledPrefix + "Nope"})
			return err
		}, "Collection.Face->Collection.load.FontLoadFailed"},
		{"bad variation tag", func(c *Collection) error {
			_, err := c.Face(ctx, FontFile{Path: icons}, Variation{Tag: "wg", Value: 1})
			return err
		}, "Collection.Face.FontVariationInvalid"},
		{"face canceled", func(c *Collection) error {
			_, err := c.Face(canceled, FontFile{Path: icons})
			return err
		}, "Collection.Face.FontLoadCanceled"},
		{"match canceled", func(c *Collection) error {
			_, err := c.Match(canceled, "sans-serif", 400, false, "")
			return err
		}, "Collection.Match.FontMatchCanceled"},
		{"bundled data is not a font", func(c *Collection) error {
			return c.AddBundled("Bad", []byte("nope"))
		}, "Collection.AddBundled.FontBundledInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(NewCollection(CollectionOpts{Matcher: &fakeMatcher{}}))
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}

	t.Run("matcher error comes back unchanged", func(t *testing.T) {
		want := errors.New("MatcherBroke", "the matcher failed")
		c := NewCollection(CollectionOpts{Matcher: &fakeMatcher{err: want}})
		_, err := c.Match(ctx, "sans-serif", 400, false, "")
		if err != want {
			t.Errorf("Match error = %v, want the matcher's error", err)
		}
	})
	t.Run("font scan cannot make its cache directory", func(t *testing.T) {
		c := NewCollection(CollectionOpts{CacheDir: filepath.Join(notFont, "cache")})
		_, err := c.Match(ctx, "sans-serif", 400, false, "")
		if appErr := routeOf(t, err); !appErr.HasRoute("Collection.Match->Collection.matcher->newScanMatcher.FontScanFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
	t.Run("two options values panic", func(t *testing.T) {
		defer func() {
			appErr := routeOf(t, recover().(error))
			if appErr.Tag != "MultipleOptions" {
				t.Errorf("tag = %s, want MultipleOptions", appErr.Tag)
			}
		}()
		NewCollection(CollectionOpts{}, CollectionOpts{})
	})
}

func TestScanMatcher(t *testing.T) {
	c := NewCollection()
	ctx := context.Background()
	files, err := c.Match(ctx, "sans-serif", 400, false, "")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(files) == 0 {
		t.Skip("the system has no fonts that fontscan found")
	}
	for _, f := range files {
		if _, err := c.Face(ctx, f); err != nil {
			t.Errorf("Face(%+v): %v", f, err)
		}
	}
	// The index is kept in $XDG_CACHE_HOME/glutter/fonts.
	matches, _ := filepath.Glob(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "glutter", "fonts", "font_index_v*.cache"))
	if len(matches) == 0 {
		t.Error("the font index was not written to the cache directory")
	}

	// A family the system has comes back first, in the weight asked for.
	s := c.scan
	var family string
	for _, fp := range s.footprints {
		if fp.Aspect.Weight == font.WeightBold {
			family = fp.Family
			break
		}
	}
	if family == "" {
		t.Skip("no bold font on the system")
	}
	got, err := c.Match(ctx, family, 700, false, "")
	if err != nil || len(got) == 0 {
		t.Fatalf("Match(%q): %v, %d files", family, err, len(got))
	}
	for _, fp := range s.footprints {
		if fp.Location.File == got[0].Path && int(fp.Location.Index) == got[0].Index {
			if fp.Family != family || fp.Aspect.Weight != font.WeightBold {
				t.Errorf("first match is %q at weight %v, want %q bold", fp.Family, fp.Aspect.Weight, family)
			}
		}
	}
}

func TestAspectDistance(t *testing.T) {
	regular := font.Aspect{Style: font.StyleNormal, Weight: 400}
	tests := []struct {
		name       string
		have, want font.Aspect
		less       font.Aspect // distance to want must be smaller than that of have
	}{
		{"closer weight wins", font.Aspect{Style: font.StyleNormal, Weight: 500}, regular, font.Aspect{Style: font.StyleNormal, Weight: 700}},
		{"same slant beats a closer weight", font.Aspect{Style: font.StyleNormal, Weight: 900}, regular, font.Aspect{Style: font.StyleItalic, Weight: 400}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if a, b := aspectDistance(tt.have, tt.want), aspectDistance(tt.less, tt.want); a >= b {
				t.Errorf("distance %v >= %v", a, b)
			}
		})
	}
}

func TestVariationsKey(t *testing.T) {
	tests := []struct {
		name string
		a, b []Variation
		same bool
	}{
		{"none", nil, []Variation{}, true},
		{"order", []Variation{{"wght", 700}, {"wdth", 75}}, []Variation{{"wdth", 75}, {"wght", 700}}, true},
		{"later wins", []Variation{{"wght", 100}, {"wght", 700}}, []Variation{{"wght", 700}}, true},
		{"different value", []Variation{{"wght", 700}}, []Variation{{"wght", 600}}, false},
		{"different axis", []Variation{{"wght", 700}}, []Variation{{"wdth", 700}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := variationsKey(tt.a) == variationsKey(tt.b); got != tt.same {
				t.Errorf("equal keys = %v, want %v", got, tt.same)
			}
		})
	}
}
