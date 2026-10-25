// Package winfonts finds the system fonts of Windows: it answers a request for a family, a weight,
// a slant and a language with the fonts in the order Windows would try them, and a request for a
// character with the first font of the fallback chain that has it (SPEC §25.8).
//
// The package is plain Go: it does not call Windows and has no build tags, so it is tested on every
// OS with lists of families and files written for the tests. The reading of the system (the font
// directories and the registry keys that list the installed fonts) is in package
// internal/platform/windows (fonts_windows.go), which gives the list of files to Scan; Scan reads
// the family, the weight and the slant of each face from the font file with go-text, as fontscan
// does. The matcher implements text.Matcher, text.Aliaser, text.Lister and text.Fallbacker.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_win_dw.cpp chooses the fonts through
// DirectWrite; Glutter does not use DirectWrite)
package winfonts
