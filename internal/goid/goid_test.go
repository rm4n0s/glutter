package goid

import (
	"sync"
	"testing"
)

func TestIDIsStablePerGoroutineAndDiffersBetweenGoroutines(t *testing.T) {
	main := ID()
	if main == 0 {
		t.Fatal("ID is 0")
	}
	if ID() != main {
		t.Error("the ID of a goroutine changed")
	}
	const n = 20
	ids := make([]uint64, n)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i] = ID()
			if ID() != ids[i] {
				t.Error("the ID of a goroutine changed")
			}
		}()
	}
	wg.Wait()
	seen := map[uint64]bool{main: true}
	for _, id := range ids {
		// The goroutines can finish before the next one starts and a number is
		// not reused while the program runs, so every ID is new.
		if id == 0 || seen[id] {
			t.Errorf("ID %d is 0 or was seen before", id)
		}
		seen[id] = true
	}
}

func BenchmarkID(b *testing.B) {
	for b.Loop() {
		ID()
	}
}
