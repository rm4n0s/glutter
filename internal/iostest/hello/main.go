// Command hello is the app of the Simulator tests of step 35.1: it starts UIKit through package ios and prints a line when the app
// has launched and a line when the scene has connected, so that the test sees them in the console of the app.
package main

import (
	"fmt"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostest/hellolines"
	_ "github.com/rm4n0s/glutter/internal/platform/darwin/mainthread" // locks the main goroutine to the main thread
	"github.com/rm4n0s/glutter/internal/platform/ios"
)

func main() {
	if os.Getenv(hellolines.ModeEnv) == "goroutine" {
		// Main on another thread than the main one is an error, which the test reads in the console.
		go func() {
			err := ios.Main(ios.MainOptions{})
			tag, route := "", false
			if appErr, ok := errors.FromError(err); ok {
				tag, route = appErr.Tag, appErr.HasRoute("Main->launch."+ios.TagIOSNotMainThread)
			}
			fmt.Printf("%s tag=%s route=%v\n", hellolines.NotMainThread, tag, route)
			os.Exit(0)
		}()
		select {}
	}
	err := ios.Main(ios.MainOptions{
		DidFinishLaunching: func() { fmt.Println(hellolines.Launched, os.Getenv(hellolines.EnvName)) },
		SceneDidConnect:    func() { fmt.Println(hellolines.SceneConnected) },
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
