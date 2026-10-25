package windows

import (
	"strings"

	"github.com/rm4n0s/glutter/internal/text/winfonts"
)

// This file is the plain-Go part of the list of the system's fonts (SPEC §25.8): which files are
// installed, from the registry keys that list them and from the font directories. Reading them
// is fonts_windows.go; reading the faces in the files is winfonts.Scan.

// The values of the registry key that lists the installed fonts (fontsKey, in fonts_windows.go) are the
// name of the face ("Segoe UI Bold (TrueType)") and the file, which is a path relative to
// %WINDIR%\Fonts for a font of the system and an absolute path for one that a user installed.

// fontEntry is one value of the fonts key.
type fontEntry struct {
	// Name is the name of the value and File its data.
	Name, File string
}

// isAbsoluteWindowsPath says whether the path starts with a drive or is a network path.
func isAbsoluteWindowsPath(p string) bool {
	if strings.HasPrefix(p, `\\`) {
		return true
	}
	return len(p) >= 3 && isDriveLetter(p[0]) && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

// fontFiles lists the font files to read: the files that the registry values name (a relative path is
// in systemFonts), then the files of the font directories (the system's and the user's), each file
// once, comparing paths without regard to case. Only the files go-text reads are kept: TrueType and
// OpenType fonts and collections, not the bitmap (.fon) and Type 1 fonts that Windows lists too. The
// values whose name starts with "@" are the vertical-writing names of fonts that the horizontal
// names already list.
func fontFiles(entries []fontEntry, systemFonts string, dirs []string, list func(dir string) []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		key := strings.ToLower(strings.ReplaceAll(p, "/", `\`))
		if p == "" || seen[key] || !winfonts.SupportedFile(p) {
			return
		}
		seen[key] = true
		out = append(out, p)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name, "@") {
			continue
		}
		file := e.File
		if !isAbsoluteWindowsPath(file) {
			file = strings.TrimRight(systemFonts, `\`) + `\` + file
		}
		add(file)
	}
	for _, dir := range dirs {
		for _, f := range list(dir) {
			add(strings.TrimRight(dir, `\`) + `\` + f)
		}
	}
	return out
}
