package svg

import (
	"context"
	"math"
	"sync"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SvgErrorBuilder builds the widget that SvgPicture shows when its SVG could not be loaded or parsed. The error is
// the one that is reported to the app's error reporter too (SvgLoadFailed, SvgParseFailed, SvgNotSvg or
// SvgInvalidValue).
//
// flutter_svg: no counterpart (vector_graphics VectorGraphic.errorBuilder)
type SvgErrorBuilder func(bc widgets.BuildContext, err error) widgets.Widget

// SvgPictureOpts are the optional parameters of the constructors of SvgPicture.
//
// flutter_svg: svg.dart SvgPicture.new
type SvgPictureOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Width is the width of the picture. If only one of Width and Height is set, the other keeps the aspect ratio of
	// the picture; with neither, the size of the picture is the size of the widget, within the constraints.
	Width foundation.Opt[float64]
	// Height is the height of the picture.
	Height foundation.Opt[float64]
	// Fit is how the picture is inscribed into the space the layout gives it; the default is BoxFitContain.
	Fit painting.BoxFit
	// Alignment is how the picture is aligned in that space; the default is AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// MatchTextDirection flips the picture horizontally in right to left contexts.
	MatchTextDirection bool
	// AllowDrawingOutsideViewBox lets the picture draw outside its viewBox; by default it is clipped to it. Use it
	// with caution: the picture may draw beyond the space it was given.
	AllowDrawingOutsideViewBox bool
	// ClipBehavior is how the picture is clipped to the widget when it is larger than it (Fit cover, or
	// an alignment that crops); the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// ColorFilter is applied to the picture as a whole.
	ColorFilter *ui.ColorFilter
	// Theme gives currentColor, em and ex to the document; the default is that of the closest DefaultSvgTheme, else the
	// default theme.
	Theme foundation.Opt[SvgTheme]
	// PlaceholderBuilder builds the widget shown while the SVG loads; the default is an empty box of Width and Height.
	PlaceholderBuilder widgets.WidgetBuilder
	// ErrorBuilder builds the widget shown when the SVG could not be loaded or parsed; the default is an empty box
	// of Width and Height. The error is reported to the app's error reporter in both cases.
	ErrorBuilder SvgErrorBuilder
	// SemanticsLabel is the label that describes the picture to screen readers; the default is the title element of the
	// document, if it has one.
	SemanticsLabel foundation.Opt[string]
	// ExcludeFromSemantics leaves the picture out of the semantics tree, and SemanticsLabel with it.
	ExcludeFromSemantics bool
}

// SvgPictureAssetOpts are the optional parameters of NewSvgPictureAsset.
//
// flutter_svg: svg.dart SvgPicture.asset
type SvgPictureAssetOpts struct {
	SvgPictureOpts
	// Bundle is the bundle to load the asset from; the default is the root bundle of the binding.
	Bundle services.AssetBundle
	// Package is the package that holds the asset; the key is then "packages/<Package>/<assetName>".
	Package string
}

// SvgPictureNetworkOpts are the optional parameters of NewSvgPictureNetwork.
//
// flutter_svg: svg.dart SvgPicture.network
type SvgPictureNetworkOpts struct {
	SvgPictureOpts
	// Headers are the HTTP headers of the request.
	Headers map[string]string
}

// SvgPicture is a widget that draws an SVG document.
//
// The size of the widget comes from Width and Height, the constraints and the size of the document (its width and
// height, or its viewBox), as flutter_svg's does. The document is loaded and parsed once for all the widgets with the
// same source and theme (CacheOf), on a goroutine that ends when the widget leaves the tree; PlaceholderBuilder is
// shown meanwhile. A document that is in the cache already is drawn in the first frame.
//
// The picture is recorded once for a document and scaled to the box of the widget. A document with a pattern, a filter
// or a stroke that does not scale (Document.ScaleDependent) is recorded at the scale it is drawn at: the device pixel
// ratio of the view times the scale of the fit, rounded up to a multiple of 1/8, and again when that changes.
//
// Deviation: flutter_svg turns the SVG into vector_graphics's binary format on an isolate; this parses the SVG into a
// tree on a goroutine and records it into a ui.Picture (SPEC §26, §12.10).
//
// flutter_svg: svg.dart SvgPicture
type SvgPicture struct {
	widgets.StatefulWidget
	src source
	o   SvgPictureOpts
}

func newSvgPicture(src source, o SvgPictureOpts) *SvgPicture {
	if o.Fit == painting.BoxFitUnspecified {
		o.Fit = painting.BoxFitContain
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &SvgPicture{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), src: src, o: o}
}

// NewSvgPictureAsset creates a widget that draws the SVG asset of a bundle: the one of the options or the binding's.
// The asset is loaded on a goroutine.
//
// flutter_svg: svg.dart SvgPicture.asset
func NewSvgPictureAsset(assetName string, o ...SvgPictureAssetOpts) *SvgPicture {
	ao := opts.First(o)
	return newSvgPicture(&assetSource{name: assetName, pkg: ao.Package, bundle: ao.Bundle}, ao.SvgPictureOpts)
}

// NewSvgPictureString creates a widget that draws the SVG in the string. It is parsed when the widget is first built.
//
// flutter_svg: svg.dart SvgPicture.string
func NewSvgPictureString(source string, o ...SvgPictureOpts) *SvgPicture {
	return newSvgPicture(&stringSource{source}, opts.First(o))
}

// NewSvgPictureMemory creates a widget that draws the SVG in the bytes (text, or compressed with gzip). It is parsed
// when the widget is first built.
//
// flutter_svg: svg.dart SvgPicture.memory
func NewSvgPictureMemory(bytes []byte, o ...SvgPictureOpts) *SvgPicture {
	return newSvgPicture(&memorySource{b: bytes}, opts.First(o))
}

// NewSvgPictureFile creates a widget that draws the SVG in the file. The file is read on a goroutine. On the web and
// on Android a path is not a file the app can read, and the widget shows its ErrorBuilder.
//
// flutter_svg: svg.dart SvgPicture.file
func NewSvgPictureFile(path string, o ...SvgPictureOpts) *SvgPicture {
	return newSvgPicture(&fileSource{path}, opts.First(o))
}

// NewSvgPictureNetwork creates a widget that draws the SVG at the URL. It is requested on a goroutine with the HTTP
// client of the binding and the headers. All network pictures are cached whatever the headers of the answer say.
//
// flutter_svg: svg.dart SvgPicture.network
func NewSvgPictureNetwork(url string, o ...SvgPictureNetworkOpts) *SvgPicture {
	no := opts.First(o)
	return newSvgPicture(&networkSource{url: url, headers: no.Headers}, no.SvgPictureOpts)
}

// Width is the width the picture is given, if any.
//
// flutter_svg: svg.dart SvgPicture.width
func (w *SvgPicture) Width() foundation.Opt[float64] { return w.o.Width }

// Height is the height the picture is given, if any.
//
// flutter_svg: svg.dart SvgPicture.height
func (w *SvgPicture) Height() foundation.Opt[float64] { return w.o.Height }

// Fit is how the picture is inscribed into the box.
//
// flutter_svg: svg.dart SvgPicture.fit
func (w *SvgPicture) Fit() painting.BoxFit { return w.o.Fit }

// Alignment is how the picture is aligned in the box.
//
// flutter_svg: svg.dart SvgPicture.alignment
func (w *SvgPicture) Alignment() painting.AlignmentGeometry { return w.o.Alignment }

// String describes the source of the picture, as flutter_svg's loaders do ("SvgAssetLoader(icons/home.svg)").
//
// flutter_svg: svg.dart SvgPicture.bytesLoader
func (w *SvgPicture) String() string { return "SvgPicture(" + w.src.String() + ")" }

// CreateState creates the state.
//
// flutter_svg: svg.dart SvgPicture.build
func (w *SvgPicture) CreateState() widgets.State { return &svgPictureState{} }

// entry is a parsed document in the cache, with the pictures recorded from it.
type entry struct {
	doc *Document
	// bytes is the size that the cache counts.
	bytes int

	mu   sync.Mutex
	pics []recorded
}

// recorded is a picture of a document at a scale.
type recorded struct {
	scale float64
	pic   *ui.Picture
}

// maxRecorded is how many pictures at different scales an entry keeps (the oldest is dropped), so that an animation
// of the size of a picture with a filter does not fill the memory.
const maxRecorded = 4

// scaleStep is the step that the scale a picture is recorded at is rounded up to, and maxScale the largest scale.
const (
	scaleStep = 1.0 / 8
	maxScale  = 32.0
)

// picture returns the picture of the document for a drawing at the scale (device pixels to a unit of the document),
// recording it if there is none. A document that is not scale dependent has one picture, whatever the scale.
func (e *entry) picture(scale float64, text ui.ParagraphFactory) *ui.Picture {
	if !e.doc.d.ScaleDependent() {
		scale = 1
	} else {
		scale = math.Min(maxScale, math.Max(scaleStep, math.Ceil(scale/scaleStep)*scaleStep))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range e.pics {
		if r.scale == scale {
			return r.pic
		}
	}
	pic := e.doc.PictureWith(PictureOpts{Scale: scale, Text: text})
	if len(e.pics) >= maxRecorded {
		e.pics = e.pics[1:]
	}
	e.pics = append(e.pics, recorded{scale, pic})
	return pic
}

type svgPictureState struct {
	widgets.StateOf[*SvgPicture]
	cache *Cache
	key   cacheKey
	have  bool
	entry *entry
	err   error
	// cancel ends the load that is going on; gen numbers the loads, so that the answer of an old one is dropped.
	cancel context.CancelFunc
	gen    int
}

func (s *svgPictureState) Dispose() {
	s.stop()
}

// stop ends the load that is going on and drops its answer.
func (s *svgPictureState) stop() {
	s.gen++
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *svgPictureState) DidChangeDependencies() { s.reload() }

func (s *svgPictureState) DidUpdateWidget(old *SvgPicture) {
	w := s.Widget()
	if !old.src.equal(w.src) || old.o.Theme != w.o.Theme {
		s.reload()
	}
}

// theme is the theme the document is parsed with: the option, else the closest DefaultSvgTheme, else the default.
func (s *svgPictureState) theme() SvgTheme {
	if t, ok := s.Widget().o.Theme.Get(); ok {
		return t.withDefaults()
	}
	if d, ok := DefaultSvgThemeOf(s.Context()); ok {
		return d.theme.withDefaults()
	}
	return SvgTheme{}.withDefaults()
}

// reload finds the document for the source and the theme as they are now: in the cache, or by parsing the source
// now (a string, bytes), or on a goroutine.
func (s *svgPictureState) reload() {
	w := s.Widget()
	bc := s.Context()
	binding := widgets.WidgetsBindingOf(bc)
	env := &loadEnv{bundle: binding.Painting().AssetBundle(), client: binding.Painting().HTTPClient()}
	theme := s.theme()
	cache := CacheOf(bc)
	key := cacheKey{source: w.src.cacheKey(env), theme: theme}
	if s.have && s.cache == cache && s.key == key {
		return
	}
	s.stop()
	s.cache, s.key, s.have = cache, key, true
	s.entry, s.err = nil, nil
	if e, ok := cache.get(key); ok {
		s.entry = e
		return
	}
	src := w.src
	if src.immediate() {
		e, err := cache.load(context.Background(), key, func(ctx context.Context) (*entry, error) {
			data, err := src.load(ctx, env)
			if err != nil {
				return nil, err
			}
			return parseEntry(data, theme)
		})
		s.finish(s.gen, e, err)
		return
	}
	loadCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	gen := s.gen
	s.Go(func(ctx context.Context) {
		// The load ends when the state is disposed (ctx) or replaced (cancel).
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
		e, err := cache.load(loadCtx, key, func(ctx context.Context) (*entry, error) {
			data, err := src.load(ctx, env)
			if err != nil {
				return nil, err
			}
			return parseEntry(data, theme)
		})
		if e == nil && err == nil {
			return
		}
		s.SetState(func() { s.finish(gen, e, err) })
	})
}

// finish takes the answer of a load, unless another one has begun, and reports the error.
func (s *svgPictureState) finish(gen int, e *entry, err error) {
	if gen != s.gen {
		return
	}
	s.cancel = nil
	s.entry, s.err = e, err
	if err != nil {
		widgets.WidgetsBindingOf(s.Context()).ErrorReporter()(err)
	}
}

func (s *svgPictureState) Build(bc widgets.BuildContext) widgets.Widget {
	o := s.Widget().o
	empty := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: o.Width, Height: o.Height})
	}
	var child widgets.Widget
	label := o.SemanticsLabel
	switch {
	case s.err != nil:
		if o.ErrorBuilder != nil {
			child = o.ErrorBuilder(bc, s.err)
		} else {
			child = empty()
		}
	case s.entry == nil:
		if o.PlaceholderBuilder != nil {
			child = o.PlaceholderBuilder(bc)
		} else {
			child = empty()
		}
	default:
		child = s.picture(bc)
		if label.IsNone() && s.entry.doc.Title() != "" {
			label = foundation.Some(s.entry.doc.Title())
		}
	}
	if !o.ExcludeFromSemantics {
		child = widgets.NewSemantics(child, widgets.SemanticsOpts{Container: label.IsSome(), Image: foundation.Some(true),
			Label: foundation.Some(label.OrElse(""))})
	}
	return child
}

// picture is the widget of the loaded document: a box of the size flutter_svg works out, and the painter in it.
func (s *svgPictureState) picture(bc widgets.BuildContext) widgets.Widget {
	o := s.Widget().o
	size := s.entry.doc.Size()
	width, height := o.Width, o.Height
	switch {
	case size.IsEmpty():
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: width, Height: height})
	case width.IsNone() && height.IsNone():
		width, height = foundation.Some(size.Width), foundation.Some(size.Height)
	case width.IsNone():
		width = foundation.Some(height.Value() / size.Height * size.Width)
	case height.IsNone():
		height = foundation.Some(width.Value() / size.Width * size.Height)
	}
	dpr, ok := widgets.MediaQueryMaybeDevicePixelRatioOf(bc)
	if !ok || !(dpr > 0) {
		dpr = 1
	}
	direction, _ := widgets.MaybeDirectionalityOf(bc)
	p := &picturePainter{
		CustomPainterBase: rendering.NewCustomPainterBase(nil),
		entry:             s.entry, fit: o.Fit, alignment: o.Alignment, direction: direction, clip: o.ClipBehavior,
		flip: o.MatchTextDirection, outside: o.AllowDrawingOutsideViewBox, filter: o.ColorFilter, dpr: dpr,
		text: widgets.WidgetsBindingOf(bc).Painting().TextEngine(),
	}
	return widgets.NewSizedBox(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: p, Size: size}),
		widgets.SizedBoxOpts{Width: width, Height: height})
}

// picturePainter draws the picture of a document in a box: it fits it as FittedBox does, flips it for
// MatchTextDirection, clips it, and applies the color filter.
type picturePainter struct {
	rendering.CustomPainterBase
	entry     *entry
	fit       painting.BoxFit
	alignment painting.AlignmentGeometry
	direction ui.TextDirection
	clip      ui.Clip
	flip      bool
	outside   bool
	filter    *ui.ColorFilter
	dpr       float64
	text      ui.ParagraphFactory
}

// Paint draws the picture in the box.
//
// flutter_svg: no counterpart (vector_graphics RenderVectorGraphic.paint)
func (p *picturePainter) Paint(canvas *ui.Canvas, size ui.Size) {
	doc := p.entry.doc.Size()
	sizes := painting.ApplyBoxFit(p.fit, doc, size)
	if size.IsEmpty() || doc.IsEmpty() || sizes.Source.IsEmpty() || sizes.Destination.IsEmpty() {
		return
	}
	direction := p.direction
	if direction == ui.TextDirectionUnspecified {
		direction = ui.TextDirectionLTR
	}
	align := p.alignment.Resolve(direction)
	scaleX := sizes.Destination.Width / sizes.Source.Width
	scaleY := sizes.Destination.Height / sizes.Source.Height
	source := align.Inscribe(sizes.Source, ui.OffsetZero.And(doc))
	dest := align.Inscribe(sizes.Destination, ui.OffsetZero.And(size))
	overflow := source.Width() < doc.Width || source.Height() < doc.Height

	// The picture is recorded at the scale it is drawn at, if that matters to it.
	pic := p.entry.picture(p.dpr*math.Max(scaleX, scaleY), p.text)

	box := ui.OffsetZero.And(size)
	canvas.Save()
	defer canvas.Restore()
	if overflow && p.clip != ui.ClipNone {
		antiAlias := p.clip != ui.ClipHardEdge
		canvas.ClipRect(box, ui.ClipRectOpts{DoAntiAlias: &antiAlias})
		if p.clip == ui.ClipAntiAliasWithSaveLayer {
			canvas.SaveLayer(&box, ui.NewPaint())
			defer canvas.Restore()
		}
	}
	if p.flip && direction == ui.TextDirectionRTL {
		x := (dest.Left + dest.Right) / 2
		canvas.Translate(x, 0)
		canvas.Scale(-1, 1)
		canvas.Translate(-x, 0)
	}
	canvas.Translate(dest.Left, dest.Top)
	canvas.Scale(scaleX, scaleY)
	canvas.Translate(-source.Left, -source.Top)
	var bounds *ui.Rect
	if !p.outside {
		r := ui.OffsetZero.And(doc)
		bounds = &r
		canvas.ClipRect(r)
	}
	if p.filter != nil {
		paint := ui.NewPaint()
		paint.SetColorFilter(p.filter)
		canvas.SaveLayer(bounds, paint)
		canvas.DrawPicture(pic)
		canvas.Restore()
		return
	}
	canvas.DrawPicture(pic)
}

// ShouldRepaint is true when the painter draws something else than the old one.
//
// flutter_svg: no counterpart
func (p *picturePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*picturePainter)
	return !ok || o.entry != p.entry || o.fit != p.fit || o.alignment != p.alignment || o.direction != p.direction ||
		o.clip != p.clip || o.flip != p.flip || o.outside != p.outside || o.filter != p.filter || o.dpr != p.dpr ||
		o.text != p.text
}
