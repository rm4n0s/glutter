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
		fmt.Fprintln(os.Stderr, "glutter-ios:", msg) // the process ends next; nowhere to report a failed write
		os.Exit(1)
	}
}

const usage = `usage:
  glutter-ios build [-sim] [-o App.app] [-name n] [-id dev.x.y] [-version v] [-icon file.png] [-team ID -profile file.mobileprovision] [-identity name] [-debug] [-orientations portrait,landscape-left,...] [-usage KEY=text]... <package>
  glutter-ios run [build flags] [-device iphone|ipad] [-set dir] [-env GLUTTER_X=v]... <package>
  glutter-ios shot [-device iphone|ipad] [-set dir] [-screen] [-env GLUTTER_X=v]... [-timeout d] -o file.png <package>`

// run does what the command line asks. stdout gets what the commands print.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(tagIOSUsage, usage)
	}
	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "run":
		return runRun(ctx, args[1:], stdout, stderr)
	case "shot":
		return runShot(ctx, args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		if _, err := fmt.Fprintln(stdout, usage); err != nil {
			return errors.NewErr(tagIOSUsage, err)
		}
		return nil
	}
	return errors.New(tagIOSUsage, "unknown command "+args[0]+"\n"+usage)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSUsage = "IOSUsage"
)
