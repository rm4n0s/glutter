package leaktest_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/leaktest"
)

// fakeTB records the failures of a scenario instead of failing the test.
type fakeTB struct {
	testing.TB
	errs []string
}

func (f *fakeTB) Helper()                         {}
func (f *fakeTB) Logf(format string, args ...any) {}
func (f *fakeTB) Errorf(format string, args ...any) {
	f.errs = append(f.errs, format)
}

var (
	sink    [][]byte
	counter int
)

func TestRunFindsALeak(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	tests := []struct {
		name string
		step func() func()
		want []string // parts of the formats of the failures; none: the scenario is clean
	}{
		{"clean", func() func() {
			return func() { _ = make([]byte, 1<<10) }
		}, nil},
		{"heap", func() func() {
			sink = nil
			return func() { sink = append(sink, make([]byte, 4<<10)) }
		}, []string{"live heap grew"}},
		{"goroutines", func() func() {
			return func() { go func() { <-stop }() }
		}, []string{"goroutines grew"}},
		{"counter", func() func() {
			return func() { counter++ }
		}, []string{"%s: %s grew from"}},
	}
	counter = 0
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			counter = 0
			f := &fakeTB{TB: t}
			leaktest.Run(f, tc.name, tc.step, leaktest.Opts{N: 300, Counters: map[string]func() int{"widgets": func() int { return counter }}})
			joined := strings.Join(f.errs, "\n")
			if len(tc.want) == 0 && len(f.errs) != 0 {
				t.Fatalf("a clean scenario failed: %s", joined)
			}
			for _, w := range tc.want {
				if !strings.Contains(joined, w) {
					t.Errorf("failures %q do not say %q", joined, w)
				}
			}
		})
	}
	sink = nil
}
