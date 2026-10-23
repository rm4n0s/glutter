package semantics

import (
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

func itoa(i int) string { return strconv.Itoa(i) }

// assertNotEmpty panics with the tag in debug builds if the string is empty.
func assertNotEmpty(tag, name, value string) {
	if foundation.DebugMode && value == "" {
		panic(errors.New(tag, "the "+name+" of a custom semantics action must not be empty"))
	}
}

// assertNoDuplicates panics in debug builds with SemanticsConfigurationDuplicated
// if a configuration is in the lists twice.
func assertNoDuplicates(mergeUp []*SemanticsConfiguration, groups [][]*SemanticsConfiguration) {
	if !foundation.DebugMode {
		return
	}
	seen := map[*SemanticsConfiguration]bool{}
	check := func(c *SemanticsConfiguration) {
		if seen[c] {
			panic(errors.New(tagSemanticsConfigurationDuplicated,
				"Duplicated SemanticsConfigurations. This can happen if the same SemanticsConfiguration was marked twice in MarkAsMergeUp and/or MarkAsSiblingMergeGroup"))
		}
		seen[c] = true
	}
	for _, c := range mergeUp {
		check(c)
	}
	for _, g := range groups {
		for _, c := range g {
			check(c)
		}
	}
}

// optEqual is Dart's == for two nullable comparable values.
func optEqual[T comparable](a, b foundation.Opt[T]) bool {
	av, aok := a.Get()
	bv, bok := b.Get()
	return aok == bok && av == bv
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSemanticsConfigurationDuplicated = "SemanticsConfigurationDuplicated"
)
