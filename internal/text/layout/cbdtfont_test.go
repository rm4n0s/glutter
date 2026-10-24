package layout

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"sort"
)

// This file builds a tiny sfnt font with CBLC and CBDT tables, the color bitmap font that
// Android 8 to 12 uses for emoji (Noto Color Emoji), so that the tests need no font file and no
// license. The font has four glyphs and no outlines (Noto Color Emoji's outlines are empty
// too):
//
//	gid 0  .notdef, nothing
//	gid 1  'A'        a blue square, format 17 (small metrics, PNG)
//	gid 2  U+1F600    a red square, format 17
//	gid 3  U+1F601    a green square, format 18 (big metrics, PNG)
//
// Every strike has its own pixels per em and its own marker square, so that a test can tell
// which strike was drawn.

// testUpem is the units per em of the font; the advance of every glyph is one em.
const testUpem = 2048

// Colors of the glyphs and the markers of the strikes.
var (
	testBlue  = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	testRed   = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	testGreen = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
)

// testStrike describes a strike of the test font: the glyphs are squares of size pixels.
type testStrike struct {
	ppem               uint8
	size               int
	bearingX, bearingY int8        // of the small metrics and of the horizontal big metrics
	marker             color.NRGBA // the color of the square in the top left corner of every glyph
	// png, when it has an entry for a glyph, replaces the image of the glyph (a broken one).
	png map[int][]byte
}

// testStrikes are the two strikes of the default test font: 109 is Noto Color Emoji's, 32 a
// small one. The bearings scale with the ppem, so that every strike places the glyph the same.
var testStrikes = []testStrike{
	{ppem: 109, size: 109, bearingX: 7, bearingY: 95, marker: color.NRGBA{A: 255}},                       // black marker
	{ppem: 32, size: 32, bearingX: 2, bearingY: 28, marker: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}, // white marker
}

// testGlyphPNG draws the image of a glyph: a square of the color with a half transparent border
// one pixel wide and a marker square of a quarter of the size, an eighth of the size from the
// top left corner (so that a flip or a shift shows).
func testGlyphPNG(size int, base, marker color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := base
			if x == 0 || y == 0 || x == size-1 || y == size-1 {
				c.A = 128
			}
			if x >= size/8 && x < size/8+size/4 && y >= size/8 && y < size/8+size/4 {
				c = marker
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding to memory does not fail
	}
	return buf.Bytes()
}

// be appends big-endian numbers.
type be []byte

func (b *be) u8(v int)     { *b = append(*b, byte(v)) }
func (b *be) u16(v int)    { *b = binary.BigEndian.AppendUint16(*b, uint16(v)) }
func (b *be) u32(v int)    { *b = binary.BigEndian.AppendUint32(*b, uint32(v)) }
func (b *be) raw(v []byte) { *b = append(*b, v...) }

// testGlyphs lists the glyphs that have bitmaps: gid, color and the format of the image.
var testGlyphs = []struct {
	gid    int
	color  color.NRGBA
	format int
}{{1, testBlue, 17}, {2, testRed, 17}, {3, testGreen, 18}}

// buildCBDTFont makes the font with the strikes. The tables are the ones go-text needs to load a
// font (cmap, head, maxp), the ones that give the advances and the line (hhea, hmtx) and the
// bitmap tables.
func buildCBDTFont(strikes []testStrike) []byte {
	numGlyphs := 4

	var head be
	head.u32(0x00010000)
	head.u32(0x00010000)
	head.u32(0)
	head.u32(0x5F0F3CF5)
	head.u16(0)
	head.u16(testUpem)
	head.raw(make([]byte, 16)) // created, modified
	head.raw(make([]byte, 8))  // the bounding box
	head.u16(0)                // macStyle
	head.u16(8)                // lowestRecPPEM
	head.u16(2)                // fontDirectionHint
	head.u16(0)                // indexToLocFormat
	head.u16(0)                // glyphDataFormat

	var hhea be
	hhea.u32(0x00010000)
	hhea.u16(1900)          // ascender
	hhea.u16(-500 & 0xffff) // descender
	hhea.u16(0)             // lineGap
	hhea.u16(testUpem)
	hhea.raw(make([]byte, 6*2)) // minLSB, minRSB, xMaxExtent, caretSlopeRise, caretSlopeRun, caretOffset
	hhea.raw(make([]byte, 4*2)) // reserved
	hhea.u16(0)                 // metricDataFormat
	hhea.u16(numGlyphs)         // numberOfHMetrics

	var maxp be
	maxp.u32(0x00005000)
	maxp.u16(numGlyphs)

	var hmtx be
	for range numGlyphs {
		hmtx.u16(testUpem)
		hmtx.u16(0)
	}

	// cmap: one format 12 subtable for Windows, Unicode full repertoire.
	var cmap be
	cmap.u16(0)
	cmap.u16(1)
	cmap.u16(3)
	cmap.u16(10)
	cmap.u32(12)
	cmap.u16(12)
	cmap.u16(0)
	cmap.u32(16 + 2*12)
	cmap.u32(0)
	cmap.u32(2)
	for _, g := range [][3]int{{'A', 'A', 1}, {0x1F600, 0x1F601, 2}} {
		cmap.u32(g[0])
		cmap.u32(g[1])
		cmap.u32(g[2])
	}

	var post be
	post.u32(0x00030000)
	post.raw(make([]byte, 28))

	cblc, cbdt := buildBitmapTables(strikes)

	return buildSfnt(map[string][]byte{
		"head": head, "hhea": hhea, "maxp": maxp, "hmtx": hmtx, "cmap": cmap, "post": post,
		"CBLC": cblc, "CBDT": cbdt,
	})
}

// buildBitmapTables makes CBLC and CBDT. Each strike has two index subtables: glyphs 1 to 2 in
// the image format 17 with the index format 1 (32 bit offsets), and glyph 3 in the image format
// 18 with the index format 3 (16 bit offsets).
func buildBitmapTables(strikes []testStrike) (cblc, cbdt []byte) {
	var data be
	data.u16(3) // CBDT version 3.0
	data.u16(0)

	// put appends the data of a glyph of a strike and returns its offset in CBDT.
	put := func(s testStrike, gid int, c color.NRGBA, format int) int {
		off := len(data)
		pngData := testGlyphPNG(s.size, c, s.marker)
		if p, ok := s.png[gid]; ok {
			pngData = p
		}
		data.u8(s.size) // height
		data.u8(s.size) // width
		data.u8(int(s.bearingX))
		data.u8(int(s.bearingY))
		data.u8(s.size) // advance (horizontal)
		if format == 18 {
			data.u8(0) // vertBearingX
			data.u8(0) // vertBearingY
			data.u8(s.size)
		}
		data.u32(len(pngData))
		data.raw(pngData)
		return off
	}

	// The subtables are laid out after the header and the strike records.
	var records, subtables be
	base := 8 + 48*len(strikes)
	for _, s := range strikes {
		off := make([]int, 4)
		for _, g := range testGlyphs {
			off[g.gid] = put(s, g.gid, g.color, g.format)
		}
		end := len(data)

		arrayAt := base + len(subtables)
		var table be
		// The array of the two subtables: 16 bytes, then the subtable 1 (8 + 4 * 3 bytes: the header and
		// three offsets), then the subtable 2 (8 + 2 * 3: the header and three 16 bit offsets, the last
		// of which pads nothing).
		table.u16(1)
		table.u16(2)
		table.u32(16)
		table.u16(3)
		table.u16(3)
		table.u32(16 + 8 + 4*3)
		// Subtable 1: glyphs 1 and 2, offsets from the image of glyph 1.
		table.u16(1)
		table.u16(17)
		table.u32(off[1])
		table.u32(0)
		table.u32(off[2] - off[1])
		table.u32(off[3] - off[1])
		// Subtable 2: glyph 3, offsets from its image.
		table.u16(3)
		table.u16(18)
		table.u32(off[3])
		table.u16(0)
		table.u16(end - off[3])
		table.u16(0) // padding to four bytes
		subtables.raw(table)

		// The record of the strike.
		records.u32(arrayAt)
		records.u32(len(table))
		records.u32(2)
		records.u32(0)
		for range 2 { // hori and vert line metrics
			records.u8(int(s.bearingY))
			records.u8(-5 & 0xff)
			records.raw(make([]byte, 10))
		}
		records.u16(1)
		records.u16(3)
		records.u8(int(s.ppem))
		records.u8(int(s.ppem))
		records.u8(32)
		records.u8(1)
	}

	var head be
	head.u16(3)
	head.u16(0)
	head.u32(len(strikes))
	head.raw(records)
	head.raw(subtables)
	return head, data
}

// buildSfnt joins tables into an sfnt file: the directory sorted by tag, each table padded to
// four bytes. The checksums are zero, which the loader does not check.
func buildSfnt(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	var out be
	out.u32(0x00010000)
	out.u16(len(tags))
	out.u16(0) // searchRange, entrySelector and rangeShift are not used by the loader
	out.u16(0)
	out.u16(0)
	offset := 12 + 16*len(tags)
	for _, tag := range tags {
		out.raw([]byte(tag))
		out.u32(0)
		out.u32(offset)
		out.u32(len(tables[tag]))
		offset += (len(tables[tag]) + 3) &^ 3
	}
	for _, tag := range tags {
		out.raw(tables[tag])
		out.raw(make([]byte, -len(tables[tag])&3))
	}
	return out
}
