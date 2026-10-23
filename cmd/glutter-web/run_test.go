package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// hello is a program that builds fast: glutter-web's tests build it instead of an example.
const hello = "../../internal/webtool/testdata/hello"

// syncBuffer is a bytes.Buffer that the test and the command can use at the same time.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		fn   string
	}{
		{"no command", nil, "run"},
		{"unknown command", []string{"deploy"}, "run"},
		{"build without a package", []string{"build"}, "parse"},
		{"build with two packages", []string{"build", "a", "b"}, "parse"},
		{"build with an unknown flag", []string{"build", "-nope", "a"}, "parse"},
		{"serve without a package", []string{"serve"}, "parse"},
		{"shot without a package", []string{"shot", "-browser", "firefox", "-o", "x.png"}, "parse"},
		{"shot without a browser", []string{"shot", "-o", "x.png", "./p"}, "runShot"},
		{"shot without an output", []string{"shot", "-browser", "firefox", "./p"}, "runShot"},
		{"perf without a browser", []string{"perf", "./p"}, "runPerf"},
		{"perf without a package", []string{"perf", "-browser", "firefox"}, "runPerf"},
		{"perf blank with a package", []string{"perf", "-browser", "firefox", "-blank", "./p"}, "runPerf"},
		{"perf with a bad click", []string{"perf", "-browser", "firefox", "-blank", "-clicks", "12"}, "parseClicks"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := run(context.Background(), tc.args, &out, &errOut)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagWebUsage || !appErr.HasRoute(tc.fn+".WebUsage") && !appErr.HasRoute(tc.fn+"->") {
				t.Errorf("got %s via %s, want WebUsage via %s", appErr.Tag, appErr.Route(), tc.fn)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var out bytes.Buffer
		if err := run(context.Background(), []string{arg}, &out, io.Discard); err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		for _, want := range []string{"glutter-web build", "glutter-web serve", "glutter-web shot"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: the help lacks %q", arg, want)
			}
		}
	}
}

func TestShotRejectsAnUnknownBrowserBeforeBuilding(t *testing.T) {
	err := run(context.Background(), []string{"shot", "-browser", "safari", "-o", filepath.Join(t.TempDir(), "x.png"), hello}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "BrowserUnknown" || !appErr.HasRoute("runShot->ParseBrowser.BrowserUnknown") {
		t.Fatalf("got %v, want BrowserUnknown via runShot->ParseBrowser", err)
	}
}

func TestQueryFlag(t *testing.T) {
	var q queryFlag
	for _, v := range []string{"GLUTTER_DEV_TAPS=3&GLUTTER_A=x%20y", "?GLUTTER_B=1", "GLUTTER_B=2"} {
		if err := q.Set(v); err != nil {
			t.Fatalf("Set(%q): %v", v, err)
		}
	}
	got := url.Values(q)
	if got.Get("GLUTTER_DEV_TAPS") != "3" || got.Get("GLUTTER_A") != "x y" || strings.Join(got["GLUTTER_B"], ",") != "1,2" {
		t.Errorf("query = %v", got)
	}
	err := q.Set("a=%zz")
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagWebUsage || !appErr.HasRoute("queryFlag.Set.WebUsage") {
		t.Errorf("a bad query gave %v, want WebUsage via queryFlag.Set", err)
	}
}

func TestBuildCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	out := filepath.Join(t.TempDir(), "site")
	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"build", "-o", out, "-host", "#app", hello}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "wasm_exec.js", "app.wasm", "app.wasm.gz"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("no %s: %v", name, err)
		}
	}
	if !regexp.MustCompile(`built .* into .*app\.wasm [0-9.]+ MB, app\.wasm\.gz [0-9.]+ MB`).MatchString(stdout.String()) {
		t.Errorf("the command printed %q", stdout.String())
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil || !strings.Contains(string(index), `"hostElement":"#app"`) {
		t.Errorf("index.html lacks the host element (err %v)", err)
	}
}

func TestBuildCommandReportsTheBuildError(t *testing.T) {
	err := run(context.Background(), []string{"build", "-o", t.TempDir(), "./nothing-here"}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WebBuildFailed" || !appErr.HasRoute("runBuild->Build->runGo.WebBuildFailed") {
		t.Fatalf("got %v, want WebBuildFailed via runBuild->Build->runGo", err)
	}
}

func TestServeCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdout := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"serve", "-addr", "127.0.0.1:0", hello}, stdout, io.Discard) }()
	var base string
	deadline := time.Now().Add(60 * time.Second)
	for base == "" && time.Now().Before(deadline) {
		if m := regexp.MustCompile(`at (http://[^/]+/)`).FindStringSubmatch(stdout.String()); m != nil {
			base = m[1]
		}
		select {
		case err := <-done:
			t.Fatalf("serve ended early: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if base == "" {
		t.Fatalf("serve printed no address: %q", stdout.String())
	}
	get := func(path string, accept string) (*http.Response, []byte) {
		req, err := http.NewRequest("GET", base+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		// A transport that does not decode, to see what the server sent.
		res, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res, body
	}
	res, body := get("/", "")
	if res.StatusCode != 200 || !strings.Contains(string(body), "glutterConfig") {
		t.Errorf("GET / = %d %q", res.StatusCode, body)
	}
	plain, plainBody := get("/app.wasm", "")
	zipped, zippedBody := get("/app.wasm", "gzip")
	if plain.Header.Get("Content-Type") != "application/wasm" || plain.Header.Get("Content-Encoding") != "" {
		t.Errorf("plain app.wasm: type %q encoding %q", plain.Header.Get("Content-Type"), plain.Header.Get("Content-Encoding"))
	}
	if zipped.Header.Get("Content-Encoding") != "gzip" {
		t.Errorf("app.wasm for a browser that accepts gzip has the encoding %q", zipped.Header.Get("Content-Encoding"))
	}
	zr, err := gzip.NewReader(bytes.NewReader(zippedBody))
	if err != nil {
		t.Fatalf("the gzip body is not gzip: %v", err)
	}
	unzipped, err := io.ReadAll(zr)
	if err != nil || !bytes.Equal(unzipped, plainBody) {
		t.Errorf("the gzip body does not hold the plain one (err %v)", err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve ended with %v after the interrupt", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop")
	}
}

func TestServeCommandListenFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	err := run(context.Background(), []string{"serve", "-addr", "256.0.0.1:1", hello}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagWebServeFailed || !appErr.HasRoute("runServe.WebServeFailed") {
		t.Fatalf("got %v, want WebServeFailed via runServe", err)
	}
}

func TestShotCommandWithoutABrowser(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	t.Setenv("PATH", filepath.Dir(mustGo(t))) // go is found; no browser is
	err := run(context.Background(), []string{"shot", "-browser", "firefox", "-o", filepath.Join(t.TempDir(), "x.png"), hello}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "BrowserNotFound" {
		t.Fatalf("got %v, want BrowserNotFound", err)
	}
}

// mustGo finds the go command.
func mustGo(t *testing.T) string {
	t.Helper()
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if p := filepath.Join(dir, "go"); fileExists(p) {
			return p
		}
	}
	t.Skip("go is not on PATH")
	return ""
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func TestParseClicks(t *testing.T) {
	tests := []struct {
		in   string
		want [][2]float64
		bad  bool
	}{
		{"", nil, false},
		{"100,218", [][2]float64{{100, 218}}, false},
		{" 100 , 218 ; 5.5,6 ;", [][2]float64{{100, 218}, {5.5, 6}}, false},
		{"100", nil, true},
		{"a,b", nil, true},
		{"1,2;3", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseClicks(tc.in)
			if tc.bad {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tagWebUsage || !appErr.HasRoute("parseClicks.WebUsage") {
					t.Fatalf("parseClicks(%q) = %v, want WebUsage via parseClicks", tc.in, err)
				}
				return
			}
			if err != nil || len(got) != len(tc.want) {
				t.Fatalf("parseClicks(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("click %d = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
