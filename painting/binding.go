package painting

import (
	"bytes"
	"context"
	"image"
	"net/http"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// TargetImageSize is the size an image should be decoded at. A None side keeps
// the aspect ratio, or the intrinsic size if the other is None too.
//
// Flutter: dart:ui TargetImageSize
type TargetImageSize struct {
	// Width is the target width in pixels.
	Width foundation.Opt[int]
	// Height is the target height in pixels.
	Height foundation.Opt[int]
}

// TargetImageSizeCallback picks the size to decode an image at, from its
// intrinsic size.
//
// Flutter: dart:ui TargetImageSizeCallback
type TargetImageSizeCallback func(intrinsicWidth, intrinsicHeight int) TargetImageSize

// ImageDecoderCallback decodes the bytes of an image. getTargetSize may be nil,
// to decode at the intrinsic size.
//
// Flutter: painting/image_provider.dart ImageDecoderCallback
type ImageDecoderCallback func(ctx context.Context, data []byte, getTargetSize TargetImageSizeCallback) (*ui.Codec, error)

// ImageErrorDetails describes an error that happened while resolving an image.
//
// Flutter: foundation/assertions.dart FlutterErrorDetails
type ImageErrorDetails struct {
	// Err is the error.
	Err error
	// Library is the part of the framework that reported it.
	Library string
	// Context says what was going on, such as "while resolving an image".
	Context string
	// Silent says that the error is expected, such as a network error, and the
	// reporter may show less of it.
	Silent bool
	// Information describes the image provider, its configuration and its key, in
	// debug builds.
	Information func() []foundation.DiagnosticsNode
}

// PaintingBindingOpts are the optional parameters of NewPaintingBinding.
//
// Flutter: painting/binding.dart PaintingBinding
type PaintingBindingOpts struct {
	// Post runs a function on the UI thread, from any goroutine; the default is
	// the Post of the scheduler's PlatformDispatcher.
	Post func(fn func())
	// AfterFunc calls fn on the UI thread after d, and returns the function that
	// cancels it; the default is time.AfterFunc and Post.
	AfterFunc func(d time.Duration, fn func()) (stop func())
	// ImageCache is the cache the binding uses; the default is a new one with the
	// default limits.
	ImageCache *ImageCache
	// AssetBundle is the root bundle: the one an AssetImage reads from when the
	// provider and the ImageConfiguration name none.
	AssetBundle services.AssetBundle
	// HTTPClient makes the requests of NetworkImage; the default is a new
	// http.Client.
	HTTPClient *http.Client
	// ErrorReporter receives the details of the errors nobody handled. When it
	// is nil they go to Reporter.
	ErrorReporter func(ImageErrorDetails)
	// Reporter is the framework's error reporter (SPEC §14.2); it receives the
	// error of each report that ErrorReporter does not take.
	Reporter foundation.ErrorReporter
}

// PaintingBinding is the glue between the painting library and the Flutter
// engine: it owns the image cache and decodes the images.
//
// Flutter: painting/binding.dart PaintingBinding
type PaintingBinding struct {
	ctx           context.Context
	scheduler     *scheduler.SchedulerBinding
	post          func(fn func())
	afterFunc     func(d time.Duration, fn func()) (stop func())
	imageCache    *ImageCache
	errorReporter func(ImageErrorDetails)
	reporter      foundation.ErrorReporter
	systemFonts   *foundation.ChangeNotifier
	assetBundle   services.AssetBundle
	httpClient    *http.Client
}

// NewPaintingBinding creates the binding. The context is the lifetime of the
// application: image loads stop when it is cancelled.
//
// Flutter: painting/binding.dart PaintingBinding.initInstances
func NewPaintingBinding(ctx context.Context, sched *scheduler.SchedulerBinding, opts ...PaintingBindingOpts) *PaintingBinding {
	o := firstOpts(opts)
	b := &PaintingBinding{ctx: ctx, scheduler: sched, post: o.Post, afterFunc: o.AfterFunc, imageCache: o.ImageCache,
		errorReporter: o.ErrorReporter, reporter: o.Reporter, systemFonts: foundation.NewChangeNotifier(), assetBundle: o.AssetBundle, httpClient: o.HTTPClient}
	if b.httpClient == nil {
		b.httpClient = &http.Client{}
	}
	if b.post == nil {
		d := sched.PlatformDispatcher()
		b.post = d.Post
	}
	if b.afterFunc == nil {
		b.afterFunc = func(d time.Duration, fn func()) func() {
			t := time.AfterFunc(d, func() { b.post(fn) })
			return func() { t.Stop() }
		}
	}
	if b.imageCache == nil {
		b.imageCache = NewImageCache(sched)
	}
	return b
}

// Context is the lifetime of the application.
//
// Flutter: no counterpart
func (b *PaintingBinding) Context() context.Context { return b.ctx }

// Scheduler is the scheduler binding the painting binding runs on.
//
// Flutter: no counterpart
func (b *PaintingBinding) Scheduler() *scheduler.SchedulerBinding { return b.scheduler }

// Post runs the function on the UI thread; it is safe from any goroutine.
//
// Flutter: no counterpart (Dart futures complete on the UI isolate)
func (b *PaintingBinding) Post(fn func()) { b.post(fn) }

// AfterFunc calls the function on the UI thread after the duration, and returns
// the function that cancels the call.
//
// Flutter: no counterpart (Dart's Timer)
func (b *PaintingBinding) AfterFunc(d time.Duration, fn func()) (stop func()) {
	return b.afterFunc(d, fn)
}

// AssetBundle is the root bundle, or nil if the binding has none.
//
// Flutter: services/asset_bundle.dart rootBundle
func (b *PaintingBinding) AssetBundle() services.AssetBundle { return b.assetBundle }

// HTTPClient is the client NetworkImage requests with.
//
// Flutter: painting/_network_image_io.dart debugNetworkImageHttpClientProvider
func (b *PaintingBinding) HTTPClient() *http.Client { return b.httpClient }

// ImageCache is the singleton cache of images that have been loaded.
//
// Flutter: painting/binding.dart PaintingBinding.imageCache
func (b *PaintingBinding) ImageCache() *ImageCache { return b.imageCache }

// TextEngine is the text engine of the platform dispatcher, which lays out the
// paragraphs of a TextPainter.
//
// Flutter: no counterpart
func (b *PaintingBinding) TextEngine() ui.ParagraphFactory {
	return b.scheduler.PlatformDispatcher().TextEngine()
}

// ReportError gives an error nobody handled to the reporter: ErrorReporter if
// it was set, else the framework's error reporter.
//
// Flutter: foundation/assertions.dart FlutterError.reportError
func (b *PaintingBinding) ReportError(details ImageErrorDetails) {
	switch {
	case b == nil:
	case b.errorReporter != nil:
		b.errorReporter(details)
	case b.reporter != nil && details.Err != nil:
		b.reporter(details.Err)
	}
}

// panicReporter is the reporter for the panics recovered in image listeners: the
// framework's error reporter, or ErrorReporter when only that was set; nil when
// the binding has neither, and then the panic goes on.
func (b *PaintingBinding) panicReporter() foundation.ErrorReporter {
	switch {
	case b == nil:
		return nil
	case b.reporter != nil:
		return b.reporter
	case b.errorReporter != nil:
		return func(err error) { b.errorReporter(ImageErrorDetails{Err: err, Library: "image resource service"}) }
	}
	return nil
}

// InstantiateImageCodecWithSize decodes an image at the size getTargetSize
// picks. It fails with the tag ImageDecodeFailed if the data is not an image.
//
// Flutter: painting/binding.dart PaintingBinding.instantiateImageCodecWithSize
func (b *PaintingBinding) InstantiateImageCodecWithSize(ctx context.Context, data []byte, getTargetSize TargetImageSizeCallback) (*ui.Codec, error) {
	if getTargetSize == nil {
		return ui.InstantiateImageCodec(ctx, data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Let the decoder report why the data is not an image.
		return ui.InstantiateImageCodec(ctx, data)
	}
	target := getTargetSize(cfg.Width, cfg.Height)
	var o ui.InstantiateImageCodecOpts
	if v, ok := target.Width.Get(); ok {
		o.TargetWidth = v
	}
	if v, ok := target.Height.Get(); ok {
		o.TargetHeight = v
	}
	return ui.InstantiateImageCodec(ctx, data, o)
}

// Evict clears the image cache, as when an asset changed.
//
// Flutter: painting/binding.dart PaintingBinding.evict
func (b *PaintingBinding) Evict(asset string) {
	b.imageCache.Clear()
	b.imageCache.ClearLiveImages()
}

// HandleMemoryPressure clears the image cache.
//
// Flutter: painting/binding.dart PaintingBinding.handleMemoryPressure
func (b *PaintingBinding) HandleMemoryPressure() { b.imageCache.Clear() }

// SystemFonts notifies when the fonts of the system change.
//
// Flutter: painting/binding.dart PaintingBinding.systemFonts
func (b *PaintingBinding) SystemFonts() foundation.Listenable { return b.systemFonts }

// HandleSystemFontsChange tells the listeners of SystemFonts that the fonts of
// the system changed.
//
// Flutter: painting/binding.dart PaintingBinding.handleSystemMessage (fontsChange)
func (b *PaintingBinding) HandleSystemFontsChange() { b.systemFonts.NotifyListeners() }
