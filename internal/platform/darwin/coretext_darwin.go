//go:build darwin

package darwin

import (
	"io"
	"os"
	"sync"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// This file is the part of Core Text that the font matcher of macOS needs (SPEC §29.8): the list of the
// font files, the system UI font, and the font that the system chooses for a character that another font
// lacks. Fonts are named by file and by their index in it (a .ttc holds several), the way the text layout
// opens them; Core Text itself draws nothing. Core Text is safe to call from any goroutine.

// FontFile is a font as the matcher needs to know it: the file, the face in it and the family name that
// Core Text gives it.
//
// Flutter: no counterpart
type FontFile struct {
	Path   string
	Index  int
	Family string
}

// kCTFontUIFontSystem is the UI font type of the system font (San Francisco).
const kCTFontUIFontSystem int32 = 2

// kCFURLPOSIXPathStyle is the style of CFURLCopyFileSystemPath that gives a POSIX path.
const kCFURLPOSIXPathStyle int64 = 0

// defaultFontSize is the size of the fonts made for a question about fallback: Core Text's default.
const defaultFontSize = 12.0

type ctFuncs struct {
	availableFontURLs     func() uintptr
	createUIFont          func(uiType int32, size float64, language uintptr) uintptr
	createWithName        func(name uintptr, size float64, matrix uintptr) uintptr
	createForString       func(font, str uintptr, rng cf.Range) uintptr
	copyAttribute         func(font, attribute uintptr) uintptr
	copyPostScriptName    func(font uintptr) uintptr
	copyFamilyName        func(font uintptr) uintptr
	descriptorsFromURL    func(url uintptr) uintptr
	descriptorAttribute   func(desc, attribute uintptr) uintptr
	arrayCount            func(arr uintptr) int64
	arrayAt               func(arr uintptr, i int64) uintptr
	urlPath               func(url uintptr, style int64) uintptr
	urlAttribute, nameKey uintptr
}

var coreText = sync.OnceValues(func() (*ctFuncs, error) {
	// The fonts of Core Text are made after the UI framework is loaded (see uiFramework).
	if _, err := dl.OpenFramework(uiFramework); err != nil {
		return nil, err
	}
	ct, err := dl.OpenFramework("CoreText")
	if err != nil {
		return nil, err
	}
	cfw, err := dl.OpenFramework("CoreFoundation")
	if err != nil {
		return nil, err
	}
	f := new(ctFuncs)
	for _, b := range []struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}{
		{ct, &f.availableFontURLs, "CTFontManagerCopyAvailableFontURLs"},
		{ct, &f.createUIFont, "CTFontCreateUIFontForLanguage"},
		{ct, &f.createWithName, "CTFontCreateWithName"},
		{ct, &f.createForString, "CTFontCreateForString"},
		{ct, &f.copyAttribute, "CTFontCopyAttribute"},
		{ct, &f.copyPostScriptName, "CTFontCopyPostScriptName"},
		{ct, &f.copyFamilyName, "CTFontCopyFamilyName"},
		{ct, &f.descriptorsFromURL, "CTFontManagerCreateFontDescriptorsFromURL"},
		{ct, &f.descriptorAttribute, "CTFontDescriptorCopyAttribute"},
		{cfw, &f.arrayCount, "CFArrayGetCount"},
		{cfw, &f.arrayAt, "CFArrayGetValueAtIndex"},
		{cfw, &f.urlPath, "CFURLCopyFileSystemPath"},
	} {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	for _, c := range []struct {
		dst    *uintptr
		symbol string
	}{{&f.urlAttribute, "kCTFontURLAttribute"}, {&f.nameKey, "kCTFontNameAttribute"}} {
		addr, err := ct.Symbol(c.symbol)
		if err != nil {
			return nil, err
		}
		*c.dst = *(*uintptr)(dl.Pointer(addr)) // the CFString that the exported variable holds
	}
	return f, nil
})

// CoreText is the Core Text of the process.
//
// Flutter: no counterpart
type CoreText struct{ f *ctFuncs }

// OpenCoreText loads Core Text. It fails with MacOSFrameworkMissing or SymbolMissing when the system lacks
// it or one of its functions.
//
// Flutter: no counterpart
func OpenCoreText() (*CoreText, error) {
	f, err := coreText()
	if err != nil {
		return nil, err
	}
	return &CoreText{f: f}, nil
}

// FontFiles returns the files of the fonts that are installed or activated for the user, as Core Text lists
// them (CTFontManagerCopyAvailableFontURLs), in the order it gives them and with no file twice.
//
// Flutter: no counterpart
func (c *CoreText) FontFiles() []string {
	arr := c.f.availableFontURLs()
	if arr == 0 {
		return nil
	}
	defer cf.Release(cf.Ref(arr))
	n := c.f.arrayCount(arr)
	files := make([]string, 0, n)
	seen := make(map[string]bool, n)
	for i := range n {
		if p := c.pathOf(c.f.arrayAt(arr, i)); p != "" && !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	return files
}

// SystemUIFont returns the system UI font (San Francisco), made for the language (a BCP 47 tag, "" for the
// default), as a file and a face. It is false when Core Text gives none or the font has no file.
//
// Flutter: no counterpart
func (c *CoreText) SystemUIFont(lang string) (FontFile, bool) {
	font := c.uiFont(lang)
	if font == 0 {
		return FontFile{}, false
	}
	defer cf.Release(cf.Ref(font))
	return c.fileOf(font)
}

// Fallback returns the font that Core Text chooses to show the character r where the font of the family
// lacks it (CTFontCreateForString): the family is a name that CTFontCreateWithName knows, or "" for the
// system UI font, which is made for the language when it is not "". The result may be the font of the
// family itself, or the LastResort font, which the caller decides about. It is false when Core Text gives no
// font with a file.
//
// Flutter: no counterpart
func (c *CoreText) Fallback(family string, r rune, lang string) (FontFile, bool) {
	var base uintptr
	if family == "" {
		base = c.uiFont(lang)
	} else {
		name := cf.NewString(family)
		base = c.f.createWithName(uintptr(name), defaultFontSize, 0)
		cf.Release(name)
	}
	if base == 0 {
		return FontFile{}, false
	}
	defer cf.Release(cf.Ref(base))
	str := cf.NewString(string(r))
	if str == 0 {
		return FontFile{}, false
	}
	defer cf.Release(str)
	units := int64(len(utf16.Encode([]rune{r})))
	font := c.f.createForString(base, uintptr(str), cf.Range{Location: 0, Length: units})
	if font == 0 {
		return FontFile{}, false
	}
	defer cf.Release(cf.Ref(font))
	return c.fileOf(font)
}

// uiFont makes the system UI font for the language (+1: release it); 0 when Core Text gives none.
func (c *CoreText) uiFont(lang string) uintptr {
	var language cf.Ref
	if lang != "" {
		language = cf.NewString(lang)
		defer cf.Release(language)
	}
	return c.f.createUIFont(kCTFontUIFontSystem, 0, uintptr(language))
}

// fileOf is the file, the face and the family of a Core Text font.
func (c *CoreText) fileOf(font uintptr) (FontFile, bool) {
	url := c.f.copyAttribute(font, c.f.urlAttribute)
	if url == 0 {
		return FontFile{}, false
	}
	defer cf.Release(cf.Ref(url))
	path := c.pathOf(url)
	if path == "" {
		return FontFile{}, false
	}
	family := c.f.copyFamilyName(font)
	ps := c.f.copyPostScriptName(font)
	defer cf.Release(cf.Ref(family))
	defer cf.Release(cf.Ref(ps))
	return FontFile{Path: path, Index: c.faceIndex(path, url, cf.String(cf.Ref(ps))), Family: cf.String(cf.Ref(family))}, true
}

// faceIndex is the index, in the file of the URL, of the font with the PostScript name: the position of its
// descriptor among the ones Core Text makes for the file, which is the order of the fonts of a collection.
// It is 0 for a file that is not a collection.
func (c *CoreText) faceIndex(path string, url uintptr, postScript string) int {
	if !isCollection(path) {
		// The descriptors of a variable font are its named instances, not faces of the file.
		return 0
	}
	descs := c.f.descriptorsFromURL(url)
	if descs == 0 {
		return 0
	}
	defer cf.Release(cf.Ref(descs))
	n := c.f.arrayCount(descs)
	if n <= 1 {
		return 0
	}
	for i := range n {
		name := c.f.descriptorAttribute(c.f.arrayAt(descs, i), c.f.nameKey)
		found := cf.String(cf.Ref(name)) == postScript
		cf.Release(cf.Ref(name))
		if found {
			return int(i)
		}
	}
	return 0
}

// isCollection says that the file is a font collection (.ttc, .otc), by its first four bytes.
func isCollection(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }() // read only; a failed close loses nothing
	var tag [4]byte
	_, err = io.ReadFull(f, tag[:])
	return err == nil && string(tag[:]) == "ttcf"
}

// pathOf is the POSIX path of a CFURL, "" when it has none.
func (c *CoreText) pathOf(url uintptr) string {
	s := c.f.urlPath(url, kCFURLPOSIXPathStyle)
	if s == 0 {
		return ""
	}
	defer cf.Release(cf.Ref(s))
	return cf.String(cf.Ref(s))
}
