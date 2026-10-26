package examples

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readFile(t *testing.T, path ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// specExamples are the names that SPEC §19 lists, in backticks, in the paragraph under its
// heading.
func specExamples(t *testing.T) []string {
	t.Helper()
	spec := readFile(t, "..", "docs", "SPEC.md")
	_, section, ok := strings.Cut(spec, "## 19. Examples (`examples/`)\n")
	if !ok {
		t.Fatal("SPEC.md has no section 19")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	names := regexp.MustCompile("`([a-z]+)`").FindAllStringSubmatch(section, -1)
	var out []string
	for _, n := range names {
		out = append(out, n[1])
	}
	return out
}

// TestEveryExampleOfTheSpecExists checks what step 18.2 asks: each example of SPEC §19 has a main
// package, a row in the README, and a screenshot there that is in docs/implemented-proof-images.
func TestEveryExampleOfTheSpecExists(t *testing.T) {
	names := specExamples(t)
	if len(names) < 10 {
		t.Fatalf("SPEC §19 lists %d examples, want at least 10: %v", len(names), names)
	}
	readme := readFile(t, "..", "README.md")
	rows := map[string]string{}
	for _, line := range strings.Split(readme, "\n") {
		if m := regexp.MustCompile("^\\| \\[`([a-z]+)`\\]\\(examples/").FindStringSubmatch(line); m != nil {
			rows[m[1]] = line
		}
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(name, "main.go")); err != nil {
				t.Errorf("the example has no main.go: %v", err)
			}
			row, ok := rows[name]
			if !ok {
				t.Fatal("the README has no row for the example")
			}
			image := regexp.MustCompile(`\]\((docs/implemented-proof-images/[^)]+\.png)\)`).FindStringSubmatch(row)
			if image == nil {
				t.Fatal("the row of the README has no screenshot")
			}
			if _, err := os.Stat(filepath.Join("..", image[1])); err != nil {
				t.Errorf("the screenshot is missing: %v", err)
			}
		})
	}
}

// TestREADMEImagesExist checks that every image that the README shows is in the repository.
func TestREADMEImagesExist(t *testing.T) {
	readme := readFile(t, "..", "README.md")
	images := regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`).FindAllStringSubmatch(readme, -1)
	if len(images) == 0 {
		t.Fatal("the README shows no image")
	}
	for _, m := range images {
		if _, err := os.Stat(filepath.Join("..", m[1])); err != nil {
			t.Errorf("the README shows %s: %v", m[1], err)
		}
	}
}

// notWebExamples are the examples that do not run on the web (SPEC §19, §23): a page has no second window and
// no tray.
var notWebExamples = map[string]bool{"multiwindow": true, "tray": true}

// webExamples are the directories of examples/ that hold a main package, except those that are not for the web.
// They are found by reading the directory, so that a new example is built too.
func webExamples(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "dev" || notWebExamples[e.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(e.Name(), "main.go")); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

// allExamples are the directories of examples/ that hold a main package, `tray` and `multiwindow` included: every
// example runs on Windows (SPEC §19, §25). They are found by reading the directory, so that a new example is built
// too.
func allExamples(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "dev" {
			continue
		}
		if _, err := os.Stat(filepath.Join(e.Name(), "main.go")); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

// TestEveryWebExampleBuilds checks what step 21.7 asks: every example except multiwindow and tray builds for
// js/wasm with the standard Go toolchain (SPEC §19, §23.8). The builds are cut short by -short, because the
// first one after a change builds the framework for the web.
func TestEveryWebExampleBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds every example for js/wasm; skipped by -short")
	}
	names := webExamples(t)
	if len(names) < 9 {
		t.Fatalf("found %d examples for the web, want at least 9: %v", len(names), names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			// One after the other: the first build compiles the framework for js/wasm and the
			// others find it in the build cache.
			buildForWeb(t, name)
		})
	}
}

// TestSpecListsTheExamplesThatAreNotForTheWeb keeps notWebExamples and SPEC §19 together: both examples that the
// spec leaves out of the web exist, and the web examples are all the other ones.
func TestSpecListsTheExamplesThatAreNotForTheWeb(t *testing.T) {
	spec := readFile(t, "..", "docs", "SPEC.md")
	if !strings.Contains(spec, "On the web (1.1) every example except `multiwindow` and `tray` builds and runs") {
		t.Error("SPEC §19 no longer says which examples are not for the web")
	}
	listed := map[string]bool{}
	for _, n := range specExamples(t) {
		listed[n] = true
	}
	for name := range notWebExamples {
		if !listed[name] {
			t.Errorf("%s is not an example of SPEC §19", name)
		}
	}
	for _, name := range webExamples(t) {
		if !listed[name] {
			t.Errorf("%s builds for the web but SPEC §19 does not list it", name)
		}
	}
}

// TestSpecListsEveryExampleThatRunsOnWindows keeps allExamples and SPEC §19 together: the spec lists each of the
// examples, says that every one runs on Windows, and names the two tests of step 28.15.
func TestSpecListsEveryExampleThatRunsOnWindows(t *testing.T) {
	spec := readFile(t, "..", "docs", "SPEC.md")
	for _, want := range []string{
		"On Windows (1.8) every example builds and runs, `tray` and `multiwindow` included",
		"TestEveryExampleDrawsItsFirstFrameUnderWine",
	} {
		if !strings.Contains(spec, want) {
			t.Errorf("SPEC §19 no longer says %q", want)
		}
	}
	listed := map[string]bool{}
	for _, n := range specExamples(t) {
		listed[n] = true
	}
	names := allExamples(t)
	if len(names) < 11 {
		t.Errorf("found %d examples, want at least 11 (SPEC §19): %v", len(names), names)
	}
	for _, name := range names {
		if !listed[name] {
			t.Errorf("%s is an example but SPEC §19 does not list it", name)
		}
	}
}
