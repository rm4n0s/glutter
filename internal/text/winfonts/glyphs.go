package winfonts

import (
	"os"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

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
		return false
	}
	gid, found := cmap.Lookup(r)
	return found && gid != 0
}

// readCmap loads the cmap of the face, nil when the file or the face cannot be read.
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
	ft, err := font.NewFont(loaders[index])
	if err != nil {
		return nil
	}
	return ft.Cmap
}
