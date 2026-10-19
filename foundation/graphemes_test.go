package foundation_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

const (
	family = "👨‍👩‍👦" // 18 bytes
	flag   = "🇬🇷"    // two regional indicators, 8 bytes
	accent = "é"    // 3 bytes
)

func TestGraphemes(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		boundaries []int
	}{
		{"empty", "", []int{0}},
		{"ascii", "abc", []int{0, 1, 2, 3}},
		{"a combining accent", "a" + accent + "b", []int{0, 1, 4, 5}},
		{"a flag", flag + "x", []int{0, 8, 9}},
		{"a family", family, []int{0, 18}},
		{"two families in a row", family + family, []int{0, 18, 36}},
		{"three families in a row", family + family + family, []int{0, 18, 36, 54}},
		{"a family between letters", "a" + family + "b", []int{0, 1, 19, 20}},
		{"a joiner that joins no emoji", "a\u200db", []int{0, 4, 5}},
		{"an emoji and a letter after a joiner", "👨\u200db", []int{0, 7, 8}},
		{"a carriage return and a line feed", "a\r\nb", []int{0, 1, 3, 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := foundation.GraphemeBoundaries(tc.text); !reflect.DeepEqual(got, tc.boundaries) {
				t.Errorf("GraphemeBoundaries = %v, want %v", got, tc.boundaries)
			}
			count := len(tc.boundaries) - 1
			if got := foundation.GraphemeCount(tc.text); got != count {
				t.Errorf("GraphemeCount = %d, want %d", got, count)
			}
			for n := 0; n <= count+1; n++ {
				want := len(tc.text)
				if n < len(tc.boundaries) {
					want = tc.boundaries[n]
				}
				if got := foundation.GraphemeOffset(tc.text, n); got != want {
					t.Errorf("GraphemeOffset(%d) = %d, want %d", n, got, want)
				}
			}
		})
	}
}
