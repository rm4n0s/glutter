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
		fmt.Fprintln(os.Stderr, "glutter-macos:", msg) // the process ends next; nowhere to report a failed write
		os.Exit(1)
	}
}

const usage = `usage:
  glutter-macos build [-o App.app] [-name n] [-id dev.x.y] [-version v] [-icon file.png] [-universal] [-hardened] [-debug] [-usage KEY=text]... <package>
  glutter-macos run [build flags] [-env GLUTTER_X=v]... <package>
  glutter-macos shot [-env GLUTTER_X=v]... [-front] [-bundle [build flags]] [-timeout d] -o file.png <package>
  glutter-macos perf [build flags] [-starts n] [-seconds n] [-settle d] [-env GLUTTER_X=v]... [-renderer r] [-format text|markdown] [-o file] <package>`

// run does what the command line asks. stdout gets what the commands print.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(tagMacOSUsage, usage)
	}
	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "run":
		return runRun(ctx, args[1:], stdout, stderr)
	case "shot":
		return runShot(ctx, args[1:], stdout, stderr)
	case "perf":
		return runPerf(ctx, args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		if _, err := fmt.Fprintln(stdout, usage); err != nil {
			return errors.NewErr(tagMacOSUsage, err)
		}
		return nil
	}
	return errors.New(tagMacOSUsage, "unknown command "+args[0]+"\n"+usage)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSUsage = "MacOSUsage"
)
