package web

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPowerOfTwo(t *testing.T) {
	tests := []struct{ in, want int }{{-3, 1}, {0, 1}, {1, 1}, {2, 2}, {3, 4}, {1024, 1024}, {1025, 2048}, {4800, 8192}, {1 << 20, 1 << 20}}
	for _, tt := range tests {
		if got := PowerOfTwo(tt.in); got != tt.want {
			t.Errorf("PowerOfTwo(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestRingFrames(t *testing.T) {
	tests := []struct {
		rate   int
		buffer time.Duration
		want   int
	}{
		{48000, 100 * time.Millisecond, 8192}, // 4800 frames and a fifth more
		{44100, 100 * time.Millisecond, 8192},
		{96000, 100 * time.Millisecond, 16384},
		{8000, 100 * time.Millisecond, 1024},
		{48000, 0, 1024},
		{48000, 500 * time.Millisecond, 32768},
	}
	for _, tt := range tests {
		if got := RingFrames(tt.rate, tt.buffer); got != tt.want {
			t.Errorf("RingFrames(%d, %v) = %d, want %d", tt.rate, tt.buffer, got, tt.want)
		}
	}
}

func TestFramesToProduce(t *testing.T) {
	tests := []struct {
		name                 string
		write, read          uint32
		capacity, target, to int
	}{
		{"empty: the whole target", 0, 0, 8192, 4800, 4800},
		{"half full", 2400, 0, 8192, 4800, 2400},
		{"full to the target", 4800, 0, 8192, 4800, 0},
		{"more than the target", 6000, 0, 8192, 4800, 0},
		{"the target is more than fits", 0, 0, 1024, 4800, 1024},
		{"counters that wrapped", 100, 0xFFFFFF00, 8192, 4800, 4800 - 356},
		{"nothing read yet, counters high", 0xFFFFFFFF, 0xFFFFFFFF, 8192, 100, 100},
	}
	for _, tt := range tests {
		if got := FramesToProduce(tt.write, tt.read, tt.capacity, tt.target); got != tt.to {
			t.Errorf("%s: %d, want %d", tt.name, got, tt.to)
		}
	}
}

func TestAudioRingBasics(t *testing.T) {
	r := NewAudioRing(2, 5) // 8 frames
	if r.Capacity != 8 || r.H[RingCapacity] != 8 || r.H[RingChannels] != 2 {
		t.Fatalf("capacity %d, header %v", r.Capacity, r.H)
	}
	if n := r.Write([]float32{1, 2, 3, 4, 5, 6}); n != 3 || r.Available() != 3 || r.Free() != 5 {
		t.Fatalf("wrote %d, available %d, free %d", n, r.Available(), r.Free())
	}
	// A partial frame is not written.
	if n := r.Write([]float32{7, 8, 9}); n != 1 || r.Available() != 4 {
		t.Errorf("wrote %d of a frame and a half", n)
	}
	dst := make([]float32, 6)
	if n := r.Read(dst); n != 3 || dst[0] != 1 || dst[5] != 6 {
		t.Errorf("read %d: %v", n, dst)
	}
	// Writing past the end wraps.
	r.Write([]float32{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21})
	if r.Available() != 7 {
		t.Fatalf("available %d", r.Available())
	}
	out := make([]float32, 14)
	if n := r.Read(out); n != 7 || out[0] != 7 || out[2] != 10 || out[13] != 21 {
		t.Errorf("read %d: %v", n, out)
	}
	// It does not take more than fits.
	big := make([]float32, 40)
	if n := r.Write(big); n != 8 || r.Free() != 0 {
		t.Errorf("wrote %d into a ring of 8", n)
	}
}

func TestAudioRingPlayAndUnderruns(t *testing.T) {
	r := NewAudioRing(1, 8)
	dst := make([]float32, 4)
	// Silence before the first frame is not an underrun.
	if n := r.Play(dst, 4); n != 0 || r.H[RingUnderruns] != 0 {
		t.Fatalf("n %d, underruns %d", n, r.H[RingUnderruns])
	}
	r.Write([]float32{1, 2})
	for i := range dst {
		dst[i] = -1
	}
	if n := r.Play(dst, 4); n != 2 || dst[0] != 1 || dst[1] != 2 || dst[2] != 0 || dst[3] != 0 {
		t.Fatalf("n %d, %v", n, dst)
	}
	// The quantum that starts the sound is not an underrun either: the sound starts when its first frames come.
	if r.H[RingUnderruns] != 0 || r.H[RingPrimed] != 1 {
		t.Errorf("underruns %d, primed %d", r.H[RingUnderruns], r.H[RingPrimed])
	}
	// After that, no frames at all is an underrun.
	r.Play(dst, 4)
	if r.H[RingUnderruns] != 1 {
		t.Errorf("underruns %d", r.H[RingUnderruns])
	}
	// A full quantum is none.
	r.Write([]float32{1, 2, 3, 4})
	r.Play(dst, 4)
	if r.H[RingUnderruns] != 1 {
		t.Errorf("underruns %d after a full quantum", r.H[RingUnderruns])
	}
	// A short one is.
	r.Write([]float32{1})
	r.Play(dst, 4)
	if r.H[RingUnderruns] != 2 {
		t.Errorf("underruns %d after a short quantum", r.H[RingUnderruns])
	}
	// A paused ring plays silence and keeps what it has.
	r.Write([]float32{5, 6})
	r.H[RingFlags] = RingFlagPaused
	dst[0], dst[1] = 9, 9
	if n := r.Play(dst, 2); n != 0 || dst[0] != 0 || dst[1] != 0 || r.Available() != 2 || r.H[RingUnderruns] != 2 {
		t.Errorf("paused: n %d, %v, available %d, underruns %d", n, dst, r.Available(), r.H[RingUnderruns])
	}
}

func TestAudioRingRecord(t *testing.T) {
	r := NewAudioRing(2, 4)
	if n := r.Record([]float32{1, 2, 3, 4}, 2); n != 2 || r.H[RingOverruns] != 0 {
		t.Fatalf("n %d, overruns %d", n, r.H[RingOverruns])
	}
	if n := r.Record([]float32{5, 6, 7, 8, 9, 10, 11, 12}, 4); n != 2 || r.H[RingOverruns] != 1 {
		t.Errorf("a quantum that does not fit: n %d, overruns %d", n, r.H[RingOverruns])
	}
	r.H[RingFlags] = RingFlagPaused
	if n := r.Record([]float32{0, 0}, 1); n != 0 || r.H[RingOverruns] != 1 {
		t.Errorf("paused: n %d, overruns %d", n, r.H[RingOverruns])
	}
}

func TestAudioRingCountersWrap(t *testing.T) {
	r := NewAudioRing(1, 8)
	r.H[RingWrite], r.H[RingRead] = int32(-4), int32(-4) // 2^32 - 4: the counters wrap after four frames
	for i := 0; i < 5; i++ {
		src := []float32{float32(3*i + 1), float32(3*i + 2), float32(3*i + 3)}
		if n := r.Write(src); n != 3 {
			t.Fatalf("round %d wrote %d", i, n)
		}
		dst := make([]float32, 3)
		if n := r.Read(dst); n != 3 || dst[0] != src[0] || dst[2] != src[2] {
			t.Fatalf("round %d read %d: %v", i, n, dst)
		}
	}
	if r.Available() != 0 || uint32(r.H[RingWrite]) != 11 {
		t.Errorf("available %d, write %d", r.Available(), uint32(r.H[RingWrite]))
	}
}

// ringOp is an operation that the Go model and the JavaScript ring both run.
type ringOp struct {
	Op      string    `json:"op"`
	Samples []float32 `json:"samples,omitempty"`
	Frames  int       `json:"frames,omitempty"`
	Flags   int       `json:"flags,omitempty"`
}

type opResult struct {
	N   uint32    `json:"n"`
	Out []float32 `json:"out"`
}

type ringRun struct {
	Results []opResult `json:"results"`
	Header  []int32    `json:"header"`
	Data    []float32  `json:"data"`
}

// randomOps makes a sequence of operations; the samples are multiples of 1/64, which float32 and JSON keep exactly.
func randomOps(rng *rand.Rand, channels, n int) []ringOp {
	sample := func() float32 { return float32(rng.Intn(129)-64) / 64 }
	var ops []ringOp
	for range n {
		switch rng.Intn(8) {
		case 0, 1, 2:
			frames := rng.Intn(40)
			s := make([]float32, frames*channels)
			for i := range s {
				s[i] = sample()
			}
			ops = append(ops, ringOp{Op: "write", Samples: s})
		case 3:
			ops = append(ops, ringOp{Op: "read", Frames: rng.Intn(30)})
		case 4, 5:
			ops = append(ops, ringOp{Op: "play", Frames: 1 + rng.Intn(30)})
		case 6:
			frames := 1 + rng.Intn(20)
			s := make([]float32, frames*channels)
			for i := range s {
				s[i] = sample()
			}
			ops = append(ops, ringOp{Op: "record", Samples: s, Frames: frames})
		default:
			ops = append(ops, ringOp{Op: "flags", Flags: rng.Intn(2) * RingFlagPaused})
		}
	}
	return append(ops, ringOp{Op: "available"})
}

// runModel runs the operations on the Go model.
func runModel(channels, capacity int, write, read uint32, ops []ringOp) ringRun {
	r := NewAudioRing(channels, capacity)
	r.H[RingWrite], r.H[RingRead] = int32(write), int32(read)
	var out ringRun
	for _, o := range ops {
		switch o.Op {
		case "write":
			out.Results = append(out.Results, opResult{N: uint32(r.Write(o.Samples))})
		case "read":
			dst := make([]float32, o.Frames*channels)
			for i := range dst {
				dst[i] = -1
			}
			n := r.Read(dst)
			out.Results = append(out.Results, opResult{N: uint32(n), Out: dst})
		case "play":
			dst := make([]float32, o.Frames*channels)
			for i := range dst {
				dst[i] = -1
			}
			n := r.Play(dst, o.Frames)
			out.Results = append(out.Results, opResult{N: uint32(n), Out: dst})
		case "record":
			out.Results = append(out.Results, opResult{N: uint32(r.Record(o.Samples, o.Frames))})
		case "flags":
			r.H[RingFlags] = int32(o.Flags)
			out.Results = append(out.Results, opResult{})
		case "available":
			out.Results = append(out.Results, opResult{N: r.Available()})
		}
	}
	out.Header = append([]int32(nil), r.H[:]...)
	out.Data = r.D
	return out
}

// TestAudioRingMatchesTheWorklet runs the same random operations through the model and through the ring of the worklet under Node,
// and compares every answer and the memory at the end, from counters that start far from 0 and some that are about to wrap.
func TestAudioRingMatchesTheWorklet(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed (sudo dnf install nodejs)")
	}
	worklet, err := filepath.Abs("../../webtool/audio_worklet.js")
	if err != nil {
		t.Fatal(err)
	}
	runner, err := filepath.Abs("testdata/ring_runner.js")
	if err != nil {
		t.Fatal(err)
	}
	starts := []struct {
		name        string
		write, read uint32
	}{
		{"from zero", 0, 0},
		{"from the middle", 1000, 1000},
		{"about to wrap", 0xFFFFFFF0, 0xFFFFFFF0},
		{"already wrapped", 5, 5},
		{"a few frames waiting", 0xFFFFFFFA, 0xFFFFFFF8},
	}
	for _, channels := range []int{1, 2, 6} {
		for _, st := range starts {
			for seed := int64(1); seed <= 3; seed++ {
				rng := rand.New(rand.NewSource(seed*100 + int64(channels)))
				ops := randomOps(rng, channels, 300)
				const capacity = 64
				want := runModel(channels, capacity, st.write, st.read, ops)
				in, err := json.Marshal(map[string]any{"channels": channels, "capacity": capacity, "write": int32(st.write), "read": int32(st.read), "ops": ops})
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(node, runner, worklet)
				cmd.Stdin = bytes.NewReader(in)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("node: %v\n%s", err, stderr.String())
				}
				var got ringRun
				if err := json.Unmarshal(out, &got); err != nil {
					t.Fatal(err)
				}
				if len(got.Results) != len(want.Results) {
					t.Fatalf("%d channels, %s, seed %d: %d results, want %d", channels, st.name, seed, len(got.Results), len(want.Results))
				}
				for i := range want.Results {
					if got.Results[i].N != want.Results[i].N || !sameFloats(got.Results[i].Out, want.Results[i].Out) {
						t.Fatalf("%d channels, %s, seed %d: operation %d (%s) gave %+v, the model %+v", channels, st.name, seed, i, ops[i].Op, got.Results[i], want.Results[i])
					}
				}
				for i := range want.Header {
					if got.Header[i] != want.Header[i] {
						t.Errorf("%d channels, %s, seed %d: header word %d is %d, the model's %d", channels, st.name, seed, i, got.Header[i], want.Header[i])
					}
				}
				if !sameFloats(got.Data, want.Data) {
					t.Errorf("%d channels, %s, seed %d: the memory differs", channels, st.name, seed)
				}
			}
		}
	}
}

func sameFloats(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] && !(math.IsNaN(float64(a[i])) && math.IsNaN(float64(b[i]))) {
			return false
		}
	}
	return true
}

// TestWorkletConstantsAreTheModelsConstants checks that the numbers of the header in the JavaScript are the ones of this package.
func TestWorkletConstantsAreTheModelsConstants(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed (sudo dnf install nodejs)")
	}
	worklet, err := filepath.Abs("../../webtool/audio_worklet.js")
	if err != nil {
		t.Fatal(err)
	}
	script := `const w = require(process.argv[1]); console.log(JSON.stringify([w.H_WRITE, w.H_READ, w.H_UNDERRUNS, w.H_FLAGS, w.H_CHANNELS, w.H_CAPACITY, w.H_OVERRUNS, w.H_PRIMED, w.HEADER_WORDS, w.FLAG_PAUSED, w.FLAG_CLOSED]))`
	out, err := exec.Command(node, "-e", script, worklet).Output()
	if err != nil {
		t.Fatal(err)
	}
	var got []int
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	want := []int{RingWrite, RingRead, RingUnderruns, RingFlags, RingChannels, RingCapacity, RingOverruns, RingPrimed, RingHeaderWords, RingFlagPaused, RingFlagClosed}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("constant %d is %d in the worklet and %d here", i, got[i], want[i])
		}
	}
}
