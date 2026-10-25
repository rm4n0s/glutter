package d3d12

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
)

// dxbcOf makes data with a DXBC header and the given total size field.
func dxbcOf(n int, size uint32) []byte {
	b := make([]byte, n)
	copy(b, "DXBC")
	if n >= dxbcHeaderSize {
		binary.LittleEndian.PutUint32(b[24:], size)
	}
	return b
}

func TestCacheKeyChangesWithEachPart(t *testing.T) {
	key := func() string {
		return CacheKey("float4 f() { return 0; }", "f", TargetPixel, CompileOptimizationLevel3, "10.0.19041.1")
	}
	base, again := key(), key()
	tests := []struct {
		name string
		key  string
	}{
		{"the HLSL", CacheKey("float4 f() { return 1; }", "f", TargetPixel, CompileOptimizationLevel3, "10.0.19041.1")},
		{"the entry point", CacheKey("float4 f() { return 0; }", "g", TargetPixel, CompileOptimizationLevel3, "10.0.19041.1")},
		{"the target", CacheKey("float4 f() { return 0; }", "f", TargetVertex, CompileOptimizationLevel3, "10.0.19041.1")},
		{"the flags", CacheKey("float4 f() { return 0; }", "f", TargetPixel, CompileDebug|CompileSkipOptimization, "10.0.19041.1")},
		{"the version of the compiler", CacheKey("float4 f() { return 0; }", "f", TargetPixel, CompileOptimizationLevel3, "10.0.19041.2")},
		{"parts that run together", CacheKey("float4 f() { return 0; }f", "", TargetPixel, CompileOptimizationLevel3, "10.0.19041.1")},
	}
	if base != again {
		t.Fatal("the same parts gave two keys")
	}
	seen := map[string]string{base: "the base"}
	for _, tt := range tests {
		if other, ok := seen[tt.key]; ok {
			t.Errorf("a change of %s gives the key of %s", tt.name, other)
		}
		seen[tt.key] = tt.name
	}
}

func TestValidDXBC(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"a container", dxbcOf(100, 100), true},
		{"only the header", dxbcOf(dxbcHeaderSize, dxbcHeaderSize), true},
		{"nothing", nil, false},
		{"cut inside the header", dxbcOf(20, 20), false},
		{"another magic", append([]byte("DXIL"), dxbcOf(100, 100)[4:]...), false},
		{"a size field that is not the length", dxbcOf(100, 90), false},
		{"cut short", dxbcOf(100, 100)[:60], false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidDXBC(tt.data); got != tt.want {
				t.Errorf("ValidDXBC = %v, want %v", got, tt.want)
			}
		})
	}
}

// The cache over a directory: a miss, a hit, a file with a bad header, and a directory that cannot
// be written (SPEC §9.7).
func TestCompilerOverADirectory(t *testing.T) {
	good := dxbcOf(64, 64)
	t.Run("miss then hit", func(t *testing.T) {
		be := &fakeBackend{version: "1.0", dxbc: good}
		c := NewCompiler(be, shadercache.New(t.TempDir(), nil))
		first, err := c.Compile("hlsl", "main", TargetPixel, 7)
		if err != nil || first.FromCache {
			t.Fatalf("first = %+v, %v; want a compile", first, err)
		}
		second, err := c.Compile("hlsl", "main", TargetPixel, 7)
		if err != nil || !second.FromCache || string(second.DXBC) != string(good) {
			t.Fatalf("second = %+v, %v; want the bytecode from the cache", second, err)
		}
		if be.calls != 1 {
			t.Errorf("D3DCompile ran %d times, want 1", be.calls)
		}
		if st := c.Stats(); st != (CompilerStats{Compiled: 1, FromCache: 1}) {
			t.Errorf("stats %+v", st)
		}
	})
	t.Run("a second process reads the files of the first", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := NewCompiler(&fakeBackend{version: "1.0", dxbc: good}, shadercache.New(dir, nil)).Compile("hlsl", "main", TargetPixel, 7); err != nil {
			t.Fatal(err)
		}
		be := &fakeBackend{version: "1.0", dxbc: good}
		res, err := NewCompiler(be, shadercache.New(dir, nil)).Compile("hlsl", "main", TargetPixel, 7)
		if err != nil || !res.FromCache || be.calls != 0 {
			t.Errorf("res = %+v, %v, calls %d; want a hit", res, err, be.calls)
		}
	})
	t.Run("a changed compiler version is a miss", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := NewCompiler(&fakeBackend{version: "1.0", dxbc: good}, shadercache.New(dir, nil)).Compile("hlsl", "main", TargetPixel, 7); err != nil {
			t.Fatal(err)
		}
		be := &fakeBackend{version: "1.1", dxbc: good}
		res, err := NewCompiler(be, shadercache.New(dir, nil)).Compile("hlsl", "main", TargetPixel, 7)
		if err != nil || res.FromCache || be.calls != 1 {
			t.Errorf("res = %+v, %v, calls %d; want a compile", res, err, be.calls)
		}
	})
	t.Run("a file with a bad header is dropped and replaced", func(t *testing.T) {
		dir := t.TempDir()
		cache := shadercache.New(dir, nil)
		key := CacheKey("hlsl", "main", TargetPixel, 7, "1.0")
		cache.Store(key, []byte("this is not bytecode"))
		be := &fakeBackend{version: "1.0", dxbc: good}
		c := NewCompiler(be, cache)
		res, err := c.Compile("hlsl", "main", TargetPixel, 7)
		if err != nil || res.FromCache || be.calls != 1 {
			t.Fatalf("res = %+v, %v, calls %d; want a compile", res, err, be.calls)
		}
		if st := c.Stats(); st.Refused != 1 {
			t.Errorf("stats %+v, want one refused file", st)
		}
		if data, ok := cache.Load(key); !ok || string(data) != string(good) {
			t.Errorf("the file was not replaced by the new bytecode: %q", data)
		}
	})
	t.Run("a directory that cannot be written is reported once and ignored", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		reports := 0
		cache := shadercache.New(filepath.Join(file, "sub"), func(error) { reports++ }) // a path through a file
		be := &fakeBackend{version: "1.0", dxbc: good}
		c := NewCompiler(be, cache)
		for range 2 {
			res, err := c.Compile("hlsl", "main", TargetPixel, 7)
			if err != nil || string(res.DXBC) != string(good) {
				t.Fatalf("res = %+v, %v; want the bytecode though the cache is unwritable", res, err)
			}
		}
		if be.calls != 2 || reports != 1 {
			t.Errorf("calls %d, reports %d; want 2 compiles and one report", be.calls, reports)
		}
	})
}

func TestNoCacheIsNotAnError(t *testing.T) {
	tests := []struct {
		name    string
		version string
		cache   BlobCache
	}{
		{"no cache directory", "1.0", nil},
		{"a compiler without a version", "", &memCache{files: map[string][]byte{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := &fakeBackend{version: tt.version, dxbc: dxbcOf(64, 64)}
			c := NewCompiler(be, tt.cache)
			for range 2 {
				res, err := c.Compile("hlsl", "main", TargetPixel, 7)
				if err != nil || res.FromCache {
					t.Fatalf("res = %+v, %v", res, err)
				}
			}
			if be.calls != 2 {
				t.Errorf("D3DCompile ran %d times, want 2", be.calls)
			}
			if m, ok := tt.cache.(*memCache); ok && len(m.files) != 0 {
				t.Errorf("a compiler without a version cached %d files", len(m.files))
			}
		})
	}
}

func TestCompileFlags(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want uint32
	}{
		{"a release build", nil, CompileOptimizationLevel3},
		{"the debug variable at 1", map[string]string{DebugEnvVar: "1"}, CompileDebug | CompileSkipOptimization},
		{"the debug variable at 0", map[string]string{DebugEnvVar: "0"}, CompileOptimizationLevel3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompileFlags(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("flags = %#x, want %#x", got, tt.want)
			}
		})
	}
}
