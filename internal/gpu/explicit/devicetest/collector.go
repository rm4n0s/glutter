package devicetest

import (
	"fmt"
	"strings"
)

// Result is what one part of a scene came to.
//
// Flutter: no counterpart
type Result struct {
	// Name is the names of the scene and its parts, joined with "/".
	Name string `json:"name"`
	// Failures are the messages of the failures; empty when it passed.
	Failures []string `json:"failures"`
	// Skipped is why the part did not run, when it did not.
	Skipped string `json:"skipped,omitempty"`
}

// Collector is a T that records the failures instead of reporting them to a test runner, for a
// program that is not a test. Its parts run one after the other on the calling goroutine.
//
// Flutter: no counterpart
type Collector struct {
	name    string
	results *[]Result
	failed  []string
	fatal   bool
	skipped string
}

// NewCollector returns the T for the scenes of a program, and the function that returns the results
// of every part that ran, a part that has parts of its own included.
//
// Flutter: no counterpart
func NewCollector() (*Collector, func() []Result) {
	results := &[]Result{}
	return &Collector{results: results}, func() []Result { return *results }
}

// fatalStop is what a Fatalf panics with to end the part that called it, and skipStop what a Skipf
// panics with.
type (
	fatalStop struct{}
	skipStop  struct{}
)

// Skipf implements T.
func (c *Collector) Skipf(format string, args ...any) {
	c.skipped = fmt.Sprintf(format, args...)
	panic(skipStop{})
}

// Helper implements T.
func (c *Collector) Helper() {}

// Errorf implements T.
func (c *Collector) Errorf(format string, args ...any) {
	c.failed = append(c.failed, fmt.Sprintf(format, args...))
}

// Fatalf implements T.
func (c *Collector) Fatalf(format string, args ...any) {
	c.Errorf(format, args...)
	c.fatal = true
	panic(fatalStop{})
}

// Run implements T.
func (c *Collector) Run(name string, fn func(t T)) {
	child := &Collector{name: strings.TrimPrefix(c.name+"/"+name, "/"), results: c.results}
	func() {
		defer func() {
			if r := recover(); r != nil {
				switch r.(type) {
				case fatalStop, skipStop:
				default:
					child.failed = append(child.failed, fmt.Sprintf("panic: %v", r))
				}
			}
		}()
		fn(child)
	}()
	*c.results = append(*c.results, Result{Name: child.name, Failures: child.failed, Skipped: child.skipped})
	if len(child.failed) > 0 {
		c.failed = append(c.failed, child.failed...)
	}
}
