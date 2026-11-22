//go:build darwin && !ios && glutter_display

package examples

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/mactool"
)

// lsregister is the tool of Launch Services that registers a bundle and forgets it.
const lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

// TestLaunchServicesSendsTheAppItsLinksAndDocuments is Theatro's issue (step 56.1) from end to end, as the Finder
// and a browser do it: `glutter-macos build` writes a URL scheme and a file type into the plist of
// examples/dev/macos-open, Launch Services learns the bundle, and `open` of a file of the type starts the app,
// which gets the file as an open request though it was not listening yet; `open` of a link of the scheme and of
// two more files then reach the running app. The scheme, the extension and the bundle identifier are this test
// run's own, so that no other app of the Mac answers, and Launch Services forgets the bundle at the end. The bundle is in
// .macos-test at the module root: Launch Services does not use an app that is in a temporary directory. The app
// is an accessory app and is opened in the background (open -g): it takes no focus.
func TestLaunchServicesSendsTheAppItsLinksAndDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and starts an app; skipped by -short")
	}
	if _, err := os.Stat(lsregister); err != nil {
		t.Skipf("no lsregister: %v", err)
	}
	// Launch Services gives no link and no document to an app in a temporary directory (t.TempDir is under /var/folders),
	// so the bundle and the documents are in .macos-test at the module root, which git ignores, for the time of the test.
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, tmp := range []string{"/tmp/", "/private/tmp/", "/var/folders/", "/private/var/folders/"} {
		if strings.HasPrefix(root, tmp) {
			t.Skipf("the module is in a temporary directory (%s), where Launch Services uses no app", root)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".macos-test"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(filepath.Join(root, ".macos-test"), "open-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	run := fmt.Sprintf("p%dt%d", os.Getpid(), time.Now().Unix())
	scheme, ext := "glutter-open-"+run, "glutteropen"+run
	res, err := mactool.Build(ctx, mactool.BuildOptions{Debug: true, Package: "./examples/dev/macos-open", Dir: "..",
		Out: filepath.Join(dir, "Open.app"), Name: "Open", ID: "dev.glutter.opentest." + run,
		URLSchemes:    []string{scheme},
		DocumentTypes: []mactool.DocumentType{{Name: "Glutter open test", Extensions: []string{ext}}}})
	if err != nil {
		t.Fatalf("glutter-macos build: %v", err)
	}
	app := res.Layout.App
	if out, err := exec.CommandContext(ctx, lsregister, "-f", app).CombinedOutput(); err != nil {
		t.Fatalf("lsregister -f: %v\n%s", err, out)
	}
	logFile := filepath.Join(dir, "open.log")
	lines := func() []string {
		data, err := os.ReadFile(logFile)
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	t.Cleanup(func() {
		// The app is stopped, and Launch Services forgets the bundle before the directory is removed.
		for _, l := range lines() {
			if pid, err := strconv.Atoi(strings.TrimPrefix(l, "ready ")); err == nil && strings.HasPrefix(l, "ready ") {
				if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
					t.Errorf("stopping the app (pid %d): %v", pid, err)
				}
			}
		}
		if out, err := exec.Command(lsregister, "-u", app).CombinedOutput(); err != nil {
			t.Errorf("lsregister -u: %v\n%s", err, out)
		}
	})
	waitFor := func(what string, want ...string) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for {
			got := lines()
			missing := ""
			for _, w := range want {
				found := false
				for _, l := range got {
					found = found || l == w
				}
				if !found {
					missing = w
				}
			}
			if missing == "" {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: the app did not log %q; its log:\n%s", what, missing, strings.Join(got, "\n"))
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	file := func(name string) string {
		t.Helper()
		path := filepath.Join(dir, name+"."+ext)
		if err := os.WriteFile(path, []byte("a document of the test"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	open := func(args ...string) {
		t.Helper()
		args = append([]string{"-g", "--env", "GLUTTER_DEV_OPEN_LOG=" + logFile, "--env", "GLUTTER_MACOS_ACCESSORY=1"}, args...)
		if out, err := exec.CommandContext(ctx, "open", args...).CombinedOutput(); err != nil {
			t.Fatalf("open %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	// A double click on a document of the app's type, with the app not running: Launch Services finds the app by
	// the extension, starts it, and the file comes as an open request before the first widget listens.
	first := file("the first")
	open(first)
	waitFor("the document that starts the app", "file "+first)
	if got := lines(); len(got) == 0 || !strings.HasPrefix(got[0], "ready ") {
		t.Errorf("the log does not start with the app's start: %q", got)
	}

	// A click on a link of the app's scheme, with the app running.
	link := scheme + "://host/path?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn=A%20name"
	open(link)
	waitFor("the link", "url "+link)

	// Two more documents at once.
	second, third := file("second"), file("third")
	open(second, third)
	waitFor("two documents", "file "+second, "file "+third)

	starts := 0
	for _, l := range lines() {
		if strings.HasPrefix(l, "ready ") {
			starts++
		}
	}
	if starts != 1 {
		t.Errorf("the app started %d times, want once: the later requests go to the running app\n%s", starts, strings.Join(lines(), "\n"))
	}
}
