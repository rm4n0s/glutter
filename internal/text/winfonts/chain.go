package winfonts

import "strings"

// The tables of this file are the choices of Windows that SPEC §25.8 lists: what the generic
// families stand for, the fallback chain by script and language, and the names that stand for
// another family.

// generics are the families a generic family name stands for, the first installed one first. The
// first of each is the font of Windows (SPEC §25.8); the ones after it are the stand-ins of a
// machine that lacks it (Wine, Windows Server Core): Arial, Times New Roman and Courier New are the
// fonts the classic Windows applications ask for, and Liberation and DejaVu are the fonts that
// Wine's prefix has next to them.
var generics = map[string][]string{
	"sans-serif": {"Segoe UI", "Arial", "Tahoma", "Liberation Sans", "DejaVu Sans"},
	"serif":      {"Times New Roman", "Georgia", "Liberation Serif", "DejaVu Serif"},
	"monospace":  {"Consolas", "Courier New", "Liberation Mono", "DejaVu Sans Mono"},
	"cursive":    {"Segoe Script", "Comic Sans MS"},
	"fantasy":    {"Impact", "Segoe UI"},
	"system-ui":  {"Segoe UI Variable Text", "Segoe UI Variable", "Segoe UI", "Arial", "Tahoma"},
	"math":       {"Cambria Math", "Segoe UI Symbol", "Segoe UI"},
	"emoji":      {"Segoe UI Emoji"},
}

// chainEntry is a family of the fallback chain. Lang is the language the family is written for (a
// BCP 47 tag with the script for Chinese): a request for that language puts the family first.
type chainEntry struct {
	Family string
	Lang   string
}

// fallbackChain is the order in which Windows' text stack tries the fonts for a character the
// first font lacks, as SPEC §25.8 lists it: the interface font, the color emoji, the symbols, the
// East Asian fonts, the fonts of the scripts that Windows has a font for, the legacy fonts, and the
// stand-ins of a machine without the fonts of Windows last.
var fallbackChain = []chainEntry{
	{"Segoe UI", ""},
	{"Segoe UI Emoji", ""},
	{"Segoe UI Symbol", ""},
	{"Microsoft YaHei", "zh-Hans"},
	{"Microsoft JhengHei", "zh-Hant"},
	{"Yu Gothic", "ja"},
	{"Meiryo", "ja"},
	{"Malgun Gothic", "ko"},
	{"Nirmala UI", ""},
	{"Leelawadee UI", ""},
	{"Gadugi", ""},
	{"Ebrima", ""},
	{"Myanmar Text", ""},
	{"Javanese Text", ""},
	{"Segoe UI Historic", ""},
	{"MS Gothic", "ja"},
	{"SimSun", "zh-Hans"},
	// The stand-ins come last: on a machine that has the fonts above they are never reached for
	// a character those have; on one that lacks them (Wine) they are the only fonts of the chain
	// (Arial and Liberation Sans have Latin, Greek, Cyrillic and Hebrew).
	{"Arial", ""},
	{"Tahoma", ""},
	{"Liberation Sans", ""},
	{"DejaVu Sans", ""},
}

// aliases are the names that stand for another family on Windows: the families the platforms
// before it used, and the logical fonts of the classic API.
var aliases = map[string][]string{
	"helvetica":          {"Arial"},
	"helvetica neue":     {"Arial"},
	"times":              {"Times New Roman"},
	"courier":            {"Courier New"},
	"ms shell dlg":       {"Microsoft Sans Serif", "Tahoma"},
	"ms shell dlg 2":     {"Tahoma"},
	"segoe ui variable":  {"Segoe UI Variable Text", "Segoe UI Variable Display", "Segoe UI Variable Small", "Segoe UI"},
	"-apple-system":      {"Segoe UI"},
	"blinkmacsystemfont": {"Segoe UI"},
	"san francisco":      {"Segoe UI"},
	"sf pro text":        {"Segoe UI"},
}

// langKey reduces a BCP 47 tag to what the chain tells apart: "ja", "ko", "zh-Hans" or "zh-Hant"
// (a Chinese tag without a script is Simplified, except for the regions that write Traditional), or
// the language alone, lower case. It is "" for no tag.
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
