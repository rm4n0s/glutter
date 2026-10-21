//go:build glutter_browser

package web_test

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// These tests run the web platform in real browsers (SPEC §17 "Browser tests"). The app is the
// counter, which runs through the engine, this platform and the 2D canvas renderer; pages with
// other configs of the host element are written beside index.html in the same build.

var (
	buildOnce sync.Once
	siteDir   string
	buildErr  error
)

// pages are the extra pages of the build: name, and the page.
var pages = map[string]webtool.Page{
	"missing.html": {Title: "missing", HostElement: "#nope"},
	"badsel.html":  {Title: "bad selector", HostElement: "###"},
	"stage.html":   {Title: "stage", HostElement: "#stage", Body: `<p style="margin:8px">above</p><div id="stage" style="width:300px;height:200px"></div>`},
}

func site(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		if siteDir, buildErr = os.MkdirTemp("", "glutter-web-platform-"); buildErr != nil {
			return
		}
		if _, buildErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./examples/counter", Dir: webtest.ModuleRoot(t), OutDir: siteDir,
		}); buildErr != nil {
			return
		}
		for name, p := range pages {
			if !strings.HasSuffix(name, ".html") {
				continue
			}
			index, err := webtool.IndexHTML(p)
			if err != nil {
				buildErr = err
				return
			}
			if buildErr = os.WriteFile(filepath.Join(siteDir, name), index, 0o644); buildErr != nil {
				return
			}
		}
		// A page whose config has a host element of the wrong type.
		index, err := webtool.IndexHTML(webtool.Page{})
		if err != nil {
			buildErr = err
			return
		}
		bad := bytes.Replace(index, []byte("globalThis.glutterConfig = {};"), []byte("globalThis.glutterConfig = {hostElement: 42};"), 1)
		buildErr = os.WriteFile(filepath.Join(siteDir, "notstring.html"), bad, 0o644)
	})
	if buildErr != nil {
		t.Fatalf("building the counter for the web: %v", buildErr)
	}
	return siteDir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if siteDir != "" {
		_ = os.RemoveAll(siteDir) // a temporary directory: a leftover is harmless
	}
	cleanupInputSite()
	cleanupTextSite()
	os.Exit(code)
}

// shot opens the page of the build in the browser and returns the PNG the app posts, or the
// error of Shot.
func shot(t *testing.T, b webtest.Browser, page string, query url.Values) (width, height int, err error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	data, err := webtest.Shot(ctx, webtest.ShotOptions{Dir: site(t), Browser: b, Page: "/" + page, Query: query, Width: 480, Height: 640})
	if err != nil {
		return 0, 0, err
	}
	img, derr := png.DecodeConfig(bytes.NewReader(data))
	if derr != nil {
		t.Fatalf("the screenshot is not a PNG: %v", derr)
	}
	return img.Width, img.Height, nil
}

// logOf is what the failed app wrote, from the details of the error of Shot.
func logOf(err error) string {
	appErr, ok := errors.FromError(err)
	if !ok {
		return err.Error()
	}
	return fmt.Sprint(appErr.Metadata["log"], appErr.Metadata["message"])
}

// TestBrowserHostElement: a host element of the page is the view's size; one that matches nothing
// is HostElementNotFound, one that is not a valid selector too, and a config of the wrong type is
// WebConfigInvalid. The failures reach the test through the loader's report of how the program ended.
func TestBrowserHostElement(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		t.Run("fills the page", func(t *testing.T) {
			w, h, err := shot(t, b, "index.html", nil)
			if err != nil || w != 480 || h != 640 {
				t.Errorf("got %dx%d, %v; want 480x640", w, h, err)
			}
		})
		t.Run("fills the host element", func(t *testing.T) {
			w, h, err := shot(t, b, "stage.html", nil)
			if err != nil || w != 300 || h != 200 {
				t.Errorf("got %dx%d, %v; want 300x200, the size of #stage", w, h, err)
			}
		})
		for _, tc := range []struct{ page, want string }{
			{"missing.html", "glutterConfig.hostElement matches no element of the page"},
			{"badsel.html", "glutterConfig.hostElement matches no element of the page"},
			{"notstring.html", "glutterConfig.hostElement must be a CSS selector string"},
		} {
			t.Run(tc.page, func(t *testing.T) {
				_, _, err := shot(t, b, tc.page, nil)
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != "WebAppFailed" {
					t.Fatalf("got %v, want WebAppFailed", err)
				}
				if log := logOf(err); !strings.Contains(log, tc.want) {
					t.Errorf("the app wrote %q, want it to say %q", log, tc.want)
				}
			})
		}
	})
}

// TestBrowserRendererNames: GLUTTER_RENDERER reaches the program through the query string, and the
// web has only soft for now (SPEC §9.6): the other names are RendererUnavailable, and a name
// that is no renderer is RendererUnknown.
func TestBrowserRendererNames(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		for _, name := range []string{"soft", "auto"} {
			t.Run(name, func(t *testing.T) {
				w, h, err := shot(t, b, "index.html", url.Values{"GLUTTER_RENDERER": {name}})
				if err != nil || w != 480 || h != 640 {
					t.Errorf("got %dx%d, %v; want a frame of 480x640", w, h, err)
				}
			})
		}
		for _, tc := range []struct{ name, want string }{
			{"webgl", "this OS or build does not have that renderer"},
			{"webgpu", "this OS or build does not have that renderer"},
			{"vulkan", "this OS or build does not have that renderer"},
			{"bogus", "the renderer is not auto, vulkan, gles, webgpu, webgl or soft"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := shot(t, b, "index.html", url.Values{"GLUTTER_RENDERER": {tc.name}})
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != "WebAppFailed" {
					t.Fatalf("got %v, want WebAppFailed", err)
				}
				if log := logOf(err); !strings.Contains(log, tc.want) {
					t.Errorf("the app wrote %q, want it to say %q", log, tc.want)
				}
			})
		}
	})
}
