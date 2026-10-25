// Package shadercache keeps what the GPU drivers compile for Glutter between runs: the binaries of the
// OpenGL ES programs and the pipeline cache of Vulkan (SPEC §9.7). It is a directory of files; a file
// is named by the hash of its key, and is written whole or not at all. A cache that cannot be read or
// written is reported once and ignored: the drivers compile again, as they did before the cache.
//
// Flutter: no counterpart
package shadercache

import (
	"crypto/sha256"
	"encoding/hex"
	stdErrors "errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rm4n0s/errors"
)

// EnvVar is the environment variable that turns the cache off when it is "0".
//
// Flutter: no counterpart
const EnvVar = "GLUTTER_SHADER_CACHE"

// Dir is the directory of the cache: $XDG_CACHE_HOME/glutter/shaders, or ~/.cache/glutter/shaders; empty when
// the cache is turned off (EnvVar is "0") or the environment names no home (the web).
//
// Flutter: no counterpart
func Dir(getenv func(string) string) string {
	if getenv(EnvVar) == "0" {
		return ""
	}
	if base := getenv("XDG_CACHE_HOME"); base != "" {
		return filepath.Join(base, "glutter", "shaders")
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".cache", "glutter", "shaders")
	}
	return ""
}

// Cache is a directory of cached binaries. It is used from the UI thread, as the renderers are.
//
// Flutter: no counterpart
type Cache struct {
	dir      string
	report   func(error)
	reported bool
}

// New makes a cache in the directory. The directory is made when the first file is stored. report receives the
// first error of the cache (a file that cannot be read or written) and none after it; it may be nil.
//
// Flutter: no counterpart
func New(dir string, report func(error)) *Cache {
	return &Cache{dir: dir, report: report}
}

// Dir is the directory of the cache.
//
// Flutter: no counterpart
func (c *Cache) Dir() string { return c.dir }

// path is the file of a key.
func (c *Cache) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:16])+".bin")
}

func (c *Cache) fail(err error) {
	if c.reported {
		return
	}
	c.reported = true
	if c.report != nil {
		c.report(err)
	}
}

// Load returns the data stored under the key, or false when there is none (or it cannot be read).
//
// Flutter: no counterpart
func (c *Cache) Load(key string) ([]byte, bool) {
	data, err := os.ReadFile(c.path(key))
	if err != nil {
		// A directory that is not there, or a path through a file, is a miss; the Store that follows says why the
		// cache cannot be written.
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			c.fail(errors.NewErr(tagShaderCacheReadFailed, err).SetMetadata("dir", c.dir))
		}
		return nil, false
	}
	return data, true
}

// Store keeps the data under the key, replacing what was there. The file is written to a temporary file and
// renamed, so that a reader never sees half of it. A failure is reported once and ignored.
//
// Flutter: no counterpart
func (c *Cache) Store(key string, data []byte) {
	if len(data) == 0 {
		return
	}
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		c.fail(errors.NewErr(tagShaderCacheWriteFailed, err).SetMetadata("dir", c.dir, "step", "mkdir"))
		return
	}
	tmp, err := os.CreateTemp(c.dir, "tmp-*")
	if err != nil {
		c.fail(errors.NewErr(tagShaderCacheWriteFailed, err).SetMetadata("dir", c.dir, "step", "create"))
		return
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp.Name())
		c.fail(errors.NewErr(tagShaderCacheWriteFailed, stdErrors.Join(werr, cerr)).SetMetadata("dir", c.dir, "step", "write"))
		return
	}
	if err := os.Rename(tmp.Name(), c.path(key)); err != nil {
		_ = os.Remove(tmp.Name())
		c.fail(errors.NewErr(tagShaderCacheWriteFailed, err).SetMetadata("dir", c.dir, "step", "rename"))
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShaderCacheReadFailed  = "ShaderCacheReadFailed"
	tagShaderCacheWriteFailed = "ShaderCacheWriteFailed"
)
