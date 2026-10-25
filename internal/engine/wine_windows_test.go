//go:build windows && glutter_wine

package engine

import (
	"context"
	"image/color"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/winfonts"
	"github.com/rm4n0s/glutter/ui"
)

// TestRunDrawsIntoAWindowAndSavesTheFrame runs the engine on the Win32 platform under Wine
// (SPEC §25.13): the first window opens, the soft renderer draws the scene into it with GDI, and
// the self-capture saves the frame as a PNG and ends Run. Run it with
// GOOS=windows go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 ./internal/engine
func TestRunDrawsIntoAWindowAndSavesTheFrame(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	shot := filepath.Join(t.TempDir(), "frame.png")
	env := map[string]string{ScreenshotEnv: shot}
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 30), gputest.FillPaint(0xFF3366CC))
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := Run(ctx, Config{
		Logger: quiet(), Getenv: func(k string) string { return env[k] },
		Window: platform.WindowConfig{Title: "Engine", Size: ui.Size{Width: 200, Height: 100}},
		Scene:  func(ui.Size, float64) *ui.Scene { return scene },
	})
	if err != nil {
		t.Fatal(err)
	}
	img, err := goldens.ReadPNG(shot)
	if err != nil {
		t.Fatal(err)
	}
	scale := float64(img.Bounds().Dx()) / 200
	if scale < 1 || img.Bounds().Dy() != int(100*scale) {
		t.Fatalf("the frame is %v, want 200x100 logical pixels", img.Bounds().Size())
	}
	at := func(x, y float64) color.RGBA { return img.RGBAAt(int(x*scale), int(y*scale)) }
	if got := at(30, 25); got != (color.RGBA{0x33, 0x66, 0xCC, 255}) {
		t.Errorf("a pixel of the rectangle = %v, want #3366CC", got)
	}
	if got := at(150, 80); got.A != 255 {
		t.Errorf("a pixel outside = %v, want an opaque one", got)
	}
}

// TestSystemFontMatcherIsTheMatcherOfWindows checks that the engine gives the Collection the
// matcher of Windows (SPEC §25.8), here over the fonts of the prefix of Wine, and that a font it
// finds loads.
func TestSystemFontMatcherIsTheMatcherOfWindows(t *testing.T) {
	t.Setenv("LocalAppData", t.TempDir())
	ctx := context.Background()
	m := SystemFontMatcher(ctx, quiet())
	if _, ok := m.(*winfonts.Matcher); !ok {
		t.Fatalf("the matcher is %T, want the matcher of winfonts", m)
	}
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	files, err := c.Match(ctx, "Liberation Sans", 400, false, "")
	if err != nil || len(files) == 0 {
		t.Fatalf("Match(Liberation Sans) = %+v, %v", files, err)
	}
	face, err := c.Face(ctx, files[0])
	if err != nil || face == nil {
		t.Fatalf("Face = %v, %v", face, err)
	}
	families, err := c.Families(ctx)
	if err != nil || len(families) == 0 {
		t.Errorf("Families = %v, %v", families, err)
	}
}
