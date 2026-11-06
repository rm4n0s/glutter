//go:build windows

// Command hlslcheck compiles the HLSL of generated shaders with D3DCompile (SPEC §27.3). glutter-gen
// gpushaders builds it for Windows and runs it under Wine (scripts/wine-exec.sh), so that an HLSL file
// that Direct3D 12 would refuse shows when the shaders are generated, not when a pipeline is made.
//
// Usage: hlslcheck <input.json>, where the file is a list of {"name", "hlsl", "entries": [{"name",
// "target"}]}. It prints "HLSLCHECK FAIL <shader> <entry> <target>: <messages>" for each entry that does
// not compile and "HLSLCHECK DONE <n>" at the end, the number of entries compiled; its exit status is 1
// when an entry failed and 2 when it could not run.
//
// Flutter: no counterpart
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/d3d12"
)

type input struct {
	Name    string `json:"name"`
	HLSL    string `json:"hlsl"`
	Entries []struct {
		Name   string `json:"name"`
		Target string `json:"target"`
	} `json:"entries"`
}

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if len(os.Args) != 2 {
		logger.Error("usage: hlslcheck <input.json>")
		return 2
	}
	shaders, err := read(os.Args[1])
	if err != nil {
		logger.Error("hlslcheck: the input cannot be read", "err", err)
		return 2
	}
	dll, err := d3d12.LoadCompilerDLL()
	if err != nil {
		logger.Error("hlslcheck: d3dcompiler_47.dll cannot be loaded", "err", err)
		return 2
	}
	compiler := d3d12.NewCompiler(dll, nil)
	flags := d3d12.CompileFlags(os.Getenv)
	failed, done := 0, 0
	for _, s := range shaders {
		for _, e := range s.Entries {
			if _, err := compiler.Compile(s.HLSL, e.Name, e.Target, flags); err != nil {
				failed++
				fmt.Printf("HLSLCHECK FAIL %s %s %s: %s\n", s.Name, e.Name, e.Target, oneLine(err))
				continue
			}
			done++
		}
	}
	fmt.Printf("HLSLCHECK DONE %d\n", done)
	if failed > 0 {
		return 1
	}
	return 0
}

// read parses the input file.
func read(path string) ([]input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.NewErr(tagHLSLCheckInput, err).SetMetadata("file", path)
	}
	var shaders []input
	if err := json.Unmarshal(data, &shaders); err != nil {
		return nil, errors.NewErr(tagHLSLCheckInput, err).SetMetadata("file", path)
	}
	return shaders, nil
}

// oneLine writes the messages of a failed compile on one line.
func oneLine(err error) string {
	text := err.Error()
	if appErr, ok := errors.FromError(err); ok {
		if m, ok := appErr.Metadata["messages"].(string); ok {
			text = m
		}
	}
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "\n", " | ")), " ")
}

// The tag of the error that this file makes (rm4n0s/errors, R5).
const tagHLSLCheckInput = "HLSLCheckInput"
