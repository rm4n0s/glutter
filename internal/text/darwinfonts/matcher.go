package darwinfonts

import (
	"context"
	"slices"
	"strings"

	"github.com/go-text/typesetting/font"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

var (
	_ text.Matcher    = (*Matcher)(nil)
	_ text.Aliaser    = (*Matcher)(nil)
	_ text.Lister     = (*Matcher)(nil)
	_ text.Fallbacker = (*Matcher)(nil)
	_ text.Coverer    = (*Matcher)(nil)
)

// Options are the parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Faces are the installed faces (Scan reads them from the files). New fails when there are none and no
	// SystemUI font.
	Faces []Face
	// Locale is the BCP 47 tag of the language used when a request names none (the user's first
	// language); "" keeps the order of the chain.
	Locale string
	// HasGlyph says whether the face at index of a font file has a glyph for r. It is how Fallback checks a
	// font. It defaults to the cmap of the font, read with go-text and kept per file and index.
	HasGlyph func(path string, index int, r rune) bool
	// SystemUI gives the file and the face of the system UI font (San Francisco), made for the language
	// ("" for the default); it is false when there is none. The font becomes the family SystemUIFamily, with
	// the weight and slant that its file says. It is called once, by New, with Locale.
	SystemUI func(lang string) (FontFile, bool)
	// SystemFallback asks the system which font it would use for r where the family lacks it (Core Text's
	// CTFontCreateForString): family is "" for the system UI font, lang the language of the text. The
	// answer is checked with HasGlyph, and the LastResort font is never used. It may be nil.
	SystemFallback func(family string, r rune, lang string) (FontFile, bool)
}

// FontFile is a font that the system names: the file, the face in it and the family.
//
// Flutter: no counterpart
type FontFile struct {
	Path   string
	Index  int
	Family string
}

// family is the faces of one family.
type family struct {
	name  string
	faces []Face
}

// Matcher chooses the system fonts of macOS from the list of installed faces and the answers of Core Text
// (SPEC §29.8). Build it with New. It is safe for concurrent use.
//
// Flutter: no counterpart
type Matcher struct {
	hasGlyph func(path string, index int, r rune) bool
	// covers is hasGlyph that also says whether it could tell: the default one knows a font it cannot read.
	covers   func(path string, index int, r rune) (has, known bool)
	locale   string
	fallback func(family string, r rune, lang string) (FontFile, bool)
	// listed are the installed faces, for Covers.
	listed map[faceKey]bool
	// byKey finds a family by its normalized name; sorted is the families by name.
	byKey  map[string]*family
	sorted []*family
}

// New builds the matcher from the installed faces and the system UI font. It fails with
// DarwinFontsUnavailable when there are no faces at all.
//
// Flutter: no counterpart
func New(opts Options) (*Matcher, error) {
	faces := dedupe(opts.Faces)
	m := &Matcher{hasGlyph: opts.HasGlyph, locale: opts.Locale, fallback: opts.SystemFallback,
		byKey: map[string]*family{}, listed: map[faceKey]bool{}}
	if m.hasGlyph == nil {
		checker := newGlyphChecker()
		m.hasGlyph, m.covers = checker.hasGlyph, checker.covers
	} else {
		hasGlyph := m.hasGlyph // the app's HasGlyph always tells
		m.covers = func(path string, index int, r rune) (bool, bool) { return hasGlyph(path, index, r), true }
	}
	if opts.SystemUI != nil {
		if sys, ok := opts.SystemUI(opts.Locale); ok {
			faces = append(faces, systemFace(sys))
		}
	}
	if len(faces) == 0 {
		return nil, errors.New(TagUnavailable, "no installed font could be read")
	}
	for _, f := range faces {
		m.listed[faceKey{f.Path, f.Index}] = true
		key := font.NormalizeFamily(f.Family)
		fam := m.byKey[key]
		if fam == nil {
			fam = &family{name: f.Family}
			m.byKey[key] = fam
			m.sorted = append(m.sorted, fam)
		}
		fam.faces = append(fam.faces, f)
	}
	slices.SortFunc(m.sorted, func(a, b *family) int { return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name)) })
	return m, nil
}

// systemFace is the face of the system UI font: the family SystemUIFamily, and the weight and slant of the
// file when it can be read (a variable font has the weight of its default instance, and the layout sets the
// weight axis).
func systemFace(f FontFile) Face {
	face := Face{Family: SystemUIFamily, Path: f.Path, Index: f.Index, Weight: 400}
	if described, err := Describe(f.Path); err == nil {
		for _, d := range described {
			if d.Index == f.Index {
				face.Weight, face.Italic = d.Weight, d.Italic
			}
		}
	}
	return face
}

// find returns the installed family of a name, nil for none.
func (m *Matcher) find(name string) *family { return m.byKey[font.NormalizeFamily(name)] }

// firstInstalled returns the first of the names that is installed.
func (m *Matcher) firstInstalled(names []string) *family {
	for _, n := range names {
		if fam := m.find(n); fam != nil {
			return fam
		}
	}
	return nil
}

// chain returns the installed families of the fallback chain in the order for the language: the families
// written for it first, the others in the order of the chain.
func (m *Matcher) chain(lang string) []*family {
	want := langKey(lang)
	if want == "" {
		want = langKey(m.locale)
	}
	var first, rest []*family
	seen := map[*family]bool{}
	for _, e := range fallbackChain {
		fam := m.find(e.Family)
		if fam == nil || seen[fam] {
			continue
		}
		seen[fam] = true
		if want != "" && e.Lang == want {
			first = append(first, fam)
		} else {
			rest = append(rest, fam)
		}
	}
	return append(first, rest...)
}

// head is the family a request starts with: the family itself, or the first installed font of a generic
// family, or the first installed target of an alias. generic says the name was a generic family.
func (m *Matcher) head(name string) (fam *family, generic bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		lower = "sans-serif"
	}
	if names, ok := generics[lower]; ok {
		return m.firstInstalled(names), true
	}
	if fam := m.find(lower); fam != nil {
		return fam, false
	}
	return m.firstInstalled(aliases[lower]), false
}

// faceKey identifies a face.
type faceKey struct {
	path  string
	index int
}

// Match returns the fonts for the family, the OpenType weight (400 regular), the slant and the BCP 47
// language, best first: the family itself (its face nearest to the weight and slant), then, for a generic
// family (sans-serif, serif, monospace, cursive, fantasy, system-ui, math and emoji), the families of the
// fallback chain, each with its nearest face, in the order for the language. A family that is not generic
// gives its own face only, or that of the family an alias stands for (Segoe UI is San Francisco); one that
// is not installed gives none and no error, so the caller can go on to the next family. The empty family and
// SystemUIFamily are the system UI font. It fails with DarwinFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Match(ctx context.Context, name string, weight int, italic bool, lang string) ([]text.FontFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err).SetMetadata("family", name)
	}
	head, generic := m.head(name)
	var out []text.FontFile
	seen := map[faceKey]bool{}
	add := func(fam *family) {
		if f := nearest(fam.faces, weight, italic); f != nil && !seen[faceKey{f.Path, f.Index}] {
			seen[faceKey{f.Path, f.Index}] = true
			out = append(out, text.FontFile{Path: f.Path, Index: f.Index, Family: fam.name})
		}
	}
	if head != nil {
		add(head)
	}
	if generic {
		for _, fam := range m.chain(lang) {
			add(fam)
		}
	}
	return out, nil
}

// Fallback returns the font that has a glyph for r: the font of the family (when it is installed), then,
// for an emoji, Apple Color Emoji, then the font that Core Text chooses, then the families of the fallback
// chain in the order for the language. For each family only the face nearest to the weight and slant is
// asked. It is false when no font has the character. It fails with DarwinFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Fallback(ctx context.Context, name string, weight int, italic bool, lang string, r rune) (text.FontFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return text.FontFile{}, false, errors.NewErr(tagCanceled, err).SetMetadata("family", name)
	}
	try := func(fam *family) (text.FontFile, bool) {
		f := nearest(fam.faces, weight, italic)
		if f != nil && m.hasGlyph(f.Path, f.Index, r) {
			return text.FontFile{Path: f.Path, Index: f.Index, Family: fam.name}, true
		}
		return text.FontFile{}, false
	}
	head, _ := m.head(name)
	if head != nil {
		if f, ok := try(head); ok {
			return f, true, nil
		}
	}
	if text.IsEmoji(r) {
		if emoji := m.find("Apple Color Emoji"); emoji != nil {
			if f, ok := try(emoji); ok {
				return f, true, nil
			}
		}
	}
	if m.fallback != nil {
		asked := ""
		if head != nil && head.name != SystemUIFamily {
			asked = head.name
		}
		if c, ok := m.fallback(asked, r, lang); ok && !isLastResort(c) && m.hasGlyph(c.Path, c.Index, r) {
			return text.FontFile{Path: c.Path, Index: c.Index, Family: c.Family}, true, nil
		}
	}
	for _, fam := range m.chain(lang) {
		if f, ok := try(fam); ok {
			return f, true, nil
		}
	}
	return text.FontFile{}, false, nil
}

// isLastResort says that the font is Apple's LastResort, whose glyphs are boxes that name the block of the
// character: a font that has a glyph for everything and so never says that a character is missing.
func isLastResort(f FontFile) bool {
	return strings.Contains(strings.ToLower(f.Path), "lastresort") || strings.Contains(strings.ToLower(f.Family), "lastresort")
}

// Covers implements text.Coverer with the reading of the font's cmap that Fallback uses (Options.HasGlyph),
// so that the text layout does not parse a font that cannot show a character. It knows only the installed
// faces and the system UI font, and is unknown for a font it cannot read.
//
// Flutter: no counterpart
func (m *Matcher) Covers(_ context.Context, f text.FontFile, r rune) (has, known bool) {
	if !m.listed[faceKey{f.Path, f.Index}] {
		return false, false
	}
	return m.covers(f.Path, f.Index, r)
}

// Aliases returns the installed families that the name stands for, most preferred first: San Francisco for
// Segoe UI. A name that is not an alias, or an alias with no installed target, has none. It fails with
// DarwinFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Aliases(ctx context.Context, name string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err).SetMetadata("family", name)
	}
	var out []string
	for _, target := range aliases[strings.ToLower(strings.TrimSpace(name))] {
		if fam := m.find(target); fam != nil {
			out = append(out, fam.name)
		}
	}
	return out, nil
}

// Families returns the names of the installed families, sorted without regard to case. The families whose
// names start with a dot are the fonts of the system itself (San Francisco, the interface fonts of the CJK
// and Arabic scripts), which a user does not choose by name: they are not listed, but they are matched. It
// fails with DarwinFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Families(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err)
	}
	out := make([]string, 0, len(m.sorted))
	for _, fam := range m.sorted {
		if !strings.HasPrefix(fam.name, ".") {
			out = append(out, fam.name)
		}
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagUnavailable is the tag of New when there are no installed faces.
	//
	// Flutter: no counterpart
	TagUnavailable = "DarwinFontsUnavailable"

	tagCanceled = "DarwinFontsCanceled"
)
