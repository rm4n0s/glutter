// Runs a sequence of operations on the ring of the audio worklet (internal/webtool/audio_worklet.js) and prints what each one
// returned and the state at the end, as JSON, for the test of the Go model (audioring_test.go) to compare. The input is JSON on
// stdin: {channels, capacity, write, read, ops: [...]}; write and read are the counters to start from (they let a sequence cross
// the wrap of the 32-bit counters).
"use strict";
const fs = require("fs");
const path = require("path");
const worklet = require(process.argv[2]);

const input = JSON.parse(fs.readFileSync(0, "utf8"));
const ring = worklet.makeRing(input.channels, input.capacity, null);
Atomics.store(ring.h, worklet.H_WRITE, input.write | 0);
Atomics.store(ring.h, worklet.H_READ, input.read | 0);

const results = [];
for (const op of input.ops) {
  // The Go side leaves out the fields that are empty.
  op.samples = op.samples || [];
  op.frames = op.frames | 0;
  op.flags = op.flags | 0;
  switch (op.op) {
    case "write": {
      const n = ring.write(Float32Array.from(op.samples), op.samples.length / input.channels);
      results.push({ n });
      break;
    }
    case "read": {
      const dst = new Float32Array(op.frames * input.channels).fill(-1);
      const n = ring.read(dst, op.frames);
      results.push({ n, out: Array.from(dst) });
      break;
    }
    case "play": {
      const dst = new Float32Array(op.frames * input.channels).fill(-1);
      const n = ring.play(dst, op.frames);
      results.push({ n, out: Array.from(dst) });
      break;
    }
    case "record": {
      const n = ring.record(Float32Array.from(op.samples), op.frames);
      results.push({ n });
      break;
    }
    case "flags": {
      Atomics.store(ring.h, worklet.H_FLAGS, op.flags);
      results.push({});
      break;
    }
    case "available": {
      results.push({ n: ring.available() });
      break;
    }
    default:
      throw new Error("unknown op " + op.op);
  }
}
process.stdout.write(
  JSON.stringify({ results, header: Array.from(ring.h), data: Array.from(ring.d) }),
);
