package androidfonts

import (
	"slices"
	"testing"
)

func TestLangCandidates(t *testing.T) {
	tests := []struct {
		tag  string
		want []string
	}{
		{"ja", []string{"ja", "ja-Jpan", "und-Jpan"}},
		{"ja-JP", []string{"ja-JP", "ja-Jpan-JP", "ja-Jpan", "und-Jpan", "ja"}},
		{"ko-KR", []string{"ko-KR", "ko-Kore-KR", "ko-Kore", "und-Kore", "ko"}},
		{"zh", []string{"zh", "zh-Hans", "und-Hans"}},
		{"zh-CN", []string{"zh-CN", "zh-Hans-CN", "zh-Hans", "und-Hans", "zh"}},
		{"zh_TW", []string{"zh-TW", "zh-Hant-TW", "zh-Hant", "und-Hant", "zh"}},
		{"zh-HK", []string{"zh-HK", "zh-Hant-HK", "zh-Hant", "und-Hant", "zh"}},
		{"zh-Hant-TW", []string{"zh-Hant-TW", "zh-Hant", "und-Hant", "zh"}},
		{"zh-hans", []string{"zh-Hans", "und-Hans", "zh"}},
		{"ZH-hant", []string{"zh-Hant", "und-Hant", "zh"}},
		{"ar-EG", []string{"ar-EG", "ar-Arab-EG", "ar-Arab", "und-Arab", "ar"}},
		{"sr-Latn", []string{"sr-Latn", "und-Latn", "sr"}},
		{"en-US", []string{"en-US", "en"}},
		{"en", []string{"en"}},
		{"es-419", []string{"es-419", "es"}},
		{"und-Zsye", []string{"und-Zsye"}},
		{"ja-JP-u-ca-japanese", []string{"ja-JP", "ja-Jpan-JP", "ja-Jpan", "und-Jpan", "ja"}},
		{"de-1996", []string{"de"}},
		{"", nil},
		{"e", nil},
		{"123", nil},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := langCandidates(tt.tag); !slices.Equal(got, tt.want) {
				t.Errorf("langCandidates(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestRequestedLanguages(t *testing.T) {
	tests := []struct {
		name, lang, fallback string
		want                 []string
	}{
		{"one tag", "ja", "", []string{"ja", "ja-Jpan", "und-Jpan"}},
		{"the locale when the request has none", "", "ko", []string{"ko", "ko-Kore", "und-Kore"}},
		{"the request wins over the locale", "ja", "ko", []string{"ja", "ja-Jpan", "und-Jpan"}},
		{"a list keeps its order and no repeats", "en-US, zh-TW zh", "", []string{"en-US", "en", "zh-TW", "zh-Hant-TW", "zh-Hant", "und-Hant", "zh", "zh-Hans", "und-Hans"}},
		{"none", "", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requestedLanguages(tt.lang, tt.fallback); !slices.Equal(got, tt.want) {
				t.Errorf("requestedLanguages(%q, %q) = %q, want %q", tt.lang, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestLanguageMatches(t *testing.T) {
	tests := []struct {
		name      string
		languages []string
		tag       string
		want      bool
	}{
		{"equal", []string{"ja"}, "ja", true},
		{"the family tag extends the request", []string{"zh-Hant-TW"}, "zh-Hant", true},
		{"the request is more specific", []string{"zh-Hant"}, "zh-Hant-TW", false},
		{"the end of a subtag is the end of the match", []string{"jam"}, "ja", false},
		{"case does not matter", []string{"zh-hant"}, "zh-Hant", true},
		{"the second language", []string{"zh-Hant", "zh-Bopo"}, "zh-Bopo", true},
		{"no languages", nil, "ja", false},
		{"und script", []string{"und-Zsye"}, "und-Zsye", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := languageMatches(tt.languages, tt.tag); got != tt.want {
				t.Errorf("languageMatches(%q, %q) = %t, want %t", tt.languages, tt.tag, got, tt.want)
			}
		})
	}
}
