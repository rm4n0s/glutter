package stl

import (
	"context"
	"math"
	"runtime"
	"sync"

	"github.com/rm4n0s/glutter/model/internal/vtx"
)

// minPerWorker is the least number of items that make a worker worth its goroutine.
const minPerWorker = 1 << 15

// workersFor is how many workers to put on n items.
func workersFor(n int) int {
	w := min(runtime.GOMAXPROCS(0), 8)
	for w > 1 && n/w < minPerWorker {
		w--
	}
	return max(w, 1)
}

// parallel splits [0, n) into workers ranges and runs f(worker, lo, hi) on each, on goroutines of their own when there are several.
// The workers look at the context themselves and stop early; the caller looks at it after parallel returns.
func parallel(n, workers int, f func(worker, lo, hi int)) {
	if workers <= 1 {
		f(0, 0, n)
		return
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		lo, hi := n*w/workers, n*(w+1)/workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			f(w, lo, hi)
		}()
	}
	wg.Wait()
}

// key is the key under which a position is welded: the bits of its float64 value (-0 is 0), or its cell on a grid of tol.
func key[T float32 | float64](p []T, tol float64) vtx.Key {
	if tol > 0 {
		cell := func(v T) int64 {
			return int64(math.Max(-1e18, math.Min(1e18, math.Floor(float64(v)/tol))))
		}
		return vtx.Key{cell(p[0]), cell(p[1]), cell(p[2])}
	}
	return vtx.Key{
		int64(math.Float64bits(float64(p[0]) + 0)),
		int64(math.Float64bits(float64(p[1]) + 0)),
		int64(math.Float64bits(float64(p[2]) + 0)),
	}
}

// weld numbers the vertices of the corners (the positions, three numbers each) that are welded together: idx[c] is the number of
// the vertex of corner c, and verts has the position of each vertex, the one of its first corner. The vertices are numbered in the
// order of their first corners, whatever the number of workers. The work is shared among the workers by the hash of the keys, each
// worker having a table of its own; it stops early, with nothing, if the context is done.
func weld[T float32 | float64](ctx context.Context, pos []T, tol float64, workers int) (idx []uint32, verts []T) {
	nc := len(pos) / 3
	idx = make([]uint32, nc)
	type shard struct {
		first []uint32 // the first corner of each vertex of the shard, in order
		renum []uint32 // the number of each vertex of the shard in the whole
	}
	shards := make([]shard, workers)
	// The shard of each corner, by the high bits of the hash of its key; each worker then takes the corners of its own shard.
	var shardOf []uint8
	if workers > 1 {
		shardOf = make([]uint8, nc)
		parallel(nc, workers, func(_, lo, hi int) {
			for c := lo; c < hi; c++ {
				if c&(minPerWorker-1) == 0 && ctx.Err() != nil {
					return
				}
				h := vtx.Hash(key(pos[3*c:3*c+3:3*c+3], tol))
				shardOf[c] = uint8((h >> 32) * uint64(workers) >> 32)
			}
		})
	}
	parallel(workers, workers, func(s, _, _ int) {
		hint := nc/(2*workers) + 16
		tab := vtx.New(hint)
		first := make([]uint32, 0, hint)
		for c := 0; c < nc; c++ {
			if c&(minPerWorker-1) == 0 && ctx.Err() != nil {
				return
			}
			if workers > 1 && int(shardOf[c]) != s {
				continue
			}
			k := key(pos[3*c:3*c+3:3*c+3], tol)
			n, added := tab.FindHash(k, vtx.Hash(k))
			if added {
				first = append(first, uint32(c))
			}
			idx[c] = n*uint32(workers) + uint32(s)
		}
		shards[s].first = first
	})
	if ctx.Err() != nil {
		return nil, nil
	}
	// The vertices in the order of their first corners: a merge of the lists of the shards, which are in order.
	total := 0
	for _, s := range shards {
		total += len(s.first)
	}
	verts = make([]T, 0, 3*total)
	heads := make([]int, workers)
	for s := range shards {
		shards[s].renum = make([]uint32, 0, len(shards[s].first))
	}
	for g := 0; g < total; g++ {
		best := -1
		for s := range shards {
			if heads[s] < len(shards[s].first) && (best < 0 || shards[s].first[heads[s]] < shards[best].first[heads[best]]) {
				best = s
			}
		}
		c := int(shards[best].first[heads[best]])
		heads[best]++
		shards[best].renum = append(shards[best].renum, uint32(g))
		verts = append(verts, pos[3*c], pos[3*c+1], pos[3*c+2])
	}
	w := uint32(workers)
	parallel(nc, workersFor(nc), func(_, lo, hi int) {
		for c := lo; c < hi; c++ {
			v := idx[c]
			idx[c] = shards[v%w].renum[v/w]
		}
	})
	return idx, verts
}
