//go:build linux && !android

package fontconfig

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// aliasLibrary is a fake library that answers FC_FAMILY queries from a table: the
// pattern's families are those of the family that was added last.
func aliasLibrary(table map[string][]string) *library {
	lib := fakeLibrary()
	var family string
	lib.patternAddString = func(_, _, value uintptr) int32 {
		family = dl.GoString(value)
		return 1
	}
	lib.patternGetString = func(_, _ uintptr, n int32, out uintptr) int32 {
		names := table[family]
		if int(n) >= len(names) {
			return 1
		}
		*(*uintptr)(dl.Pointer(out)) = dl.CString(names[n]) // freed with the process: it is a test
		return fcResultMatch
	}
	return lib
}

func TestAliases(t *testing.T) {
	table := map[string][]string{
		probeFamily:   {probeFamily, "sans-serif", "DejaVu Sans"},
		"Arial":       {"Arial", "Liberation Sans", "sans-serif", "DejaVu Sans", "Arimo", "liberation sans"},
		"Courier New": {"Courier New", "Liberation Mono", "monospace"},
		"Nothing":     {"Nothing", "sans-serif", "DejaVu Sans"},
		"arial":       {"arial"},
	}
	tests := []struct {
		family string
		want   []string
	}{
		{"Arial", []string{"Liberation Sans", "Arimo"}},
		{"Courier New", []string{"Liberation Mono", "monospace"}},
		{"Nothing", nil},
		{"arial", nil},
		{"Unknown", nil},
	}
	c, err := newConfig(aliasLibrary(table))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			got, err := c.Aliases(context.Background(), tt.family)
			if err != nil {
				t.Fatalf("Aliases: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Aliases(%q) = %v, want %v", tt.family, got, tt.want)
			}
		})
	}
}

func TestAliasesFailures(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(*library)
		route string
	}{
		{"pattern is not created", func(l *library) { l.patternCreate = func() uintptr { return 0 } }, "Config.Aliases->Config.substitutedFamilies.FontconfigPatternFailed"},
		{"family is not added", func(l *library) { l.patternAddString = func(_, _, _ uintptr) int32 { return 0 } }, "Config.Aliases->Config.substitutedFamilies->Config.addString.FontconfigPatternFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := aliasLibrary(nil)
			tt.tweak(lib)
			c, err := newConfig(lib)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			_, err = c.Aliases(context.Background(), "Arial")
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c, err := newConfig(fakeLibrary())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_, err = c.Aliases(ctx, "Arial")
		if appErr := routeOf(t, err); !appErr.HasRoute("Config.Aliases.FontconfigCanceled") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
	t.Run("closed", func(t *testing.T) {
		c, err := newConfig(fakeLibrary())
		if err != nil {
			t.Fatal(err)
		}
		c.Close()
		_, err = c.Aliases(context.Background(), "Arial")
		if appErr := routeOf(t, err); !appErr.HasRoute("Config.Aliases.FontconfigClosed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
}

func TestAliasesWithTheRealLibrary(t *testing.T) {
	c := openOrSkip(t)
	got, err := c.Aliases(context.Background(), "Glutter No Such Family")
	if err != nil {
		t.Fatalf("Aliases: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a family that nothing mentions has aliases: %v", got)
	}
}

func TestFamilies(t *testing.T) {
	c := openOrSkip(t)
	got, err := c.Families(context.Background())
	if err != nil {
		t.Fatalf("Families: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("the system has no font family")
	}
	for i, f := range got {
		if f == "" {
			t.Errorf("family %d is empty", i)
		}
		if i > 0 && strings.ToLower(got[i-1]) >= strings.ToLower(f) {
			t.Errorf("families %q and %q are not sorted without duplicates", got[i-1], f)
		}
	}
}

func TestFamiliesFailures(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(*library)
		route string
	}{
		{"no pattern", func(l *library) { l.patternCreate = func() uintptr { return 0 } }, "Config.Families.FontconfigPatternFailed"},
		{"no object set", func(l *library) { l.objectSetCreate = func() uintptr { return 0 } }, "Config.Families.FontconfigPatternFailed"},
		{"the family is not added", func(l *library) { l.objectSetAdd = func(_, _ uintptr) int32 { return 0 } }, "Config.Families.FontconfigPatternFailed"},
		{"no list", func(l *library) { l.fontList = func(_, _, _ uintptr) uintptr { return 0 } }, "Config.Families.FontconfigListFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := fakeLibrary()
			lib.objectSetCreate = func() uintptr { return 3 }
			lib.objectSetAdd = func(_, _ uintptr) int32 { return 1 }
			lib.objectSetDestroy = func(uintptr) {}
			lib.fontList = func(_, _, _ uintptr) uintptr { return 0 }
			tt.tweak(lib)
			c, err := newConfig(lib)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			_, err = c.Families(context.Background())
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c, err := newConfig(fakeLibrary())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_, err = c.Families(ctx)
		if appErr := routeOf(t, err); !appErr.HasRoute("Config.Families.FontconfigCanceled") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
	t.Run("closed", func(t *testing.T) {
		c, err := newConfig(fakeLibrary())
		if err != nil {
			t.Fatal(err)
		}
		c.Close()
		_, err = c.Families(context.Background())
		if appErr := routeOf(t, err); !appErr.HasRoute("Config.Families.FontconfigClosed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
}
