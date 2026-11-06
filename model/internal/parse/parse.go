// Package parse has the small text readers that the loaders of package model share: fields, integers and decimal numbers read from
// byte slices with no allocation (an OBJ file of a million triangles has some seven million of them).
package parse

import (
	"math"
	"strconv"
)

// IsSpace is true for the bytes that separate fields: a space, a tab and a carriage return.
func IsSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f' }

// Field returns the first field of the line (the bytes up to the next separator) and the rest of the line after it, or a nil
// field when the line has no more.
func Field(line []byte) (field, rest []byte) {
	i := 0
	for i < len(line) && IsSpace(line[i]) {
		i++
	}
	if i == len(line) {
		return nil, nil
	}
	j := i
	for j < len(line) && !IsSpace(line[j]) {
		j++
	}
	return line[i:j], line[j:]
}

// Trim removes the separators at both ends of the line.
func Trim(line []byte) []byte {
	for len(line) > 0 && IsSpace(line[0]) {
		line = line[1:]
	}
	for len(line) > 0 && IsSpace(line[len(line)-1]) {
		line = line[:len(line)-1]
	}
	return line
}

var pow10 = [...]float64{1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22}

// Float reads a decimal number such as -12, .5, 3.25e-4 or 1E+3. The common numbers (at most 15 digits and a small exponent) are
// exact by the method of Clinger and take no allocation; the others go to strconv. It is false when the field is not a number.
// "inf" and "nan" are numbers: the caller says whether it takes them.
func Float(b []byte) (float64, bool) {
	i, n := 0, len(b)
	if n == 0 {
		return 0, false
	}
	neg := false
	switch b[0] {
	case '-':
		neg = true
		i++
	case '+':
		i++
	}
	var mant uint64
	digits, exp10 := 0, 0
	seen := false
	for ; i < n && b[i] >= '0' && b[i] <= '9'; i++ {
		seen = true
		if digits < 19 {
			mant = mant*10 + uint64(b[i]-'0')
			if mant != 0 {
				digits++
			}
		} else {
			exp10++
		}
	}
	if i < n && b[i] == '.' {
		i++
		for ; i < n && b[i] >= '0' && b[i] <= '9'; i++ {
			seen = true
			if digits < 19 {
				mant = mant*10 + uint64(b[i]-'0')
				if mant != 0 {
					digits++
				}
				exp10--
			}
		}
	}
	if !seen {
		return slow(b)
	}
	if i < n && (b[i] == 'e' || b[i] == 'E') {
		i++
		eneg := false
		if i < n && (b[i] == '-' || b[i] == '+') {
			eneg = b[i] == '-'
			i++
		}
		if i == n {
			return 0, false
		}
		e := 0
		for ; i < n && b[i] >= '0' && b[i] <= '9'; i++ {
			if e < 10000 {
				e = e*10 + int(b[i]-'0')
			}
		}
		if eneg {
			e = -e
		}
		exp10 += e
	}
	if i != n {
		return slow(b)
	}
	if digits > 15 || mant >= 1<<53 || exp10 < -22 || exp10 > 22 {
		return slow(b)
	}
	f := float64(mant)
	if exp10 < 0 {
		f /= pow10[-exp10]
	} else {
		f *= pow10[exp10]
	}
	if neg {
		f = -f
	}
	return f, true
}

func slow(b []byte) (float64, bool) {
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		// A number out of range is +Inf or -Inf with a range error: the caller sees it as not finite.
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return f, true
		}
		return 0, false
	}
	return f, true
}

// Finite is true when f is neither infinite nor NaN.
func Finite(f float64) bool { return !math.IsInf(f, 0) && !math.IsNaN(f) }

// Int reads a decimal integer with an optional sign. It is false for anything else and for a number that does not fit an int32.
func Int(b []byte) (int, bool) {
	i, n := 0, len(b)
	if n == 0 {
		return 0, false
	}
	neg := false
	switch b[0] {
	case '-':
		neg = true
		i++
	case '+':
		i++
	}
	if i == n || n-i > 10 {
		return 0, false
	}
	v := 0
	for ; i < n; i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + int(c-'0')
	}
	if neg {
		v = -v
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, false
	}
	return v, true
}
