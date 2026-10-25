package winfonts

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

// The fonts that Material bundles are the real font files of the tests.
const (
	robotoPath = "../../../material/fonts/Roboto-Variable.ttf"
	iconsPath  = "../../../material/fonts/MaterialIcons-Regular.otf"
)

func needFonts(t *testing.T) {
	t.Helper()
	for _, p := range []string{robotoPath, iconsPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("no font file to test with: %v", err)
		}
	}
}

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestDescribeReadsTheFamilyAndTheStyle(t *testing.T) {
	needFonts(t)
	faces, err := Describe(robotoPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(faces) != 1 || faces[0].Family != "Roboto" || faces[0].Path != robotoPath || faces[0].Index != 0 || faces[0].Italic {
		t.Fatalf("faces = %+v", faces)
	}
	if w := faces[0].Weight; w < 100 || w > 900 {
		t.Errorf("weight = %d", w)
	}
	faces, err = Describe(iconsPath)
	if err != nil || len(faces) != 1 || faces[0].Family != "Material Icons" {
		t.Errorf("faces = %+v, %v", faces, err)
	}
}

func TestDescribeOfAFileThatIsNotAFont(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.ttf")
	if err := os.WriteFile(p, []byte("this is not a font"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Describe(p)
	assertRoute(t, err, "WindowsFontUnreadable", "Describe.WindowsFontUnreadable")
}

func TestScanKeepsTheOrderAndReportsWhatItCannotRead(t *testing.T) {
	needFonts(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.ttf")
	if err := os.WriteFile(bad, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	var failed []string
	faces := Scan(context.Background(), []string{iconsPath, bad, robotoPath, filepath.Join(dir, "missing.ttf")}, nil, func(p string, err error) { failed = append(failed, p) })
	if len(faces) != 2 || faces[0].Family != "Material Icons" || faces[1].Family != "Roboto" {
		t.Errorf("faces = %+v", faces)
	}
	if len(failed) != 2 || failed[0] != bad {
		t.Errorf("failed = %v", failed)
	}
	if got := Scan(context.Background(), nil, nil, nil); got != nil {
		t.Errorf("no files gave %v", got)
	}
}

func TestScanWithAnEndedContextReadsNothingMore(t *testing.T) {
	needFonts(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	faces := Scan(ctx, []string{robotoPath, iconsPath, robotoPath, iconsPath}, nil, nil)
	if len(faces) > 4 {
		t.Errorf("%d faces", len(faces))
	}
}

func TestTheCacheSavesReadingTheFilesAgain(t *testing.T) {
	needFonts(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "faces.json")
	cache := LoadCache(file)
	if cache.Dirty() {
		t.Error("a new cache is dirty")
	}
	files := []string{robotoPath, iconsPath}
	first := Scan(context.Background(), files, cache, nil)
	if !cache.Dirty() {
		t.Error("the cache learned nothing")
	}
	keep := func(p string) bool { return p == robotoPath || p == iconsPath }
	if err := cache.Save(file, keep); err != nil {
		t.Fatal(err)
	}
	if cache.Dirty() {
		t.Error("the cache is dirty after Save")
	}

	// A second run reads it from the file: a failing reader would show; a cached face is not read.
	loaded := LoadCache(file)
	second := Scan(context.Background(), files, loaded, func(p string, err error) { t.Errorf("read %s again: %v", p, err) })
	if !reflect.DeepEqual(first, second) {
		t.Errorf("the cache gave %+v, the files %+v", second, first)
	}
	if loaded.Dirty() {
		t.Error("nothing was learned, but the cache is dirty")
	}
	// A file with another stamp is read again.
	loaded.entries[robotoPath] = cacheEntry{Stamp: Stamp{Size: 1}, Faces: []Face{{Family: "Stale", Path: robotoPath}}}
	third := Scan(context.Background(), files, loaded, nil)
	if third[0].Family != "Roboto" {
		t.Errorf("a stale entry was used: %+v", third)
	}
	// Files that are gone leave the file.
	if err := loaded.Save(file, func(p string) bool { return p == iconsPath }); err != nil {
		t.Fatal(err)
	}
	if got := LoadCache(file); len(got.entries) != 1 {
		t.Errorf("%d entries after dropping one file", len(got.entries))
	}
}

func TestADamagedCacheIsEmpty(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{"garbage": "not json", "other version": `{"Version":99,"Entries":{"a":{}}}`, "no entries": `{"Version":1}`} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if c := LoadCache(p); len(c.entries) != 0 {
			t.Errorf("%s: %d entries", name, len(c.entries))
		}
	}
	if c := LoadCache(filepath.Join(dir, "missing")); len(c.entries) != 0 {
		t.Error("a missing file gave entries")
	}
}

func TestSavingTheCacheFailsInAFolderThatIsNotThere(t *testing.T) {
	c := NewCache()
	c.store("a", Stamp{}, nil)
	err := c.Save(filepath.Join(t.TempDir(), "no", "such", "dir", "faces.json"), func(string) bool { return true })
	assertRoute(t, err, "WindowsFontCacheWriteFailed", "Cache.Save.WindowsFontCacheWriteFailed")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the error does not keep its cause: %v", err)
	}
}

func TestSupportedFile(t *testing.T) {
	for p, want := range map[string]bool{
		`C:\Windows\Fonts\segoeui.ttf`: true, "A.TTF": true, "a.otf": true, "a.ttc": true, "a.OTC": true,
		"vgasys.fon": false, "cour.pfb": false, "desktop.ini": false, "noext": false, "": false,
	} {
		if got := SupportedFile(p); got != want {
			t.Errorf("SupportedFile(%q) = %v", p, got)
		}
	}
}
