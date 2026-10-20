// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import "github.com/rm4n0s/errors"

// VertexMode is how the vertices of a Vertices are joined into triangles.
//
// Flutter: ui/painting.dart VertexMode
type VertexMode int

// The values of VertexMode follow Flutter's declaration order.
const (
	// VertexModeUnspecified is the zero value: VertexModeTriangles.
	//
	// Flutter: no counterpart
	VertexModeUnspecified VertexMode = iota
	// VertexModeTriangles is Flutter's VertexMode.triangles. Draw each sequence
	// of three vertices as the vertices of a triangle.
	//
	// Flutter: ui/painting.dart VertexMode.triangles
	VertexModeTriangles
	// VertexModeTriangleStrip is Flutter's VertexMode.triangleStrip. Draw each
	// sliding window of three vertices as the vertices of a triangle.
	//
	// Flutter: ui/painting.dart VertexMode.triangleStrip
	VertexModeTriangleStrip
	// VertexModeTriangleFan is Flutter's VertexMode.triangleFan. Draw the first
	// vertex and each sliding window of two vertices as the vertices of a
	// triangle.
	//
	// Flutter: ui/painting.dart VertexMode.triangleFan
	VertexModeTriangleFan
)

var vertexModeNames = [...]string{"unspecified", "triangles", "triangleStrip", "triangleFan"}

// String returns the name Dart prints for the value, such as
// "VertexMode.triangles".
func (e VertexMode) String() string { return dartEnumName(vertexModeNames[:], int(e), "VertexMode") }

// PointMode is how Canvas.DrawPoints joins the points it is given.
//
// Flutter: ui/painting.dart PointMode
type PointMode int

// The values of PointMode follow Flutter's declaration order.
const (
	// PointModeUnspecified is the zero value: PointModePoints.
	//
	// Flutter: no counterpart
	PointModeUnspecified PointMode = iota
	// PointModePoints is Flutter's PointMode.points. Draw each point separately.
	// If the paint's stroke cap is round, this draws circles; otherwise it draws
	// squares.
	//
	// Flutter: ui/painting.dart PointMode.points
	PointModePoints
	// PointModeLines is Flutter's PointMode.lines. Draw each sequence of two
	// points as a line segment.
	//
	// Flutter: ui/painting.dart PointMode.lines
	PointModeLines
	// PointModePolygon is Flutter's PointMode.polygon. Draw the entire sequence of
	// points as one line.
	//
	// Flutter: ui/painting.dart PointMode.polygon
	PointModePolygon
)

var pointModeNames = [...]string{"unspecified", "points", "lines", "polygon"}

// String returns the name Dart prints for the value, such as "PointMode.lines".
func (e PointMode) String() string { return dartEnumName(pointModeNames[:], int(e), "PointMode") }

// ClipOp is the way a clip combines with the current clip.
//
// Flutter: ui/painting.dart ClipOp
type ClipOp int

// The values of ClipOp follow Flutter's declaration order.
const (
	// ClipOpUnspecified is the zero value: ClipOpIntersect.
	//
	// Flutter: no counterpart
	ClipOpUnspecified ClipOp = iota
	// ClipOpDifference is Flutter's ClipOp.difference. Subtract the clip shape
	// from the current clip.
	//
	// Flutter: ui/painting.dart ClipOp.difference
	ClipOpDifference
	// ClipOpIntersect is Flutter's ClipOp.intersect. Intersect the clip shape with
	// the current clip. This is the default.
	//
	// Flutter: ui/painting.dart ClipOp.intersect
	ClipOpIntersect
)

var clipOpNames = [...]string{"unspecified", "difference", "intersect"}

// String returns the name Dart prints for the value, such as "ClipOp.intersect".
func (e ClipOp) String() string { return dartEnumName(clipOpNames[:], int(e), "ClipOp") }

// Vertices is a set of vertex positions, and optionally colors, texture
// coordinates and indices, to draw as triangles with Canvas.DrawVertices. It is
// immutable.
//
// Flutter: ui/painting.dart Vertices
type Vertices struct {
	mode       VertexMode
	positions  []Offset
	colors     []Color
	texCoords  []Offset
	indices    []uint16
	hasColors  bool
	hasTexture bool
	hasIndices bool
}

// VerticesOpts are the optional arguments of NewVertices.
//
// Flutter: ui/painting.dart Vertices
type VerticesOpts struct {
	// Colors are the colors at the vertices, as many as the positions, or nil.
	Colors []Color
	// TextureCoordinates are the points of the shader's image to use at the
	// vertices, as many as the positions, or nil.
	TextureCoordinates []Offset
	// Indices pick the positions to build the triangles from; nil means the
	// positions in order. Each index is less than the number of positions, and
	// at most 65535.
	Indices []int
}

// NewVertices creates a set of vertices. It panics with the tag
// VerticesColorCountMismatch, VerticesTextureCountMismatch or
// VerticesIndexInvalid when the lists do not fit the positions.
//
// Flutter: ui/painting.dart Vertices
func NewVertices(mode VertexMode, positions []Offset, opts ...VerticesOpts) *Vertices {
	o := oneOpts(opts)
	if o.Colors != nil && len(o.Colors) != len(positions) {
		panic(errors.New("VerticesColorCountMismatch", `"positions" and "colors" lengths must match.`,
			"positions", len(positions), "colors", len(o.Colors)))
	}
	if o.TextureCoordinates != nil && len(o.TextureCoordinates) != len(positions) {
		panic(errors.New("VerticesTextureCountMismatch", `"positions" and "textureCoordinates" lengths must match.`,
			"positions", len(positions), "textureCoordinates", len(o.TextureCoordinates)))
	}
	if mode == VertexModeUnspecified {
		mode = VertexModeTriangles
	}
	v := &Vertices{
		mode:       mode,
		positions:  append([]Offset(nil), positions...),
		hasColors:  o.Colors != nil,
		hasTexture: o.TextureCoordinates != nil,
		hasIndices: o.Indices != nil,
	}
	if v.hasColors {
		v.colors = append([]Color(nil), o.Colors...)
	}
	if v.hasTexture {
		v.texCoords = append([]Offset(nil), o.TextureCoordinates...)
	}
	if v.hasIndices {
		v.indices = make([]uint16, len(o.Indices))
		for i, idx := range o.Indices {
			if idx < 0 || idx >= len(positions) || idx > 0xFFFF {
				panic(errors.New("VerticesIndexInvalid",
					`"indices" values must be valid indices in the positions list (and at most 65535).`,
					"index", i, "value", idx, "positions", len(positions)))
			}
			v.indices[i] = uint16(idx)
		}
	}
	return v
}

// Mode is how the vertices are joined into triangles.
//
// Flutter: ui/painting.dart Vertices
func (v *Vertices) Mode() VertexMode { return v.mode }

// Positions returns a copy of the vertex positions.
//
// Flutter: ui/painting.dart Vertices
func (v *Vertices) Positions() []Offset { return append([]Offset(nil), v.positions...) }

// Colors returns a copy of the vertex colors, or nil when there are none.
//
// Flutter: ui/painting.dart Vertices
func (v *Vertices) Colors() []Color {
	if !v.hasColors {
		return nil
	}
	return append([]Color{}, v.colors...)
}

// TextureCoordinates returns a copy of the texture coordinates, or nil when there
// are none.
//
// Flutter: ui/painting.dart Vertices
func (v *Vertices) TextureCoordinates() []Offset {
	if !v.hasTexture {
		return nil
	}
	return append([]Offset{}, v.texCoords...)
}

// Indices returns a copy of the indices, or nil when there are none.
//
// Flutter: ui/painting.dart Vertices
func (v *Vertices) Indices() []uint16 {
	if !v.hasIndices {
		return nil
	}
	return append([]uint16{}, v.indices...)
}

// Dispose releases the vertices. It does nothing in this implementation, which
// holds no native resources.
//
// Flutter: ui/painting.dart Vertices.dispose
func (v *Vertices) Dispose() {}
