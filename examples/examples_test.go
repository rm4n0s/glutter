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
