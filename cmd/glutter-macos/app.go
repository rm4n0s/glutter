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
	sign, minMacOS               string
	universal, hardened, debug   bool
	usage                        envFlags
	urlSchemes, docTypes         listFlags
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
	fs.BoolVar(&b.debug, "debug", false, "build with -tags debug_mode, not stripped (a release build, stripped with -ldflags=-s -w, otherwise)")
	fs.Var(&b.usage, "usage", "a usage string for the plist, KEY=text (repeatable), such as NSCameraUsageDescription=...")
	fs.Var(&b.urlSchemes, "url-scheme", "a URL scheme the app handles, such as magnet (repeatable): a click on such a link reaches the app as a ui.OpenRequest")
	fs.Var(&b.docTypes, "doc-type", "a type of file the app opens, NAME=ext[,ext...][:viewer|editor], such as \"BitTorrent file=torrent\" (repeatable): a double click on such a file reaches the app as a ui.OpenRequest")
	fs.StringVar(&b.sign, "sign", "", "the identity to sign with, as codesign takes it (a name in the keychain or a SHA-1); ad hoc (-) by default")
	fs.StringVar(&b.minMacOS, "min-macos", "", "the oldest macOS the app runs on, such as 14.0: the plist's LSMinimumSystemVersion and the build's MACOSX_DEPLOYMENT_TARGET (default "+mactool.MinSystemVersion+")")
}

// options makes the options of mactool.Build.
func (b *buildFlags) options(pkg string) (mactool.BuildOptions, error) {
	o := mactool.BuildOptions{Package: pkg, Out: b.out, Name: b.name, ID: b.id, Version: b.version, Icon: b.icon, Debug: b.debug,
		Hardened: b.hardened, Usage: map[string]string{}, MinMacOS: b.minMacOS, Sign: mactool.SignOptions{Identity: b.sign}}
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
	o.URLSchemes = append(o.URLSchemes, b.urlSchemes...)
	for _, v := range b.docTypes {
		t, ok := parseDocType(v)
		if !ok {
			return o, errors.New(tagMacOSUsage, "-doc-type takes NAME=ext[,ext...][:viewer|editor]", "value", v)
		}
		o.DocumentTypes = append(o.DocumentTypes, t)
	}
	return o, nil
}

// listFlags is a flag that may be given several times; the values are kept in their order.
type listFlags []string

func (l *listFlags) String() string { return strings.Join(*l, " ") }

// Set adds a value.
func (l *listFlags) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// parseDocType reads the value of -doc-type, NAME=ext[,ext...][:viewer|editor]: "BitTorrent file=torrent",
// "Text=txt,md:editor". The extensions may have their dot. What the names and the extensions may be is mactool.InfoPlist's
// to say.
func parseDocType(v string) (mactool.DocumentType, bool) {
	name, rest, ok := strings.Cut(v, "=")
	if !ok || strings.TrimSpace(name) == "" {
		return mactool.DocumentType{}, false
	}
	t := mactool.DocumentType{Name: strings.TrimSpace(name)}
	exts, role, hasRole := strings.Cut(rest, ":")
	if hasRole {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "viewer":
			t.Role = mactool.RoleViewer
		case "editor":
			t.Role = mactool.RoleEditor
		default:
			return mactool.DocumentType{}, false
		}
	}
	for _, e := range strings.Split(exts, ",") {
		if e = strings.TrimPrefix(strings.TrimSpace(e), "."); e == "" {
			return mactool.DocumentType{}, false
		}
		t.Extensions = append(t.Extensions, e)
	}
	return t, true
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

// tempDir makes a temporary directory and the function that removes it.
func tempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", func() {}, errors.NewErr(tagMacOSUsage, err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil // best effort: a temporary directory
}
