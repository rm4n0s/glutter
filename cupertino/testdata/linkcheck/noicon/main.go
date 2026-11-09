// Command noicon imports package cupertino and uses no icon; the test of the package
// links it and checks that the CupertinoIcons font is not in the binary.
package main

import (
	"fmt"

	"github.com/rm4n0s/glutter/cupertino"
)

func main() {
	fmt.Println(cupertino.CupertinoColors.SystemBlue.Color)
}
