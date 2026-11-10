package text

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// outlineOf is the outline of a glyph of a face, as text.
func outlineOf(face *font.Face, gid font.GID) string {
	out, ok := face.GlyphData(gid).(font.GlyphOutline)
	if !ok {
		return ""
	}
	return fmt.Sprint(out.Segments)
}

// The 'gvar' table of a variable font is read one glyph at a time (third_party/typesetting, SPEC §10, step 37.1): the outlines
// that the variations make are the same ones, the weight axis does move them, a glyph drawn twice and on two faces agrees, and
// goroutines drawing at once do not disturb each other.
func TestVariableOutlinesAreReadOnDemand(t *testing.T) {
	c := bundledCollection(t)
	ctx := context.Background()
	f := FontFile{Path: BundledPrefix + "Roboto"}
	regular, err := c.Face(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	bold, err := c.Face(ctx, f, Variation{Tag: "wght", Value: 700})
	if err != nil {
		t.Fatal(err)
	}
	gid, _ := regular.NominalGlyph('M')
	if outlineOf(regular, gid) == "" || outlineOf(regular, gid) == outlineOf(bold, gid) {
		t.Fatal("the weight axis did not change the outline of M")
	}
	first := outlineOf(bold, gid)
	if again := outlineOf(bold, gid); again != first {
		t.Error("a glyph drawn twice has two outlines")
	}
	// The outline of the bold face does not depend on which glyphs were read before it.
	other, err := c.Face(ctx, f, Variation{Tag: "wght", Value: 700}, Variation{Tag: "wdth", Value: 75})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range "Hamburgefonstiv" {
		g, _ := regular.NominalGlyph(r)
		_ = outlineOf(other, g)
	}
	if got := outlineOf(bold, gid); got != first {
		t.Error("the outline of a face changed after another face read other glyphs")
	}

	var wg sync.WaitGroup
	results := make([]map[rune]string, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := map[rune]string{}
			// Faces are not shared between goroutines, but the font under them is: the glyphs' variations are read under its lock.
			face := font.NewFace(regular.Font)
			face.SetVariations([]font.Variation{{Tag: ot.MustNewTag("wght"), Value: 500}})
			for _, r := range "The quick brown fox jumps over the lazy dog 0123456789" {
				g, _ := face.NominalGlyph(r)
				m[r] = outlineOf(face, g)
			}
			results[i] = m
		}()
	}
	wg.Wait()
	for i := 1; i < len(results); i++ {
		if !reflect.DeepEqual(results[0], results[i]) {
			t.Fatalf("goroutine %d drew other outlines than goroutine 0", i)
		}
	}
}

// Opening a variable font does not keep the variations of every glyph: the memory it holds stays near the size of the file's
// tables, where unpacking every glyph's deltas took several times that.
func TestOpeningAVariableFontKeepsLittle(t *testing.T) {
	c := bundledCollection(t)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	face, err := c.Face(context.Background(), FontFile{Path: BundledPrefix + "Roboto"})
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(face)
	held := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	const file = 2352808 // material/fonts/Roboto-Variable.ttf
	if held > 2*file {
		t.Errorf("opening Roboto-Variable holds %d bytes, more than twice the %d of its file", held, file)
	}
}
