package goid

import (
	"sync"
	"testing"
)

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   uint64
	}{
		{"one digit", "goroutine 1 [running]:", 1},
		{"several digits", "goroutine 123 [running]:\nmain.main()", 123},
		{"large", "goroutine 18446744073709551 [running]:", 18446744073709551},
		{"cut by a small buffer", "goroutine 4567", 4567},
		{"cut in the middle of the number", "goroutine 45", 45},
		{"no number", "goroutine [running]:", 0},
		{"only the prefix", "goroutine ", 0},
		{"too short", "gorout", 0},
		{"empty", "", 0},
		{"other text", "panic: something went wrong", 0},
		{"waiting goroutine", "goroutine 7 [chan receive, 3 minutes]:", 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseHeader([]byte(tt.header)); got != tt.want {
				t.Errorf("parseHeader(%q) = %d, want %d", tt.header, got, tt.want)
			}
		})
	}
}

// growSink keeps the frames of TestTokenIsTheSameOnOneGoroutineAtAnyDepth big.
var growSink byte

// at calls fn n frames deeper than the caller.
//
//go:noinline
func at(n int, fn func()) {
	if n > 0 {
		at(n-1, fn)
		return
	}
	fn()
}

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

func TestTokenIsTheSameOnOneGoroutineAtAnyDepth(t *testing.T) {
	want := Token()
	if want == 0 {
		t.Fatal("Token is 0")
	}
	// No subtests here: a subtest runs on a goroutine of its own.
	for _, depth := range []int{0, 1, 5, 50, 200, 2000} {
		at(depth, func() {
			if got := Token(); got != want {
				t.Errorf("Token at depth %d = %#x, want %#x", depth, got, want)
			}
		})
	}
	// A goroutine's stack moves when it grows; its token does not.
	var grow func(n int) uint64
	grow = func(n int) uint64 {
		var pad [256]byte
		pad[n%len(pad)] = 1
		growSink = pad[0] // keeps pad in the frame
		if n == 0 {
			return Token()
		}
		return grow(n - 1)
	}
	if got := grow(5000); got != want {
		t.Errorf("Token after the stack grew = %#x, want %#x", got, want)
	}
}

func TestTokenEqualsIDOnManyGoroutinesAtAnyDepth(t *testing.T) {
	tests := []struct {
		name       string
		goroutines int
		depth      int
	}{
		{"one goroutine, shallow", 1, 0},
		{"ten goroutines, depth 5", 10, 5},
		{"two hundred goroutines, depth 50", 200, 50},
		{"twenty goroutines, depth 2000", 20, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			bad := make(chan string, tt.goroutines)
			for range tt.goroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()
					at(tt.depth, func() {
						if tok, id := Token(), ID(); tok != id || tok == 0 {
							bad <- "Token != ID"
						}
					})
				}()
			}
			wg.Wait()
			close(bad)
			for msg := range bad {
				t.Error(msg)
			}
		})
	}
}

// TestTokenIsNeverGivenToALaterGoroutine is the regression of 1.9.3: goroutines
// that end hand their g to the next ones, and a token that was the address of the
// g was then the same on two goroutines, one after the other.
func TestTokenIsNeverGivenToALaterGoroutine(t *testing.T) {
	tests := []struct {
		name    string
		rounds  int
		perHand int
	}{
		{"one at a time", 500, 1},
		{"eight at a time", 200, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := map[uint64]bool{Token(): true}
			for round := 0; round < tt.rounds; round++ {
				tokens := make([]uint64, tt.perHand)
				var wg sync.WaitGroup
				for i := range tokens {
					wg.Add(1)
					go func() {
						defer wg.Done()
						tokens[i] = Token()
					}()
				}
				wg.Wait()
				for _, tok := range tokens {
					if tok == 0 || seen[tok] {
						t.Fatalf("round %d: token %d is 0 or was the token of a goroutine before", round, tok)
					}
					seen[tok] = true
				}
			}
		})
	}
}

func TestTokensOfLiveGoroutinesDiffer(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"two", 2},
		{"twenty", 20},
		{"two thousand", 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := make([]uint64, tt.n)
			var started, wg sync.WaitGroup
			release := make(chan struct{})
			started.Add(tt.n)
			wg.Add(tt.n)
			for i := range tokens {
				go func() {
					defer wg.Done()
					tokens[i] = Token()
					started.Done()
					// Stay alive until every goroutine has its token, so that no
					// token can have been handed on to a later goroutine.
					<-release
					if Token() != tokens[i] {
						t.Error("the token of a goroutine changed")
					}
				}()
			}
			started.Wait()
			close(release)
			wg.Wait()
			seen := map[uint64]int{Token(): -1}
			for i, tok := range tokens {
				if j, dup := seen[tok]; dup || tok == 0 {
					t.Fatalf("goroutine %d has token %#x, also seen for %d (or 0)", i, tok, j)
				}
				seen[tok] = i
			}
		})
	}
}

func BenchmarkID(b *testing.B) {
	for _, depth := range []int{5, 200} {
		b.Run(depthName(depth), func(b *testing.B) {
			at(depth, func() {
				for b.Loop() {
					ID()
				}
			})
		})
	}
}

func BenchmarkToken(b *testing.B) {
	for _, depth := range []int{5, 200} {
		b.Run(depthName(depth), func(b *testing.B) {
			at(depth, func() {
				for b.Loop() {
					Token()
				}
			})
		})
	}
}

func depthName(n int) string {
	if n == 5 {
		return "depth5"
	}
	return "depth200"
}
