package webtest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
)

// tinyPNG is a valid PNG header followed by anything: the server checks only the magic.
var tinyPNG = append(append([]byte{}, pngMagic...), []byte("rest of the file")...)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("the app"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(dir)
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, url, contentType string, body []byte) *http.Response {
	t.Helper()
	res, err := http.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func TestServerServesTheBuildAndAddedPages(t *testing.T) {
	srv := newTestServer(t)
	srv.AddPage("/extra.html", []byte("<p>extra</p>"))
	srv.AddPage("/index.html", []byte("<p>replaced</p>"))
	tests := []struct{ path, want string }{
		{"/", "the app"}, // not replaced: only "/index.html" is a page of its own
		{"/extra.html", "<p>extra</p>"},
		{"/index.html", "<p>replaced</p>"},
	}
	for _, tc := range tests {
		res, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil || string(body) != tc.want {
			t.Errorf("GET %s = %q (err %v), want %q", tc.path, body, err, tc.want)
		}
	}
	res, err := http.Get(srv.URL + "/extra.html")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if got := res.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("an added page has the type %q", got)
	}
}

func TestServerTakesScreenshots(t *testing.T) {
	srv := newTestServer(t)
	if res := post(t, srv.URL+web.ScreenshotPath, "image/png", tinyPNG); res.StatusCode != http.StatusNoContent {
		t.Fatalf("a PNG got status %d, want 204", res.StatusCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := srv.WaitScreenshot(ctx)
	if err != nil || !bytes.Equal(got, tinyPNG) {
		t.Fatalf("WaitScreenshot = %q, %v; want the posted PNG", got, err)
	}
	tests := []struct {
		name   string
		method string
		body   []byte
		status int
	}{
		{"not a PNG", http.MethodPost, []byte("GIF89a"), http.StatusBadRequest},
		{"empty", http.MethodPost, nil, http.StatusBadRequest},
		{"get", http.MethodGet, nil, http.StatusMethodNotAllowed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, srv.URL+web.ScreenshotPath, bytes.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = res.Body.Close()
			if res.StatusCode != tc.status {
				t.Errorf("status %d, want %d", res.StatusCode, tc.status)
			}
		})
	}
	select {
	case <-srv.Screenshots():
		t.Error("a request that was refused left a screenshot")
	default:
	}
}

func TestServerTakesResults(t *testing.T) {
	srv := newTestServer(t)
	body := `{"kind":"exit","exitCode":2,"message":"m","log":"line\n","data":{"width":480}}`
	if res := post(t, srv.URL+web.ResultsPath, "application/json", []byte(body)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("a result got status %d, want 204", res.StatusCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := srv.WaitResult(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "exit" || got.ExitCode != 2 || got.Message != "m" || got.Log != "line\n" || string(got.Data) != `{"width":480}` {
		t.Errorf("result %+v, want the posted one", got)
	}
	if res := post(t, srv.URL+web.ResultsPath, "application/json", []byte("{not json")); res.StatusCode != http.StatusBadRequest {
		t.Errorf("bad JSON got status %d, want 400", res.StatusCode)
	}
}

func TestServerKeepsOnlyAFewPosts(t *testing.T) {
	srv := newTestServer(t)
	var last int
	for range 40 {
		last = post(t, srv.URL+web.ScreenshotPath, "image/png", tinyPNG).StatusCode
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("the 40th screenshot got status %d, want 429: the server must not grow without limit", last)
	}
}

func TestWaitsTimeOutWithTheRoute(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, shotErr := srv.WaitScreenshot(ctx)
	_, resErr := srv.WaitResult(ctx)
	for name, err := range map[string]error{"WaitScreenshot": shotErr, "WaitResult": resErr} {
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("%s: expected *errors.Error, got %T", name, err)
		}
		if appErr.Tag != tagBrowserTimeout || !appErr.HasRoute("Server."+name+".BrowserTimeout") {
			t.Errorf("%s: got %s via %s", name, appErr.Tag, appErr.Route())
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("%s: the cause is not the context's deadline", name)
		}
	}
}

func TestFramePage(t *testing.T) {
	page := string(FramePage("/index.html?GLUTTER_A=1&x=<y>", 480, 640, []Resize{{After: 700 * time.Millisecond, Width: 360, Height: 500}}))
	for _, want := range []string{"width:480px;height:640px", `f.src="/index.html?GLUTTER_A=1\u0026x=\u003cy\u003e";`,
		`f.style.width="360px";f.style.height="500px";},700);});`, "whenCanvas", "margin:0"} {
		if !strings.Contains(page, want) {
			t.Errorf("the frame page lacks %q:\n%s", want, page)
		}
	}
	if plain := string(FramePage("/a", 10, 20, nil)); strings.Contains(plain, "setTimeout") {
		t.Error("a frame page without resizes has timers")
	}
}
