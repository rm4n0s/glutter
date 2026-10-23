package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
		in    string // what the message has
	}{
		{name: "no command", args: nil, route: "run." + tagAndroidUsage, in: "glutter-android build"},
		{name: "unknown command", args: []string{"fly"}, route: "run." + tagAndroidUsage, in: "unknown command fly"},
		{name: "build without a package", args: []string{"build"}, route: "run->runBuild->parse." + tagAndroidUsage, in: "one package is needed"},
		{name: "build with two packages", args: []string{"build", "a", "b"}, route: "run->runBuild->parse." + tagAndroidUsage, in: "one package is needed"},
		{name: "build with a bad flag", args: []string{"build", "-nope", "a"}, route: "run->runBuild->parse." + tagAndroidUsage},
		{name: "install without a package", args: []string{"install"}, route: "run->runInstall->parse." + tagAndroidUsage},
		{name: "run without a package", args: []string{"run"}, route: "run->runRun->parse." + tagAndroidUsage},
		{name: "run with an env that is not GLUTTER_", args: []string{"run", "-env", "HOME=/x", "./examples/counter"}, route: "run->runRun->parseEnvs." + tagAndroidUsage, in: "GLUTTER_NAME=value"},
		{name: "run with an env with no value", args: []string{"run", "-env", "GLUTTER_X", "./examples/counter"}, route: "run->runRun->parseEnvs." + tagAndroidUsage},
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

func TestHelp(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"help"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"glutter-android build", "glutter-android install", "glutter-android run"} {
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
	if err := fs.Parse([]string{"-o", "x.apk", "-abi", "arm64, amd64", "-release", "-app-id", "a.b", "-label", "L", "-permission", "p.one", "-permission", "p.two", "-key", "k", "-cert", "c", "-vulkan-validation", "/vvl"}); err != nil {
		t.Fatal(err)
	}
	o := bf.options("./pkg")
	if o.Package != "./pkg" || o.Out != "x.apk" || strings.Join(o.ABIs, "|") != "arm64|amd64" || !o.Release || o.AppID != "a.b" ||
		o.Label != "L" || strings.Join(o.Permissions, "|") != "p.one|p.two" || o.Key != "k" || o.Cert != "c" || o.ValidationLayerDir != "/vvl" {
		t.Errorf("options %+v", o)
	}
}

func TestSize(t *testing.T) {
	if got := size(3 << 19); got != "1.5 MB" {
		t.Errorf("size = %s", got)
	}
}

func TestShotUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
		in    string
	}{
		{name: "no output file", args: []string{"shot", "./examples/counter"}, route: "run->runShot." + tagAndroidUsage, in: "-o"},
		{name: "no package", args: []string{"shot", "-o", "x.png"}, route: "run->runShot->parse." + tagAndroidUsage},
		{name: "a rotation that is not 0 to 3", args: []string{"shot", "-o", "x.png", "-rotation", "4", "./examples/counter"}, route: "run->runShot." + tagAndroidUsage, in: "-rotation"},
		{name: "an env that is not GLUTTER_", args: []string{"shot", "-o", "x.png", "-env", "PATH=/x", "./examples/counter"}, route: "run->runShot->parseEnvs." + tagAndroidUsage, in: "GLUTTER_NAME=value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, io.Discard, io.Discard)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.route) || !strings.Contains(appErr.Message, tt.in) {
				t.Errorf("got %v, want route %s with %q", err, tt.route, tt.in)
			}
		})
	}
}

func TestParseEnvs(t *testing.T) {
	got, err := parseEnvs([]string{"GLUTTER_A=1", "GLUTTER_B=x=y", "GLUTTER_C="})
	if err != nil || got["GLUTTER_A"] != "1" || got["GLUTTER_B"] != "x=y" || got["GLUTTER_C"] != "" || len(got) != 3 {
		t.Errorf("parseEnvs = %v, %v", got, err)
	}
}
