package model

import (
	"context"
	"io"
	"io/fs"
	"math"

	"github.com/rm4n0s/errors"
)

// TagModelReadFailed is the tag of the error of a loader that could not read its input: an I/O error of the reader or the file
// system, or a context that was done. The cause is wrapped, so errors.Is(err, context.Canceled) works; the metadata has the
// offset or the line that was reached, and the file name when there is one.
//
// Flutter: no counterpart
const TagModelReadFailed = "ModelReadFailed"

// readChunk is how many bytes ReadAll reads between two looks at the context.
const readChunk = 1 << 20

// sizeHint is how many bytes the reader will give, when it knows: a bytes.Reader or a strings.Reader has Len, and a file has Stat.
func sizeHint(r io.Reader) int {
	switch r := r.(type) {
	case interface{ Len() int }:
		return r.Len()
	case interface{ Stat() (fs.FileInfo, error) }:
		if info, err := r.Stat(); err == nil && info.Mode().IsRegular() && info.Size() < math.MaxInt32 {
			return int(info.Size())
		}
	}
	return 0
}

// ReadAll reads the whole input, as the loaders do, looking at the context between reads of a megabyte. It fails with
// ModelReadFailed for an error of the reader or a done context ("offset" is how many bytes were read).
//
// Flutter: no counterpart
func ReadAll(ctx context.Context, r io.Reader) ([]byte, error) {
	data := make([]byte, 0, max(sizeHint(r)+1, 512))
	for {
		if err := ctx.Err(); err != nil {
			return nil, errors.NewErr(TagModelReadFailed, err).SetMetadata("offset", len(data))
		}
		if len(data) == cap(data) {
			grown := make([]byte, len(data), 2*cap(data))
			copy(grown, data)
			data = grown
		}
		// At most a chunk at a time, so that a reader that gives a lot is not read for long between two looks at the context.
		n, err := r.Read(data[len(data):min(cap(data), len(data)+readChunk)])
		data = data[:len(data)+n]
		if err == io.EOF {
			return data, nil
		}
		if err != nil {
			return nil, errors.NewErr(TagModelReadFailed, err).SetMetadata("offset", len(data))
		}
	}
}
