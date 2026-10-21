package webtool

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// buildDir makes a directory like the one Build writes, with a fake wasm file.
func buildDir(t *testing.T) (dir string, wasm []byte) {
	t.Helper()
	dir = t.TempDir()
	wasm = append([]byte("\x00asm\x01\x00\x00\x00"), bytes.Repeat([]byte("wasm!"), 100)...)
	var zipped bytes.Buffer
	zw := gzip.NewWriter(&zipped)
	if _, err := zw.Write(wasm); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"index.html": []byte("<!doctype html>index"), "wasm_exec.js": []byte("// exec"),
		"app.wasm": wasm, "app.wasm.gz": zipped.Bytes(), "notes.xyz": []byte("x"),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, wasm
}

func TestHandler(t *testing.T) {
	dir, wasm := buildDir(t)
	h := Handler(dir)
	tests := []struct {
		name     string
		method   string
		path     string
		accept   string
		status   int
		typ      string
		encoding string
		body     []byte // nil: not checked
		gunzip   bool
	}{
		{"root is index.html", "GET", "/", "", 200, "text/html; charset=utf-8", "", []byte("<!doctype html>index"), false},
		{"index", "GET", "/index.html", "", 200, "text/html; charset=utf-8", "", nil, false},
		{"loader script", "GET", "/wasm_exec.js", "", 200, "text/javascript; charset=utf-8", "", []byte("// exec"), false},
		{"wasm without gzip", "GET", "/app.wasm", "", 200, "application/wasm", "", wasm, false},
		{"wasm with gzip", "GET", "/app.wasm", "gzip, deflate, br", 200, "application/wasm", "gzip", wasm, true},
		{"wasm with gzip among others", "GET", "/app.wasm", "br, gzip;q=0.8", 200, "application/wasm", "gzip", wasm, true},
		{"wasm with gzip refused", "GET", "/app.wasm", "gzip;q=0", 200, "application/wasm", "", wasm, false},
		{"wasm with only brotli", "GET", "/app.wasm", "br", 200, "application/wasm", "", wasm, false},
		{"head", "HEAD", "/app.wasm", "", 200, "application/wasm", "", nil, false},
		{"unknown type", "GET", "/notes.xyz", "", 200, "application/octet-stream", "", nil, false},
		{"the gzip file itself is not content-encoded", "GET", "/app.wasm.gz", "gzip", 200, "application/octet-stream", "", nil, false},
		{"missing", "GET", "/nothing", "", 404, "", "", nil, false},
		{"a directory", "GET", "/sub", "", 404, "", "", nil, false},
		{"outside the directory", "GET", "/../../etc/passwd", "", 404, "", "", nil, false},
		{"dots are cleaned", "GET", "/a/../index.html", "", 200, "text/html; charset=utf-8", "", nil, false},
		{"post", "POST", "/index.html", "", 405, "", "", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.accept != "" {
				req.Header.Set("Accept-Encoding", tc.accept)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			res := rec.Result()
			defer func() { _ = res.Body.Close() }()
			if res.StatusCode != tc.status {
				t.Fatalf("status %d, want %d", res.StatusCode, tc.status)
			}
			if tc.status != 200 {
				return
			}
			if got := res.Header.Get("Content-Type"); got != tc.typ {
				t.Errorf("Content-Type %q, want %q", got, tc.typ)
			}
			if got := res.Header.Get("Content-Encoding"); got != tc.encoding {
				t.Errorf("Content-Encoding %q, want %q", got, tc.encoding)
			}
			if got := res.Header.Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control %q, want no-store", got)
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if tc.gunzip {
				zr, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Fatalf("the body is not gzip: %v", err)
				}
				if body, err = io.ReadAll(zr); err != nil {
					t.Fatal(err)
				}
			}
			if tc.body != nil && !bytes.Equal(body, tc.body) {
				t.Errorf("body %q, want %q", body, tc.body)
			}
		})
	}
}

func TestHandlerWithoutAGzipFileSendsThePlainOne(t *testing.T) {
	dir, wasm := buildDir(t)
	if err := os.Remove(filepath.Join(dir, "app.wasm.gz")); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/app.wasm", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	Handler(dir).ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "" || !bytes.Equal(rec.Body.Bytes(), wasm) {
		t.Errorf("got encoding %q and %d bytes, want the plain file", rec.Header().Get("Content-Encoding"), rec.Body.Len())
	}
}

func TestHandlerOverHTTP(t *testing.T) {
	dir, wasm := buildDir(t)
	srv := httptest.NewServer(Handler(dir))
	defer srv.Close()
	// The Go client asks for gzip itself and decodes it, so the body is the wasm either way.
	res, err := http.Get(srv.URL + "/app.wasm")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, wasm) || res.Header.Get("Content-Type") != "application/wasm" {
		t.Errorf("got %d bytes of %q, want the wasm file", len(body), res.Header.Get("Content-Type"))
	}
}
