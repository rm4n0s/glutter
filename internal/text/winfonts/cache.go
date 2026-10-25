package winfonts

import (
	"encoding/json"
	"os"
	"time"

	"github.com/rm4n0s/errors"
)

// Stamp identifies the state of a font file: the cache trusts the faces it holds for a file while
// the size and the modification time are the same.
//
// Flutter: no counterpart
type Stamp struct {
	Size    int64
	ModTime int64 // Unix nanoseconds
}

// stampOf is the stamp of a file, false when it cannot be statted.
func stampOf(path string) (Stamp, bool) {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return Stamp{}, false
	}
	return Stamp{Size: st.Size(), ModTime: st.ModTime().UnixNano()}, true
}

// Cache keeps the faces that Scan read, so that the next start of the app does not open every font
// file of the machine (hundreds on Windows). It is kept in a file in the cache directory of the user
// (Load and Save); a damaged file is an empty cache. It is not safe for concurrent use; Scan locks it.
//
// Flutter: no counterpart
type Cache struct {
	entries map[string]cacheEntry
	dirty   bool
}

type cacheEntry struct {
	Stamp Stamp
	Faces []Face
}

// cacheVersion changes when the format of the file does.
const cacheVersion = 1

type cacheFile struct {
	Version int
	Entries map[string]cacheEntry
}

// NewCache makes an empty cache.
//
// Flutter: no counterpart
func NewCache() *Cache { return &Cache{entries: map[string]cacheEntry{}} }

func (c *Cache) lookup(path string, stamp Stamp) ([]Face, bool) {
	e, ok := c.entries[path]
	if !ok || e.Stamp != stamp {
		return nil, false
	}
	return e.Faces, true
}

func (c *Cache) store(path string, stamp Stamp, faces []Face) {
	c.entries[path] = cacheEntry{Stamp: stamp, Faces: faces}
	c.dirty = true
}

// Dirty says whether Scan learned a file since the cache was loaded or saved.
//
// Flutter: no counterpart
func (c *Cache) Dirty() bool { return c.dirty }

// LoadCache reads the cache from a file. A file that is missing, damaged or of another version gives
// an empty cache, with no error: the next Scan reads the fonts and Save writes the file again.
//
// Flutter: no counterpart
func LoadCache(path string) *Cache {
	c := NewCache()
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	var f cacheFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != cacheVersion || f.Entries == nil {
		return c
	}
	c.entries = f.Entries
	return c
}

// Save writes the cache to the file, when it learned something. The entries of files that are no
// longer in the list are dropped first (keep says which files are). It fails with
// WindowsFontCacheWriteFailed.
//
// Flutter: no counterpart
func (c *Cache) Save(path string, keep func(path string) bool) error {
	for p := range c.entries {
		if !keep(p) {
			delete(c.entries, p)
			c.dirty = true
		}
	}
	if !c.dirty {
		return nil
	}
	data, err := json.Marshal(cacheFile{Version: cacheVersion, Entries: c.entries})
	if err != nil {
		return errors.NewErr(tagCacheWriteFailed, err).SetMetadata("path", path)
	}
	tmp := path + ".tmp." + time.Now().Format("150405.000000")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return errors.NewErr(tagCacheWriteFailed, err).SetMetadata("path", path)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // the rename error is the one to report
		return errors.NewErr(tagCacheWriteFailed, err).SetMetadata("path", path)
	}
	c.dirty = false
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCacheWriteFailed = "WindowsFontCacheWriteFailed"
)
