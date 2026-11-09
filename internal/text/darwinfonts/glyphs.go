package darwinfonts

import (
	"os"
	"slices"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// hasDrawableTable says that the face has a table that holds glyphs the layout can draw.
func hasDrawableTable(ld *ot.Loader) bool {
	for _, have := range ld.Tables() {
		if slices.Contains(drawableTables, have.String()) {
			return true
		}
	}
	return false
}

// glyphChecker is the default Options.HasGlyph: it reads the cmap of a font once, with go-text, and
// keeps it per file and index. A file that cannot be read or has no face at the index has no glyph.
type glyphChecker struct {
	mu    sync.Mutex
	cmaps map[glyphKey]font.Cmap
}

type glyphKey struct {
	path  string
	index int
}

func newGlyphChecker() *glyphChecker {
	return &glyphChecker{cmaps: map[glyphKey]font.Cmap{}}
}

// hasGlyph says whether the face at index of the file has a glyph for r.
func (c *glyphChecker) hasGlyph(path string, index int, r rune) bool {
	has, _ := c.covers(path, index, r)
	return has
}

// covers is hasGlyph that also says whether it could tell: known is false for a file that cannot be read
// or has no face at the index, which text.Collection.Covers then reads itself and reports.
func (c *glyphChecker) covers(path string, index int, r rune) (has, known bool) {
	key := glyphKey{path, index}
	c.mu.Lock()
	cmap, ok := c.cmaps[key]
	c.mu.Unlock()
	if !ok {
		cmap = readCmap(path, index)
		c.mu.Lock()
		c.cmaps[key] = cmap
		c.mu.Unlock()
	}
	if cmap == nil {
		return false, false
	}
	gid, found := cmap.Lookup(r)
	return found && gid != 0, true
}

// noGlyphs is the cmap of a face that cannot be drawn: it has no glyph for any character.
type noGlyphs struct{}

func (noGlyphs) Lookup(rune) (font.GID, bool) { return 0, false }

func (noGlyphs) Iter() font.CmapIter { return nil }

// drawableTables are the tables that hold glyphs which the text layout can draw: TrueType and CFF
// outlines, COLR color glyphs and the sbix and CBDT bitmaps. The private fonts of macOS that keep
// their glyphs in Apple's hvgl table (the interface fonts of PingFang) have none of them: go-text
// reads them and finds every glyph empty.
var drawableTables = []string{"glyf", "CFF ", "CFF2", "sbix", "CBDT", "COLR"}

// readCmap loads the cmap of the face, nil when the file or the face cannot be read. A face that has
// no table that holds drawable glyphs has a cmap with no characters.
func readCmap(path string, index int) font.Cmap {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }() // the cmap is in memory; a failed close of a read-only file loses nothing
	loaders, err := ot.NewLoaders(file)
	if err != nil || index < 0 || index >= len(loaders) {
		return nil
	}
	if !hasDrawableTable(loaders[index]) {
		return noGlyphs{}
	}
	ft, err := font.NewFont(loaders[index])
	if err != nil {
		return nil
	}
	return ft.Cmap
}
