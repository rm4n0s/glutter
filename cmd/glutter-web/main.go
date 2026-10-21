package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/rm4n0s/errors"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		msg := err.Error()
		if appErr, ok := errors.FromError(err); ok {
			msg = appErr.Tag + ": " + appErr.Message
			for k, v := range appErr.Metadata {
				if s, isString := v.(string); isString && s != "" {
					msg += "\n  " + k + ": " + s
				}
			}
		}
		fmt.Fprintln(os.Stderr, "glutter-web:", msg) // the process ends next; nowhere to report a failed write
		os.Exit(1)
	}
}

const usage = `usage:
  glutter-web build [-o dir] [-release] [-host selector] <package>
  glutter-web serve [-addr :8080] [-release] [-host selector] <package>
  glutter-web shot  -browser chromium|firefox [-renderer r] [-query k=v&k=v] [-width n] [-height n] -o file.png <package>`

// run does what the command line asks. stdout gets what the commands print; flag errors go to
// stderr.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("WebUsage", usage)
	}
	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "serve":
		return runServe(ctx, args[1:], stdout, stderr)
	case "shot":
		return runShot(ctx, args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		_, err := fmt.Fprintln(stdout, usage)
		if err != nil {
			return errors.NewErr("WebUsage", err)
		}
		return nil
	}
	return errors.New("WebUsage", "unknown command "+args[0]+"\n"+usage)
}
