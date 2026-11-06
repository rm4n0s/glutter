// Package stl reads STL files, the binary and the ASCII kind (SPEC §27.9), into a model.Model.
//
// The kind is told by the content, not by the header: a binary file is one whose size is 84 bytes plus 50 for each triangle that its
// count says, even when its header begins with "solid" (a habit of some writers). Binary files may give a color for each triangle
// in the two attribute bytes, by the VisCAM and SolidView convention. Equal vertices are welded (exactly, or within a tolerance),
// and the normals are smooth across edges whose faces differ by less than a crease angle.
package stl

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	intopts "github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/model"
)

// TagSTLInvalid is the tag of the error of a file that is not a valid STL file: a truncated binary file, a number that is not one
// or that is not finite, a keyword in the wrong place, a file with no triangle. The metadata has "offset" (the byte, for a binary
// file) or "line" (for an ASCII file), and "reason".
//
// Flutter: no counterpart
const TagSTLInvalid = "STLInvalid"

// DefaultCreaseAngle is the angle between the normals of two triangles at a vertex above which the vertex is split in two, so that
// the edge stays sharp: 30 degrees.
//
// Flutter: no counterpart
const DefaultCreaseAngle = math.Pi / 6

// Options are the options of Decode and Load.
//
// Flutter: no counterpart
type Options struct {
	// Name is the name of the model, its node and its mesh. Load sets the base name of the file when it is empty.
	Name string
	// Float64 keeps the positions of an ASCII file in float64 when some of them have more digits than a float32 holds. A binary file
	// is float32 and is not changed by it.
	Float64 bool
	// WeldTolerance welds vertices that fall in the same cell of a grid of this size, in the units of the file. Zero welds the
	// vertices that are equal bit for bit (and -0 with 0), which is what CAD writers produce.
	WeldTolerance float64
	// Flat gives every triangle three vertices of its own with the normal of the triangle: nothing is welded and no normal is
	// smoothed.
	Flat bool
	// CreaseAngle is the angle in radians between the normals of two triangles at a vertex above which they do not share the
	// vertex. Zero is DefaultCreaseAngle; a negative angle smooths every edge that is shared.
	CreaseAngle float64
	// Logger gets the notes of the loader (triangles that were dropped); slog.Default() when nil.
	Logger *slog.Logger
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

func (o Options) cosCrease() float64 {
	switch {
	case o.CreaseAngle == 0:
		return math.Cos(DefaultCreaseAngle)
	case o.CreaseAngle < 0 || o.CreaseAngle >= math.Pi:
		return -2
	}
	return math.Cos(o.CreaseAngle)
}

// Decode reads an STL file from r. It checks ctx while it reads and parses.
//
// It fails with model.TagModelReadFailed when the reader fails or ctx is done, and with TagSTLInvalid for a file that is not STL.
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

// Load reads the STL file name of fsys. The model's name is the file's base name without its extension, unless the options give
// one.
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

// binaryHeader is the size of the header and the count of a binary file; binaryRecord the size of one triangle.
const (
	binaryHeader = 84
	binaryRecord = 50
)

// binaryCount is the count of triangles that the data says, or false when the data is too short to have one.
func binaryCount(data []byte) (uint64, bool) {
	if len(data) < binaryHeader {
		return 0, false
	}
	return uint64(data[80]) | uint64(data[81])<<8 | uint64(data[82])<<16 | uint64(data[83])<<24, true
}

func startsWithSolid(data []byte) bool {
	i := 0
	// A byte order mark and blanks may come first.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		i = 3
	}
	for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\r' || data[i] == '\n') {
		i++
	}
	return len(data)-i >= 5 && strings.EqualFold(string(data[i:i+5]), "solid")
}

// decodeBytes tells the kinds apart: a file whose size is exactly what its count of triangles needs is binary, even when its header
// begins with "solid"; of the others, one that begins with "solid" is ASCII, and the rest are binary (a binary file may have bytes
// after its triangles, and one that is cut short is reported as a binary file).
func decodeBytes(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	if count, ok := binaryCount(data); ok && binaryHeader+binaryRecord*count == uint64(len(data)) {
		return decodeBinary(ctx, data, o)
	}
	if startsWithSolid(data) {
		return decodeASCII(ctx, data, o)
	}
	return decodeBinary(ctx, data, o)
}

func newModel(name string) *model.Model {
	return &model.Model{Name: name}
}

func addMesh(m *model.Model, mesh *model.Mesh) {
	node := model.NewNode(mesh.Name)
	node.Meshes = []*model.Mesh{mesh}
	m.Meshes = append(m.Meshes, mesh)
	m.Nodes = append(m.Nodes, node)
}
