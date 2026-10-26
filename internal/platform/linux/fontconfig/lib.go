//go:build linux && !android

package fontconfig

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of
// libfontconfig, for setups such as NixOS (SPEC §8.1).
const LibraryEnv = "GLUTTER_FONTCONFIG_LIBRARY"

// Constants from fontconfig.h.
const (
	fcTrue         = 1
	fcMatchPattern = 0 // FcMatchPattern
	fcResultMatch  = 0 // FcResultMatch
)

// The object names of fontconfig.h (FC_FAMILY and the like).
const (
	objFamily = "family"
	objSlant  = "slant"
	objWeight = "weight"
	objFile   = "file"
	objIndex  = "index"
	objLang   = "lang"
	// objCharSet is FC_CHARSET: the characters a font has.
	objCharSet = "charset"
)

// library holds the bound functions of libfontconfig. Pointers (FcConfig*,
// FcPattern*, FcFontSet*) are uintptr; the strings that go in are C strings.
type library struct {
	initLoadConfigAndFonts func() uintptr
	configDestroy          func(config uintptr)
	patternCreate          func() uintptr
	patternDestroy         func(pat uintptr)
	patternAddString       func(pat, object, value uintptr) int32
	patternAddInteger      func(pat, object uintptr, value int32) int32
	configSubstitute       func(config, pat uintptr, kind int32) int32
	defaultSubstitute      func(pat uintptr)
	fontSort               func(config, pat uintptr, trim int32, csp, result uintptr) uintptr
	patternGetString       func(pat, object uintptr, n int32, out uintptr) int32
	patternGetInteger      func(pat, object uintptr, n int32, out uintptr) int32
	fontSetDestroy         func(set uintptr)
	objectSetCreate        func() uintptr
	objectSetAdd           func(set, object uintptr) int32
	objectSetDestroy       func(set uintptr)
	fontList               func(config, pat, objectSet uintptr) uintptr
	patternGetCharSet      func(pat, object uintptr, n int32, out uintptr) int32
	charSetCopy            func(cs uintptr) uintptr
	charSetDestroy         func(cs uintptr)
	charSetHasChar         func(cs uintptr, ucs4 uint32) int32
}

// loadLibrary opens libfontconfig.so.1 and binds the functions. It fails with
// FontconfigLoadFailed when the library cannot be loaded, and with SymbolMissing
// when a function is absent.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libfontconfig.so.1", "libfontconfig.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(tagFontconfigLoadFailed, "could not load libfontconfig",
			"names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.initLoadConfigAndFonts, "FcInitLoadConfigAndFonts"},
		{&l.configDestroy, "FcConfigDestroy"},
		{&l.patternCreate, "FcPatternCreate"},
		{&l.patternDestroy, "FcPatternDestroy"},
		{&l.patternAddString, "FcPatternAddString"},
		{&l.patternAddInteger, "FcPatternAddInteger"},
		{&l.configSubstitute, "FcConfigSubstitute"},
		{&l.defaultSubstitute, "FcDefaultSubstitute"},
		{&l.fontSort, "FcFontSort"},
		{&l.patternGetString, "FcPatternGetString"},
		{&l.patternGetInteger, "FcPatternGetInteger"},
		{&l.fontSetDestroy, "FcFontSetDestroy"},
		{&l.objectSetCreate, "FcObjectSetCreate"},
		{&l.objectSetAdd, "FcObjectSetAdd"},
		{&l.objectSetDestroy, "FcObjectSetDestroy"},
		{&l.fontList, "FcFontList"},
		{&l.patternGetCharSet, "FcPatternGetCharSet"},
		{&l.charSetCopy, "FcCharSetCopy"},
		{&l.charSetDestroy, "FcCharSetDestroy"},
		{&l.charSetHasChar, "FcCharSetHasChar"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontconfigLoadFailed = "FontconfigLoadFailed"
)
