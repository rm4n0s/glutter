package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// PrecacheImageOpts are the optional parameters of PrecacheImage.
//
// Flutter: widgets/image.dart precacheImage
type PrecacheImageOpts struct {
	// Size is the size the image will be shown at, if it is known.
	Size foundation.Opt[ui.Size]
	// OnError is called when the image fails to load; the default reports the
	// error without making noise.
	OnError painting.ImageErrorListener
}

// PrecacheImage prefetches an image into the image cache. The returned channel
// is closed when the image has loaded, or failed to load.
//
// Flutter: widgets/image.dart precacheImage
func PrecacheImage(provider painting.ImageProvider, bc BuildContext, opts ...PrecacheImageOpts) <-chan struct{} {
	o := firstOpts(opts)
	config := CreateLocalImageConfiguration(bc, o.Size)
	binding := WidgetsBindingOf(bc)
	done := make(chan struct{})
	completed := false
	complete := func() {
		if !completed {
			completed = true
			close(done)
		}
	}
	stream := painting.ImageProviderResolve(binding.Painting(), provider, config)
	var listener *painting.ImageStreamListener
	listener = painting.NewImageStreamListener(
		func(image *painting.ImageInfo, _ bool) {
			complete()
			binding.Scheduler().AddPostFrameCallback(func(time.Duration) {
				if image != nil {
					image.Dispose()
				}
				stream.RemoveListener(listener)
			})
		},
		painting.ImageStreamListenerOpts{OnError: func(err error) {
			complete()
			stream.RemoveListener(listener)
			if o.OnError != nil {
				o.OnError(err)
			} else {
				binding.ErrorReporter()(errors.NewErr(tagImagePrecacheFailed, err))
			}
		}})
	stream.AddListener(listener)
	return done
}

// ImageFrameBuilder builds an image frame. The child is the image, frame is the
// number of the frame (none until the first one) and wasSynchronouslyLoaded says
// whether the image was available at once.
//
// Flutter: widgets/image.dart ImageFrameBuilder
type ImageFrameBuilder func(bc BuildContext, child Widget, frame foundation.Opt[int], wasSynchronouslyLoaded bool) Widget

// ImageLoadingBuilder builds the widget shown while the image loads. The
// loadingProgress is none until the first chunk.
//
// Flutter: widgets/image.dart ImageLoadingBuilder
type ImageLoadingBuilder func(bc BuildContext, child Widget, loadingProgress foundation.Opt[painting.ImageChunkEvent]) Widget

// ImageErrorWidgetBuilder builds the widget shown when the image fails to load.
//
// Flutter: widgets/image.dart ImageErrorWidgetBuilder
type ImageErrorWidgetBuilder func(bc BuildContext, err error) Widget

// ImageOpts are the optional parameters of NewImage and the named constructors.
//
// Flutter: widgets/image.dart Image.new
type ImageOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FrameBuilder builds the widget around each frame of the image.
	FrameBuilder ImageFrameBuilder
	// LoadingBuilder builds the widget shown while the image loads.
	LoadingBuilder ImageLoadingBuilder
	// ErrorBuilder builds the widget shown when the image fails to load.
	ErrorBuilder ImageErrorWidgetBuilder
	// SemanticLabel is the label that describes the image in the semantics tree.
	SemanticLabel foundation.Opt[string]
	// ExcludeFromSemantics is whether to exclude the image from the semantics
	// tree.
	ExcludeFromSemantics bool
	// Width is the width to give the image, if the constraints allow it.
	Width foundation.Opt[float64]
	// Height is the height to give the image, if the constraints allow it.
	Height foundation.Opt[float64]
	// Color is blended with the image using ColorBlendMode.
	Color foundation.Opt[ui.Color]
	// Opacity is the opacity of the image, as an animation or other listenable.
	Opacity foundation.ValueListenable[float64]
	// ColorBlendMode is how Color is blended; the default is srcIn.
	ColorBlendMode ui.BlendMode
	// Fit is how the image is inscribed into the box.
	Fit painting.BoxFit
	// Alignment is how the image is aligned within its bounds; the default is
	// AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// Repeat is how the image is repeated in the area it does not cover; the
	// default is ImageRepeatNoRepeat.
	Repeat painting.ImageRepeat
	// CenterSlice is the center slice for a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// MatchTextDirection is whether the image is flipped in right to left text.
	MatchTextDirection bool
	// GaplessPlayback is whether to keep showing the old image while the new one
	// loads.
	GaplessPlayback bool
	// IsAntiAlias is whether to paint the image with anti-aliasing.
	IsAntiAlias bool
	// FilterQuality is the quality of image sampling; the default is
	// FilterQualityMedium.
	FilterQuality ui.FilterQuality
}

// ImageNetworkOpts are the optional parameters of NewImageNetwork.
//
// Flutter: widgets/image.dart Image.network
type ImageNetworkOpts struct {
	ImageOpts
	// Scale is the scale of the image; the default is 1.
	Scale foundation.Opt[float64]
	// Headers are the HTTP headers of the request.
	Headers map[string]string
	// CacheWidth resizes the image to this width when decoding it.
	CacheWidth foundation.Opt[int]
	// CacheHeight resizes the image to this height when decoding it.
	CacheHeight foundation.Opt[int]
}

// ImageFileOpts are the optional parameters of NewImageFile.
//
// Flutter: widgets/image.dart Image.file
type ImageFileOpts struct {
	ImageOpts
	// Scale is the scale of the image; the default is 1.
	Scale foundation.Opt[float64]
	// CacheWidth resizes the image to this width when decoding it.
	CacheWidth foundation.Opt[int]
	// CacheHeight resizes the image to this height when decoding it.
	CacheHeight foundation.Opt[int]
}

// ImageAssetOpts are the optional parameters of NewImageAsset.
//
// Flutter: widgets/image.dart Image.asset
type ImageAssetOpts struct {
	ImageOpts
	// Bundle is the bundle to load the asset from; the default is the root bundle
	// of the binding.
	Bundle services.AssetBundle
	// Scale is the scale of the image; when set, the asset is not resolved to a
	// variant by the pixel ratio of the device.
	Scale foundation.Opt[float64]
	// Package is the package that holds the asset.
	Package string
	// CacheWidth resizes the image to this width when decoding it.
	CacheWidth foundation.Opt[int]
	// CacheHeight resizes the image to this height when decoding it.
	CacheHeight foundation.Opt[int]
}

// ImageMemoryOpts are the optional parameters of NewImageMemory.
//
// Flutter: widgets/image.dart Image.memory
type ImageMemoryOpts struct {
	ImageOpts
	// Scale is the scale of the image; the default is 1.
	Scale foundation.Opt[float64]
	// CacheWidth resizes the image to this width when decoding it.
	CacheWidth foundation.Opt[int]
	// CacheHeight resizes the image to this height when decoding it.
	CacheHeight foundation.Opt[int]
}

// Image is a widget that displays an image.
//
// Deviation: the image provider is not wrapped in a ScrollAwareImageProvider,
// there is no TickerMode to pause animated images, and web images are not
// supported (SPEC §12.10).
//
// Flutter: widgets/image.dart Image
type Image struct {
	StatefulWidget
	image painting.ImageProvider
	o     ImageOpts
}

// NewImage creates a widget that displays an image from the provider. In debug
// builds it panics with the tag ImageInvalid when the cache width or height is
// not greater than zero.
//
// Flutter: widgets/image.dart Image.new
func NewImage(image painting.ImageProvider, opts ...ImageOpts) *Image {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityMedium
	}
	return &Image{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), image, o}
}

func assertCacheSize(width, height foundation.Opt[int]) {
	if v, ok := width.Get(); ok && v <= 0 && foundation.DebugMode {
		panic(errors.New(tagImageInvalid, "cacheWidth must be greater than zero", "cacheWidth", v))
	}
	if v, ok := height.Get(); ok && v <= 0 && foundation.DebugMode {
		panic(errors.New(tagImageInvalid, "cacheHeight must be greater than zero", "cacheHeight", v))
	}
}

// NewImageNetwork creates a widget that displays the image at the URL.
//
// Flutter: widgets/image.dart Image.network
func NewImageNetwork(src string, opts ...ImageNetworkOpts) *Image {
	o := firstOpts(opts)
	assertCacheSize(o.CacheWidth, o.CacheHeight)
	provider := painting.ResizeImageIfNeeded(o.CacheWidth, o.CacheHeight, painting.NewNetworkImage(src, painting.NetworkImageOpts{Scale: o.Scale, Headers: o.Headers}))
	return NewImage(provider, o.ImageOpts)
}

// NewImageFile creates a widget that displays the image in the file.
//
// Flutter: widgets/image.dart Image.file
func NewImageFile(path string, opts ...ImageFileOpts) *Image {
	o := firstOpts(opts)
	assertCacheSize(o.CacheWidth, o.CacheHeight)
	o.LoadingBuilder = nil
	scale := []float64{}
	if s, ok := o.Scale.Get(); ok {
		scale = append(scale, s)
	}
	return NewImage(painting.ResizeImageIfNeeded(o.CacheWidth, o.CacheHeight, painting.NewFileImage(path, scale...)), o.ImageOpts)
}

// NewImageAsset creates a widget that displays the asset.
//
// Flutter: widgets/image.dart Image.asset
func NewImageAsset(name string, opts ...ImageAssetOpts) *Image {
	o := firstOpts(opts)
	assertCacheSize(o.CacheWidth, o.CacheHeight)
	o.LoadingBuilder = nil
	var provider painting.ImageProvider
	if s, ok := o.Scale.Get(); ok {
		provider = painting.NewExactAssetImage(name, painting.ExactAssetImageOpts{Scale: foundation.Some(s), Bundle: o.Bundle, Package: o.Package})
	} else {
		provider = painting.NewAssetImage(name, painting.AssetImageOpts{Bundle: o.Bundle, Package: o.Package})
	}
	return NewImage(painting.ResizeImageIfNeeded(o.CacheWidth, o.CacheHeight, provider), o.ImageOpts)
}

// NewImageMemory creates a widget that displays the image in the bytes.
//
// Flutter: widgets/image.dart Image.memory
func NewImageMemory(bytes []byte, opts ...ImageMemoryOpts) *Image {
	o := firstOpts(opts)
	assertCacheSize(o.CacheWidth, o.CacheHeight)
	o.LoadingBuilder = nil
	scale := []float64{}
	if s, ok := o.Scale.Get(); ok {
		scale = append(scale, s)
	}
	return NewImage(painting.ResizeImageIfNeeded(o.CacheWidth, o.CacheHeight, painting.NewMemoryImage(bytes, scale...)), o.ImageOpts)
}

// Image is the image provider.
//
// Flutter: widgets/image.dart Image.image
func (w *Image) Image() painting.ImageProvider { return w.image }

// Width is the width the image is given, if any.
//
// Flutter: widgets/image.dart Image.width
func (w *Image) Width() foundation.Opt[float64] { return w.o.Width }

// Height is the height the image is given, if any.
//
// Flutter: widgets/image.dart Image.height
func (w *Image) Height() foundation.Opt[float64] { return w.o.Height }

// Fit is how the image is inscribed into the box.
//
// Flutter: widgets/image.dart Image.fit
func (w *Image) Fit() painting.BoxFit { return w.o.Fit }

// CreateState creates the state.
//
// Flutter: widgets/image.dart Image.createState
func (w *Image) CreateState() State { return &imageState{} }

type imageState struct {
	StateOf[*Image]
	WidgetsBindingObserverBase
	stream          *painting.ImageStream
	imageInfo       *painting.ImageInfo
	loadingProgress foundation.Opt[painting.ImageChunkEvent]
	listening       bool
	invertColors    bool
	frameNumber     foundation.Opt[int]
	wasSynchronous  bool
	lastError       error
	completerHandle *painting.ImageStreamCompleterHandle
	listener        *painting.ImageStreamListener
	isPaused        bool
}

func (s *imageState) InitState() { WidgetsBindingOf(s.Context()).AddObserver(s) }

func (s *imageState) Dispose() {
	if b := MaybeWidgetsBindingOf(s.Context()); b != nil {
		b.RemoveObserver(s)
	}
	s.stopListening(false)
	if s.completerHandle != nil {
		s.completerHandle.Dispose()
		s.completerHandle = nil
	}
	s.replaceImage(nil)
}

func (s *imageState) DidChangeDependencies() {
	s.updateInvertColors()
	s.resolveImage()
	s.isPaused = MediaQueryMaybeDisableAnimationsValue(s.Context())
	if s.isPaused && s.frameNumber.IsSome() {
		s.stopListening(true)
	} else {
		s.listenToStream()
	}
}

// MediaQueryMaybeDisableAnimationsValue is whether animations are disabled by
// the MediaQuery above the context; false when there is none.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeDisableAnimationsOf
func MediaQueryMaybeDisableAnimationsValue(bc BuildContext) bool {
	v, _ := MediaQueryMaybeDisableAnimationsOf(bc)
	return v
}

func (s *imageState) DidUpdateWidget(old *Image) {
	w := s.Widget()
	if s.listening && (w.o.LoadingBuilder == nil) != (old.o.LoadingBuilder == nil) {
		oldListener := s.getListener(false)
		s.stream.AddListener(s.getListener(true))
		s.stream.RemoveListener(oldListener)
	}
	if w.image != old.image {
		s.resolveImage()
		s.listenToStream()
	}
}

func (s *imageState) DidChangeAccessibilityFeatures() {
	s.SetState(func() { s.updateInvertColors() })
}

func (s *imageState) Reassemble() { s.resolveImage() }

func (s *imageState) updateInvertColors() {
	if v, ok := MediaQueryMaybeInvertColorsOf(s.Context()); ok {
		s.invertColors = v
		return
	}
	s.invertColors = WidgetsBindingOf(s.Context()).PlatformDispatcher().AccessibilityFeatures().InvertColors
}

func (s *imageState) resolveImage() {
	w := s.Widget()
	size := foundation.None[ui.Size]()
	if width, ok := w.o.Width.Get(); ok {
		if height, ok := w.o.Height.Get(); ok {
			size = foundation.Some(ui.Size{Width: width, Height: height})
		}
	}
	config := CreateLocalImageConfiguration(s.Context(), size)
	s.updateSourceStream(painting.ImageProviderResolve(config.Binding, w.image, config))
}

func (s *imageState) getListener(recreate bool) *painting.ImageStreamListener {
	if s.listener == nil || recreate {
		s.lastError = nil
		w := s.Widget()
		lo := painting.ImageStreamListenerOpts{ReportErrors: foundation.Some(w.o.ErrorBuilder == nil)}
		if w.o.LoadingBuilder != nil {
			lo.OnChunk = s.handleImageChunk
		}
		if w.o.ErrorBuilder != nil || foundation.DebugMode {
			lo.OnError = func(err error) { s.SetState(func() { s.lastError = err }) }
		}
		s.listener = painting.NewImageStreamListener(s.handleImageFrame, lo)
	}
	return s.listener
}

func (s *imageState) handleImageFrame(info *painting.ImageInfo, synchronous bool) {
	s.SetState(func() {
		s.replaceImage(info)
		s.loadingProgress = foundation.None[painting.ImageChunkEvent]()
		s.lastError = nil
		s.frameNumber = foundation.Some(s.frameNumber.OrElse(-1) + 1)
		s.wasSynchronous = s.wasSynchronous || synchronous
	})
	if s.isPaused {
		s.stopListening(true)
	}
}

func (s *imageState) handleImageChunk(event painting.ImageChunkEvent) {
	s.SetState(func() {
		s.loadingProgress = foundation.Some(event)
		s.lastError = nil
	})
}

func (s *imageState) replaceImage(info *painting.ImageInfo) {
	if old := s.imageInfo; old != nil {
		if b := MaybeWidgetsBindingOf(s.Context()); b != nil {
			b.Scheduler().AddPostFrameCallback(func(time.Duration) { old.Dispose() })
		} else {
			old.Dispose()
		}
	}
	s.imageInfo = info
}

func (s *imageState) updateSourceStream(newStream *painting.ImageStream) {
	if s.stream != nil && s.stream.Key() == newStream.Key() {
		return
	}
	if s.listening {
		s.stream.RemoveListener(s.getListener(false))
	}
	if !s.Widget().o.GaplessPlayback {
		s.SetState(func() { s.replaceImage(nil) })
	}
	s.SetState(func() {
		s.loadingProgress = foundation.None[painting.ImageChunkEvent]()
		s.frameNumber = foundation.None[int]()
		s.wasSynchronous = false
	})
	s.stream = newStream
	if s.listening {
		s.stream.AddListener(s.getListener(false))
	}
}

func (s *imageState) listenToStream() {
	if s.listening {
		return
	}
	s.listening = true
	s.stream.AddListener(s.getListener(false))
	if s.completerHandle != nil {
		s.completerHandle.Dispose()
		s.completerHandle = nil
	}
}

func (s *imageState) stopListening(keepStreamAlive bool) {
	if !s.listening {
		return
	}
	if keepStreamAlive && s.completerHandle == nil && s.stream.Completer() != nil {
		s.completerHandle = s.stream.Completer().KeepAlive()
	}
	if s.stream.Completer() != nil && s.Widget().o.ErrorBuilder != nil {
		// Without a listener the error would be reported as unhandled, but the
		// error builder handles it.
		s.stream.Completer().AddEphemeralErrorListener(func(error) {})
	}
	s.stream.RemoveListener(s.getListener(false))
	s.listening = false
}

func (s *imageState) debugErrorWidget(err error) Widget {
	return NewStack([]Widget{
		NewPositionedFill(NewPlaceholder(nil, PlaceholderOpts{Color: foundation.Some(ui.NewColor(0xCF8D021F))})),
		NewPadding(painting.NewEdgeInsetsAll(4), NewFittedBox(NewText(err.Error(), TextOpts{
			TextAlign: ui.TextAlignCenter, TextDirection: ui.TextDirectionLTR,
			Style: painting.NewTextStyle(painting.TextStyleOpts{Shadows: []ui.Shadow{{BlurRadius: 1}}})}))),
	}, StackOpts{Alignment: painting.AlignmentCenter})
}

func (s *imageState) Build(bc BuildContext) Widget {
	w := s.Widget()
	o := w.o
	if s.lastError != nil {
		if o.ErrorBuilder != nil {
			return o.ErrorBuilder(bc, s.lastError)
		}
		if foundation.DebugMode {
			return s.debugErrorWidget(s.lastError)
		}
	}
	raw := RawImageOpts{
		Width: o.Width, Height: o.Height, Scale: 1, Color: o.Color, Opacity: o.Opacity, ColorBlendMode: o.ColorBlendMode, Fit: o.Fit,
		Alignment: o.Alignment, Repeat: o.Repeat, CenterSlice: o.CenterSlice, MatchTextDirection: o.MatchTextDirection,
		InvertColors: s.invertColors, IsAntiAlias: o.IsAntiAlias, FilterQuality: o.FilterQuality,
	}
	var image *ui.Image
	if s.imageInfo != nil {
		image = s.imageInfo.Image
		raw.DebugImageLabel = s.imageInfo.DebugLabel
		raw.Scale = s.imageInfo.Scale
	}
	var result Widget = NewRawImage(image, raw)
	if !o.ExcludeFromSemantics {
		result = NewSemantics(result, SemanticsOpts{Container: o.SemanticLabel.IsSome(), Image: foundation.Some(true), Label: foundation.Some(o.SemanticLabel.OrElse(""))})
	}
	if o.FrameBuilder != nil {
		result = o.FrameBuilder(bc, result, s.frameNumber, s.wasSynchronous)
	}
	if o.LoadingBuilder != nil {
		result = o.LoadingBuilder(bc, result, s.loadingProgress)
	}
	return result
}

// RawImageOpts are the optional parameters of NewRawImage.
//
// Flutter: widgets/basic.dart RawImage.new
type RawImageOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DebugImageLabel is a description of the image, for debugging.
	DebugImageLabel string
	// Width is the width to give the image, if the constraints allow it.
	Width foundation.Opt[float64]
	// Height is the height to give the image, if the constraints allow it.
	Height foundation.Opt[float64]
	// Scale is the number of image pixels per logical pixel; the default is 1.
	Scale float64
	// Color is blended with the image using ColorBlendMode.
	Color foundation.Opt[ui.Color]
	// Opacity is the opacity of the image.
	Opacity foundation.ValueListenable[float64]
	// ColorBlendMode is how Color is blended; the default is srcIn.
	ColorBlendMode ui.BlendMode
	// Fit is how the image is inscribed into the box.
	Fit painting.BoxFit
	// Alignment is how the image is aligned within its bounds; the default is
	// AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// Repeat is how the image is repeated in the area it does not cover.
	Repeat painting.ImageRepeat
	// CenterSlice is the center slice for a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// MatchTextDirection is whether the image is flipped in right to left text.
	MatchTextDirection bool
	// InvertColors is whether the colors of the image are inverted.
	InvertColors bool
	// FilterQuality is the quality of image sampling; the default is
	// FilterQualityMedium.
	FilterQuality ui.FilterQuality
	// IsAntiAlias is whether to paint the image with anti-aliasing.
	IsAntiAlias bool
}

// RawImage is a widget that displays a dart:ui Image directly.
//
// Flutter: widgets/basic.dart RawImage
type RawImage struct {
	LeafRenderObjectWidget
	image *ui.Image
	o     RawImageOpts
}

// NewRawImage creates the widget; the image may be nil. The widget does not take
// the ownership of the image: the render object holds a clone.
//
// Flutter: widgets/basic.dart RawImage.new
func NewRawImage(image *ui.Image, opts ...RawImageOpts) *RawImage {
	o := firstOpts(opts)
	if o.Scale == 0 {
		o.Scale = 1
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityMedium
	}
	return &RawImage{NewLeafRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), image, o}
}

// Image is the image to display, or nil.
//
// Flutter: widgets/basic.dart RawImage.image
func (w *RawImage) Image() *ui.Image { return w.image }

func (w *RawImage) textDirection(bc BuildContext) ui.TextDirection {
	if _, isAlignment := w.o.Alignment.(painting.Alignment); w.o.MatchTextDirection || !isAlignment {
		return DirectionalityOf(bc)
	}
	return ui.TextDirectionUnspecified
}

func (w *RawImage) cloneImage() *ui.Image {
	if w.image == nil {
		return nil
	}
	return w.image.Clone()
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart RawImage.createRenderObject
func (w *RawImage) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	o := w.o
	return rendering.NewRenderImage(rendering.RenderImageOpts{
		Image: w.cloneImage(), DebugImageLabel: o.DebugImageLabel, Width: o.Width, Height: o.Height, Scale: foundation.Some(o.Scale),
		Color: o.Color, Opacity: o.Opacity, ColorBlendMode: o.ColorBlendMode, Fit: o.Fit, Alignment: o.Alignment, Repeat: o.Repeat,
		CenterSlice: o.CenterSlice, MatchTextDirection: o.MatchTextDirection, TextDirection: w.textDirection(bc),
		InvertColors: o.InvertColors, IsAntiAlias: o.IsAntiAlias, FilterQuality: o.FilterQuality,
	})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart RawImage.updateRenderObject
func (w *RawImage) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	o := w.o
	r := renderObject.(*rendering.RenderImage)
	r.SetImage(w.cloneImage())
	r.SetDebugImageLabel(o.DebugImageLabel)
	r.SetWidth(o.Width)
	r.SetHeight(o.Height)
	r.SetScale(o.Scale)
	r.SetColor(o.Color)
	r.SetOpacity(o.Opacity)
	r.SetColorBlendMode(o.ColorBlendMode)
	r.SetFit(o.Fit)
	r.SetAlignment(o.Alignment)
	r.SetRepeat(o.Repeat)
	r.SetCenterSlice(o.CenterSlice)
	r.SetMatchTextDirection(o.MatchTextDirection)
	r.SetTextDirection(w.textDirection(bc))
	r.SetInvertColors(o.InvertColors)
	r.SetIsAntiAlias(o.IsAntiAlias)
	r.SetFilterQuality(o.FilterQuality)
}

// DidUnmountRenderObject lets go of the image.
//
// Flutter: widgets/basic.dart RawImage.didUnmountRenderObject
func (w *RawImage) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderImage).SetImage(nil)
}

// ImageIconOpts are the optional parameters of NewImageIcon.
//
// Flutter: widgets/image_icon.dart ImageIcon.new
type ImageIconOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Size is the size of the icon; the default comes from the icon theme.
	Size foundation.Opt[float64]
	// Color is the color of the icon; the default comes from the icon theme.
	Color foundation.Opt[ui.Color]
	// SemanticLabel is the label that describes the icon in the semantics tree.
	SemanticLabel foundation.Opt[string]
	// UseOriginalColors is whether to show the image in its own colors.
	UseOriginalColors bool
}

// ImageIcon is an icon that is drawn from an image.
//
// Flutter: widgets/image_icon.dart ImageIcon
type ImageIcon struct {
	StatelessWidget
	image painting.ImageProvider
	o     ImageIconOpts
}

// NewImageIcon creates the icon; the image may be nil. In debug builds it panics
// with the tag ImageIconColorAndOriginal when there is a color and
// UseOriginalColors.
//
// Flutter: widgets/image_icon.dart ImageIcon.new
func NewImageIcon(image painting.ImageProvider, opts ...ImageIconOpts) *ImageIcon {
	o := firstOpts(opts)
	if foundation.DebugMode && o.UseOriginalColors && o.Color.IsSome() {
		panic(errors.New(tagImageIconColorAndOriginal, "cannot provide a color while useOriginalColors is true; to use a specific color, set useOriginalColors to false or omit it"))
	}
	return &ImageIcon{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), image, o}
}

// Build makes the image.
//
// Flutter: widgets/image_icon.dart ImageIcon.build
func (w *ImageIcon) Build(bc BuildContext) Widget {
	iconTheme := IconThemeOf(bc)
	sizeOpt := orSome(w.o.Size, iconTheme.size)
	if w.image == nil {
		return NewSemantics(NewSizedBox(nil, SizedBoxOpts{Width: sizeOpt, Height: sizeOpt}), SemanticsOpts{Label: w.o.SemanticLabel})
	}
	color := w.o.Color.OrElse(iconTheme.color.OrElse(ui.NewColor(0xFF000000)))
	if opacity, ok := iconTheme.Opacity().Get(); ok && opacity != 1 {
		color = color.WithOpacity(color.Opacity() * opacity)
	}
	imageColor := foundation.Some(color)
	if w.o.UseOriginalColors {
		imageColor = foundation.None[ui.Color]()
	}
	return NewSemantics(NewImage(w.image, ImageOpts{Width: sizeOpt, Height: sizeOpt, Color: imageColor, Fit: painting.BoxFitScaleDown,
		ExcludeFromSemantics: true}), SemanticsOpts{Label: w.o.SemanticLabel})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagImageIconColorAndOriginal = "ImageIconColorAndOriginal"
	tagImageInvalid              = "ImageInvalid"
	tagImagePrecacheFailed       = "ImagePrecacheFailed"
)
