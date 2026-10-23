// Package androiddex implements the `glutter-gen android-dex` subcommand: it compiles the Java
// of the Android embedding with javac and d8 into internal/androidtool/classes.dex (D22,
// SPEC §24.2).
//
// Flutter: no counterpart.
package androiddex

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// Run executes `glutter-gen android-dex [-root dir]`: from the module root it reads
// internal/androidtool/java and writes internal/androidtool/classes.dex. It needs javac and
// the SDK's build tools and platform.
//
// Flutter: no counterpart
func Run(args []string) error {
	flags := flag.NewFlagSet("android-dex", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "root directory of the module")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagAndroidDexBadFlags, err).SetMetadata("args", args)
	}
	home, err := androidtool.FindHome(os.Getenv)
	if err != nil {
		return err
	}
	tools, err := androidtool.FindBuildTools(home)
	if err != nil {
		return err
	}
	platform, err := androidtool.FindPlatform(home)
	if err != nil {
		return err
	}
	sdk := &androidtool.SDK{Home: home, BuildTools: tools, Platform: platform}
	return androidtool.CompileDex(context.Background(), androidtool.ExecRunner{}, sdk,
		filepath.Join(*root, androidtool.JavaDir), filepath.Join(*root, "internal", "androidtool", "classes.dex"))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidDexBadFlags = "AndroidDexBadFlags"
)
