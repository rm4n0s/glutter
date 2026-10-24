//go:build glutter_android

package androidtest

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fontLines are the names of the lines of examples/dev/android-fonts.
var fontLines = []string{"latin", "greek", "cyrillic", "arabic", "hebrew", "devanagari", "thai", "chinese", "japanese", "korean", "emoji"}

// fontsApp installs examples/dev/android-fonts and starts it with the renderer soft and the locale
// (GLUTTER_DEMO); it returns once the app has drawn its first frame and logged the coverage of every
// line.
func fontsApp(ctx context.Context, t *testing.T, e *Emulator, locale string) (appID, log string) {
	t.Helper()
	appID = installApp(ctx, t, e, "./examples/dev/android-fonts")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	extras := map[string]string{"GLUTTER_RENDERER": "soft"}
	if locale != "" {
		extras["GLUTTER_DEMO"] = locale
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	log, err := e.WaitLog(ctx, appID, "coverage line=emoji", time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID, log
}

// TestEveryCharacterOfTheTextPageHasASystemFont has the system font matcher of the app asked, on the
// device, for a font with a glyph for each character of Latin, Greek, Cyrillic, Arabic, Hebrew,
// Devanagari, Thai, Chinese, Japanese, Korean and emoji text, in the locales en-US and ja-JP; none is
// left to a missing-glyph box. The screen is saved in GLUTTER_PROOF_DIR when that is set.
func TestEveryCharacterOfTheTextPageHasASystemFont(t *testing.T) {
	for _, locale := range []string{"en-US", "ja-JP"} {
		t.Run(locale, func(t *testing.T) {
			forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
				_, log := fontsApp(ctx, t, e, locale)
				for _, name := range fontLines {
					// The last one is this run's: the log of an emulator that ran the app a moment ago can still
					// hold the end of that run.
					got := lines(log, "coverage line="+name+" ")
					if len(got) == 0 {
						t.Errorf("line %s: the log has no coverage line", name)
						continue
					}
					if last := got[len(got)-1]; !strings.Contains(last, " missing=0 ") {
						t.Errorf("line %s has characters with no font: %s", name, last)
					}
				}
				// The page's background is the surface color of the theme, which is nearly white: the screen
				// is the app's, not the launcher's.
				var img *image.RGBA
				for end := time.Now().Add(40 * time.Second); ; time.Sleep(time.Second) {
					img = screenWithContent(ctx, t, e)
					light := 0
					for i := 0; i < len(img.Pix); i += 4 {
						if img.Pix[i] > 240 && img.Pix[i+1] > 235 && img.Pix[i+2] > 240 {
							light++
						}
					}
					if light > len(img.Pix)/4/2 {
						break
					}
					if time.Now().After(end) {
						t.Fatalf("the screen is not the app's page (%d light pixels of %d)", light, len(img.Pix)/4)
					}
				}
				if dir := os.Getenv("GLUTTER_PROOF_DIR"); dir != "" {
					b, err := e.ADB.Screencap(ctx)
					if err != nil {
						t.Fatal(Describe(err))
					}
					name := "fonts-" + strings.TrimPrefix(t.Name()[strings.LastIndex(t.Name(), "/")+1:], "") + "-" + locale + ".png"
					if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			})
		})
	}
}
