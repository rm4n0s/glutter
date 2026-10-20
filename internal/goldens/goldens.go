// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

// Package goldens compares rendered images with the golden PNGs of a package's
// tests.
//
// A golden lives in the testdata/goldens directory of the package under test. On
// a mismatch Check writes the image it got and a picture of the differences
// beside the golden, as <name>.actual.png and <name>.diff.png. Running the tests
// with -update-goldens rewrites the goldens; every changed one must be looked at
// before it is committed.
//
// Flutter: flutter_test/lib/src/goldens.dart
package goldens

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
)

var update = flag.Bool("update-goldens", false, "rewrite the golden images instead of comparing with them")

// DefaultTolerance is the largest difference in one channel of one pixel, out of
// 255, that Check accepts. It absorbs the float rounding of different
// processors.
const DefaultTolerance = 2

// testingT is the part of testing.TB that Check needs.
type testingT interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

// Check compares img with testdata/goldens/<name>.png, with DefaultTolerance.
//
// Flutter: flutter_test/lib/src/matchers.dart matchesGoldenFile
func Check(t testingT, name string, img image.Image) {
	t.Helper()
	CheckTolerance(t, name, img, DefaultTolerance)
}

// CheckTolerance is Check with an explicit tolerance for each channel of each
// pixel.
//
// Flutter: flutter_test/lib/src/matchers.dart matchesGoldenFile
func CheckTolerance(t testingT, name string, img image.Image, tolerance int) {
	t.Helper()
	check(t, filepath.Join("testdata", "goldens"), name, img, tolerance, *update)
}

func check(t testingT, dir, name string, img image.Image, tolerance int, update bool) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name)+".png")
	got := toRGBA(img)
	if update {
		if err := WritePNG(path, got); err != nil {
			t.Fatalf("%v", err)
			return
		}
		t.Logf("updated golden %s", path)
		return
	}
	want, err := ReadPNG(path)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == "GoldenMissing" {
			t.Fatalf("golden %s does not exist; run the test with -update-goldens to create it", path)
			return
		}
		t.Fatalf("%v", err)
		return
	}
	res := Compare(want, got, tolerance)
	if res.OK() {
		return
	}
	base := filepath.Join(dir, filepath.FromSlash(name))
	var werrs []error
	werrs = append(werrs, WritePNG(base+".actual.png", got))
	if res.Diff != nil {
		werrs = append(werrs, WritePNG(base+".diff.png", res.Diff))
	}
	for _, e := range werrs {
		if e != nil {
			t.Errorf("%v", e)
		}
	}
	if res.SizeMismatch {
		t.Errorf("golden %s: size is %v, want %v", path, got.Bounds().Size(), want.Bounds().Size())
		return
	}
	t.Errorf("golden %s: %d pixels differ by more than %d (largest difference %d); see %s.actual.png and %s.diff.png",
		path, res.Pixels, tolerance, res.MaxDiff, base, base)
}

// Result is the outcome of comparing two images.
//
// Flutter: flutter_test/lib/src/goldens.dart ComparisonResult
type Result struct {
	// SizeMismatch says that the images have different sizes; nothing else is
	// set then.
	SizeMismatch bool
	// Pixels is the number of pixels with a channel that differs by more than
	// the tolerance.
	Pixels int
	// MaxDiff is the largest difference of any channel of any pixel.
	MaxDiff int
	// Diff shows the pixels that differ more than the tolerance in red over a
	// faded copy of the golden; nil if the images match.
	Diff *image.RGBA
}

// OK says that the images match.
func (r Result) OK() bool { return !r.SizeMismatch && r.Pixels == 0 }

// Compare compares two images pixel by pixel; a channel may differ by up to
// tolerance.
//
// Flutter: flutter_test/lib/src/goldens.dart GoldenFileComparator.compare
func Compare(want, got *image.RGBA, tolerance int) Result {
	if want.Bounds().Size() != got.Bounds().Size() {
		return Result{SizeMismatch: true}
	}
	b := want.Bounds()
	res := Result{}
	diff := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			w := want.Pix[want.PixOffset(b.Min.X+x, b.Min.Y+y):][:4]
			g := got.Pix[got.PixOffset(got.Bounds().Min.X+x, got.Bounds().Min.Y+y):][:4]
			worst := 0
			for i := 0; i < 4; i++ {
				worst = max(worst, absDiff(w[i], g[i]))
			}
			res.MaxDiff = max(res.MaxDiff, worst)
			if worst > tolerance {
				res.Pixels++
				diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
				continue
			}
			// A faded copy of the golden, over white.
			a := int(w[3])
			diff.SetRGBA(x, y, color.RGBA{R: lighten(w[0], a), G: lighten(w[1], a), B: lighten(w[2], a), A: 255})
		}
	}
	if res.Pixels > 0 {
		res.Diff = diff
	}
	return res
}

// lighten composites a premultiplied channel over white and fades it toward white.
func lighten(c uint8, a int) uint8 {
	over := int(c) + (255 - a)
	return uint8(255 - (255-over)/3)
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// toRGBA returns the image as premultiplied RGBA with its origin at 0, 0.
func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

// WritePNG saves an image as a PNG file, creating the directories.
//
// Flutter: no counterpart
func WritePNG(path string, img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.NewErr("GoldenWriteFailed", err).SetMetadata("path", path)
	}
	f, err := os.Create(path)
	if err != nil {
		return errors.NewErr("GoldenWriteFailed", err).SetMetadata("path", path)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return errors.NewErr("GoldenWriteFailed", err).SetMetadata("path", path)
	}
	if err := f.Close(); err != nil {
		return errors.NewErr("GoldenWriteFailed", err).SetMetadata("path", path)
	}
	return nil
}

// ReadPNG loads a PNG file as premultiplied RGBA. A missing file is the error
// GoldenMissing.
//
// Flutter: no counterpart
func ReadPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewErr("GoldenMissing", err).SetMetadata("path", path)
		}
		return nil, errors.NewErr("GoldenReadFailed", err).SetMetadata("path", path)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		return nil, errors.NewErr("GoldenReadFailed", err).SetMetadata("path", path)
	}
	return toRGBA(img), nil
}
