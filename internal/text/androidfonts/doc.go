// Package androidfonts finds the system fonts of an Android device the way Skia's Android font
// manager does (src/ports/SkFontMgr_android.cpp and SkFontMgr_android_parser.cpp): it reads the
// font configuration of the system (/system/etc/fonts.xml, and the fonts_customization.xml of the
// vendor), builds the families with their names, aliases, weights, styles, languages and variants,
// and answers a request for a family, a weight, a slant and a language with the fonts in the order
// the system would try them, and a request for a character with the first font that has it.
//
// The package is plain Go: it does not call the system and has no build tags, so it is tested
// on every OS with fixture files. The matcher implements text.Matcher, text.Aliaser, text.Lister
// and text.Fallbacker. The files, the existence of a file and the glyphs of a font come through
// the functions of Options, so a test does not need real fonts.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.cpp)
package androidfonts
