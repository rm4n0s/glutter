package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// runBuild is `glutter-ios build`: it makes the signed bundle of a package and says where it is.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	var bf buildFlags
	bf.add(fs, true)
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return err
	}
	o, err := bf.options(pkg, bf.sim)
	if err != nil {
		return err
	}
	res, err := iostool.Build(ctx, o)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s\n%s\n", res.Layout.App, summary(res, o.Simulator)); err != nil {
		return errors.NewErr(tagIOSUsage, err)
	}
	return nil
}

// summary says what a build made in one line.
func summary(res iostool.BuildResult, simulator bool) string {
	target := "device"
	if simulator {
		target = "Simulator"
	}
	return fmt.Sprintf("%s %s (%s), for the %s, %.1f MB", res.Info.Name, res.Info.Version, res.Info.ID, target, float64(res.Size)/(1<<20))
}
