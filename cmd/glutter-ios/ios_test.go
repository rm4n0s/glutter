package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

func requireUsage(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is %T: %v", err, err)
	}
	if appErr.Tag != tagIOSUsage || !appErr.HasRoute(route) {
		t.Errorf("tag %s, route %s; want IOSUsage through %s", appErr.Tag, appErr.Route(), route)
	}
}

func TestRunRejectsWhatItDoesNotKnow(t *testing.T) {
	var out, errOut bytes.Buffer
	requireUsage(t, run(context.Background(), nil, &out, &errOut), "run")
	requireUsage(t, run(context.Background(), []string{"launch"}, &out, &errOut), "run")
	requireUsage(t, run(context.Background(), []string{"build"}, &out, &errOut), "runBuild->parseWith")
	requireUsage(t, run(context.Background(), []string{"run"}, &out, &errOut), "runRun->parseWith")
	requireUsage(t, run(context.Background(), []string{"shot", "./x"}, &out, &errOut), "runShot")
	requireUsage(t, run(context.Background(), []string{"shot", "-o", "out.jpg", "./x"}, &out, &errOut), "runShot")
	requireUsage(t, run(context.Background(), []string{"build", "-nonsense", "./x"}, &out, &errOut), "runBuild->parseWith")
	requireUsage(t, run(context.Background(), []string{"build", "-usage", "novalue", "./x"}, &out, &errOut), "runBuild->parseWith")
	requireUsage(t, run(context.Background(), []string{"build", "-orientations", "sideways", "./x"}, &out, &errOut), "runBuild->buildFlags.options")
	if err := run(context.Background(), []string{"help"}, &out, &errOut); err != nil || !strings.Contains(out.String(), "glutter-ios build") {
		t.Errorf("help: %v, %q", err, out.String())
	}
}

func TestBuildFlagsBecomeOptions(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	var bf buildFlags
	bf.add(fs, true)
	pkg, err := parseWith(fs, []string{"-sim", "-o", "X.app", "-name", "X", "-id", "dev.glutter.x", "-version", "2", "-debug",
		"-orientations", "portrait, landscape-left", "-usage", "NSCameraUsageDescription=to scan", "-usage", "A=b=c", "./examples/x"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	o, err := bf.options(pkg, bf.sim)
	if err != nil {
		t.Fatal(err)
	}
	if o.Package != "./examples/x" || !o.Simulator || o.Out != "X.app" || o.Name != "X" || o.ID != "dev.glutter.x" || o.Version != "2" || !o.Debug {
		t.Errorf("options = %+v", o)
	}
	if want := []string{iostool.OrientationPortrait, iostool.OrientationLandscapeLeft}; !slices.Equal(o.Orientations, want) {
		t.Errorf("orientations = %v, want %v", o.Orientations, want)
	}
	if o.Usage["NSCameraUsageDescription"] != "to scan" || o.Usage["A"] != "b=c" {
		t.Errorf("usage = %v", o.Usage)
	}
}

func TestSummary(t *testing.T) {
	res := iostool.BuildResult{Info: iostool.Info{Name: "Counter", Version: "1.0.0", ID: "dev.glutter.counter"}, Size: 3 << 20}
	if got, want := summary(res, true), "Counter 1.0.0 (dev.glutter.counter), for the Simulator, 3.0 MB"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if got := summary(res, false); !strings.Contains(got, "for the device") {
		t.Errorf("summary = %q", got)
	}
}
