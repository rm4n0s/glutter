package main

import (
	"flag"
	"io"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// listFlags collects the values of a flag that can be repeated, NAME=value.
type listFlags struct {
	what   string
	values []string
}

func (l *listFlags) String() string { return strings.Join(l.values, " ") }
func (l *listFlags) Set(v string) error {
	if !strings.Contains(v, "=") {
		return errors.New(tagIOSUsage, l.what+" takes NAME=value", "value", v)
	}
	l.values = append(l.values, v)
	return nil
}

// buildFlags are the flags of `build`, which `run` and `shot` take too.
type buildFlags struct {
	sim          bool
	out          string
	name, id     string
	version      string
	icon         string
	team         string
	profile      string
	identity     string
	debug        bool
	orientations string
	usage        listFlags
}

// add declares the flags; sim says whether -sim is a flag (`run` and `shot` always build for the Simulator).
func (b *buildFlags) add(fs *flag.FlagSet, sim bool) {
	b.usage.what = "-usage"
	if sim {
		fs.BoolVar(&b.sim, "sim", false, "build for the iOS Simulator, signed ad hoc, instead of for a device")
	}
	fs.StringVar(&b.out, "o", "", "the bundle to write (<name>.app by default)")
	fs.StringVar(&b.name, "name", "", "the name of the app (the last element of the package by default)")
	fs.StringVar(&b.id, "id", "", "the bundle identifier (dev.glutter.<name> by default)")
	fs.StringVar(&b.version, "version", "", "the version of the app ("+iostool.DefaultVersion+" by default)")
	fs.StringVar(&b.icon, "icon", "", "a square PNG of at least 167 pixels for the icon (Glutter's by default)")
	fs.StringVar(&b.team, "team", "", "the developer team of a build for a device")
	fs.StringVar(&b.profile, "profile", "", "the provisioning profile of a build for a device")
	fs.StringVar(&b.identity, "identity", "", "the signing identity of a build for a device (Apple Development by default)")
	fs.BoolVar(&b.debug, "debug", false, "build with -tags debug_mode, not stripped (a release build, stripped, otherwise)")
	fs.StringVar(&b.orientations, "orientations", "", "the orientations to support, separated by commas: portrait, portrait-upside-down, landscape-left, landscape-right")
	fs.Var(&b.usage, "usage", "a usage string of the Info.plist, KEY=text (repeatable)")
}

var orientationNames = map[string]string{
	"portrait":             iostool.OrientationPortrait,
	"portrait-upside-down": iostool.OrientationPortraitUpsideDown,
	"landscape-left":       iostool.OrientationLandscapeLeft,
	"landscape-right":      iostool.OrientationLandscapeRight,
}

// options turns the flags into the options of the build.
func (b *buildFlags) options(pkg string, simulator bool) (iostool.BuildOptions, error) {
	o := iostool.BuildOptions{Package: pkg, Out: b.out, Simulator: simulator, Debug: b.debug, Team: b.team, Profile: b.profile,
		Identity: b.identity, ID: b.id, Name: b.name, Version: b.version, Icon: b.icon, Usage: map[string]string{}}
	for _, kv := range b.usage.values {
		k, v, _ := strings.Cut(kv, "=")
		o.Usage[k] = v
	}
	if b.orientations != "" {
		for _, n := range strings.Split(b.orientations, ",") {
			v, ok := orientationNames[strings.TrimSpace(n)]
			if !ok {
				return o, errors.New(tagIOSUsage, "an orientation is not portrait, portrait-upside-down, landscape-left or landscape-right", "value", n)
			}
			o.Orientations = append(o.Orientations, v)
		}
	}
	return o, nil
}

// parseWith parses the flags of fs and returns the one package that remains.
func parseWith(fs *flag.FlagSet, args []string, stderr io.Writer) (string, error) {
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return "", errors.NewErr(tagIOSUsage, err)
	}
	if fs.NArg() != 1 {
		return "", errors.New(tagIOSUsage, "the command needs one package\n"+usage)
	}
	return fs.Arg(0), nil
}
