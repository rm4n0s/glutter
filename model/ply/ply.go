// Package ply reads PLY files, the ASCII kind and the binary kind in both byte orders (SPEC §28.5), into a model.Model.
//
// A file has a vertex element with the position (x, y and z) and optionally the normal (nx, ny, nz), the color (red, green, blue and
// alpha) and the texture coordinates (s and t, u and v, texture_u and texture_v), and a face element whose list property
// vertex_indices (or vertex_index) names the corners of each polygon, which is cut into triangles. A file with no faces is a point
// cloud: its mesh is of points, which viewport3d draws with its point material. Positions that the file stores as double are kept in
// float64 (model.Mesh.Positions64) when some of them are not exact in float32. Other elements and properties are skipped, with one
// line in the log for each kind.
package ply

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	intopts "github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/model"
)

// TagPLYInvalid is the tag of the error of a file that is not a valid PLY file: a bad header, a body that is cut short, a number that
// is not one or that is not finite, an index that names no vertex, a file with no vertex or no triangle. The metadata has "reason" (a
// short fixed phrase), and "line" (the header, and the body of an ASCII file) or "offset" (the byte, in a binary body).
//
// Flutter: no counterpart
const TagPLYInvalid = "PLYInvalid"

// Options are the options of Decode and Load.
//
// Flutter: no counterpart
type Options struct {
	// Name is the name of the model, its node and its mesh. Load sets the base name of the file when it is empty.
	Name string
	// Logger gets the notes of the loader (the elements and properties that it skipped, the faces that it dropped); slog.Default()
	// when nil.
	Logger *slog.Logger
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// Decode reads a PLY file from r. It checks ctx while it reads and parses.
//
// It fails with model.TagModelReadFailed when the reader fails or ctx is done, and with TagPLYInvalid for a file that is not PLY.
//
// Flutter: no counterpart
func Decode(ctx context.Context, r io.Reader, opts ...Options) (*model.Model, error) {
	o := intopts.First(opts)
	data, err := model.ReadAll(ctx, r)
	if err != nil {
		return nil, err
	}
	return decodeBytes(ctx, data, o)
}

// Load reads the PLY file name of fsys. The model's name is the file's base name without its extension, unless the options give one.
//
// Flutter: no counterpart
func Load(ctx context.Context, fsys fs.FS, name string, opts ...Options) (*model.Model, error) {
	o := intopts.First(opts)
	f, err := fsys.Open(name)
	if err != nil {
		return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("file", name)
	}
	defer func() { _ = f.Close() }() // the file is read; a failed close of a read-only file loses nothing
	if o.Name == "" {
		o.Name = strings.TrimSuffix(path.Base(name), path.Ext(name))
	}
	data, err := model.ReadAll(ctx, f)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			return nil, appErr.SetMetadata("file", name)
		}
		return nil, err
	}
	m, err := decodeBytes(ctx, data, o)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			return nil, appErr.SetMetadata("file", name)
		}
		return nil, err
	}
	return m, nil
}

// decodeBytes reads the header and then the body.
func decodeBytes(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	h, err := parseHeader(data)
	if err != nil {
		return nil, err
	}
	d := &decoder{ctx: ctx, o: o, h: h, c: cursor{data: data, pos: h.bodyAt, line: h.lines + 1, format: h.format}}
	return d.run()
}
