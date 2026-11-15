package issues

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// goodAtlasFullReport is the report of a renderer that has the fix: the numbers of Vulkan on
// lavapipe.
func goodAtlasFullReport() AtlasFullReport {
	return AtlasFullReport{Frames: []AtlasFullFrame{
		{Name: "many shadows", AtlasFullStats: AtlasFullStats{TilesRasterized: 636}, Pixels: "aa", Shadow: 180},
		{Name: "many shadows again", Pixels: "aa", Shadow: 180},
		{Name: "one shadow", AtlasFullStats: AtlasFullStats{TilesRasterized: 0}, Pixels: "bb", Shadow: 200},
		{Name: "many shadows on new pages", AtlasFullStats: AtlasFullStats{TilesRasterized: 190}, Pixels: "aa", Shadow: 180},
	}}
}

func TestAtlasFullProblems(t *testing.T) {
	tests := []struct {
		name   string
		change func(r *AtlasFullReport)
		want   string // a part of the one problem; empty for none
	}{
		{"the fixed renderer", func(*AtlasFullReport) {}, ""},
		{"a frame is missing", func(r *AtlasFullReport) { r.Frames = r.Frames[:3] }, "has 3 frames"},
		{"another frame", func(r *AtlasFullReport) { r.Frames[2].Name = "two shadows" }, `is "two shadows"`},
		{"a shape left out for a full atlas", func(r *AtlasFullReport) {
			r.Frames[0].AtlasFull, r.Frames[0].Unsupported = 3, []string{"coverage atlas full"}
		}, "left something out"},
		{"a command skipped", func(r *AtlasFullReport) { r.Frames[3].Skipped = 1 }, "left something out"},
		{"a frame that fits the kept pages", func(r *AtlasFullReport) { r.Frames[0].TilesRasterized = 512; r.Frames[3].TilesRasterized = 100 }, "not the frame of the issue"},
		{"the tiles were not kept", func(r *AtlasFullReport) { r.Frames[1].TilesRasterized = 636 }, "again rasterized 636"},
		{"no page was freed", func(r *AtlasFullReport) { r.Frames[3].TilesRasterized = 0 }, "freed no page"},
		{"every page was freed", func(r *AtlasFullReport) { r.Frames[3].TilesRasterized = 636 }, "freed no page, or all"},
		{"other pixels the second time", func(r *AtlasFullReport) { r.Frames[1].Pixels = "cc" }, "many shadows again: the pixels"},
		{"other pixels on new pages", func(r *AtlasFullReport) { r.Frames[3].Pixels = "cc" }, "new pages: the pixels"},
		{"the small frame drew the large one", func(r *AtlasFullReport) { r.Frames[2].Pixels = "aa" }, "one shadow has the pixels"},
		{"no shadow", func(r *AtlasFullReport) { r.Frames[0].Shadow = 3 }, "no shadow under the panels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := goodAtlasFullReport()
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

func TestAtlasFullReportIsReadBackFromItsLines(t *testing.T) {
	tests := []struct {
		name   string
		report AtlasFullReport
	}{
		{"a good report", goodAtlasFullReport()},
		{"with features left out", func() AtlasFullReport {
			r := goodAtlasFullReport()
			r.Frames[0].AtlasFull, r.Frames[0].Skipped = 2, 1
			r.Frames[0].Unsupported = []string{"coverage atlas full", "a \"quoted\" one"}
			return r
		}()},
		{"no frame", AtlasFullReport{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The lines among the other output of an app, each after a prefix of the log.
			lines := []string{"renderer ready", ""}
			for _, l := range tt.report.Lines() {
				lines = append(lines, "2026-11-15 I/app: "+l, "another line")
			}
			got, err := ParseAtlasFull(lines)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.report) {
				t.Errorf("read back\n%+v\nwant\n%+v", got, tt.report)
			}
		})
	}
}

func TestParseAtlasFullRejectsALineThatDoesNotRead(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"cut short", `atlas-full: frame "many shadows" tiles 636 full 0`},
		{"a number that is not one", `atlas-full: frame "many shadows" tiles many full 0 skipped 0 unsupported "" pixels aa shadow 180`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAtlasFull([]string{tt.line})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseAtlasFull." + TagIssueReportInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["line"] != tt.line {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

func TestRunAtlasFull(t *testing.T) {
	const tagTestDrawFailed = "TestDrawFailed"
	frame := func() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, AtlasFullWidth, AtlasFullHeight)) }
	tests := []struct {
		name       string
		draw       func(n int) (*image.RGBA, error)
		wantFrames int
		wantRoute  string
	}{
		{"every frame drawn", func(int) (*image.RGBA, error) { return frame(), nil }, 4, ""},
		{"the first frame fails, as before the fix", func(int) (*image.RGBA, error) {
			return nil, errors.New(tagTestDrawFailed, "the frame failed")
		}, 0, "." + tagTestDrawFailed},
		{"the third frame fails", func(n int) (*image.RGBA, error) {
			if n == 2 {
				return nil, errors.New(tagTestDrawFailed, "the frame failed")
			}
			return frame(), nil
		}, 2, "." + tagTestDrawFailed},
		{"no frame", func(int) (*image.RGBA, error) { return nil, nil }, 0, "RunAtlasFull." + tagAtlasFullNoFrame},
		{"a frame of another size", func(int) (*image.RGBA, error) { return image.NewRGBA(image.Rect(0, 0, 64, 64)), nil }, 0, "RunAtlasFull." + tagAtlasFullNoFrame},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 0
			rep, first, err := RunAtlasFull(func(scene *ui.Scene, w, h int) (*image.RGBA, AtlasFullStats, error) {
				if scene == nil || w != AtlasFullWidth || h != AtlasFullHeight {
					t.Errorf("draw(%v, %d, %d)", scene, w, h)
				}
				img, err := tt.draw(n)
				n++
				return img, AtlasFullStats{TilesRasterized: n}, err
			})
			if len(rep.Frames) != tt.wantFrames {
				t.Errorf("%d frames in the report, want %d", len(rep.Frames), tt.wantFrames)
			}
			if (first != nil) != (tt.wantFrames > 0) {
				t.Errorf("the first frame: %v", first != nil)
			}
			if tt.wantRoute == "" {
				if err != nil {
					t.Fatal(err)
				}
				if rep.Frames[3].Name != "many shadows on new pages" || rep.Frames[3].TilesRasterized != 4 || len(rep.Frames[3].Pixels) != 64 {
					t.Errorf("the last frame: %+v", rep.Frames[3])
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

func TestShadowsSceneHasItsShadows(t *testing.T) {
	for _, n := range []int{1, 10} {
		if ShadowsScene(n) == nil {
			t.Fatalf("no scene of %d shadows", n)
		}
	}
	if got := len(AtlasFullScenes()); got != 4 {
		t.Fatalf("%d frames", got)
	}
}
