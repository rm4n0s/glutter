package darwinfonts

import "strings"

// SystemUIFamily is the name that the system UI font (San Francisco) has in the matcher and in the Material
// typography of macOS: the name Core Text gives its family. The matcher makes the family from the file that
// Core Text names (Options.SystemUI), because the font is not in the list of the installed fonts.
//
// Flutter: no counterpart
const SystemUIFamily = ".AppleSystemUIFont"

// The tables of this file are the choices of macOS that SPEC §29.8 lists: what the generic families stand
// for, the fallback chain by script and language, and the names that stand for another family.

// generics are the families a generic family name stands for, the first installed one first. The first of
// each is the font of macOS (SPEC §29.8): San Francisco for the interface, Times for serif, Menlo for
// monospace.
var generics = map[string][]string{
	"sans-serif": {SystemUIFamily, "Helvetica Neue", "Helvetica", "Arial"},
	"serif":      {"Times", "Times New Roman", "Georgia", "Palatino"},
	"monospace":  {"SF Mono", "Menlo", "Monaco", "Courier New", "Courier"},
	"cursive":    {"Apple Chancery", "Snell Roundhand", "Zapfino"},
	"fantasy":    {"Papyrus", "Chalkduster", SystemUIFamily},
	"system-ui":  {SystemUIFamily, "Helvetica Neue", "Helvetica"},
	"math":       {"STIX Two Math", "Apple Symbols", SystemUIFamily},
	"emoji":      {"Apple Color Emoji"},
}

// chainEntry is a family of the fallback chain. Lang is the language the family is written for (a BCP 47 tag
// with the script for Chinese): a request for that language puts the family first.
type chainEntry struct {
	Family string
	Lang   string
}

// fallbackChain is the order in which the matcher tries the fonts of macOS for a character that the font of
// the request and Core Text's own choice lack: the interface font, the color emoji, the symbols, the East
// Asian fonts, then the fonts of the scripts that macOS has a font for. Core Text's cascade list is asked
// before this chain, so the chain matters when Core Text is not (the tests, a machine where it fails) and when
// its choice is a font that cannot be drawn: macOS 13 and later keep PingFang in Apple's hvgl format, which
// go-text cannot read, and Hiragino Sans GB, Heiti and Songti are the Chinese fonts that it can.
var fallbackChain = []chainEntry{
	{SystemUIFamily, ""},
	{"Apple Color Emoji", ""},
	{"Apple Symbols", ""},
	{"PingFang SC", "zh-Hans"},
	{"PingFang TC", "zh-Hant"},
	{"Hiragino Sans GB", "zh-Hans"},
	{"Heiti SC", "zh-Hans"},
	{"Songti SC", "zh-Hans"},
	{"Heiti TC", "zh-Hant"},
	{"Songti TC", "zh-Hant"},
	{"Hiragino Sans", "ja"},
	{"Hiragino Kaku Gothic ProN", "ja"},
	{"Apple SD Gothic Neo", "ko"},
	{"Helvetica Neue", ""},
	{"Lucida Grande", ""},
	{"Geeza Pro", ""},
	{"Arial Hebrew", ""},
	{"Kohinoor Devanagari", ""},
	{"Thonburi", ""},
	{"Arial Unicode MS", ""},
	{"Arial", ""},
}

// aliases are the names that stand for another family on macOS: the names of the system font that the web
// and the other platforms use, and the names of the fonts of Windows that a Mac replaces.
var aliases = map[string][]string{
	"system-ui":          {SystemUIFamily, "Helvetica Neue"},
	"-apple-system":      {SystemUIFamily},
	"blinkmacsystemfont": {SystemUIFamily},
	"san francisco":      {SystemUIFamily},
	"sf pro":             {SystemUIFamily},
	"sf pro text":        {SystemUIFamily},
	"sf pro display":     {SystemUIFamily},
	"sf ui text":         {SystemUIFamily},
	// The families of the Cupertino text styles (SPEC §31.2): Flutter's iOS engine maps them to San
	// Francisco Text and Display; the system UI font is both here, at the optical size that its file
	// has by default.
	"cupertinosystemtext":    {SystemUIFamily},
	"cupertinosystemdisplay": {SystemUIFamily},
	"segoe ui":               {SystemUIFamily, "Helvetica Neue"},
	"ms shell dlg":           {"Helvetica Neue", "Helvetica"},
	"ms shell dlg 2":         {"Helvetica Neue", "Helvetica"},
	"consolas":               {"Menlo", "Monaco"},
	"times":                  {"Times New Roman"},
	"courier":                {"Courier New"},
}

// langKey reduces a BCP 47 tag to what the chain tells apart: "ja", "ko", "zh-Hans" or "zh-Hant" (a Chinese
// tag without a script is Simplified, except for the regions that write Traditional), or the language
// alone, lower case. It is "" for no tag.
func langKey(tag string) string {
	tag = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"))
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, "-")
	if parts[0] != "zh" {
		return parts[0]
	}
	for _, p := range parts[1:] {
		switch p {
		case "hant", "tw", "hk", "mo":
			return "zh-Hant"
		case "hans", "cn", "sg":
			return "zh-Hans"
		}
	}
	return "zh-Hans"
}
