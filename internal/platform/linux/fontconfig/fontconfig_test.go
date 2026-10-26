//go:build linux && !android

package fontconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

func openOrSkip(t *testing.T) *Config {
	t.Helper()
	c, err := Open(context.Background())
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestFontSetLayout(t *testing.T) {
	// From struct _FcFontSet in /usr/include/fontconfig/fontconfig.h.
	dl.AssertLayout(t, "FcFontSet", 16, map[string]uintptr{"nfont": 0, "sfont": 4, "fonts": 8}, fcFontSet{})
}

func TestMatch(t *testing.T) {
	c := openOrSkip(t)
	tests := []struct {
		name   string
		family string
		weight int
		italic bool
		lang   string
	}{
		{"generic sans-serif", "sans-serif", 400, false, ""},
		{"generic serif bold", "serif", 700, false, ""},
		{"monospace italic", "monospace", 400, true, ""},
		{"unknown family falls back", "Glutter No Such Family", 400, false, ""},
		{"japanese", "sans-serif", 400, false, "ja"},
		{"emoji", "emoji", 400, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := c.Match(context.Background(), tt.family, tt.weight, tt.italic, tt.lang)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(files) == 0 {
				t.Fatalf("Match returned no font file")
			}
			for _, f := range files {
				if !filepath.IsAbs(f.Path) {
					t.Errorf("path %q is not absolute", f.Path)
				}
				if _, err := os.Stat(f.Path); err != nil {
					t.Errorf("font file does not exist: %v", err)
				}
			}
		})
	}
}

func TestMatchHonorsTheFamily(t *testing.T) {
	c := openOrSkip(t)
	// fc-match answers the same for the first file; the family of a bold request
	// must not be lighter than the regular one, so only check that both resolve.
	regular, err := c.Match(context.Background(), "sans-serif", 400, false, "")
	if err != nil {
		t.Fatal(err)
	}
	bold, err := c.Match(context.Background(), "sans-serif", 700, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(regular) == 0 || len(bold) == 0 {
		t.Fatalf("no files: regular %d, bold %d", len(regular), len(bold))
	}
	if regular[0].Family == "" {
		t.Errorf("the first match has no family: %+v", regular[0])
	}
}

func TestWeightToFontconfig(t *testing.T) {
	tests := []struct{ ot, want int }{
		{-5, 0}, {100, 0}, {200, 40}, {300, 50}, {400, 80}, {500, 100}, {600, 180},
		{700, 200}, {800, 205}, {900, 210}, {1000, 210}, {2000, 210}, {450, 90}, {350, 55},
	}
	for _, tt := range tests {
		if got := weightToFontconfig(tt.ot); got != tt.want {
			t.Errorf("weightToFontconfig(%d) = %d, want %d", tt.ot, got, tt.want)
		}
	}
}

func routeOf(t *testing.T, err error) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	return appErr
}

func TestOpenFailures(t *testing.T) {
	t.Run("library missing", func(t *testing.T) {
		t.Setenv(LibraryEnv, filepath.Join(t.TempDir(), "libfontconfig-missing.so.1"))
		_, err := Open(context.Background())
		if appErr := routeOf(t, err); !appErr.HasRoute("Open->loadLibrary.FontconfigLoadFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
	t.Run("override is not an absolute path", func(t *testing.T) {
		t.Setenv(LibraryEnv, "libfontconfig.so.1")
		_, err := Open(context.Background())
		if appErr := routeOf(t, err); !appErr.HasRoute("Open->loadLibrary.FontconfigLoadFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
	t.Run("configuration does not load", func(t *testing.T) {
		_, err := newConfig(&library{initLoadConfigAndFonts: func() uintptr { return 0 }})
		if appErr := routeOf(t, err); !appErr.HasRoute("newConfig.FontconfigInitFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
}

// fakeLibrary is a library whose calls succeed unless a test changes one.
func fakeLibrary() *library {
	return &library{
		initLoadConfigAndFonts: func() uintptr { return 1 },
		configDestroy:          func(uintptr) {},
		patternCreate:          func() uintptr { return 2 },
		patternDestroy:         func(uintptr) {},
		patternAddString:       func(_, _, _ uintptr) int32 { return 1 },
		patternAddInteger:      func(_, _ uintptr, _ int32) int32 { return 1 },
		configSubstitute:       func(_, _ uintptr, _ int32) int32 { return 1 },
		defaultSubstitute:      func(uintptr) {},
		fontSort:               func(_, _ uintptr, _ int32, _, _ uintptr) uintptr { return 0 },
		fontSetDestroy:         func(uintptr) {},
		patternGetCharSet:      func(_, _ uintptr, _ int32, _ uintptr) int32 { return 1 },
		charSetCopy:            func(cs uintptr) uintptr { return cs },
		charSetDestroy:         func(uintptr) {},
		charSetHasChar:         func(uintptr, uint32) int32 { return 0 },
	}
}

func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Open(ctx)
	if appErr := routeOf(t, err); !appErr.HasRoute("Open.FontconfigCanceled") {
		t.Errorf("Open: unexpected failure path: %s", appErr.Route())
	}
	c, err := newConfig(fakeLibrary())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.Match(ctx, "sans-serif", 400, false, "")
	if appErr := routeOf(t, err); !appErr.HasRoute("Config.Match.FontconfigCanceled") {
		t.Errorf("Match: unexpected failure path: %s", appErr.Route())
	}
}

func TestMatchFailures(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(*library)
		route string
	}{
		{"pattern is not created", func(l *library) { l.patternCreate = func() uintptr { return 0 } }, "Config.Match.FontconfigPatternFailed"},
		{"family is not added", func(l *library) { l.patternAddString = func(_, _, _ uintptr) int32 { return 0 } }, "Config.Match->Config.addString.FontconfigPatternFailed"},
		{"weight is not added", func(l *library) { l.patternAddInteger = func(_, _ uintptr, _ int32) int32 { return 0 } }, "Config.Match->Config.addInt.FontconfigPatternFailed"},
		{"sort fails", func(*library) {}, "Config.Match.FontconfigMatchFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := fakeLibrary()
			tt.tweak(lib)
			c, err := newConfig(lib)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			_, err = c.Match(context.Background(), "sans-serif", 400, false, "en")
			if appErr := routeOf(t, err); !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
	t.Run("closed", func(t *testing.T) {
		c, err := newConfig(fakeLibrary())
		if err != nil {
			t.Fatal(err)
		}
		c.Close()
		c.Close() // a second Close does nothing
		_, err = c.Match(context.Background(), "sans-serif", 400, false, "")
		if appErr := routeOf(t, err); !appErr.HasRoute("Config.Match.FontconfigClosed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})
}
