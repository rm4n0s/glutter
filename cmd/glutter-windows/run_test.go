package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/wintool"
)

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
		in    string // what the message has
	}{
		{name: "no command", args: nil, route: "run." + tagWindowsUsage, in: "glutter-windows build"},
		{name: "unknown command", args: []string{"fly"}, route: "run." + tagWindowsUsage, in: "unknown command fly"},
		{name: "build without a package", args: []string{"build"}, route: "run->runBuild->parse." + tagWindowsUsage, in: "one package is needed"},
		{name: "build with two packages", args: []string{"build", "a", "b"}, route: "run->runBuild->parse." + tagWindowsUsage, in: "one package is needed"},
		{name: "build with a bad flag", args: []string{"build", "-nope", "a"}, route: "run->runBuild->parse." + tagWindowsUsage},
		{name: "run without a package", args: []string{"run"}, route: "run->runRun->parse." + tagWindowsUsage},
		{name: "run with an env that is not GLUTTER_", args: []string{"run", "-env", "HOME=/x", "./examples/counter"}, route: "run->runRun->parseEnvs." + tagWindowsUsage, in: "GLUTTER_NAME=value"},
		{name: "run with an env with no value", args: []string{"run", "-env", "GLUTTER_X", "./examples/counter"}, route: "run->runRun->parseEnvs." + tagWindowsUsage},
		{name: "shot without a package", args: []string{"shot", "-o", "x.png"}, route: "run->runShot->parse." + tagWindowsUsage},
		{name: "shot without an output file", args: []string{"shot", "./examples/counter"}, route: "run->runShot." + tagWindowsUsage, in: "-o"},
		{name: "shot with an env that is not GLUTTER_", args: []string{"shot", "-o", "x.png", "-env", "PATH=/x", "./examples/counter"}, route: "run->runShot->parseEnvs." + tagWindowsUsage, in: "GLUTTER_NAME=value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, io.Discard, io.Discard)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
			if !strings.Contains(appErr.Message, tt.in) {
				t.Errorf("message %q has no %q", appErr.Message, tt.in)
			}
		})
	}
}

func TestBuildErrorsComeFromTheTool(t *testing.T) {
	// An architecture that is not known is the builder's error, passed up as it is.
	err := run(context.Background(), []string{"build", "-arch", "mips", "./examples/counter"}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != wintool.TagWindowsBuildInvalid || !appErr.HasRoute("run->runBuild->Build.WindowsBuildInvalid") {
		t.Errorf("got %v, want WindowsBuildInvalid via run->runBuild->Build", err)
	}
}

func TestHelp(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"help"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"glutter-windows build", "glutter-windows run", "glutter-windows shot", "-console", "-icon", "-version", "-screen"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help has no %q:\n%s", want, out.String())
		}
	}
}

func TestBuildFlagsOptions(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var bf buildFlags
	bf.add(fs, true)
	if err := fs.Parse([]string{"-o", "x.exe", "-arch", "amd64, arm64", "-release", "-console", "-name", "N", "-app-id", "a.b", "-icon", "i.png", "-version", "1.2"}); err != nil {
		t.Fatal(err)
	}
	o := bf.options("./pkg")
	if o.Package != "./pkg" || o.Out != "x.exe" || strings.Join(o.Archs, "|") != "amd64|arm64" || !o.Release || !o.Console ||
		o.Name != "N" || o.AppID != "a.b" || o.Icon != "i.png" || o.Version != "1.2" {
		t.Errorf("options %+v", o)
	}

	// run and shot take the build flags without -o and -arch: they build for this machine.
	fs = flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bf = buildFlags{}
	bf.add(fs, false)
	if err := fs.Parse([]string{"-o", "x.exe"}); err == nil {
		t.Error("run accepts -o")
	}
}

func TestParseEnvs(t *testing.T) {
	got, err := parseEnvs([]string{"GLUTTER_A=1", "GLUTTER_B=x=y", "GLUTTER_C="})
	if err != nil || len(got) != 3 {
		t.Errorf("parseEnvs = %v, %v", got, err)
	}
}

func TestSize(t *testing.T) {
	if got := size(3 << 19); got != "1.5 MB" {
		t.Errorf("size = %s", got)
	}
}

func TestSayBuilt(t *testing.T) {
	var out bytes.Buffer
	res := &wintool.BuildResult{AppID: "org.glutter.x", Exes: []wintool.Built{{Arch: "amd64", Path: "/o/x_amd64.exe", Size: 2 << 20}, {Arch: "arm64", Path: "/o/x_arm64.exe", Size: 3 << 20}}}
	if err := sayBuilt(&out, "./x", res); err != nil {
		t.Fatal(err)
	}
	want := "built ./x into /o/x_amd64.exe (amd64, 2.0 MB; org.glutter.x)\nbuilt ./x into /o/x_arm64.exe (arm64, 3.0 MB; org.glutter.x)\n"
	if out.String() != want {
		t.Errorf("sayBuilt =\n%s\nwant\n%s", out.String(), want)
	}
}
