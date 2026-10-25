package d3d12

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// BlobCache is where the bytecode is kept between runs: *shadercache.Cache is one (SPEC §9.7). A nil
// BlobCache is no cache.
//
// Flutter: no counterpart
type BlobCache interface {
	// Load returns what is stored under the key, and false when there is nothing.
	Load(key string) ([]byte, bool)
	// Store keeps the data under the key, replacing what was there.
	Store(key string, data []byte)
}

// CacheKey is the key of the bytecode that D3DCompile makes from the HLSL: the hash of the text, the
// entry point, the target ("vs_5_1" or "ps_5_1"), the compile flags and the file version of
// d3dcompiler_47.dll, so that a new compiler never reads the bytecode of an old one (SPEC §9.7). Each
// part is length-prefixed, so no two different parts make the same key.
//
// Flutter: no counterpart
func CacheKey(hlsl, entry, target string, flags uint32, compilerVersion string) string {
	h := sha256.New()
	part := func(s string) {
		var n [8]byte
		binary.LittleEndian.PutUint64(n[:], uint64(len(s)))
		h.Write(n[:])
		h.Write([]byte(s))
	}
	part("glutter d3d12 dxbc 1")
	part(hlsl)
	part(entry)
	part(target)
	var f [4]byte
	binary.LittleEndian.PutUint32(f[:], flags)
	part(string(f[:]))
	part(compilerVersion)
	return "d3d12-dxbc-" + hex.EncodeToString(h.Sum(nil))
}

// dxbcHeaderSize is the size of the header of a DXBC container: the four-byte magic, a 16-byte
// checksum, the version (1), the total size and the number of chunks.
const dxbcHeaderSize = 32

// ValidDXBC says whether the data is a DXBC container as far as its header tells: the magic "DXBC",
// room for the header, and a total size that is the length of the data. A file of the cache that
// fails this (cut short, or something else) is not used: the shader is compiled again and the new
// bytecode replaces the file (SPEC §9.7).
//
// Flutter: no counterpart
func ValidDXBC(data []byte) bool {
	if len(data) < dxbcHeaderSize || string(data[:4]) != "DXBC" {
		return false
	}
	return binary.LittleEndian.Uint32(data[24:]) == uint32(len(data))
}
