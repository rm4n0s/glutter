package winfonts

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
)

// Options are the parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Faces are the installed faces (Scan reads them from the files). New fails when there are none.
	Faces []Face
	// Locale is the BCP 47 tag of the language used when a request names none (the user's first
	// language); "" keeps the order of the chain.
	Locale string
	// HasGlyph says whether the face at index of a font file has a glyph for r. It is how Fallback
	// finds a font for a character. It defaults to the cmap of the font, read with go-text and kept
	// per file and index.
	HasGlyph func(path string, index int, r rune) bool
}

// family is the faces of one family.
type family struct {
	name  string
	faces []Face
}

// Matcher chooses the system fonts of Windows from the list of installed faces (SPEC §25.8). Build
// it with New. It is safe for concurrent use.
//
// Flutter: no counterpart
type Matcher struct {
	hasGlyph func(path string, index int, r rune) bool
	locale   string
	// byKey finds a family by its normalized name; sorted is the families by name.
	byKey  map[string]*family
	sorted []*family
}

// New builds the matcher from the installed faces. It fails with WindowsFontsUnavailable when there
// are no faces.
//
// Flutter: no counterpart
func New(opts Options) (*Matcher, error) {
	faces := dedupe(opts.Faces)
	if len(faces) == 0 {
		return nil, errors.New(TagUnavailable, "no installed font could be read")
	}
	m := &Matcher{hasGlyph: opts.HasGlyph, locale: opts.Locale, byKey: map[string]*family{}}
	if m.hasGlyph == nil {
		m.hasGlyph = newGlyphChecker().hasGlyph
	}
	for _, f := range faces {
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

// chain returns the installed families of the fallback chain in the order for the language: the
// families written for it first, the others in the order of the chain.
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

// head is the family a request starts with: the family itself, or the first installed font of a
// generic family, or the first installed target of an alias. generic says the name was a generic
// family.
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

// Match returns the fonts for the family, the OpenType weight (400 regular), the slant and the BCP
// 47 language, best first: the family itself (its face nearest to the weight and slant), then, for a
// generic family (sans-serif, serif, monospace, cursive, fantasy, system-ui, math and emoji), the
// families of the fallback chain, each with its nearest face, in the order for the language. A
// family that is not generic gives its own face only, or that of the family an alias stands for
// (Helvetica is Arial); one that is not installed gives none and no error, so the caller can go on
// to the next family. It fails with WindowsFontsCanceled when ctx is done.
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

// faceKey identifies a face.
type faceKey struct {
	path  string
	index int
}

// Fallback returns the first font that has a glyph for r: the font of the family (when it is
// installed), then, for an emoji, Segoe UI Emoji, then the families of the fallback chain in the
// order for the language. For each family only the face nearest to the weight and slant is asked.
// It is false when no font has the character. It fails with WindowsFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Fallback(ctx context.Context, name string, weight int, italic bool, lang string, r rune) (text.FontFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return text.FontFile{}, false, errors.NewErr(tagCanceled, err).SetMetadata("family", name)
	}
	var order []*family
	if head, _ := m.head(name); head != nil {
		order = append(order, head)
	}
	chain := m.chain(lang)
	if text.IsEmoji(r) {
		if emoji := m.find("Segoe UI Emoji"); emoji != nil {
			order = append(order, emoji)
		}
	}
	order = append(order, chain...)
	for _, fam := range order {
		f := nearest(fam.faces, weight, italic)
		if f != nil && m.hasGlyph(f.Path, f.Index, r) {
			return text.FontFile{Path: f.Path, Index: f.Index, Family: fam.name}, true, nil
		}
	}
	return text.FontFile{}, false, nil
}

// Aliases returns the installed families that the name stands for, most preferred first: Arial for
// Helvetica. A name that is not an alias, or an alias with no installed target, has none. It fails
// with WindowsFontsCanceled when ctx is done.
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

// Families returns the names of the installed families, sorted without regard to case. It fails
// with WindowsFontsCanceled when ctx is done.
//
// Flutter: no counterpart
func (m *Matcher) Families(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err)
	}
	out := make([]string, len(m.sorted))
	for i, fam := range m.sorted {
		out[i] = fam.name
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagUnavailable is the tag of New when there are no installed faces.
	//
	// Flutter: no counterpart
	TagUnavailable = "WindowsFontsUnavailable"

	tagCanceled = "WindowsFontsCanceled"
)
