package engine

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestScreenshotFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantNil   bool
		wantPath  string
		wantAfter time.Duration
		wantTag   string
	}{
		{name: "not asked for", env: map[string]string{}, wantNil: true},
		{name: "a delay without a path is ignored", env: map[string]string{ScreenshotAfterEnv: "1s"}, wantNil: true},
		{name: "path only", env: map[string]string{ScreenshotEnv: "out.png"}, wantPath: "out.png"},
		{name: "path and delay", env: map[string]string{ScreenshotEnv: "a/b.png", ScreenshotAfterEnv: "500ms"}, wantPath: "a/b.png", wantAfter: 500 * time.Millisecond},
		{name: "zero delay", env: map[string]string{ScreenshotEnv: "x.png", ScreenshotAfterEnv: "0s"}, wantPath: "x.png"},
		{name: "a delay that is not a duration", env: map[string]string{ScreenshotEnv: "x.png", ScreenshotAfterEnv: "soon"}, wantTag: "ScreenshotAfterInvalid"},
		{name: "a number without a unit", env: map[string]string{ScreenshotEnv: "x.png", ScreenshotAfterEnv: "500"}, wantTag: "ScreenshotAfterInvalid"},
		{name: "a negative delay", env: map[string]string{ScreenshotEnv: "x.png", ScreenshotAfterEnv: "-1s"}, wantTag: "ScreenshotAfterInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ScreenshotFromEnv(env(tt.env))
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag || !appErr.HasRoute("ScreenshotFromEnv.ScreenshotAfterInvalid") {
					t.Fatalf("error = %v, want %s via ScreenshotFromEnv", err, tt.wantTag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantNil {
				if s != nil {
					t.Fatalf("got %+v, want nil", s)
				}
				return
			}
			if s.Path() != tt.wantPath || s.After() != tt.wantAfter {
				t.Errorf("path %q after %v, want %q after %v", s.Path(), s.After(), tt.wantPath, tt.wantAfter)
			}
		})
	}
}

func TestSaveWritesAPNGThatDecodesToTheSamePixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	colors := []color.RGBA{{0x67, 0x50, 0xA4, 255}, {255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {1, 2, 3, 255}, {9, 8, 7, 255}}
	for i, c := range colors {
		img.SetRGBA(i%3, i/3, c)
	}
	// The directory does not exist yet: Save creates it.
	path := filepath.Join(t.TempDir(), "nested", "dir", "shot.png")
	s := &Screenshot{path: path}
	if err := s.Save(img); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatalf("the file is not a PNG: %v", err)
	}
	if b := got.Bounds(); b.Dx() != 3 || b.Dy() != 2 {
		t.Fatalf("size = %v, want 3x2", b)
	}
	for i, want := range colors {
		r, g, b, a := got.At(i%3, i/3).RGBA()
		if uint8(r>>8) != want.R || uint8(g>>8) != want.G || uint8(b>>8) != want.B || uint8(a>>8) != want.A {
			t.Errorf("pixel %d = %v, want %v", i, []uint32{r >> 8, g >> 8, b >> 8, a >> 8}, want)
		}
	}
}

func TestSaveFailures(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	dir := t.TempDir()
	file := filepath.Join(dir, "plain-file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
	}{
		{"the parent is a file", filepath.Join(file, "shot.png")},
		{"the path is a directory", dir},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Screenshot{path: tt.path}).Save(img)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagScreenshotWriteFailed || !appErr.HasRoute("Screenshot.Save.ScreenshotWriteFailed") {
				t.Fatalf("Save = %v, want ScreenshotWriteFailed via Screenshot.Save", err)
			}
			if appErr.Metadata["path"] != tt.path {
				t.Errorf("Metadata[path] = %v, want %s", appErr.Metadata["path"], tt.path)
			}
		})
	}
}
