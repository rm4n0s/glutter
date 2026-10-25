package engine

import (
	"bytes"
	"image"
	"image/png"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
)

// Environment variables of the self-capture used for proof screenshots
// (docs/IMPLEMENTATION.md, proof method B).
const (
	// ScreenshotEnv names the file to save the window content to. When it is set,
	// the app saves a PNG of its window and exits with code 0.
	ScreenshotEnv = "GLUTTER_SCREENSHOT"
	// ScreenshotAfterEnv delays the capture by a duration such as "500ms", to
	// catch an animation midway.
	ScreenshotAfterEnv = "GLUTTER_SCREENSHOT_AFTER"
)

// androidScreenshotPath is the file of the self-capture on Android: the value "1" (what
// glutter-android shot sets, SPEC §24.13) means glutter-screenshot.png in the app's files
// directory, HOME, which the embedding sets; any other value is a path as on Linux. It is plain
// Go so that it is tested everywhere.
func androidScreenshotPath(value string, getenv func(string) string) string {
	if value != "1" {
		return value
	}
	return path.Join(getenv("HOME"), "glutter-screenshot.png") // an Android path, with slashes on every host
}

// Screenshot is a pending capture of the window content to a PNG file.
//
// Flutter: no counterpart
type Screenshot struct {
	path  string
	after time.Duration
}

// ScreenshotFromEnv reads GLUTTER_SCREENSHOT and GLUTTER_SCREENSHOT_AFTER with
// getenv (os.Getenv in an app). It returns nil when no capture is asked for.
// A delay that is not a non-negative duration fails with ScreenshotAfterInvalid.
//
// Flutter: no counterpart
func ScreenshotFromEnv(getenv func(string) string) (*Screenshot, error) {
	path := getenv(ScreenshotEnv)
	if path == "" {
		return nil, nil
	}
	s := &Screenshot{path: screenshotPath(path, getenv)}
	if v := getenv(ScreenshotAfterEnv); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 {
			return nil, errors.New(tagScreenshotAfterInvalid, "GLUTTER_SCREENSHOT_AFTER must be a duration such as 500ms",
				"value", v)
		}
		s.after = d
	}
	return s, nil
}

// Path returns the file the PNG is written to.
//
// Flutter: no counterpart
func (s *Screenshot) Path() string { return s.path }

// After returns how long to wait, after the first frame was presented, before
// capturing.
//
// Flutter: no counterpart
func (s *Screenshot) After() time.Duration { return s.after }

// Encode returns img as a PNG. It fails with ScreenshotWriteFailed. A platform that
// cannot write files hands the bytes to the host instead of calling Save
// (platform.ScreenshotPoster).
//
// Flutter: no counterpart
func (s *Screenshot) Encode(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, errors.NewErr(tagScreenshotWriteFailed, err).SetMetadata("path", s.path)
	}
	return buf.Bytes(), nil
}

// Save writes img to the file as a PNG, creating the directory if needed, and
// logs the path. It fails with ScreenshotWriteFailed.
//
// Flutter: no counterpart
func (s *Screenshot) Save(img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return errors.NewErr(tagScreenshotWriteFailed, err).SetMetadata("path", s.path)
	}
	f, err := os.Create(s.path)
	if err != nil {
		return errors.NewErr(tagScreenshotWriteFailed, err).SetMetadata("path", s.path)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close() // the encoding error is the one to report
		return errors.NewErr(tagScreenshotWriteFailed, err).SetMetadata("path", s.path)
	}
	if err := f.Close(); err != nil {
		return errors.NewErr(tagScreenshotWriteFailed, err).SetMetadata("path", s.path)
	}
	slog.Info("screenshot saved", "path", s.path, "width", img.Bounds().Dx(), "height", img.Bounds().Dy())
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagScreenshotAfterInvalid = "ScreenshotAfterInvalid"
	tagScreenshotWriteFailed  = "ScreenshotWriteFailed"
)
