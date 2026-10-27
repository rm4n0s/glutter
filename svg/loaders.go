package svg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/services"
)

// maxSourceBytes is the largest SVG that a loader reads: 64 MiB.
const maxSourceBytes = 64 << 20

// loadEnv is what the loaders take from the app: its asset bundle and its HTTP client.
type loadEnv struct {
	bundle services.AssetBundle
	client *http.Client
	// limit is the largest SVG a loader reads; zero is maxSourceBytes.
	limit int
}

func (e *loadEnv) maxBytes() int {
	if e.limit > 0 {
		return e.limit
	}
	return maxSourceBytes
}

// source says where the bytes of an SvgPicture come from. It stands for flutter_svg's SvgLoader and its subclasses.
type source interface {
	// cacheKey is a comparable value that is the same for the sources that give the same bytes (the key of the cache
	// is this and the theme).
	cacheKey(env *loadEnv) any
	// load reads the bytes. It runs on a goroutine of its own, except for the sources that are immediate.
	load(ctx context.Context, env *loadEnv) ([]byte, error)
	// immediate says that the bytes are in memory: the widget parses them when it is built, not on a goroutine.
	immediate() bool
	// equal says whether the other source is the same one, for DidUpdateWidget.
	equal(other source) bool
	// String describes the source, as flutter_svg's loaders do.
	String() string
}

// parseEntry parses the bytes of a source for a theme and measures the document, for the cache.
func parseEntry(data []byte, theme SvgTheme) (*entry, error) {
	doc, err := Parse(data, ParseOpts{Theme: theme})
	if err != nil {
		return nil, err
	}
	return &entry{doc: doc, bytes: doc.d.ApproximateBytes()}, nil
}

// bundleKey makes a comparable value of an asset bundle, which may be of a type that cannot be a map key.
func bundleKey(b services.AssetBundle) any {
	if b == nil {
		return nil
	}
	if reflect.TypeOf(b).Comparable() {
		return b
	}
	return fmt.Sprintf("%T", b)
}

// sameBundle compares two bundles without panicking on the types that cannot be compared.
func sameBundle(a, b services.AssetBundle) bool {
	return bundleKey(a) == bundleKey(b)
}

// assetSource is SvgPicture.asset: SvgAssetLoader.
type assetSource struct {
	name, pkg string
	// bundle is the bundle of the options, or nil for the one of the binding.
	bundle services.AssetBundle
}

type assetKey struct {
	name, pkg string
	bundle    any
}

func (s *assetSource) resolve(env *loadEnv) services.AssetBundle {
	if s.bundle != nil {
		return s.bundle
	}
	return env.bundle
}

func (s *assetSource) key() string {
	if s.pkg != "" {
		return "packages/" + s.pkg + "/" + s.name
	}
	return s.name
}

func (s *assetSource) cacheKey(env *loadEnv) any {
	return assetKey{s.name, s.pkg, bundleKey(s.resolve(env))}
}

func (s *assetSource) immediate() bool { return false }

func (s *assetSource) equal(o source) bool {
	t, ok := o.(*assetSource)
	return ok && t.name == s.name && t.pkg == s.pkg && sameBundle(t.bundle, s.bundle)
}

func (s *assetSource) String() string { return "SvgAssetLoader(" + s.name + ")" }

// load reads the asset from the bundle. A bundle is an interface that an app can implement, so its error is tagged
// here with SvgLoadFailed (the cause is the Unwrap of the error, AssetNotFound for the bundle of Glutter).
func (s *assetSource) load(ctx context.Context, env *loadEnv) ([]byte, error) {
	bundle := s.resolve(env)
	if bundle == nil {
		return nil, errors.New(TagSvgLoadFailed, "there is no asset bundle to load the SVG from", "source", s.key())
	}
	data, err := bundle.Load(ctx, s.key())
	if err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.key())
	}
	return data, nil
}

// stringSource is SvgPicture.string: SvgStringLoader.
type stringSource struct{ s string }

type stringKey struct{ s string }

func (s *stringSource) cacheKey(*loadEnv) any { return stringKey{s.s} }
func (s *stringSource) immediate() bool       { return true }
func (s *stringSource) equal(o source) bool {
	t, ok := o.(*stringSource)
	return ok && t.s == s.s
}
func (s *stringSource) String() string                                 { return "SvgStringLoader" }
func (s *stringSource) load(context.Context, *loadEnv) ([]byte, error) { return []byte(s.s), nil }

// memorySource is SvgPicture.memory: SvgBytesLoader.
type memorySource struct {
	b []byte

	once sync.Once
	sum  [sha256.Size]byte
}

type memoryKey struct {
	size int
	sum  [sha256.Size]byte
}

func (s *memorySource) cacheKey(*loadEnv) any {
	// Two slices with the same bytes give the same document, which flutter_svg's identity of the list would not say.
	s.once.Do(func() { s.sum = sha256.Sum256(s.b) })
	return memoryKey{len(s.b), s.sum}
}
func (s *memorySource) immediate() bool { return true }
func (s *memorySource) equal(o source) bool {
	t, ok := o.(*memorySource)
	return ok && bytes.Equal(t.b, s.b)
}
func (s *memorySource) String() string                                 { return "SvgBytesLoader" }
func (s *memorySource) load(context.Context, *loadEnv) ([]byte, error) { return s.b, nil }

// fileSource is SvgPicture.file: SvgFileLoader.
type fileSource struct{ path string }

type fileKey struct{ path string }

func (s *fileSource) cacheKey(*loadEnv) any { return fileKey{s.path} }
func (s *fileSource) immediate() bool       { return false }
func (s *fileSource) equal(o source) bool {
	t, ok := o.(*fileSource)
	return ok && t.path == s.path
}
func (s *fileSource) String() string { return "SvgFileLoader(" + s.path + ")" }

// load reads the file. On the web and on Android a path is not a file the app can read, and this fails.
func (s *fileSource) load(ctx context.Context, _ *loadEnv) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.path)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.path)
	}
	return data, nil
}

// networkSource is SvgPicture.network: SvgNetworkLoader.
type networkSource struct {
	url     string
	headers map[string]string
}

type networkKey struct{ url, headers string }

func (s *networkSource) headerKey() string {
	names := make([]string, 0, len(s.headers))
	for name := range s.headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		b.WriteString(strconv.Quote(name) + ":" + strconv.Quote(s.headers[name]) + "\n")
	}
	return b.String()
}

func (s *networkSource) cacheKey(*loadEnv) any { return networkKey{s.url, s.headerKey()} }
func (s *networkSource) immediate() bool       { return false }
func (s *networkSource) equal(o source) bool {
	t, ok := o.(*networkSource)
	return ok && t.url == s.url && t.headerKey() == s.headerKey()
}
func (s *networkSource) String() string { return "SvgNetworkLoader(" + s.url + ")" }

// load requests the URL with the headers and reads the answer: only a status of 200 is a document. All network
// pictures are cached whatever the HTTP headers of the answer say, as flutter_svg's are.
func (s *networkSource) load(ctx context.Context, env *loadEnv) ([]byte, error) {
	client := env.client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.url)
	}
	for name, value := range s.headers {
		req.Header.Add(name, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.url)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(TagSvgLoadFailed, "HTTP request failed, statusCode: "+strconv.Itoa(resp.StatusCode),
			"source", s.url, "statusCode", resp.StatusCode)
	}
	limit := env.maxBytes()
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, errors.NewErr(TagSvgLoadFailed, err).SetMetadata("source", s.url)
	}
	if len(data) > limit {
		return nil, errors.New(TagSvgLoadFailed, "the SVG is larger than the limit of the loaders", "source", s.url, "limit", limit)
	}
	return data, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagSvgLoadFailed is the tag of an asset, a file or a URL that could not be read; the metadata is "source" (the
	// asset key, the path or the URL), and "statusCode" for a status of HTTP other than 200. The cause, if any, is the
	// Unwrap of the error.
	TagSvgLoadFailed = "SvgLoadFailed"
)
