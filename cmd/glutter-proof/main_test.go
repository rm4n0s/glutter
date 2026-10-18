package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestRun(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sub", "shapes.png")
	var stdout, stderr bytes.Buffer

	if err := run([]string{"-list"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "shapes-basic\t640x480\t") {
		t.Errorf("-list printed %q", stdout.String())
	}

	if err := run([]string{"-scene", "shapes-basic", "-o", out}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 640 || b.Dy() != 480 {
		t.Errorf("image size = %v", b)
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		tag  string
	}{
		{"no arguments", nil, "ProofUsage"},
		{"no output", []string{"-scene", "shapes-basic"}, "ProofUsage"},
		{"bad flag", []string{"-nope"}, "ProofUsage"},
		{"unknown scene", []string{"-scene", "nope", "-o", filepath.Join(t.TempDir(), "x.png")}, "ProofSceneUnknown"},
		{"cannot write", []string{"-scene", "shapes-basic", "-o", t.TempDir()}, "GoldenWriteFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) {
				t.Errorf("expected %s, got %v", tt.tag, err)
			}
		})
	}
}
