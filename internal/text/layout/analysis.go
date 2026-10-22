package layout

import (
	"unicode/utf8"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/foundation"
)

// What the engine finds out about a text before it shapes it does not depend on the fonts or the
// styles, only on the text, the direction of the paragraph and its language: the bidi levels, the
// pieces of one level and one script, the line break opportunities, the grapheme clusters and (when
// a query asks) the words. It costs about as much as shaping, so the engine keeps it by text: a line
// that is built again with other styles, or one whose spans did not change, finds it again.

const (
	// maxAnalyses and maxAnalysisBytes bound the cache of analyses.
	maxAnalyses       = 2048
	maxAnalysisBytes  = 4 << 20
	analysisOverhead  = 256
	analysisPerRune   = 8 + 1 + 8 + 8 // a segment's share, a level, a grapheme index and a rune
	analysisPerPiece  = 48
	analysisPerSegmnt = 40
)

// piece is a stretch of the text of one embedding level and one script: what shaping.Segmenter makes of
// a run of one level.
type piece struct {
	start, end int
	dir        di.Direction
	script     language.Script
	lang       language.Language
}

// analysis is what the engine knows of a text, shared by the paragraphs of that text. Nothing in
// it changes after it is made, except the words, which a query makes the first time.
type analysis struct {
	levels    []uint8
	pieces    []piece
	segments  []segment
	graphemes []int

	words     []wordSeg
	wordsDone bool
}

func weighAnalysis(key string, a *analysis) int {
	return analysisOverhead + len(key) + len(a.levels)*analysisPerRune + len(a.pieces)*analysisPerPiece + len(a.segments)*analysisPerSegmnt
}

// analysisKey builds the key of a text in a buffer: its base direction, the language and the text.
func (e *Engine) analysisKey(text string, baseRTL bool, lang language.Language) []byte {
	if cap(e.analysisBuf) > 1<<16 {
		e.analysisBuf = nil // a long text does not keep its room
	}
	k := e.analysisBuf[:0]
	if baseRTL {
		k = append(k, 1)
	} else {
		k = append(k, 0)
	}
	k = append(k, byte(len(lang)))
	k = append(k, lang...)
	k = append(k, text...)
	e.analysisBuf = k
	return k
}

// analyze returns the analysis of a text: from the cache, or made now.
func (e *Engine) analyze(text string, runes, shapeRunes []rune, baseRTL bool, lang language.Language) *analysis {
	if e.noAnalysisCache {
		return e.makeAnalysis(runes, shapeRunes, baseRTL, lang)
	}
	key := e.analysisKey(text, baseRTL, lang)
	if a, ok := lruGetBytes(e.analyses, key); ok {
		return a
	}
	a := e.makeAnalysis(runes, shapeRunes, baseRTL, lang)
	e.analyses.put(string(key), a)
	return a
}

// makeAnalysis finds the levels, the pieces, the break opportunities and the grapheme clusters.
func (e *Engine) makeAnalysis(runes, shapeRunes []rune, baseRTL bool, lang language.Language) *analysis {
	a := &analysis{}
	if !baseRTL && !e.noFastAnalysis && isASCII(runes) {
		// Left-to-right text with no character of another direction is all at level 0, and the bidi
		// run of go-text is the whole of it: only the split by script is left to do.
		a.levels = make([]uint8, len(runes))
		a.pieces = asciiPieces(shapeRunes, lang)
	} else {
		a.levels = resolveLevels(runes, baseRTL)
		a.pieces = e.pieces(shapeRunes, a.levels, lang)
	}
	a.segments = lineSegments(runes)
	a.graphemes = graphemeStarts(runes, e.noFastAnalysis)
	return a
}

// isASCII says whether every rune is below 0x80.
func isASCII(runes []rune) bool {
	for _, r := range runes {
		if r >= 0x80 {
			return false
		}
	}
	return true
}

// pieces splits the text by level, then by script (shaping.Segmenter.Split).
func (e *Engine) pieces(shapeRunes []rune, levels []uint8, lang language.Language) []piece {
	n := len(levels)
	var out []piece
	for lo := 0; lo < n; {
		hi := lo
		for hi < n && levels[hi] == levels[lo] {
			hi++
		}
		dir := di.DirectionLTR
		if levels[lo]%2 == 1 {
			dir = di.DirectionRTL
		}
		whole := shaping.Input{Text: shapeRunes, RunStart: lo, RunEnd: hi, Direction: dir, Size: shapeSize * 64, Language: lang}
		for _, in := range e.segmenter.Split(whole, sentinelFontmap{}) {
			// The segmenter resolves bidi too, with the first strong character; the level
			// run is one direction already.
			out = append(out, piece{start: in.RunStart, end: in.RunEnd, dir: dir, script: in.Script, lang: in.Language})
		}
		lo = hi
	}
	return out
}

// asciiScripts is the script of each ASCII character, as go-text's tables have it: the letters are
// Latin, everything else Common.
var asciiScripts = func() (t [128]language.Script) {
	for r := range t {
		t[r] = language.LookupScript(rune(r))
	}
	return t
}()

// asciiDelims is the index of an ASCII character in go-text's list of paired delimiters (an even
// index opens a pair and the next one closes it), or -1.
var asciiDelims = func() (t [128]int8) {
	for i := range t {
		t[i] = -1
	}
	for i, r := range []rune{'(', ')', '<', '>', '[', ']', '{', '}'} {
		t[r] = int8(i)
	}
	return t
}()

// asciiPieces is what shaping.Segmenter.Split makes of left-to-right text that is all ASCII (but for the
// zero width spaces that stand for line breaks): one piece for each script. It is splitByScript and enforceLanguages of go-text (shaping/input.go)
// for text that bidi leaves whole, with the pairs of delimiters that ASCII has; a test compares it
// with Split on random text.
func asciiPieces(text []rune, lang language.Language) []piece {
	n := len(text)
	if n == 0 {
		return nil
	}
	type delim struct {
		index  int
		script language.Script
	}
	var stack []delim
	var out []piece
	cur := piece{start: 0, end: n, dir: di.DirectionLTR, script: language.Common, lang: lang}
	for i, r := range text {
		// The text that the shaper sees has a zero width space for each line break.
		rScript, delimIndex := language.LookupScript(r), -1
		if r < 128 {
			rScript = asciiScripts[r]
			if !rScript.Strong() {
				delimIndex = int(asciiDelims[r])
			}
		}
		if delimIndex >= 0 {
			if delimIndex%2 == 0 {
				stack = append(stack, delim{delimIndex, cur.script})
			} else {
				j := len(stack) - 1
				for ; j >= 0; j-- {
					if stack[j].index == delimIndex-1 {
						rScript = stack[j].script
						break
					}
				}
				if j == -1 {
					j = 0
				}
				stack = stack[:j]
			}
		}
		switch {
		case !rScript.Strong() || rScript == cur.script:
		case cur.script == language.Common:
			for k := range stack {
				stack[k].script = rScript
			}
			cur.script = rScript
		default:
			if i != 0 {
				cur.end = i
				out = append(out, cur)
			}
			cur.start = i
			cur.script = rScript
		}
	}
	cur.end = n
	out = append(out, cur)

	// enforceLanguages: a language that does not fit the script becomes the script's.
	initial := lang
	if initial == "" {
		initial = "en"
	}
	if id, ok := language.NewLangID(initial); ok {
		for i := range out {
			resolved := id
			if !id.UseScript(out[i].script) {
				if l := language.ScriptToLang[out[i].script]; l != 0 {
					resolved = l
				}
			}
			out[i].lang = resolved.Language()
		}
	}
	return out
}

// lineSegments finds the line break opportunities of the text (UAX #14) and cuts it into the stretches
// between them.
func lineSegments(runes []rune) []segment {
	var segs []segment
	var sg segmenter.Segmenter
	sg.Init(runes)
	it := sg.LineIterator()
	for it.Next() {
		l := it.Line()
		s := segment{start: l.Offset, end: l.Offset + len(l.Text)}
		// UAX #14 also ends the text with a mandatory break (LB3); only a line break
		// character ends a line.
		s.mandatory = l.IsMandatoryBreak && s.end > s.start && isLineBreakRune(runes[s.end-1])
		s.visEnd = s.end
		for s.visEnd > s.start && isHangingSpace(runes[s.visEnd-1]) {
			s.visEnd--
		}
		segs = append(segs, s)
	}
	return segs
}

// graphemeStarts are the indices of the runes where a grapheme cluster starts, and the number of runes at
// the end. The clusters come from foundation, which fixes the emoji sequences that the segmenter splits.
// Text of runes below U+0300 has no combining mark, joiner or emoji sequence, so it breaks between every
// rune but a carriage return and the line feed after it (UAX #29, rule GB3); the general path is for
// the rest, and for the tests that compare the two (general).
func graphemeStarts(runes []rune, general bool) []int {
	out := make([]int, 0, len(runes)+1)
	if len(runes) == 0 {
		return append(out, 0)
	}
	if !general && func() bool {
		for _, r := range runes {
			if r >= 0x300 {
				return false
			}
		}
		return true
	}() {
		for i := range runes {
			if i > 0 && runes[i-1] == '\r' && runes[i] == '\n' {
				continue
			}
			out = append(out, i)
		}
		return append(out, len(runes))
	}
	runeAt, byteAt := 0, 0
	for _, b := range foundation.GraphemeBoundaries(string(runes)) {
		for byteAt < b {
			byteAt += utf8.RuneLen(runes[runeAt])
			runeAt++
		}
		out = append(out, runeAt)
	}
	return out
}

// wordBreaks segments the text into words (UAX #29), on the first call.
func (a *analysis) wordBreaks(runes []rune) []wordSeg {
	if a.wordsDone {
		return a.words
	}
	a.wordsDone = true
	a.words = []wordSeg{}
	var sg segmenter.Segmenter
	sg.Init(runes)
	it := sg.WordIterator()
	for it.Next() {
		w := it.Word()
		seg := wordSeg{start: w.Offset, end: w.Offset + len(w.Text)}
		for _, r := range w.Text {
			if isWordRune(r) {
				seg.isWord = true
				break
			}
		}
		a.words = append(a.words, seg)
	}
	return a.words
}
