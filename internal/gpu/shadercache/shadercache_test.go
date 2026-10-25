package shadercache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestDir(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"xdg cache home", map[string]string{"XDG_CACHE_HOME": "/x/cache", "HOME": "/home/u"}, "/x/cache/glutter/shaders"},
		{"home", map[string]string{"HOME": "/home/u"}, "/home/u/.cache/glutter/shaders"},
		{"turned off", map[string]string{EnvVar: "0", "HOME": "/home/u"}, ""},
		{"on for any other value", map[string]string{EnvVar: "1", "HOME": "/home/u"}, "/home/u/.cache/glutter/shaders"},
		{"no home (the web)", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Dir(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("Dir = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStoreAndLoad(t *testing.T) {
	tests := []struct {
		name  string
		steps func(c *Cache) ([]byte, bool)
		want  string
		found bool
	}{
		{"a miss", func(c *Cache) ([]byte, bool) { return c.Load("a") }, "", false},
		{"a hit", func(c *Cache) ([]byte, bool) { c.Store("a", []byte("one")); return c.Load("a") }, "one", true},
		{"another key is a miss", func(c *Cache) ([]byte, bool) { c.Store("a", []byte("one")); return c.Load("b") }, "", false},
		{"a second store replaces the first", func(c *Cache) ([]byte, bool) {
			c.Store("a", []byte("one"))
			c.Store("a", []byte("two"))
			return c.Load("a")
		}, "two", true},
		{"empty data is not stored", func(c *Cache) ([]byte, bool) { c.Store("a", nil); return c.Load("a") }, "", false},
		{"a key that looks like a path stays in the directory", func(c *Cache) ([]byte, bool) {
			c.Store("../../x", []byte("one"))
			return c.Load("../../x")
		}, "one", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "glutter", "shaders") // made by the first store
			var failed error
			c := New(dir, func(err error) { failed = err })
			got, ok := tt.steps(c)
			if ok != tt.found || string(got) != tt.want {
				t.Errorf("Load = %q, %v; want %q, %v", got, ok, tt.want, tt.found)
			}
			if failed != nil {
				t.Errorf("the cache reported %v", failed)
			}
			// Only the files of the cache are in the directory: no temporary file is left.
			entries, err := os.ReadDir(dir)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			for _, e := range entries {
				if filepath.Ext(e.Name()) != ".bin" {
					t.Errorf("%q is left in the cache directory", e.Name())
				}
			}
		})
	}
}

func TestStoreFailures(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	readonly := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(readonly, 0o500); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		dir   string
		skip  func() bool
		route string
		step  string
	}{
		{"the directory cannot be made", filepath.Join(blocker, "shaders"), nil, "Cache.Store.ShaderCacheWriteFailed", "mkdir"},
		{"the directory cannot be written", readonly, func() bool { return os.Geteuid() == 0 }, "Cache.Store.ShaderCacheWriteFailed", "create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != nil && tt.skip() {
				t.Skip("the user can write anywhere")
			}
			var reports []error
			c := New(tt.dir, func(err error) { reports = append(reports, err) })
			c.Store("a", []byte("one"))
			c.Store("b", []byte("two")) // fails too, and is not reported again
			if len(reports) != 1 {
				t.Fatalf("reports = %d, want 1 (the first failure only)", len(reports))
			}
			appErr, ok := errors.FromError(reports[0])
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", reports[0])
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["step"] != tt.step {
				t.Errorf("step = %v, want %q", appErr.Metadata["step"], tt.step)
			}
			if _, ok := c.Load("a"); ok {
				t.Error("a value that could not be stored was found")
			}
		})
	}
}

func TestLoadFailureIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	var reports []error
	c := New(dir, func(err error) { reports = append(reports, err) })
	c.Store("a", []byte("one"))
	// The file of the key is replaced by a directory: reading it fails with something other than "not exist".
	file := c.path("a")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(file, 0o700); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, ok := c.Load("a"); ok {
			t.Fatal("a directory was loaded")
		}
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	appErr, ok := errors.FromError(reports[0])
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", reports[0])
	}
	if !appErr.HasRoute("Cache.Load.ShaderCacheReadFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestNilReportIsFine(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := New(filepath.Join(blocker, "shaders"), nil)
	c.Store("a", []byte("one")) // fails, and nobody is told
	if _, ok := c.Load("a"); ok {
		t.Error("a value that could not be stored was found")
	}
}
