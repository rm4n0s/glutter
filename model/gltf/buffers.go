package gltf

import (
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/meshopt"
	"github.com/rm4n0s/glutter/model"
)

// The names of the extension of the compressed buffer views: the one of the specification of 2022 and the one that the Khronos
// Group ratified later, whose JSON is the same.
const (
	extensionMeshopt    = "EXT_meshopt_compression"
	extensionMeshoptKHR = "KHR_meshopt_compression"
)

// view is the bytes of a buffer view, as the accessors read them: the stride is 0 for a view that is tightly packed.
type view struct {
	data   []byte
	stride int
}

// checkContext fails when the context is done.
func (l *loader) checkContext() error {
	if err := l.ctx.Err(); err != nil {
		return errors.NewErr(model.TagModelReadFailed, err).SetMetadata("phase", "build")
	}
	return nil
}

// resource reads what a URI names: a data: URI is decoded, any other is a path, relative to the directory of the file, that the
// file system reads. The path is a JSON pointer of the place that names it.
func (l *loader) resource(uri, where string) ([]byte, error) {
	if rest, ok := strings.CutPrefix(uri, "data:"); ok {
		meta, payload, found := strings.Cut(rest, ",")
		if !found {
			return nil, errors.New(TagGLTFInvalid, "a data URI has no comma", "path", where, "reason", "bad data URI")
		}
		if !strings.HasSuffix(meta, ";base64") {
			return nil, errors.New(TagGLTFInvalid, "a data URI is not base64", "path", where, "reason", "bad data URI")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", where, "reason", "bad data URI")
		}
		return data, nil
	}

	name, err := url.PathUnescape(uri)
	if err != nil {
		return nil, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", where, "reason", "bad URI", "uri", uri)
	}
	name = path.Join(l.opts.Dir, name)
	if !fs.ValidPath(name) || strings.HasPrefix(uri, "/") || strings.Contains(uri, "://") {
		return nil, errors.New(TagGLTFInvalid, "a URI is not a path inside the directory of the file", "path", where, "reason", "bad URI", "uri", uri)
	}
	if l.opts.FS == nil {
		return nil, errors.New(model.TagModelReadFailed, "the file names a path and the options have no file system", "path", where, "resource", name)
	}
	data, err := fs.ReadFile(l.opts.FS, name)
	if err != nil {
		return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("path", where, "resource", name)
	}
	return data, nil
}

// buffer is the bytes of a buffer, read when first used.
func (l *loader) buffer(i int, where string) ([]byte, error) {
	if i < 0 || i >= len(l.doc.Buffers) {
		return nil, errors.New(TagGLTFInvalid, "a buffer does not exist", "path", where, "reason", "index out of range", "index", i)
	}
	if b, ok := l.buffers[i]; ok {
		return b, nil
	}
	if err := l.checkContext(); err != nil {
		return nil, err
	}
	bj := &l.doc.Buffers[i]
	here := ptr("buffers", i)
	if bj.ByteLength == nil || *bj.ByteLength < 1 {
		return nil, errors.New(TagGLTFInvalid, "a buffer has no length", "path", here, "reason", "no byteLength")
	}
	var data []byte
	if bj.URI == "" {
		// the buffer of the binary chunk is the first
		if i != 0 || l.bin == nil {
			return nil, errors.New(TagGLTFInvalid, "a buffer has no URI and is not the binary chunk", "path", here, "reason", "no data")
		}
		data = l.bin
	} else {
		var err error
		if data, err = l.resource(bj.URI, ptr("buffers", i, "uri")); err != nil {
			return nil, err
		}
	}
	// the binary chunk may be padded to four bytes
	if len(data) < *bj.ByteLength {
		return nil, errors.New(TagGLTFInvalid, "a buffer is shorter than its byteLength", "path", here, "reason", "short buffer", "length", len(data), "byteLength", *bj.ByteLength)
	}
	data = data[:*bj.ByteLength]
	l.buffers[i] = data
	return data, nil
}

// view is the bytes of a buffer view, decompressed when it has EXT_meshopt_compression.
func (l *loader) view(i int, where string) (view, error) {
	if i < 0 || i >= len(l.doc.BufferViews) {
		return view{}, errors.New(TagGLTFInvalid, "a buffer view does not exist", "path", where, "reason", "index out of range", "index", i)
	}
	if v, ok := l.views[i]; ok {
		return v, nil
	}
	bv := &l.doc.BufferViews[i]
	here := ptr("bufferViews", i)
	if bv.ByteStride != 0 && (bv.ByteStride < 4 || bv.ByteStride > 252 || bv.ByteStride%4 != 0) {
		return view{}, errors.New(TagGLTFInvalid, "a buffer view has a byteStride that is not a multiple of four from 4 to 252", "path", ptr("bufferViews", i, "byteStride"), "reason", "bad byteStride", "byteStride", bv.ByteStride)
	}

	for _, name := range [...]string{extensionMeshoptKHR, extensionMeshopt} {
		raw, ok := bv.Extensions[name]
		if !ok {
			continue
		}
		v, err := l.meshoptView(raw, ptr("bufferViews", i, "extensions", name))
		if err != nil {
			return view{}, err
		}
		l.views[i] = v
		return v, nil
	}

	if bv.Buffer == nil || bv.ByteLength == nil {
		return view{}, errors.New(TagGLTFInvalid, "a buffer view has no buffer or no byteLength", "path", here, "reason", "missing property")
	}
	buf, err := l.buffer(*bv.Buffer, ptr("bufferViews", i, "buffer"))
	if err != nil {
		return view{}, err
	}
	if bv.ByteOffset < 0 || *bv.ByteLength < 1 || bv.ByteOffset > len(buf) || *bv.ByteLength > len(buf)-bv.ByteOffset {
		return view{}, errors.New(TagGLTFInvalid, "a buffer view reaches past its buffer", "path", here, "reason", "view out of bounds", "byteOffset", bv.ByteOffset, "byteLength", *bv.ByteLength, "bufferLength", len(buf))
	}
	v := view{data: buf[bv.ByteOffset : bv.ByteOffset+*bv.ByteLength], stride: bv.ByteStride}
	l.views[i] = v
	return v, nil
}

// meshoptView decodes a buffer view that EXT_meshopt_compression compresses. The view's own buffer, offset and length are the
// uncompressed fallback of the file and are not read.
func (l *loader) meshoptView(raw json.RawMessage, where string) (view, error) {
	var ext meshoptJSON
	if err := json.Unmarshal(raw, &ext); err != nil {
		return view{}, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", where, "reason", "a value has the wrong type")
	}
	if ext.Buffer == nil || ext.ByteLength == nil || ext.ByteStride == nil || ext.Count == nil || ext.Mode == "" {
		return view{}, errors.New(TagGLTFInvalid, "the compression of a buffer view lacks a property", "path", where, "reason", "missing property")
	}
	count, stride := *ext.Count, *ext.ByteStride
	if count < 1 || stride < 1 || stride > 256 || count > maxDecoded/stride {
		return view{}, errors.New(TagGLTFInvalid, "the compression of a buffer view has a count or a byteStride that is wrong", "path", where, "reason", "bad count", "count", count, "byteStride", stride)
	}
	buf, err := l.buffer(*ext.Buffer, where+"/buffer")
	if err != nil {
		return view{}, err
	}
	if ext.ByteOffset < 0 || *ext.ByteLength < 1 || ext.ByteOffset > len(buf) || *ext.ByteLength > len(buf)-ext.ByteOffset {
		return view{}, errors.New(TagGLTFInvalid, "the compressed data of a buffer view reaches past its buffer", "path", where, "reason", "view out of bounds", "byteOffset", ext.ByteOffset, "byteLength", *ext.ByteLength, "bufferLength", len(buf))
	}
	data := buf[ext.ByteOffset : ext.ByteOffset+*ext.ByteLength]

	var filter meshopt.Filter
	switch ext.Filter {
	case "", "NONE":
		filter = meshopt.FilterNone
	case "OCTAHEDRAL":
		filter = meshopt.FilterOctahedral
	case "QUATERNION":
		filter = meshopt.FilterQuaternion
	case "EXPONENTIAL":
		filter = meshopt.FilterExponential
	case "COLOR":
		filter = meshopt.FilterColor
	default:
		return view{}, errors.New(TagGLTFInvalid, "the compression of a buffer view has a filter that is not known", "path", where, "reason", "bad filter", "filter", ext.Filter)
	}

	out := make([]byte, count*stride)
	var decodeErr error
	switch ext.Mode {
	case "ATTRIBUTES":
		if stride%4 != 0 {
			return view{}, errors.New(TagGLTFInvalid, "compressed attributes have a byteStride that is not a multiple of four", "path", where, "reason", "bad byteStride", "byteStride", stride)
		}
		if decodeErr = meshopt.DecodeVertexBuffer(out, count, stride, data); decodeErr == nil {
			decodeErr = meshopt.DecodeFilter(filter, out, count, stride)
		}
	case "TRIANGLES", "INDICES":
		if filter != meshopt.FilterNone {
			return view{}, errors.New(TagGLTFInvalid, "compressed indices have a filter", "path", where, "reason", "bad filter", "filter", ext.Filter)
		}
		if stride != 2 && stride != 4 {
			return view{}, errors.New(TagGLTFInvalid, "compressed indices have a byteStride that is not 2 or 4", "path", where, "reason", "bad byteStride", "byteStride", stride)
		}
		indices := make([]uint32, count)
		if ext.Mode == "TRIANGLES" {
			decodeErr = meshopt.DecodeIndexBuffer(indices, data)
		} else {
			decodeErr = meshopt.DecodeIndexSequence(indices, data)
		}
		if decodeErr != nil {
			break
		}
		for k, v := range indices {
			if stride == 2 {
				if v > 0xffff {
					return view{}, errors.New(TagGLTFInvalid, "a compressed index does not fit the byteStride of 2", "path", where, "reason", "index out of range", "index", v)
				}
				out[2*k], out[2*k+1] = byte(v), byte(v>>8)
			} else {
				out[4*k], out[4*k+1], out[4*k+2], out[4*k+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
			}
		}
	default:
		return view{}, errors.New(TagGLTFInvalid, "the compression of a buffer view has a mode that is not known", "path", where, "reason", "bad mode", "mode", ext.Mode)
	}
	if decodeErr != nil {
		// the loader's own tag, with the reason of the decoder
		cause := decodeErr.Error()
		if appErr, ok := errors.FromError(decodeErr); ok {
			if r, ok := appErr.Metadata["reason"].(string); ok {
				cause = r
			}
		}
		return view{}, errors.New(TagGLTFInvalid, "the compressed data of a buffer view is malformed", "path", where, "reason", "meshopt", "cause", cause)
	}
	return view{data: out, stride: stride}, nil
}

// maxDecoded is the most bytes that a compressed buffer view may decode to.
const maxDecoded = 1 << 28
