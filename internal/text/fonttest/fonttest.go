// Package fonttest makes test fonts from the fonts that the module bundles: the same outlines under
// another character map, so that a test can have many fonts that cover different characters, and a font
// whose character map cannot be read, without keeping font files in the repository.
//
// It is for tests only.
package fonttest

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Range returns n runes starting at from.
//
// Flutter: no counterpart
func Range(from rune, n int) []rune {
	out := make([]rune, n)
	for i := range out {
		out[i] = from + rune(i)
	}
	return out
}

// WithCmap returns a copy of the TrueType font ttf in which the character map maps exactly the given
// runes (in increasing order), the i-th of them to glyph i+1 of the font. The other tables are the
// font's, so the copy is as large and as slow to parse as the font is.
//
// Flutter: no counterpart
func WithCmap(tb testing.TB, ttf []byte, runes []rune) []byte {
	tb.Helper()
	return Replace(tb, ttf, map[string][]byte{"cmap": cmapTable(runes)})
}

// WithBrokenCmap returns a copy of the font whose cmap table is too short to be one.
//
// Flutter: no counterpart
func WithBrokenCmap(tb testing.TB, ttf []byte) []byte {
	tb.Helper()
	return Replace(tb, ttf, map[string][]byte{"cmap": {0, 0, 0}})
}

// Replace returns a copy of the TrueType font with the content of some tables replaced (by tag).
//
// Flutter: no counterpart
func Replace(tb testing.TB, ttf []byte, tables map[string][]byte) []byte {
	tb.Helper()
	if len(ttf) < 12 || binary.BigEndian.Uint32(ttf) != 0x00010000 {
		tb.Fatal("fonttest: not a TrueType font")
	}
	n := int(binary.BigEndian.Uint16(ttf[4:]))
	dir := 12 + 16*n
	if len(ttf) < dir {
		tb.Fatal("fonttest: the table directory is cut")
	}
	var out bytes.Buffer
	out.Write(ttf[:12])
	records := make([]byte, 16*n)
	out.Write(records) // the directory, filled when the tables are placed
	for i := 0; i < n; i++ {
		rec := ttf[12+16*i:]
		tag := string(rec[:4])
		offset, length := int(binary.BigEndian.Uint32(rec[8:])), int(binary.BigEndian.Uint32(rec[12:]))
		if offset+length > len(ttf) {
			tb.Fatalf("fonttest: the table %q is cut", tag)
		}
		data := ttf[offset : offset+length]
		if repl, ok := tables[tag]; ok {
			data = repl
		}
		for out.Len()%4 != 0 {
			out.WriteByte(0)
		}
		copy(records[16*i:], tag)
		binary.BigEndian.PutUint32(records[16*i+4:], checksum(data))
		binary.BigEndian.PutUint32(records[16*i+8:], uint32(out.Len()))
		binary.BigEndian.PutUint32(records[16*i+12:], uint32(len(data)))
		out.Write(data)
	}
	b := out.Bytes()
	copy(b[12:], records)
	return b
}

// checksum is the sum of a table as 32-bit big-endian words, the last one padded with zeros.
func checksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var w [4]byte
		copy(w[:], data[i:])
		sum += binary.BigEndian.Uint32(w[:])
	}
	return sum
}

// cmapTable is a cmap with one Unicode subtable of format 12, one group per run of consecutive runes.
func cmapTable(runes []rune) []byte {
	type group struct{ start, end, gid uint32 }
	var groups []group
	for i, r := range runes {
		gid := uint32(i + 1)
		if n := len(groups); n > 0 && groups[n-1].end+1 == uint32(r) && groups[n-1].gid+(groups[n-1].end-groups[n-1].start)+1 == gid {
			groups[n-1].end = uint32(r)
			continue
		}
		groups = append(groups, group{uint32(r), uint32(r), gid})
	}
	var b bytes.Buffer
	put16 := func(v uint16) { _ = binary.Write(&b, binary.BigEndian, v) } // a Buffer does not fail
	put32 := func(v uint32) { _ = binary.Write(&b, binary.BigEndian, v) }
	put16(0)  // version
	put16(1)  // one encoding record
	put16(3)  // platform: Windows
	put16(10) // encoding: Unicode full repertoire
	put32(12) // offset of the subtable
	put16(12) // format
	put16(0)  // reserved
	put32(uint32(16 + 12*len(groups)))
	put32(0) // language
	put32(uint32(len(groups)))
	for _, g := range groups {
		put32(g.start)
		put32(g.end)
		put32(g.gid)
	}
	return b.Bytes()
}
