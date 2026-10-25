package winfonts

import (
	"context"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/rm4n0s/errors"
)

// Face is one face of a font file that is installed: the family it belongs to, as its name table
// says (the preferred family, so that "Segoe UI Semibold" is a weight of "Segoe UI"), its weight and
// slant, and where it is. A font collection (.ttc) has a face for each font in it.
//
// Flutter: no counterpart
type Face struct {
	// Family is the name of the family.
	Family string
	// Path is the font file; Index the face in it (0 for a file with one font).
	Path  string
	Index int
	// Weight is the OpenType weight (400 regular, 700 bold) and Italic whether the face is slanted
	// (italic or oblique).
	Weight int
	Italic bool
}

// Describe reads the faces of a font file: the family, the weight and the slant of each font in
// it. The file is not kept open. It fails with WindowsFontUnreadable when the file cannot be opened
// or is not a font.
//
// Flutter: no counterpart
func Describe(path string) ([]Face, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.NewErr(tagUnreadable, err).SetMetadata("path", path)
	}
	defer func() { _ = file.Close() }() // only the name tables were read; a failed close of a read-only file loses nothing
	loaders, err := ot.NewLoaders(file)
	if err != nil {
		return nil, errors.NewErr(tagUnreadable, err).SetMetadata("path", path)
	}
	var buf []byte
	faces := make([]Face, 0, len(loaders))
	for i, ld := range loaders {
		var desc font.Description
		desc, buf = font.Describe(ld, buf)
		if desc.Family == "" {
			continue
		}
		faces = append(faces, Face{
			Family: desc.Family, Path: path, Index: i,
			Weight: int(desc.Aspect.Weight), Italic: desc.Aspect.Style == font.StyleItalic,
		})
	}
	return faces, nil
}

// Scan reads the faces of the files, several at a time. A file that cannot be read is left out and
// reported to onError (when it is not nil); the faces keep the order of the files. A cache that has
// the file with the same stamp (see Stamp) saves reading it, and learns the files it did not have.
//
// Flutter: no counterpart
func Scan(ctx context.Context, files []string, cache *Cache, onError func(path string, err error)) []Face {
	type result struct {
		faces []Face
		err   error
	}
	results := make([]result, len(files))
	var wg sync.WaitGroup
	next := make(chan int)
	workers := min(max(runtime.NumCPU(), 2), 8)
	var mu sync.Mutex // the cache
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				path := files[i]
				stamp, hasStamp := stampOf(path)
				if cache != nil && hasStamp {
					mu.Lock()
					faces, ok := cache.lookup(path, stamp)
					mu.Unlock()
					if ok {
						results[i].faces = faces
						continue
					}
				}
				faces, err := Describe(path)
				results[i] = result{faces, err}
				if err == nil && cache != nil && hasStamp {
					mu.Lock()
					cache.store(path, stamp, faces)
					mu.Unlock()
				}
			}
		}()
	}
loop:
	for i := range files {
		select {
		case next <- i:
		case <-ctx.Done():
			break loop
		}
	}
	close(next)
	wg.Wait()
	var out []Face
	for i, r := range results {
		if r.err != nil {
			if onError != nil {
				onError(files[i], r.err)
			}
			continue
		}
		out = append(out, r.faces...)
	}
	return out
}

// SupportedFile says whether go-text can read the file, judging by its extension: TrueType and
// OpenType fonts and their collections. Windows also lists bitmap fonts (.fon) and Type 1 fonts.
//
// Flutter: no counterpart
func SupportedFile(path string) bool {
	i := strings.LastIndexByte(path, '.')
	if i < 0 {
		return false
	}
	switch strings.ToLower(path[i:]) {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	}
	return false
}

// dedupe drops the faces that repeat a file and index, keeping the first.
func dedupe(faces []Face) []Face {
	type key struct {
		path  string
		index int
	}
	seen := map[key]bool{}
	return slices.DeleteFunc(slices.Clone(faces), func(f Face) bool {
		k := key{strings.ToLower(f.Path), f.Index}
		if seen[k] {
			return true
		}
		seen[k] = true
		return false
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagUnreadable = "WindowsFontUnreadable"
)
