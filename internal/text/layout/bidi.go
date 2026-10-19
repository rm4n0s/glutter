package layout

import (
	"golang.org/x/text/unicode/bidi"
)

// resolveLevels gives each rune its embedding level (UAX #9, rules P2 to L1 without the
// explicit embeddings and isolates and without paired brackets): the base level is the
// direction the paragraph was given, not the first strong character's. A line break
// character ends a paragraph for the rules; each paragraph has the same base level.
//
// Flutter: engine/src/flutter/third_party/txt (ICU's ubidi)
func resolveLevels(runes []rune, baseRTL bool) []uint8 {
	levels := make([]uint8, len(runes))
	start := 0
	for i := 0; i <= len(runes); i++ {
		if i == len(runes) || isLineBreakRune(runes[i]) {
			end := i
			if i < len(runes) {
				end = i + 1
			}
			resolveParagraph(runes[start:end], baseRTL, levels[start:end])
			start = end
		}
	}
	return levels
}

// resolveParagraph resolves one paragraph into levels.
func resolveParagraph(runes []rune, baseRTL bool, levels []uint8) {
	n := len(runes)
	if n == 0 {
		return
	}
	base := uint8(0)
	sos := bidi.L
	if baseRTL {
		base, sos = 1, bidi.R
	}
	orig := make([]bidi.Class, n)
	t := make([]bidi.Class, n)
	for i, r := range runes {
		p, _ := bidi.LookupRune(r)
		c := p.Class()
		switch c {
		case bidi.LRE, bidi.RLE, bidi.LRO, bidi.RLO, bidi.PDF, bidi.LRI, bidi.RLI, bidi.FSI, bidi.PDI:
			c = bidi.BN // the explicit formatting characters are not supported: they are ignored
		}
		orig[i], t[i] = c, c
	}
	// W1: a nonspacing mark takes the class of the character before it.
	prev := sos
	for i := range t {
		if t[i] == bidi.NSM {
			t[i] = prev
		}
		if t[i] != bidi.BN {
			prev = t[i]
		}
	}
	// W2, W3: European numbers after an Arabic letter are Arabic numbers; Arabic letters are R.
	last := sos
	for i := range t {
		switch t[i] {
		case bidi.L, bidi.R, bidi.AL:
			last = t[i]
		case bidi.EN:
			if last == bidi.AL {
				t[i] = bidi.AN
			}
		}
	}
	for i := range t {
		if t[i] == bidi.AL {
			t[i] = bidi.R
		}
	}
	// W4: a single separator between two numbers of one type joins them.
	for i := 1; i < n-1; i++ {
		switch t[i] {
		case bidi.ES:
			if t[i-1] == bidi.EN && t[i+1] == bidi.EN {
				t[i] = bidi.EN
			}
		case bidi.CS:
			if (t[i-1] == bidi.EN || t[i-1] == bidi.AN) && t[i-1] == t[i+1] {
				t[i] = t[i-1]
			}
		}
	}
	// W5: terminators next to a European number are European numbers.
	for i := 0; i < n; {
		if t[i] != bidi.ET {
			i++
			continue
		}
		j := i
		for j < n && t[j] == bidi.ET {
			j++
		}
		if (i > 0 && t[i-1] == bidi.EN) || (j < n && t[j] == bidi.EN) {
			for k := i; k < j; k++ {
				t[k] = bidi.EN
			}
		}
		i = j
	}
	// W6: the remaining separators and terminators are neutral.
	for i := range t {
		switch t[i] {
		case bidi.ES, bidi.ET, bidi.CS:
			t[i] = bidi.ON
		}
	}
	// W7: European numbers after a left-to-right letter are left-to-right.
	last = sos
	for i := range t {
		switch t[i] {
		case bidi.L, bidi.R:
			last = t[i]
		case bidi.EN:
			if last == bidi.L {
				t[i] = bidi.L
			}
		}
	}
	// N1, N2: a run of neutrals between two characters of one direction takes it (numbers
	// count as right to left); otherwise it takes the base direction.
	strong := func(c bidi.Class) (bidi.Class, bool) {
		switch c {
		case bidi.L:
			return bidi.L, true
		case bidi.R, bidi.EN, bidi.AN:
			return bidi.R, true
		}
		return 0, false
	}
	for i := 0; i < n; {
		if _, ok := strong(t[i]); ok || t[i] == bidi.BN {
			i++
			continue
		}
		j := i
		for j < n {
			if _, ok := strong(t[j]); ok {
				break
			}
			j++
		}
		before, after := sos, sos
		if i > 0 {
			before, _ = strong(t[i-1])
		}
		if j < n {
			after, _ = strong(t[j])
		}
		dir := bidi.L
		if baseRTL {
			dir = bidi.R
		}
		if before == after {
			dir = before
		}
		for k := i; k < j; k++ {
			if t[k] != bidi.BN {
				t[k] = dir
			}
		}
		i = j
	}
	// I1, I2: the levels.
	for i := range t {
		lvl := base
		switch {
		case base%2 == 0 && t[i] == bidi.R:
			lvl = base + 1
		case base%2 == 0 && (t[i] == bidi.AN || t[i] == bidi.EN):
			lvl = base + 2
		case base%2 == 1 && (t[i] == bidi.L || t[i] == bidi.EN || t[i] == bidi.AN):
			lvl = base + 1
		}
		levels[i] = lvl
	}
	// L1: separators, and the white space before them and at the end of the paragraph, take
	// the base level.
	trailing := true
	for i := n - 1; i >= 0; i-- {
		switch orig[i] {
		case bidi.B, bidi.S:
			levels[i] = base
			trailing = true
		case bidi.WS, bidi.BN:
			if trailing {
				levels[i] = base
			}
		default:
			trailing = false
		}
	}
	// A control character that was ignored takes the level of the text before it.
	for i := range runes {
		if orig[i] == bidi.BN && i > 0 && (levels[i] != base || !trailing) {
			levels[i] = levels[i-1]
		}
	}
}
