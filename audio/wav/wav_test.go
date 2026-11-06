package wav

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio"
)

// file is a WAV file that a test writes.
type file struct {
	tag      uint16
	channels uint16
	rate     uint32
	bits     uint16
	// block overrides the size of a frame when not 0.
	block uint16
	data  []byte
	// extensible writes the extensible form of the fmt chunk, with the sub format tag.
	extensible bool
	// before are chunks that come before the fmt chunk (LIST); between come before the data chunk.
	before, between []byte
	// dataSize overrides the size that the data chunk says when not nil.
	dataSize *uint32
}

func chunk(id string, body []byte) []byte {
	out := []byte(id)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body)))
	out = append(out, body...)
	if len(body)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func (f file) bytes() []byte {
	block := f.block
	if block == 0 {
		block = f.channels * f.bits / 8
	}
	var fmtc []byte
	fmtc = binary.LittleEndian.AppendUint16(fmtc, f.tag)
	if f.extensible {
		fmtc[0], fmtc[1] = 0xFE, 0xFF
	}
	fmtc = binary.LittleEndian.AppendUint16(fmtc, f.channels)
	fmtc = binary.LittleEndian.AppendUint32(fmtc, f.rate)
	fmtc = binary.LittleEndian.AppendUint32(fmtc, f.rate*uint32(block))
	fmtc = binary.LittleEndian.AppendUint16(fmtc, block)
	fmtc = binary.LittleEndian.AppendUint16(fmtc, f.bits)
	if f.extensible {
		fmtc = binary.LittleEndian.AppendUint16(fmtc, 22)
		fmtc = binary.LittleEndian.AppendUint16(fmtc, f.bits)
		fmtc = binary.LittleEndian.AppendUint32(fmtc, 3)
		fmtc = binary.LittleEndian.AppendUint16(fmtc, f.tag)
		fmtc = append(fmtc, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71)
	}
	body := []byte("WAVE")
	body = append(body, f.before...)
	body = append(body, chunk("fmt ", fmtc)...)
	body = append(body, f.between...)
	d := chunk("data", f.data)
	if f.dataSize != nil {
		binary.LittleEndian.PutUint32(d[4:8], *f.dataSize)
	}
	body = append(body, d...)
	out := []byte("RIFF")
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body)))
	return append(out, body...)
}

func pcm16(v ...int16) []byte {
	var b []byte
	for _, s := range v {
		b = binary.LittleEndian.AppendUint16(b, uint16(s))
	}
	return b
}

func TestDecodeEveryFormat(t *testing.T) {
	f32 := func(v ...float32) []byte {
		var b []byte
		for _, s := range v {
			b = binary.LittleEndian.AppendUint32(b, math.Float32bits(s))
		}
		return b
	}
	f64 := func(v ...float64) []byte {
		var b []byte
		for _, s := range v {
			b = binary.LittleEndian.AppendUint64(b, math.Float64bits(s))
		}
		return b
	}
	tests := []struct {
		name  string
		file  file
		float bool
		bits  int
		want  []float32
	}{
		{"8 bits", file{tag: 1, channels: 1, rate: 8000, bits: 8, data: []byte{128, 255, 0, 192}}, false, 8, []float32{0, 127.0 / 128, -1, 0.5}},
		{"16 bits", file{tag: 1, channels: 1, rate: 44100, bits: 16, data: pcm16(0, 32767, -32768, 16384)}, false, 16, []float32{0, 32767.0 / 32768, -1, 0.5}},
		{"24 bits", file{tag: 1, channels: 1, rate: 48000, bits: 24, data: []byte{0, 0, 0, 0xFF, 0xFF, 0x7F, 0x00, 0x00, 0x80, 0x00, 0x00, 0x40}}, false, 24, []float32{0, 8388607.0 / 8388608, -1, 0.5}},
		{"32 bits", file{tag: 1, channels: 1, rate: 48000, bits: 32, data: binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, 0x40000000), 0x80000000)}, false, 32, []float32{0.5, -1}},
		{"float 32", file{tag: 3, channels: 1, rate: 48000, bits: 32, data: f32(0.25, -0.75, 1.5)}, true, 32, []float32{0.25, -0.75, 1.5}},
		{"float 64", file{tag: 3, channels: 1, rate: 96000, bits: 64, data: f64(0.25, -0.5)}, true, 64, []float32{0.25, -0.5}},
		{"stereo", file{tag: 1, channels: 2, rate: 22050, bits: 16, data: pcm16(16384, -16384, 8192, -8192)}, false, 16, []float32{0.5, -0.5, 0.25, -0.25}},
		{"extensible PCM", file{tag: 1, channels: 2, rate: 48000, bits: 16, extensible: true, data: pcm16(16384, -16384)}, false, 16, []float32{0.5, -0.5}},
		{"extensible float", file{tag: 3, channels: 1, rate: 48000, bits: 32, extensible: true, data: f32(0.5)}, true, 32, []float32{0.5}},
		{"skips other chunks", file{tag: 1, channels: 1, rate: 8000, bits: 16, before: chunk("LIST", []byte("INFOabc")), between: chunk("fact", []byte{1, 0, 0, 0}), data: pcm16(16384)}, false, 16, []float32{0.5}},
		{"a partial frame is dropped", file{tag: 1, channels: 2, rate: 8000, bits: 16, data: append(pcm16(16384, 16384), 7, 7)}, false, 16, []float32{0.5, 0.5}},
		{"eight channels", file{tag: 1, channels: 8, rate: 48000, bits: 16, data: pcm16(1, 2, 3, 4, 5, 6, 7, 8)}, false, 16,
			[]float32{1.0 / 32768, 2.0 / 32768, 3.0 / 32768, 4.0 / 32768, 5.0 / 32768, 6.0 / 32768, 7.0 / 32768, 8.0 / 32768}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := DecodeBytes(tt.file.bytes())
			if err != nil {
				t.Fatal(err)
			}
			if a.Format.Channels != int(tt.file.channels) || a.Format.Rate != int(tt.file.rate) || a.BitsPerSample != tt.bits || a.Float != tt.float {
				t.Errorf("format = %+v bits %d float %v", a.Format, a.BitsPerSample, a.Float)
			}
			if len(a.Samples) != len(tt.want) {
				t.Fatalf("samples = %v, want %v", a.Samples, tt.want)
			}
			for i := range tt.want {
				if a.Samples[i] != tt.want[i] {
					t.Errorf("sample %d = %v, want %v", i, a.Samples[i], tt.want[i])
				}
			}
			if a.Frames() != len(tt.want)/int(tt.file.channels) {
				t.Errorf("frames = %d", a.Frames())
			}
		})
	}
}

func TestDecodeDataThatTakesWhatThereIs(t *testing.T) {
	// A file written to a pipe has the size 0 or 0xFFFFFFFF, and a file that was cut is shorter than its chunk says.
	for _, size := range []uint32{0, 0xFFFFFFFF, 1000} {
		f := file{tag: 1, channels: 1, rate: 8000, bits: 16, data: pcm16(16384, -16384, 8192), dataSize: &size}
		a, err := DecodeBytes(f.bytes())
		if err != nil {
			t.Fatalf("size %#x: %v", size, err)
		}
		if len(a.Samples) != 3 {
			t.Errorf("size %#x: %d samples", size, len(a.Samples))
		}
	}
	// A data chunk that says less than there is takes what it says.
	size := uint32(4)
	a, err := DecodeBytes(file{tag: 1, channels: 1, rate: 8000, bits: 16, data: pcm16(1, 2, 3, 4), dataSize: &size}.bytes())
	if err != nil || len(a.Samples) != 2 {
		t.Errorf("a short chunk: %v, %v", a, err)
	}
	d := file{tag: 1, channels: 1, rate: 48000, bits: 16, data: pcm16(16384, 16384, 16384, 16384, 16384, 16384, 16384, 16384, 16384, 16384)}
	got, err := DecodeBytes(d.bytes())
	if err != nil || got.Duration() <= 0 {
		t.Errorf("duration %v, %v", got, err)
	}
}

func TestDecodeInvalidFilesHaveTheirRoutes(t *testing.T) {
	good := file{tag: 1, channels: 2, rate: 44100, bits: 16, data: pcm16(1, 2, 3, 4)}
	goodBytes := good.bytes()
	tests := []struct {
		name  string
		data  []byte
		route string
		msg   string
	}{
		{"empty", nil, "Decode.WAVInvalid", "RIFF header"},
		{"short header", goodBytes[:8], "Decode.WAVInvalid", "RIFF header"},
		{"not RIFF", append([]byte("OggS"), goodBytes[4:]...), "Decode.WAVInvalid", "not a RIFF file"},
		{"not WAVE", append(append([]byte("RIFF\x00\x00\x00\x00"), "AVI "...), goodBytes[12:]...), "Decode.WAVInvalid", "not a WAVE file"},
		{"ends before fmt", goodBytes[:12], "Decode.WAVInvalid", "before its fmt chunk"},
		{"ends before data", goodBytes[:12+8+16], "Decode.WAVInvalid", "before its data chunk"},
		{"ends inside fmt", goodBytes[:12+8+10], "Decode.WAVInvalid", "inside its fmt chunk"},
		{"data before fmt", append(append([]byte("RIFF\x00\x00\x00\x00WAVE"), chunk("data", []byte{1, 2})...), chunk("fmt ", make([]byte, 16))...), "Decode.WAVInvalid", "before the fmt chunk"},
		{"ends inside a skipped chunk", append(append([]byte(nil), goodBytes[:12]...), append([]byte("LIST\xff\x00\x00\x00"), 1, 2, 3)...), "Decode.WAVInvalid", "inside a chunk that is skipped"},
		{"ADPCM", file{tag: 2, channels: 1, rate: 8000, bits: 4, block: 1, data: []byte{1, 2}}.bytes(), "Decode->parseFormat.WAVInvalid", "not linear PCM"},
		{"PCM of 12 bits", file{tag: 1, channels: 1, rate: 8000, bits: 12, block: 2, data: []byte{1, 2}}.bytes(), "Decode->parseFormat.WAVInvalid", "8, 16, 24 or 32"},
		{"float of 16 bits", file{tag: 3, channels: 1, rate: 8000, bits: 16, data: []byte{1, 2}}.bytes(), "Decode->parseFormat.WAVInvalid", "32 or 64"},
		{"no channels", file{tag: 1, channels: 0, rate: 8000, bits: 16, block: 2, data: []byte{1, 2}}.bytes(), "Decode->parseFormat.WAVInvalid", "outside what a stream takes"},
		{"nine channels", file{tag: 1, channels: 9, rate: 8000, bits: 16, data: make([]byte, 18)}.bytes(), "Decode->parseFormat.WAVInvalid", "outside what a stream takes"},
		{"rate 0", file{tag: 1, channels: 1, rate: 0, bits: 16, data: []byte{1, 2}}.bytes(), "Decode->parseFormat.WAVInvalid", "outside what a stream takes"},
		{"wrong block size", file{tag: 1, channels: 2, rate: 8000, bits: 16, block: 3, data: make([]byte, 12)}.bytes(), "Decode->parseFormat.WAVInvalid", "size of a frame"},
		{"short extensible", func() []byte {
			f := file{tag: 1, channels: 1, rate: 8000, bits: 16, data: []byte{1, 2}}
			b := f.bytes()
			b[20], b[21] = 0xFE, 0xFF // the tag of the fmt chunk becomes WAVE_FORMAT_EXTENSIBLE, and the chunk is too short
			return b
		}(), "Decode->parseFormat.WAVInvalid", "extensible"},
		{"no frame", file{tag: 1, channels: 2, rate: 8000, bits: 16, data: []byte{1, 2, 3}}.bytes(), "Decode->format.decode.WAVInvalid", "no frame"},
		{"fmt chunk of 4 bytes", func() []byte {
			b := append([]byte("RIFF\x00\x00\x00\x00WAVE"), chunk("fmt ", []byte{1, 0, 1, 0})...)
			return b
		}(), "Decode.WAVInvalid", "size that no WAV file has"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeBytes(tt.data)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagInvalid || !appErr.HasRoute(tt.route) || !strings.Contains(appErr.Message, tt.msg) {
				t.Errorf("unexpected failure: tag %s, message %q, route %s", appErr.Tag, appErr.Message, appErr.Route())
			}
		})
	}
}

func TestDecodeReaderFailuresHaveTheirRoutes(t *testing.T) {
	good := file{tag: 1, channels: 1, rate: 8000, bits: 16, before: chunk("LIST", []byte("abcd")), data: pcm16(1, 2, 3, 4)}.bytes()
	boom := errors.New(tagReaderBroke, "the reader broke")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name  string
		ctx   context.Context
		r     io.Reader
		route string
	}{
		{"header", context.Background(), io.MultiReader(iotest.ErrReader(boom)), "Decode.WAVReadFailed"},
		{"chunk header", context.Background(), io.MultiReader(bytes.NewReader(good[:12]), iotest.ErrReader(boom)), "Decode.WAVReadFailed"},
		{"fmt chunk", context.Background(), io.MultiReader(bytes.NewReader(good[:12+8+4+8]), iotest.ErrReader(boom)), "Decode.WAVReadFailed"},
		{"skipped chunk", context.Background(), io.MultiReader(bytes.NewReader(good[:12+8+2]), iotest.ErrReader(boom)), "Decode.WAVReadFailed"},
		{"data chunk", context.Background(), io.MultiReader(bytes.NewReader(good[:len(good)-6]), iotest.ErrReader(boom)), "Decode.WAVReadFailed"},
		{"canceled context", canceled, bytes.NewReader(good), "Decode.WAVReadFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.ctx, tt.r)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagReadFailed || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, route %s", appErr.Tag, appErr.Route())
			}
		})
	}
	// A reader that gives its bytes one at a time reads the same file.
	a, err := Decode(context.Background(), iotest.OneByteReader(bytes.NewReader(good)))
	if err != nil || len(a.Samples) != 4 {
		t.Errorf("one byte at a time: %v, %v", a, err)
	}
}

func TestDecodedFileGoesThroughAStream(t *testing.T) {
	a, err := DecodeBytes(file{tag: 1, channels: 1, rate: 8000, bits: 16, data: pcm16(1000, -1000)}.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !(audio.Format{Rate: a.Format.Rate, Channels: a.Format.Channels}).Valid() {
		t.Errorf("the format %+v is not one that a stream takes", a.Format)
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(file{tag: 1, channels: 2, rate: 44100, bits: 16, data: pcm16(1, 2, 3, 4)}.bytes())
	f.Add(file{tag: 3, channels: 1, rate: 48000, bits: 32, extensible: true, data: make([]byte, 8)}.bytes())
	f.Add([]byte("RIFF\x00\x00\x00\x00WAVE"))
	f.Fuzz(func(t *testing.T, data []byte) {
		a, err := DecodeBytes(data)
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not a tagged one: %v", err)
			}
			return
		}
		if a.Format.Channels < 1 || a.Format.Channels > 8 || len(a.Samples) == 0 || len(a.Samples)%a.Format.Channels != 0 {
			t.Fatalf("a file that decoded has %d samples in %d channels", len(a.Samples), a.Format.Channels)
		}
	})
}

// tagReaderBroke is the tag of the failure of the readers of the tests.
const tagReaderBroke = "ReaderBroke"
