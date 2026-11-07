// glutter-audio-worklet.js: the audio worklet of Glutter's web backend (SPEC §28.3, step 31.6). glutter-web writes it next to
// index.html; the page loads it with audioWorklet.addModule. It has two processors:
//
//   glutter-playback  plays what the Go program puts in a ring of frames, and
//   glutter-capture   puts what the microphone gives in a ring of frames (or posts it, see below).
//
// The ring is a header of eight 32-bit integers followed by interleaved float32 frames. With a SharedArrayBuffer (the page is
// cross-origin isolated, which `glutter-web serve` makes it) the Go program and the worklet use the same memory, and the two sides
// never wait for each other: the producer writes the frames and then stores the new write counter, the consumer reads the frames
// and then stores the new read counter. Without one (a page that cannot be isolated) each side keeps its own ring and the frames
// travel by postMessage: the Go program sends {type:"data", samples} to a playback processor, which reports how many frames it has
// played ({type:"level", read}); a capture processor posts {type:"data", samples}.
//
// The ring model of the Go side (internal/platform/web/audioring.go) is the same algorithm: the counters are unsigned 32-bit
// numbers that wrap, the capacity is a power of two, and a frame is at counter & (capacity - 1). A test runs the same sequences
// through both and compares what they return.
"use strict";

// The words of the header.
const H_WRITE = 0; // frames written, a counter that wraps at 2^32
const H_READ = 1; // frames read, the same
const H_UNDERRUNS = 2; // the times that the playback found no frames (counted by the consumer, after the first frame played)
const H_FLAGS = 3; // bit 0: the device is paused; bit 1: the device is closed
const H_CHANNELS = 4;
const H_CAPACITY = 5; // frames, a power of two
const H_OVERRUNS = 6; // the times that the capture found no room (counted by the producer)
const H_PRIMED = 7; // 1 after the first frame was played: until then silence is not an underrun
const HEADER_WORDS = 8;
const HEADER_BYTES = HEADER_WORDS * 4;

const FLAG_PAUSED = 1;
const FLAG_CLOSED = 2;

// How often a playback without shared memory tells how much it played: every 4 quanta (about 10 ms).
const REPORT_QUANTA = 4;

class Ring {
  // header is an Int32Array of HEADER_WORDS words and data a Float32Array of capacity * channels samples, on shared memory or not.
  constructor(header, data) {
    this.h = header;
    this.d = data;
    this.channels = header[H_CHANNELS];
    this.capacity = header[H_CAPACITY];
    this.mask = this.capacity - 1;
  }

  // The number of frames that wait to be read.
  available() {
    return (Atomics.load(this.h, H_WRITE) - Atomics.load(this.h, H_READ)) >>> 0;
  }

  // The number of frames that fit.
  free() {
    return this.capacity - this.available();
  }

  // Writes up to frames frames of src (interleaved, from the sample at offset) and returns how many were written.
  write(src, frames, offset = 0) {
    const w = Atomics.load(this.h, H_WRITE) >>> 0;
    const n = Math.min(frames, this.capacity - ((w - Atomics.load(this.h, H_READ)) >>> 0));
    const ch = this.channels;
    for (let i = 0; i < n; i++) {
      const at = ((w + i) & this.mask) * ch;
      for (let c = 0; c < ch; c++) this.d[at + c] = src[offset + i * ch + c];
    }
    Atomics.store(this.h, H_WRITE, (w + n) | 0);
    return n;
  }

  // Reads up to frames frames into dst (interleaved, from the sample at offset) and returns how many were read.
  read(dst, frames, offset = 0) {
    const r = Atomics.load(this.h, H_READ) >>> 0;
    const n = Math.min(frames, (Atomics.load(this.h, H_WRITE) - r) >>> 0);
    const ch = this.channels;
    for (let i = 0; i < n; i++) {
      const at = ((r + i) & this.mask) * ch;
      for (let c = 0; c < ch; c++) dst[offset + i * ch + c] = this.d[at + c];
    }
    Atomics.store(this.h, H_READ, (r + n) | 0);
    return n;
  }

  // What a playback does for a render quantum of frames frames: it reads what there is into dst, which it then fills with silence
  // (the part of the quantum that has no frames), and counts an underrun when frames were wanted and not there, once the first one
  // has played. It returns how many frames it read.
  play(dst, frames) {
    const flags = Atomics.load(this.h, H_FLAGS);
    const ch = this.channels;
    if (flags & FLAG_PAUSED) {
      dst.fill(0, 0, frames * ch);
      return 0;
    }
    const n = this.read(dst, frames);
    if (n < frames) {
      dst.fill(0, n * ch, frames * ch);
      if (Atomics.load(this.h, H_PRIMED) !== 0) Atomics.add(this.h, H_UNDERRUNS, 1);
    }
    if (n > 0) Atomics.store(this.h, H_PRIMED, 1);
    return n;
  }

  // What a capture does with a render quantum (interleaved, frames frames): it writes what fits and counts an overrun when it all
  // did not. It returns how many frames were written.
  record(src, frames) {
    if (Atomics.load(this.h, H_FLAGS) & FLAG_PAUSED) return 0;
    const n = this.write(src, frames);
    if (n < frames) Atomics.add(this.h, H_OVERRUNS, 1);
    return n;
  }
}

// The header and the data of a ring that is made without shared memory.
function makeRing(channels, capacity, shared) {
  let header, data;
  if (shared) {
    header = new Int32Array(shared, 0, HEADER_WORDS);
    data = new Float32Array(shared, HEADER_BYTES, capacity * channels);
  } else {
    const buf = new ArrayBuffer(HEADER_BYTES + capacity * channels * 4);
    header = new Int32Array(buf, 0, HEADER_WORDS);
    data = new Float32Array(buf, HEADER_BYTES, capacity * channels);
  }
  header[H_CHANNELS] = channels;
  header[H_CAPACITY] = capacity;
  return new Ring(header, data);
}

if (typeof AudioWorkletProcessor === "function") {
  class GlutterPlayback extends AudioWorkletProcessor {
    constructor(options) {
      super();
      const o = options.processorOptions;
      this.channels = o.channels;
      this.ring = makeRing(o.channels, o.capacity, o.shared || null);
      this.shared = !!o.shared;
      this.scratch = new Float32Array(128 * o.channels);
      this.quanta = 0;
      this.port.onmessage = (e) => {
        const m = e.data;
        if (m.type === "data") {
          this.ring.write(m.samples, m.samples.length / this.channels);
        } else if (m.type === "flags") {
          Atomics.store(this.ring.h, H_FLAGS, m.flags);
        }
      };
    }

    process(inputs, outputs) {
      const out = outputs[0];
      const frames = out[0].length;
      const ch = this.channels;
      if (this.scratch.length < frames * ch) this.scratch = new Float32Array(frames * ch);
      this.ring.play(this.scratch, frames);
      for (let c = 0; c < out.length; c++) {
        const plane = out[c];
        const src = Math.min(c, ch - 1);
        for (let i = 0; i < frames; i++) plane[i] = this.scratch[i * ch + src];
      }
      if (!this.shared && ++this.quanta >= REPORT_QUANTA) {
        this.quanta = 0;
        this.port.postMessage({
          type: "level",
          read: Atomics.load(this.ring.h, H_READ) >>> 0,
          underruns: Atomics.load(this.ring.h, H_UNDERRUNS),
        });
      }
      return !(Atomics.load(this.ring.h, H_FLAGS) & FLAG_CLOSED);
    }
  }
  registerProcessor("glutter-playback", GlutterPlayback);

  class GlutterCapture extends AudioWorkletProcessor {
    constructor(options) {
      super();
      const o = options.processorOptions;
      this.channels = o.channels;
      this.shared = !!o.shared;
      this.ring = makeRing(o.channels, o.capacity, o.shared || null);
      this.scratch = new Float32Array(128 * o.channels);
      // Without shared memory the frames are posted in blocks of this many frames.
      this.block = o.block || 480;
      this.pending = new Float32Array(this.block * o.channels);
      this.pendingFrames = 0;
      this.port.onmessage = (e) => {
        if (e.data.type === "flags") Atomics.store(this.ring.h, H_FLAGS, e.data.flags);
      };
    }

    process(inputs) {
      const input = inputs[0];
      const ch = this.channels;
      if (!input || input.length === 0 || (Atomics.load(this.ring.h, H_FLAGS) & FLAG_PAUSED)) {
        return !(Atomics.load(this.ring.h, H_FLAGS) & FLAG_CLOSED);
      }
      const frames = input[0].length;
      if (this.scratch.length < frames * ch) this.scratch = new Float32Array(frames * ch);
      for (let i = 0; i < frames; i++) {
        for (let c = 0; c < ch; c++) this.scratch[i * ch + c] = input[Math.min(c, input.length - 1)][i];
      }
      if (this.shared) {
        this.ring.record(this.scratch, frames);
      } else {
        let done = 0;
        while (done < frames) {
          const n = Math.min(frames - done, this.block - this.pendingFrames);
          this.pending.set(this.scratch.subarray(done * ch, (done + n) * ch), this.pendingFrames * ch);
          this.pendingFrames += n;
          done += n;
          if (this.pendingFrames === this.block) {
            const samples = this.pending.slice();
            this.port.postMessage({ type: "data", samples }, [samples.buffer]);
            this.pendingFrames = 0;
          }
        }
      }
      return !(Atomics.load(this.ring.h, H_FLAGS) & FLAG_CLOSED);
    }
  }
  registerProcessor("glutter-capture", GlutterCapture);
}

// For the tests that run this file under Node: the ring and the numbers of the header.
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    Ring,
    makeRing,
    H_WRITE, H_READ, H_UNDERRUNS, H_FLAGS, H_CHANNELS, H_CAPACITY, H_OVERRUNS, H_PRIMED,
    HEADER_WORDS, HEADER_BYTES, FLAG_PAUSED, FLAG_CLOSED, REPORT_QUANTA,
  };
}
