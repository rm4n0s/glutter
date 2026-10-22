package layout

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// listingMatcher is a fontconfigLike matcher that also lists its families.
type listingMatcher struct {
	fontconfigLike
	families []string
	err      error
}

func (m listingMatcher) Families(context.Context) ([]string, error) { return m.families, m.err }

func queryEngine(t *testing.T, m text.Matcher) *Engine {
	t.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	return NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) { t.Errorf("font error: %v", err) }})
}

func TestResolveFont(t *testing.T) {
	sys := systemFontFiles(t)
	e := queryEngine(t, sys)
	for _, data := range [][]byte{gomonobold.TTF, gomonoitalic.TTF} {
		if err := e.LoadFontFromList(data, "Loaded Mono"); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name   string
		family string
		weight ui.FontWeight
		style  ui.FontStyle
		want   ui.ResolvedFont
		ok     bool
	}{
		{"a loaded family, the bold request", "Loaded Mono", ui.FontWeightW700, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Loaded Mono", Origin: ui.FontOriginLoaded, Weight: 600}, true}, // Go Mono Bold says 600
		{"a loaded family, the italic request", "loaded mono", ui.FontWeightW400, ui.FontStyleItalic,
			ui.ResolvedFont{Family: "Loaded Mono", Origin: ui.FontOriginLoaded, Weight: 400, Italic: true}, true},
		{"a loaded family, a zero weight is regular, and a slant that is wrong counts more than a weight", "Loaded Mono", 0, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Loaded Mono", Origin: ui.FontOriginLoaded, Weight: 600}, true},
		{"a bundled family", "Roboto", ui.FontWeightW400, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Roboto", Origin: ui.FontOriginBundled, Weight: 400}, true},
		{"a system family", "Go Mono", ui.FontWeightW400, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Go Mono", Path: sys.mono.Path, Origin: ui.FontOriginSystem, Weight: 400}, true},
		{"a generic family", "monospace", ui.FontWeightW400, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Go Mono", Path: sys.mono.Path, Origin: ui.FontOriginSystem, Weight: 400}, true},
		{"an alias", "Arial", ui.FontWeightW400, ui.FontStyleNormal,
			ui.ResolvedFont{Family: "Go Mono", Path: sys.mono.Path, Origin: ui.FontOriginSystem, Weight: 400}, true},
		{"a family nothing has", "Missing", ui.FontWeightW400, ui.FontStyleNormal, ui.ResolvedFont{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := e.ResolveFont(tt.family, tt.weight, tt.style)
			if ok != tt.ok || got != tt.want {
				t.Errorf("ResolveFont(%q) = %+v, %v; want %+v, %v", tt.family, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestResolveFontReportsAFontThatDoesNotLoad(t *testing.T) {
	bad := text.FontFile{Path: t.TempDir() + "/none.ttf", Family: "Broken"}
	var reported []error
	c := text.NewCollection(text.CollectionOpts{Matcher: fontconfigLike{sans: bad, mono: bad}})
	e := NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) { reported = append(reported, err) }})
	if got, ok := e.ResolveFont("sans-serif", 400, ui.FontStyleNormal); ok {
		t.Errorf("ResolveFont = %+v, want none", got)
	}
	if len(reported) == 0 {
		t.Fatal("the failure to load the font was not reported")
	}
	if appErr, ok := errors.FromError(reported[0]); !ok || appErr.Tag != "FontLoadFailed" {
		t.Errorf("reported %v, want FontLoadFailed", reported[0])
	}
}

func TestInstalledFontFamilies(t *testing.T) {
	sys := systemFontFiles(t)
	tests := []struct {
		name    string
		matcher text.Matcher
		load    string
		want    []string
	}{
		{"a matcher that cannot list", sys, "", []string{"MaterialIcons", "Roboto"}},
		{"a matcher that lists", listingMatcher{fontconfigLike: sys, families: []string{"Zeta", "go mono", "Alpha"}}, "",
			[]string{"Alpha", "go mono", "MaterialIcons", "Roboto", "Zeta"}},
		{"duplicates in any case, and the loaded fonts", listingMatcher{fontconfigLike: sys, families: []string{"Roboto", "ROBOTO", "Mono"}}, "Terminal",
			[]string{"MaterialIcons", "Mono", "Roboto", "Terminal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := queryEngine(t, tt.matcher)
			if tt.load != "" {
				if err := e.LoadFontFromList(gomono.TTF, tt.load); err != nil {
					t.Fatal(err)
				}
			}
			got := e.InstalledFontFamilies()
			if len(got) != len(tt.want) {
				t.Fatalf("InstalledFontFamilies = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("family %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestInstalledFontFamiliesWhenTheListingFails(t *testing.T) {
	var reported []error
	m := listingMatcher{fontconfigLike: systemFontFiles(t), err: errors.New("ListBroken", "no")}
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) { reported = append(reported, err) }})
	got := e.InstalledFontFamilies()
	if len(got) != 2 || got[0] != "MaterialIcons" || got[1] != "Roboto" {
		t.Errorf("InstalledFontFamilies = %v, want the bundled families", got)
	}
	if len(reported) != 1 {
		t.Errorf("reported %v, want the listing error", reported)
	}
}
