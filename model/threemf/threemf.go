// Package threemf reads 3MF files (SPEC §28.5) into a model.Model: the core specification of the 3D Manufacturing Format, a zip
// archive whose model part (3D/3dmodel.model, found through the package's relationships) is XML, and the base materials and colors
// of its materials extension.
//
// Objects with a mesh become meshes (one for each material of its triangles; colors of a color group are vertex colors, and the
// base material of a triangle is the material of its mesh), and objects with components become nodes with children. Every build item
// is a top node, with the transform of the item; an object that is used several times is built once, and its meshes are shared.
// The unit of the file is model.Model.Units; the positions are not scaled. Beam lattices, slices, production paths and textures are
// not read: a required extension that is not read is an error, one that is only used is left out with one line in the log.
package threemf

import (
	"archive/zip"
	"bytes"
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

// TagThreeMFInvalid is the tag of the error of a file that is not a valid 3MF file: an archive that is not a zip, a package with no
// model part, XML that is not well formed, a reference to an object or a property that is not there, an index outside the vertices,
// a required extension that is not read, a model with no item to build. The metadata has "reason" (a short fixed phrase) and "part"
// (the part of the archive), and for XML "line" and "column".
//
// Flutter: no counterpart
const TagThreeMFInvalid = "ThreeMFInvalid"

// Options are the options of Decode and Load.
//
// Flutter: no counterpart
type Options struct {
	// Name is the name of the model. Load sets the base name of the file when it is empty.
	Name string
	// Float64 keeps the positions in float64 when some of them have more digits than a float32 holds.
	Float64 bool
	// MaxModelBytes is the most that the model part may be once it is unpacked; a part that is larger is an error (a small archive
	// can unpack to any size). Zero is 1 GiB.
	MaxModelBytes int64
	// Logger gets the notes of the loader (the parts that are not read, the triangles that were dropped); slog.Default() when nil.
	Logger *slog.Logger
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// Limits that keep a hostile archive from asking for work or memory that the file cannot back.
const (
	// defaultMaxModelBytes is the most XML that the model part may hold once it is unpacked, when the options say nothing.
	defaultMaxModelBytes = 1 << 30
	// maxRelsBytes is the most that the relationships part may hold.
	maxRelsBytes = 1 << 20
	// maxDepth is the deepest nesting of components.
	maxDepth = 64
	// maxItems is the most build items, objects and components together that a file may name.
	maxItems = 1 << 20
)

// Decode reads a 3MF file from r. It checks ctx while it reads and parses.
//
// It fails with model.TagModelReadFailed when the reader fails or ctx is done, and with TagThreeMFInvalid for a file that is not 3MF.
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

// Load reads the 3MF file name of fsys. The model's name is the file's base name without its extension, unless the options give one.
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

// modelRelationship is the type of the relationship that names the model part.
const modelRelationship = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"

// defaultModelPart is the model part when the relationships do not name one.
const defaultModelPart = "3D/3dmodel.model"

// decodeBytes opens the archive, finds the model part and reads it.
func decodeBytes(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "not a zip archive")
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[strings.ToLower(strings.TrimPrefix(f.Name, "/"))] = f
	}
	part, err := findModelPart(files)
	if err != nil {
		return nil, err
	}
	rc, err := part.Open()
	if err != nil {
		return nil, errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "bad part", "part", part.Name)
	}
	defer func() { _ = rc.Close() }() // the part is read; a failed close of a read-only part loses nothing
	limit := o.MaxModelBytes
	if limit <= 0 {
		limit = defaultMaxModelBytes
	}
	d := &decoder{ctx: ctx, o: o, part: part.Name}
	// One byte more than the limit is read: a part that has it is too large.
	return d.run(&io.LimitedReader{R: rc, N: limit + 1})
}
