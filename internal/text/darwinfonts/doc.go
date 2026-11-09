// Package darwinfonts finds the system fonts of macOS: it answers a request for a family, a weight, a
// slant and a language with the fonts in the order a Mac would try them, and a request for a character with
// the font that Core Text chooses for it, or the first font of a fallback chain that has it (SPEC §29.8).
//
// The package is plain Go: it does not call macOS and has no build tags, so it is tested on every OS with
// lists of families and files written for the tests. The reading of the system (the list of the font files,
// the system UI font and the fallback of Core Text) is in internal/platform/darwin (coretext_darwin.go),
// which engine.SystemFontMatcher gives to New through Options; Scan reads the family, the weight and the
// slant of each face from the font file with go-text, as fontscan does. The matcher implements text.Matcher,
// text.Aliaser, text.Lister, text.Fallbacker and text.Coverer.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_mac_ct.cpp chooses the fonts through Core Text;
// Glutter reads the files itself and asks Core Text only for what the files do not tell)
package darwinfonts
