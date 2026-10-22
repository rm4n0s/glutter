package layout

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
)

// The tests of the caches of the text engine (step 22.10): the shaped runs, the analyses of
// texts and the shortcuts that go with them give what the engine gave before it had them. The
// reference here is the pipeline as it was: a segmenter of its own, shaping.SplitByFace and one call
// of the shaper per run.

// refRun is what a run holds that shaping decides.
type refRun struct {
	Start, End                 int
	RTL                        bool
	Level, Style               int
	Face                       *font.Face
	Size                       float64
	Glyphs                     []glyph
	Ascent, Descent, Gap       float64
	AscentC, DescentC, AscentU float64
}

// refFontMetrics is fontMetrics as it was before the engine kept the extents of a face.
func refFontMetrics(face *font.Face, size float64) (ascent, descent, gap float64) {
	if face == nil {
		return 0.93 * size, 0.24 * size, 0
	}
	ext, ok := face.FontHExtents()
	if !ok || (ext.Ascender == 0 && ext.Descender == 0) {
		return 0.93 * size, 0.24 * size, 0
	}
	upem := float64(face.Upem())
	return float64(ext.Ascender) / upem * size, -float64(ext.Descender) / upem * size, float64(ext.LineGap) / upem * size
}

// refRuns shapes the text of the paragraph (it must have no placeholder) the way the engine did
// before it kept anything: no cache, a new segmenter, the general split by face.
func refRuns(p *paragraph) []refRun {
	n := len(p.runes)
	if n == 0 {
		return nil
	}
	baseRTL := p.style.Direction() == ui.TextDirectionRTL
	levels := resolveLevels(p.runes, baseRTL)
	lang := language.NewLanguage(p.baseLocale.LanguageTag())
	var seg shaping.Segmenter
	var out []refRun
	for lo := 0; lo < n; {
		hi := lo
		for hi < n && levels[hi] == levels[lo] {
			hi++
		}
		dir := di.DirectionLTR
		if levels[lo]%2 == 1 {
			dir = di.DirectionRTL
		}
		whole := shaping.Input{Text: p.shapeRunes, RunStart: lo, RunEnd: hi, Direction: dir, Size: shapeSize * 64, Language: lang}
		for _, piece := range seg.Split(whole, sentinelFontmap{}) {
			piece.Direction = dir
			start := piece.RunStart
			for start < piece.RunEnd {
				si := p.styleAt[start]
				end := start
				for end < piece.RunEnd && p.styleAt[end] == si {
					end++
				}
				sub := piece
				sub.RunStart, sub.RunEnd = start, end
				sub.Face = nil
				st := p.styles[si]
				if st.Locale.LanguageCode != "" {
					sub.Language = language.NewLanguage(st.Locale.LanguageTag())
				}
				fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
				for _, in := range shaping.SplitByFace(sub, fm) {
					out = append(out, refShapeRun(p, levels, in, si, fm))
				}
				start = end
			}
		}
		lo = hi
	}
	return out
}

func refShapeRun(p *paragraph, levels []uint8, in shaping.Input, si int, fm *styleFonts) refRun {
	st := p.styles[si]
	r := refRun{Start: in.RunStart, End: in.RunEnd, RTL: in.Direction.Progression() == di.TowardTopLeft, Style: si, Face: in.Face, Size: st.FontSize}
	r.Level = int(levels[in.RunStart])
	r.Ascent, r.Descent, r.Gap = refFontMetrics(in.Face, r.Size)
	even := st.LeadingDistribution == ui.TextLeadingDistributionEven
	r.AscentC, r.DescentC = correctedMetrics(r.Ascent, r.Descent, r.Gap, r.Size, st.Height, st.HasHeight, even)
	r.AscentU = r.Ascent + r.Gap/2
	tmp := &run{start: r.Start, end: r.End, rtl: r.RTL}
	if in.Face == nil || st.FontSize <= 0 {
		r.Glyphs = p.missingGlyphs(tmp)
		return r
	}
	in.Size = shapeSize * 64
	for _, f := range st.FontFeatures {
		in.FontFeatures = append(in.FontFeatures, shaping.FontFeature{Tag: tagOf(f.Feature), Value: uint32(f.Value)})
	}
	out := fm.shaper.Shape(in)
	scale := r.Size / shapeSize / 64
	tmp.glyphs = make([]glyph, len(out.Glyphs))
	for i, g := range out.Glyphs {
		tmp.glyphs[i] = glyph{id: g.GlyphID, cluster: g.TextIndex(), runes: g.RunesCount(),
			advance: float64(g.Advance) * scale, xOffset: float64(g.XOffset) * scale, yOffset: float64(g.YOffset) * scale}
	}
	applySpacing(tmp, in.Text, st.LetterSpacing, st.WordSpacing)
	r.Glyphs = tmp.glyphs
	return r
}

// runsOf is the runs of a paragraph, as refRun.
func runsOf(p *paragraph) []refRun {
	var out []refRun
	for _, r := range p.runs {
		out = append(out, refRun{Start: r.start, End: r.end, RTL: r.rtl, Level: r.level, Style: r.style, Face: r.face, Size: r.size, Glyphs: r.glyphs,
			Ascent: r.ascent, Descent: r.descent, Gap: r.gap, AscentC: r.ascentC, DescentC: r.descentC, AscentU: r.ascentU})
	}
	return out
}

// fuzzStyles are the styles that the fuzzed texts are shaped in.
var fuzzStyles = []ui.TextStyleOpts{
	{},
	{FontFamily: "Go Mono", FontSize: ui.Ptr(14.0)},
	{FontFamily: "Go Mono", FontSize: ui.Ptr(20.0), FontWeight: ui.FontWeightW700},
	{FontFamily: "Go Mono", FontSize: ui.Ptr(11.0), FontStyle: ui.FontStyleItalic, LetterSpacing: ui.Ptr(1.0), WordSpacing: ui.Ptr(3.0)},
	{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0)},
	{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), FontFeatures: []ui.FontFeature{ui.NewFontFeature("liga", 0)}},
	{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(13.0), FontWeight: ui.FontWeightW700, Height: ui.Ptr(1.4)},
	{FontFamily: "Go Mono", FontFamilyFallback: []string{fonts.RobotoFamily}, FontSize: ui.Ptr(15.0), Locale: &ui.Locale{LanguageCode: "ja"}},
}

// fuzzPieces are what the fuzzed texts are made of: Latin, digits and brackets, accents and
// combining marks, ligatures, other scripts (some of them right to left), emoji sequences, tabs and
// line breaks.
var fuzzPieces = []string{
	"a", "b", "c", "func", "if", "x", " ", " ", "  ", "(", ")", "{", "}", "[", "]", "\"", "'", "<", ">", "=", "+", "-", "/", "_", "0", "12", "3.5",
	"é", "é", "ñ", "ﬁ", "fi", "ffl", "\t", "\n", "\u200d", "Привет", "αβγ", "שלום", "مرحبا", "بببب", "日本語", "한국어", "😀", "👨‍👩‍👧", "🇬🇷",
}

type fuzzSpan struct {
	text  string
	style int
}

func randomSpans(rng *rand.Rand) []fuzzSpan {
	spans := make([]fuzzSpan, 1+rng.Intn(6))
	for i := range spans {
		var sb strings.Builder
		for n := rng.Intn(10); n >= 0; n-- {
			sb.WriteString(fuzzPieces[rng.Intn(len(fuzzPieces))])
		}
		spans[i] = fuzzSpan{sb.String(), rng.Intn(len(fuzzStyles))}
	}
	return spans
}

func buildSpans(e *Engine, spans []fuzzSpan, dir ui.TextDirection) *paragraph {
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{FontFamily: "Go Mono", FontSize: ui.Ptr(14.0), TextDirection: dir}))
	for _, s := range spans {
		st := fuzzStyles[s.style]
		b.PushStyle(ui.NewTextStyle(st))
		b.AddText(s.text)
		b.Pop()
	}
	return b.Build().Layouter().(*paragraph)
}

func describeSpans(spans []fuzzSpan) string {
	var parts []string
	for _, s := range spans {
		parts = append(parts, fmt.Sprintf("%d:%q", s.style, s.text))
	}
	return strings.Join(parts, " ")
}

// TestCachesGiveWhatFreshShapingGives builds random paragraphs again and again, and with spans
// changed, and compares every run of the cached engine with the reference, glyph by glyph.
func TestCachesGiveWhatFreshShapingGives(t *testing.T) {
	e := codeEngine(t)
	rng := rand.New(rand.NewSource(11))
	for iter := 0; iter < 400; iter++ {
		spans := randomSpans(rng)
		dir := ui.TextDirectionLTR
		if rng.Intn(4) == 0 {
			dir = ui.TextDirectionRTL
		}
		// Built three times: cold, then from the caches, then with a span changed or moved, so
		// that some runs are in the cache and some are not.
		variants := [][]fuzzSpan{spans, spans}
		changed := append([]fuzzSpan(nil), spans...)
		i := rng.Intn(len(changed))
		switch rng.Intn(3) {
		case 0:
			changed[i].text += fuzzPieces[rng.Intn(len(fuzzPieces))]
		case 1:
			changed[i].style = rng.Intn(len(fuzzStyles))
		default:
			changed = append(changed[:i], changed[i+1:]...)
			if len(changed) == 0 {
				changed = spans
			}
		}
		variants = append(variants, changed)
		for v, sp := range variants {
			p := buildSpans(e, sp, dir)
			want := refRuns(p)
			got := runsOf(p)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("iteration %d, build %d, %v: the runs differ from fresh shaping\nspans: %s\ngot:  %+v\nwant: %+v", iter, v, dir, describeSpans(sp), got, want)
			}
			e.noFastAnalysis = true // the general path, to compare with
			fresh := e.makeAnalysis(p.runes, p.shapeRunes, dir == ui.TextDirectionRTL, language.NewLanguage(p.baseLocale.LanguageTag()))
			e.noFastAnalysis = false
			if !reflect.DeepEqual(p.an.levels, fresh.levels) || !reflect.DeepEqual(p.an.pieces, fresh.pieces) ||
				!reflect.DeepEqual(p.an.segments, fresh.segments) || !reflect.DeepEqual(p.an.graphemes, fresh.graphemes) {
				t.Fatalf("iteration %d, build %d: the analysis differs from a fresh one\nspans: %s", iter, v, describeSpans(sp))
			}
			if !reflect.DeepEqual(p.an.wordBreaks(p.runes), fresh.wordBreaks(p.runes)) {
				t.Fatalf("iteration %d, build %d: the words differ\nspans: %s", iter, v, describeSpans(sp))
			}
		}
	}
	if e.shapes.hits == 0 {
		t.Error("the fuzz never hit the cache")
	}
}

// TestCachesGiveTheSameLayout compares whole paragraphs, line by line, of an engine whose caches are
// off with one whose caches are full.
func TestCachesGiveTheSameLayout(t *testing.T) {
	cached, fresh := codeEngine(t), codeEngine(t)
	fresh.shapes.off, fresh.noAnalysisCache = true, true
	rng := rand.New(rand.NewSource(5))
	for iter := 0; iter < 150; iter++ {
		spans := randomSpans(rng)
		width := float64(40 + rng.Intn(300))
		buildSpans(cached, spans, ui.TextDirectionLTR) // warm
		a, b := buildSpans(cached, spans, ui.TextDirectionLTR), buildSpans(fresh, spans, ui.TextDirectionLTR)
		a.Layout(ui.NewParagraphConstraints(width))
		b.Layout(ui.NewParagraphConstraints(width))
		if a.Height() != b.Height() || a.LongestLine() != b.LongestLine() || a.MaxIntrinsicWidth() != b.MaxIntrinsicWidth() || a.MinIntrinsicWidth() != b.MinIntrinsicWidth() {
			t.Fatalf("sizes differ: %v x %v and %v x %v\nspans: %s", a.LongestLine(), a.Height(), b.LongestLine(), b.Height(), describeSpans(spans))
		}
		if !reflect.DeepEqual(a.ComputeLineMetrics(), b.ComputeLineMetrics()) {
			t.Fatalf("line metrics differ\nspans: %s", describeSpans(spans))
		}
		n := len(a.content.Text)
		if !reflect.DeepEqual(a.GetBoxesForRange(0, n, ui.BoxHeightStyleTight, ui.BoxWidthStyleTight), b.GetBoxesForRange(0, n, ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)) {
			t.Fatalf("boxes differ\nspans: %s", describeSpans(spans))
		}
		var da, db []ui.ParagraphDraw
		a.PaintParagraph(func(d ui.ParagraphDraw) { da = append(da, d) })
		b.PaintParagraph(func(d ui.ParagraphDraw) { db = append(db, d) })
		if len(da) != len(db) {
			t.Fatalf("%d draws and %d draws\nspans: %s", len(da), len(db), describeSpans(spans))
		}
	}
}

// fakeFontmap gives a rune the face that a table says; the runes it does not list get the first face.
type fakeFontmap struct {
	faces  []*font.Face
	choose func(r rune) int
}

func (f fakeFontmap) ResolveFace(r rune) *font.Face { return f.faces[f.choose(r)] }

// TestSplitByFaceShortcutMatchesTheLibrary compares the split of ASCII text with the one of go-text.
func TestSplitByFaceShortcutMatchesTheLibrary(t *testing.T) {
	e := codeEngine(t)
	var faces []*font.Face
	for _, fam := range []string{"Go Mono", fonts.RobotoFamily, fonts.MaterialIconsFamily} {
		fm := e.fontsFor([]string{fam}, ui.FontWeightW400, false, ui.Locale{}, nil)
		faces = append(faces, fm.ResolveFace('a'))
	}
	rng := rand.New(rand.NewSource(3))
	alphabet := []rune("ab c\t\n\x00\x1f\x7f(){}XYZ0189_ ")
	for iter := 0; iter < 2000; iter++ {
		text := make([]rune, 1+rng.Intn(24))
		for i := range text {
			text[i] = alphabet[rng.Intn(len(alphabet))]
		}
		// Which face a rune gets: by a random table, so that faces change in the middle of the text.
		table := map[rune]int{}
		for _, r := range alphabet {
			table[r] = rng.Intn(len(faces))
		}
		fm := fakeFontmap{faces, func(r rune) int { return table[r] }}
		lo := rng.Intn(len(text))
		hi := lo + 1 + rng.Intn(len(text)-lo)
		in := shaping.Input{Text: text, RunStart: lo, RunEnd: hi, Direction: di.DirectionLTR, Script: language.Latin}
		want := shaping.SplitByFace(in, fm)
		got := append([]shaping.Input(nil), e.splitByFace(in, fm)...)
		if len(got) != len(want) {
			t.Fatalf("%q [%d,%d): %d parts, the library has %d", string(text), lo, hi, len(got), len(want))
		}
		for i := range got {
			if got[i].RunStart != want[i].RunStart || got[i].RunEnd != want[i].RunEnd || got[i].Face != want[i].Face {
				t.Fatalf("%q [%d,%d): part %d is [%d,%d) in face %p, the library has [%d,%d) in %p", string(text), lo, hi, i,
					got[i].RunStart, got[i].RunEnd, got[i].Face, want[i].RunStart, want[i].RunEnd, want[i].Face)
			}
		}
	}
	// A text that is not ASCII goes to the library.
	in := shaping.Input{Text: []rune("aé日"), RunStart: 0, RunEnd: 3, Direction: di.DirectionLTR}
	fm := fakeFontmap{faces, func(r rune) int {
		if r > 0x7f {
			return 1
		}
		return 0
	}}
	if got, want := len(e.splitByFace(in, fm)), len(shaping.SplitByFace(in, fm)); got != want {
		t.Errorf("non-ASCII text split in %d parts, the library has %d", got, want)
	}
}

// TestUnchangedLineShapesNothing counts the runs that go to the shaper.
func TestUnchangedLineShapesNothing(t *testing.T) {
	e := codeEngine(t)
	tests := []struct {
		name string
		// build is called for each build, with its number, and gives the spans.
		spans      func(n int) []string
		wantShaped []int // the runs shaped by each build
	}{
		{"the same spans", func(int) []string { return codeSpans(0, 10) }, []int{10, 0, 0}},
		{"one span changed", func(n int) []string {
			s := codeSpans(0, 10)
			if n > 0 {
				s[4] = strings.Repeat(string(rune('a'+n)), len(s[4]))
			}
			return s
		}, []int{10, 1, 1}},
		{"the first span changed", func(n int) []string {
			s := codeSpans(20, 5)
			if n > 0 {
				s[0] = strings.Repeat(string(rune('a'+n)), len(s[0]))
			}
			return s
		}, []int{5, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.forgetText()
			for n, want := range tt.wantShaped {
				before := e.shapes.shaped
				buildCode(e, tt.spans(n))
				if got := e.shapes.shaped - before; got != want {
					t.Errorf("build %d shaped %d runs, want %d", n, got, want)
				}
			}
		})
	}
	t.Run("a page of lines with one changed", func(t *testing.T) {
		e.forgetText()
		page := func(changed string) {
			for i := 0; i < pageLines; i++ {
				spans := codeSpans(i*7, 10)
				if i == 3 && changed != "" {
					spans[2] = strings.Repeat(changed, len(spans[2]))
				}
				buildCode(e, spans)
			}
		}
		page("")
		before := e.shapes.shaped
		page("")
		if got := e.shapes.shaped - before; got != 0 {
			t.Errorf("the same page shaped %d runs, want 0", got)
		}
		before = e.shapes.shaped
		page("q")
		if got := e.shapes.shaped - before; got != 1 {
			t.Errorf("a page with one span changed shaped %d runs, want 1", got)
		}
	})
}

// TestShapedRunsOfOtherSizesShareAnEntry: the engine shapes at one size, so the size is not in the key.
func TestShapedRunsOfOtherSizesShareAnEntry(t *testing.T) {
	e := codeEngine(t)
	e.forgetText()
	var widths []float64
	for _, size := range []float64{10, 14, 40} {
		b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{FontFamily: "Go Mono", FontSize: ui.Ptr(size)}))
		b.AddText("hello world")
		widths = append(widths, b.Build().MaxIntrinsicWidth())
	}
	if e.shapes.shaped != 1 || e.shapes.hits != 2 {
		t.Errorf("shaped %d, hits %d, want 1 and 2", e.shapes.shaped, e.shapes.hits)
	}
	if !(near(widths[1]/widths[0], 1.4) && near(widths[2]/widths[0], 4)) {
		t.Errorf("widths %v do not scale with the size", widths)
	}
}

// TestShapedRunsInADifferentContextDoNotShareAnEntry is the run that HarfBuzz joins to the letters next to it:
// the same letters give other glyphs next to other letters. It needs a system font with Arabic.
func TestShapedRunsInADifferentContextDoNotShareAnEntry(t *testing.T) {
	const arabic = "بببب"
	e := systemEngine(t, arabic)
	style := ui.NewTextStyle(ui.TextStyleOpts{FontSize: ui.Ptr(20.0)})
	build := func(spans ...string) *paragraph {
		b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionRTL}))
		for i, s := range spans {
			// Each span has its own style, so that each is a run.
			b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontSize: ui.Ptr(20.0 + float64(i%2))}))
			b.AddText(s)
			b.Pop()
		}
		return b.Build().Layouter().(*paragraph)
	}
	_ = style
	// The middle span is the same text in each; its neighbors change what the letters join to.
	for _, spans := range [][]string{{arabic, arabic, arabic}, {"a", arabic, "a"}, {arabic, arabic, "a"}, {"a", arabic, arabic}, {arabic}} {
		for pass := 0; pass < 2; pass++ {
			p := build(spans...)
			if got, want := runsOf(p), refRuns(p); !reflect.DeepEqual(got, want) {
				t.Fatalf("spans %q, pass %d: the runs differ from fresh shaping\ngot:  %+v\nwant: %+v", spans, pass, got, want)
			}
		}
	}
	// Two contexts that join differently shape differently: the glyphs next to a letter differ from those at the edge.
	joined := runsOf(build(arabic, arabic, arabic))
	alone := runsOf(build(arabic))
	same := len(joined) >= 2 && len(alone) == 1 && reflect.DeepEqual(glyphIDs(joined[1].Glyphs), glyphIDs(alone[0].Glyphs))
	if same {
		t.Skip("this font joins the letters the same way in every context")
	}
}

func glyphIDs(gs []glyph) []font.GID {
	out := make([]font.GID, len(gs))
	for i, g := range gs {
		out[i] = g.id
	}
	return out
}

// TestShapeCacheBounds fills a cache with more than it holds.
func TestShapeCacheBounds(t *testing.T) {
	small := func(maxRuns, maxBytes int) *shapeCache {
		c := newShapeCache()
		c.runs = newWeightedLRU(maxRuns, maxBytes, weighShapedRun)
		return c
	}
	run := func(glyphs int) *shapedRun { return &shapedRun{glyphs: make([]rawGlyph, glyphs)} }
	keys := func(c *shapeCache) []string {
		var out []string
		for n := c.runs.head; n != nil; n = n.next {
			out = append(out, n.key)
		}
		return out
	}
	t.Run("the number of runs", func(t *testing.T) {
		c := small(3, 1<<30)
		for i := 0; i < 10; i++ {
			c.runs.put(fmt.Sprint(i), run(1))
		}
		if got := keys(c); !reflect.DeepEqual(got, []string{"9", "8", "7"}) {
			t.Errorf("keys = %v, want the three newest", got)
		}
	})
	t.Run("the bytes", func(t *testing.T) {
		one := weighShapedRun("k", run(10))
		c := small(1000, 3*one)
		for i := 0; i < 10; i++ {
			c.runs.put(fmt.Sprint(i), run(10))
			if c.runs.weight > 3*one {
				t.Fatalf("after %d puts the cache weighs %d, more than %d", i+1, c.runs.weight, 3*one)
			}
		}
		if got := keys(c); !reflect.DeepEqual(got, []string{"9", "8", "7"}) {
			t.Errorf("keys = %v, want the three newest", got)
		}
	})
	t.Run("the oldest goes first and a hit protects", func(t *testing.T) {
		c := small(3, 1<<30)
		for _, k := range []string{"a", "b", "c"} {
			c.runs.put(k, run(1))
		}
		if _, ok := lruGetBytes(c.runs, []byte("a")); !ok {
			t.Fatal("a is missing")
		}
		c.runs.put("d", run(1))
		if got := keys(c); !reflect.DeepEqual(got, []string{"d", "a", "c"}) {
			t.Errorf("keys = %v, want b gone and a protected", got)
		}
	})
	t.Run("a run that is bigger than the cache is not kept", func(t *testing.T) {
		c := small(10, 1000)
		c.runs.put("small", run(1))
		c.runs.put("huge", run(100))
		if got := keys(c); !reflect.DeepEqual(got, []string{"small"}) {
			t.Errorf("keys = %v, want only the small run", got)
		}
	})
	t.Run("a paragraph that overflows the engine's cache stays correct", func(t *testing.T) {
		e := codeEngine(t)
		e.shapes.runs = newWeightedLRU(8, 1<<30, weighShapedRun)
		for i := 0; i < 40; i++ {
			p := buildSpans(e, []fuzzSpan{{fmt.Sprintf("line %d of the text", i), 1}, {"tail", 2}}, ui.TextDirectionLTR)
			if got, want := runsOf(p), refRuns(p); !reflect.DeepEqual(got, want) {
				t.Fatalf("line %d: the runs differ from fresh shaping", i)
			}
			if e.shapes.runs.len() > 8 {
				t.Fatalf("the cache holds %d runs, at most 8", e.shapes.runs.len())
			}
		}
	})
	t.Run("the limits are those of the spec", func(t *testing.T) {
		c := newShapeCache()
		if c.runs.max != 4096 || c.runs.maxWeight != 8<<20 {
			t.Errorf("limits %d runs and %d bytes, want 4096 and 8 MiB", c.runs.max, c.runs.maxWeight)
		}
	})
}

// TestFontLoadClearsTheCaches: a font that is loaded can change the face of a run, so the shaped runs
// of before are gone.
func TestFontLoadClearsTheCaches(t *testing.T) {
	e := bundledEngine(t)
	build := func(family string) *paragraph {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: family, FontFamilyFallback: []string{fonts.RobotoFamily}, FontSize: ui.Ptr(16.0)}))
		b.AddText("hello world")
		return b.Build().Layouter().(*paragraph)
	}
	first := build("Go Mono") // the family does not exist yet: Roboto draws it
	if e.shapes.runs.len() == 0 {
		t.Fatal("nothing was cached")
	}
	oldFace := first.runs[0].face
	if err := e.LoadFontFromList(gomono.TTF, "Go Mono"); err != nil {
		t.Fatal(err)
	}
	after := build("Go Mono")
	if e.shapes.runs.len() != 1 {
		t.Errorf("the cache holds %d runs after the load and one build, want only that build's", e.shapes.runs.len())
	}
	if after.runs[0].face == oldFace {
		t.Error("the run was shaped in the face of before the load")
	}
	if got, want := runsOf(after), refRuns(after); !reflect.DeepEqual(got, want) {
		t.Errorf("the runs after the load differ from fresh shaping")
	}
	// And nothing else is lost: the same text again is a hit.
	before := e.shapes.shaped
	build("Go Mono")
	if e.shapes.shaped != before {
		t.Error("the run after the load was shaped again")
	}
}

// TestParagraphIsLaidOutOnDemand: a built paragraph that is asked before Layout answers as one that was
// laid out with an infinite width.
func TestParagraphIsLaidOutOnDemand(t *testing.T) {
	e := codeEngine(t)
	spans := codeSpans(0, 5)
	lazy := buildCode(e, spans).Layouter().(*paragraph)
	if lazy.laidOut {
		t.Error("a built paragraph was laid out")
	}
	eager := buildCode(e, spans).Layouter().(*paragraph)
	eager.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	if lazy.Height() != eager.Height() || lazy.Width() != eager.Width() || lazy.LongestLine() != eager.LongestLine() {
		t.Errorf("sizes %v x %v and %v x %v", lazy.Width(), lazy.Height(), eager.Width(), eager.Height())
	}
	tests := []struct {
		name string
		call func(p *paragraph) any
	}{
		{"boxes", func(p *paragraph) any {
			return p.GetBoxesForRange(0, 20, ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
		}},
		{"position", func(p *paragraph) any { return p.GetPositionForOffset(ui.Offset{Dx: 30, Dy: 5}) }},
		{"glyph info", func(p *paragraph) any { g, ok := p.GetGlyphInfoAt(3); return []any{g, ok} }},
		{"line metrics", func(p *paragraph) any { return p.ComputeLineMetrics() }},
		{"line number", func(p *paragraph) any { n, ok := p.GetLineNumberAt(4); return []any{n, ok} }},
		{"number of lines", func(p *paragraph) any { return p.NumberOfLines() }},
		{"line boundary", func(p *paragraph) any { return p.GetLineBoundary(ui.NewTextPosition(4)) }},
		{"word boundary", func(p *paragraph) any { return p.GetWordBoundary(ui.NewTextPosition(4)) }},
		{"baselines", func(p *paragraph) any { return []float64{p.AlphabeticBaseline(), p.IdeographicBaseline()} }},
		{"exceeded", func(p *paragraph) any { return p.DidExceedMaxLines() }},
		{"decorations", func(p *paragraph) any { return p.Decorations() }},
		{"placeholders", func(p *paragraph) any { return p.GetBoxesForPlaceholders() }},
		{"paint", func(p *paragraph) any {
			n := 0
			p.PaintParagraph(func(ui.ParagraphDraw) { n++ })
			return n
		}},
	}
	for _, tt := range tests {
		l, g := buildCode(e, spans).Layouter().(*paragraph), buildCode(e, spans).Layouter().(*paragraph)
		g.Layout(ui.NewParagraphConstraints(math.Inf(1)))
		if got, want := tt.call(l), tt.call(g); !reflect.DeepEqual(got, want) {
			t.Errorf("%s of a paragraph that was not laid out = %v, want %v", tt.name, got, want)
		}
	}
}

// TestShapeKeys says what the key of a run holds: two inputs with the same key shape to the same glyphs,
// so anything that changes the glyphs must change the key, and what does not should not.
func TestShapeKeys(t *testing.T) {
	e := codeEngine(t)
	face := e.fontsFor([]string{"Go Mono"}, ui.FontWeightW400, false, ui.Locale{}, nil).ResolveFace('a')
	bold := e.fontsFor([]string{"Go Mono"}, ui.FontWeightW700, false, ui.Locale{}, nil).ResolveFace('a')
	c := newShapeCache()
	text := []rune("xx abc yy")
	input := func(mut func(in *shaping.Input)) shaping.Input {
		in := shaping.Input{Text: text, RunStart: 3, RunEnd: 6, Direction: di.DirectionLTR, Face: face, Script: language.Latin, Language: "en", Size: shapeSize * 64}
		if mut != nil {
			mut(&in)
		}
		return in
	}
	key := func(in shaping.Input) string { return string(c.makeKey(&in)) }
	base := key(input(nil))
	tests := []struct {
		name string
		mut  func(in *shaping.Input)
		same bool
	}{
		{"the same input", nil, true},
		{"the same run in another text, at another place", func(in *shaping.Input) {
			in.Text, in.RunStart, in.RunEnd = []rune("zz abc qq"), 3, 6
		}, true},
		{"the same run at the start of a longer text", func(in *shaping.Input) {
			in.Text, in.RunStart, in.RunEnd = []rune("abc"), 0, 3
		}, false}, // a run with text before it can shape differently from one without
		{"other text", func(in *shaping.Input) { in.RunStart, in.RunEnd = 3, 5 }, false},
		{"another face", func(in *shaping.Input) { in.Face = bold }, false},
		{"another direction", func(in *shaping.Input) { in.Direction = di.DirectionRTL }, false},
		{"another script", func(in *shaping.Input) { in.Script = language.Cyrillic }, false},
		{"another language", func(in *shaping.Input) { in.Language = "fr" }, false},
		{"a feature", func(in *shaping.Input) {
			in.FontFeatures = []shaping.FontFeature{{Tag: tagOf("liga"), Value: 0}}
		}, false},
		{"another size", func(in *shaping.Input) { in.Size = 14 * 64 }, true},
		{"other letters around a Latin run", func(in *shaping.Input) { in.Text = []rune("qq abc zz") }, true},
		{"other letters around an Arabic run: the joining", func(in *shaping.Input) {
			in.Script, in.Text = language.Arabic, []rune("qq abc zz")
		}, false},
	}
	for _, tt := range tests {
		got := key(input(tt.mut)) == base
		if tt.name == "other letters around an Arabic run: the joining" {
			// Compared with the same Arabic run in its first context.
			a := input(func(in *shaping.Input) { in.Script = language.Arabic })
			b := input(tt.mut)
			got = key(a) == key(b)
		}
		if got != tt.same {
			t.Errorf("%s: keys equal = %v, want %v", tt.name, got, tt.same)
		}
	}
}

// TestFastAnalysisMatchesTheGeneralPath compares the shortcuts for text of one direction (levels,
// pieces by script, grapheme clusters) with go-text's segmenter, on random text that is ASCII, Latin-1, or
// has other scripts.
func TestFastAnalysisMatchesTheGeneralPath(t *testing.T) {
	e := codeEngine(t)
	rng := rand.New(rand.NewSource(17))
	alphabets := map[string][]rune{
		"ascii":    []rune("abcXYZ019 \t\r\n\v\f\x00\x01\x1f\x7f()[]{}<>.,;:'\"-_+*/\\|!?@#$%^&~`=Ab_9"),
		"brackets": []rune("(a)[b]{c}<d>() ((x)) 1 ( ) \"(\" a"),
		"latin1":   []rune("abc é ñ ü ß \u00ad \u00a0 ( ) \r\n \u0085 ¿ ¡ « » × ÷"),
		"mixed":    []rune("abc 日本 Привет αβγ שלום مرحبا ( ) 123 \u0300 \u200d 😀 \r\n"),
	}
	for name, alphabet := range alphabets {
		for iter := 0; iter < 800; iter++ {
			text := make([]rune, rng.Intn(40))
			for i := range text {
				text[i] = alphabet[rng.Intn(len(alphabet))]
			}
			shape := shapeText(text)
			for _, lang := range []language.Language{"", "en", "ja", "ar"} {
				for _, rtl := range []bool{false, true} {
					e.noFastAnalysis = false
					fast := e.makeAnalysis(text, shape, rtl, lang)
					e.noFastAnalysis = true
					slow := e.makeAnalysis(text, shape, rtl, lang)
					if !reflect.DeepEqual(fast, slow) {
						t.Fatalf("%s %q (language %q, rtl %v): the fast analysis differs\nfast: %+v\nslow: %+v", name, string(text), lang, rtl, fast, slow)
					}
				}
			}
		}
	}
	e.noFastAnalysis = false
}
