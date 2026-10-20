// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"strconv"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ImageInfo is a handle to an image resource and its scale. It owns a reference
// to the image: call Dispose when done.
//
// Flutter: painting/image_stream.dart ImageInfo
type ImageInfo struct {
	// Image is the raw image pixels.
	//
	// Flutter: painting/image_stream.dart ImageInfo.image
	Image *ui.Image
	// Scale is the linear scale factor for drawing this image at its intended
	// size: the image is intended for a device pixel ratio of Scale.
	//
	// Flutter: painting/image_stream.dart ImageInfo.scale
	Scale float64
	// DebugLabel is a string used for debugging; it is empty for none.
	//
	// Flutter: painting/image_stream.dart ImageInfo.debugLabel
	DebugLabel string
}

// ImageInfoOpts are the optional parameters of NewImageInfo.
//
// Flutter: painting/image_stream.dart ImageInfo.new
type ImageInfoOpts struct {
	// Scale defaults to 1.
	Scale foundation.Opt[float64]
	// DebugLabel is a string used for debugging.
	DebugLabel string
}

// NewImageInfo creates an image info that owns the reference to the image.
//
// Flutter: painting/image_stream.dart ImageInfo.new
func NewImageInfo(image *ui.Image, opts ...ImageInfoOpts) *ImageInfo {
	o := firstOpts(opts)
	return &ImageInfo{Image: image, Scale: o.Scale.OrElse(1.0), DebugLabel: o.DebugLabel}
}

// Clone returns a new handle with its own reference to the image.
//
// Flutter: painting/image_stream.dart ImageInfo.clone
func (i *ImageInfo) Clone() *ImageInfo {
	return &ImageInfo{Image: i.Image.Clone(), Scale: i.Scale, DebugLabel: i.DebugLabel}
}

// IsCloneOf reports whether the other handle is a clone of this one.
//
// Flutter: painting/image_stream.dart ImageInfo.isCloneOf
func (i *ImageInfo) IsCloneOf(other *ImageInfo) bool {
	return other.Image.IsCloneOf(i.Image) && other.Scale == i.Scale && other.DebugLabel == i.DebugLabel
}

// SizeBytes is the number of bytes of memory the image takes, as the cache
// counts it: width times height times four.
//
// Flutter: painting/image_stream.dart ImageInfo.sizeBytes
func (i *ImageInfo) SizeBytes() int { return i.Image.Height() * i.Image.Width() * 4 }

// Dispose releases the reference to the image.
//
// Flutter: painting/image_stream.dart ImageInfo.dispose
func (i *ImageInfo) Dispose() { i.Image.Dispose() }

// Equal is Dart's == on the info: the same image, scale and label.
//
// Flutter: painting/image_stream.dart ImageInfo.operator ==
func (i *ImageInfo) Equal(other *ImageInfo) bool {
	if i == other {
		return true
	}
	if i == nil || other == nil {
		return false
	}
	return other.Image == i.Image && other.Scale == i.Scale && other.DebugLabel == i.DebugLabel
}

// String describes the info.
//
// Flutter: painting/image_stream.dart ImageInfo.toString
func (i *ImageInfo) String() string {
	label := ""
	if i.DebugLabel != "" {
		label = i.DebugLabel + " "
	}
	return label + i.Image.String() + " @ " + foundation.DebugFormatDouble(foundation.Some(i.Scale)) + "x"
}

// ImageListener is called when an image is available.
//
// Flutter: painting/image_stream.dart ImageListener
type ImageListener func(image *ImageInfo, synchronousCall bool)

// ImageChunkListener is called when a chunk of an image is loaded.
//
// Flutter: painting/image_stream.dart ImageChunkListener
type ImageChunkListener func(event ImageChunkEvent)

// ImageErrorListener is called when loading an image failed.
//
// Flutter: painting/image_stream.dart ImageErrorListener
type ImageErrorListener func(err error)

// ImageStreamListener receives the events of an image stream. Compare listeners
// by pointer: the same *ImageStreamListener is used to add and remove.
//
// Flutter: painting/image_stream.dart ImageStreamListener
type ImageStreamListener struct {
	// OnImage is called when the image is loaded; the listener owns the info and
	// must dispose it.
	OnImage ImageListener
	// OnChunk is called when a chunk of the image is loaded; nil for none.
	OnChunk ImageChunkListener
	// OnError is called when loading the image failed; nil for none.
	OnError ImageErrorListener
	// ReportErrors is false if the listener handles errors itself, even without
	// OnError.
	ReportErrors bool
}

// ImageStreamListenerOpts are the optional parameters of NewImageStreamListener.
//
// Flutter: painting/image_stream.dart ImageStreamListener.new
type ImageStreamListenerOpts struct {
	// OnChunk is called when a chunk of the image is loaded.
	OnChunk ImageChunkListener
	// OnError is called when loading the image failed.
	OnError ImageErrorListener
	// ReportErrors defaults to true.
	ReportErrors foundation.Opt[bool]
}

// NewImageStreamListener creates a listener.
//
// Flutter: painting/image_stream.dart ImageStreamListener.new
func NewImageStreamListener(onImage ImageListener, opts ...ImageStreamListenerOpts) *ImageStreamListener {
	o := firstOpts(opts)
	return &ImageStreamListener{OnImage: onImage, OnChunk: o.OnChunk, OnError: o.OnError, ReportErrors: o.ReportErrors.OrElse(true)}
}

// ImageChunkEvent is an immutable notification of image bytes that have been
// incrementally loaded.
//
// Flutter: painting/image_stream.dart ImageChunkEvent
type ImageChunkEvent struct {
	// CumulativeBytesLoaded is the number of bytes that have been received.
	CumulativeBytesLoaded int
	// ExpectedTotalBytes is the number of bytes in the whole image, if known.
	ExpectedTotalBytes foundation.Opt[int]
}

// NewImageChunkEvent creates the event. In debug builds negative numbers panic
// with the tag ImageChunkEventInvalid.
//
// Flutter: painting/image_stream.dart ImageChunkEvent.new
func NewImageChunkEvent(cumulativeBytesLoaded int, expectedTotalBytes foundation.Opt[int]) ImageChunkEvent {
	if foundation.DebugMode && (cumulativeBytesLoaded < 0 || (expectedTotalBytes.IsSome() && expectedTotalBytes.Value() < 0)) {
		panic(errors.New("ImageChunkEventInvalid", "the byte counts of an image chunk event must not be negative",
			"cumulativeBytesLoaded", cumulativeBytesLoaded))
	}
	return ImageChunkEvent{cumulativeBytesLoaded, expectedTotalBytes}
}

// DebugFillProperties adds the byte counts.
//
// Flutter: painting/image_stream.dart ImageChunkEvent.debugFillProperties
func (e ImageChunkEvent) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewIntProperty("cumulativeBytesLoaded", foundation.Some(e.CumulativeBytesLoaded)))
	p.Add(foundation.NewIntProperty("expectedTotalBytes", e.ExpectedTotalBytes))
}

// ImageStream is a handle to an image resource that may not be available yet:
// the completer gets set when the image is being loaded, and listeners added
// before are moved to it.
//
// Flutter: painting/image_stream.dart ImageStream
type ImageStream struct {
	completer *ImageStreamCompleter
	listeners []*ImageStreamListener
}

// NewImageStream creates a stream with no completer.
//
// Flutter: painting/image_stream.dart ImageStream.new
func NewImageStream() *ImageStream { return &ImageStream{} }

// Completer is the completer that has been assigned to this stream, or nil.
//
// Flutter: painting/image_stream.dart ImageStream.completer
func (s *ImageStream) Completer() *ImageStreamCompleter { return s.completer }

// SetCompleter assigns the completer and moves the listeners to it. In debug
// builds setting it twice panics with the tag ImageStreamCompleterSet.
//
// Flutter: painting/image_stream.dart ImageStream.setCompleter
func (s *ImageStream) SetCompleter(value *ImageStreamCompleter) {
	if foundation.DebugMode && s.completer != nil {
		panic(errors.New("ImageStreamCompleterSet", "the completer of an ImageStream can be set only once"))
	}
	s.completer = value
	if s.listeners != nil {
		initial := s.listeners
		s.listeners = nil
		s.completer.addingInitialListeners = true
		for _, l := range initial {
			s.completer.AddListener(l)
		}
		s.completer.addingInitialListeners = false
	}
}

// AddListener adds a listener callback to be called whenever a new image is
// available.
//
// Flutter: painting/image_stream.dart ImageStream.addListener
func (s *ImageStream) AddListener(listener *ImageStreamListener) {
	if s.completer != nil {
		s.completer.AddListener(listener)
		return
	}
	s.listeners = append(s.listeners, listener)
}

// RemoveListener stops the listener from being notified.
//
// Flutter: painting/image_stream.dart ImageStream.removeListener
func (s *ImageStream) RemoveListener(listener *ImageStreamListener) {
	if s.completer != nil {
		s.completer.RemoveListener(listener)
		return
	}
	for i, l := range s.listeners {
		if l == listener {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}

// Key is an object that is unique for the image this stream is for: the
// completer once set, else the stream.
//
// Flutter: painting/image_stream.dart ImageStream.key
func (s *ImageStream) Key() any {
	if s.completer != nil {
		return s.completer
	}
	return s
}

// ToStringShort names the stream.
func (s *ImageStream) ToStringShort() string { return "ImageStream" }

// String describes the stream.
func (s *ImageStream) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the completer and the listeners.
//
// Flutter: painting/image_stream.dart ImageStream.debugFillProperties
func (s *ImageStream) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	ifPresent := ""
	if s.completer != nil {
		ifPresent = s.completer.ToStringShort()
	}
	p.Add(foundation.NewObjectFlagProperty("completer", s.completer, foundation.ObjectFlagPropertyOpts{IfPresent: ifPresent, IfNull: "unresolved"}))
	level := foundation.DiagnosticLevelInfo
	if s.completer != nil {
		level = foundation.DiagnosticLevelHidden
	}
	n := len(s.listeners)
	plural := "s"
	if n == 1 {
		plural = ""
	}
	var listeners any
	if s.listeners != nil {
		listeners = s.listeners
	}
	p.Add(foundation.NewObjectFlagProperty("listeners", listeners, foundation.ObjectFlagPropertyOpts{
		IfPresent: strconv.Itoa(n) + " listener" + plural, IfNull: "no listeners", Level: level}))
	if s.completer != nil {
		s.completer.DebugFillProperties(p)
	}
}

// ImageStreamCompleterHandle keeps an ImageStreamCompleter from being disposed
// while it has no listeners.
//
// Flutter: painting/image_stream.dart ImageStreamCompleterHandle
type ImageStreamCompleterHandle struct{ completer *ImageStreamCompleter }

// Dispose releases the handle; the completer may dispose itself if nothing else
// keeps it alive. In debug builds disposing twice panics with the tag
// ImageStreamCompleterHandleDisposed.
//
// Flutter: painting/image_stream.dart ImageStreamCompleterHandle.dispose
func (h *ImageStreamCompleterHandle) Dispose() {
	if h.completer == nil {
		panic(errors.New("ImageStreamCompleterHandleDisposed", "the handle was already disposed"))
	}
	h.completer.keepAliveHandles--
	h.completer.maybeDispose()
	h.completer = nil
}

// completerBehavior is what a kind of completer adds to the common one: Go has
// no subclasses, so the multi-frame completer plugs in here.
type completerBehavior interface {
	beforeAddListener(c *ImageStreamCompleter)
	afterRemoveListener(c *ImageStreamCompleter)
	afterMaybeDispose(c *ImageStreamCompleter)
}

// ImageStreamCompleter is the interface for the object that manages the
// listeners of an ImageStream and the images it delivers. Create one with
// NewImageStreamCompleter, NewOneFrameImageStreamCompleter or
// NewMultiFrameImageStreamCompleter.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter
type ImageStreamCompleter struct {
	binding *PaintingBinding

	listeners               []*ImageStreamListener
	ephemeralErrorListeners []ImageErrorListener
	currentImage            *ImageInfo
	currentError            *ImageErrorDetails

	// DebugLabel is a description of the image, for debugging.
	//
	// Flutter: painting/image_stream.dart ImageStreamCompleter.debugLabel
	DebugLabel string

	addingInitialListeners bool
	hadErrorListener       bool
	keepAliveHandles       int
	disposed               bool
	onLastListenerRemoved  []func()
	onDisposed             func()
	behavior               completerBehavior
}

// NewImageStreamCompleter creates a completer with no image. Call SetImage when
// the image is ready. The binding may be nil: errors nobody handled are then
// dropped.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.new
func NewImageStreamCompleter(binding *PaintingBinding) *ImageStreamCompleter {
	return &ImageStreamCompleter{binding: binding}
}

// SetOnDisposed sets the function called when the completer disposes itself (a
// subclass's onDisposed).
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.onDisposed
func (c *ImageStreamCompleter) SetOnDisposed(fn func()) { c.onDisposed = fn }

// HasListeners reports whether any listener is subscribed.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.hasListeners
func (c *ImageStreamCompleter) HasListeners() bool { return len(c.listeners) > 0 }

// ListenerCount is the number of listeners.
//
// Flutter: no counterpart
func (c *ImageStreamCompleter) ListenerCount() int { return len(c.listeners) }

// CurrentImage is the most recent image, or nil; the completer owns it.
//
// Flutter: no counterpart
func (c *ImageStreamCompleter) CurrentImage() *ImageInfo { return c.currentImage }

// AddListener adds a listener callback to be called whenever a new image is
// available. If an image is already available, the listener is called at once
// with a clone of it and synchronousCall true.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.addListener
func (c *ImageStreamCompleter) AddListener(listener *ImageStreamListener) {
	if c.behavior != nil {
		c.behavior.beforeAddListener(c)
	}
	c.checkDisposed()
	if !listener.ReportErrors {
		c.hadErrorListener = true
	}
	c.listeners = append(c.listeners, listener)
	if c.currentImage != nil {
		c.callListener("by a synchronously-called image listener", func() {
			listener.OnImage(c.currentImage.Clone(), !c.addingInitialListeners)
		})
	}
	if c.currentError != nil && listener.OnError != nil {
		c.callListener("by a synchronously-called image error listener", func() { listener.OnError(c.currentError.Err) })
	}
}

// AddEphemeralErrorListener adds an error listener that is called once if the
// loading fails and there is no image yet; it is dropped as soon as an image
// arrives.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.addEphemeralErrorListener
func (c *ImageStreamCompleter) AddEphemeralErrorListener(listener ImageErrorListener) {
	c.checkDisposed()
	if c.currentError != nil {
		listener(c.currentError.Err)
	} else if c.currentImage == nil {
		c.ephemeralErrorListeners = append(c.ephemeralErrorListeners, listener)
	}
}

// KeepAlive creates a handle that keeps the completer from disposing itself when
// it has no listeners.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.keepAlive
func (c *ImageStreamCompleter) KeepAlive() *ImageStreamCompleterHandle {
	c.checkDisposed()
	c.keepAliveHandles++
	return &ImageStreamCompleterHandle{completer: c}
}

// RemoveListener stops the listener from being notified. When the last listener
// goes, the callbacks added with AddOnLastListenerRemovedCallback run, and the
// completer disposes itself unless a handle keeps it alive.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.removeListener
func (c *ImageStreamCompleter) RemoveListener(listener *ImageStreamListener) {
	c.checkDisposed()
	for i, l := range c.listeners {
		if l == listener {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			break
		}
	}
	if len(c.listeners) == 0 {
		callbacks := append([]func(){}, c.onLastListenerRemoved...)
		for _, cb := range callbacks {
			cb()
		}
		c.onLastListenerRemoved = nil
		c.maybeDispose()
	}
	if c.behavior != nil {
		c.behavior.afterRemoveListener(c)
	}
}

// Disposed reports whether the completer disposed itself.
//
// Flutter: no counterpart
func (c *ImageStreamCompleter) Disposed() bool { return c.disposed }

// MaybeDispose disposes the completer if it has no listeners and no handles.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.maybeDispose
func (c *ImageStreamCompleter) MaybeDispose() { c.maybeDispose() }

func (c *ImageStreamCompleter) maybeDispose() {
	if c.disposed || len(c.listeners) > 0 || c.keepAliveHandles != 0 {
		return
	}
	c.ephemeralErrorListeners = nil
	if c.currentImage != nil {
		c.currentImage.Dispose()
	}
	c.currentImage = nil
	c.disposed = true
	if c.onDisposed != nil {
		c.onDisposed()
	}
	if c.behavior != nil {
		c.behavior.afterMaybeDispose(c)
	}
}

func (c *ImageStreamCompleter) checkDisposed() {
	if c.disposed {
		panic(errors.New("ImageStreamDisposed", "Stream has been disposed. An ImageStream is considered disposed once at least one "+
			"listener has been added and subsequently all listeners have been removed and no handles are outstanding from the "+
			"keepAlive method. To resolve this error, maintain at least one listener on the stream, or create an "+
			"ImageStreamCompleterHandle from the keepAlive method, or create a new stream for the image."))
	}
}

// AddOnLastListenerRemovedCallback adds a callback called once when the last
// listener is removed. Compare callbacks by the index returned.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.addOnLastListenerRemovedCallback
func (c *ImageStreamCompleter) AddOnLastListenerRemovedCallback(callback func()) (remove func()) {
	c.checkDisposed()
	c.onLastListenerRemoved = append(c.onLastListenerRemoved, callback)
	index := len(c.onLastListenerRemoved) - 1
	return func() {
		c.checkDisposed()
		if index < len(c.onLastListenerRemoved) {
			c.onLastListenerRemoved[index] = func() {}
		}
	}
}

// SetImage calls every listener with a clone of the image, and keeps the image
// for the listeners that come later. The completer owns the image.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.setImage
func (c *ImageStreamCompleter) SetImage(image *ImageInfo) {
	c.checkDisposed()
	if c.currentImage != nil {
		c.currentImage.Dispose()
	}
	c.currentImage = image
	c.ephemeralErrorListeners = nil
	if len(c.listeners) == 0 {
		return
	}
	for _, l := range append([]*ImageStreamListener{}, c.listeners...) {
		c.callListener("by an image listener", func() { l.OnImage(image.Clone(), false) })
	}
}

// callListener runs a listener. It is a recovery point (SPEC §14.2): when the
// binding has an error reporter, a panic in the listener is reported with the
// tag CallbackPanicked and the other listeners are still called.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.setImage
func (c *ImageStreamCompleter) callListener(context string, fn func()) {
	foundation.Guard(c.binding.panicReporter(), foundation.CallbackPanicked, context, func() []any {
		return []any{"library", "image resource service"}
	}, fn)
}

// ReportError tells the error listeners that loading failed. If none handled it
// and no listener opted out of error reports, the binding's reporter gets it.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.reportError
func (c *ImageStreamCompleter) ReportError(details ImageErrorDetails) {
	if details.Library == "" {
		details.Library = "image resource service"
	}
	c.currentError = &details
	var errorListeners []ImageErrorListener
	for _, l := range c.listeners {
		if l.OnError != nil {
			errorListeners = append(errorListeners, l.OnError)
		}
	}
	errorListeners = append(errorListeners, c.ephemeralErrorListeners...)
	c.ephemeralErrorListeners = nil
	handled := false
	for _, l := range errorListeners {
		c.callListener("by an image error listener", func() { l(details.Err) })
		handled = true
	}
	if !handled {
		if c.hadErrorListener {
			return
		}
		c.binding.ReportError(details)
	}
}

// ReportImageChunkEvent tells the chunk listeners that a part of the image
// loaded.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.reportImageChunkEvent
func (c *ImageStreamCompleter) ReportImageChunkEvent(event ImageChunkEvent) {
	c.checkDisposed()
	if !c.HasListeners() {
		return
	}
	var local []ImageChunkListener
	for _, l := range c.listeners {
		if l.OnChunk != nil {
			local = append(local, l.OnChunk)
		}
	}
	for _, l := range local {
		l(event)
	}
}

// ToStringShort names the completer.
func (c *ImageStreamCompleter) ToStringShort() string { return "ImageStreamCompleter" }

// String describes the completer.
func (c *ImageStreamCompleter) String() string { return foundation.DiagnosticableString(c) }

// DebugFillProperties adds the current image, the listeners and the state.
//
// Flutter: painting/image_stream.dart ImageStreamCompleter.debugFillProperties
func (c *ImageStreamCompleter) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("current", c.currentImage,
		foundation.DiagnosticsPropertyOpts{IfNull: "unresolved", ShowName: foundation.Some(false)}))
	plural := func(n int, what string) string {
		if n == 1 {
			return strconv.Itoa(n) + " " + what
		}
		return strconv.Itoa(n) + " " + what + "s"
	}
	p.Add(foundation.NewObjectFlagProperty("listeners", c.listeners,
		foundation.ObjectFlagPropertyOpts{IfPresent: plural(len(c.listeners), "listener")}))
	p.Add(foundation.NewObjectFlagProperty("ephemeralErrorListeners", c.ephemeralErrorListeners,
		foundation.ObjectFlagPropertyOpts{IfPresent: plural(len(c.ephemeralErrorListeners), "ephemeralErrorListener")}))
	p.Add(foundation.NewFlagProperty("disposed", foundation.Some(c.disposed), foundation.FlagPropertyOpts{IfTrue: "<disposed>"}))
}

// ImageResult is the outcome of loading an image: an image info, or an error.
//
// Flutter: dart:async Future<ImageInfo>
type ImageResult struct {
	Info *ImageInfo
	Err  error
}

// CodecResult is the outcome of making a codec: a codec, or an error.
//
// Flutter: dart:async Future<ui.Codec>
type CodecResult struct {
	Codec *ui.Codec
	Err   error
}

// NewOneFrameImageStreamCompleter creates a completer for an image that is
// loaded once. Send the result on the channel, once, from any goroutine; it is
// delivered on the UI thread. An error is reported silently.
//
// Flutter: painting/image_stream.dart OneFrameImageStreamCompleter
func NewOneFrameImageStreamCompleter(binding *PaintingBinding, image <-chan ImageResult, informationCollector func() []foundation.DiagnosticsNode) *ImageStreamCompleter {
	c := NewImageStreamCompleter(binding)
	done := make(chan struct{})
	prev := c.onDisposed
	c.onDisposed = func() {
		close(done)
		if prev != nil {
			prev()
		}
	}
	go func() {
		select {
		case r := <-image:
			binding.Post(func() {
				if c.disposed {
					if r.Info != nil {
						r.Info.Dispose()
					}
					return
				}
				if r.Err != nil {
					c.ReportError(ImageErrorDetails{Err: r.Err, Context: "resolving a single-frame image stream", Silent: true, Information: informationCollector})
					return
				}
				c.SetImage(r.Info)
			})
		case <-done:
		}
	}()
	return c
}

// MultiFrameImageStreamCompleterOpts are the parameters of
// NewMultiFrameImageStreamCompleter.
//
// Flutter: painting/image_stream.dart MultiFrameImageStreamCompleter
type MultiFrameImageStreamCompleterOpts struct {
	// Codec delivers the codec that provides the frames, once. Send it from any
	// goroutine.
	Codec <-chan CodecResult
	// Scale is the scale of the image.
	Scale float64
	// DebugLabel is a description of the image.
	DebugLabel string
	// ChunkEvents delivers the progress of the loading; the channel may be nil.
	ChunkEvents <-chan ImageChunkEvent
	// InformationCollector describes the image provider and the key, for errors.
	InformationCollector func() []foundation.DiagnosticsNode
}

// multiFrame is the state of a MultiFrameImageStreamCompleter.
type multiFrame struct {
	binding              *PaintingBinding
	informationCollector func() []foundation.DiagnosticsNode
	scale                float64

	done          chan struct{}
	codec         *ui.Codec
	nextFrame     *ui.FrameInfo
	shownAt       time.Duration
	frameDuration time.Duration
	hasDuration   bool
	framesEmitted int
	stopTimer     func()
	frameCallback bool
}

// NewMultiFrameImageStreamCompleter creates a completer for an image that may
// have several frames, such as a GIF. It shows each frame for its duration, on
// frame callbacks, while it has listeners.
//
// Flutter: painting/image_stream.dart MultiFrameImageStreamCompleter
func NewMultiFrameImageStreamCompleter(binding *PaintingBinding, opts MultiFrameImageStreamCompleterOpts) *ImageStreamCompleter {
	c := NewImageStreamCompleter(binding)
	c.DebugLabel = opts.DebugLabel
	m := &multiFrame{binding: binding, informationCollector: opts.InformationCollector, scale: opts.Scale, done: make(chan struct{})}
	c.behavior = m
	go func() {
		select {
		case r := <-opts.Codec:
			binding.Post(func() {
				if r.Err != nil {
					if !c.disposed {
						c.ReportError(ImageErrorDetails{Err: r.Err, Context: "resolving an image codec", Silent: true, Information: opts.InformationCollector})
					}
					return
				}
				if c.disposed {
					r.Codec.Dispose()
					return
				}
				m.handleCodecReady(c, r.Codec)
			})
		case <-m.done:
		}
	}()
	if opts.ChunkEvents != nil {
		go func() {
			for {
				select {
				case ev, ok := <-opts.ChunkEvents:
					if !ok {
						return
					}
					binding.Post(func() {
						if !c.disposed {
							c.ReportImageChunkEvent(ev)
						}
					})
				case <-m.done:
					return
				}
			}
		}()
	}
	return c
}

func (m *multiFrame) handleCodecReady(c *ImageStreamCompleter, codec *ui.Codec) {
	m.codec = codec
	if c.HasListeners() {
		m.decodeNextFrameAndSchedule(c)
	}
}

func (m *multiFrame) handleAppFrame(c *ImageStreamCompleter, timestamp time.Duration) {
	m.frameCallback = false
	if !c.HasListeners() {
		return
	}
	if !m.hasDuration || timestamp-m.shownAt >= m.frameDuration {
		m.emitFrame(c, NewImageInfo(m.nextFrame.Image.Clone(), ImageInfoOpts{Scale: foundation.Some(m.scale), DebugLabel: c.DebugLabel}))
		m.shownAt = timestamp
		m.frameDuration, m.hasDuration = m.nextFrame.Duration, true
		m.nextFrame.Image.Dispose()
		m.nextFrame = nil
		if m.codec == nil {
			return
		}
		completedCycles := m.framesEmitted / m.codec.FrameCount()
		if m.codec.RepetitionCount() == -1 || completedCycles <= m.codec.RepetitionCount() {
			m.decodeNextFrameAndSchedule(c)
			return
		}
		m.codec.Dispose()
		m.codec = nil
		return
	}
	delay := m.frameDuration - (timestamp - m.shownAt)
	dilation := m.binding.Scheduler().TimeDilation()
	m.stopTimer = m.binding.AfterFunc(time.Duration(float64(delay)*dilation), func() { m.scheduleAppFrame(c) })
}

func (m *multiFrame) decodeNextFrameAndSchedule(c *ImageStreamCompleter) {
	if m.nextFrame != nil {
		m.nextFrame.Image.Dispose()
		m.nextFrame = nil
	}
	frame, err := m.codec.GetNextFrame(m.binding.Context())
	if err != nil {
		c.ReportError(ImageErrorDetails{Err: err, Context: "resolving an image frame", Silent: true, Information: m.informationCollector})
		return
	}
	m.nextFrame = &frame
	// Dart awaits the frame, so the rest runs in a later event turn, when a
	// listener that is being added has been added.
	m.binding.Post(func() { m.afterFrameDecoded(c) })
}

func (m *multiFrame) afterFrameDecoded(c *ImageStreamCompleter) {
	if m.codec == nil || m.nextFrame == nil {
		return
	}
	if m.codec.FrameCount() == 1 {
		if !c.HasListeners() {
			return
		}
		m.emitFrame(c, NewImageInfo(m.nextFrame.Image.Clone(), ImageInfoOpts{Scale: foundation.Some(m.scale), DebugLabel: c.DebugLabel}))
		m.nextFrame.Image.Dispose()
		m.nextFrame = nil
		m.codec.Dispose()
		m.codec = nil
		return
	}
	m.scheduleAppFrame(c)
}

func (m *multiFrame) scheduleAppFrame(c *ImageStreamCompleter) {
	if m.frameCallback {
		return
	}
	m.frameCallback = true
	m.binding.Scheduler().ScheduleFrameCallback(func(ts time.Duration) { m.handleAppFrame(c, ts) })
}

func (m *multiFrame) emitFrame(c *ImageStreamCompleter, info *ImageInfo) {
	c.SetImage(info)
	m.framesEmitted++
}

func (m *multiFrame) beforeAddListener(c *ImageStreamCompleter) {
	if !c.HasListeners() && m.codec != nil && (c.currentImage == nil || m.codec.FrameCount() > 1) {
		m.decodeNextFrameAndSchedule(c)
	}
}

func (m *multiFrame) afterRemoveListener(c *ImageStreamCompleter) {
	if !c.HasListeners() && m.stopTimer != nil {
		m.stopTimer()
		m.stopTimer = nil
	}
}

func (m *multiFrame) afterMaybeDispose(c *ImageStreamCompleter) {
	if c.disposed {
		select {
		case <-m.done:
		default:
			close(m.done)
		}
		if m.codec != nil {
			m.codec.Dispose()
			m.codec = nil
		}
	}
}
