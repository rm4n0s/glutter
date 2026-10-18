package main

import (
	"fmt"
	"os"
)

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		if _, werr := fmt.Fprint(os.Stderr, formatError(err)); werr != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
