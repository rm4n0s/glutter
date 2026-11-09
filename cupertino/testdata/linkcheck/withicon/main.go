// Command withicon uses one Cupertino icon; the test of the package links it and checks
// that the CupertinoIcons font is in the binary.
package main

import (
	"fmt"

	"github.com/rm4n0s/glutter/cupertino"
)

func main() {
	fmt.Println(cupertino.CupertinoIcons.Heart().CodePoint)
}
