//go:build windows && glutter_wine

package windows

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/text/winfonts"
)

// TestWineSystemFontsListsTheFontsOfThePrefix reads the fonts that the prefix of Wine lists in its
// registry (the fonts of the host, as Z:\ paths) and has in %WINDIR%\Fonts (Liberation Sans, put
// there by scripts/wine-exec.sh), and keeps what it learned in the cache directory of the user.
func TestWineSystemFontsListsTheFontsOfThePrefix(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LocalAppData", local)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	faces := SystemFonts(context.Background(), log)
	if len(faces) == 0 {
		t.Fatal("no installed font was found")
	}
	byFamily := map[string][]winfonts.Face{}
	for _, f := range faces {
		byFamily[f.Family] = append(byFamily[f.Family], f)
		if _, err := os.Stat(f.Path); err != nil {
			t.Errorf("%s: the file of the face is not there: %v", f.Family, err)
		}
	}
	liberation := byFamily["Liberation Sans"]
	if len(liberation) == 0 {
		t.Fatalf("Liberation Sans is not among the %d faces; the families are %d", len(faces), len(byFamily))
	}
	for _, f := range liberation {
		if !winfonts.SupportedFile(f.Path) {
			t.Errorf("a face of an unsupported file: %+v", f)
		}
	}
	cacheFile := filepath.Join(local, "glutter", "fonts", "windows-faces.json")
	if st, err := os.Stat(cacheFile); err != nil || st.Size() == 0 {
		t.Fatalf("the cache %s was not written: %v", cacheFile, err)
	}

	// The second start reads the same faces, from the cache.
	again := SystemFonts(context.Background(), log)
	if len(again) != len(faces) {
		t.Errorf("the second read found %d faces, the first %d", len(again), len(faces))
	}
	cache := winfonts.LoadCache(cacheFile)
	if cache.Dirty() {
		t.Error("a loaded cache is dirty")
	}

	m, err := winfonts.New(winfonts.Options{Faces: faces})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	got, err := m.Match(ctx, "Liberation Sans", 400, false, "")
	if err != nil || len(got) != 1 || got[0].Family != "Liberation Sans" {
		t.Fatalf("Match(Liberation Sans) = %+v, %v", got, err)
	}
	if bold, _ := m.Match(ctx, "Liberation Sans", 700, false, ""); len(bold) != 1 || !strings.Contains(strings.ToLower(filepath.Base(bold[0].Path)), "bold") {
		t.Errorf("Match(Liberation Sans, 700) = %+v: want the bold face", bold)
	}
	// A generic family finds a font even where Windows' own are not installed: Liberation Sans stands in.
	if sans, err := m.Match(ctx, "sans-serif", 400, false, ""); err != nil || len(sans) == 0 {
		t.Errorf("Match(sans-serif) = %+v, %v", sans, err)
	}
	// Segoe UI, Tahoma and Consolas are Windows' fonts: the prefix may or may not have them, and the
	// generic families are the first of them that is there.
	for _, name := range []string{"Tahoma", "Segoe UI", "Consolas"} {
		fonts, _ := m.Match(ctx, name, 400, false, "")
		t.Logf("%s: %d faces in this prefix", name, len(fonts))
	}
	if _, ok, err := m.Fallback(ctx, "sans-serif", 400, false, "", 'a'); err != nil {
		t.Errorf("Fallback: %v", err)
	} else {
		t.Logf("a font for 'a' in the chain: %v (the chain is the fonts of Windows)", ok)
	}
}
