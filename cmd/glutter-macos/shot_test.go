package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestParseShot(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    shotOptions
		wantTag string
	}{
		{"the minimum", []string{"-o", "a.png", "./examples/counter"}, shotOptions{pkg: "./examples/counter", out: "a.png", timeout: 90 * time.Second}, ""},
		{"env, front and timeout", []string{"-env", "GLUTTER_RENDERER=soft", "-front", "-timeout", "5s", "-o", "a.png", "./x"},
			shotOptions{pkg: "./x", out: "a.png", env: []string{"GLUTTER_RENDERER=soft"}, front: true, timeout: 5 * time.Second}, ""},
		{"no output file", []string{"./x"}, shotOptions{}, tagMacOSUsage},
		{"no package", []string{"-o", "a.png"}, shotOptions{}, tagMacOSUsage},
		{"two packages", []string{"-o", "a.png", "./x", "./y"}, shotOptions{}, tagMacOSUsage},
		{"a bad env", []string{"-env", "NOEQUALS", "-o", "a.png", "./x"}, shotOptions{}, tagMacOSUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShot(tt.args, io.Discard)
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag {
					t.Fatalf("err = %v, want tag %s", err, tt.wantTag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.pkg != tt.want.pkg || got.out != tt.want.out || got.front != tt.want.front || got.timeout != tt.want.timeout || !slices.Equal(got.env, tt.want.env) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestShotEnvLetsEnvFlagsWin(t *testing.T) {
	o := shotOptions{env: []string{"GLUTTER_MACOS_ACCESSORY=0", "GLUTTER_RENDERER=soft"}}
	got := shotEnv([]string{"HOME=/h"}, o, "/out.png")
	want := []string{"HOME=/h", "GLUTTER_SCREENSHOT=/out.png", "GLUTTER_MACOS_ACCESSORY=1", "GLUTTER_MACOS_ACCESSORY=0", "GLUTTER_RENDERER=soft"}
	if !slices.Equal(got, want) {
		t.Errorf("env = %v, want %v", got, want)
	}
	front := shotEnv(nil, shotOptions{front: true}, "/o.png")
	if slices.Contains(front, "GLUTTER_MACOS_ACCESSORY=1") {
		t.Error("-front still makes an accessory app")
	}
}

func TestRunErrors(t *testing.T) {
	var out bytes.Buffer
	tests := []struct {
		name string
		args []string
	}{{"no command", nil}, {"unknown command", []string{"dance"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, &out, io.Discard)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagMacOSUsage || !appErr.HasRoute("run."+tagMacOSUsage) {
				t.Errorf("err = %v", err)
			}
		})
	}
	if err := run(context.Background(), []string{"help"}, &out, io.Discard); err != nil || out.Len() == 0 {
		t.Errorf("help: %v, %d bytes", err, out.Len())
	}
}

func TestShotBuildFailureIsReported(t *testing.T) {
	err := runShot(context.Background(), []string{"-o", t.TempDir() + "/a.png", "./nonexistent-package"}, io.Discard, io.Discard)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagMacOSShotFailed || !appErr.HasRoute("runShot."+tagMacOSShotFailed) {
		t.Fatalf("err = %v", err)
	}
	if out, _ := appErr.Metadata["output"].(string); out == "" {
		t.Error("the output of the failed build is not in the error")
	}
}
