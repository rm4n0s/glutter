//go:build darwin

package objcx

import (
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// classSeq makes the names of the classes of the tests unique, so that the tests can run more than once in a process.
var classSeq atomic.Int64

func uniqueName(prefix string) string { return fmt.Sprintf("%s%d", prefix, classSeq.Add(1)) }

func TestSel(t *testing.T) {
	a, b := Sel("glutterTestSelector:"), Sel("glutterTestSelector:")
	if a == 0 || a != b {
		t.Errorf("Sel gave %v and %v for the same name", a, b)
	}
	if Sel("glutterTestOther") == a {
		t.Error("two names gave the same selector")
	}
}

func TestClass(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if c, err := Class("NSString"); err != nil || c == 0 {
		t.Errorf("Class(NSString) = %v, %v", c, err)
	}
	_, err := Class("GlutterNoSuchClass")
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Class."+tagMacOSClassMissing) {
		t.Fatalf("got %v, want MacOSClassMissing via Class", err)
	}
	if appErr.Metadata["class"] != "GlutterNoSuchClass" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestStrings(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // what comes back; "" means the same as in
	}{
		{"ascii", "hello", ""},
		{"empty", "", ""},
		{"the basic plane", "héllo ☃ 日本語", ""},
		{"a surrogate pair in UTF-16", "a😀b 𝄞", ""},
		{"an embedded NUL", "ab\x00cd", ""},
		{"invalid UTF-8 becomes U+FFFD", "a\xffb", "a�b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if want == "" {
				want = tt.in
			}
			WithPool(func() {
				s := NewString(tt.in)
				defer Release(s)
				if s == 0 {
					t.Fatal("NewString gave nil")
				}
				if got := String(s); got != want {
					t.Errorf("String(NewString(%q)) = %q, want %q", tt.in, got, want)
				}
			})
		})
	}
	if got := String(0); got != "" {
		t.Errorf("String(0) = %q", got)
	}
}

func TestStringLengthInUTF16(t *testing.T) {
	// The length of an NSString counts UTF-16 units: a character outside the basic plane is two.
	tests := []struct {
		in   string
		want uint64
	}{{"abc", 3}, {"☃", 1}, {"😀", 2}, {"a😀b", 4}}
	for _, tt := range tests {
		WithPool(func() {
			s := NewString(tt.in)
			defer Release(s)
			if got := objc.Send[uint64](s, Sel("length")); got != tt.want {
				t.Errorf("length of %q = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestNumbers(t *testing.T) {
	WithPool(func() {
		i := NewNumberInt(-1 << 40)
		f := NewNumberFloat(2.5)
		b := NewNumberBool(true)
		defer Release(i)
		defer Release(f)
		defer Release(b)
		if got := Int(i); got != -1<<40 {
			t.Errorf("Int = %d", got)
		}
		if got := Float(f); got != 2.5 {
			t.Errorf("Float = %v", got)
		}
		if !Bool(b) {
			t.Error("Bool = false")
		}
	})
}

func TestArray(t *testing.T) {
	tests := []struct {
		name string
		strs []string
	}{{"empty", nil}, {"one", []string{"a"}}, {"several", []string{"x", "yy", "zzz"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WithPool(func() {
				var items []objc.ID
				for _, s := range tt.strs {
					items = append(items, NewString(s))
				}
				arr := NewArray(items...)
				for _, it := range items {
					Release(it) // the array has its own references
				}
				defer Release(arr)
				if got := ArrayCount(arr); got != len(tt.strs) {
					t.Fatalf("ArrayCount = %d, want %d", got, len(tt.strs))
				}
				for i, want := range tt.strs {
					if got := String(ArrayAt(arr, i)); got != want {
						t.Errorf("element %d = %q, want %q", i, got, want)
					}
				}
			})
		})
	}
}

func TestURLs(t *testing.T) {
	WithPool(func() {
		f := NewFileURL("/tmp/a b.txt")
		defer Release(f)
		if got := URLPath(f); got != "/tmp/a b.txt" {
			t.Errorf("URLPath = %q", got)
		}
		if got := URLString(f); got != "file:///tmp/a%20b.txt" {
			t.Errorf("URLString = %q", got)
		}
		u := NewURL("https://example.com/x?y=1")
		defer Release(u)
		if got := URLString(u); got != "https://example.com/x?y=1" {
			t.Errorf("URLString = %q", got)
		}
		if bad := NewURL("http://exa mple.com"); bad != 0 {
			Release(bad)
			t.Error("NewURL of an invalid URL did not give nil")
		}
	})
}

func TestError(t *testing.T) {
	if err := Error(0, "nothing"); err != nil {
		t.Errorf("Error(0) = %v, want nil", err)
	}
	WithPool(func() {
		domain := NewString("GlutterTestDomain")
		defer Release(domain)
		nserror := objc.ID(MustClass("NSError")).Send(Sel("errorWithDomain:code:userInfo:"), domain, int64(42), 0)
		err := Error(nserror, "open the thing")
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagNSError {
			t.Fatalf("got %v, want an error tagged NSError", err)
		}
		if !appErr.HasRoute("Error." + tagNSError) {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		if appErr.Metadata["op"] != "open the thing" || appErr.Metadata["domain"] != "GlutterTestDomain" || appErr.Metadata["code"] != int64(42) {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
		if appErr.Message == "" {
			t.Error("the message is empty; it should be the localized description")
		}
	})
}

func TestWithPoolDrainsAutoreleasedObjects(t *testing.T) {
	var deallocs atomic.Int64
	cls, err := RegisterClass(ClassDef{Name: uniqueName("PoolProbe"), Super: "NSObject", Methods: []Method{
		{"dealloc", func(self objc.ID, _ objc.SEL) {
			deallocs.Add(1)
			self.SendSuper(Sel("dealloc"))
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	WithPool(func() {
		obj := Autorelease(cls.New(0))
		if obj == 0 {
			t.Fatal("no object")
		}
		if got := deallocs.Load(); got != 0 {
			t.Errorf("%d objects freed before the pool drained, want 0", got)
		}
	})
	if got := deallocs.Load(); got != 1 {
		t.Errorf("%d objects freed after the pool drained, want 1", got)
	}
	// An object that is released and not autoreleased is freed at once.
	obj := cls.New(0)
	Release(obj)
	if got := deallocs.Load(); got != 2 {
		t.Errorf("%d objects freed after Release, want 2", got)
	}
}

func TestWithPoolDrainsWhenFnPanics(t *testing.T) {
	var deallocs atomic.Int64
	cls, err := RegisterClass(ClassDef{Name: uniqueName("PanicProbe"), Super: "NSObject", Methods: []Method{
		{"dealloc", func(self objc.ID, _ objc.SEL) {
			deallocs.Add(1)
			self.SendSuper(Sel("dealloc"))
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() { _ = recover() }()
		WithPool(func() {
			Autorelease(cls.New(0))
			panic("boom")
		})
	}()
	if got := deallocs.Load(); got != 1 {
		t.Errorf("%d objects freed after a panic, want 1", got)
	}
}

func TestRegisterClass(t *testing.T) {
	name := uniqueName("Probe")
	def := ClassDef{Name: name, Super: "NSObject", Methods: []Method{
		{"answer", func(self objc.ID, _ objc.SEL) int64 { return 42 }},
		{"handleTimes:", func(self objc.ID, _ objc.SEL, n int64) int64 { return int64(HandleOf(self)) * n }},
	}}
	before := dl.CallbackCount()
	cls, err := RegisterClass(def)
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("Glutter%s_%d", name, osPid()); cls.Name != want {
		t.Errorf("class name = %q, want %q", cls.Name, want)
	}
	if got, want := dl.CallbackCount()-before, len(def.Methods)+2; got != want {
		t.Errorf("the class counted %d callbacks, want %d (methods, and the getter and setter of the handle)", got, want)
	}
	h := NewHandle("the go object")
	defer DropHandle(h)
	obj := cls.New(h)
	defer Release(obj)
	if got := objc.Send[int64](obj, Sel("answer")); got != 42 {
		t.Errorf("answer = %d, want 42", got)
	}
	if got := HandleOf(obj); got != h {
		t.Errorf("HandleOf = %d, want %d", got, h)
	}
	if got := objc.Send[int64](obj, Sel("handleTimes:"), int64(3)); got != int64(h)*3 {
		t.Errorf("handleTimes: = %d, want %d", got, int64(h)*3)
	}
	if v, ok := LookupAs[string](HandleOf(obj)); !ok || v != "the go object" {
		t.Errorf("LookupAs = %q, %v", v, ok)
	}
}

func TestRegisterClassErrors(t *testing.T) {
	name := uniqueName("Twice")
	if _, err := RegisterClass(ClassDef{Name: name, Super: "NSObject"}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		def       ClassDef
		wantRoute string
		wantKey   string // a metadata key that must be there
	}{
		{"the same name twice", ClassDef{Name: name, Super: "NSObject"}, "RegisterClass." + tagMacOSClassRegistrationFailed, "class"},
		{"an unknown superclass", ClassDef{Name: uniqueName("NoSuper"), Super: "GlutterNoSuchSuper"}, "RegisterClass->Class." + tagMacOSClassMissing, "class"},
		{"an unknown protocol", ClassDef{Name: uniqueName("NoProto"), Super: "NSObject", Protocols: []string{"GlutterNoSuchProtocol"}}, "RegisterClass." + tagMacOSClassRegistrationFailed, "protocol"},
		{"a method that is not a function", ClassDef{Name: uniqueName("BadMethod"), Super: "NSObject", Methods: []Method{{"x", 7}}}, "RegisterClass." + tagMacOSClassRegistrationFailed, "class"},
		{"a method without the object and the selector", ClassDef{Name: uniqueName("BadArgs"), Super: "NSObject", Methods: []Method{{"y", func() {}}}}, "RegisterClass." + tagMacOSClassRegistrationFailed, "class"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RegisterClass(tt.def)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), tt.wantRoute)
			}
			if _, has := appErr.Metadata[tt.wantKey]; !has {
				t.Errorf("metadata %v has no %q", appErr.Metadata, tt.wantKey)
			}
		})
	}
}

func TestHandles(t *testing.T) {
	before := HandleCount()
	type thing struct{ n int }
	a, b := NewHandle(&thing{1}), NewHandle("text")
	if a == 0 || b == 0 || a == b {
		t.Fatalf("handles %d and %d", a, b)
	}
	if got := HandleCount() - before; got != 2 {
		t.Errorf("HandleCount grew by %d, want 2", got)
	}
	if th, ok := LookupAs[*thing](a); !ok || th.n != 1 {
		t.Errorf("LookupAs[*thing] = %v, %v", th, ok)
	}
	if _, ok := LookupAs[*thing](b); ok {
		t.Error("LookupAs found a string as a *thing")
	}
	DropHandle(a)
	if _, ok := Lookup(a); ok {
		t.Error("a handle that was dropped is still found")
	}
	DropHandle(a) // does nothing
	if _, ok := Lookup(0); ok {
		t.Error("handle 0 was found")
	}
	DropHandle(b)
	if got := HandleCount() - before; got != 0 {
		t.Errorf("HandleCount is %d above where it began after both were dropped", got)
	}
}

func TestBlocks(t *testing.T) {
	WithPool(func() {
		var items []objc.ID
		for _, s := range []string{"a", "b", "c", "d"} {
			items = append(items, NewString(s))
		}
		arr := NewArray(items...)
		defer Release(arr)
		for _, it := range items {
			Release(it)
		}
		var seen []string
		var sumIdx uint64
		blk := NewBlock(func(_ objc.Block, obj objc.ID, idx uint64, stop *bool) {
			seen = append(seen, String(obj))
			sumIdx += idx
			if idx == 2 {
				*stop = true
			}
		})
		defer blk.Release()
		arr.Send(Sel("enumerateObjectsUsingBlock:"), blk)
		if got := fmt.Sprint(seen); got != "[a b c]" {
			t.Errorf("the block saw %s, want [a b c] (it stopped at index 2)", got)
		}
		if sumIdx != 3 {
			t.Errorf("the sum of the indexes = %d, want 3", sumIdx)
		}
	})
}

func TestBlocksOfOneTypeCountOneCallback(t *testing.T) {
	type blockFn = func(objc.Block, int64) int64
	// The first block of a function type counts a callback; the next ones do not. With -count=2 the type
	// was counted by the first run, so no callback is expected then.
	wantFirst := 1
	if _, seen := blockTypes.Load(reflect.TypeFor[blockFn]()); seen {
		wantFirst = 0
	}
	before := dl.CallbackCount()
	first := NewBlock(blockFn(func(objc.Block, int64) int64 { return 1 }))
	afterFirst := dl.CallbackCount()
	second := NewBlock(blockFn(func(objc.Block, int64) int64 { return 2 }))
	third := NewBlock(blockFn(func(objc.Block, int64) int64 { return 3 }))
	for _, b := range []objc.Block{first, second, third} {
		defer b.Release()
	}
	if got := afterFirst - before; got != wantFirst {
		t.Errorf("the first block counted %d callbacks, want %d", got, wantFirst)
	}
	if got := dl.CallbackCount() - afterFirst; got != 0 {
		t.Errorf("two more blocks of the same type counted %d callbacks, want 0", got)
	}
}

func TestCallbackCountStaysUnderTheLimit(t *testing.T) {
	// SPEC §8.1 rule 2, §29.2: a process can have about 2,000 callbacks, and this package's classes and blocks count.
	if got := dl.CallbackCount(); got > dl.MaxCallbacks {
		t.Errorf("%d callbacks counted, over the limit of %d", got, dl.MaxCallbacks)
	}
}

func osPid() int { return os.Getpid() }
