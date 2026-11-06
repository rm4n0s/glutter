package shaders

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// The gpushaders subcommand (D31, SPEC §27.3): an app's WGSL files become a generated Go file of
// *gpu.Shader values, with the WGSL, the SPIR-V, the HLSL and the reflection.

const (
	gpuImport = "github.com/rm4n0s/glutter/gpu"
	spirvVal  = "spirv-val"
	// spirvTarget is the environment spirv-val validates for: the Vulkan 1.1 of the oldest device that
	// the Vulkan backend accepts (SPEC §9.3).
	spirvTarget = "vulkan1.1"
	// nagaInstall is the command that installs the naga of D20.
	nagaInstall = "cargo install naga-cli --version 30.0.1 --locked"
)

// AppOptions are what GenerateApp does besides compiling.
type AppOptions struct {
	// Package is the name of the Go package of shaders_gen.go; the name of the package of the directory when
	// empty (PackageName).
	Package string
	// ValidateSPIRV checks the module written to a file; nil checks none. RunGPU gives it spirv-val.
	ValidateSPIRV func(spv string) error
	// CheckHLSL compiles the HLSL of each shader; nil checks none. RunGPU gives it D3DCompile under Wine.
	CheckHLSL func(shaders []HLSLShader) error
}

// HLSLShader is the HLSL of a shader with its entry points, for CheckHLSL.
type HLSLShader struct {
	Name    string
	HLSL    string
	Entries []HLSLEntry
}

// HLSLEntry is an entry point of the HLSL, with the shader model it is compiled for.
type HLSLEntry struct {
	Name, Target string
}

// RunGPU executes `glutter-gen gpushaders [-check] [-naga command] [-wine auto|on|off] dir` with the
// arguments that follow the subcommand name. It reads dir/*.wgsl and writes dir/<name>.spv and
// dir/shaders_gen.go, and removes the .spv of a shader that is gone. With -check it writes nothing and
// fails with GPUShadersOutOfDate when the files in dir differ from a new generation.
//
// Flutter: no counterpart
func RunGPU(args []string) error {
	flags := flag.NewFlagSet("gpushaders", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "write nothing; fail when the files in the directory differ from a new generation")
	naga := flags.String("naga", defaultNaga, "the naga command")
	wine := flags.String("wine", "auto", "compile the HLSL with D3DCompile under Wine: auto (when scripts/wine-exec.sh is in the module), on or off")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagShadersBadFlags, err).SetMetadata("args", args)
	}
	if flags.NArg() != 1 {
		return errors.New(tagShadersBadFlags, "gpushaders takes one directory", "args", args)
	}
	dir := flags.Arg(0)
	opts := AppOptions{ValidateSPIRV: validateWithSpirvVal}
	switch *wine {
	case "auto", "on":
		check, err := newWineHLSLCheck(dir, *wine == "on")
		if err != nil {
			return err
		}
		opts.CheckHLSL = check
	case "off":
	default:
		return errors.New(tagShadersBadFlags, "-wine is auto, on or off", "wine", *wine)
	}
	c := Naga{Path: *naga, App: true}
	if *check {
		return CheckApp(dir, c, opts)
	}
	if err := GenerateApp(dir, dir, c, opts); err != nil {
		return err
	}
	return removeStaleApp(dir)
}

// validateWithSpirvVal runs spirv-val on the module when it is installed. A module it refuses is
// GPUShadersSPIRVInvalid, with its messages.
func validateWithSpirvVal(spv string) error {
	path, err := exec.LookPath(spirvVal)
	if err != nil {
		return nil // not installed: nothing to check with
	}
	out, err := exec.Command(path, "--target-env", spirvTarget, spv).CombinedOutput()
	if err != nil {
		return errors.NewErr(tagGPUShadersSPIRVInvalid, err).SetMetadata("file", spv, "output", string(out))
	}
	return nil
}

// PackageName is the name of the Go package of a directory: the package clause of a Go file in it other than
// shaders_gen.go, else the name of the directory made into an identifier ("shaders" when it has none).
func PackageName(dir string) string {
	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	sort.Strings(files)
	fset := token.NewFileSet()
	for _, f := range files {
		if filepath.Base(f) == generated || strings.HasSuffix(f, "_test.go") {
			continue
		}
		if parsed, err := parser.ParseFile(fset, f, nil, parser.PackageClauseOnly); err == nil {
			return parsed.Name.Name
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	name := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(filepath.Base(abs), ""))
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		return "shaders"
	}
	return name
}

// GenerateApp reads srcDir/*.wgsl and writes outDir/<name>.spv and outDir/shaders_gen.go. The files are the
// same for the same shaders and the same naga, so CheckApp can generate into a temporary directory and
// compare. It fails with ShadersNoShaders for a directory without a .wgsl file.
//
// Flutter: no counterpart
func GenerateApp(srcDir, outDir string, naga Compiler, opts AppOptions) error {
	files, err := filepath.Glob(filepath.Join(srcDir, "*.wgsl"))
	if err != nil {
		return errors.NewErr(tagShadersReadFailed, err).SetMetadata("dir", srcDir)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return errors.New(tagShadersNoShaders, "there is no *.wgsl in the directory", "dir", srcDir)
	}
	version, err := naga.Version()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("dir", outDir)
	}
	tmp, err := os.MkdirTemp("", "glutter-gpushaders-")
	if err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("dir", os.TempDir())
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	var names []string
	mods := map[string]Module{}
	srcs := map[string]string{}
	hlsls := map[string]string{}
	goNames := map[string]string{}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".wgsl")
		id := goName(name)
		if id == "" || (id[0] >= '0' && id[0] <= '9') {
			id = "Shader" + id
		}
		if other, ok := goNames[id]; ok {
			return errors.New(tagShadersNameTwice, "two shaders have the same name in Go", "go_name", id, "files", other+".wgsl,"+name+".wgsl")
		}
		goNames[id] = name
		spv := filepath.Join(outDir, name+".spv")
		c, err := compile(file, tmp, spv, naga, true, true)
		if err != nil {
			return err
		}
		if opts.ValidateSPIRV != nil {
			if err := opts.ValidateSPIRV(spv); err != nil {
				return err
			}
		}
		names = append(names, name)
		mods[name], srcs[name], hlsls[name] = c.mod, c.src, c.hlsl
	}
	if opts.CheckHLSL != nil {
		var units []HLSLShader
		for _, name := range names {
			u := HLSLShader{Name: name, HLSL: hlsls[name]}
			for _, e := range mods[name].Entries {
				u.Entries = append(u.Entries, HLSLEntry{Name: e.Name, Target: hlslTarget(e.Stage)})
			}
			units = append(units, u)
		}
		if err := opts.CheckHLSL(units); err != nil {
			return err
		}
	}
	pkg := opts.Package
	if pkg == "" {
		pkg = PackageName(outDir)
	}
	code, err := renderApp(pkg, version, names, goNames, mods, srcs, hlsls)
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, generated)
	if err := os.WriteFile(path, code, 0o644); err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("file", path)
	}
	return nil
}

// hlslTarget is the D3DCompile target of a stage (shader model 5.1, D25).
func hlslTarget(stage string) string {
	switch stage {
	case "Vertex":
		return "vs_5_1"
	case "Fragment":
		return "ps_5_1"
	}
	return "cs_5_1"
}

// CheckApp generates into a temporary directory and compares with dir: the files of a new generation
// must be in dir with the same bytes, and dir must hold no .spv without a .wgsl. It fails with
// GPUShadersOutOfDate, naming the files that differ.
//
// Flutter: no counterpart
func CheckApp(dir string, naga Compiler, opts AppOptions) error {
	fresh, err := os.MkdirTemp("", "glutter-gpushaders-check-")
	if err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("dir", os.TempDir())
	}
	defer func() { _ = os.RemoveAll(fresh) }()
	if opts.Package == "" {
		opts.Package = PackageName(dir)
	}
	if err := GenerateApp(dir, fresh, naga, opts); err != nil {
		return err
	}
	var differ []string
	made, err := filepath.Glob(filepath.Join(fresh, "*"))
	if err != nil {
		return errors.NewErr(tagShadersReadFailed, err).SetMetadata("dir", fresh)
	}
	for _, f := range made {
		want, err := os.ReadFile(f)
		if err != nil {
			return errors.NewErr(tagShadersReadFailed, err).SetMetadata("file", f)
		}
		got, err := os.ReadFile(filepath.Join(dir, filepath.Base(f)))
		if err != nil || !bytes.Equal(got, want) {
			differ = append(differ, filepath.Base(f))
		}
	}
	spvs, err := filepath.Glob(filepath.Join(dir, "*.spv"))
	if err != nil {
		return errors.NewErr(tagShadersReadFailed, err).SetMetadata("dir", dir)
	}
	for _, spv := range spvs {
		if _, err := os.Stat(strings.TrimSuffix(spv, ".spv") + ".wgsl"); err != nil {
			differ = append(differ, filepath.Base(spv)+" (has no .wgsl)")
		}
	}
	if len(differ) > 0 {
		sort.Strings(differ)
		return errors.New(tagGPUShadersOutOfDate, "the generated files differ from a new generation: run go generate", "dir", dir, "files", differ)
	}
	return nil
}

// removeStaleApp deletes the .spv files of shaders that have no .wgsl.
func removeStaleApp(dir string) error {
	spvs, err := filepath.Glob(filepath.Join(dir, "*.spv"))
	if err != nil {
		return errors.NewErr(tagShadersReadFailed, err).SetMetadata("dir", dir)
	}
	for _, spv := range spvs {
		if _, err := os.Stat(strings.TrimSuffix(spv, ".spv") + ".wgsl"); err == nil {
			continue
		}
		if err := os.Remove(spv); err != nil {
			return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("file", spv)
		}
	}
	return nil
}

// inputTypes maps the types of the inputs, as Parse spells them, to the constants of package gpu.
var inputTypes = map[string]string{
	"Float32": "ShaderTypeFloat32", "Float32x2": "ShaderTypeFloat32x2", "Float32x3": "ShaderTypeFloat32x3", "Float32x4": "ShaderTypeFloat32x4",
	"Sint32": "ShaderTypeSint32", "Sint32x2": "ShaderTypeSint32x2", "Sint32x3": "ShaderTypeSint32x3", "Sint32x4": "ShaderTypeSint32x4",
	"Uint32": "ShaderTypeUint32", "Uint32x2": "ShaderTypeUint32x2", "Uint32x3": "ShaderTypeUint32x3", "Uint32x4": "ShaderTypeUint32x4",
}

// storageFormatNames maps explicit's spelling of a storage format to WGSL's.
var storageFormatNames = map[string]string{
	"TextureFormatRGBA8Unorm":  "rgba8unorm",
	"TextureFormatRGBA16Float": "rgba16float",
	"TextureFormatR32Float":    "r32float",
	"TextureFormatR32Uint":     "r32uint",
}

// renderApp makes the text of an app's shaders_gen.go.
func renderApp(pkg, version string, names []string, goNames map[string]string, mods map[string]Module, srcs, hlsls map[string]string) ([]byte, error) {
	idOf := map[string]string{}
	for id, name := range goNames {
		idOf[name] = id
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by glutter-gen gpushaders; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Compiled from *.wgsl with naga %s (D20, D31), HLSL for shader model 5.1 (D25).\n\n", version)
	fmt.Fprintf(&b, "package %s\n\nimport (\n\t_ \"embed\"\n\n\t\"%s\"\n)\n", pkg, gpuImport)
	for _, name := range names {
		if strings.Contains(srcs[name], "`") {
			return nil, errors.New(tagShadersBacktick, "a WGSL file cannot hold a backquote: it is written as a raw string", "shader", name)
		}
		if strings.Contains(hlsls[name], "`") {
			return nil, errors.New(tagShadersBacktick, "the HLSL cannot hold a backquote: it is written as a raw string", "shader", name)
		}
		id := idOf[name]
		fmt.Fprintf(&b, "\n//go:embed %s.spv\nvar spirv%s []byte\n", name, id)
		fmt.Fprintf(&b, "\n// %s is %s.wgsl.\n", id, name)
		fmt.Fprintf(&b, "var %s = &gpu.Shader{\n\tName: %q,\n\tWGSL: `%s`,\n\tSPIRV: spirv%s,\n\tHLSL: `%s`,\n\tEntries: []gpu.EntryPoint{\n", id, name, srcs[name], id, hlsls[name])
		for _, e := range mods[name].Entries {
			extra := ""
			if e.Stage == "Compute" {
				extra += fmt.Sprintf(", WorkgroupSize: [3]int{%d, %d, %d}", e.WorkgroupSize[0], e.WorkgroupSize[1], e.WorkgroupSize[2])
			}
			if e.ReadsInstanceIndex {
				extra += ", ReadsInstanceIndex: true"
			}
			if e.ReadsVertexIndex {
				extra += ", ReadsVertexIndex: true"
			}
			fmt.Fprintf(&b, "\t\t{Name: %q, Stage: gpu.ShaderStage%s%s,\n", e.Name, e.Stage, extra)
			if len(e.Inputs) > 0 {
				b.WriteString("\t\t\tInputs: []gpu.VertexInput{\n")
				for _, in := range e.Inputs {
					fmt.Fprintf(&b, "\t\t\t\t{Name: %q, Location: %d, Type: gpu.%s},\n", in.Name, in.Location, inputTypes[in.Type])
				}
				b.WriteString("\t\t\t},\n")
			}
			b.WriteString("\t\t\tBindings: []gpu.ShaderBinding{\n")
			for _, bd := range e.Bindings {
				class, space, index := bd.Register()
				extra := ""
				if bd.Dim != "" {
					extra += ", Dimension: gpu." + bd.Dim
				}
				if bd.Sample != "" {
					extra += ", Sample: gpu." + bd.Sample
				}
				if bd.StorageFormat != "" {
					extra += fmt.Sprintf(", StorageFormat: %q", storageFormatNames[bd.StorageFormat])
				}
				if bd.Size != 0 {
					extra += fmt.Sprintf(", Size: %d", bd.Size)
				}
				fmt.Fprintf(&b, "\t\t\t\t{Name: %q, Group: %d, Binding: %d, Kind: gpu.%s%s, Register: gpu.ShaderRegister{Class: %q, Space: %d, Index: %d}},\n",
					bd.Name, bd.Group, bd.Binding, bd.Kind, extra, class, space, index)
			}
			b.WriteString("\t\t\t},\n\t\t},\n")
		}
		b.WriteString("\t},\n}\n")
	}
	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.NewErr(tagShadersFormatFailed, err)
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUShadersOutOfDate is the tag of the error of gpushaders -check when the generated files in the
	// directory are not what a new generation writes; the metadata are "dir" and "files".
	tagGPUShadersOutOfDate = "GPUShadersOutOfDate"
	// tagGPUShadersSPIRVInvalid is the tag of the error of a SPIR-V module that spirv-val refuses; the
	// metadata are "file" and spirv-val's "output".
	tagGPUShadersSPIRVInvalid = "GPUShadersSPIRVInvalid"
	// tagShadersNameTwice is the tag of the error of two files whose names make the same Go name
	// ("blur_pass" and "blurPass"); the metadata are "go_name" and "files".
	tagShadersNameTwice = "ShadersNameTwice"
)
