//go:build windows

package windows

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/text/winfonts"
)

// fontsKey is the registry key (under HKEY_LOCAL_MACHINE and HKEY_CURRENT_USER) whose values are the
// installed fonts.
const fontsKey = `Software\Microsoft\Windows NT\CurrentVersion\Fonts`

// SystemFonts lists the installed faces of the machine (SPEC §25.8): the files named by the font
// registry keys of the machine and of the user and the files in %WINDIR%\Fonts and
// %LOCALAPPDATA%\Microsoft\Windows\Fonts, read with winfonts.Scan. What Scan learns is kept in a
// file in the user's cache directory (os.UserCacheDir()\glutter\fonts), so that the next start does
// not open every font file. It returns no error: a registry key or a directory that cannot be read
// is logged and left out, and the result may be empty.
//
// Flutter: no counterpart
func SystemFonts(ctx context.Context, log *slog.Logger) []winfonts.Face {
	if log == nil {
		log = slog.Default()
	}
	var entries []fontEntry
	for _, root := range []uintptr{win32.HKeyLocalMachine, win32.HKeyCurrentUser} {
		values, err := win32.RegEnumStrings(root, fontsKey)
		if err != nil {
			log.Warn("the font list of the registry could not be read", "err", err)
		}
		for _, v := range values {
			entries = append(entries, fontEntry{Name: v.Name, File: v.Data})
		}
	}
	windir := os.Getenv("SYSTEMROOT")
	if windir == "" {
		windir = os.Getenv("WINDIR")
	}
	systemFonts := windir + `\Fonts`
	dirs := []string{systemFonts}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, local+`\Microsoft\Windows\Fonts`)
	}
	files := fontFiles(entries, systemFonts, dirs, func(dir string) []string {
		items, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn("a font directory could not be read", "dir", dir, "err", err)
			}
			return nil
		}
		var names []string
		for _, it := range items {
			if !it.IsDir() {
				names = append(names, it.Name())
			}
		}
		return names
	})
	cacheFile := ""
	if base, err := os.UserCacheDir(); err == nil {
		dir := filepath.Join(base, "glutter", "fonts")
		if err := os.MkdirAll(dir, 0o700); err == nil {
			cacheFile = filepath.Join(dir, "windows-faces.json")
		}
	}
	cache := winfonts.NewCache()
	if cacheFile != "" {
		cache = winfonts.LoadCache(cacheFile)
	}
	faces := winfonts.Scan(ctx, files, cache, func(path string, err error) {
		log.Debug("a font file could not be read", "path", path, "err", err)
	})
	if cacheFile != "" {
		keep := map[string]bool{}
		for _, f := range files {
			keep[f] = true
		}
		if err := cache.Save(cacheFile, func(p string) bool { return keep[p] }); err != nil {
			log.Warn("the font cache could not be written", "err", err)
		}
	}
	return faces
}

// UserLocale is the first language of the user as a BCP 47 tag ("ja-JP"), "" when it is not known.
// The font matcher puts the fonts written for it first when a request names no language.
//
// Flutter: no counterpart
func UserLocale() string {
	langs, err := win32.GetUserPreferredUILanguages(win32.MUILanguageName)
	if err != nil || len(langs) == 0 {
		return ""
	}
	return strings.TrimSpace(langs[0])
}
