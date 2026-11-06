package stl

import (
	"context"
	"math/rand"
	"slices"
	"testing"
)

func TestWeldIsTheSameForAnyNumberOfWorkers(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// Corners on a lattice, so that many of them are equal, in a random order.
	const n = 50000
	pos := make([]float32, 3*n)
	for i := range pos {
		pos[i] = float32(rng.Intn(20)) * 0.25
	}
	tests := []struct {
		name string
		tol  float64
	}{{"exact", 0}, {"a tolerance", 0.1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The naive welding: the vertices in the order of their first corners.
			want := make([]uint32, n)
			var wantVerts []float32
			seen := map[[3]int64]uint32{}
			for c := 0; c < n; c++ {
				k := key(pos[3*c:3*c+3], tt.tol)
				id, ok := seen[k]
				if !ok {
					id = uint32(len(wantVerts) / 3)
					seen[k] = id
					wantVerts = append(wantVerts, pos[3*c:3*c+3]...)
				}
				want[c] = id
			}
			for _, workers := range []int{1, 2, 3, 5, 8} {
				idx, verts := weld(context.Background(), pos, tt.tol, workers)
				if !slices.Equal(idx, want) || !slices.Equal(verts, wantVerts) {
					t.Errorf("%d workers: %d vertices (want %d), indices differ: %v", workers, len(verts)/3, len(wantVerts)/3, !slices.Equal(idx, want))
				}
			}
		})
	}
}

func TestWeldStopsWhenTheContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, workers := range []int{1, 4} {
		if idx, verts := weld(ctx, make([]float32, 3*100000), 0, workers); idx != nil || verts != nil {
			t.Errorf("%d workers: weld gave a result in a context that is done", workers)
		}
	}
}
