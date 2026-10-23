package androidtool

import (
	"context"
	_ "embed" // classes.dex
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// Dex is the compiled Java of the embedding (D22), checked in as classes.dex. `go run
// ./cmd/glutter-gen android-dex` makes it from java/; a test checks that it is up to date.
//
//go:embed classes.dex
var Dex []byte

// JavaDir is the directory of the embedding's Java sources, relative to the module root.
const JavaDir = "internal/androidtool/java"

// CompileDex compiles the .java files below javaDir with javac (--release 11, against the
// platform's android.jar) and d8 (--min-api 26), and writes the single classes.dex to out.
// It fails with AndroidJavaNotFound when javac is not installed, and with
// AndroidCommandFailed when a compiler rejects the code.
//
// Flutter: no counterpart (Gradle compiles FlutterActivity and the rest)
func CompileDex(ctx context.Context, run Runner, sdk *SDK, javaDir, out string) error {
	if _, err := lookPath("javac"); err != nil {
		return errors.NewErr(TagAndroidJavaNotFound, err).SetMetadata("install", "sudo dnf install java-25-openjdk-devel")
	}
	var sources []string
	err := filepath.WalkDir(javaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".java") {
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", javaDir)
	}
	sort.Strings(sources)
	if len(sources) == 0 {
		return errors.New(TagAndroidWriteFailed, "there is no .java file", "dir", javaDir)
	}
	work, err := os.MkdirTemp("", "glutter-dex-")
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("what", "the temporary directory of the dex")
	}
	defer func() { _ = os.RemoveAll(work) }() // best effort: a temporary directory
	classes := filepath.Join(work, "classes")
	if err := os.MkdirAll(classes, 0o755); err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", classes)
	}
	javac := append([]string{"--release", "11", "-Xlint:all", "-Xlint:-options", "-classpath", sdk.AndroidJar(), "-d", classes}, sources...)
	if _, err := run.Run(ctx, Command{Name: "javac", Args: javac}); err != nil {
		return err
	}
	var classFiles []string
	err = filepath.WalkDir(classes, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".class") {
			classFiles = append(classFiles, path)
		}
		return err
	})
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", classes)
	}
	sort.Strings(classFiles)
	dexDir := filepath.Join(work, "dex")
	if err := os.MkdirAll(dexDir, 0o755); err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", dexDir)
	}
	d8 := append([]string{"--min-api", strconv.Itoa(MinSDK), "--lib", sdk.AndroidJar(), "--output", dexDir}, classFiles...)
	if _, err := run.Run(ctx, Command{Name: sdk.Tool("d8"), Args: d8}); err != nil {
		return err
	}
	dex, err := os.ReadFile(filepath.Join(dexDir, "classes.dex"))
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", dexDir)
	}
	if err := os.WriteFile(out, dex, 0o644); err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
	}
	return nil
}

// TagAndroidJavaNotFound is the tag of the error for a missing javac, which is needed only to
// change the Java of the embedding; the metadata "install" has the command.
const TagAndroidJavaNotFound = "AndroidJavaNotFound"
