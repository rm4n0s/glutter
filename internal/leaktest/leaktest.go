// Package leaktest checks that a scenario does not leak memory: it runs a step many times, and
// compares the live heap and the goroutines after N steps and after 4N steps (SPEC §15, "Memory
// over time"). It is imported by tests only. It imports "testing", so no program may link it
// (step 29.16 keeps "testing" out of programs).
//
// A scenario leaks when what survives two garbage collections grows from N to 4N steps by more than
// a small slack (the runtime's own caches settle slowly), or when the number of goroutines grows at
// all. It does not compare with an absolute bound, so it is not flaky on a machine with another
// heap.
//
// Set LEAKTEST_SCALE to a number to multiply every N by it, for a longer soak than the default
// test run.
//
// Set LEAKTEST_REPORT to a file name to append one tab-separated line for each scenario (name, N,
// then the live heap in bytes, the heap in use, the live objects and the goroutines at the base,
// after N and after 4N steps). Set LEAKTEST_PROFILE to a directory to write the heap profile
// of every scenario at the three points (<name>.base.pb.gz, <name>.n.pb.gz, <name>.4n.pb.gz);
// run the test binary with -memprofile and -memprofilerate=1 to make the profiles exact, and
// name the site of a leak with
//
//	go tool pprof -sample_index=inuse_space -base <name>.n.pb.gz <test binary> <name>.4n.pb.gz
//
// Flutter: no counterpart
package leaktest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Stats is what the runtime reports after two garbage collections.
//
// Flutter: no counterpart
type Stats struct {
	// HeapAlloc is the bytes of live objects, HeapInuse the bytes of the spans that hold them.
	HeapAlloc, HeapInuse uint64
	// HeapObjects is the number of live objects.
	HeapObjects uint64
	// Goroutines is runtime.NumGoroutine.
	Goroutines int
}

// Snapshot collects garbage twice and reads the statistics. The goroutines that are finishing get
// a moment to be gone.
//
// Flutter: no counterpart
func Snapshot() Stats {
	var ms runtime.MemStats
	prev := -1
	for i := 0; i < 20; i++ {
		runtime.GC()
		runtime.GC()
		n := runtime.NumGoroutine()
		if n == prev {
			break
		}
		prev = n
		time.Sleep(2 * time.Millisecond)
	}
	runtime.ReadMemStats(&ms)
	return Stats{HeapAlloc: ms.HeapAlloc, HeapInuse: ms.HeapInuse, HeapObjects: ms.HeapObjects, Goroutines: runtime.NumGoroutine()}
}

// Opts are the optional parameters of Run.
//
// Flutter: no counterpart
type Opts struct {
	// N is the number of steps of the first run; the second run adds 3N. The default is 200.
	N int
	// Warmup is the number of steps before the base is taken, so that the caches that fill once
	// are full. The default is 20.
	Warmup int
	// SlackBytes is the growth of the live heap that is not a leak; the default is 256 KB, and
	// 2% of the base is added.
	SlackBytes uint64
	// Counters are other numbers that must not grow, such as the resources of a GPU device or the
	// blocks of C memory that are not freed: each is read with the heap at the base, after N and
	// after 4N steps, and may grow by at most CounterSlack from N to 4N.
	Counters map[string]func() int
	// CounterSlack is the growth of a counter that is not a leak; the default is none.
	CounterSlack int
	// SlackObjects is the growth of the number of live objects that is not a leak; the default is
	// 300.
	SlackObjects uint64
}

// Result is the numbers of a scenario.
//
// Flutter: no counterpart
type Result struct {
	Name                  string
	N                     int
	Base, AfterN, After4N Stats
	// Counters are the values of Opts.Counters at the base, after N and after 4N steps.
	Counters map[string][3]int
}

// Run runs a scenario. setup prepares it (it may use tb for cleanups) and returns the step that
// runs once for each unit of N; the step must leave the app as it found it (a route pushed is
// popped). Run fails tb when the scenario leaks, and returns its numbers.
//
// Flutter: no counterpart
func Run(tb testing.TB, name string, setup func() (step func()), opts ...Opts) Result {
	tb.Helper()
	var o Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.N == 0 {
		o.N = 200
	}
	if o.Warmup == 0 {
		o.Warmup = 20
	}
	if o.SlackBytes == 0 {
		o.SlackBytes = 256 << 10
	}
	if o.SlackObjects == 0 {
		o.SlackObjects = 300
	}
	if scale, err := strconv.Atoi(os.Getenv("LEAKTEST_SCALE")); err == nil && scale > 1 {
		o.N *= scale
	}
	step := setup()
	for i := 0; i < o.Warmup; i++ {
		step()
	}
	res := Result{Name: name, N: o.N, Counters: map[string][3]int{}}
	readCounters := func(i int) {
		for k, f := range o.Counters {
			c := res.Counters[k]
			c[i] = f()
			res.Counters[k] = c
		}
	}
	res.Base = Snapshot()
	readCounters(0)
	profile(tb, name, "base")
	for i := 0; i < o.N; i++ {
		step()
	}
	res.AfterN = Snapshot()
	readCounters(1)
	profile(tb, name, "n")
	for i := 0; i < 3*o.N; i++ {
		step()
	}
	res.After4N = Snapshot()
	readCounters(2)
	profile(tb, name, "4n")
	report(tb, res)

	grow := func(a, b uint64) uint64 {
		if b <= a {
			return 0
		}
		return b - a
	}
	slackB := o.SlackBytes + res.AfterN.HeapAlloc/50
	slackO := o.SlackObjects
	if g := grow(res.AfterN.HeapAlloc, res.After4N.HeapAlloc); g > slackB {
		tb.Errorf("%s: the live heap grew by %d bytes from %d to %d steps (slack %d)", name, g, o.N, 4*o.N, slackB)
	}
	if g := grow(res.AfterN.HeapInuse, res.After4N.HeapInuse); g > slackB {
		tb.Errorf("%s: the heap in use grew by %d bytes from %d to %d steps (slack %d)", name, g, o.N, 4*o.N, slackB)
	}
	if g := grow(res.AfterN.HeapObjects, res.After4N.HeapObjects); g > slackO {
		tb.Errorf("%s: the live objects grew by %d from %d to %d steps (slack %d)", name, g, o.N, 4*o.N, slackO)
	}
	for k, c := range res.Counters {
		if c[2] > c[1]+o.CounterSlack {
			tb.Errorf("%s: %s grew from %d to %d between %d and %d steps (slack %d)", name, k, c[1], c[2], o.N, 4*o.N, o.CounterSlack)
		}
	}
	if res.After4N.Goroutines > res.AfterN.Goroutines {
		tb.Errorf("%s: the goroutines grew from %d to %d between %d and %d steps\n%s", name,
			res.AfterN.Goroutines, res.After4N.Goroutines, o.N, 4*o.N, goroutines())
	}
	return res
}

func goroutines() string {
	var b bytes.Buffer
	if p := pprof.Lookup("goroutine"); p != nil {
		if err := p.WriteTo(&b, 1); err != nil {
			return "goroutine profile: " + err.Error()
		}
	}
	s := b.String()
	if len(s) > 6000 {
		s = s[:6000] + "..."
	}
	return s
}

func report(tb testing.TB, r Result) {
	tb.Helper()
	line := fmt.Sprintf("%s\t%d\t%d/%d/%d/%d\t%d/%d/%d/%d\t%d/%d/%d/%d", r.Name, r.N,
		r.Base.HeapAlloc, r.Base.HeapInuse, r.Base.HeapObjects, r.Base.Goroutines,
		r.AfterN.HeapAlloc, r.AfterN.HeapInuse, r.AfterN.HeapObjects, r.AfterN.Goroutines,
		r.After4N.HeapAlloc, r.After4N.HeapInuse, r.After4N.HeapObjects, r.After4N.Goroutines)
	keys := make([]string, 0, len(r.Counters))
	for k := range r.Counters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c := r.Counters[k]
		line += fmt.Sprintf("\t%s=%d/%d/%d", k, c[0], c[1], c[2])
	}
	tb.Logf("leaktest %s (alloc/inuse/objects/goroutines at base, N, 4N)", strings.ReplaceAll(line, "\t", "  "))
	if path := os.Getenv("LEAKTEST_REPORT"); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			tb.Errorf("LEAKTEST_REPORT: %v", err)
			return
		}
		_, werr := fmt.Fprintln(f, line)
		cerr := f.Close()
		if werr != nil || cerr != nil {
			tb.Errorf("LEAKTEST_REPORT: write: %v, close: %v", werr, cerr)
		}
	}
}

func profile(tb testing.TB, name, point string) {
	tb.Helper()
	dir := os.Getenv("LEAKTEST_PROFILE")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		tb.Errorf("LEAKTEST_PROFILE: %v", err)
		return
	}
	safe := strings.NewReplacer("/", "_", " ", "_").Replace(name)
	f, err := os.Create(filepath.Join(dir, safe+"."+point+".pb.gz"))
	if err != nil {
		tb.Errorf("LEAKTEST_PROFILE: %v", err)
		return
	}
	werr := pprof.Lookup("heap").WriteTo(f, 0)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		tb.Errorf("LEAKTEST_PROFILE: write: %v, close: %v", werr, cerr)
	}
}
