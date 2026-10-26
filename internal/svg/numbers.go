package svg

import (
	"strconv"
)

// isSpace reports whether b is white space in SVG attribute values.
//
// SVG 1.1: types.html#DataTypeList
func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' }

// skipSpace returns the index of the first byte of s at or after i that is not white space.
func skipSpace(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

// skipCommaSpace skips white space and at most one comma, as the separators of SVG lists allow.
//
// SVG 1.1: types.html#DataTypeList
func skipCommaSpace(s string, i int) int {
	i = skipSpace(s, i)
	if i < len(s) && s[i] == ',' {
		i = skipSpace(s, i+1)
	}
	return i
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// scanNumber reads a number at s[i:] and returns its value and the index after it.
//
// The grammar is the one of SVG: an optional sign, digits with an optional fraction (or a fraction alone), and an
// optional exponent. An exponent marker that is not followed by digits is not part of the number ("1em" is the
// number 1 and the unit em). Go's ParseFloat is not used on the whole text because it also accepts "inf", "nan",
// hexadecimal floats and underscores.
//
// SVG 1.1: types.html#DataTypeNumber
func scanNumber(s string, i int) (v float64, next int, ok bool) {
	start := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	intStart := i
	for i < len(s) && isDigit(s[i]) {
		i++
		digits++
	}
	intEnd := i
	if i < len(s) && s[i] == '.' {
		j := i + 1
		fracDigits := 0
		for j < len(s) && isDigit(s[j]) {
			j++
			fracDigits++
		}
		if digits+fracDigits == 0 {
			return 0, start, false
		}
		i = j
		digits += fracDigits
	}
	if digits == 0 {
		return 0, start, false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j < len(s) && isDigit(s[j]) {
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			i = j
		}
	}
	// Fast path: a plain integer of up to 15 digits.
	if i == intEnd && intEnd-intStart <= 15 {
		var n int64
		for k := intStart; k < intEnd; k++ {
			n = n*10 + int64(s[k]-'0')
		}
		if s[start] == '-' {
			n = -n
		}
		return float64(n), i, true
	}
	f, err := strconv.ParseFloat(s[start:i], 64)
	if err != nil {
		// Only a range error is possible here: the text has the grammar of a number. Go returns ±Inf for it, which
		// is not a usable user-space value.
		return 0, start, false
	}
	return f, i, true
}

// parseNumber parses s, which must be one number with optional white space around it.
func parseNumber(s string) (float64, bool) {
	i := skipSpace(s, 0)
	v, j, ok := scanNumber(s, i)
	if !ok || skipSpace(s, j) != len(s) {
		return 0, false
	}
	return v, true
}

// ParseNumberList parses a list of numbers separated by white space and commas, such as the values of a
// feColorMatrix or of stdDeviation. It returns false if anything else is in the text.
//
// SVG 1.1: types.html#DataTypeList
func ParseNumberList(s string) ([]float64, bool) {
	var out []float64
	i := skipSpace(s, 0)
	for i < len(s) {
		v, j, ok := scanNumber(s, i)
		if !ok {
			return nil, false
		}
		out = append(out, v)
		i = skipCommaSpace(s, j)
	}
	return out, true
}

// parseNumberOrPercent parses a number, or a percentage that becomes a fraction ("50%" is 0.5). Opacity values and
// gradient offsets are written this way.
func parseNumberOrPercent(s string) (float64, bool) {
	i := skipSpace(s, 0)
	v, j, ok := scanNumber(s, i)
	if !ok {
		return 0, false
	}
	if j < len(s) && s[j] == '%' {
		v /= 100
		j++
	}
	if skipSpace(s, j) != len(s) {
		return 0, false
	}
	return v, true
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
