package main

import "flag"

// newFlagSetForTest makes the flags of `build` over a buildFlags.
func newFlagSetForTest(bf *buildFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	bf.add(fs, true)
	return fs
}
