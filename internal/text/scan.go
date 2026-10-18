package text

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/rm4n0s/errors"
)

// discardLogger drops the messages of fontscan.
type discardLogger struct{}

func (discardLogger) Printf(string, ...any) {}

// scanMatcher is the Matcher for systems without fontconfig: it finds fonts with
// go-text's fontscan, which scans the font directories and keeps an index of them
// in a cache directory.
type scanMatcher struct {
	footprints []fontscan.Footprint
	fm         *fontscan.FontMap
}

// defaultCacheDir is $XDG_CACHE_HOME/glutter/fonts, with ~/.cache for the
// XDG_CACHE_HOME that is not set.
func defaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "glutter", "fonts"), nil
}

// newScanMatcher scans the system fonts, or reads their index from cacheDir. It
// fails with FontScanFailed.
func newScanMatcher(cacheDir string, logger fontscan.Logger) (*scanMatcher, error) {
	if cacheDir == "" {
		dir, err := defaultCacheDir()
		if err != nil {
			return nil, errors.NewErr("FontScanFailed", err)
		}
		cacheDir = dir
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, errors.NewErr("FontScanFailed", err).SetMetadata("cacheDir", cacheDir)
	}
	if logger == nil {
		logger = discardLogger{}
	}
	footprints, err := fontscan.SystemFonts(logger, cacheDir)
	if err != nil {
		return nil, errors.NewErr("FontScanFailed", err).SetMetadata("cacheDir", cacheDir)
	}
	fm := fontscan.NewFontMap(logger)
	if err := fm.UseSystemFonts(cacheDir); err != nil {
		return nil, errors.NewErr("FontScanFailed", err).SetMetadata("cacheDir", cacheDir)
	}
	return &scanMatcher{footprints: footprints, fm: fm}, nil
}

// Match puts first the fonts of the family, the closest to the weight and slant
// first; then the font that fontscan substitutes for it (what the family stands for,
// such as DejaVu Sans for sans-serif); then the font that covers the language.
func (s *scanMatcher) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]FontFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontMatchCanceled", err).SetMetadata("family", family)
	}
	aspect := font.Aspect{Style: font.StyleNormal, Weight: font.Weight(weight), Stretch: font.StretchNormal}
	if italic {
		aspect.Style = font.StyleItalic
	}
	var out []FontFile
	seen := map[fontscan.Location]bool{}
	add := func(loc fontscan.Location, fam string) {
		if seen[loc] {
			return
		}
		seen[loc] = true
		out = append(out, FontFile{Path: loc.File, Index: int(loc.Index), Family: fam})
	}

	want := font.NormalizeFamily(family)
	var exact []fontscan.Footprint
	for _, fp := range s.footprints {
		if fp.Family == want {
			exact = append(exact, fp)
		}
	}
	sort.SliceStable(exact, func(i, j int) bool {
		return aspectDistance(exact[i].Aspect, aspect) < aspectDistance(exact[j].Aspect, aspect)
	})
	for _, fp := range exact {
		add(fp.Location, family)
	}

	s.fm.SetQuery(fontscan.Query{Families: []string{family}, Aspect: aspect})
	if face := s.fm.ResolveFace('a'); face != nil {
		fam, _ := s.fm.FontMetadata(face.Font)
		add(s.fm.FontLocation(face.Font), fam)
	}
	if lang != "" {
		if id, ok := language.NewLangID(language.NewLanguage(lang)); ok {
			if face := s.fm.ResolveFaceForLang(id); face != nil {
				fam, _ := s.fm.FontMetadata(face.Font)
				add(s.fm.FontLocation(face.Font), fam)
			}
		}
	}
	return out, nil
}

// aspectDistance says how far a font is from the wanted style: a different slant
// counts more than any difference in weight.
func aspectDistance(have, want font.Aspect) float32 {
	d := have.Weight - want.Weight
	if d < 0 {
		d = -d
	}
	if have.Style != want.Style {
		d += 1000
	}
	return float32(d)
}
