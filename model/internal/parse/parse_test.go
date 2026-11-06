package parse_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/model/internal/parse"
)

func TestFloatIsStrconv(t *testing.T) {
	tests := []string{
		"0", "-0", "+0", "1", "-1", "12", "0.5", ".5", "5.", "-.5", "+.5e1", "1e3", "1E3", "1e+3", "1e-3", "3.25e-4", "123456789012345",
		"1234567890123456", "12345678901234567890", "0.1234567890123456789", "9007199254740993", "9007199254740992", "1e22", "1e23", "1e-22",
		"1e-23", "5e-324", "1e308", "1.7976931348623157e308", "0.000001", "00012.5000", "123.456e-2", "-0.0", "1e0", "4.9e-324",
		"3.4028235e38", "0.30000000000000004", "100000000000000000000", "0.00000000000000000001", "1.5e+22", "inf", "-Inf", "+inf",
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			want, err := strconv.ParseFloat(s, 64)
			if err != nil && !math.IsInf(want, 0) {
				t.Fatalf("strconv: %v", err)
			}
			got, ok := parse.Float([]byte(s))
			if !ok || got != want || math.Signbit(got) != math.Signbit(want) {
				t.Errorf("Float(%q) = %v, %v; strconv says %v", s, got, ok, want)
			}
		})
	}
	// NaN is a number that is not equal to itself.
	if got, ok := parse.Float([]byte("nan")); !ok || !math.IsNaN(got) {
		t.Errorf("Float(nan) = %v, %v", got, ok)
	}
	if got, ok := parse.Float([]byte("1e999")); !ok || !math.IsInf(got, 1) {
		t.Errorf("Float(1e999) = %v, %v: a number that is too big is infinite", got, ok)
	}
}

func TestFloatRejects(t *testing.T) {
	for _, s := range []string{"", "-", "+", ".", "e5", "1e", "1e+", "1e-", "1x", "x1", "1.2.3", "--1", "1 2", "0x10", "1,5", "--", "1e5e5", "١٢٣"} {
		if got, ok := parse.Float([]byte(s)); ok {
			t.Errorf("Float(%q) = %v, want a failure", s, got)
		}
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true}, {"-0", 0, true}, {"7", 7, true}, {"+7", 7, true}, {"-7", -7, true}, {"2147483647", math.MaxInt32, true},
		{"-2147483648", math.MinInt32, true}, {"2147483648", 0, false}, {"99999999999", 0, false}, {"", 0, false}, {"-", 0, false},
		{"1a", 0, false}, {"1.5", 0, false}, {"1/2", 0, false}, {" 1", 0, false},
	}
	for _, tt := range tests {
		got, ok := parse.Int([]byte(tt.in))
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("Int(%q) = %d, %v, want %d, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFieldAndTrim(t *testing.T) {
	line := []byte("  \tv  1.5\t2 \r")
	var fields []string
	for {
		var f []byte
		f, line = parse.Field(line)
		if f == nil {
			break
		}
		fields = append(fields, string(f))
	}
	if len(fields) != 3 || fields[0] != "v" || fields[1] != "1.5" || fields[2] != "2" {
		t.Errorf("fields %q", fields)
	}
	if got := string(parse.Trim([]byte(" \t a b \r"))); got != "a b" {
		t.Errorf("Trim = %q", got)
	}
	if f, rest := parse.Field(nil); f != nil || rest != nil {
		t.Error("a field in nothing")
	}
}

func FuzzFloat(f *testing.F) {
	for _, s := range []string{"0", "1.5", "-2e10", "1e400", ".", "9007199254740993", "0.1234567890123456789012", "1e-400"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		got, ok := parse.Float(b)
		if !ok {
			return
		}
		// What the fast path accepts, strconv accepts with the same value (strconv takes underscores only with base prefixes, and
		// the fast path does not take them at all).
		want, err := strconv.ParseFloat(string(b), 64)
		if err != nil {
			if ne, isNum := err.(*strconv.NumError); !isNum || ne.Err != strconv.ErrRange {
				t.Fatalf("Float(%q) = %v but strconv fails: %v", b, got, err)
			}
		}
		if got != want && !(math.IsNaN(got) && math.IsNaN(want)) {
			t.Fatalf("Float(%q) = %v, strconv says %v", b, got, want)
		}
	})
}
