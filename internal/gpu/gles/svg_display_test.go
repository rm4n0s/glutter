//go:build glutter_display

package gles

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/svg"
	"github.com/rm4n0s/glutter/ui"
)

// svgScene is a document of internal/svg/testdata/reference drawn at twice its size, as a scene.
func svgScene(t *testing.T, name string) (*ui.Scene, int, int) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "svg", "testdata", "reference", name+".svg"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := svg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	size := d.Size()
	const scale = 2
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Scale(scale, scale)
	c.DrawPicture(d.PictureAt(scale))
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build(), int(size.Width * scale), int(size.Height * scale)
}

// The filters of the SVG painter run on the CPU and reach the renderers as images, and masks are layers: the GPU draws
// the documents with filters, masks, markers and text within a tolerance of the software renderer (step 29.3).
func TestDisplaySVGDocumentsMatchSoftware(t *testing.T) {
	tests := []struct {
		name string
		tol  tolerance
	}{
		{"filter-blur", tolerance{over8Permille: 10, over32Permille: 2}},
		{"filter-shadow-merge", tolerance{over8Permille: 10, over32Permille: 2}},
		{"filter-graph-named", tolerance{over8Permille: 10, over32Permille: 2}},
		{"filter-color-matrix", tolerance{over8Permille: 5, over32Permille: 1}},
		{"filter-composite-ops", tolerance{over8Permille: 5, over32Permille: 1}},
		{"filter-text", tolerance{over8Permille: 10, over32Permille: 2}},
		{"filter-group-opacity", tolerance{over8Permille: 10, over32Permille: 2}},
		{"mask-luminance", tolerance{over8Permille: 10, over32Permille: 2}},
		{"mask-alpha", tolerance{over8Permille: 10, over32Permille: 2}},
		{"mask-nested", tolerance{over8Permille: 10, over32Permille: 2}},
		{"mask-group-opacity", tolerance{over8Permille: 10, over32Permille: 2}},
		{"marker-orient-auto", tolerance{over8Permille: 20, over32Permille: 5}},
		{"marker-viewbox-ref", tolerance{over8Permille: 20, over32Permille: 5}},
		{"text-basic", tolerance{over8Permille: 20, over32Permille: 5}},
		{"text-path-line", tolerance{over8Permille: 20, over32Permille: 5}},
	}
	type job struct {
		scene *ui.Scene
		w, h  int
		want  *image.RGBA
		got   *image.RGBA
	}
	jobs := make([]*job, len(tests))
	for i, tt := range tests {
		scene, w, h := svgScene(t, tt.name)
		want, err := soft.Render(scene, w, h)
		if err != nil {
			t.Fatal(err)
		}
		jobs[i] = &job{scene: scene, w: w, h: h, want: want}
	}
	var skipped int
	var skippedWhat []string
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(h); err != nil {
			return err
		}
		for i, j := range jobs {
			got, err := renderGPU(r, j.scene, j.w, j.h)
			if err != nil {
				return err
			}
			jobs[i].got = got
			st := r.LastFrame()
			t.Logf("%s: frame %+v", tests[i].name, st)
			if st.Skipped != 0 {
				skipped += st.Skipped
				skippedWhat = append(skippedWhat, st.Unsupported...)
			}
		}
		return nil
	})
	if skipped != 0 {
		t.Errorf("the GPU skipped %d commands: %v", skipped, skippedWhat)
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := jobs[i]
			tt.tol.check(t, tt.name, diffRect(j.got, j.want, j.got.Bounds()))
		})
	}
}
