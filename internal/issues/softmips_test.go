package issues

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// goodSoftMipmapsReport is the report of a renderer that has the fix.
func goodSoftMipmapsReport() SoftMipmapsReport {
	names := []string{"a spinner over a picture", "the spinner turns 1", "the spinner turns 2", "the spinner turns 3", "the spinner turns 4", "the spinner turns 5", "another picture", "the first picture again"}
	rep := SoftMipmapsReport{}
	for i, n := range names {
		f := SoftMipmapsFrame{Name: n, Chains: 1, Picture: "aa", Spinner: string(rune('a' + i)), Block: true}
		switch i {
		case 0:
			f.Built = 1
		case 6:
			f.Built, f.Chains, f.Picture = 1, 2, "bb"
		case 7:
			f.Chains = 2
		}
		rep.Frames = append(rep.Frames, f)
	}
	return rep
}

func TestSoftMipmapsProblems(t *testing.T) {
	tests := []struct {
		name   string
		change func(r *SoftMipmapsReport)
		want   string // a part of the one problem; empty for none
	}{
		{"the fixed renderer", func(*SoftMipmapsReport) {}, ""},
		{"a frame is missing", func(r *SoftMipmapsReport) { r.Frames = r.Frames[:7] }, "has 7 frames"},
		{"another frame", func(r *SoftMipmapsReport) { r.Frames[3].Name = "a pause" }, `is "a pause"`},
		{"the chain made again, the issue", func(r *SoftMipmapsReport) { r.Frames[2].Built = 1 }, "the spinner turns 2: the frame made 1 mipmap chains, want 0"},
		{"the chain made for each rectangle", func(r *SoftMipmapsReport) { r.Frames[0].Built = 3 }, "made 3 mipmap chains, want 1"},
		{"no chain for the picture", func(r *SoftMipmapsReport) { r.Frames[0].Built = 0 }, "made 0 mipmap chains, want 1"},
		{"the first picture made again", func(r *SoftMipmapsReport) { r.Frames[7].Built = 1 }, "the first picture again: the frame made 1"},
		{"no chain kept", func(r *SoftMipmapsReport) { r.Frames[4].Chains = 0 }, "holds 0 mipmap chains, want 1"},
		{"the first chain dropped", func(r *SoftMipmapsReport) { r.Frames[7].Chains = 1 }, "holds 1 mipmap chains, want 2"},
		{"the picture's color", func(r *SoftMipmapsReport) { r.Frames[5].Block = false }, "does not have the color"},
		{"other pixels of the picture", func(r *SoftMipmapsReport) { r.Frames[7].Picture = "cc" }, "the first picture again: the picture's pixels"},
		{"the first picture where the other is", func(r *SoftMipmapsReport) { r.Frames[6].Picture = "aa" }, "the pixels are those of the first picture"},
		{"a frame not drawn", func(r *SoftMipmapsReport) { r.Frames[3].Spinner = r.Frames[2].Spinner }, "the spinner turns 3: the spinner did not turn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := goodSoftMipmapsReport()
			tt.change(&r)
			got := r.Problems()
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("problems of a good report: %q", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("problems: %q, want one with %q", got, tt.want)
			}
		})
	}
}

func TestSoftMipmapsReportIsReadBackFromItsLines(t *testing.T) {
	tests := []struct {
		name   string
		report SoftMipmapsReport
	}{
		{"a good report", goodSoftMipmapsReport()},
		{"a bad one", func() SoftMipmapsReport {
			r := goodSoftMipmapsReport()
			r.Frames[1].Built, r.Frames[1].Block = 4, false
			return r
		}()},
		{"no frame", SoftMipmapsReport{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The lines among the other output of an app, each after a prefix of the log.
			lines := []string{"renderer ready", ""}
			for _, l := range tt.report.Lines() {
				lines = append(lines, "2026-11-18 I/app: "+l, "another line")
			}
			got, err := ParseSoftMipmaps(lines)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.report) {
				t.Errorf("read back\n%+v\nwant\n%+v", got, tt.report)
			}
		})
	}
}

func TestParseSoftMipmapsRejectsALineThatDoesNotRead(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"cut short", `soft-mipmaps: frame "another picture" built 1 chains`},
		{"a number that is not one", `soft-mipmaps: frame "another picture" built one chains 2 picture aa spinner bb block true`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSoftMipmaps([]string{tt.line})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseSoftMipmaps." + TagIssueReportInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["line"] != tt.line {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

// TestRunSoftMipmapsOnTheSoftwareRenderer runs the reproduction on the software renderer itself,
// as a window draws: the whole first frame, and then only the rectangles the damage tracker finds,
// with one soft.Cache and EndFrame after each frame. The issue's way (a cache for each frame, as
// renderInto had before step 49.1) is found.
func TestRunSoftMipmapsOnTheSoftwareRenderer(t *testing.T) {
	tests := []struct {
		name string
		// cache returns the cache of a frame.
		cache func(kept *soft.Cache) *soft.Cache
		want  []string // parts of the problems; none for the fixed renderer
	}{
		{"one cache, the fix", func(kept *soft.Cache) *soft.Cache { return kept }, nil},
		{"a cache for each frame, the issue", func(*soft.Cache) *soft.Cache { return soft.NewCache() }, []string{
			"the spinner turns 1: the frame made 1 mipmap chains, want 0",
			"the first picture again: the frame made 1 mipmap chains, want 0",
			"the first picture again: the renderer holds 1 mipmap chains, want 2",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept := soft.NewCache()
			tracker := gpu.NewDamageTracker()
			canvas := image.NewRGBA(image.Rect(0, 0, SoftMipmapsWidth, SoftMipmapsHeight))
			var built int
			rep, err := RunSoftMipmaps(func(scene *ui.Scene, w, h int) (*image.RGBA, SoftMipmapsStats, error) {
				c := tt.cache(kept)
				damage := tracker.Update(scene, w, h)
				rects := damage.Rects
				if damage.Full {
					rects = []image.Rectangle{canvas.Rect}
				}
				for _, r := range rects {
					for y := r.Min.Y; y < r.Max.Y; y++ {
						o := canvas.PixOffset(r.Min.X, y)
						clear(canvas.Pix[o : o+4*r.Dx()])
					}
					if err := c.RenderInto(canvas, scene, r); err != nil {
						return nil, SoftMipmapsStats{}, err
					}
				}
				c.EndFrame()
				st := c.Stats()
				if c != kept {
					built += st.Built
					st.Built = built
				}
				out := image.NewRGBA(canvas.Rect)
				copy(out.Pix, canvas.Pix)
				return out, SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			got := rep.Problems()
			if len(tt.want) == 0 && len(got) != 0 {
				t.Fatalf("problems: %q", got)
			}
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					found = found || strings.Contains(g, w)
				}
				if !found {
					t.Errorf("no problem %q in %q", w, got)
				}
			}
		})
	}
}

func TestRunSoftMipmaps(t *testing.T) {
	const tagTestDrawFailed = "TestDrawFailed"
	frame := func() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, SoftMipmapsWidth, SoftMipmapsHeight)) }
	tests := []struct {
		name       string
		draw       func(n int) (*image.RGBA, error)
		wantFrames int
		wantRoute  string
	}{
		{"every frame drawn", func(int) (*image.RGBA, error) { return frame(), nil }, 8, ""},
		{"the third frame fails", func(n int) (*image.RGBA, error) {
			if n == 2 {
				return nil, errors.New(tagTestDrawFailed, "the frame failed")
			}
			return frame(), nil
		}, 2, "." + tagTestDrawFailed},
		{"no frame", func(int) (*image.RGBA, error) { return nil, nil }, 0, "RunSoftMipmaps." + tagSoftMipmapsNoFrame},
		{"a frame of another size", func(int) (*image.RGBA, error) { return image.NewRGBA(image.Rect(0, 0, 64, 64)), nil }, 0, "RunSoftMipmaps." + tagSoftMipmapsNoFrame},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 0
			rep, err := RunSoftMipmaps(func(scene *ui.Scene, w, h int) (*image.RGBA, SoftMipmapsStats, error) {
				if scene == nil || w != SoftMipmapsWidth || h != SoftMipmapsHeight {
					t.Errorf("draw(%v, %d, %d)", scene, w, h)
				}
				img, err := tt.draw(n)
				n++
				return img, SoftMipmapsStats{Built: 3 * n, Chains: n}, err
			})
			if len(rep.Frames) != tt.wantFrames {
				t.Errorf("%d frames in the report, want %d", len(rep.Frames), tt.wantFrames)
			}
			if tt.wantRoute == "" {
				if err != nil {
					t.Fatal(err)
				}
				// Built is per frame: the difference of the renderer's counts.
				if f := rep.Frames[7]; f.Name != "the first picture again" || f.Built != 3 || f.Chains != 8 || len(f.Picture) != 64 {
					t.Errorf("the last frame: %+v", f)
				}
				if rep.Frames[0].Built != 3 {
					t.Errorf("the first frame: %+v", rep.Frames[0])
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
