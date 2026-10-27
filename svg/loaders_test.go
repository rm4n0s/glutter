package svg

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// countingBundle counts the loads of an asset bundle.
type countingBundle struct {
	services.AssetBundle
	mu    sync.Mutex
	loads map[string]int
}

func newCountingBundle(files map[string]string) *countingBundle {
	fs := fstest.MapFS{}
	for name, data := range files {
		fs[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return &countingBundle{AssetBundle: services.NewFSAssetBundle(fs), loads: map[string]int{}}
}

func (b *countingBundle) Load(ctx context.Context, key string) ([]byte, error) {
	b.mu.Lock()
	b.loads[key]++
	b.mu.Unlock()
	return b.AssetBundle.Load(ctx, key)
}

func (b *countingBundle) count(key string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.loads[key]
}

func gzipped(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// showsRed pumps a widget, waits for its document, and checks that the first pixel is red.
func showsRed(t *testing.T, w widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	pumpTopLeft(ts, w)
	waitFor(t, ts, loaded(ts))
	ts.Pump()
	if got := pixel(ts, 5, 5); got != redC {
		t.Errorf("pixel = %v, want red", got)
	}
	return ts
}

func TestSvgPictureLoaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "red.svg")
	if err := os.WriteFile(path, []byte(red20x10), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	var gotHeader atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		gotHeader.Store(r.Header.Get("X-Token"))
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(red20x10))
	}))
	t.Cleanup(srv.Close)
	bundle := newCountingBundle(map[string]string{"icons/red.svg": red20x10, "packages/pkg/red.svg": red20x10})

	tests := []struct {
		name string
		make func() *SvgPicture
		// check runs after the picture is shown.
		check func(t *testing.T)
	}{
		{"asset", func() *SvgPicture {
			return NewSvgPictureAsset("icons/red.svg", SvgPictureAssetOpts{Bundle: bundle})
		}, func(t *testing.T) {
			if bundle.count("icons/red.svg") != 1 {
				t.Errorf("the asset was loaded %d times", bundle.count("icons/red.svg"))
			}
		}},
		{"asset of a package", func() *SvgPicture {
			return NewSvgPictureAsset("red.svg", SvgPictureAssetOpts{Bundle: bundle, Package: "pkg"})
		}, func(t *testing.T) {
			if bundle.count("packages/pkg/red.svg") != 1 {
				t.Errorf("the asset was loaded %d times", bundle.count("packages/pkg/red.svg"))
			}
		}},
		{"string", func() *SvgPicture { return NewSvgPictureString(red20x10) }, nil},
		{"memory", func() *SvgPicture { return NewSvgPictureMemory([]byte(red20x10)) }, nil},
		{"memory, compressed", func() *SvgPicture { return NewSvgPictureMemory(gzipped(t, red20x10)) }, nil},
		{"file", func() *SvgPicture { return NewSvgPictureFile(path) }, nil},
		{"network", func() *SvgPicture {
			return NewSvgPictureNetwork(srv.URL+"/red.svg", SvgPictureNetworkOpts{Headers: map[string]string{"X-Token": "abc"}})
		}, func(t *testing.T) {
			if requests.Load() != 1 {
				t.Errorf("%d requests", requests.Load())
			}
			if got, _ := gotHeader.Load().(string); got != "abc" {
				t.Errorf("header X-Token = %q", got)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showsRed(t, tt.make())
			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

func TestSvgPictureAssetUsesTheBundleOfTheBinding(t *testing.T) {
	ts := newTester(t, gluttertest.Opts{Size: ui.Size{Width: 100, Height: 100}, Assets: fstest.MapFS{
		"icons/red.svg":              {Data: []byte(red20x10)},
		"packages/pkg/icons/red.svg": {Data: []byte(red20x10)},
	}})
	for _, w := range []*SvgPicture{
		NewSvgPictureAsset("icons/red.svg"),
		NewSvgPictureAsset("icons/red.svg", SvgPictureAssetOpts{Package: "pkg"}),
	} {
		pumpTopLeft(ts, w)
		waitFor(t, ts, loaded(ts))
		ts.Pump()
		if got := pixel(ts, 5, 5); got != redC {
			t.Errorf("%v: pixel = %v, want red", w, got)
		}
	}
}

func TestSvgPictureImmediateSourcesAreDrawnInTheFirstFrame(t *testing.T) {
	tests := []struct {
		name string
		make func() *SvgPicture
	}{
		{"string", func() *SvgPicture { return NewSvgPictureString(red20x10) }},
		{"memory", func() *SvgPicture { return NewSvgPictureMemory([]byte(red20x10)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, tt.make())
			if got := pixel(ts, 5, 5); got != redC {
				t.Errorf("pixel = %v, want red in the first frame", got)
			}
		})
	}
}

func TestSvgPictureFollowsItsSource(t *testing.T) {
	const blue = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="20" height="10" fill="#00f"/></svg>`
	bundle := newCountingBundle(map[string]string{"red.svg": red20x10, "blue.svg": blue})
	tests := []struct {
		name         string
		first, again func() *SvgPicture
	}{
		{"string", func() *SvgPicture { return NewSvgPictureString(red20x10) }, func() *SvgPicture { return NewSvgPictureString(blue) }},
		{"memory", func() *SvgPicture { return NewSvgPictureMemory([]byte(red20x10)) }, func() *SvgPicture { return NewSvgPictureMemory([]byte(blue)) }},
		{"asset", func() *SvgPicture { return NewSvgPictureAsset("red.svg", SvgPictureAssetOpts{Bundle: bundle}) },
			func() *SvgPicture { return NewSvgPictureAsset("blue.svg", SvgPictureAssetOpts{Bundle: bundle}) }},
		{"kind", func() *SvgPicture { return NewSvgPictureString(red20x10) },
			func() *SvgPicture { return NewSvgPictureAsset("blue.svg", SvgPictureAssetOpts{Bundle: bundle}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, tt.first())
			waitFor(t, ts, loaded(ts))
			ts.Pump()
			if got := pixel(ts, 5, 5); got != redC {
				t.Fatalf("first pixel = %v", got)
			}
			pumpTopLeft(ts, tt.again())
			waitFor(t, ts, func() bool { ts.Pump(); return pixel(ts, 5, 5) == blueC })
		})
	}
}

func TestSvgPictureTheme(t *testing.T) {
	const current = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="20" height="10" fill="currentColor"/></svg>`
	green := ui.NewColorFromARGB(255, 0, 255, 0)
	blue := ui.NewColorFromARGB(255, 0, 0, 255)
	tests := []struct {
		name string
		make func(c widgets.Widget) widgets.Widget
		opts SvgPictureOpts
		want ui.Color
	}{
		{"black by default", func(c widgets.Widget) widgets.Widget { return c }, SvgPictureOpts{}, ui.NewColorFromARGB(255, 0, 0, 0)},
		{"the theme of the options", func(c widgets.Widget) widgets.Widget { return c }, SvgPictureOpts{Theme: some(SvgTheme{CurrentColor: green})}, green},
		{"the default theme above", func(c widgets.Widget) widgets.Widget { return NewDefaultSvgTheme(SvgTheme{CurrentColor: blue}, c) }, SvgPictureOpts{}, blue},
		{"the options beat the default theme", func(c widgets.Widget) widgets.Widget { return NewDefaultSvgTheme(SvgTheme{CurrentColor: blue}, c) },
			SvgPictureOpts{Theme: some(SvgTheme{CurrentColor: green})}, green},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, tt.make(NewSvgPictureString(current, tt.opts)))
			got := pixel(ts, 5, 5)
			if int(got.R) != tt.want.Red() || int(got.G) != tt.want.Green() || int(got.B) != tt.want.Blue() || got.A != 255 {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSvgPictureFollowsTheDefaultTheme(t *testing.T) {
	const current = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="20" height="10" fill="currentColor"/></svg>`
	ts := newTester(t)
	show := func(c ui.Color) {
		pumpTopLeft(ts, NewDefaultSvgTheme(SvgTheme{CurrentColor: c}, NewSvgPictureString(current)))
	}
	show(ui.NewColorFromARGB(255, 255, 0, 0))
	if got := pixel(ts, 5, 5); got != redC {
		t.Fatalf("pixel = %v", got)
	}
	show(ui.NewColorFromARGB(255, 0, 0, 255))
	if got := pixel(ts, 5, 5); got != blueC {
		t.Errorf("pixel after the theme changed = %v, want blue", got)
	}
	// One entry for each theme.
	c := CacheOf(gluttertest.StateOf[*svgPictureState](ts, gluttertest.ByType[*SvgPicture]()).Context())
	if c.Count() != 2 {
		t.Errorf("the cache holds %d documents, want 2 (one for each theme)", c.Count())
	}
}

func TestSvgPicturePlaceholderAndError(t *testing.T) {
	missing := newCountingBundle(nil)
	placeholder := func(widgets.BuildContext) widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: some(7.0), Height: some(9.0)})
	}
	t.Run("the placeholder shows while the SVG loads", func(t *testing.T) {
		gate := newGateBundle(red20x10)
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate, SvgPictureOpts: SvgPictureOpts{PlaceholderBuilder: placeholder}}))
		if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != (ui.Size{Width: 7, Height: 9}) {
			t.Fatalf("size while loading = %v", got)
		}
		<-gate.started
		gate.release()
		waitFor(t, ts, loaded(ts))
		ts.Pump()
		if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != (ui.Size{Width: 20, Height: 10}) {
			t.Errorf("size after loading = %v", got)
		}
	})
	t.Run("the default placeholder has the size of the options", func(t *testing.T) {
		gate := newGateBundle(red20x10)
		defer gate.release()
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate, SvgPictureOpts: SvgPictureOpts{Width: some(30.0), Height: some(31.0)}}))
		if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != (ui.Size{Width: 30, Height: 31}) {
			t.Errorf("size while loading = %v", got)
		}
	})
	t.Run("the error builder gets the error, which is also reported", func(t *testing.T) {
		var seen error
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureAsset("nothing.svg", SvgPictureAssetOpts{Bundle: missing, SvgPictureOpts: SvgPictureOpts{
			ErrorBuilder: func(_ widgets.BuildContext, err error) widgets.Widget {
				seen = err
				return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: some(5.0), Height: some(6.0)})
			}}}))
		waitFor(t, ts, loaded(ts))
		ts.Pump()
		if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != (ui.Size{Width: 5, Height: 6}) {
			t.Errorf("size of the error widget = %v", got)
		}
		reported := ts.TakeErrors()
		if len(reported) != 1 {
			t.Fatalf("%d errors reported, want 1", len(reported))
		}
		appErr, ok := errors.FromError(reported[0])
		if !ok || appErr.Tag != TagSvgLoadFailed {
			t.Fatalf("reported = %v", reported[0])
		}
		if seen != reported[0] {
			t.Errorf("the builder got %v, the reporter %v", seen, reported[0])
		}
	})
	parseErrors := []struct {
		name string
		src  string
		tag  string
	}{
		{"a document that is not XML", `<svg`, TagSvgParseFailed},
		{"a document that is not svg", `<html xmlns="http://www.w3.org/1999/xhtml"/>`, TagSvgNotSvg},
		{"a viewBox with a negative size", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 -1 5"/>`, TagSvgInvalidValue},
	}
	for _, tt := range parseErrors {
		t.Run(tt.name+" shows the default error widget", func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, NewSvgPictureString(tt.src, SvgPictureOpts{Width: some(12.0), Height: some(13.0)}))
			if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != (ui.Size{Width: 12, Height: 13}) {
				t.Errorf("size = %v", got)
			}
			reported := ts.TakeErrors()
			if len(reported) != 1 {
				t.Fatalf("%d errors reported, want 1", len(reported))
			}
			if appErr, ok := errors.FromError(reported[0]); !ok || appErr.Tag != tt.tag {
				t.Errorf("reported = %v, want tag %s", reported[0], tt.tag)
			}
			// The error is reported once, not for every build.
			ts.Pump()
			pumpTopLeft(ts, NewSvgPictureString(tt.src, SvgPictureOpts{Width: some(12.0), Height: some(13.0)}))
			if n := len(ts.TakeErrors()); n != 0 {
				t.Errorf("%d more errors reported", n)
			}
		})
	}
}

// gateBundle is a bundle whose Load waits until it is released, or until its context ends.
type gateBundle struct {
	data     string
	started  chan struct{}
	released chan struct{}
	once     sync.Once
	loads    atomic.Int32
	canceled atomic.Int32
}

func newGateBundle(data string) *gateBundle {
	return &gateBundle{data: data, started: make(chan struct{}, 16), released: make(chan struct{})}
}

func (b *gateBundle) release() { b.once.Do(func() { close(b.released) }) }

func (b *gateBundle) Load(ctx context.Context, _ string) ([]byte, error) {
	b.loads.Add(1)
	b.started <- struct{}{}
	select {
	case <-b.released:
		return []byte(b.data), nil
	case <-ctx.Done():
		b.canceled.Add(1)
		return nil, ctx.Err()
	}
}

func (b *gateBundle) Variants(context.Context, string) ([]services.AssetVariant, error) {
	return nil, nil
}

func TestSvgPictureEndsItsLoadWhenItLeavesTheTree(t *testing.T) {
	t.Run("an asset", func(t *testing.T) {
		gate := newGateBundle(red20x10)
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate}))
		<-gate.started
		pumpTopLeft(ts, widgets.NewSizedBox(nil))
		waitFor(t, ts, func() bool { return gate.canceled.Load() == 1 })
	})
	t.Run("a request", func(t *testing.T) {
		started := make(chan struct{})
		ended := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done()
			close(ended)
		}))
		t.Cleanup(srv.Close)
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureNetwork(srv.URL))
		<-started
		pumpTopLeft(ts, widgets.NewSizedBox(nil))
		select {
		case <-ended:
		case <-time.After(5 * time.Second):
			t.Fatal("the request was not canceled")
		}
	})
	t.Run("an answer that comes later is dropped", func(t *testing.T) {
		gate := newGateBundle(red20x10)
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate}))
		<-gate.started
		pumpTopLeft(ts, widgets.NewSizedBox(nil))
		gate.release()
		for range 5 {
			ts.Pump()
		}
		if n := len(ts.TakeErrors()); n != 0 {
			t.Errorf("%d errors reported", n)
		}
	})
}

func TestNetworkSourceHeadersAndStatus(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+"|"+r.Header.Get("X-A"))
		mu.Unlock()
		if r.URL.Path == "/missing.svg" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(red20x10))
	}))
	t.Cleanup(srv.Close)

	t.Run("equal headers share a document, other headers do not", func(t *testing.T) {
		mu.Lock()
		seen = nil
		mu.Unlock()
		ts := newTester(t)
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
			NewSvgPictureNetwork(srv.URL+"/a.svg", SvgPictureNetworkOpts{Headers: map[string]string{"X-A": "1"}}),
			NewSvgPictureNetwork(srv.URL+"/a.svg", SvgPictureNetworkOpts{Headers: map[string]string{"X-A": "1"}}),
			NewSvgPictureNetwork(srv.URL+"/a.svg", SvgPictureNetworkOpts{Headers: map[string]string{"X-A": "2"}}),
		})))
		waitFor(t, ts, func() bool {
			for _, e := range ts.Elements(gluttertest.ByType[*SvgPicture]()) {
				s := e.(*widgets.StatefulElement).State().(*svgPictureState)
				if s.entry == nil {
					return false
				}
			}
			return true
		})
		mu.Lock()
		defer mu.Unlock()
		if len(seen) != 2 {
			t.Errorf("requests = %v, want two", seen)
		}
	})
	t.Run("a status other than 200 is an error", func(t *testing.T) {
		ts := newTester(t)
		pumpTopLeft(ts, NewSvgPictureNetwork(srv.URL+"/missing.svg"))
		waitFor(t, ts, loaded(ts))
		errs := ts.TakeErrors()
		if len(errs) != 1 {
			t.Fatalf("%d errors", len(errs))
		}
		appErr, _ := errors.FromError(errs[0])
		if appErr == nil || appErr.Tag != TagSvgLoadFailed || appErr.Metadata["statusCode"] != 404 {
			t.Errorf("error = %v", errs[0])
		}
	})
}

func TestLoadersFailWithSvgLoadFailed(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing.svg":
			http.NotFound(w, r)
		case "/big.svg":
			_, _ = w.Write(bytes.Repeat([]byte("x"), 100))
		default:
			_, _ = w.Write([]byte(red20x10))
		}
	}))
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	t.Cleanup(srv.Close)

	tests := []struct {
		name      string
		src       source
		env       *loadEnv
		ctx       context.Context
		route     string
		wantMeta  string
		wantCause error
	}{
		{"asset: not in the bundle", &assetSource{name: "icons/none.svg"}, &loadEnv{bundle: newCountingBundle(nil)},
			context.Background(), "assetSource.load.SvgLoadFailed", "icons/none.svg", &errors.Error{Tag: "AssetNotFound"}},
		{"asset: a package", &assetSource{name: "x.svg", pkg: "pkg"}, &loadEnv{bundle: newCountingBundle(nil)},
			context.Background(), "assetSource.load.SvgLoadFailed", "packages/pkg/x.svg", nil},
		{"asset: no bundle", &assetSource{name: "icons/none.svg"}, &loadEnv{}, context.Background(),
			"assetSource.load.SvgLoadFailed", "icons/none.svg", nil},
		{"file: missing", &fileSource{path: filepath.Join(dir, "none.svg")}, &loadEnv{}, context.Background(),
			"fileSource.load.SvgLoadFailed", filepath.Join(dir, "none.svg"), os.ErrNotExist},
		{"file: a directory", &fileSource{path: dir}, &loadEnv{}, context.Background(),
			"fileSource.load.SvgLoadFailed", dir, nil},
		{"file: context ended", &fileSource{path: dir}, &loadEnv{}, canceled,
			"fileSource.load.SvgLoadFailed", dir, context.Canceled},
		{"network: status 404", &networkSource{url: srv.URL + "/missing.svg"}, &loadEnv{}, context.Background(),
			"networkSource.load.SvgLoadFailed", srv.URL + "/missing.svg", nil},
		{"network: connection refused", &networkSource{url: closedURL + "/a.svg"}, &loadEnv{}, context.Background(),
			"networkSource.load.SvgLoadFailed", closedURL + "/a.svg", nil},
		{"network: invalid URL", &networkSource{url: "http://bad host/\x7f"}, &loadEnv{}, context.Background(),
			"networkSource.load.SvgLoadFailed", "http://bad host/\x7f", nil},
		{"network: context ended", &networkSource{url: srv.URL + "/a.svg"}, &loadEnv{}, canceled,
			"networkSource.load.SvgLoadFailed", srv.URL + "/a.svg", context.Canceled},
		{"network: larger than the limit", &networkSource{url: srv.URL + "/big.svg"}, &loadEnv{limit: 50}, context.Background(),
			"networkSource.load.SvgLoadFailed", srv.URL + "/big.svg", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.src.load(tt.ctx, tt.env)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["source"] != tt.wantMeta {
				t.Errorf("source = %v, want %q", appErr.Metadata["source"], tt.wantMeta)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("error %v is not %v", err, tt.wantCause)
			}
		})
	}
}

func TestParseErrorsKeepTheirTags(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		route string
	}{
		{"not well formed", `<svg><g></svg>`, "Parse->Parse->readXML.SvgParseFailed"},
		{"not svg", `<html xmlns="http://www.w3.org/1999/xhtml"/>`, "Parse->Parse.SvgNotSvg"},
		{"invalid viewBox", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 -1 1"/>`, "Parse->Parse->parseViewBox.SvgInvalidValue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestSourceDescriptions(t *testing.T) {
	tests := []struct {
		src  source
		want string
	}{
		{&assetSource{name: "a.svg"}, "SvgAssetLoader(a.svg)"},
		{&stringSource{"x"}, "SvgStringLoader"},
		{&memorySource{b: []byte("x")}, "SvgBytesLoader"},
		{&fileSource{path: "/a.svg"}, "SvgFileLoader(/a.svg)"},
		{&networkSource{url: "http://x/a.svg"}, "SvgNetworkLoader(http://x/a.svg)"},
	}
	for _, tt := range tests {
		if got := NewSvgPictureString("x").String(); got != "SvgPicture(SvgStringLoader)" {
			t.Errorf("String = %q", got)
		}
		if got := tt.src.String(); got != tt.want {
			t.Errorf("%T = %q, want %q", tt.src, got, tt.want)
		}
	}
}
