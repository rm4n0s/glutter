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
		fmt.Fprintln(os.Stderr, "glutter-android:", msg) // the process ends next; nowhere to report a failed write
		os.Exit(1)
	}
}

const usage = `usage:
  glutter-android build   [-o app.apk] [-abi arm64,amd64] [-release] [-key k.pk8 -cert c.pem] [-app-id id] [-label name] [-icon icon.png] [-permission NAME]... <package>
  glutter-android install [-device serial] [build flags] <package|app.apk>
  glutter-android run     [-device serial] [-env GLUTTER_X=v]... [build flags] <package>`

// run does what the command line asks. stdout gets what the commands print; flag errors go to
// stderr.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(tagAndroidUsage, usage)
	}
	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "install":
		return runInstall(ctx, args[1:], stdout, stderr)
	case "run":
		return runRun(ctx, args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		_, err := fmt.Fprintln(stdout, usage)
		if err != nil {
			return errors.NewErr(tagAndroidUsage, err)
		}
		return nil
	}
	return errors.New(tagAndroidUsage, "unknown command "+args[0]+"\n"+usage)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidUsage = "AndroidUsage"
)
