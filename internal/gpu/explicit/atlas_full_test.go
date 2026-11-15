package explicit_test

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// wholePages draws n paths that each need a whole page of an atlas of 256 pixel pages (they
// are larger than the surface, and each has a size of its own), starting with the one of the
// number from, and after them a rectangle that needs no atlas.
func wholePages(from, n int) *ui.Scene {
	return sceneOf(func(c *ui.Canvas) {
		for i := from; i < from+n; i++ {
			p := ui.NewPath()
			p.AddRect(ui.NewRectFromLTWH(-20, -20, 400+float64(i), 400))
			c.DrawPath(p, solid(0xFF000000))
		}
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 20, 20), solid(0xFFFF0000))
	})
}

// warnings keeps the records of a logger at Warn level and above.
type warnings struct{ records []slog.Record }

func (w *warnings) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelWarn }
func (w *warnings) Handle(_ context.Context, r slog.Record) error {
	w.records = append(w.records, r)
	return nil
}
func (w *warnings) WithAttrs([]slog.Attr) slog.Handler { return w }
func (w *warnings) WithGroup(string) slog.Handler      { return w }

// TestAFullAtlasNeverFailsTheFrame checks a frame whose shapes need more tiles than the atlas
// may have (Fishball's crash of 2026-11-15: a frame of many shadows returned AtlasFull, and
// the app stopped): the shapes that do not fit are left out and counted, the rest is drawn,
// the frame has no error whether the renderer skips what it cannot draw or not, and the log
// says it once.
func TestAFullAtlasNeverFailsTheFrame(t *testing.T) {
	tests := []struct {
		name        string
		skip        bool
		kept, limit int
		shapes      int
		drawn, full int // the shapes that are drawn and the ones left out
	}{
		{"strict, an atlas that does not grow", false, 1, 0, 3, 1, 2},
		{"skipping, an atlas that does not grow", true, 1, 0, 3, 1, 2},
		{"strict, an atlas that grows to its frame limit", false, 1, 3, 5, 3, 2},
		{"skipping, an atlas that grows to its frame limit", true, 1, 3, 5, 3, 2},
		{"strict, a frame that fits the frame limit", false, 1, 3, 3, 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &warnings{}
			r, _, _ := newRenderer(t, explicit.RendererOpts{SkipUnsupported: tt.skip, Log: slog.New(log)})
			if _, err := r.Draw(window, 200, 100, nil, false); err != nil {
				t.Fatal(err)
			}
			explicit.SetMaskAtlasForTest(r, 256, tt.kept, tt.limit)
			for frame := 0; frame < 3; frame++ {
				if _, err := r.Draw(window, 200, 100, wholePages(0, tt.shapes), false); err != nil {
					t.Fatalf("frame %d failed: %v", frame, err)
				}
				st := r.LastFrame()
				if st.AtlasFull != tt.full || st.Skipped != tt.full || st.Drawn != tt.drawn+1 || st.Commands != tt.shapes+1 {
					t.Errorf("frame %d: stats %+v, want %d shapes left out and %d commands drawn", frame, st, tt.full, tt.drawn+1)
				}
				var want []string
				if tt.full > 0 {
					want = []string{"coverage atlas full"}
				}
				if !reflect.DeepEqual(st.Unsupported, want) {
					t.Errorf("frame %d: unsupported = %v, want %v", frame, st.Unsupported, want)
				}
				// A tile for each shape that fits, and the rectangle after the ones that did not.
				if st.Instances != tt.drawn+1 {
					t.Errorf("frame %d: %d instances, want %d", frame, st.Instances, tt.drawn+1)
				}
			}
			// Three frames left shapes out: one line in the log.
			wantLines := 0
			if tt.full > 0 {
				wantLines = 1
			}
			if len(log.records) != wantLines {
				t.Fatalf("%d warnings, want %d", len(log.records), wantLines)
			}
			if wantLines == 1 {
				got := map[string]int64{}
				log.records[0].Attrs(func(a slog.Attr) bool {
					if a.Value.Kind() == slog.KindInt64 {
						got[a.Key] = a.Value.Int64()
					}
					return true
				})
				if log.records[0].Level != slog.LevelWarn || got["shapes"] != int64(tt.full) || got["pages"] != int64(max(tt.limit, tt.kept)) {
					t.Errorf("the warning is %q with %v", log.records[0].Message, got)
				}
			}
		})
	}
}

// TestAtlasPagesOfAFrameAreFreedAfterIt checks the textures of the pages: a frame that needs
// more pages than the atlas keeps gets a texture for each, the frame that no longer does has
// them destroyed, and a page made later has a texture again.
func TestAtlasPagesOfAFrameAreFreedAfterIt(t *testing.T) {
	r, dev, _ := newRenderer(t)
	if _, err := r.Draw(window, 200, 100, nil, false); err != nil {
		t.Fatal(err)
	}
	explicit.SetMaskAtlasForTest(r, 256, 1, 4)
	// The textures of a frame without a path: the rectangle is drawn by every frame below.
	if _, err := r.Draw(window, 200, 100, wholePages(0, 0), false); err != nil {
		t.Fatal(err)
	}
	base := dev.Live().Textures
	frames := []struct {
		name    string
		from, n int
		pages   int // the pages, and so their textures, after the frame
		over    int // how many of them are more than the atlas keeps
		raster  int // the tiles the frame rasterized
	}{
		{"four shapes", 0, 4, 4, 3, 4},
		{"the same four", 0, 4, 4, 3, 0},
		{"the first alone", 0, 1, 1, 0, 0},
		{"the first two", 0, 2, 2, 1, 1},
		{"two others: the tiles of the frame before give way first", 4, 2, 2, 1, 2},
		{"nothing: back to the page the atlas keeps", 0, 0, 1, 0, 0},
		{"four again", 0, 4, 4, 3, 4},
	}
	freed := 0
	for _, fr := range frames {
		if _, err := r.Draw(window, 200, 100, wholePages(fr.from, fr.n), false); err != nil {
			t.Fatalf("%s: %v", fr.name, err)
		}
		st, atlas := r.LastFrame(), explicit.MaskAtlasStatsForTest(r)
		if st.AtlasFull != 0 || st.Skipped != 0 || st.TilesRasterized != fr.raster || st.Instances != fr.n+1 {
			t.Errorf("%s: stats %+v, want %d tiles rasterized and %d instances", fr.name, st, fr.raster, fr.n+1)
		}
		if atlas.Pages != fr.pages || atlas.PagesOverBudget != fr.over {
			t.Errorf("%s: the atlas has %d pages, %d over what it keeps; want %d, %d", fr.name, atlas.Pages, atlas.PagesOverBudget, fr.pages, fr.over)
		}
		if got := dev.Live().Textures - base; got != fr.pages {
			t.Errorf("%s: %d textures for %d pages", fr.name, got, fr.pages)
		}
		freed = atlas.PagesFreed
	}
	if freed != 4 {
		t.Errorf("%d pages freed, want 4", freed)
	}
	r.Close()
	if live := dev.Live(); live.Textures != 0 {
		t.Errorf("%d textures alive after Close", live.Textures)
	}
}

// TestAFrameOfManyShadowsIsDrawnWhole draws, with the atlas of the renderer as it is, a frame
// like the one that stopped Fishball: shadows of panels and cards whose masks need more tiles
// at once than the eight pages the atlas keeps. Before step 46.1 Draw returned AtlasFull.
func TestAFrameOfManyShadowsIsDrawnWhole(t *testing.T) {
	if testing.Short() {
		t.Skip("rasterizes more than 500 tiles of 256 x 256 pixels")
	}
	const w, h, shadows = 1920, 1080, 10
	scene := sceneOf(func(c *ui.Canvas) {
		for i := 0; i < shadows; i++ {
			// Each panel has a size of its own, so that no two share their tiles.
			path := ui.NewPath()
			path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20+float64(i), 20+float64(i), 1800-float64(7*i), 1000-float64(5*i)), 12, 12))
			c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false)
		}
	})
	r, dev, _ := newRenderer(t)
	if _, err := r.Draw(window, w, h, scene, false); err != nil {
		t.Fatalf("the frame of %d shadows failed: %v", shadows, err)
	}
	st, atlas := r.LastFrame(), explicit.MaskAtlasStatsForTest(r)
	if st.AtlasFull != 0 || st.Skipped != 0 || len(st.Unsupported) != 0 || st.Drawn != shadows {
		t.Errorf("stats %+v", st)
	}
	if atlas.Pages <= gpu.AtlasMaxPages || atlas.Pages > gpu.AtlasFramePages || atlas.PagesOverBudget != atlas.Pages-gpu.AtlasMaxPages {
		t.Fatalf("the atlas has %d pages (%d over), want more than the %d it keeps: the frame is not the one of the issue", atlas.Pages, atlas.PagesOverBudget, gpu.AtlasMaxPages)
	}
	t.Logf("%d tiles rasterized, %d pages", st.TilesRasterized, atlas.Pages)
	// The same frame again reads the tiles it has: nothing is rasterized, nothing freed.
	if _, err := r.Draw(window, w, h, scene, false); err != nil {
		t.Fatal(err)
	}
	if again := r.LastFrame(); again.TilesRasterized != 0 || explicit.MaskAtlasStatsForTest(r).Pages != atlas.Pages {
		t.Errorf("the same frame again rasterized %d tiles and has %d pages", again.TilesRasterized, explicit.MaskAtlasStatsForTest(r).Pages)
	}
	// A frame of one shadow: the pages over the kept number are freed (their textures are
	// counted in TestAtlasPagesOfAFrameAreFreedAfterIt; here the layers of the shadows have
	// textures too).
	one := sceneOf(func(c *ui.Canvas) {
		path := ui.NewPath()
		path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 300, 200), 12, 12))
		c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false)
	})
	if _, err := r.Draw(window, w, h, one, false); err != nil {
		t.Fatal(err)
	}
	after := explicit.MaskAtlasStatsForTest(r)
	if after.Pages > gpu.AtlasMaxPages || after.PagesOverBudget != 0 {
		t.Errorf("after a small frame the atlas has %d pages, %d over what it keeps", after.Pages, after.PagesOverBudget)
	}
	if after.PagesFreed != atlas.Pages-after.Pages {
		t.Errorf("%d pages freed, the atlas went from %d to %d", after.PagesFreed, atlas.Pages, after.Pages)
	}
	r.Close()
	if live := dev.Live(); live.Textures != 0 {
		t.Errorf("%d textures alive after Close", live.Textures)
	}
}
