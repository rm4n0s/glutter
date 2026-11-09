package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/mactool"
)

// runBuild is `glutter-macos build`: it makes the signed bundle of a package and says where it is.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	var bf buildFlags
	bf.add(fs, true)
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return err
	}
	o, err := bf.options(pkg)
	if err != nil {
		return err
	}
	res, err := mactool.Build(ctx, o)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s\n%s\n", res.Layout.App, summary(res)); err != nil {
		return errors.NewErr(tagMacOSUsage, err)
	}
	return nil
}

// summary says what a build made in one line.
func summary(res mactool.BuildResult) string {
	parts := []string{fmt.Sprintf("%s %s (%s)", res.Info.Name, res.Info.Version, res.Info.ID), strings.Join(res.Archs, "+"), fmt.Sprintf("%.1f MB", float64(res.Size)/(1<<20))}
	return strings.Join(parts, ", ")
}
