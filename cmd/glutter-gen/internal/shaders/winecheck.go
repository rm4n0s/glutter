package shaders

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rm4n0s/errors"
)

// The check of the HLSL with D3DCompile under Wine (SPEC §27.3). It is for the Glutter repository: it
// needs scripts/wine-exec.sh and the source of internal/gpu/d3d12/hlslcheck, which an app does not have.

const (
	modulePath  = "github.com/rm4n0s/glutter"
	wineScript  = "scripts/wine-exec.sh"
	checkerPkg  = "./internal/gpu/d3d12/hlslcheck"
	wineMissing = 69 // the exit status of wine-exec.sh when Wine or Xvfb is not installed
)

// newWineHLSLCheck returns the function that compiles the HLSL of the shaders under Wine. With force false
// (-wine=auto) it returns nil, so that nothing is checked, when the directory is not in the Glutter module,
// or Wine is not installed or does not run the program. With force true each of these is
// GPUShadersWineUnavailable.
func newWineHLSLCheck(dir string, force bool) (func([]HLSLShader) error, error) {
	root := moduleRoot(dir)
	if root == "" || !fileExists(filepath.Join(root, wineScript)) {
		if force {
			return nil, errors.New(tagGPUShadersWineUnavailable, "-wine=on needs scripts/wine-exec.sh of the Glutter module", "dir", dir)
		}
		return nil, nil
	}
	return func(shaders []HLSLShader) error {
		return runWineCheck(root, shaders, force)
	}, nil
}

// moduleRoot is the directory with the go.mod of the Glutter module that holds dir, or "".
func moduleRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for d := abs; ; d = filepath.Dir(d) {
		if data, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			if strings.Contains(string(data), "module "+modulePath+"\n") {
				return d
			}
			return ""
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// runWineCheck builds hlslcheck for Windows, runs it under Wine and reads what it prints. It fails with
// GPUShadersHLSLInvalid, naming the shader, entry point and target of each entry that does not compile,
// with D3DCompile's messages. A run that did not get to the compiler (Wine is not installed or does not
// start) checks nothing, or fails with GPUShadersWineUnavailable when force is set.
func runWineCheck(root string, shaders []HLSLShader, force bool) error {
	tmp, err := os.MkdirTemp("", "glutter-hlslcheck-")
	if err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("dir", os.TempDir())
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	type entry struct {
		Name   string `json:"name"`
		Target string `json:"target"`
	}
	type unit struct {
		Name    string  `json:"name"`
		HLSL    string  `json:"hlsl"`
		Entries []entry `json:"entries"`
	}
	var units []unit
	for _, s := range shaders {
		u := unit{Name: s.Name, HLSL: s.HLSL}
		for _, e := range s.Entries {
			u.Entries = append(u.Entries, entry(e))
		}
		units = append(units, u)
	}
	data, err := json.Marshal(units)
	if err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("file", "hlslcheck input")
	}
	input := filepath.Join(tmp, "in.json")
	if err := os.WriteFile(input, data, 0o644); err != nil {
		return errors.NewErr(tagShadersWriteFailed, err).SetMetadata("file", input)
	}
	exe := filepath.Join(tmp, "hlslcheck.exe")
	build := exec.Command("go", "build", "-o", exe, checkerPkg)
	build.Dir = root
	build.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		if force {
			return errors.New(tagGPUShadersWineUnavailable, "the Windows build of hlslcheck failed", "output", string(out))
		}
		return nil
	}
	run := exec.Command(filepath.Join(root, wineScript), exe, input)
	run.Dir = root
	out, err := run.CombinedOutput()
	text := string(out)
	var fails []string
	done := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "HLSLCHECK FAIL "):
			fails = append(fails, strings.TrimPrefix(line, "HLSLCHECK FAIL "))
		case strings.HasPrefix(line, "HLSLCHECK DONE "):
			done = true
		}
	}
	if len(fails) > 0 {
		return errors.New(tagGPUShadersHLSLInvalid, "D3DCompile refuses the HLSL of a shader", "failures", fails)
	}
	if !done {
		why := "the HLSL was not compiled: Wine did not run hlslcheck"
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == wineMissing {
			why = "Wine is not installed"
		}
		if force {
			return errors.New(tagGPUShadersWineUnavailable, why, "output", text)
		}
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUShadersHLSLInvalid is the tag of the error of HLSL that D3DCompile refuses under Wine; the
	// metadata "failures" lists "<shader> <entry> <target>: <messages>" for each entry.
	tagGPUShadersHLSLInvalid = "GPUShadersHLSLInvalid"
	// tagGPUShadersWineUnavailable is the tag of the error of -wine=on when the check cannot run; the
	// metadata "output" has what the run printed.
	tagGPUShadersWineUnavailable = "GPUShadersWineUnavailable"
)
