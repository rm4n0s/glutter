// Package obj reads Wavefront OBJ files, with the subset of MTL that matters to a viewer, into a model.Model (SPEC §27.9).
//
// Positions, texture coordinates and normals are read with negative indices (counted back from the last one read), polygons are cut
// into triangles (a fan when they are convex, ear clipping when not), `o` and `g` make the nodes, `usemtl` splits the faces of a node
// into one mesh for each material, and `l` and `p` make meshes of lines and of points. The material library named by `mtllib` is
// read through the same file system as the OBJ file; Kd, Ks, Ns, d and map_Kd are taken, and a library that is not there gives the
// default material and one line in the log. A position may have colors after it (`v x y z r g b`).
//
// The texture coordinate v is turned (v becomes 1-v) so that the model's texture coordinates have their origin at the top left,
// like the images of Glutter.
package obj

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

// TagOBJInvalid is the tag of the error of a file that is not a valid OBJ file: a number that is not one, an index that is zero or
// beyond the data, a face with fewer than three corners. The metadata has "line" and "reason".
//
// Flutter: no counterpart
const TagOBJInvalid = "OBJInvalid"

// Options are the options of Decode and Load.
//
// Flutter: no counterpart
type Options struct {
	// Name is the name of the model and of the node that holds the faces that come before any `o` or `g`. Load sets the base name of
	// the file when it is empty.
	Name string
	// Float64 keeps the positions in float64 when some of them have more digits than a float32 holds.
	Float64 bool
	// FS is where `mtllib` files and the images of materials are read from; Dir is the directory in it that paths are relative to.
	// Load sets both from its arguments. Without FS, a material library is not read, with one line in the log.
	FS  fs.FS
	Dir string
	// Logger gets the notes of the loader (a material library that cannot be read); slog.Default() when nil.
	Logger *slog.Logger
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// Decode reads an OBJ file from r. It checks ctx while it reads and parses.
//
// It fails with model.TagModelReadFailed when the reader fails or ctx is done, and with TagOBJInvalid for a file that is not OBJ.
//
// Flutter: no counterpart
func Decode(ctx context.Context, r io.Reader, opts ...Options) (*model.Model, error) {
	o := intopts.First(opts)
	data, err := model.ReadAll(ctx, r)
	if err != nil {
		return nil, err
	}
	return decode(ctx, data, o)
}

// Load reads the OBJ file name of fsys, and the material libraries and images that it names from the same file system, in the
// directory of the file. The model's name is the file's base name without its extension, unless the options give one.
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
	o.FS, o.Dir = fsys, path.Dir(name)
	data, err := model.ReadAll(ctx, f)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			return nil, appErr.SetMetadata("file", name)
		}
		return nil, err
	}
	m, err := decode(ctx, data, o)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			return nil, appErr.SetMetadata("file", name)
		}
		return nil, err
	}
	return m, nil
}
