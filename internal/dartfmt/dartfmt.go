// Package dartfmt formats numbers the way Dart's toString and toStringAsFixed
// do, so that text printed by ported Flutter code (diagnostics, dumps) reads
// the same.
package dartfmt

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ToStringAsFixed formats x with the given number of decimals as Dart's
// toStringAsFixed does: exact ties are rounded away from zero, where Go's
// formatting rounds them to even.
func ToStringAsFixed(x float64, digits int) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}
	s := strconv.FormatFloat(x, 'f', digits, 64)
	exact := strconv.FormatFloat(math.Abs(x), 'f', 80, 64)
	dot := strings.IndexByte(exact, '.')
	if dot < 0 || exact[dot+1+digits] != '5' || strings.Trim(exact[dot+2+digits:], "0") != "" {
		return s
	}
	// An exact tie, such as 0.25 with one decimal: round away from zero by
	// adding one unit in the last place to the truncated digits.
	r := incrementDecimal(exact[:dot+1+digits])
	r = strings.TrimSuffix(r, ".")
	if math.Signbit(x) {
		r = "-" + r
	}
	return r
}

// incrementDecimal adds one to the last digit of a decimal string such as
// "12.3", carrying as needed.
func incrementDecimal(s string) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case '.':
			continue
		case '9':
			b[i] = '0'
		default:
			b[i]++
			return string(b)
		}
	}
	return "1" + string(b)
}

// Double formats x as Dart's double.toString does: "1.0" for an integral
// value, the shortest digits that read back as x otherwise, and an exponent
// ("1e-5", "1e+21") below 1e-4 and from 1e21.
func Double(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x == 0:
		if math.Signbit(x) {
			return "-0.0"
		}
		return "0.0"
	}
	if a := math.Abs(x); a < 1e-4 || a >= 1e21 {
		s := strconv.FormatFloat(x, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		sign := exp[0]
		exp = strings.TrimLeft(exp[1:], "0")
		if sign == '-' {
			exp = "-" + exp
		} else {
			exp = "+" + exp
		}
		return mant + "e" + exp
	}
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// Duration formats d as Dart's Duration.toString does: "H:MM:SS.mmmmmm", with
// a leading "-" for a negative duration and any number of hour digits.
func Duration(d time.Duration) string {
	us := d.Microseconds()
	sign := ""
	if us < 0 {
		sign = "-"
		us = -us
	}
	hours := us / 3_600_000_000
	minutes := us / 60_000_000 % 60
	seconds := us / 1_000_000 % 60
	micros := us % 1_000_000
	return fmt.Sprintf("%s%d:%02d:%02d.%06d", sign, hours, minutes, seconds, micros)
}
