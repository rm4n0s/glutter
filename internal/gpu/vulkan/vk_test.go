//go:build linux

package vulkan

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

// TestNoFunctionTakesAStructByValue checks the rule of SPEC §8.1: purego cannot pass
// a struct by value everywhere, so every function of the table takes numbers and
// pointers (to C memory) only, and returns a number.
func TestNoFunctionTakesAStructByValue(t *testing.T) {
	typ := reflect.TypeOf(vkFuncs{})
	if typ.NumField() < 60 {
		t.Fatalf("the table has only %d functions", typ.NumField())
	}
	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type.Kind() != reflect.Func {
			t.Errorf("%s is not a function", f.Name)
			continue
		}
		for j := range f.Type.NumIn() {
			switch k := f.Type.In(j).Kind(); k {
			case reflect.Struct, reflect.Array, reflect.Interface, reflect.Map, reflect.Slice, reflect.String, reflect.Pointer:
				t.Errorf("%s: parameter %d is a %s; use uintptr for a pointer to C memory", f.Name, j, k)
			}
		}
		for j := range f.Type.NumOut() {
			if k := f.Type.Out(j).Kind(); k == reflect.Struct || k == reflect.Array {
				t.Errorf("%s returns a %s by value", f.Name, k)
			}
		}
		if f.Type.NumOut() > 1 {
			t.Errorf("%s returns %d values", f.Name, f.Type.NumOut())
		}
	}
}

func TestEveryFunctionIsInTheSymbolList(t *testing.T) {
	f := &vkFuncs{}
	seen := map[any]bool{}
	for _, s := range f.symbols() {
		if seen[s.fptr] {
			t.Errorf("%v is bound twice", s.names)
		}
		seen[s.fptr] = true
		if len(s.names) == 0 || s.names[0][:2] != "vk" {
			t.Errorf("symbol names %v do not look like Vulkan functions", s.names)
		}
	}
	if got, want := len(seen), reflect.TypeOf(vkFuncs{}).NumField()-1; got != want { // getInstanceProcAddr is bound by name
		t.Errorf("%d functions are bound, the table has %d (without vkGetInstanceProcAddr)", got, want)
	}
}

func TestBindReportsAMissingSymbol(t *testing.T) {
	// Every function is "found" at a made-up address, except one. The functions are
	// bound but never called.
	missing := "vkCreateImageView"
	f := &vkFuncs{}
	err := f.bind(levelDevice, func(_ level, name string) uintptr {
		if name == missing {
			return 0
		}
		return 0x1000
	})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "SymbolMissing" || !appErr.HasRoute("vkFuncs.bind.SymbolMissing") {
		t.Fatalf("got %v, want SymbolMissing via vkFuncs.bind", err)
	}
	if appErr.Metadata["symbol"] != missing {
		t.Errorf("symbol = %v, want %s", appErr.Metadata["symbol"], missing)
	}
}

func TestBindTriesTheExtensionName(t *testing.T) {
	f := &vkFuncs{}
	var asked []string
	err := f.bind(levelDevice, func(_ level, name string) uintptr {
		asked = append(asked, name)
		if name == "vkCmdBeginRendering" || name == "vkCmdPipelineBarrier2" {
			return 0 // a Vulkan 1.1 device has the KHR names only
		}
		return 0x1000
	})
	if err != nil {
		t.Fatalf("a function with an extension name failed: %v", err)
	}
	found := map[string]bool{}
	for _, n := range asked {
		found[n] = true
	}
	if !found["vkCmdBeginRenderingKHR"] || !found["vkCmdPipelineBarrier2KHR"] {
		t.Error("the extension names were not tried")
	}
}

func TestBindSkipsAMissingOptionalFunction(t *testing.T) {
	f := &vkFuncs{}
	err := f.bind(levelGlobal, func(_ level, name string) uintptr {
		if name == "vkEnumerateInstanceVersion" {
			return 0 // a Vulkan 1.0 loader
		}
		return 0x1000
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.enumerateInstanceVersion != nil {
		t.Error("a missing optional function was bound")
	}
}

func TestBindOnlyTheLevelAskedFor(t *testing.T) {
	f := &vkFuncs{}
	if err := f.bind(levelInstance, func(lv level, name string) uintptr {
		if lv != levelInstance {
			t.Errorf("asked for %s at level %d", name, lv)
		}
		return 0x1000
	}); err != nil {
		t.Fatal(err)
	}
	if f.destroyDevice != nil || f.createInstance != nil || f.destroyInstance == nil {
		t.Error("bind touched another level")
	}
}

func TestLoadFailed(t *testing.T) {
	env := map[string]string{libraryEnvVar: "/nonexistent-glutter-dir/libvulkan.so.1"}
	_, err := loadLibrary(func(k string) string { return env[k] })
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "VulkanLoadFailed" || !appErr.HasRoute("loadLibrary.VulkanLoadFailed") {
		t.Fatalf("got %v, want VulkanLoadFailed via loadLibrary", err)
	}
	if names, _ := appErr.Metadata["names"].([]string); len(names) != 1 || names[0] != env[libraryEnvVar] {
		t.Errorf("names = %v, want the override", appErr.Metadata["names"])
	}
	// Open returns the same error before it touches anything else.
	_, err = Open(Options{Getenv: func(k string) string { return env[k] }})
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != "VulkanLoadFailed" || !appErr.HasRoute("Open->loadLibrary.VulkanLoadFailed") {
		t.Errorf("Open: got %v, want VulkanLoadFailed via Open->loadLibrary", err)
	}
}

func TestVersionString(t *testing.T) {
	for v, want := range map[uint32]string{apiVersion11: "1.1.0", apiVersion13: "1.3.0", 0x004030FD: "1.3.253"} {
		if got := versionString(v); got != want {
			t.Errorf("versionString(%#x) = %q, want %q", v, got, want)
		}
	}
}
