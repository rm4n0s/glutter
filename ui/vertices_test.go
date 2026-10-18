package ui

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestVertices(t *testing.T) {
	pos := []Offset{{0, 0}, {10, 0}, {0, 10}, {10, 10}}
	cols := []Color{NewColor(0xFFFF0000), NewColor(0xFF00FF00), NewColor(0xFF0000FF), NewColor(0xFFFFFFFF)}
	tex := []Offset{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	v := NewVertices(VertexModeTriangleStrip, pos, VerticesOpts{Colors: cols, TextureCoordinates: tex, Indices: []int{0, 1, 2, 3, 65535 % 4}})
	if v.Mode() != VertexModeTriangleStrip || !reflect.DeepEqual(v.Positions(), pos) || !reflect.DeepEqual(v.Colors(), cols) ||
		!reflect.DeepEqual(v.TextureCoordinates(), tex) || !reflect.DeepEqual(v.Indices(), []uint16{0, 1, 2, 3, 3}) {
		t.Errorf("accessors: %+v", v)
	}
	// Everything is copied, in and out.
	pos[0], cols[0], tex[0] = Offset{99, 99}, Color{}, Offset{99, 99}
	v.Positions()[1] = Offset{77, 77}
	if v.Positions()[0] != (Offset{0, 0}) || v.Positions()[1] != (Offset{10, 0}) || v.Colors()[0] != NewColor(0xFFFF0000) || v.TextureCoordinates()[0] != (Offset{0, 0}) {
		t.Error("the vertices must copy the lists they are given and return copies")
	}
	plain := NewVertices(VertexModeUnspecified, pos)
	if plain.Mode() != VertexModeTriangles || plain.Colors() != nil || plain.TextureCoordinates() != nil || plain.Indices() != nil {
		t.Errorf("a plain one: %+v", plain)
	}
	// Empty lists are not nil lists: they must still match the positions.
	empty := NewVertices(VertexModeTriangles, nil, VerticesOpts{Colors: []Color{}})
	if empty.Colors() == nil {
		t.Error("an empty color list is kept as empty, not nil")
	}
	plain.Dispose()
	if VertexModeTriangleFan.String() != "VertexMode.triangleFan" || PointModePolygon.String() != "PointMode.polygon" || ClipOpDifference.String() != "ClipOp.difference" {
		t.Error("enum strings")
	}
}

func TestVerticesPanics(t *testing.T) {
	pos := []Offset{{0, 0}, {1, 1}, {2, 2}}
	tests := []struct {
		name string
		opts VerticesOpts
		tag  string
	}{
		{"too few colors", VerticesOpts{Colors: []Color{{}}}, "VerticesColorCountMismatch"},
		{"too many colors", VerticesOpts{Colors: make([]Color, 4)}, "VerticesColorCountMismatch"},
		{"texture coordinates", VerticesOpts{TextureCoordinates: []Offset{{}}}, "VerticesTextureCountMismatch"},
		{"an index past the end", VerticesOpts{Indices: []int{0, 1, 3}}, "VerticesIndexInvalid"},
		{"a negative index", VerticesOpts{Indices: []int{-1}}, "VerticesIndexInvalid"},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				appErr, ok := errors.FromError(recoverError(t, recover()))
				if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("NewVertices."+tt.tag) {
					t.Errorf("%s: got %v", tt.name, appErr)
				}
			}()
			NewVertices(VertexModeTriangles, pos, tt.opts)
		}()
	}
	// More than 65535 positions cannot be indexed.
	func() {
		defer func() {
			appErr, ok := errors.FromError(recoverError(t, recover()))
			if !ok || appErr.Tag != "VerticesIndexInvalid" {
				t.Errorf("a huge index: got %v", appErr)
			}
		}()
		NewVertices(VertexModeTriangles, make([]Offset, 70000), VerticesOpts{Indices: []int{66000}})
	}()
}
