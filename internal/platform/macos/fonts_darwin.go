//go:build darwin && !ios

package macos

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/text/darwinfonts"
)

// FontOptions reads the fonts of the Mac for darwinfonts.New (SPEC §29.8): the faces of the files that Core
// Text lists (read with go-text, and kept in a file of the user's cache directory between runs), the system
// UI font, Core Text's choice of a font for a character, and the user's first language. A file that cannot
// be read is logged and left out. It fails with the loader's errors when Core Text is missing.
//
// Flutter: no counterpart
func FontOptions(ctx context.Context, log *slog.Logger) (darwinfonts.Options, error) {
	if log == nil {
		log = slog.Default()
	}
	ct, err := darwin.OpenCoreText()
	if err != nil {
		return darwinfonts.Options{}, err
	}
	var files []string
	for _, f := range ct.FontFiles() {
		if darwinfonts.SupportedFile(f) {
			files = append(files, f)
		}
	}
	cacheFile := ""
	if base, err := os.UserCacheDir(); err == nil {
		dir := filepath.Join(base, "glutter", "fonts")
		if err := os.MkdirAll(dir, 0o700); err == nil {
			cacheFile = filepath.Join(dir, "darwin-faces.json")
		}
	}
	cache := darwinfonts.NewCache()
	if cacheFile != "" {
		cache = darwinfonts.LoadCache(cacheFile)
	}
	faces := darwinfonts.Scan(ctx, files, cache, func(path string, err error) {
		log.Debug("a font file could not be read", "path", path, "err", err)
	})
	if cacheFile != "" {
		keep := make(map[string]bool, len(files))
		for _, f := range files {
			keep[f] = true
		}
		if err := cache.Save(cacheFile, func(p string) bool { return keep[p] }); err != nil {
			log.Warn("the font cache could not be written", "err", err)
		}
	}
	return darwinfonts.Options{
		Faces:  faces,
		Locale: userLocale(),
		SystemUI: func(lang string) (darwinfonts.FontFile, bool) {
			f, ok := ct.SystemUIFont(lang)
			return darwinfonts.FontFile(f), ok
		},
		SystemFallback: func(family string, r rune, lang string) (darwinfonts.FontFile, bool) {
			f, ok := ct.Fallback(family, r, lang)
			return darwinfonts.FontFile(f), ok
		},
	}, nil
}

// userLocale is the first of the user's preferred languages (a BCP 47 tag), "" when there is none.
func userLocale() (tag string) {
	objcx.WithPool(func() {
		langs := class("NSLocale").Send(sel("preferredLanguages"))
		if objcx.ArrayCount(langs) > 0 {
			tag = objcx.String(objcx.ArrayAt(langs, 0))
		}
	})
	return tag
}
