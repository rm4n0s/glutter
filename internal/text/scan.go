package text

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"

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

	mu sync.Mutex // fontscan's FontMap is not safe for concurrent use
	fm *fontscan.FontMap
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

	s.mu.Lock()
	defer s.mu.Unlock()
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

// Fallback finds the font that has a glyph for r, as fontscan picks it (by the script of the
// rune and the families it substitutes for the requested one).
func (s *scanMatcher) Fallback(ctx context.Context, family string, weight int, italic bool, lang string, r rune) (FontFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return FontFile{}, false, errors.NewErr("FontMatchCanceled", err).SetMetadata("family", family)
	}
	aspect := font.Aspect{Style: font.StyleNormal, Weight: font.Weight(weight), Stretch: font.StretchNormal}
	if italic {
		aspect.Style = font.StyleItalic
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if IsEmoji(r) {
		// The color emoji font, when there is one, rather than a black and white symbol font.
		for _, fp := range s.footprints {
			if fp.Family == "notocoloremoji" && fp.Runes.Contains(r) {
				return FontFile{Path: fp.Location.File, Index: int(fp.Location.Index), Family: "Noto Color Emoji"}, true, nil
			}
		}
	}
	s.fm.SetQuery(fontscan.Query{Families: []string{family}, Aspect: aspect})
	s.fm.SetScript(language.LookupScript(r))
	face := s.fm.ResolveFace(r)
	if face == nil {
		return FontFile{}, false, nil
	}
	if _, ok := face.NominalGlyph(r); !ok {
		return FontFile{}, false, nil
	}
	fam, _ := s.fm.FontMetadata(face.Font)
	loc := s.fm.FontLocation(face.Font)
	return FontFile{Path: loc.File, Index: int(loc.Index), Family: fam}, true, nil
}

// probeFamily is a family name that no installed font has.
const probeFamily = "glutter-probe-family-with-no-font"

// Aliases implements Aliaser: the family of the font that fontscan substitutes for the
// family, when it is not the font that it gives for a name nothing mentions (its default).
func (s *scanMatcher) Aliases(ctx context.Context, family string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontMatchCanceled", err).SetMetadata("family", family)
	}
	regular := font.Aspect{Style: font.StyleNormal, Weight: 400, Stretch: font.StretchNormal}
	s.mu.Lock()
	defer s.mu.Unlock()
	resolve := func(name string) string {
		s.fm.SetQuery(fontscan.Query{Families: []string{name}, Aspect: regular})
		face := s.fm.ResolveFace('a')
		if face == nil {
			return ""
		}
		fam, _ := s.fm.FontMetadata(face.Font)
		return fam
	}
	got := resolve(family)
	if got == "" || normalizeFamily(got) == normalizeFamily(family) || normalizeFamily(got) == normalizeFamily(resolve(probeFamily)) {
		return nil, nil
	}
	return []string{got}, nil
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
