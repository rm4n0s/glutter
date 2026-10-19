package ui

import "strings"

// ParseLocale reads a locale from a BCP 47 language tag ("en-US",
// "zh-Hans-CN") or a POSIX locale name ("en_US.UTF-8", "de_DE@euro"). The
// second result is false for a name that holds no language, such as "C" or
// "POSIX" or an empty string.
//
// Flutter: no counterpart (the embedder parses the platform's locale names)
func ParseLocale(tag string) (Locale, bool) {
	// A POSIX name carries an encoding and a modifier after the locale itself.
	if i := strings.IndexAny(tag, ".@"); i >= 0 {
		tag = tag[:i]
	}
	tag = strings.ReplaceAll(tag, "_", "-")
	parts := strings.Split(tag, "-")
	lang := strings.ToLower(parts[0])
	if len(lang) < 2 || len(lang) > 3 || lang == "c" || lang == "posix" || !isLetters(lang) {
		return Locale{}, false
	}
	l := Locale{LanguageCode: lang}
	for _, p := range parts[1:] {
		switch {
		case len(p) == 4 && isLetters(p) && l.ScriptCode == "" && l.CountryCode == "":
			l.ScriptCode = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		case (len(p) == 2 && isLetters(p)) || (len(p) == 3 && isDigits(p)):
			if l.CountryCode == "" {
				l.CountryCode = strings.ToUpper(p)
			}
		}
	}
	return l, true
}

func isLetters(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i] | 0x20; c < 'a' || c > 'z' {
			return false
		}
	}
	return s != ""
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}
