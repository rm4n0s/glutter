package painting

import (
	"context"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// ImageConfiguration configures an image provider to the situation in which the
// image is used: the bundle, device pixel ratio, locale, size and so on.
//
// Flutter: painting/image_provider.dart ImageConfiguration
type ImageConfiguration struct {
	// Binding is the PaintingBinding the image resolves with. Deviation: Flutter
	// reaches it through PaintingBinding.instance.
	Binding *PaintingBinding
	// Bundle is the asset bundle from which assets are loaded; nil for none.
	Bundle services.AssetBundle
	// DevicePixelRatio is the device pixel ratio where the image will be used.
	DevicePixelRatio foundation.Opt[float64]
	// Locale is the language and region for which to select the image.
	Locale foundation.Opt[ui.Locale]
	// TextDirection is the reading direction, for directional images;
	// ui.TextDirectionUnspecified for none.
	TextDirection ui.TextDirection
	// Size is the size at which the image will be rendered.
	Size foundation.Opt[ui.Size]
	// Platform is the platform the image is for.
	Platform foundation.Opt[foundation.TargetPlatform]
}

// ImageConfigurationEmpty is an image configuration that provides no
// additional information.
//
// Flutter: painting/image_provider.dart ImageConfiguration.empty
var ImageConfigurationEmpty = ImageConfiguration{}

// ImageConfigurationCopy holds the replacements for ImageConfiguration.CopyWith.
//
// Flutter: painting/image_provider.dart ImageConfiguration.copyWith
type ImageConfigurationCopy struct {
	Binding          *PaintingBinding
	Bundle           services.AssetBundle
	DevicePixelRatio foundation.Opt[float64]
	Locale           foundation.Opt[ui.Locale]
	TextDirection    foundation.Opt[ui.TextDirection]
	Size             foundation.Opt[ui.Size]
	Platform         foundation.Opt[foundation.TargetPlatform]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/image_provider.dart ImageConfiguration.copyWith
func (c ImageConfiguration) CopyWith(r ImageConfigurationCopy) ImageConfiguration {
	out := c
	if r.Binding != nil {
		out.Binding = r.Binding
	}
	if r.Bundle != nil {
		out.Bundle = r.Bundle
	}
	if r.DevicePixelRatio.IsSome() {
		out.DevicePixelRatio = r.DevicePixelRatio
	}
	if r.Locale.IsSome() {
		out.Locale = r.Locale
	}
	if r.TextDirection.IsSome() {
		out.TextDirection = r.TextDirection.Value()
	}
	if r.Size.IsSome() {
		out.Size = r.Size
	}
	if r.Platform.IsSome() {
		out.Platform = r.Platform
	}
	return out
}

// String describes the configuration.
//
// Flutter: painting/image_provider.dart ImageConfiguration.toString
func (c ImageConfiguration) String() string {
	var parts []string
	if c.Bundle != nil {
		parts = append(parts, "bundle: "+bundleString(c.Bundle))
	}
	if v, ok := c.DevicePixelRatio.Get(); ok {
		parts = append(parts, "devicePixelRatio: "+dartfmt.ToStringAsFixed(v, 1))
	}
	if v, ok := c.Locale.Get(); ok {
		parts = append(parts, "locale: "+v.String())
	}
	if c.TextDirection != ui.TextDirectionUnspecified {
		parts = append(parts, "textDirection: "+c.TextDirection.String())
	}
	if v, ok := c.Size.Get(); ok {
		parts = append(parts, "size: "+v.String())
	}
	if v, ok := c.Platform.Get(); ok {
		parts = append(parts, "platform: "+v.Name())
	}
	return "ImageConfiguration(" + strings.Join(parts, ", ") + ")"
}

func bundleString(b services.AssetBundle) string {
	if s, ok := b.(interface{ String() string }); ok {
		return s.String()
	}
	return foundation.RuntimeTypeName(b)
}

// ImageProvider identifies an image without committing to the precise final
// asset. Implementations: AssetImage, ExactAssetImage, FileImage, MemoryImage,
// NetworkImage and ResizeImage. The key an implementation gives must be
// comparable (a struct of comparable values, or a pointer).
//
// Deviation: ObtainKey is synchronous and takes the PaintingBinding, which
// stands for Flutter's PaintingBinding.instance and rootBundle.
//
// Flutter: painting/image_provider.dart ImageProvider
type ImageProvider interface {
	// ObtainKey converts the provider and the configuration to a key that
	// identifies the image in the cache.
	//
	// Flutter: painting/image_provider.dart ImageProvider.obtainKey
	ObtainKey(binding *PaintingBinding, configuration ImageConfiguration) (key any, err error)
	// LoadImage converts a key into an ImageStreamCompleter, and begins fetching
	// the image using decode.
	//
	// Flutter: painting/image_provider.dart ImageProvider.loadImage
	LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter
}

// errorImageCompleter is the completer of a stream whose key could not be made.
func newErrorImageCompleter(binding *PaintingBinding) *ImageStreamCompleter {
	return NewImageStreamCompleter(binding)
}

func imageProviderInformation(p ImageProvider, configuration ImageConfiguration, key any) func() []foundation.DiagnosticsNode {
	return func() []foundation.DiagnosticsNode {
		return []foundation.DiagnosticsNode{
			foundation.NewDiagnosticsProperty("Image provider", p),
			foundation.NewDiagnosticsProperty("Image configuration", configuration),
			foundation.NewDiagnosticsProperty("Image key", key, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}),
		}
	}
}

// ImageProviderResolve returns a stream for the image the provider names in the
// configuration. If obtaining the key fails, the error is reported to the
// stream's listeners in the next event turn.
//
// Flutter: painting/image_provider.dart ImageProvider.resolve
func ImageProviderResolve(binding *PaintingBinding, p ImageProvider, configuration ImageConfiguration) *ImageStream {
	stream := NewImageStream()
	var obtainedKey any
	didError := false
	handleError := func(err error) {
		if didError {
			return
		}
		didError = true
		key := obtainedKey
		binding.Post(func() {
			if stream.Completer() == nil {
				stream.SetCompleter(newErrorImageCompleter(binding))
			}
			stream.Completer().ReportError(ImageErrorDetails{Err: err, Context: "while resolving an image", Silent: true,
				Information: imageProviderInformation(p, configuration, key)})
		})
	}
	key, err := obtainKeySafely(binding, p, configuration)
	if err != nil {
		handleError(err)
		return stream
	}
	obtainedKey = key
	if err := safely(func() { ImageProviderResolveStreamForKey(binding, p, configuration, stream, key, handleError) }); err != nil {
		handleError(err)
	}
	return stream
}

// safely runs fn and returns the error it panics with, if the panic value is an
// error.
func safely(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	fn()
	return nil
}

func obtainKeySafely(binding *PaintingBinding, p ImageProvider, configuration ImageConfiguration) (key any, err error) {
	if e := safely(func() { key, err = p.ObtainKey(binding, configuration) }); e != nil {
		return nil, e
	}
	return key, err
}

// ImageProviderResolveStreamForKey called by ImageProviderResolve after the key
// is obtained: it finds the completer in the cache, or loads the image, and sets
// it on the stream.
//
// Flutter: painting/image_provider.dart ImageProvider.resolveStreamForKey
func ImageProviderResolveStreamForKey(binding *PaintingBinding, p ImageProvider, configuration ImageConfiguration,
	stream *ImageStream, key any, handleError ImageErrorListener) {
	cache := binding.ImageCache()
	if stream.Completer() != nil {
		cache.PutIfAbsent(key, func() *ImageStreamCompleter { return stream.Completer() }, handleError)
		return
	}
	completer := cache.PutIfAbsent(key, func() *ImageStreamCompleter {
		return p.LoadImage(binding, key, binding.InstantiateImageCodecWithSize)
	}, handleError)
	if completer != nil {
		stream.SetCompleter(completer)
	}
}

// ImageProviderObtainCacheStatus finds out where the image is in the cache. It
// returns false when the key could not be made; the error goes to handleError,
// or to the binding's reporter if it is nil.
//
// Flutter: painting/image_provider.dart ImageProvider.obtainCacheStatus
func ImageProviderObtainCacheStatus(binding *PaintingBinding, p ImageProvider, configuration ImageConfiguration,
	handleError ImageErrorListener) (ImageCacheStatus, bool) {
	key, err := obtainKeySafely(binding, p, configuration)
	if err != nil {
		if handleError != nil {
			handleError(err)
		} else {
			binding.ReportError(ImageErrorDetails{Err: err, Library: "painting library", Context: "while checking the cache location of an image",
				Information: imageProviderInformation(p, configuration, nil)})
		}
		return ImageCacheStatus{}, false
	}
	return binding.ImageCache().StatusForKey(key), true
}

// ImageProviderEvict evicts an entry from the image cache, the default cache of
// the binding if cache is nil.
//
// Flutter: painting/image_provider.dart ImageProvider.evict
func ImageProviderEvict(binding *PaintingBinding, p ImageProvider, cache *ImageCache, configuration ImageConfiguration) (bool, error) {
	if cache == nil {
		cache = binding.ImageCache()
	}
	key, err := obtainKeySafely(binding, p, configuration)
	if err != nil {
		return false, err
	}
	return cache.Evict(key), nil
}

// AssetBundleImageKey is the key used by AssetBundleImage providers: the bundle,
// the name of the asset and its scale.
//
// Flutter: painting/image_provider.dart AssetBundleImageKey
type AssetBundleImageKey struct {
	// Bundle is the bundle from which the image will be obtained.
	Bundle services.AssetBundle
	// Name is the name of the asset, such as "images/2.0x/logo.png".
	Name string
	// Scale is the scale to place in the ImageInfo.
	Scale float64
}

// String describes the key.
//
// Flutter: painting/image_provider.dart AssetBundleImageKey.toString
func (k AssetBundleImageKey) String() string {
	return "AssetBundleImageKey(bundle: " + bundleString(k.Bundle) + `, name: "` + k.Name + `", scale: ` + dartfmt.Double(k.Scale) + ")"
}

// assetBundleLoadImage is AssetBundleImageProvider.loadImage: it loads the asset
// of the key and decodes it, off the UI thread.
//
// Flutter: painting/image_provider.dart AssetBundleImageProvider.loadImage
func assetBundleLoadImage(binding *PaintingBinding, p ImageProvider, key AssetBundleImageKey, decode ImageDecoderCallback) *ImageStreamCompleter {
	ch := make(chan CodecResult, 1)
	ctx := binding.Context()
	go func() {
		data, err := key.Bundle.Load(ctx, key.Name)
		if err != nil {
			binding.Post(func() { binding.ImageCache().Evict(key) })
			ch <- CodecResult{Err: err}
			return
		}
		codec, err := decode(ctx, data, nil)
		ch <- CodecResult{Codec: codec, Err: err}
	}()
	return NewMultiFrameImageStreamCompleter(binding, MultiFrameImageStreamCompleterOpts{
		Codec: ch, Scale: key.Scale, DebugLabel: key.Name,
		InformationCollector: func() []foundation.DiagnosticsNode {
			return []foundation.DiagnosticsNode{
				foundation.NewDiagnosticsProperty("Image provider", p),
				foundation.NewDiagnosticsProperty("Image key", key),
			}
		},
	})
}

// ResizeImagePolicy is how ResizeImage resizes the image.
//
// Flutter: painting/image_provider.dart ResizeImagePolicy
type ResizeImagePolicy int

// The values of ResizeImagePolicy follow Flutter's declaration order.
const (
	// ResizeImagePolicyUnspecified is the zero value: exact.
	ResizeImagePolicyUnspecified ResizeImagePolicy = iota
	// ResizeImagePolicyExact makes the image the size, regardless of its aspect
	// ratio.
	//
	// Flutter: painting/image_provider.dart ResizeImagePolicy.exact
	ResizeImagePolicyExact
	// ResizeImagePolicyFit makes the image fit in the size, keeping its aspect
	// ratio.
	//
	// Flutter: painting/image_provider.dart ResizeImagePolicy.fit
	ResizeImagePolicyFit
)

// Name returns the Dart enum name, such as "fit".
func (p ResizeImagePolicy) Name() string {
	if p == ResizeImagePolicyFit {
		return "fit"
	}
	return "exact"
}

// ResizeImageKey is the key of a ResizeImage.
//
// Flutter: painting/image_provider.dart ResizeImageKey
type ResizeImageKey struct {
	providerCacheKey any
	policy           ResizeImagePolicy
	width, height    foundation.Opt[int]
	allowUpscaling   bool
}

// ResizeImage instructs Flutter to decode the image at the specified dimensions
// instead of at its native size.
//
// Flutter: painting/image_provider.dart ResizeImage
type ResizeImage struct {
	// ImageProvider is the provider of the original image.
	ImageProvider ImageProvider
	// Width is the width the image should decode to and cache; None for the
	// intrinsic width, or the aspect ratio of Height.
	Width foundation.Opt[int]
	// Height is the height the image should decode to and cache.
	Height foundation.Opt[int]
	// Policy is how the dimensions apply; the default is exact.
	Policy ResizeImagePolicy
	// AllowUpscaling says whether the image may be made larger than it is.
	AllowUpscaling bool
}

// ResizeImageOpts are the optional parameters of NewResizeImage.
//
// Flutter: painting/image_provider.dart ResizeImage.new
type ResizeImageOpts struct {
	// Width is the width the image should decode to.
	Width foundation.Opt[int]
	// Height is the height the image should decode to.
	Height foundation.Opt[int]
	// Policy defaults to exact.
	Policy ResizeImagePolicy
	// AllowUpscaling defaults to false.
	AllowUpscaling bool
}

// NewResizeImage creates a provider that decodes the image at the dimensions. In
// debug builds giving neither Width nor Height panics with the tag
// ResizeImageNoSize.
//
// Flutter: painting/image_provider.dart ResizeImage.new
func NewResizeImage(imageProvider ImageProvider, opts ...ResizeImageOpts) *ResizeImage {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Width.IsNone() && o.Height.IsNone() {
		panic(errors.New("ResizeImageNoSize", "a ResizeImage needs a width or a height"))
	}
	if o.Policy == ResizeImagePolicyUnspecified {
		o.Policy = ResizeImagePolicyExact
	}
	return &ResizeImage{imageProvider, o.Width, o.Height, o.Policy, o.AllowUpscaling}
}

// ResizeImageIfNeeded returns the provider wrapped in a ResizeImage if a cache
// width or height is given.
//
// Flutter: painting/image_provider.dart ResizeImage.resizeIfNeeded
func ResizeImageIfNeeded(cacheWidth, cacheHeight foundation.Opt[int], provider ImageProvider) ImageProvider {
	if cacheWidth.IsSome() || cacheHeight.IsSome() {
		return NewResizeImage(provider, ResizeImageOpts{Width: cacheWidth, Height: cacheHeight})
	}
	return provider
}

// ObtainKey makes the key from the key of the wrapped provider.
//
// Flutter: painting/image_provider.dart ResizeImage.obtainKey
func (r *ResizeImage) ObtainKey(binding *PaintingBinding, configuration ImageConfiguration) (any, error) {
	key, err := r.ImageProvider.ObtainKey(binding, configuration)
	if err != nil {
		return nil, err
	}
	return ResizeImageKey{key, r.Policy, r.Width, r.Height, r.AllowUpscaling}, nil
}

// LoadImage loads the wrapped image, decoding it at the size.
//
// Flutter: painting/image_provider.dart ResizeImage.loadImage
func (r *ResizeImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	k := key.(ResizeImageKey)
	decodeResize := func(ctx context.Context, data []byte, getTargetSize TargetImageSizeCallback) (*ui.Codec, error) {
		if getTargetSize != nil {
			return nil, errors.New("ResizeImageComposed", "ResizeImage cannot be composed with another ImageProvider that applies getTargetSize")
		}
		return decode(ctx, data, func(intrinsicWidth, intrinsicHeight int) TargetImageSize {
			return r.targetSize(intrinsicWidth, intrinsicHeight)
		})
	}
	completer := r.ImageProvider.LoadImage(binding, k.providerCacheKey, decodeResize)
	if foundation.DebugMode {
		completer.DebugLabel = completer.DebugLabel + " - Resized(" + optIntString(k.width) + "×" + optIntString(k.height) + ")"
	}
	completer.AddEphemeralErrorListener(func(error) {
		binding.Post(func() { binding.ImageCache().Evict(key) })
	})
	return completer
}

func optIntString(o foundation.Opt[int]) string {
	if v, ok := o.Get(); ok {
		return strconv.Itoa(v)
	}
	return "null"
}

func (r *ResizeImage) targetSize(intrinsicWidth, intrinsicHeight int) TargetImageSize {
	switch r.Policy {
	case ResizeImagePolicyFit:
		aspectRatio := float64(intrinsicWidth) / float64(intrinsicHeight)
		maxWidth := r.Width.OrElse(intrinsicWidth)
		maxHeight := r.Height.OrElse(intrinsicHeight)
		targetWidth, targetHeight := intrinsicWidth, intrinsicHeight
		if targetWidth > maxWidth {
			targetWidth = maxWidth
			targetHeight = int(float64(targetWidth) / aspectRatio)
		}
		if targetHeight > maxHeight {
			targetHeight = maxHeight
			targetWidth = int(float64(targetHeight) * aspectRatio)
		}
		if r.AllowUpscaling {
			switch {
			case r.Width.IsNone():
				targetHeight = r.Height.Value()
				targetWidth = int(float64(targetHeight) * aspectRatio)
			case r.Height.IsNone():
				targetWidth = r.Width.Value()
				targetHeight = int(float64(targetWidth) / aspectRatio)
			default:
				derivedMaxWidth := int(float64(maxHeight) * aspectRatio)
				derivedMaxHeight := int(float64(maxWidth) / aspectRatio)
				targetWidth = min(maxWidth, derivedMaxWidth)
				targetHeight = min(maxHeight, derivedMaxHeight)
			}
		}
		return TargetImageSize{foundation.Some(targetWidth), foundation.Some(targetHeight)}
	default:
		targetWidth, targetHeight := r.Width, r.Height
		if !r.AllowUpscaling {
			if w, ok := targetWidth.Get(); ok && w > intrinsicWidth {
				targetWidth = foundation.Some(intrinsicWidth)
			}
			if h, ok := targetHeight.Get(); ok && h > intrinsicHeight {
				targetHeight = foundation.Some(intrinsicHeight)
			}
		}
		return TargetImageSize{targetWidth, targetHeight}
	}
}

// NetworkImageLoadException is the error NetworkImage fails with when the server
// answers with a status other than 200.
//
// Flutter: painting/image_provider.dart NetworkImageLoadException
func newNetworkImageLoadError(statusCode int, uri string) error {
	return errors.New("NetworkImageLoadFailed", "HTTP request failed, statusCode: "+strconv.Itoa(statusCode)+", "+uri,
		"statusCode", statusCode, "uri", uri)
}

// NetworkImageKey is the key of a NetworkImage: the URL, the scale and the
// headers.
//
// Flutter: painting/image_provider.dart NetworkImage
type NetworkImageKey struct {
	URL     string
	Scale   float64
	headers string
}

// NetworkImage fetches the image from the URL, associating it with the given
// scale.
//
// Flutter: painting/image_provider.dart NetworkImage
type NetworkImage struct {
	// URL is where the image will be fetched from.
	URL string
	// Scale is the scale to place in the ImageInfo.
	Scale float64
	// Headers are HTTP headers sent with the request; nil for none.
	Headers map[string]string
}

// NetworkImageOpts are the optional parameters of NewNetworkImage.
//
// Flutter: painting/image_provider.dart NetworkImage.new
type NetworkImageOpts struct {
	// Scale defaults to 1.
	Scale foundation.Opt[float64]
	// Headers are HTTP headers sent with the request.
	Headers map[string]string
}

// NewNetworkImage creates a provider for the image at the URL.
//
// Flutter: painting/image_provider.dart NetworkImage.new
func NewNetworkImage(url string, opts ...NetworkImageOpts) *NetworkImage {
	o := firstOpts(opts)
	return &NetworkImage{URL: url, Scale: o.Scale.OrElse(1.0), Headers: o.Headers}
}

func (n *NetworkImage) key() NetworkImageKey {
	names := make([]string, 0, len(n.Headers))
	for k := range n.Headers {
		names = append(names, k)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		b.WriteString(strconv.Quote(k) + ":" + strconv.Quote(n.Headers[k]) + ";")
	}
	return NetworkImageKey{n.URL, n.Scale, b.String()}
}

// ObtainKey makes the key from the URL, the scale and the headers.
//
// Flutter: painting/_network_image_io.dart NetworkImage.obtainKey
func (n *NetworkImage) ObtainKey(*PaintingBinding, ImageConfiguration) (any, error) {
	return n.key(), nil
}

// LoadImage requests the image and decodes the response.
//
// Flutter: painting/_network_image_io.dart NetworkImage.loadImage
func (n *NetworkImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	chunks := make(chan ImageChunkEvent, 16)
	codecs := make(chan CodecResult, 1)
	ctx := binding.Context()
	go func() {
		defer close(chunks)
		codec, err := n.loadAsync(ctx, binding.HTTPClient(), chunks, decode)
		if err != nil {
			binding.Post(func() { binding.ImageCache().Evict(key) })
		}
		codecs <- CodecResult{Codec: codec, Err: err}
	}()
	return NewMultiFrameImageStreamCompleter(binding, MultiFrameImageStreamCompleterOpts{
		Codec: codecs, Scale: n.Scale, DebugLabel: n.URL, ChunkEvents: chunks,
		InformationCollector: func() []foundation.DiagnosticsNode {
			return []foundation.DiagnosticsNode{
				foundation.NewDiagnosticsProperty("Image provider", n),
				foundation.NewDiagnosticsProperty("Image key", key),
			}
		},
	})
}

func (n *NetworkImage) loadAsync(ctx context.Context, client *http.Client, chunks chan<- ImageChunkEvent, decode ImageDecoderCallback) (*ui.Codec, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.URL, nil)
	if err != nil {
		return nil, errors.NewErr("NetworkImageRequestFailed", err).SetMetadata("url", n.URL)
	}
	for name, value := range n.Headers {
		req.Header.Add(name, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.NewErr("NetworkImageRequestFailed", err).SetMetadata("url", n.URL)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, newNetworkImageLoadError(resp.StatusCode, req.URL.String())
	}
	total := foundation.None[int]()
	if resp.ContentLength >= 0 {
		total = foundation.Some(int(resp.ContentLength))
	}
	var data []byte
	buf := make([]byte, 32*1024)
	for {
		read, rerr := resp.Body.Read(buf)
		data = append(data, buf[:read]...)
		if read > 0 {
			select {
			case chunks <- ImageChunkEvent{CumulativeBytesLoaded: len(data), ExpectedTotalBytes: total}:
			case <-ctx.Done():
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return nil, errors.NewErr("NetworkImageRequestFailed", rerr).SetMetadata("url", n.URL)
		}
	}
	if len(data) == 0 {
		return nil, errors.New("NetworkImageEmpty", "NetworkImage is an empty file: "+req.URL.String(), "url", n.URL)
	}
	return decode(ctx, data, nil)
}

// String describes the provider.
//
// Flutter: painting/_network_image_io.dart NetworkImage.toString
func (n *NetworkImage) String() string {
	return `NetworkImage("` + n.URL + `", scale: ` + dartfmt.ToStringAsFixed(n.Scale, 1) + ")"
}

// FileImage decodes the given file as an image, associating it with the given
// scale.
//
// Flutter: painting/image_provider.dart FileImage
type FileImage struct {
	// Path is the path of the image file.
	Path string
	// Scale is the scale to place in the ImageInfo.
	Scale float64
}

// NewFileImage creates a provider for the image file.
//
// Flutter: painting/image_provider.dart FileImage.new
func NewFileImage(path string, scale ...float64) FileImage {
	s := 1.0
	if len(scale) > 0 {
		s = scale[0]
	}
	return FileImage{path, s}
}

// ObtainKey returns the provider itself as the key.
//
// Flutter: painting/image_provider.dart FileImage.obtainKey
func (f FileImage) ObtainKey(*PaintingBinding, ImageConfiguration) (any, error) { return f, nil }

// LoadImage reads the file and decodes it.
//
// Flutter: painting/image_provider.dart FileImage.loadImage
func (f FileImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	ch := make(chan CodecResult, 1)
	ctx := binding.Context()
	go func() {
		data, err := os.ReadFile(f.Path)
		if err == nil && len(data) == 0 {
			binding.Post(func() { binding.ImageCache().Evict(key) })
			err = errors.New("ImageFileEmpty", f.Path+" is empty and cannot be loaded as an image.", "path", f.Path)
		} else if err != nil {
			err = errors.NewErr("ImageFileReadFailed", err).SetMetadata("path", f.Path)
		}
		if err != nil {
			ch <- CodecResult{Err: err}
			return
		}
		codec, err := decode(ctx, data, nil)
		ch <- CodecResult{Codec: codec, Err: err}
	}()
	return NewMultiFrameImageStreamCompleter(binding, MultiFrameImageStreamCompleterOpts{
		Codec: ch, Scale: f.Scale, DebugLabel: f.Path,
		InformationCollector: func() []foundation.DiagnosticsNode {
			return []foundation.DiagnosticsNode{foundation.NewDiagnosticsNodeMessage("Path: " + f.Path)}
		},
	})
}

// String describes the provider.
//
// Flutter: painting/image_provider.dart FileImage.toString
func (f FileImage) String() string {
	return `FileImage("` + f.Path + `", scale: ` + dartfmt.ToStringAsFixed(f.Scale, 1) + ")"
}

// MemoryImage decodes the given bytes as an image, associating it with the given
// scale. Two providers are equal when they share the same slice, as Dart's
// Uint8List compares by identity.
//
// Flutter: painting/image_provider.dart MemoryImage
type MemoryImage struct {
	// Bytes is the bytes of the image.
	Bytes []byte
	// Scale is the scale to place in the ImageInfo.
	Scale float64
	id    *byte
}

// MemoryImageKey is the key of a MemoryImage.
//
// Flutter: no counterpart
type MemoryImageKey struct {
	id    *byte
	n     int
	Scale float64
}

// NewMemoryImage creates a provider for the bytes of an encoded image.
//
// Flutter: painting/image_provider.dart MemoryImage.new
func NewMemoryImage(bytes []byte, scale ...float64) *MemoryImage {
	s := 1.0
	if len(scale) > 0 {
		s = scale[0]
	}
	m := &MemoryImage{Bytes: bytes, Scale: s}
	if len(bytes) > 0 {
		m.id = &bytes[0]
	}
	return m
}

// ObtainKey makes the key from the identity of the bytes and the scale.
//
// Flutter: painting/image_provider.dart MemoryImage.obtainKey
func (m *MemoryImage) ObtainKey(*PaintingBinding, ImageConfiguration) (any, error) {
	return MemoryImageKey{m.id, len(m.Bytes), m.Scale}, nil
}

// LoadImage decodes the bytes.
//
// Flutter: painting/image_provider.dart MemoryImage.loadImage
func (m *MemoryImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	ch := make(chan CodecResult, 1)
	ctx := binding.Context()
	go func() {
		codec, err := decode(ctx, m.Bytes, nil)
		ch <- CodecResult{Codec: codec, Err: err}
	}()
	return NewMultiFrameImageStreamCompleter(binding, MultiFrameImageStreamCompleterOpts{
		Codec: ch, Scale: m.Scale, DebugLabel: "MemoryImage(" + strconv.Itoa(len(m.Bytes)) + " bytes)",
	})
}

// ExactAssetImage loads an image from an AssetBundle using the exact asset
// specified by name, without choosing a variant for the device pixel ratio.
//
// Flutter: painting/image_provider.dart ExactAssetImage
type ExactAssetImage struct {
	// AssetName is the key of the asset.
	AssetName string
	// Scale is the scale to place in the ImageInfo.
	Scale float64
	// Bundle is the bundle to load the asset from; nil uses the configuration's,
	// then the binding's root bundle.
	Bundle services.AssetBundle
	// Package is the package the asset is in; "" for none.
	Package string
}

// ExactAssetImageOpts are the optional parameters of NewExactAssetImage.
//
// Flutter: painting/image_provider.dart ExactAssetImage.new
type ExactAssetImageOpts struct {
	// Scale defaults to 1.
	Scale foundation.Opt[float64]
	// Bundle is the bundle to load the asset from.
	Bundle services.AssetBundle
	// Package is the package the asset is in.
	Package string
}

// NewExactAssetImage creates a provider for the asset.
//
// Flutter: painting/image_provider.dart ExactAssetImage.new
func NewExactAssetImage(assetName string, opts ...ExactAssetImageOpts) *ExactAssetImage {
	o := firstOpts(opts)
	return &ExactAssetImage{AssetName: assetName, Scale: o.Scale.OrElse(1.0), Bundle: o.Bundle, Package: o.Package}
}

// KeyName is the name of the asset with the package prefix.
//
// Flutter: painting/image_provider.dart ExactAssetImage.keyName
func (e *ExactAssetImage) KeyName() string { return keyNameFor(e.AssetName, e.Package) }

func keyNameFor(name, pkg string) string {
	if pkg == "" {
		return name
	}
	return "packages/" + pkg + "/" + name
}

func chooseBundle(binding *PaintingBinding, own services.AssetBundle, configuration ImageConfiguration) (services.AssetBundle, error) {
	switch {
	case own != nil:
		return own, nil
	case configuration.Bundle != nil:
		return configuration.Bundle, nil
	case binding.AssetBundle() != nil:
		return binding.AssetBundle(), nil
	}
	return nil, errors.New("AssetBundleMissing", "no asset bundle: the provider, the ImageConfiguration and the PaintingBinding have none")
}

// ObtainKey makes the key from the bundle, the asset name and the scale.
//
// Flutter: painting/image_provider.dart ExactAssetImage.obtainKey
func (e *ExactAssetImage) ObtainKey(binding *PaintingBinding, configuration ImageConfiguration) (any, error) {
	bundle, err := chooseBundle(binding, e.Bundle, configuration)
	if err != nil {
		return nil, err
	}
	return AssetBundleImageKey{bundle, e.KeyName(), e.Scale}, nil
}

// LoadImage loads the asset of the key.
//
// Flutter: painting/image_provider.dart AssetBundleImageProvider.loadImage
func (e *ExactAssetImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	return assetBundleLoadImage(binding, e, key.(AssetBundleImageKey), decode)
}

// String describes the provider.
//
// Flutter: painting/image_provider.dart ExactAssetImage.toString
func (e *ExactAssetImage) String() string {
	b := "null"
	if e.Bundle != nil {
		b = bundleString(e.Bundle)
	}
	return `ExactAssetImage(name: "` + e.KeyName() + `", scale: ` + dartfmt.ToStringAsFixed(e.Scale, 1) + ", bundle: " + b + ")"
}
