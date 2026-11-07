package gltf

import (
	"context"
	"slices"
	"testing"
)

// TestMatrixColumnsArePadded checks the layout of matrices (glTF 2.0, 3.6.2.4): a column of 1 or 2 byte components starts at a
// multiple of four bytes.
func TestMatrixColumnsArePadded(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		ct   int
		norm bool
		data []byte
		want []float32
	}{
		{"MAT2 of bytes", "MAT2", ctUbyte, false, []byte{1, 2, 0, 0, 3, 4, 0, 0}, []float32{1, 2, 3, 4}},
		{"MAT3 of bytes", "MAT3", ctByte, false, []byte{1, 2, 3, 0, 4, 5, 6, 0, 7, 8, 9, 0}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{"MAT3 of shorts", "MAT3", ctShort, false,
			[]byte{1, 0, 2, 0, 3, 0, 0, 0, 4, 0, 5, 0, 6, 0, 0, 0, 7, 0, 8, 0, 9, 0, 0, 0},
			[]float32{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{"MAT2 of shorts", "MAT2", ctUshort, false, []byte{1, 0, 2, 0, 3, 0, 4, 0}, []float32{1, 2, 3, 4}},
		{"MAT3 of normalized bytes", "MAT3", ctUbyte, true, []byte{255, 0, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0}, []float32{1, 0, 1, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 1
			size := len(tt.data)
			bv, buf := 0, size
			doc := &document{
				Buffers:     []bufferJSON{{ByteLength: &buf, URI: "data:application/octet-stream;base64,"}},
				BufferViews: []bufferViewJSON{{Buffer: &bv, ByteLength: &size}},
				Accessors:   []accessorJSON{{BufferView: &bv, ComponentType: tt.ct, Normalized: tt.norm, Count: n, Type: tt.typ}},
			}
			l := newLoader(context.Background(), doc, nil, Options{})
			l.buffers[0] = tt.data
			a, err := l.accessor(0, "")
			if err != nil {
				t.Fatal(err)
			}
			if a.elemSize != len(tt.data) {
				t.Errorf("element size %d, want %d", a.elemSize, len(tt.data))
			}
			got, err := l.floats(a)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("components %v, want %v", got, tt.want)
			}
		})
	}
}
