// Package gltf reads glTF 2.0 files, the text kind (.gltf, with its buffers and images in files or in data: URIs) and the binary kind
// (.glb), into a model.Model (SPEC §28.5).
//
// It reads meshes (every kind of primitive, made into lists of triangles, segments or points), materials of the metallic-roughness
// model with their textures and samplers, nodes, cameras, skins, morph targets, animations (as data: viewport3d plays them) and
// scenes. Accessors may be strided, sparse, normalized and quantized. The extensions that it reads are KHR_mesh_quantization,
// KHR_texture_transform, KHR_materials_emissive_strength, KHR_materials_unlit, KHR_lights_punctual and EXT_meshopt_compression. An
// extension that the file requires and that the package does not know (Draco, KTX2 and Basis among them) is an error; one that the
// file only uses is left out and noted once in the log.
//
// The format is the specification of the Khronos Group (https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html) and of its
// extensions; the decoder of EXT_meshopt_compression is internal/meshopt.
package gltf

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	intopts "github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/model"
)

// Options are the options of Decode and Load.
//
// Flutter: no counterpart
type Options struct {
	// Name is the name of the model. Load sets the base name of the file when it is empty.
	Name string
	// FS is where the buffers and the images that a .gltf file names by a path are read from; Dir is the directory in it that the
	// paths are relative to. Load sets both from its arguments. Without FS, a file that names a path fails with
	// model.TagModelReadFailed (a .glb file and data: URIs need none).
	FS  fs.FS
	Dir string
	// Logger gets the notes of the loader (an extension that is left out, a primitive that has nothing to draw);
	// slog.Default() when nil.
	Logger *slog.Logger
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// Decode reads a glTF file, text or binary (it is told by the first four bytes), from r. It checks ctx while it reads and builds.
//
// It fails with model.TagModelReadFailed when the reader or the file system fails or ctx is done, with TagGLTFInvalid for a file
// that is not valid glTF 2.0, and with TagGLTFExtensionUnsupported for a file that requires an extension that is not read.
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

// Load reads the glTF file name of fsys (.gltf or .glb), and the buffers and images that it names by a path from the same file
// system, in the directory of the file. The model's name is the file's base name without its extension, unless the options give
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

const (
	glbMagic     = 0x46546C67 // "glTF"
	glbChunkJSON = 0x4E4F534A // "JSON"
	glbChunkBIN  = 0x004E4942 // "BIN\0"
)

// splitGLB reads the container of a .glb file. It returns the JSON chunk and the binary chunk (nil when there is none), or the
// reason and the offset of the failure.
func splitGLB(data []byte) (jsonChunk, bin []byte, reason string, offset int) {
	if len(data) < 20 {
		return nil, nil, "the binary file is shorter than its header and its first chunk header", len(data)
	}
	if v := binary.LittleEndian.Uint32(data[4:]); v != 2 {
		return nil, nil, "the binary container is not version 2", 4
	}
	total := int(binary.LittleEndian.Uint32(data[8:]))
	if total < 20 || total > len(data) {
		return nil, nil, "the length of the binary file is not the length of its data", 8
	}
	data = data[:total]
	pos := 12
	first := true
	for pos+8 <= len(data) {
		n := int(binary.LittleEndian.Uint32(data[pos:]))
		kind := binary.LittleEndian.Uint32(data[pos+4:])
		start := pos + 8
		if n < 0 || n > len(data)-start {
			return nil, nil, "a chunk is longer than the file", pos
		}
		chunk := data[start : start+n]
		switch {
		case first && kind != glbChunkJSON:
			return nil, nil, "the first chunk is not the JSON chunk", pos
		case first:
			jsonChunk = chunk
		case kind == glbChunkBIN && bin == nil:
			bin = chunk
		}
		first = false
		// the chunks are padded to four bytes
		pos = start + (n+3)&^3
	}
	if jsonChunk == nil {
		return nil, nil, "the binary file has no JSON chunk", pos
	}
	return jsonChunk, bin, "", 0
}

// decode reads the container and the JSON, and builds the model.
func decode(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	var bin []byte
	if len(data) >= 4 && binary.LittleEndian.Uint32(data) == glbMagic {
		jsonChunk, chunk, reason, offset := splitGLB(data)
		if reason != "" {
			return nil, errors.New(TagGLTFInvalid, "the binary container is malformed", "path", "", "reason", reason, "offset", offset)
		}
		data, bin = jsonChunk, chunk
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		reason, where, offset := "the JSON is malformed", "", 0
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &syntaxErr):
			offset = int(syntaxErr.Offset)
		case errors.As(err, &typeErr):
			reason, where, offset = "a value has the wrong type", typeErr.Field, int(typeErr.Offset)
		}
		return nil, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", where, "reason", reason, "offset", offset)
	}

	l := newLoader(ctx, &doc, bin, o)
	return l.build()
}

// TagGLTFInvalid is the tag of the error of a file that is not valid glTF 2.0: a container or JSON that is malformed, an index that
// points nowhere, an accessor that reaches past its buffer, a component type that the attribute cannot have, data that does not
// decode. The metadata has "path", the JSON pointer of the place of the failure ("/meshes/0/primitives/1/attributes/NORMAL", empty
// for the whole file), and "reason"; "offset" is the byte of a binary container or of the JSON, "file" the name of the file that
// Load read, and a failure of a compressed buffer has "cause" with the decoder's reason.
//
// Flutter: no counterpart
const TagGLTFInvalid = "GLTFInvalid"

// TagGLTFExtensionUnsupported is the tag of the error of a file that requires an extension that this package does not read (Draco
// and KTX2 or Basis textures among them). The metadata has "extension", its name, and "path", "/extensionsRequired/<index>".
//
// Flutter: no counterpart
const TagGLTFExtensionUnsupported = "GLTFExtensionUnsupported"
