package goldens

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeT records what Check reports instead of failing the real test.
type fakeT struct {
	errors []string
	fatal  string
	logs   []string
}

func (f *fakeT) Helper() {}
func (f *fakeT) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}
func (f *fakeT) Fatalf(format string, args ...any) { f.fatal = fmt.Sprintf(format, args...) }
func (f *fakeT) Logf(format string, args ...any) {
	f.logs = append(f.logs, fmt.Sprintf(format, args...))
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestCompare(t *testing.T) {
	base := solid(4, 4, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	near := solid(4, 4, color.RGBA{R: 102, G: 99, B: 100, A: 255})
	far := solid(4, 4, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	far.SetRGBA(1, 2, color.RGBA{R: 100, G: 100, B: 110, A: 255})
	tests := []struct {
		name       string
		got        *image.RGBA
		tol        int
		wantOK     bool
		wantPixels int
		wantMax    int
		wantSize   bool
	}{
		{"identical", base, 0, true, 0, 0, false},
		{"within tolerance", near, 2, true, 0, 2, false},
		{"over tolerance", near, 1, false, 16, 2, false},
		{"one pixel differs", far, 2, false, 1, 10, false},
		{"size differs", solid(3, 4, color.RGBA{A: 255}), 2, false, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Compare(base, tt.got, tt.tol)
			if res.OK() != tt.wantOK || res.Pixels != tt.wantPixels || res.MaxDiff != tt.wantMax || res.SizeMismatch != tt.wantSize {
				t.Errorf("got %+v, want ok=%v pixels=%d max=%d size=%v", res, tt.wantOK, tt.wantPixels, tt.wantMax, tt.wantSize)
			}
			if (res.Diff != nil) != (tt.wantPixels > 0) {
				t.Errorf("Diff = %v, want it only when pixels differ", res.Diff != nil)
			}
		})
	}
	// The differing pixel is red in the diff, the others are faded.
	res := Compare(base, far, 2)
	if got := res.Diff.RGBAAt(1, 2); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("diff pixel = %v, want red", got)
	}
	if got := res.Diff.RGBAAt(0, 0); got.R != got.G || got.A != 255 || got.R <= 100 {
		t.Errorf("faded pixel = %v, want a lighter gray", got)
	}
}

func TestCheckFlow(t *testing.T) {
	dir := t.TempDir()
	img := solid(3, 2, color.RGBA{R: 200, A: 255})

	// A missing golden is fatal and says how to create it.
	ft := &fakeT{}
	check(ft, dir, "a/b", img, 2, false)
	if !strings.Contains(ft.fatal, "-update-goldens") {
		t.Errorf("missing golden: fatal = %q", ft.fatal)
	}

	// Updating writes the golden, creating directories.
	ft = &fakeT{}
	check(ft, dir, "a/b", img, 2, true)
	if ft.fatal != "" || len(ft.errors) != 0 {
		t.Fatalf("update failed: %+v", ft)
	}
	if _, err := os.Stat(filepath.Join(dir, "a", "b.png")); err != nil {
		t.Fatalf("golden not written: %v", err)
	}

	// The same image matches and leaves no files.
	ft = &fakeT{}
	check(ft, dir, "a/b", img, 2, false)
	if ft.fatal != "" || len(ft.errors) != 0 {
		t.Errorf("match reported %+v", ft)
	}

	// A different image fails and writes the actual and the diff beside the golden.
	other := solid(3, 2, color.RGBA{R: 0, A: 255})
	ft = &fakeT{}
	check(ft, dir, "a/b", other, 2, false)
	if len(ft.errors) != 1 || !strings.Contains(ft.errors[0], "6 pixels differ") {
		t.Errorf("mismatch reported %+v", ft)
	}
	for _, f := range []string{"b.actual.png", "b.diff.png"} {
		if _, err := os.Stat(filepath.Join(dir, "a", f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}

	// A different size fails with the sizes.
	ft = &fakeT{}
	check(ft, dir, "a/b", solid(5, 5, color.RGBA{A: 255}), 2, false)
	if len(ft.errors) != 1 || !strings.Contains(ft.errors[0], "size is") {
		t.Errorf("size mismatch reported %+v", ft)
	}
	if _, err := os.Stat(filepath.Join(dir, "a", "b.diff.png")); err == nil {
		// The diff of the earlier mismatch is still there; only the size report matters.
		t.Log("diff of the earlier run still present")
	}

	// Images that are not RGBA with an origin at 0,0 are converted.
	gray := image.NewGray(image.Rect(5, 5, 8, 7))
	ft = &fakeT{}
	check(ft, dir, "gray", gray, 2, true)
	check(ft, dir, "gray", gray, 2, false)
	if ft.fatal != "" || len(ft.errors) != 0 {
		t.Errorf("gray image: %+v", ft)
	}
}

func TestPNGErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		fn   func() error
		tag  string
	}{
		{"missing", func() error { _, err := ReadPNG(filepath.Join(dir, "none.png")); return err }, "GoldenMissing"},
		{"corrupt", func() error { _, err := ReadPNG(bad); return err }, "GoldenReadFailed"},
		{"unreadable", func() error { _, err := ReadPNG(dir); return err }, "GoldenReadFailed"},
		{"directory is a file", func() error { return WritePNG(filepath.Join(blocker, "x.png"), solid(1, 1, color.RGBA{})) }, "GoldenWriteFailed"},
		{"path is a directory", func() error { return WritePNG(dir, solid(1, 1, color.RGBA{})) }, "GoldenWriteFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) {
				t.Errorf("expected %s, got %v", tt.tag, err)
			}
		})
	}
}
