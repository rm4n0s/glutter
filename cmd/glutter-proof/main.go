// Command glutter-proof renders a proof scene with the software renderer and
// saves it as a PNG, as proof that a step of docs/IMPLEMENTATION.md works.
//
//	go run ./cmd/glutter-proof -list
//	go run ./cmd/glutter-proof -scene shapes-basic -o docs/implemented-proof-images/2.6-shapes-basic.png
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	_ "github.com/rm4n0s/glutter/internal/proof/widgetscenes" // registers the scenes made with gluttertest
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintln(os.Stderr, "glutter-proof:", appErr.Message)
		} else {
			fmt.Fprintln(os.Stderr, "glutter-proof:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("glutter-proof", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("scene", "", "the proof scene to render")
	out := fs.String("o", "", "the PNG file to write")
	list := fs.Bool("list", false, "list the proof scenes")
	if err := fs.Parse(args); err != nil {
		return errors.NewErr("ProofUsage", err)
	}
	if *list {
		for _, n := range proof.Names() {
			s, err := proof.Lookup(n)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "%s\t%dx%d\t%s\n", s.Name, s.Width, s.Height, s.Description); err != nil {
				return errors.NewErr("ProofUsage", err)
			}
		}
		return nil
	}
	if *name == "" || *out == "" {
		return errors.New("ProofUsage", "usage: glutter-proof -scene <name> -o <file.png> | glutter-proof -list")
	}
	s, err := proof.Lookup(*name)
	if err != nil {
		return err
	}
	img, err := soft.Render(s.Build(), s.Width, s.Height)
	if err != nil {
		return err
	}
	return goldens.WritePNG(*out, img)
}
