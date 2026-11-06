package vtx_test

import (
	"testing"

	"github.com/rm4n0s/glutter/model/internal/vtx"
)

func TestKeysAreNumberedInTheOrderTheyCome(t *testing.T) {
	tab := vtx.New(0)
	const n = 5000
	for round := 0; round < 3; round++ {
		for i := 0; i < n; i++ {
			k := vtx.Key{int64(i % 17), int64(i), int64(-i)}
			got, added := tab.Find(k)
			if got != uint32(i) || added != (round == 0) {
				t.Fatalf("round %d key %d: number %d, added %v", round, i, got, added)
			}
		}
	}
	if tab.Len() != n {
		t.Errorf("Len = %d, want %d", tab.Len(), n)
	}
}

func TestKeysThatDifferInOneNumberAreDifferent(t *testing.T) {
	tab := vtx.New(4)
	seen := map[uint32]bool{}
	for _, k := range []vtx.Key{{1, 2, 3}, {1, 2, 4}, {1, 3, 3}, {2, 2, 3}, {3, 2, 1}, {0, 0, 0}, {-1, 0, 0}, {0, -1, 0}, {0, 0, -1}} {
		n, added := tab.Find(k)
		if !added || seen[n] {
			t.Errorf("key %v: number %d added %v", k, n, added)
		}
		seen[n] = true
	}
}
