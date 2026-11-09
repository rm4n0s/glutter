package main

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/mactool"
)

// buildFlags are the flags that make a bundle: those of `build`, and of `run`, `shot -bundle` and `perf`, which build one first.
type buildFlags struct {
	out, name, id, version, icon string
	universal, hardened, release bool
	usage                        envFlags
}

// add registers the flags. keepOut says that the command keeps the bundle, so -o means something.
func (b *buildFlags) add(fs *flag.FlagSet, keepOut bool) {
	if keepOut {
		fs.StringVar(&b.out, "o", "", "the bundle to write, ending in .app (default <name>.app in the current directory)")
	}
	fs.StringVar(&b.name, "name", "", "the name of the app (default: the last element of the package path)")
	fs.StringVar(&b.id, "id", "", "the bundle identifier (default dev.glutter.<name>)")
	fs.StringVar(&b.version, "version", "", "the version of the app (default "+mactool.DefaultVersion+")")
	fs.StringVar(&b.icon, "icon", "", "a square PNG for the icon (default: Glutter's icon)")
	fs.BoolVar(&b.universal, "universal", false, "make one executable with an arm64 and an amd64 slice (default: this machine's architecture)")
	fs.BoolVar(&b.hardened, "hardened", false, "sign with the hardened runtime and the entitlements of the microphone, as a notarized app needs")
	fs.BoolVar(&b.release, "release", false, "build with -tags glutter_release and -ldflags=-s -w")
	fs.Var(&b.usage, "usage", "a usage string for the plist, KEY=text (repeatable), such as NSCameraUsageDescription=...")
}

// options makes the options of mactool.Build.
func (b *buildFlags) options(pkg string) (mactool.BuildOptions, error) {
	o := mactool.BuildOptions{Package: pkg, Out: b.out, Name: b.name, ID: b.id, Version: b.version, Icon: b.icon, Release: b.release,
		Hardened: b.hardened, Usage: map[string]string{}}
	if b.universal {
		o.Archs = []string{"arm64", "amd64"}
	}
	for _, kv := range b.usage {
		k, v, _ := strings.Cut(kv, "=")
		if k == "" {
			return o, errors.New(tagMacOSUsage, "-usage takes KEY=text", "value", kv)
		}
		o.Usage[k] = v
	}
	return o, nil
}

// parseWith parses the flags of a command and returns its one package.
func parseWith(fs *flag.FlagSet, args []string, stderr io.Writer) (string, error) {
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return "", errors.NewErr(tagMacOSUsage, err)
	}
	if fs.NArg() != 1 {
		return "", errors.New(tagMacOSUsage, "the command needs one package\n"+usage)
	}
	return fs.Arg(0), nil
}

// buildBundle builds the bundle of a package for a command that runs it: in dir, so that it is removed with it. It returns the
// result of the build.
func buildBundle(ctx context.Context, pkg string, bf *buildFlags, dir string) (mactool.BuildResult, error) {
	o, err := bf.options(pkg)
	if err != nil {
		return mactool.BuildResult{}, err
	}
	name := o.Name
	if name == "" {
		name = mactool.DefaultName(pkg, "")
	}
	o.Out = filepath.Join(dir, name+".app")
	return mactool.Build(ctx, o)
}

// absOf makes a path absolute for the process that is started somewhere else.
func absOf(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", errors.NewErr(tagMacOSUsage, err).SetMetadata("path", p)
	}
	return abs, nil
}

// tempDir makes a temporary directory and the function that removes it.
func tempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", func() {}, errors.NewErr(tagMacOSUsage, err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil // best effort: a temporary directory
}
