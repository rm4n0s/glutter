package layout

import (
	"encoding/binary"
	"unicode/utf8"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
)

// The cache of shaped runs (SPEC §10, "Shaped run cache"): shaping is the main cost of
// building a paragraph, and the lines of an editor are built again with spans that mostly
// did not change.

const (
	// maxShapedRuns and maxShapedBytes bound the cache: it keeps at most this many runs, and
	// this many bytes of keys and glyphs, and drops the run that was used least recently when
	// either is passed.
	maxShapedRuns  = 4096
	maxShapedBytes = 8 << 20

	// shapedRunOverhead is what an entry weighs besides its key and its glyphs: the node, the
	// map slot and the slice header.
	shapedRunOverhead = 128
)

// rawGlyph is a glyph as the shaper made it, at shapeSize: the advances and offsets are in
// 1/64 of a pixel, and the cluster is relative to the start of the run, so that the same
// glyphs serve a run at another place of another text.
type rawGlyph struct {
	id                        font.GID
	cluster, runes            int32
	advance, xOffset, yOffset int32
}

// rawGlyphSize is the size of a rawGlyph in bytes.
const rawGlyphSize = 24

// shapedRun is what the cache keeps of a run: the glyphs, in the order the shaper gave them.
type shapedRun struct {
	glyphs []rawGlyph
}

func weighShapedRun(key string, v *shapedRun) int {
	return shapedRunOverhead + len(key) + len(v.glyphs)*rawGlyphSize
}

// shapeCache is the engine's cache of shaped runs, with the counters the tests read.
type shapeCache struct {
	runs *lruCache[string, *shapedRun]
	// faces has what the engine knows of each face: its number (so that a key does not hold a
	// pointer) and the extents of its font.
	faces map[*font.Face]*faceInfo
	key   []byte

	// shaped counts the runs that went to the shaper, and hits the runs that did not.
	shaped, hits int
	// off turns the cache off: every run is shaped. The tests compare with it.
	off bool
}

func newShapeCache() *shapeCache {
	return &shapeCache{runs: newWeightedLRU(maxShapedRuns, maxShapedBytes, weighShapedRun), faces: map[*font.Face]*faceInfo{}}
}

// clear forgets every run and every face: the fonts changed.
func (c *shapeCache) clear() {
	c.runs.clear()
	clear(c.faces)
}

// faceInfo is what the engine keeps for a face.
type faceInfo struct {
	id uint32
	// hasExtents is false when the font has no horizontal extents, and then the others are not set.
	hasExtents bool
	// strikes are the bitmap strikes of the font (CBDT, sbix, EBDT); its glyphs may be bitmaps (bitmap.go).
	strikes                      []font.BitmapSize
	upem                         float64
	ascender, descender, lineGap float32
}

// info returns what the engine knows of the face, finding it the first time.
func (c *shapeCache) info(f *font.Face) *faceInfo {
	if fi, ok := c.faces[f]; ok {
		return fi
	}
	fi := &faceInfo{id: uint32(len(c.faces)) + 1, strikes: f.BitmapSizes()}
	if ext, ok := f.FontHExtents(); ok && (ext.Ascender != 0 || ext.Descender != 0) {
		fi.hasExtents, fi.upem = true, float64(f.Upem())
		fi.ascender, fi.descender, fi.lineGap = ext.Ascender, ext.Descender, ext.LineGap
	}
	c.faces[f] = fi
	return fi
}

// needsContext says whether HarfBuzz reads the text around a run of this script: the scripts
// that join letters look at the letters before and after the run. The scripts that are listed do
// not; for any other the key holds the context.
func needsContext(s language.Script) bool {
	switch s {
	case language.Latin, language.Common, language.Cyrillic, language.Greek, language.Han, language.Hiragana,
		language.Katakana, language.Hangul, language.Hebrew, language.Thai, language.Armenian, language.Georgian:
		return false
	}
	return true
}

// contextLength is how many runes of context HarfBuzz reads on each side (buffer.go of
// go-text's harfbuzz).
const contextLength = 5

// makeKey builds the key of the input in the cache's buffer. Two inputs with the same key shape
// to the same glyphs: the face, the direction, the script, the language, the features and the text
// of the run decide the glyphs, and HarfBuzz also looks at the text before the run (whether there
// is any) and, for the scripts that join, at five runes on each side.
func (c *shapeCache) makeKey(in *shaping.Input) []byte {
	k := c.key[:0]
	k = binary.LittleEndian.AppendUint32(k, c.info(in.Face).id)
	k = append(k, byte(in.Direction))
	k = binary.LittleEndian.AppendUint32(k, uint32(in.Script))
	k = append(k, byte(len(in.Language)))
	k = append(k, in.Language...)
	k = append(k, byte(len(in.FontFeatures)))
	for _, f := range in.FontFeatures {
		k = binary.LittleEndian.AppendUint32(k, uint32(f.Tag))
		k = binary.LittleEndian.AppendUint32(k, f.Value)
	}
	switch {
	case needsContext(in.Script):
		k = append(k, 2)
		pre := in.Text[max(in.RunStart-contextLength, 0):in.RunStart]
		k = append(k, byte(len(pre)))
		for _, r := range pre {
			k = utf8.AppendRune(k, r)
		}
		post := in.Text[in.RunEnd:min(in.RunEnd+contextLength, len(in.Text))]
		k = append(k, byte(len(post)))
		for _, r := range post {
			k = utf8.AppendRune(k, r)
		}
	case in.RunStart > 0:
		k = append(k, 1)
	default:
		k = append(k, 0)
	}
	for _, r := range in.Text[in.RunStart:in.RunEnd] {
		k = utf8.AppendRune(k, r)
	}
	c.key = k
	return k
}

// shape returns the shaped run of the input: from the cache when it has it, else from the
// shaper, which it then keeps.
func (c *shapeCache) shape(shaper *shaping.HarfbuzzShaper, in shaping.Input) *shapedRun {
	if c.off {
		c.shaped++
		return shapeRaw(shaper, in)
	}
	key := c.makeKey(&in)
	if v, ok := lruGetBytes(c.runs, key); ok {
		c.hits++
		return v
	}
	c.shaped++
	v := shapeRaw(shaper, in)
	c.runs.put(string(key), v)
	return v
}

// shapeRaw shapes the input with the shaper.
func shapeRaw(shaper *shaping.HarfbuzzShaper, in shaping.Input) *shapedRun {
	out := shaper.Shape(in)
	v := &shapedRun{glyphs: make([]rawGlyph, len(out.Glyphs))}
	for i, g := range out.Glyphs {
		v.glyphs[i] = rawGlyph{id: g.GlyphID, cluster: int32(g.TextIndex() - in.RunStart), runes: int32(g.RunesCount()),
			advance: int32(g.Advance), xOffset: int32(g.XOffset), yOffset: int32(g.YOffset)}
	}
	return v
}
