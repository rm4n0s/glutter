package webtool

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// contentTypes are the types of the files of a web build. Everything else is served as
// application/octet-stream.
var contentTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".js":   "text/javascript; charset=utf-8",
	".wasm": "application/wasm",
	".json": "application/json",
	".css":  "text/css; charset=utf-8",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".txt":  "text/plain; charset=utf-8",
}

// Handler serves the directory of a web build (SPEC §23.1): index.html for "/", every file
// with its type (app.wasm as application/wasm, which instantiateStreaming needs), and, to a
// browser that accepts gzip, app.wasm.gz in place of app.wasm with Content-Encoding gzip.
// Nothing outside the directory is reachable, and nothing is cached by the browser, so a new
// build is the one that loads.
//
// Flutter: no counterpart
func Handler(dir string) http.Handler { return handler(dir, "no-store") }

// CachingHandler is Handler for a build whose files the browser may keep for a while, as a
// site that is deployed would let it: the measurement of the start with app.wasm in the
// browser's cache (SPEC §15) needs it. A new build is not seen until the time is up.
//
// Flutter: no counterpart
func CachingHandler(dir string) http.Handler { return handler(dir, "max-age=600") }

// handler serves dir with the given Cache-Control.
func handler(dir, cacheControl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := path.Clean("/" + r.URL.Path)
		if name == "/" {
			name = "/index.html"
		}
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(name, "/")))
		encoding := ""
		if name == "/app.wasm" && acceptsGzip(r) {
			if _, err := os.Stat(file + ".gz"); err == nil {
				file, encoding = file+".gz", "gzip"
			}
		}
		f, err := os.Open(file)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }() // read-only: a failed close loses nothing
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		typ := contentTypes[strings.ToLower(path.Ext(name))]
		if typ == "" {
			typ = "application/octet-stream"
		}
		w.Header().Set("Content-Type", typ)
		w.Header().Set("Cache-Control", cacheControl)
		w.Header().Add("Vary", "Accept-Encoding")
		if encoding != "" {
			w.Header().Set("Content-Encoding", encoding)
		}
		http.ServeContent(w, r, name, info.ModTime(), f)
	})
}

// acceptsGzip says whether the request's Accept-Encoding lists gzip with a quality above 0.
func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			continue
		}
		if q, ok := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q="); ok {
			v, err := strconv.ParseFloat(q, 64)
			return err == nil && v > 0
		}
		return true
	}
	return false
}
