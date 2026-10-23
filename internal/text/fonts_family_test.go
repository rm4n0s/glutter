package text

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
)

func TestIsGenericFamily(t *testing.T) {
	tests := []struct {
		family string
		want   bool
	}{
		{"sans-serif", true}, {"serif", true}, {"monospace", true}, {"cursive", true}, {"fantasy", true},
		{"emoji", true}, {"math", true}, {"Monospace", true}, {" SERIF ", true},
		{"Roboto", false}, {"", false}, {"Noto Sans Mono", false}, {"mono", false}, {"sans", false},
	}
	for _, tt := range tests {
		if got := IsGenericFamily(tt.family); got != tt.want {
			t.Errorf("IsGenericFamily(%q) = %v, want %v", tt.family, got, tt.want)
		}
	}
}

// aliasMatcher is a fakeMatcher that also knows aliases, as fontconfig does.
type aliasMatcher struct {
	fakeMatcher
	aliases map[string][]string
	err     error
	asked   []string
}

func (m *aliasMatcher) Aliases(_ context.Context, family string) ([]string, error) {
	m.asked = append(m.asked, family)
	return m.aliases[family], m.err
}

func TestMatchGivesOnlyTheFontsOfTheFamily(t *testing.T) {
	sys := []FontFile{
		{Path: "/fonts/sans.ttf", Family: "Noto Sans"}, // the default font, which fontconfig puts first for every family
		{Path: "/fonts/mono.ttf", Family: "Fira Code"},
		{Path: "/fonts/mono-bold.ttf", Family: "Fira Code"},
		{Path: "/fonts/lib.ttf", Family: "Liberation Sans"},
		{Path: "/fonts/arimo.ttf", Family: "Arimo"},
	}
	paths := func(files []FontFile) []string {
		var out []string
		for _, f := range files {
			out = append(out, f.Path)
		}
		return out
	}
	tests := []struct {
		name    string
		family  string
		aliases map[string][]string
		want    []string
	}{
		{"a family that is installed", "Fira Code", nil, []string{"/fonts/mono.ttf", "/fonts/mono-bold.ttf"}},
		{"names ignore case and blanks", "fira code", nil, []string{"/fonts/mono.ttf", "/fonts/mono-bold.ttf"}},
		{"a family with no font gives nothing, not the default font", "Missing Mono", nil, nil},
		{"generic monospace gives the whole list", "monospace", nil, paths(sys)},
		{"generic sans-serif gives the whole list", "sans-serif", nil, paths(sys)},
		{"generic serif gives the whole list", "Serif", nil, paths(sys)},
		{"an alias", "Arial", map[string][]string{"Arial": {"Liberation Sans", "Arimo"}}, []string{"/fonts/lib.ttf", "/fonts/arimo.ttf"}},
		{"an alias of a family that has no font", "Arial", map[string][]string{"Arial": {"Nothing"}}, nil},
		{"aliases are not asked for a generic family", "monospace", map[string][]string{"monospace": {"Arimo"}}, paths(sys)},
		{"the family itself and its aliases", "Fira Code", map[string][]string{"Fira Code": {"Arimo"}}, []string{"/fonts/mono.ttf", "/fonts/mono-bold.ttf", "/fonts/arimo.ttf"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &aliasMatcher{fakeMatcher: fakeMatcher{files: sys}, aliases: tt.aliases}
			c := NewCollection(CollectionOpts{Matcher: m})
			got, err := c.Match(context.Background(), tt.family, 400, false, "")
			if err != nil {
				t.Fatal(err)
			}
			gp := paths(got)
			if len(gp) != len(tt.want) {
				t.Fatalf("Match = %v, want %v", gp, tt.want)
			}
			for i := range gp {
				if gp[i] != tt.want[i] {
					t.Errorf("file %d = %s, want %s", i, gp[i], tt.want[i])
				}
			}
			if IsGenericFamily(tt.family) && len(m.asked) != 0 {
				t.Errorf("the matcher was asked for the aliases of a generic family: %v", m.asked)
			}
		})
	}
}

func TestMatchWithoutAnAliaser(t *testing.T) {
	m := &fakeMatcher{files: []FontFile{{Path: "/fonts/lib.ttf", Family: "Liberation Sans"}}}
	got, err := NewCollection(CollectionOpts{Matcher: m}).Match(context.Background(), "Arial", 400, false, "")
	if err != nil || len(got) != 0 {
		t.Errorf("Match = %v, %v; a matcher that cannot say what Arial stands for gives no font", got, err)
	}
}

func TestMatchAsksForAliasesOnce(t *testing.T) {
	m := &aliasMatcher{fakeMatcher: fakeMatcher{files: []FontFile{{Path: "/fonts/lib.ttf", Family: "Liberation Sans"}}},
		aliases: map[string][]string{"Arial": {"Liberation Sans"}}}
	c := NewCollection(CollectionOpts{Matcher: m})
	for _, weight := range []int{400, 700, 400} {
		if got, err := c.Match(context.Background(), "Arial", weight, false, ""); err != nil || len(got) != 1 {
			t.Fatalf("Match = %v, %v", got, err)
		}
	}
	if len(m.asked) != 1 {
		t.Errorf("the matcher was asked %d times for the aliases of Arial, want 1", len(m.asked))
	}
}

func TestMatchAliasesFailure(t *testing.T) {
	m := &aliasMatcher{fakeMatcher: fakeMatcher{files: []FontFile{{Path: "/fonts/a.ttf", Family: "A"}}},
		err: errors.New(tagAliasLookupBroken, "the matcher cannot say")}
	_, err := NewCollection(CollectionOpts{Matcher: m}).Match(context.Background(), "Arial", 400, false, "")
	if appErr := routeOf(t, err); appErr.Tag != tagAliasLookupBroken || !appErr.HasRoute(tagAliasLookupBroken) {
		t.Errorf("err = %v, want the matcher's own error unchanged", err)
	}
}

func TestMatchLoadedFontsComeBeforeTheSystemOnesOfTheSameFamily(t *testing.T) {
	m := &fakeMatcher{files: []FontFile{{Path: "/fonts/mono.ttf", Family: "Go Mono"}}}
	c := NewCollection(CollectionOpts{Matcher: m})
	if err := c.AddFont("Go Mono", gomono.TTF); err != nil {
		t.Fatal(err)
	}
	got, err := c.Match(context.Background(), "Go Mono", 400, false, "")
	if err != nil || len(got) != 2 || !got[0].IsLoaded() || got[1].Path != "/fonts/mono.ttf" {
		t.Errorf("Match = %+v, %v", got, err)
	}
}

func TestScanMatcherAliases(t *testing.T) {
	if systemFontsSkip != "" {
		t.Skip(systemFontsSkip)
	}
	s, err := newScanMatcher("", nil)
	if err != nil {
		t.Skipf("the font scan is not available: %v", err)
	}
	got, err := s.Aliases(context.Background(), "Glutter No Such Family")
	if err != nil || len(got) != 0 {
		t.Errorf("Aliases of a family nothing mentions = %v, %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Aliases(ctx, "Arial")
	if appErr := routeOf(t, err); !appErr.HasRoute("scanMatcher.Aliases.FontMatchCanceled") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// listingMatcher is a fakeMatcher that also lists families.
type listingMatcher struct {
	fakeMatcher
	families []string
	err      error
}

func (m *listingMatcher) Families(context.Context) ([]string, error) { return m.families, m.err }

func TestCollectionFamilies(t *testing.T) {
	tests := []struct {
		name  string
		match Matcher
		build func(c *Collection)
		want  []string
	}{
		{"nothing", &fakeMatcher{}, func(*Collection) {}, nil},
		{"loaded fonts, in the name they were first given", &fakeMatcher{}, func(c *Collection) {
			_ = c.AddFont("Zed Mono", gomono.TTF)
			_ = c.AddFont("zed mono", gomonobold.TTF)
			_ = c.AddFont("Alpha", gomono.TTF)
		}, []string{"Alpha", "Zed Mono"}},
		{"bundled and loaded", &fakeMatcher{}, func(c *Collection) {
			_ = c.AddBundled("Roboto", gomono.TTF)
			_ = c.AddFont("roboto", gomonobold.TTF)
		}, []string{"roboto"}}, // the loaded font wins, in its name
		{"system families are merged without duplicates", &listingMatcher{families: []string{"Noto Sans", "fira code", "Fira Code", "Arimo"}},
			func(c *Collection) { _ = c.AddFont("Fira Code", gomono.TTF) }, []string{"Arimo", "Fira Code", "Noto Sans"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollection(CollectionOpts{Matcher: tt.match})
			tt.build(c)
			got, err := c.Families(context.Background())
			if err != nil {
				t.Fatalf("Families: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Families = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("family %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCollectionFamiliesFailures(t *testing.T) {
	t.Run("the listing fails: the others are returned with the error", func(t *testing.T) {
		m := &listingMatcher{err: errors.New(tagListBroken, "no")}
		c := NewCollection(CollectionOpts{Matcher: m})
		_ = c.AddFont("Mine", gomono.TTF)
		got, err := c.Families(context.Background())
		if appErr := routeOf(t, err); appErr.Tag != tagListBroken {
			t.Errorf("err = %v, want the matcher's own error", err)
		}
		if len(got) != 1 || got[0] != "Mine" {
			t.Errorf("Families = %v, want [Mine]", got)
		}
	})
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := noSystem(t).Families(ctx)
		if appErr := routeOf(t, err); !appErr.HasRoute("Collection.Families.FontMatchCanceled") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
}

func TestScanMatcherFamilies(t *testing.T) {
	if systemFontsSkip != "" {
		t.Skip(systemFontsSkip)
	}
	s, err := newScanMatcher("", nil)
	if err != nil {
		t.Skipf("the font scan is not available: %v", err)
	}
	got, err := s.Families(context.Background())
	if err != nil {
		t.Fatalf("Families: %v", err)
	}
	if len(got) == 0 {
		t.Skip("the system has no fonts that the scan finds")
	}
	for _, f := range got {
		if f == "" {
			t.Error("an empty family name")
		}
		if f != strings.TrimSpace(f) {
			t.Errorf("family %q has blanks around it", f)
		}
	}
	again, _ := s.Families(context.Background())
	if len(again) != len(got) {
		t.Error("the second call gives another list")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Families(ctx)
	if appErr := routeOf(t, err); !appErr.HasRoute("scanMatcher.Families.FontMatchCanceled") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAliasLookupBroken = "AliasLookupBroken"
	tagListBroken        = "ListBroken"
)
