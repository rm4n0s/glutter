package painting

import (
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

var red = color.RGBA{255, 0, 0, 255}

// resolveInto resolves the provider and returns the recorder of its listener.
func resolveInto(pt *paintingTester, p ImageProvider, cfg ImageConfiguration) (*ImageStream, *imageRecorder) {
	cfg.Binding = pt.PaintingBinding
	stream := ImageProviderResolve(pt.PaintingBinding, p, cfg)
	rec := &imageRecorder{}
	stream.AddListener(rec.listener())
	return stream, rec
}

func requireRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), route)
	}
}

func TestMemoryImage(t *testing.T) {
	pt := newPaintingTester(t)
	data := pngBytes(t, 6, 4, red)
	p := NewMemoryImage(data, 2)
	s1, rec := resolveInto(pt, p, ImageConfigurationEmpty)
	pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
	info := rec.images[0]
	if info.Image.Width() != 6 || info.Image.Height() != 4 || info.Scale != 2 {
		t.Errorf("image = %v", info)
	}
	// The same provider, and another one on the same bytes, share the cache.
	s2, rec2 := resolveInto(pt, NewMemoryImage(data, 2), ImageConfigurationEmpty)
	if s1.Completer() != s2.Completer() || len(rec2.images) != 1 || !rec2.syncs[0] {
		t.Error("the second resolve must hit the cache and deliver at once")
	}
	// Another scale is another image.
	s3, _ := resolveInto(pt, NewMemoryImage(data, 1), ImageConfigurationEmpty)
	if s3.Completer() == s1.Completer() {
		t.Error("the scale is part of the key")
	}
	// Other bytes are another image even when equal in content.
	s4, _ := resolveInto(pt, NewMemoryImage(append([]byte(nil), data...), 2), ImageConfigurationEmpty)
	if s4.Completer() == s1.Completer() {
		t.Error("MemoryImage compares the bytes by identity")
	}
	status, ok := ImageProviderObtainCacheStatus(pt.PaintingBinding, p, ImageConfigurationEmpty, nil)
	if !ok || !status.Live {
		t.Errorf("status = %v, %v", status, ok)
	}
	evicted, err := ImageProviderEvict(pt.PaintingBinding, p, nil, ImageConfigurationEmpty)
	if err != nil || !evicted {
		t.Errorf("evict = %v, %v", evicted, err)
	}
}

func TestMemoryImageDecodeFailure(t *testing.T) {
	pt := newPaintingTester(t)
	_, rec := resolveInto(pt, NewMemoryImage([]byte("not an image")), ImageConfigurationEmpty)
	pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
	requireRoute(t, rec.errs[0], "ImageDecodeFailed")
}

func TestExactAssetImage(t *testing.T) {
	pt := newPaintingTester(t, PaintingBindingOpts{AssetBundle: services.NewFSAssetBundle(fstest.MapFS{
		"images/a.png":              {Data: pngBytes(t, 2, 2, red)},
		"packages/pkg/images/b.png": {Data: pngBytes(t, 3, 3, red)},
	})})
	tests := []struct {
		name  string
		p     *ExactAssetImage
		wantW int
		scale float64
		key   string
	}{
		{"plain", NewExactAssetImage("images/a.png"), 2, 1, "images/a.png"},
		{"scale", NewExactAssetImage("images/a.png", ExactAssetImageOpts{Scale: foundation.Some(3.0)}), 2, 3, "images/a.png"},
		{"package", NewExactAssetImage("images/b.png", ExactAssetImageOpts{Package: "pkg"}), 3, 1, "packages/pkg/images/b.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.p.KeyName() != tt.key {
				t.Errorf("KeyName = %q", tt.p.KeyName())
			}
			_, rec := resolveInto(pt, tt.p, ImageConfigurationEmpty)
			pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
			if got := rec.images[0]; got.Image.Width() != tt.wantW || got.Scale != tt.scale || got.DebugLabel != tt.key {
				t.Errorf("image = %v", got)
			}
		})
	}
	// The bundle of the configuration beats the root bundle.
	other := services.NewFSAssetBundle(fstest.MapFS{"images/a.png": {Data: pngBytes(t, 7, 7, red)}})
	_, rec := resolveInto(pt, NewExactAssetImage("images/a.png"), ImageConfiguration{Bundle: other})
	pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
	if rec.images[0].Image.Width() != 7 {
		t.Errorf("width = %d, want 7 from the configuration's bundle", rec.images[0].Image.Width())
	}
}

func TestAssetNotFound(t *testing.T) {
	pt := newPaintingTester(t, PaintingBindingOpts{AssetBundle: services.NewFSAssetBundle(fstest.MapFS{})})
	_, rec := resolveInto(pt, NewAssetImage("missing.png"), ImageConfigurationEmpty)
	pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
	requireRoute(t, rec.errs[0], "FSAssetBundle.Load.AssetNotFound")
	// The failed image is evicted, so a retry loads again.
	pt.waitFor(t, "the eviction", func() bool {
		return !pt.ImageCache().ContainsKey(AssetBundleImageKey{
			Bundle: pt.AssetBundle(), Name: "missing.png", Scale: 1})
	})
}

func TestAssetBundleMissing(t *testing.T) {
	pt := newPaintingTester(t)
	stream, rec := resolveInto(pt, NewAssetImage("a.png"), ImageConfigurationEmpty)
	if stream.Completer() != nil {
		t.Fatal("the error is reported in a later event turn")
	}
	pt.host.runPosted()
	if len(rec.errs) != 1 {
		t.Fatalf("errors = %v", rec.errs)
	}
	requireRoute(t, rec.errs[0], "AssetBundleMissing")
}

func TestAssetImageVariants(t *testing.T) {
	fsys := fstest.MapFS{
		"images/a.png":      {Data: pngBytes(t, 1, 1, red)},
		"images/2.0x/a.png": {Data: pngBytes(t, 2, 2, red)},
		"images/3.0x/a.png": {Data: pngBytes(t, 3, 3, red)},
		"images/b.png":      {Data: pngBytes(t, 1, 1, red)},
		"images/4.0x/b.png": {Data: pngBytes(t, 4, 4, red)},
		"images/c.png":      {Data: pngBytes(t, 1, 1, red)},
		"images/2.0x/c.txt": {Data: []byte("x")},
	}
	pt := newPaintingTester(t, PaintingBindingOpts{AssetBundle: services.NewFSAssetBundle(fsys)})
	tests := []struct {
		asset     string
		dpr       foundation.Opt[float64]
		wantName  string
		wantScale float64
	}{
		{"images/a.png", foundation.None[float64](), "images/a.png", 1},
		{"images/a.png", foundation.Some(1.0), "images/a.png", 1},
		{"images/a.png", foundation.Some(2.0), "images/2.0x/a.png", 2},
		{"images/a.png", foundation.Some(1.5), "images/2.0x/a.png", 2},
		{"images/a.png", foundation.Some(2.4), "images/2.0x/a.png", 2},
		{"images/a.png", foundation.Some(2.6), "images/3.0x/a.png", 3},
		{"images/a.png", foundation.Some(5.0), "images/3.0x/a.png", 3},
		{"images/a.png", foundation.Some(0.5), "images/a.png", 1},
		{"images/b.png", foundation.Some(3.0), "images/4.0x/b.png", 4},
		{"images/b.png", foundation.Some(1.0), "images/b.png", 1},
		{"images/c.png", foundation.Some(2.0), "images/c.png", 1},
	}
	for _, tt := range tests {
		key, err := NewAssetImage(tt.asset).ObtainKey(pt.PaintingBinding, ImageConfiguration{DevicePixelRatio: tt.dpr})
		if err != nil {
			t.Fatal(err)
		}
		k := key.(AssetBundleImageKey)
		if k.Name != tt.wantName || k.Scale != tt.wantScale {
			t.Errorf("%s at %v: got %s @ %v, want %s @ %v", tt.asset, tt.dpr, k.Name, k.Scale, tt.wantName, tt.wantScale)
		}
	}
	// The chosen variant is loaded, with its scale.
	_, rec := resolveInto(pt, NewAssetImage("images/a.png"), ImageConfiguration{DevicePixelRatio: foundation.Some(3.0)})
	pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
	if got := rec.images[0]; got.Image.Width() != 3 || got.Scale != 3 {
		t.Errorf("image = %v", got)
	}
	if got := NewAssetImage("x.png", AssetImageOpts{Package: "p"}).String(); got != `AssetImage(bundle: null, name: "packages/p/x.png")` {
		t.Errorf("String = %q", got)
	}
}

func TestFileImage(t *testing.T) {
	pt := newPaintingTester(t)
	dir := t.TempDir()
	good := filepath.Join(dir, "good.png")
	empty := filepath.Join(dir, "empty.png")
	if err := os.WriteFile(good, pngBytes(t, 5, 5, red), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		path      string
		wantRoute string
	}{
		{"good", good, ""},
		{"empty", empty, "ImageFileEmpty"},
		{"missing", filepath.Join(dir, "missing.png"), "ImageFileReadFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rec := resolveInto(pt, NewFileImage(tt.path, 1.5), ImageConfigurationEmpty)
			if tt.wantRoute == "" {
				pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
				if rec.images[0].Image.Width() != 5 || rec.images[0].Scale != 1.5 {
					t.Errorf("image = %v", rec.images[0])
				}
				return
			}
			pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
			requireRoute(t, rec.errs[0], tt.wantRoute)
		})
	}
	if got, want := NewFileImage("/x.png", 2).String(), `FileImage("/x.png", scale: 2.0)`; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestNetworkImage(t *testing.T) {
	var gotHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/ok.png", func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Test")
		_, _ = w.Write(pngBytes(t, 4, 3, red))
	})
	mux.HandleFunc("/empty.png", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/bad.png", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("junk")) })
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	pt := newPaintingTester(t, PaintingBindingOpts{HTTPClient: server.Client()})

	t.Run("ok", func(t *testing.T) {
		p := NewNetworkImage(server.URL+"/ok.png", NetworkImageOpts{Headers: map[string]string{"X-Test": "yes"}, Scale: foundation.Some(2.0)})
		_, rec := resolveInto(pt, p, ImageConfigurationEmpty)
		pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
		if rec.images[0].Image.Width() != 4 || rec.images[0].Scale != 2 || gotHeader != "yes" {
			t.Errorf("image %v header %q", rec.images[0], gotHeader)
		}
		if len(rec.chunks) == 0 {
			// The chunks arrive on the UI thread; they may come after the image.
			pt.waitFor(t, "a chunk", func() bool { return len(rec.chunks) > 0 })
		}
		if rec.chunks[len(rec.chunks)-1].CumulativeBytesLoaded == 0 {
			t.Errorf("chunks = %v", rec.chunks)
		}
	})
	failures := []struct {
		name  string
		path  string
		route string
	}{
		{"not found", "/missing.png", "NetworkImageLoadFailed"},
		{"empty", "/empty.png", "NetworkImageEmpty"},
		{"not an image", "/bad.png", "ImageDecodeFailed"},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			_, rec := resolveInto(pt, NewNetworkImage(server.URL+tt.path), ImageConfigurationEmpty)
			pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
			requireRoute(t, rec.errs[0], tt.route)
		})
	}
	t.Run("unreachable", func(t *testing.T) {
		url := server.URL + "/ok.png"
		server2 := httptest.NewServer(mux)
		bad := server2.URL
		server2.Close()
		_ = url
		_, rec := resolveInto(pt, NewNetworkImage(bad+"/x.png"), ImageConfigurationEmpty)
		pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
		requireRoute(t, rec.errs[0], "NetworkImageRequestFailed")
	})
	t.Run("keys", func(t *testing.T) {
		a, _ := NewNetworkImage("u", NetworkImageOpts{Headers: map[string]string{"a": "1", "b": "2"}}).ObtainKey(pt.PaintingBinding, ImageConfigurationEmpty)
		b, _ := NewNetworkImage("u", NetworkImageOpts{Headers: map[string]string{"b": "2", "a": "1"}}).ObtainKey(pt.PaintingBinding, ImageConfigurationEmpty)
		c, _ := NewNetworkImage("u").ObtainKey(pt.PaintingBinding, ImageConfigurationEmpty)
		if a != b || a == c {
			t.Errorf("keys %v %v %v", a, b, c)
		}
	})
}

func TestResizeImageTargetSize(t *testing.T) {
	i := foundation.Some[int]
	none := foundation.None[int]()
	tests := []struct {
		name         string
		w, h         foundation.Opt[int]
		policy       ResizeImagePolicy
		upscale      bool
		iw, ih       int
		wantW, wantH foundation.Opt[int]
	}{
		{"exact smaller", i(5), i(2), ResizeImagePolicyExact, false, 10, 8, i(5), i(2)},
		{"exact width only", i(5), none, ResizeImagePolicyExact, false, 10, 8, i(5), none},
		{"exact no upscaling", i(50), i(20), ResizeImagePolicyExact, false, 10, 8, i(10), i(8)},
		{"exact upscaling", i(50), i(20), ResizeImagePolicyExact, true, 10, 8, i(50), i(20)},
		{"fit by width", i(5), i(100), ResizeImagePolicyFit, false, 10, 8, i(5), i(4)},
		{"fit by height", i(100), i(4), ResizeImagePolicyFit, false, 10, 8, i(5), i(4)},
		{"fit no upscaling", i(100), i(100), ResizeImagePolicyFit, false, 10, 8, i(10), i(8)},
		{"fit upscaling both", i(40), i(40), ResizeImagePolicyFit, true, 10, 8, i(40), i(32)},
		{"fit upscaling width only", i(40), none, ResizeImagePolicyFit, true, 10, 8, i(40), i(32)},
		{"fit upscaling height only", none, i(16), ResizeImagePolicyFit, true, 10, 8, i(20), i(16)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResizeImage(NewMemoryImage(nil), ResizeImageOpts{Width: tt.w, Height: tt.h, Policy: tt.policy, AllowUpscaling: tt.upscale})
			got := r.targetSize(tt.iw, tt.ih)
			if got.Width != tt.wantW || got.Height != tt.wantH {
				t.Errorf("targetSize = %v x %v, want %v x %v", got.Width, got.Height, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestResizeImageLoadsAtTheSize(t *testing.T) {
	pt := newPaintingTester(t)
	data := pngBytes(t, 10, 8, red)
	inner := NewMemoryImage(data)
	r := NewResizeImage(inner, ResizeImageOpts{Width: foundation.Some(5)})
	_, rec := resolveInto(pt, r, ImageConfigurationEmpty)
	pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
	if got := rec.images[0].Image; got.Width() != 5 || got.Height() != 4 {
		t.Errorf("image = %d x %d, want 5 x 4", got.Width(), got.Height())
	}
	// The resized image is cached apart from the original.
	_, orig := resolveInto(pt, inner, ImageConfigurationEmpty)
	pt.waitFor(t, "the original", func() bool { return len(orig.images) == 1 })
	if orig.images[0].Image.Width() != 10 {
		t.Errorf("original width = %d", orig.images[0].Image.Width())
	}
	if ResizeImageIfNeeded(foundation.None[int](), foundation.None[int](), inner) != ImageProvider(inner) {
		t.Error("no cache size, no resize")
	}
	if _, ok := ResizeImageIfNeeded(foundation.Some(3), foundation.None[int](), inner).(*ResizeImage); !ok {
		t.Error("a cache width wraps the provider")
	}
	// A failed resize is evicted from the cache.
	bad := NewResizeImage(NewMemoryImage([]byte("junk")), ResizeImageOpts{Width: foundation.Some(5)})
	key, _ := bad.ObtainKey(pt.PaintingBinding, ImageConfigurationEmpty)
	_, rec2 := resolveInto(pt, bad, ImageConfigurationEmpty)
	pt.waitFor(t, "the error", func() bool { return len(rec2.errs) == 1 })
	pt.waitFor(t, "the eviction", func() bool { return !pt.ImageCache().ContainsKey(key) })
}

func TestImageConfigurationBundleAndString(t *testing.T) {
	cfg := ImageConfiguration{
		DevicePixelRatio: foundation.Some(2.0), Locale: foundation.Some(ui.NewLocale("en", "US")),
		TextDirection: ui.TextDirectionRTL, Size: foundation.Some(ui.Size{Width: 3, Height: 4}),
		Platform: foundation.Some(foundation.TargetPlatformLinux),
	}
	want := "ImageConfiguration(devicePixelRatio: 2.0, locale: en_US, textDirection: TextDirection.rtl, size: Size(3.0, 4.0), platform: linux)"
	if got := cfg.String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if got := ImageConfigurationEmpty.String(); got != "ImageConfiguration()" {
		t.Errorf("empty String = %q", got)
	}
	copy := cfg.CopyWith(ImageConfigurationCopy{DevicePixelRatio: foundation.Some(3.0), Bundle: services.NewFSAssetBundle(fstest.MapFS{})})
	if copy.DevicePixelRatio.Value() != 3 || copy.Bundle == nil || copy.TextDirection != ui.TextDirectionRTL {
		t.Errorf("copy = %+v", copy)
	}
	if copy == cfg {
		t.Error("the copy differs")
	}
}

func TestResizeImageComposed(t *testing.T) {
	pt := newPaintingTester(t)
	inner := NewResizeImage(NewMemoryImage(pngBytes(t, 10, 10, red)), ResizeImageOpts{Width: foundation.Some(5)})
	outer := NewResizeImage(inner, ResizeImageOpts{Width: foundation.Some(2)})
	_, rec := resolveInto(pt, outer, ImageConfigurationEmpty)
	pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
	requireRoute(t, rec.errs[0], "ResizeImageComposed")
}
