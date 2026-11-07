package web

import (
	"math/bits"
	"time"
)

// The ring of frames between the Go program and the audio worklet of the web backend (SPEC §28.3, step 31.6). This file is the
// plain-Go model of the ring that internal/webtool/audio_worklet.js implements: the same header, the same counters, the same
// algorithm. The browser's side of the ring is memory of the page (a SharedArrayBuffer, which Go reads and writes through
// syscall/js), so the Go program does not run this model there; the model is what the tests of both sides agree on: a test runs the
// same operations through it and through the JavaScript, and compares what they return (audioring_test.go).

// The words of the header of a ring, in the order of the JavaScript's H_* constants.
const (
	// RingWrite is the number of frames written, an unsigned counter that wraps at 2^32.
	//
	// Flutter: no counterpart
	RingWrite = 0
	// RingRead is the number of frames read, the same.
	//
	// Flutter: no counterpart
	RingRead = 1
	// RingUnderruns counts the render quanta in which a playback found too few frames, after the first frame was played.
	//
	// Flutter: no counterpart
	RingUnderruns = 2
	// RingFlags has RingFlagPaused and RingFlagClosed.
	//
	// Flutter: no counterpart
	RingFlags = 3
	// RingChannels is the number of channels of a frame.
	//
	// Flutter: no counterpart
	RingChannels = 4
	// RingCapacity is the number of frames the ring holds, a power of two.
	//
	// Flutter: no counterpart
	RingCapacity = 5
	// RingOverruns counts the render quanta in which a capture found no room.
	//
	// Flutter: no counterpart
	RingOverruns = 6
	// RingPrimed is 1 after the first frame was played: until then silence is not an underrun.
	//
	// Flutter: no counterpart
	RingPrimed = 7
	// RingHeaderWords is the size of the header in 32-bit words; the frames follow it.
	//
	// Flutter: no counterpart
	RingHeaderWords = 8
)

// The bits of RingFlags.
const (
	// RingFlagPaused is a device that its owner paused: a playback plays silence and keeps its frames, a capture drops what it hears.
	//
	// Flutter: no counterpart
	RingFlagPaused = 1
	// RingFlagClosed is a device that was closed: the processor ends.
	//
	// Flutter: no counterpart
	RingFlagClosed = 2
)

// AudioRing is the model of the ring: a header of RingHeaderWords words and interleaved frames in a slice of float32.
//
// Flutter: no counterpart
type AudioRing struct {
	H        [RingHeaderWords]int32
	D        []float32
	Channels int
	Capacity int
	mask     uint32
}

// NewAudioRing makes a ring of capacity frames of channels channels. The capacity is rounded up to a power of two (the counters wrap
// at 2^32, which only a power of two divides evenly).
//
// Flutter: no counterpart
func NewAudioRing(channels, capacity int) *AudioRing {
	capacity = PowerOfTwo(capacity)
	r := &AudioRing{D: make([]float32, capacity*channels), Channels: channels, Capacity: capacity, mask: uint32(capacity - 1)}
	r.H[RingChannels] = int32(channels)
	r.H[RingCapacity] = int32(capacity)
	return r
}

// PowerOfTwo is the smallest power of two that is not less than n (1 for n of 1 or less).
//
// Flutter: no counterpart
func PowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

// Available is the number of frames that wait to be read.
//
// Flutter: no counterpart
func (r *AudioRing) Available() uint32 { return uint32(r.H[RingWrite]) - uint32(r.H[RingRead]) }

// Free is the number of frames that fit.
//
// Flutter: no counterpart
func (r *AudioRing) Free() int { return r.Capacity - int(r.Available()) }

// Write writes as many whole frames of src as fit and returns how many.
//
// Flutter: no counterpart
func (r *AudioRing) Write(src []float32) int {
	frames := len(src) / r.Channels
	n := min(frames, r.Free())
	w := uint32(r.H[RingWrite])
	for i := 0; i < n; i++ {
		at := int((w+uint32(i))&r.mask) * r.Channels
		copy(r.D[at:at+r.Channels], src[i*r.Channels:(i+1)*r.Channels])
	}
	r.H[RingWrite] = int32(w + uint32(n))
	return n
}

// Read reads up to len(dst) / channels frames into dst and returns how many.
//
// Flutter: no counterpart
func (r *AudioRing) Read(dst []float32) int {
	n := min(len(dst)/r.Channels, int(r.Available()))
	rd := uint32(r.H[RingRead])
	for i := 0; i < n; i++ {
		at := int((rd+uint32(i))&r.mask) * r.Channels
		copy(dst[i*r.Channels:(i+1)*r.Channels], r.D[at:at+r.Channels])
	}
	r.H[RingRead] = int32(rd + uint32(n))
	return n
}

// Play is what a playback does for a render quantum of frames frames: it reads what there is into dst, fills the rest with silence
// and, once a frame has been played, counts an underrun when the frames were not all there. A paused ring plays silence and keeps
// its frames. It returns how many frames it read.
//
// Flutter: no counterpart
func (r *AudioRing) Play(dst []float32, frames int) int {
	ch := r.Channels
	if r.H[RingFlags]&RingFlagPaused != 0 {
		clear(dst[:frames*ch])
		return 0
	}
	n := r.Read(dst[:frames*ch])
	if n < frames {
		clear(dst[n*ch : frames*ch])
		if r.H[RingPrimed] != 0 {
			r.H[RingUnderruns]++
		}
	}
	if n > 0 {
		r.H[RingPrimed] = 1
	}
	return n
}

// Record is what a capture does with a render quantum of frames frames (interleaved in src): it writes what fits, and counts an
// overrun when it all did not. A paused ring drops the quantum. It returns how many frames it wrote.
//
// Flutter: no counterpart
func (r *AudioRing) Record(src []float32, frames int) int {
	if r.H[RingFlags]&RingFlagPaused != 0 {
		return 0
	}
	n := r.Write(src[:frames*r.Channels])
	if n < frames {
		r.H[RingOverruns]++
	}
	return n
}

// FramesToProduce is how many frames a producer puts in a ring that holds capacity frames and has write and read as its counters, to
// bring what waits to target frames (and no more than fits): 0 when it has that much already.
//
// Flutter: no counterpart
func FramesToProduce(write, read uint32, capacity, target int) int {
	waiting := int(write - read)
	return max(0, min(target, capacity)-waiting)
}

// RingFrames is the capacity, in frames, of a ring that holds a buffer of the given length at a rate: the next power of two, with room
// for 20% more than the buffer (so that a producer that tops it up to the buffer never finds it full) and 1024 frames at least.
//
// Flutter: no counterpart
func RingFrames(rate int, buffer time.Duration) int {
	frames := int(int64(rate) * int64(buffer) / int64(time.Second))
	return PowerOfTwo(max(1024, frames+frames/5))
}
