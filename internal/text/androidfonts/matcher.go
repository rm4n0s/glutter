package androidfonts

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

var (
	_ text.Matcher    = (*Matcher)(nil)
	_ text.Aliaser    = (*Matcher)(nil)
	_ text.Lister     = (*Matcher)(nil)
	_ text.Fallbacker = (*Matcher)(nil)
)

// defaultFamily is the family of a request with no name, and the default of Skia's findDefaultStyleSet.
const defaultFamily = "sans-serif"

// emojiFamily is the generic name the text layout asks for the color emoji fonts with.
const emojiFamily = "emoji"

// styleSet is the fonts of one family: Skia's SkFontStyleSet_Android and its SkTypeface_AndroidSystem
// data (names, languages, variant), without the typefaces. Only the fonts whose file exists are in it.
type styleSet struct {
	// names are the names of the family in lower case; the first is the one of FontFile.Family.
	names       []string
	fonts       []face
	languages   []string
	variant     fontVariant
	fallbackFor string
	isFallback  bool
	// unnamed is true for a family with no name, which has a made-up one (Skia: "00##fallback").
	unnamed bool
	// seq is the position of the family in its map (the named families or the fallback families).
	seq int
}

// hasName says whether the family has the name (in lower case).
func (s *styleSet) hasName(name string) bool { return slices.Contains(s.names, name) }

// elegant says whether the family is used where Skia asks for the variant elegant: the variant
// default is "compact and elegant" (SkFontStyleSet_Android: kCompact | kElegant).
func (s *styleSet) elegant() bool {
	return s.variant == variantDefault || s.variant&variantElegant != 0
}

// canonical is the family name that FontFile.Family has: the first name, "" for a family with no name.
func (s *styleSet) canonical() string {
	if s.unnamed {
		return ""
	}
	return s.names[0]
}

// isEmoji says whether the family is a color emoji font: its language is und-Zsye, or (Android 8, which
// has no language on it) one of its files is a "...Emoji..." font.
func (s *styleSet) isEmoji() bool {
	if languageMatches(s.languages, "und-Zsye") {
		return true
	}
	if len(s.languages) > 0 {
		return false
	}
	for _, f := range s.fonts {
		if strings.Contains(strings.ToLower(path.Base(f.path)), "emoji") {
			return true
		}
	}
	return false
}

// Matcher chooses the system fonts of an Android device as Skia's Android font manager does. Build
// it with New. It is safe for concurrent use.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp SkFontMgr_Android)
type Matcher struct {
	logger   *slog.Logger
	hasGlyph func(path string, index int, r rune) bool
	locale   string

	// sets are the families in the order of the configuration; fallbacks are the ones with no name.
	sets      []*styleSet
	named     []*styleSet
	fallbacks []*styleSet
	// byName finds the family of a name: a named family first, then a fallback one (Skia: onMatchFamily).
	byName map[string]*styleSet
}

// New reads the font configuration and builds the matcher. Every file of FontsXML is read in
// order, then the files of Customization; the fonts whose file does not exist are left out. A file
// that is missing is skipped. It fails with AndroidFontsXMLInvalid when no file of FontsXML gave
// a family and one of them was not well formed (the error of the first such file), and with
// AndroidFontsUnavailable when no file gave a family or none of the fonts they name exists. A
// customization file with a problem is logged and skipped.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp SkFontMgr_Android::SkFontMgr_Android)
func New(opts Options) (*Matcher, error) {
	opts = opts.withDefaults()
	var families []*fontFamily
	loaded := 0
	var firstParseErr error
	for _, file := range opts.FontsXML {
		data, err := opts.ReadFile(file)
		if err != nil {
			opts.Logger.Info("font configuration not read", "file", file, "err", err)
			continue
		}
		before := len(families)
		if _, err := parseConfig(opts.Logger, file, data, &families, opts.FontDir, false); err != nil {
			opts.Logger.Error("font configuration not valid", "err", err)
			if firstParseErr == nil {
				firstParseErr = err
			}
			continue
		}
		if len(families) == before {
			opts.Logger.Warn("font configuration has no family", "file", file)
			continue
		}
		loaded++
	}
	if loaded == 0 {
		if firstParseErr != nil {
			return nil, firstParseErr
		}
		return nil, errors.New(TagUnavailable, "no font configuration file could be read", "files", opts.FontsXML)
	}
	for _, file := range opts.Customization {
		data, err := opts.ReadFile(file)
		if err != nil {
			opts.Logger.Debug("font customization not read", "file", file, "err", err)
			continue
		}
		if _, err := parseConfig(opts.Logger, file, data, &families, customizationDir(file, opts.FontDir), false); err != nil {
			opts.Logger.Error("font customization not valid", "err", err)
		}
	}
	m := &Matcher{logger: opts.Logger, hasGlyph: opts.HasGlyph, locale: opts.Locale, byName: map[string]*styleSet{}}
	m.build(families, opts.FileExists)
	if len(m.sets) == 0 {
		return nil, errors.New(TagUnavailable, "none of the fonts of the configuration exists", "files", opts.FontsXML, "dir", opts.FontDir)
	}
	return m, nil
}

// build makes the style sets: Skia's buildNameToFamilyMap and addFamily. A family is followed by the
// families its fallbackFor fonts made.
func (m *Matcher) build(families []*fontFamily, exists func(string) bool) {
	known := map[string]bool{}
	index := 0
	add := func(family *fontFamily) {
		i := index
		index++
		set := &styleSet{
			names: slices.Clone(family.Names), languages: family.Languages, variant: family.Variant,
			fallbackFor: family.FallbackFor, isFallback: family.IsFallbackFont,
		}
		if family.IsFallbackFont && len(set.names) == 0 {
			set.names = []string{fmt.Sprintf("%.2x##fallback", uint32(i))}
			set.unnamed = true
		}
		for _, f := range family.Fonts {
			if f.FileName == "" {
				continue
			}
			p := path.Join(family.BasePath, f.FileName)
			ok, seen := known[p]
			if !seen {
				ok = exists(p)
				known[p] = ok
			}
			if !ok {
				m.logger.Debug("font file does not exist", "path", p)
				continue
			}
			weight := f.Weight
			if weight == 0 {
				weight = 400 // Skia takes the weight from the font file here
			}
			set.fonts = append(set.fonts, face{path: p, index: f.Index, weight: weight, italic: f.Style == styleItalic})
		}
		if len(set.fonts) == 0 {
			return
		}
		if set.isFallback {
			set.seq = len(m.fallbacks)
			m.fallbacks = append(m.fallbacks, set)
		} else {
			set.seq = len(m.named)
			m.named = append(m.named, set)
		}
		m.sets = append(m.sets, set)
	}
	for _, family := range families {
		add(family)
		for _, fb := range family.fallbackFamilies {
			add(fb)
		}
	}
	for _, set := range m.named {
		for _, name := range set.names {
			if _, dup := m.byName[name]; !dup {
				m.byName[name] = set
			}
		}
	}
	for _, set := range m.fallbacks {
		for _, name := range set.names {
			if _, dup := m.byName[name]; !dup {
				m.byName[name] = set
			}
		}
	}
}

// find returns the family of a name in lower case, nil for none. The generic names that a
// device may not define stand for another family (fantasy for casual, math for sans-serif).
func (m *Matcher) find(name string) *styleSet {
	if set := m.byName[name]; set != nil {
		return set
	}
	switch name {
	case "fantasy":
		if set := m.byName["casual"]; set != nil {
			return set
		}
		return m.byName[defaultFamily]
	case "math":
		return m.byName[defaultFamily]
	}
	return nil
}

// defaultSet is the family of a request with no name: sans-serif, else the first family
// (Skia: findDefaultStyleSet).
func (m *Matcher) defaultSet() *styleSet {
	if set := m.byName[defaultFamily]; set != nil {
		return set
	}
	return m.sets[0]
}

// ordered returns all the families in the order that Skia's onMatchFamilyStyleCharacter tries them for the family
// name (in lower case, "" for none) and the languages: the variant elegant first (the families with the
// variant default are elegant and compact), then the families for the first language tag, its more general tags, and
// the other languages last; in each of these the named family, the fallback families made for it
// (fallbackFor), the fallback families for no family, the other named families, the fallback families
// for another family; and the order of the configuration after that.
func (m *Matcher) ordered(familyName string, langs []string) []*styleSet {
	type ranked struct {
		set                      *styleSet
		variant, lang, step, seq int
	}
	rs := make([]ranked, 0, len(m.sets))
	for _, s := range m.sets {
		r := ranked{set: s, lang: len(langs), seq: s.seq}
		if !s.elegant() {
			r.variant = 1
		}
		for i, l := range langs {
			if languageMatches(s.languages, l) {
				r.lang = i
				break
			}
		}
		switch {
		case !s.isFallback && familyName != "" && s.hasName(familyName):
			r.step = 1
		case !s.isFallback:
			r.step = 4
		case familyName != "" && s.fallbackFor == familyName:
			r.step = 2
		case s.fallbackFor == "":
			r.step = 3
		default:
			r.step = 5
		}
		rs = append(rs, r)
	}
	slices.SortStableFunc(rs, func(a, b ranked) int {
		for _, d := range [...]int{a.variant - b.variant, a.lang - b.lang, a.step - b.step, a.seq - b.seq} {
			if d != 0 {
				return d
			}
		}
		return 0
	})
	out := make([]*styleSet, len(rs))
	for i, r := range rs {
		out[i] = r.set
	}
	return out
}

// Match returns the fonts for the family, the OpenType weight (400 regular), the slant and the BCP
// 47 language, best first: the family itself (its font nearest to the weight and slant, found by
// its name or by an alias), then the families of the fallback chain, each with its nearest font, in
// the order Skia's onMatchFamilyStyleCharacter tries them for the language: the families written
// for the language and its more general tags first (Japanese fonts for ja, Traditional Chinese
// for zh-TW), then the others in the order of the configuration. A font that is in the list once
// is not repeated: the fonts of a variable font file that fonts.xml lists once per weight are one
// font (the caller sets the weight with a variation). The names are those of the configuration
// (sans-serif, serif, monospace, sans-serif-condensed, casual, arial...), without regard to
// case; "" is sans-serif. An unknown family gives no fonts and no error: Skia returns no typeface for
// it either, so that the app can try another family; Fallback finds a font for a character in any
// case. The generic name emoji gives the color emoji fonts first. The FontFile.Family of a font is the
// first name of its family, "" for a family with no name. It fails with AndroidFontsCanceled when ctx is done.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp SkFontMgr_Android::onMatchFamilyStyle)
func (m *Matcher) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]text.FontFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err).SetMetadata("family", family)
	}
	name := strings.ToLower(strings.TrimSpace(family))
	if name == "" {
		name = defaultFamily
	}
	var head *styleSet
	if name != emojiFamily {
		head = m.find(name)
		if head == nil && text.IsGenericFamily(name) {
			head = m.defaultSet()
		}
		if head == nil {
			return nil, nil
		}
	}
	langs := requestedLanguages(lang, m.locale)
	var chain []*styleSet
	if name == emojiFamily {
		// The emoji fonts first, then the chain of the default family.
		chain = slices.SortedStableFunc(slices.Values(m.ordered(defaultFamily, langs)), func(a, b *styleSet) int {
			return boolInt(!a.isEmoji()) - boolInt(!b.isEmoji())
		})
	} else {
		chain = m.ordered(name, langs)
	}
	var out []text.FontFile
	seen := map[face]bool{}
	add := func(s *styleSet) {
		f := matchStyle(s.fonts, weight, italic)
		if f == nil {
			return
		}
		key := face{path: f.path, index: f.index}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, text.FontFile{Path: f.path, Index: f.index, Family: s.canonical()})
	}
	if head != nil {
		add(head)
	}
	for _, s := range chain {
		if s != head {
			add(s)
		}
	}
	return out, nil
}

// boolInt is 1 for true.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Fallback returns the first font that has a glyph for r, in the order of Skia's
// onMatchFamilyStyleCharacter for the family and the language (see Match; unlike Match it does not
// put the family first: it is the character's turn). For each family only the font nearest to the
// weight and slant is asked, with Options.HasGlyph. The family may be unknown or "". It is false
// when no font has the character. It fails with AndroidFontsCanceled when ctx is done.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp SkFontMgr_Android::onMatchFamilyStyleCharacter)
func (m *Matcher) Fallback(ctx context.Context, family string, weight int, italic bool, lang string, r rune) (text.FontFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return text.FontFile{}, false, errors.NewErr(tagCanceled, err).SetMetadata("family", family)
	}
	name := strings.ToLower(strings.TrimSpace(family))
	if name == emojiFamily {
		name = ""
	}
	for _, s := range m.ordered(name, requestedLanguages(lang, m.locale)) {
		f := matchStyle(s.fonts, weight, italic)
		if f != nil && m.hasGlyph(f.path, f.index, r) {
			return text.FontFile{Path: f.path, Index: f.index, Family: s.canonical()}, true, nil
		}
	}
	return text.FontFile{}, false, nil
}

// Aliases returns the other names of the family that has the name, its first name first: for arial
// that is sans-serif and the other names fonts.xml gives it (helvetica, tahoma, verdana...). A name
// that is not a family, or a family with one name, has none. It fails with AndroidFontsCanceled
// when ctx is done.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android_parser.cpp lmpParser::aliasHandler)
func (m *Matcher) Aliases(ctx context.Context, family string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err).SetMetadata("family", family)
	}
	name := strings.ToLower(strings.TrimSpace(family))
	set := m.byName[name]
	if set == nil || set.unnamed {
		return nil, nil
	}
	var out []string
	for _, n := range set.names {
		if n != name && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// Families returns the names of the named families, the aliases included, in the order of the configuration
// without repeats (Skia: onCountFamilies and onGetFamilyName). It fails with AndroidFontsCanceled when
// ctx is done.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp SkFontMgr_Android::onGetFamilyName)
func (m *Matcher) Families(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagCanceled, err)
	}
	var out []string
	for _, set := range m.named {
		for _, n := range set.names {
			if !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagUnavailable is the tag of New when no font configuration file gave a family or none
	// of the fonts the files name exists.
	//
	// Flutter: no counterpart
	TagUnavailable = "AndroidFontsUnavailable"

	tagCanceled = "AndroidFontsCanceled"
)
