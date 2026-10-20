// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import "github.com/rm4n0s/errors"

// Picture is an object that holds a sequence of drawing commands, recorded by a
// Canvas and ended by PictureRecorder.EndRecording. It is immutable, and it can
// be drawn into any number of canvases or composited into a Scene.
//
// Flutter: ui/painting.dart Picture
type Picture struct {
	list *DisplayList
}

// DisplayList returns the recorded commands, for the renderers.
//
// Flutter: no counterpart
func (p *Picture) DisplayList() *DisplayList { return p.list }

// CullRect is the rectangle the canvas that recorded the picture was created
// with: the picture has no content that matters outside it.
//
// Flutter: no counterpart
func (p *Picture) CullRect() Rect { return p.list.cullRect }

// ApproximateBytesUsed estimates the memory the picture holds, in bytes. It is
// meant to decide how many pictures to keep, not to account for memory exactly.
//
// Flutter: ui/painting.dart Picture.approximateBytesUsed
func (p *Picture) ApproximateBytesUsed() int {
	n := 64
	for _, op := range p.list.ops {
		n += approximateBytes(op)
	}
	return n
}

// Dispose releases the picture. It does nothing in this implementation, which
// holds no native resources.
//
// Flutter: ui/painting.dart Picture.dispose
func (p *Picture) Dispose() {}

// Rasterizer turns a scene into pixels. The renderers implement it: the software
// renderer (internal/gpu/soft) draws on the CPU, and the GLES renderer reads back
// from the GPU. A Picture or a Scene cannot draw itself, so ToImage is given the
// rasterizer of the binding it is used under.
//
// Flutter: no counterpart (the engine's rasterizer is native)
type Rasterizer interface {
	// RasterizeScene draws the scene into a new image of width x height pixels.
	RasterizeScene(scene *Scene, width, height int) (*Image, error)
}

// ToImageOpts are the optional arguments of Picture.ToImage and Scene.ToImage.
//
// Flutter: ui/painting.dart Picture.toImage
type ToImageOpts struct {
	// Rasterizer draws the image. Nil makes ToImage fail with RasterizerMissing.
	Rasterizer Rasterizer
}

// ToImage draws the picture into an image of the given size in pixels. The
// picture is drawn from its origin with no scaling. It fails with
// ImageSizeInvalid for a size that is not positive, and RasterizerMissing when no
// rasterizer is given.
//
// Deviation: Flutter's returns a Future and draws with the engine's rasterizer;
// this one is synchronous and is given the rasterizer (SPEC §12.10).
//
// Flutter: ui/painting.dart Picture.toImage
func (p *Picture) ToImage(width, height int, opts ...ToImageOpts) (*Image, error) {
	o := oneOpts(opts)
	if width <= 0 || height <= 0 {
		return nil, errors.New("ImageSizeInvalid", "Invalid image dimensions.", "width", width, "height", height)
	}
	if o.Rasterizer == nil {
		return nil, errors.New("RasterizerMissing", "Picture.ToImage needs a Rasterizer in its options.")
	}
	b := NewSceneBuilder()
	b.AddPicture(Offset{}, p)
	return o.Rasterizer.RasterizeScene(b.Build(), width, height)
}

// ToImageSync is ToImage.
//
// Flutter: ui/painting.dart Picture.toImageSync
func (p *Picture) ToImageSync(width, height int, opts ...ToImageOpts) (*Image, error) {
	return p.ToImage(width, height, opts...)
}

// String describes the picture, as Dart's toString does.
//
// Flutter: ui/painting.dart Picture.toString
func (p *Picture) String() string { return "Picture" }

// PictureRecorder records the drawing commands of a Canvas into a Picture.
//
// Flutter: ui/painting.dart PictureRecorder
type PictureRecorder struct {
	canvas *Canvas
}

// NewPictureRecorder creates a recorder that is not recording yet: creating a
// Canvas for it starts the recording.
//
// Flutter: ui/painting.dart PictureRecorder
func NewPictureRecorder() *PictureRecorder { return &PictureRecorder{} }

// IsRecording reports whether a canvas is recording into the recorder.
//
// Flutter: ui/painting.dart PictureRecorder.isRecording
func (r *PictureRecorder) IsRecording() bool { return r.canvas != nil }

// EndRecording finishes recording and returns the picture of everything the
// canvas drew. The canvas can no longer draw. It panics with the tag
// PictureRecorderNotRecording when nothing was recording.
//
// Flutter: ui/painting.dart PictureRecorder.endRecording
func (r *PictureRecorder) EndRecording() *Picture {
	c := r.canvas
	if c == nil {
		panic(errors.New("PictureRecorderNotRecording", "PictureRecorder did not start recording."))
	}
	list := &DisplayList{ops: c.ops, cullRect: c.cullRect}
	c.ops, c.recorder = nil, nil
	r.canvas = nil
	return &Picture{list: list}
}

// String describes the recorder, as Dart's toString does.
//
// Flutter: ui/painting.dart PictureRecorder.toString
func (r *PictureRecorder) String() string {
	return "PictureRecorder(recording: " + boolString(r.IsRecording()) + ")"
}
