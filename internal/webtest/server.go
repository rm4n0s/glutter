package webtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// maxPost is the largest body the server takes: a frame as a PNG is far smaller.
const maxPost = 64 << 20

// pngMagic starts every PNG file.
var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// Result is what a page posts to ResultsPath: the loader's report of how the program ended
// (Kind "exit": the exit code and what it wrote to its standard output and error), or what a
// test program of its own sends.
//
// Flutter: no counterpart
type Result struct {
	// Kind says what the result is: "exit" from the loader.
	Kind string `json:"kind"`
	// ExitCode is the exit code of the Go program, for Kind "exit".
	ExitCode int `json:"exitCode"`
	// Message is a message of the page: the error of a program that did not start.
	Message string `json:"message"`
	// Log is what the Go program wrote to its standard output and error.
	Log string `json:"log"`
	// Data is the rest of what a test program sent, as it sent it.
	Data json.RawMessage `json:"data,omitempty"`
}

// Server serves a web build over HTTP for a browser, and keeps what the page posts: the
// PNG of the engine's self-capture (web.ScreenshotPath) and results (web.ResultsPath). It
// is an httptest.Server: Close stops it.
//
// Flutter: no counterpart
type Server struct {
	*httptest.Server
	mux     *http.ServeMux
	shots   chan []byte
	results chan Result

	mu    sync.Mutex
	pages map[string][]byte
}

// NewServer starts a server for the build in dir (the directory webtool.Build wrote). Close
// it when done.
//
// Flutter: no counterpart
func NewServer(dir string) *Server {
	s := &Server{
		mux:     http.NewServeMux(),
		shots:   make(chan []byte, 16),
		results: make(chan Result, 16),
		pages:   map[string][]byte{},
	}
	s.mux.HandleFunc(web.ScreenshotPath, s.takeScreenshot)
	s.mux.HandleFunc(web.ResultsPath, s.takeResult)
	s.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		page, ok := s.pages[r.URL.Path]
		s.mu.Unlock()
		if ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(page) // the browser went away: nothing to report
			return
		}
		webtool.Handler(dir).ServeHTTP(w, r)
	}))
	s.Server = httptest.NewServer(s.mux)
	return s
}

// AddPage serves html at the path (such as "/resize.html"), beside the files of the build:
// a page of a test that embeds the app.
//
// Flutter: no counterpart
func (s *Server) AddPage(path string, html []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pages[path] = html
}

// takeScreenshot is the handler of ScreenshotPath: it keeps the PNG that is posted.
func (s *Server) takeScreenshot(w http.ResponseWriter, r *http.Request) {
	body, ok := readPost(w, r)
	if !ok {
		return
	}
	if !bytes.HasPrefix(body, pngMagic) {
		http.Error(w, "the body is not a PNG", http.StatusBadRequest)
		return
	}
	select {
	case s.shots <- body:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "too many screenshots", http.StatusTooManyRequests)
	}
}

// takeResult is the handler of ResultsPath: it keeps the result that is posted.
func (s *Server) takeResult(w http.ResponseWriter, r *http.Request) {
	body, ok := readPost(w, r)
	if !ok {
		return
	}
	var res Result
	if err := json.Unmarshal(body, &res); err != nil {
		http.Error(w, "the body is not a result: "+err.Error(), http.StatusBadRequest)
		return
	}
	select {
	case s.results <- res:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "too many results", http.StatusTooManyRequests)
	}
}

// readPost reads the body of a POST, answering the request itself when it cannot.
func readPost(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return nil, false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPost))
	if err != nil {
		http.Error(w, "the body could not be read: "+err.Error(), http.StatusBadRequest)
		return nil, false
	}
	return body, true
}

// Screenshots is the channel of the PNGs the page posted, in order.
//
// Flutter: no counterpart
func (s *Server) Screenshots() <-chan []byte { return s.shots }

// Results is the channel of the results the page posted, in order.
//
// Flutter: no counterpart
func (s *Server) Results() <-chan Result { return s.results }

// WaitScreenshot waits for the next PNG the page posts. It fails with BrowserTimeout when ctx
// ends first.
//
// Flutter: no counterpart
func (s *Server) WaitScreenshot(ctx context.Context) ([]byte, error) {
	select {
	case png := <-s.shots:
		return png, nil
	case <-ctx.Done():
		return nil, errors.NewErr("BrowserTimeout", ctx.Err()).SetMetadata("waiting for", "a screenshot")
	}
}

// WaitResult waits for the next result the page posts. It fails with BrowserTimeout when ctx
// ends first.
//
// Flutter: no counterpart
func (s *Server) WaitResult(ctx context.Context) (Result, error) {
	select {
	case res := <-s.results:
		return res, nil
	case <-ctx.Done():
		return Result{}, errors.NewErr("BrowserTimeout", ctx.Err()).SetMetadata("waiting for", "a result")
	}
}
