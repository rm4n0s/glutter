//go:build linux && !android

package fontconfig

import (
	"context"
	"os"
	"testing"
	"unsafe"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// TestCoversWithTheRealLibrary compares the character sets of fontconfig with the cmap tables of the
// fonts it listed.
func TestCoversWithTheRealLibrary(t *testing.T) {
	c := openOrSkip(t)
	files, err := c.Match(context.Background(), "monospace", 400, false, "")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, f := range files {
		if checked == 20 {
			break
		}
		cmap := cmapOf(f.Path, f.Index&0xffff)
		if cmap == nil {
			continue
		}
		checked++
		for _, r := range []rune{'a', '\t', 'α', '日', 0x1F600} {
			has, known := c.Covers(f.Path, f.Index, r)
			if !known {
				t.Fatalf("Covers(%s, %q) is unknown for a font that Match listed", f.Path, r)
			}
			_, want := cmap.Lookup(r)
			if has != want {
				t.Errorf("Covers(%s#%d, %q) = %t, the cmap says %t", f.Path, f.Index, r, has, want)
			}
		}
	}
	if checked == 0 {
		t.Skip("no font file to compare with")
	}
	if _, known := c.Covers("/nowhere/not-listed.ttf", 0, 'a'); known {
		t.Error("Covers knows a font that no Match listed")
	}
}

// cmapOf reads the cmap of a face, nil when it cannot be read.
func cmapOf(path string, index int) font.Cmap {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }() // read only
	loaders, err := ot.NewLoaders(file)
	if err != nil || index >= len(loaders) {
		return nil
	}
	ft, err := font.NewFont(loaders[index])
	if err != nil {
		return nil
	}
	return ft.Cmap
}

// TestCoversKeepsAReferenceOfTheSet checks the life of the character sets with a fake library: a set is
// copied once for the font it belongs to, asked after the font set of the Match is gone, and destroyed by Close.
func TestCoversKeepsAReferenceOfTheSet(t *testing.T) {
	lib := fakeLibrary()
	copies, destroyed := 0, 0
	// A font set in C memory with two patterns, 11 and 12, as fontconfig would return it.
	patterns := dl.Calloc(2, 8)
	set := dl.Calloc(1, 16)
	defer dl.Free(patterns)
	defer dl.Free(set)
	pats := unsafe.Slice((*uintptr)(dl.Pointer(patterns)), 2)
	pats[0], pats[1] = 11, 12
	fs := (*fcFontSet)(dl.Pointer(set))
	fs.nfont, fs.fonts = 2, patterns
	lib.fontSort = func(_, _ uintptr, _ int32, _, _ uintptr) uintptr { return set }
	paths := map[uintptr]string{11: "/f/a.ttf", 12: "/f/b.ttf"}
	lib.patternGetString = func(pat, object uintptr, _ int32, out uintptr) int32 {
		if dl.GoString(object) != objFile {
			return 1
		}
		*(*uintptr)(dl.Pointer(out)) = dl.CString(paths[pat]) // freed with the process: it is a test
		return fcResultMatch
	}
	lib.patternGetInteger = func(_, _ uintptr, _ int32, _ uintptr) int32 { return 1 }
	lib.patternGetCharSet = func(pat, _ uintptr, _ int32, out uintptr) int32 {
		*(*uintptr)(dl.Pointer(out)) = pat * 1000 // the set of pattern 11 is 11000
		return fcResultMatch
	}
	lib.charSetCopy = func(cs uintptr) uintptr { copies++; return cs }
	lib.charSetDestroy = func(uintptr) { destroyed++ }
	lib.charSetHasChar = func(cs uintptr, ucs4 uint32) int32 {
		if cs == 11000 && ucs4 == 'a' || cs == 12000 && ucs4 == 'b' {
			return 1
		}
		return 0
	}
	c, err := newConfig(lib)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // the second Match finds the sets known and copies nothing
		if _, err := c.Match(context.Background(), "monospace", 400, false, ""); err != nil {
			t.Fatal(err)
		}
	}
	if copies != 2 {
		t.Errorf("%d references of sets were taken for 2 fonts matched twice, want 2", copies)
	}
	tests := []struct {
		name      string
		path      string
		index     int
		r         rune
		has, know bool
	}{
		{"a letter that the first font has", "/f/a.ttf", 0, 'a', true, true},
		{"a letter that it lacks", "/f/a.ttf", 0, 'b', false, true},
		{"the second font", "/f/b.ttf", 0, 'b', true, true},
		{"the instance in the high bits of the index is ignored", "/f/b.ttf", 3 << 16, 'b', true, true},
		{"a font that no Match listed", "/f/c.ttf", 0, 'a', false, false},
		{"a negative rune", "/f/a.ttf", 0, -1, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, known := c.Covers(tt.path, tt.index, tt.r)
			if has != tt.has || known != tt.know {
				t.Errorf("Covers = %t, %t; want %t, %t", has, known, tt.has, tt.know)
			}
		})
	}
	c.Close()
	if destroyed != 2 {
		t.Errorf("Close destroyed %d references, want 2", destroyed)
	}
	if _, known := c.Covers("/f/a.ttf", 0, 'a'); known {
		t.Error("Covers answers after Close")
	}
}
