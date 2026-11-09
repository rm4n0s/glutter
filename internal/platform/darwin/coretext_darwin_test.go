//go:build darwin

package darwin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreTextFindsTheSystemFont(t *testing.T) {
	ct, err := OpenCoreText()
	if err != nil {
		t.Fatal(err)
	}
	f, ok := ct.SystemUIFont("")
	if !ok {
		t.Fatal("Core Text has no system UI font")
	}
	if !filepath.IsAbs(f.Path) {
		t.Errorf("the path of the system font is %q", f.Path)
	}
	if _, err := os.Stat(f.Path); err != nil {
		t.Errorf("the file of the system font: %v", err)
	}
	t.Logf("system font: %+v", f)
	if f.Family == "" || f.Index < 0 {
		t.Errorf("system font = %+v", f)
	}
}

func TestCoreTextListsFontFiles(t *testing.T) {
	ct, err := OpenCoreText()
	if err != nil {
		t.Fatal(err)
	}
	files := ct.FontFiles()
	if len(files) < 20 {
		t.Fatalf("%d font files", len(files))
	}
	seen := map[string]bool{}
	for _, p := range files {
		if !filepath.IsAbs(p) || seen[p] {
			t.Fatalf("the list has %q (absolute: %v, twice: %v)", p, filepath.IsAbs(p), seen[p])
		}
		seen[p] = true
	}
}

func TestCoreTextFallbackFindsAFontForAHanCharacterAndAnEmoji(t *testing.T) {
	ct, err := OpenCoreText()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		r    rune
		lang string
		want string // part of the file name
	}{
		{"han", '汉', "zh-Hans", ""},
		{"emoji", 0x1F600, "", "Emoji"},
	} {
		f, ok := ct.Fallback("", tt.r, tt.lang)
		if !ok {
			t.Errorf("%s: no fallback", tt.name)
			continue
		}
		t.Logf("%s: %+v", tt.name, f)
		if _, err := os.Stat(f.Path); err != nil {
			t.Errorf("%s: %v", tt.name, err)
		}
		if strings.Contains(f.Path, "LastResort") {
			t.Errorf("%s: the fallback is the LastResort font %+v", tt.name, f)
		}
		if tt.want != "" && !strings.Contains(f.Path, tt.want) {
			t.Errorf("%s: fallback %+v, want a file with %q", tt.name, f, tt.want)
		}
	}
	// A family that Core Text does not know still gets a font (its default one).
	if _, ok := ct.Fallback("No Such Family 12345", 'a', ""); !ok {
		t.Error("no fallback for the unknown family")
	}
}
