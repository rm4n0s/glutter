// Package wav reads WAV files (SPEC §28.3): RIFF files with linear PCM of 8, 16, 24 or 32 bits (also in the WAVE_FORMAT_EXTENSIBLE
// form) or IEEE float of 32 or 64 bits, with 1 to 8 channels. The samples come out as float32, ready for audio.Stream.Put. Other
// codecs (ADPCM, Ogg Vorbis, MP3, Opus) are not in Glutter: an app decodes them itself and puts the samples. Flutter has no audio.
//
// It follows the subset of src/audio/SDL_wave.c of SDL 3 (zlib license, third_party/sdl/LICENSE) that the format list above is,
// translated to Go and changed: the chunks are read one after another, those that are not "fmt " and "data" are skipped, and a data
// chunk that is longer than the file (a file written to a pipe has the size 0 or 0xFFFFFFFF) takes what there is.
//
// Flutter: no counterpart
package wav

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio"
)

// Audio is a decoded WAV file.
//
// Flutter: no counterpart
type Audio struct {
	// Format is the sample rate and the number of channels.
	Format audio.Format
	// Samples are the interleaved samples from -1 to 1, a whole number of frames.
	Samples []float32
	// BitsPerSample is the size of the samples in the file, and Float says that they were IEEE float.
	BitsPerSample int
	Float         bool
}

// Frames is the number of frames of the file.
//
// Flutter: no counterpart
func (a *Audio) Frames() int { return len(a.Samples) / a.Format.Channels }

// Duration is how long the file plays at its own rate.
//
// Flutter: no counterpart
func (a *Audio) Duration() time.Duration { return a.Format.Duration(a.Frames()) }

// The formats of the "fmt " chunk.
const (
	formatPCM        = 1
	formatFloat      = 3
	formatExtensible = 0xFFFE
)

// maxChunk is the most that is read of a data chunk, which keeps a file that lies about its size from filling the memory.
const maxChunk = 1 << 31

// Decode reads a WAV file from r. It fails with WAVInvalid when the file is not a WAV file of the subset above (with a message that
// says what is wrong), with WAVReadFailed when r fails or ctx ends, and the data it did read is dropped.
//
// Flutter: no counterpart
func Decode(ctx context.Context, r io.Reader) (*Audio, error) {
	cr := &ctxReader{ctx, r}
	var head [12]byte
	if _, err := io.ReadFull(cr, head[:]); err != nil {
		if endOfFile(err) {
			return nil, errors.New(TagInvalid, "the file has no RIFF header")
		}
		return nil, errors.NewErr(TagReadFailed, err).SetMetadata("while", "the file has no RIFF header")
	}
	if string(head[0:4]) != "RIFF" && string(head[0:4]) != "RF64" {
		return nil, errors.New(TagInvalid, "the file is not a RIFF file", "magic", string(head[0:4]))
	}
	if string(head[8:12]) != "WAVE" {
		return nil, errors.New(TagInvalid, "the RIFF file is not a WAVE file", "type", string(head[8:12]))
	}
	var f *format
	var data []byte
	haveData := false
	for !haveData {
		var ch [8]byte
		if _, err := io.ReadFull(cr, ch[:]); err != nil {
			if endOfFile(err) {
				if f == nil {
					return nil, errors.New(TagInvalid, "the file ends before its fmt chunk")
				}
				return nil, errors.New(TagInvalid, "the file ends before its data chunk")
			}
			return nil, errors.NewErr(TagReadFailed, err).SetMetadata("while", "the chunks")
		}
		id, size := string(ch[0:4]), int64(binary.LittleEndian.Uint32(ch[4:8]))
		switch id {
		case "fmt ":
			if size < 16 || size > 1<<16 {
				return nil, errors.New(TagInvalid, "the fmt chunk has a size that no WAV file has", "size", size)
			}
			buf := make([]byte, size+size&1)
			if _, err := io.ReadFull(cr, buf); err != nil {
				if endOfFile(err) {
					return nil, errors.New(TagInvalid, "the file ends inside its fmt chunk")
				}
				return nil, errors.NewErr(TagReadFailed, err).SetMetadata("while", "the file ends inside its fmt chunk")
			}
			var err error
			if f, err = parseFormat(buf[:size]); err != nil {
				return nil, err
			}
		case "data":
			if f == nil {
				return nil, errors.New(TagInvalid, "the data chunk comes before the fmt chunk")
			}
			// A size of 0 or 0xFFFFFFFF is a file that was written to a pipe: the data goes to the end of the file.
			limit := size
			if size == 0 || size == 0xFFFFFFFF {
				limit = maxChunk
			}
			var err error
			data, err = io.ReadAll(io.LimitReader(cr, min(limit, maxChunk)))
			if err != nil {
				// ReadAll takes the end of the file for the end of the data, so what fails here is the reader.
				return nil, errors.NewErr(TagReadFailed, err).SetMetadata("while", "the data chunk cannot be read")
			}
			haveData = true
		default:
			// Chunks of other kinds (LIST, fact, cue) are skipped; a chunk has an even number of bytes, so an odd size is padded.
			if _, err := io.CopyN(io.Discard, cr, size+size&1); err != nil {
				if endOfFile(err) {
					return nil, errors.New(TagInvalid, "the file ends inside a chunk that is skipped", "chunk", id)
				}
				return nil, errors.NewErr(TagReadFailed, err).SetMetadata("while", "a chunk is skipped")
			}
		}
	}
	return f.decode(data)
}

// DecodeBytes reads a WAV file that is in memory. The errors are those of Decode (it does no I/O, so it never fails with
// WAVReadFailed).
//
// Flutter: no counterpart
func DecodeBytes(data []byte) (*Audio, error) {
	return Decode(context.Background(), bytes.NewReader(data))
}

// format is the "fmt " chunk of a file that this package reads.
type format struct {
	float    bool
	channels int
	rate     int
	bits     int
	// block is the size of a frame in bytes.
	block int
}

func parseFormat(b []byte) (*format, error) {
	tag := int(binary.LittleEndian.Uint16(b[0:2]))
	f := &format{
		channels: int(binary.LittleEndian.Uint16(b[2:4])),
		rate:     int(binary.LittleEndian.Uint32(b[4:8])),
		block:    int(binary.LittleEndian.Uint16(b[12:14])),
		bits:     int(binary.LittleEndian.Uint16(b[14:16])),
	}
	if tag == formatExtensible {
		// WAVE_FORMAT_EXTENSIBLE: the real tag is the first two bytes of the sub format GUID, after the size of the extension (2),
		// the valid bits (2) and the channel mask (4).
		if len(b) < 40 {
			return nil, errors.New(TagInvalid, "the extensible fmt chunk is too short", "size", len(b))
		}
		tag = int(binary.LittleEndian.Uint16(b[24:26]))
	}
	switch tag {
	case formatPCM:
		if f.bits != 8 && f.bits != 16 && f.bits != 24 && f.bits != 32 {
			return nil, errors.New(TagInvalid, "the PCM samples have a size that is not 8, 16, 24 or 32 bits", "bits", f.bits)
		}
	case formatFloat:
		if f.bits != 32 && f.bits != 64 {
			return nil, errors.New(TagInvalid, "the float samples have a size that is not 32 or 64 bits", "bits", f.bits)
		}
		f.float = true
	default:
		return nil, errors.New(TagInvalid, "the samples are not linear PCM or IEEE float (the codec is not read here)", "format", tag)
	}
	if !(audio.Format{Rate: f.rate, Channels: f.channels}).Valid() {
		return nil, errors.New(TagInvalid, "the rate or the channels are outside what a stream takes", "rate", f.rate, "channels", f.channels)
	}
	if f.block != f.channels*f.bits/8 {
		return nil, errors.New(TagInvalid, "the size of a frame is not the channels times the size of a sample", "block", f.block, "channels", f.channels, "bits", f.bits)
	}
	return f, nil
}

// decode turns the data chunk into samples; a last partial frame is dropped.
func (f *format) decode(data []byte) (*Audio, error) {
	frames := len(data) / f.block
	if frames == 0 {
		return nil, errors.New(TagInvalid, "the data chunk holds no frame", "bytes", len(data))
	}
	n := frames * f.channels
	out := make([]float32, n)
	switch {
	case f.float && f.bits == 32:
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
		}
	case f.float:
		for i := range out {
			out[i] = float32(math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:])))
		}
	case f.bits == 8:
		for i := range out {
			out[i] = (float32(data[i]) - 128) * (1.0 / 128.0)
		}
	case f.bits == 16:
		for i := range out {
			out[i] = float32(int16(binary.LittleEndian.Uint16(data[i*2:]))) * (1.0 / 32768.0)
		}
	case f.bits == 24:
		for i := range out {
			v := int32(data[i*3]) | int32(data[i*3+1])<<8 | int32(int8(data[i*3+2]))<<16
			out[i] = float32(v) * (1.0 / 8388608.0)
		}
	default:
		for i := range out {
			out[i] = float32(int32(binary.LittleEndian.Uint32(data[i*4:]))) * (1.0 / 2147483648.0)
		}
	}
	return &Audio{Format: audio.Format{Rate: f.rate, Channels: f.channels}, Samples: out, BitsPerSample: f.bits, Float: f.float}, nil
}

// ctxReader stops reading when the context ends.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// endOfFile says that a read ended because the file did, which makes the file invalid; any other error is the reader's.
func endOfFile(err error) bool { return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) }

// The tags of the errors of this package (SPEC §28.7).
const (
	// TagInvalid is the tag of the error of a file that is not a WAV file of the subset this package reads: a message says what is
	// wrong (the magic, a chunk, the format, the rate, the channels, the size of the data).
	//
	// Flutter: no counterpart
	TagInvalid = "WAVInvalid"
	// TagReadFailed is the tag of the error of a reader that failed, or whose context ended, while the file was read.
	//
	// Flutter: no counterpart
	TagReadFailed = "WAVReadFailed"
)
