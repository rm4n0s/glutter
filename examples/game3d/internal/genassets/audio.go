package main

import (
	"encoding/binary"
	"math"
	"math/rand"
)

const sampleRate = 22050

// wavBytes is a mono WAV file of 16-bit samples.
func wavBytes(samples []float64) []byte {
	data := make([]byte, 0, 2*len(samples))
	for _, s := range samples {
		v := int16(math.Round(math.Max(-1, math.Min(1, s)) * 32767))
		data = binary.LittleEndian.AppendUint16(data, uint16(v))
	}
	out := make([]byte, 0, 44+len(data))
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(36+len(data)))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16)
	out = binary.LittleEndian.AppendUint16(out, 1) // linear PCM
	out = binary.LittleEndian.AppendUint16(out, 1) // mono
	out = binary.LittleEndian.AppendUint32(out, sampleRate)
	out = binary.LittleEndian.AppendUint32(out, 2*sampleRate)
	out = binary.LittleEndian.AppendUint16(out, 2)
	out = binary.LittleEndian.AppendUint16(out, 16)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	return append(out, data...)
}

func frames(seconds float64) int { return int(seconds * sampleRate) }

// tone adds a note: a sine with some of its third harmonic, with an attack of 4 ms and an exponential decay.
func tone(out []float64, at float64, freq, length, decay, gain float64) {
	start := frames(at)
	n := frames(length)
	for i := 0; i < n && start+i < len(out); i++ {
		t := float64(i) / sampleRate
		env := math.Exp(-t*decay) * math.Min(1, t/0.004)
		ph := 2 * math.Pi * freq * t
		out[start+i] += gain * env * (math.Sin(ph) + 0.25*math.Sin(3*ph))
	}
}

func makeCoin() []byte {
	out := make([]float64, frames(0.45))
	tone(out, 0, 1318.5, 0.12, 14, 0.35)
	tone(out, 0.07, 1975.5, 0.38, 9, 0.35)
	return wavBytes(out)
}

func makeJump() []byte {
	out := make([]float64, frames(0.22))
	phase := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		f := 280 + 520*t/0.22
		phase += 2 * math.Pi * f / sampleRate
		out[i] = 0.4 * math.Sin(phase) * math.Exp(-t*7) * math.Min(1, t/0.004)
	}
	return wavBytes(out)
}

func makeHit() []byte {
	rng := rand.New(rand.NewSource(7))
	out := make([]float64, frames(0.42))
	phase := 0.0
	last := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		f := 120 * math.Exp(-t*5)
		phase += 2 * math.Pi * f / sampleRate
		// Low-passed noise for the crunch, a falling sine for the thump.
		last += 0.35 * ((rng.Float64()*2 - 1) - last)
		out[i] = 0.55*math.Sin(phase)*math.Exp(-t*6) + 0.45*last*math.Exp(-t*14)
	}
	return wavBytes(out)
}

// makeMusic is eight seconds of a loop in A minor: a bass, a pad and an arpeggio, four chords of two seconds. The notes that
// ring past the end are wrapped to the beginning, so the loop has no click.
func makeMusic() []byte {
	length := 8.0
	out := make([]float64, frames(length+1))                                           // a second more, for the notes that ring past the end
	note := func(semitones float64) float64 { return 220 * math.Pow(2, semitones/12) } // 0 is A3
	chords := [][3]float64{{0, 3, 7}, {-4, 0, 3}, {3, 7, 10}, {-2, 2, 5}}              // Am, F, C, G
	bass := []float64{-24, -28, -21, -26}
	for c, chord := range chords {
		at := float64(c) * 2
		// The bass on each beat of the chord, the pad held for the two seconds.
		for b := 0; b < 4; b++ {
			tone(out, at+float64(b)*0.5, note(bass[c]), 0.45, 5, 0.22)
		}
		for _, s := range chord {
			tone(out, at, note(s-12), 2.4, 1.1, 0.07)
		}
		// Eighth notes in the order of an arpeggio, an octave up.
		order := []int{0, 1, 2, 1, 0, 1, 2, 1}
		for i, o := range order {
			tone(out, at+float64(i)*0.25, note(chord[o]+12), 0.5, 7, 0.10)
		}
	}
	// Notes that went past the end come back to the beginning.
	n := frames(length)
	for i := n; i < len(out); i++ {
		out[i-n] += out[i]
	}
	return wavBytes(out[:n])
}
