// Package shaders implements the `glutter-gen shaders` subcommand: it compiles the
// WGSL shaders of the explicit renderer to SPIR-V with naga, and writes the Go file
// that holds them (D20).
//
// Flutter: no counterpart.
package shaders

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

const (
	defaultDir  = "internal/gpu/explicit"
	shadersDir  = "shaders"
	generated   = "shaders_gen.go"
	defaultNaga = "naga"
)

// Compiler is naga, or a stand-in for it in tests.
type Compiler interface {
	// Version is naga's version, such as "30.0.1".
	Version() (string, error)
	// Compile validates the WGSL file and writes its SPIR-V to the second file.
	Compile(wgsl, spirv string) error
}

// Naga runs the naga command.
type Naga struct {
	// Path is the command; "naga" when empty.
	Path string
}

func (n Naga) path() string {
	if n.Path == "" {
		return defaultNaga
	}
	return n.Path
}

// Version runs `naga --version`. It fails with ShadersNagaMissing when naga is not
// installed.
func (n Naga) Version() (string, error) {
	out, err := exec.Command(n.path(), "--version").Output()
	if err != nil {
		return "", errors.NewErr("ShadersNagaMissing", err).SetMetadata("command", n.path(),
			"install", "cargo install naga-cli --locked")
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", errors.New("ShadersNagaVersionUnknown", "naga --version printed nothing", "command", n.path())
	}
	return fields[len(fields)-1], nil
}

// Compile runs `naga in.wgsl out.spv`, which validates the shader and writes the
// SPIR-V. It fails with ShadersNagaFailed, carrying naga's message, when the
// shader is not valid.
func (n Naga) Compile(wgsl, spirv string) error {
	out, err := exec.Command(n.path(), wgsl, spirv).CombinedOutput()
	if err != nil {
		return errors.NewErr("ShadersNagaFailed", err).SetMetadata("file", wgsl, "output", string(out))
	}
	return nil
}

// Run executes `glutter-gen shaders [-dir dir] [-naga command]` with the arguments
// that follow the subcommand name. In dir (internal/gpu/explicit) it reads
// shaders/*.wgsl, writes shaders/*.spv and shaders_gen.go, and removes the .spv of a
// shader that is gone.
//
// Flutter: no counterpart
func Run(args []string) error {
	flags := flag.NewFlagSet("shaders", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("dir", defaultDir, "the package directory with shaders/*.wgsl")
	naga := flags.String("naga", defaultNaga, "the naga command")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr("ShadersBadFlags", err).SetMetadata("args", args)
	}
	if err := Generate(*dir, *dir, Naga{Path: *naga}); err != nil {
		return err
	}
	return removeStale(*dir)
}

// Generate reads srcDir/shaders/*.wgsl and writes outDir/shaders/*.spv and
// outDir/shaders_gen.go. The files are the same for the same shaders and the same
// naga, so a test can generate into a temporary directory and compare.
func Generate(srcDir, outDir string, naga Compiler) error {
	files, err := filepath.Glob(filepath.Join(srcDir, shadersDir, "*.wgsl"))
	if err != nil {
		return errors.NewErr("ShadersReadFailed", err).SetMetadata("dir", srcDir)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return errors.New("ShadersNoShaders", "there is no shaders/*.wgsl in the directory", "dir", srcDir)
	}
	version, err := naga.Version()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outDir, shadersDir), 0o755); err != nil {
		return errors.NewErr("ShadersWriteFailed", err).SetMetadata("dir", outDir)
	}
	// naga compiles the text with the includes expanded, which is also the text of the
	// generated file; it is written to a temporary directory under the file's name.
	tmp, err := os.MkdirTemp("", "glutter-shaders-")
	if err != nil {
		return errors.NewErr("ShadersWriteFailed", err).SetMetadata("dir", os.TempDir())
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	var names []string
	mods := map[string]Module{}
	srcs := map[string]string{}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".wgsl")
		src, err := expand(file, nil)
		if err != nil {
			return err
		}
		mod, err := Parse(src)
		if err != nil {
			return wrapParse(err, file)
		}
		expanded := filepath.Join(tmp, filepath.Base(file))
		if err := os.WriteFile(expanded, []byte(src), 0o644); err != nil {
			return errors.NewErr("ShadersWriteFailed", err).SetMetadata("file", expanded)
		}
		if err := naga.Compile(expanded, filepath.Join(outDir, shadersDir, name+".spv")); err != nil {
			return err
		}
		names = append(names, name)
		mods[name], srcs[name] = mod, src
	}
	code, err := render(version, names, mods, srcs)
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, generated)
	if err := os.WriteFile(path, code, 0o644); err != nil {
		return errors.NewErr("ShadersWriteFailed", err).SetMetadata("file", path)
	}
	return nil
}

// includeLine is a line that includes another file, relative to the including file's directory.
var includeLine = regexp.MustCompile(`(?m)^[ \t]*//#include[ \t]+"([^"]+)"[ \t]*$`)

// expand reads a WGSL file and replaces each `//#include "file"` line with the text of the file,
// which may include others (what several shaders share: the paint, the blend modes). stack is
// the files being expanded, to find a cycle. It fails with ShadersIncludeMissing for a file that
// cannot be read and ShadersIncludeCycle for one that includes itself.
func expand(file string, stack []string) (string, error) {
	for _, f := range stack {
		if f == file {
			return "", errors.New("ShadersIncludeCycle", "a shader includes itself", "file", file, "stack", stack)
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if len(stack) > 0 {
			return "", errors.NewErr("ShadersIncludeMissing", err).SetMetadata("file", file, "includedBy", stack[len(stack)-1])
		}
		return "", errors.NewErr("ShadersReadFailed", err).SetMetadata("file", file)
	}
	var failure error
	text := includeLine.ReplaceAllStringFunc(string(data), func(line string) string {
		if failure != nil {
			return line
		}
		name := includeLine.FindStringSubmatch(line)[1]
		inc, err := expand(filepath.Join(filepath.Dir(file), name), append(append([]string(nil), stack...), file))
		if err != nil {
			failure = err
			return line
		}
		return strings.TrimRight(inc, "\n")
	})
	return text, failure
}

// wrapParse adds the file to the metadata of an error of Parse, keeping its tag.
func wrapParse(err error, file string) error {
	if appErr, ok := errors.FromError(err); ok {
		appErr.Metadata["file"] = file
	}
	return err
}

// removeStale deletes the .spv files of shaders that have no .wgsl.
func removeStale(dir string) error {
	spvs, err := filepath.Glob(filepath.Join(dir, shadersDir, "*.spv"))
	if err != nil {
		return errors.NewErr("ShadersReadFailed", err).SetMetadata("dir", dir)
	}
	for _, spv := range spvs {
		wgsl := strings.TrimSuffix(spv, ".spv") + ".wgsl"
		if _, err := os.Stat(wgsl); err == nil {
			continue
		}
		if err := os.Remove(spv); err != nil {
			return errors.NewErr("ShadersWriteFailed", err).SetMetadata("file", spv)
		}
	}
	return nil
}

var nameParts = regexp.MustCompile(`[a-zA-Z0-9]+`)

// goName turns a file name such as "blur_pass" into "BlurPass".
func goName(name string) string {
	var b strings.Builder
	for _, p := range nameParts.FindAllString(name, -1) {
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// render makes the text of shaders_gen.go.
func render(version string, names []string, mods map[string]Module, srcs map[string]string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by glutter-gen shaders; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Compiled from shaders/*.wgsl with naga %s (D20).\n\n", version)
	b.WriteString("package explicit\n\nimport _ \"embed\"\n")
	for _, name := range names {
		if strings.Contains(srcs[name], "`") {
			return nil, errors.New("ShadersBacktick", "a WGSL file cannot hold a backquote: it is written as a raw string", "shader", name)
		}
		id := goName(name)
		fmt.Fprintf(&b, "\n//go:embed shaders/%s.spv\nvar spirv%s []byte\n", name, id)
		fmt.Fprintf(&b, "\n// Shader%s is shaders/%s.wgsl.\n//\n// Flutter: no counterpart\n", id, name)
		fmt.Fprintf(&b, "var Shader%s = &Shader{\n\tName: %q,\n\tWGSL: `%s`,\n\tSPIRV: spirv%s,\n\tEntries: []EntryPoint{\n", id, name, srcs[name], id)
		for _, e := range mods[name].Entries {
			fmt.Fprintf(&b, "\t\t{Name: %q, Stage: ShaderStage%s, Bindings: []ShaderBinding{\n", e.Name, e.Stage)
			for _, bd := range e.Bindings {
				fmt.Fprintf(&b, "\t\t\t{Name: %q, Group: %d, Binding: %d, Kind: %s},\n", bd.Name, bd.Group, bd.Binding, bd.Kind)
			}
			b.WriteString("\t\t}},\n")
		}
		b.WriteString("\t},\n}\n")
	}
	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.NewErr("ShadersFormatFailed", err)
	}
	return out, nil
}
