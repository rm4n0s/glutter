// Command hello is the tiny Windows program of the winetest tests: it prints a line, its arguments
// and one environment variable, says whether it runs under Wine, and exits with the status that
// its first argument names, or sleeps for a minute when the argument is "sleep"
// (GOOS=windows go build ./internal/winetest/testdata/hello).
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/internal/winetest"
)

func main() {
	fmt.Println("hello from Windows")
	fmt.Println("args:", os.Args[1:])
	fmt.Println("GLUTTER_HELLO:", os.Getenv("GLUTTER_HELLO"))
	fmt.Println("wayland:", os.Getenv("WAYLAND_DISPLAY") != "")
	fmt.Println("under wine:", winetest.UnderWine())
	if len(os.Args) > 1 && os.Args[1] == "sleep" {
		time.Sleep(time.Minute)
	}
	if len(os.Args) > 1 {
		if code, err := strconv.Atoi(os.Args[1]); err == nil {
			os.Exit(code)
		}
	}
}
