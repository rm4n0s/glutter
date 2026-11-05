//go:build linux

package vulkan

import (
	"encoding/binary"
	"math"

	"github.com/rm4n0s/errors"
)

// Vulkan wants a vertex shader that is drawn as points to write PointSize; WGSL has no such output,
// and naga's command line cannot be told to add one. A pipeline with the point list topology
// therefore gets a copy of the SPIR-V in which every vertex entry point stores 1.0 in a PointSize
// output (WebGPU's points are one pixel wide too). The rewrite adds a variable, its decoration and
// the store before each return of the entry functions; it needs nothing else of the module.

// The SPIR-V words that the rewrite reads or writes (SPIR-V 1.x, unified).
const (
	spvMagic             = 0x07230203
	spvOpEntryPoint      = 15
	spvOpTypeVoid        = 19
	spvOpTypeFloat       = 22
	spvOpTypePointer     = 32
	spvOpTypeForwardPtr  = 39
	spvOpConstant        = 43
	spvOpFunction        = 54
	spvOpFunctionEnd     = 56
	spvOpVariable        = 59
	spvOpStore           = 62
	spvOpDecorate        = 71
	spvOpReturn          = 253
	spvModelVertex       = 0
	spvStorageOutput     = 3
	spvDecorationBuiltIn = 11
	spvBuiltInPointSize  = 1
	spvHeaderWords       = 5
)

// withPointSize returns the SPIR-V with a PointSize output written by every vertex entry point. It
// fails with VulkanSPIRVInvalid when the words are not a module it understands.
func withPointSize(spirv []byte) ([]byte, error) {
	if len(spirv)%4 != 0 || len(spirv) < spvHeaderWords*4 {
		return nil, errors.New(tagVulkanSPIRVInvalid, "the SPIR-V is not a whole number of words", "bytes", len(spirv))
	}
	words := make([]uint32, len(spirv)/4)
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(spirv[i*4:])
	}
	if words[0] != spvMagic {
		return nil, errors.New(tagVulkanSPIRVInvalid, "the SPIR-V has the wrong magic number", "magic", words[0])
	}
	bound := words[3]

	type entry struct{ at, function int }
	var entries []entry
	var floatType, pointerType, one uint32
	firstFunction, firstType := -1, -1
	for at := spvHeaderWords; at < len(words); {
		op, n := words[at]&0xffff, int(words[at]>>16)
		if n == 0 || at+n > len(words) {
			return nil, errors.New(tagVulkanSPIRVInvalid, "an instruction of the SPIR-V runs past its end", "word", at)
		}
		switch {
		case op == spvOpEntryPoint && words[at+1] == spvModelVertex:
			entries = append(entries, entry{at: at, function: int(words[at+2])})
		case op == spvOpTypeFloat && words[at+2] == 32:
			floatType = words[at+1]
		case op == spvOpTypePointer && words[at+2] == spvStorageOutput && words[at+3] == floatType && floatType != 0:
			pointerType = words[at+1]
		case op == spvOpConstant && words[at+1] == floatType && floatType != 0 && words[at+3] == math.Float32bits(1):
			one = words[at+2]
		case op == spvOpFunction && firstFunction < 0:
			firstFunction = at
		}
		if op >= spvOpTypeVoid && op <= spvOpTypeForwardPtr && firstType < 0 {
			firstType = at
		}
		at += n
	}
	if len(entries) == 0 {
		return words2bytes(words), nil
	}
	if firstFunction < 0 || firstType < 0 {
		return nil, errors.New(tagVulkanSPIRVInvalid, "the SPIR-V has no function or no type")
	}

	next := func() uint32 { bound++; return bound - 1 }
	var globals []uint32
	if floatType == 0 {
		floatType = next()
		globals = append(globals, 3<<16|spvOpTypeFloat, floatType, 32)
	}
	if pointerType == 0 {
		pointerType = next()
		globals = append(globals, 4<<16|spvOpTypePointer, pointerType, spvStorageOutput, floatType)
	}
	if one == 0 {
		one = next()
		globals = append(globals, 4<<16|spvOpConstant, floatType, one, math.Float32bits(1))
	}
	variable := next()
	globals = append(globals, 4<<16|spvOpVariable, pointerType, variable, spvStorageOutput)
	decorate := []uint32{4<<16 | spvOpDecorate, variable, spvDecorationBuiltIn, spvBuiltInPointSize}
	store := []uint32{3<<16 | spvOpStore, variable, one}

	isEntryFunction := map[uint32]bool{}
	for _, e := range entries {
		isEntryFunction[uint32(e.function)] = true
	}
	entryAt := map[int]bool{}
	for _, e := range entries {
		entryAt[e.at] = true
	}

	out := make([]uint32, 0, len(words)+len(globals)+len(decorate)+len(entries)+4*len(entries))
	out = append(out, words[:spvHeaderWords]...)
	inEntryFunction := false
	for at := spvHeaderWords; at < len(words); {
		op, n := words[at]&0xffff, int(words[at]>>16)
		if at == firstType {
			out = append(out, decorate...)
		}
		if at == firstFunction {
			out = append(out, globals...)
		}
		switch {
		case entryAt[at]:
			out = append(out, uint32(n+1)<<16|op)
			out = append(out, words[at+1:at+n]...)
			out = append(out, variable)
		case op == spvOpFunction:
			inEntryFunction = isEntryFunction[words[at+2]]
			out = append(out, words[at:at+n]...)
		case op == spvOpFunctionEnd:
			inEntryFunction = false
			out = append(out, words[at:at+n]...)
		case op == spvOpReturn && inEntryFunction:
			out = append(out, store...)
			out = append(out, words[at:at+n]...)
		default:
			out = append(out, words[at:at+n]...)
		}
		at += n
	}
	out[3] = bound
	return words2bytes(out), nil
}

func words2bytes(words []uint32) []byte {
	out := make([]byte, len(words)*4)
	for i, w := range words {
		binary.LittleEndian.PutUint32(out[i*4:], w)
	}
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagVulkanSPIRVInvalid = "VulkanSPIRVInvalid"
)
