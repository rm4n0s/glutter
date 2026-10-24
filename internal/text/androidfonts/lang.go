package androidfonts

import "strings"

// This file is the language handling of Skia's SkFontMgr_android.cpp (onMatchFamilyStyleCharacter
// walks the requested language and its parents, SkLanguage::getParent) with two additions that
// Android's own Minikin has: the script that a language is written in when the tag names none
// (zh-TW is zh-Hant-TW), and "und-<Script>" families, which fonts.xml uses for scripts.

// likelyScripts is the script of the languages that Android's fonts.xml lists families for by
// script. Chinese depends on the region (see likelyScript).
var likelyScripts = map[string]string{
	"ja": "Jpan", "ko": "Kore",
	"ar": "Arab", "fa": "Arab", "ur": "Arab", "ps": "Arab", "sd": "Arab", "ug": "Arab",
	"he": "Hebr", "yi": "Hebr",
	"hi": "Deva", "mr": "Deva", "ne": "Deva", "sa": "Deva",
	"bn": "Beng", "as": "Beng", "gu": "Gujr", "pa": "Guru", "kn": "Knda", "ml": "Mlym",
	"or": "Orya", "ta": "Taml", "te": "Telu", "si": "Sinh",
	"th": "Thai", "lo": "Laoo", "km": "Khmr", "my": "Mymr", "bo": "Tibt",
	"ka": "Geor", "hy": "Armn", "am": "Ethi", "ti": "Ethi", "el": "Grek",
	"ru": "Cyrl", "uk": "Cyrl", "bg": "Cyrl", "sr": "Cyrl", "be": "Cyrl", "kk": "Cyrl", "mk": "Cyrl", "mn": "Cyrl",
}

// likelyScript returns the script of a language and region that have no script, "" when it is not known.
func likelyScript(lang, region string) string {
	if lang == "zh" {
		switch region {
		case "TW", "HK", "MO":
			return "Hant"
		}
		return "Hans"
	}
	return likelyScripts[lang]
}

// splitTag reads the language, the script and the region of a BCP 47 tag (underscores count as
// hyphens; variants and extensions are dropped). A tag with no language gives "".
func splitTag(tag string) (lang, script, region string) {
	subs := strings.Split(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"), "-")
	if len(subs) == 0 || !isAlpha(subs[0], 2, 8) {
		return "", "", ""
	}
	lang = strings.ToLower(subs[0])
	subs = subs[1:]
	if len(subs) > 0 && isAlpha(subs[0], 4, 4) {
		script = strings.ToUpper(subs[0][:1]) + strings.ToLower(subs[0][1:])
		subs = subs[1:]
	}
	if len(subs) > 0 && (isAlpha(subs[0], 2, 2) || (len(subs[0]) == 3 && isDigits(subs[0]))) {
		region = strings.ToUpper(subs[0])
	}
	return lang, script, region
}

func isAlpha(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i] | 0x20; c < 'a' || 'z' < c {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return false
		}
	}
	return s != ""
}

// langCandidates are the language tags to look for, most specific first, for the requested tag
// (Skia walks the tag and its parents with SkLanguage::getParent): the tag itself, with its script when
// it has none and the language's script is known, the language and the script, "und" and the
// script, and the language. zh-TW gives zh-TW, zh-Hant-TW, zh-Hant, und-Hant, zh.
func langCandidates(tag string) []string {
	lang, script, region := splitTag(tag)
	if lang == "" {
		return nil
	}
	if script == "" {
		script = likelyScript(lang, region)
	}
	var out []string
	add := func(parts ...string) {
		s := strings.Join(parts, "-")
		for _, o := range out {
			if o == s {
				return
			}
		}
		out = append(out, s)
	}
	given := []string{lang}
	_, givenScript, _ := splitTag(tag)
	if givenScript != "" {
		given = append(given, givenScript)
	}
	if region != "" {
		given = append(given, region)
	}
	add(given...)
	if script != "" {
		if region != "" {
			add(lang, script, region)
		}
		add(lang, script)
		add("und", script)
	}
	if lang != "und" {
		add(lang)
	}
	return out
}

// requestedLanguages are the candidate tags of a request: the tags of lang in order (a list separated by
// commas or blanks), each with its candidates, without repeats. An empty lang gives those of fallback.
func requestedLanguages(lang, fallback string) []string {
	tags := splitLanguages(lang)
	if len(tags) == 0 {
		tags = splitLanguages(fallback)
	}
	var out []string
	seen := map[string]bool{}
	for _, tag := range tags {
		for _, c := range langCandidates(tag) {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// languageMatches says whether one of the languages of a family is the tag or a tag that
// extends it (Skia: the family's tag startsWith the requested tag; here the match ends at a hyphen, so ja
// does not match jam). Tags are compared without regard to case.
func languageMatches(languages []string, tag string) bool {
	for _, l := range languages {
		if len(l) < len(tag) || !strings.EqualFold(l[:len(tag)], tag) {
			continue
		}
		if len(l) == len(tag) || l[len(tag)] == '-' {
			return true
		}
	}
	return false
}
