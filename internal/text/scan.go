package text

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/rm4n0s/errors"
)

// discardLogger drops the messages of fontscan.
type discardLogger struct{}

func (discardLogger) Printf(string, ...any) {}

var _ Coverer = (*scanMatcher)(nil)

// scanMatcher is the Matcher for systems without fontconfig: it finds fonts with
// go-text's fontscan, which scans the font directories and keeps an index of them
// in a cache directory.
type scanMatcher struct {
	footprints []fontscan.Footprint

	mu sync.Mutex // fontscan's FontMap is not safe for concurrent use
	fm *fontscan.FontMap
	// families is the answer of Families, made on the first call.
	families []string
	// byFace finds a footprint by its file and face, made on the first call of Covers.
	byFace map[faceID]int
}

// faceID is a file and a face in it.
type faceID struct {
	path  string
	index uint16
}

// defaultCacheDir is $XDG_CACHE_HOME/glutter/fonts, with ~/.cache for the
// XDG_CACHE_HOME that is not set. It fails with FontScanFailed.
func defaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", errors.NewErr(tagFontScanFailed, err)
	}
	return filepath.Join(base, "glutter", "fonts"), nil
}

// newScanMatcher scans the system fonts, or reads their index from cacheDir. It
// fails with FontScanFailed.
func newScanMatcher(cacheDir string, logger fontscan.Logger) (*scanMatcher, error) {
	if cacheDir == "" {
		dir, err := defaultCacheDir()
		if err != nil {
			return nil, err
		}
		cacheDir = dir
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, errors.NewErr(tagFontScanFailed, err).SetMetadata("cacheDir", cacheDir)
	}
	if logger == nil {
		logger = discardLogger{}
	}
	footprints, err := fontscan.SystemFonts(logger, cacheDir)
	if err != nil {
		return nil, errors.NewErr(tagFontScanFailed, err).SetMetadata("cacheDir", cacheDir)
	}
	fm := fontscan.NewFontMap(logger)
	if err := fm.UseSystemFonts(cacheDir); err != nil {
		return nil, errors.NewErr(tagFontScanFailed, err).SetMetadata("cacheDir", cacheDir)
	}
	return &scanMatcher{footprints: footprints, fm: fm}, nil
}

// Match puts first the fonts of the family, the closest to the weight and slant
// first; then the font that fontscan substitutes for it (what the family stands for,
// such as DejaVu Sans for sans-serif); then the font that covers the language.
func (s *scanMatcher) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]FontFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontMatchCanceled, err).SetMetadata("family", family)
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
		return FontFile{}, false, errors.NewErr(tagFontMatchCanceled, err).SetMetadata("family", family)
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

// Covers implements Coverer from the footprints of the scan, which keep the runes of every font (the
// set that the scan read from its cmap): no font is opened. A font that the scan does not list is unknown.
func (s *scanMatcher) Covers(_ context.Context, f FontFile, r rune) (has, known bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byFace == nil {
		s.byFace = make(map[faceID]int, len(s.footprints))
		for i, fp := range s.footprints {
			id := faceID{fp.Location.File, fp.Location.Index}
			if _, dup := s.byFace[id]; !dup {
				s.byFace[id] = i
			}
		}
	}
	i, ok := s.byFace[faceID{f.Path, uint16(f.Index & 0xffff)}]
	if !ok {
		return false, false
	}
	return s.footprints[i].Runes.Contains(r), true
}

// Families implements Lister: the family names of the fonts that the scan found. fontscan
// keeps the family only in a normalized form, so the name each family has in its font is read
// from the first file of the family (only the name table is read).
func (s *scanMatcher) Families(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontMatchCanceled, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.families != nil {
		return s.families, nil
	}
	first := map[string]fontscan.Location{}
	for _, fp := range s.footprints {
		if _, ok := first[fp.Family]; !ok {
			first[fp.Family] = fp.Location
		}
	}
	out := make([]string, 0, len(first))
	var buf []byte
	for norm, loc := range first {
		name := displayName(loc, &buf)
		if name == "" {
			name = norm
		}
		out = append(out, name)
	}
	s.families = out
	return out, nil
}

// displayName reads the family name of the font at loc, or "" when the file cannot be read.
func displayName(loc fontscan.Location, buf *[]byte) string {
	file, err := os.Open(loc.File)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }() // read only
	loaders, err := ot.NewLoaders(file)
	if err != nil || int(loc.Index&0xffff) >= len(loaders) {
		return ""
	}
	var desc font.Description
	desc, *buf = font.Describe(loaders[loc.Index&0xffff], *buf)
	return desc.Family
}

// probeFamily is a family name that no installed font has.
const probeFamily = "glutter-probe-family-with-no-font"

// Aliases implements Aliaser: the family of the font that fontscan substitutes for the
// family, when it is not the font that it gives for a name nothing mentions (its default).
func (s *scanMatcher) Aliases(ctx context.Context, family string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontMatchCanceled, err).SetMetadata("family", family)
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontScanFailed = "FontScanFailed"
)
