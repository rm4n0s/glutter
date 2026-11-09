//go:build darwin

package objcx

import (
	"strings"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// nsUTF8StringEncoding is NSUTF8StringEncoding.
const nsUTF8StringEncoding = 4

// Retain adds a reference to the object and returns it. 0 stays 0.
//
// Flutter: no counterpart
func Retain(id objc.ID) objc.ID {
	if id == 0 {
		return 0
	}
	return id.Send(Sel("retain"))
}

// Release gives up a reference to the object (the +1 of a New... function, or of a Retain). 0 does
// nothing.
//
// Flutter: no counterpart
func Release(id objc.ID) {
	if id != 0 {
		id.Send(Sel("release"))
	}
}

// Autorelease hands the reference that the caller owns to the current autorelease pool and returns the
// object: it is released when the pool drains (WithPool). 0 stays 0.
//
// Flutter: no counterpart
func Autorelease(id objc.ID) objc.ID {
	if id == 0 {
		return 0
	}
	return id.Send(Sel("autorelease"))
}

// RetainCount is the retain count of the object, for the memory tests that look for a leak.
//
// Flutter: no counterpart
func RetainCount(id objc.ID) uint64 {
	if id == 0 {
		return 0
	}
	return objc.Send[uint64](id, Sel("retainCount"))
}

// NewString makes an NSString from a Go string (+1: release it). Bytes that are not valid UTF-8 become
// U+FFFD, because NSString refuses them. An embedded NUL is kept.
//
// Flutter: no counterpart
func NewString(s string) objc.ID {
	foundation()
	s = strings.ToValidUTF8(s, "�")
	var data unsafe.Pointer
	if len(s) > 0 {
		data = unsafe.Pointer(unsafe.StringData(s))
	}
	str := objc.ID(MustClass("NSString")).Send(Sel("alloc"))
	return str.Send(Sel("initWithBytes:length:encoding:"), data, uint64(len(s)), uint64(nsUTF8StringEncoding))
}

// String copies an NSString into a Go string. 0 gives "". An embedded NUL is kept.
//
// Flutter: no counterpart
func String(id objc.ID) string {
	if id == 0 {
		return ""
	}
	n := objc.Send[uint64](id, Sel("lengthOfBytesUsingEncoding:"), uint64(nsUTF8StringEncoding))
	if n == 0 {
		return ""
	}
	p := objc.Send[uintptr](id, Sel("UTF8String"))
	return string(unsafe.Slice((*byte)(dl.Pointer(p)), n))
}

// NewNumberInt makes an NSNumber from an integer (+1: release it).
//
// Flutter: no counterpart
func NewNumberInt(v int64) objc.ID {
	foundation()
	return objc.ID(MustClass("NSNumber")).Send(Sel("alloc")).Send(Sel("initWithLongLong:"), v)
}

// NewNumberFloat makes an NSNumber from a float (+1: release it).
//
// Flutter: no counterpart
func NewNumberFloat(v float64) objc.ID {
	foundation()
	return objc.ID(MustClass("NSNumber")).Send(Sel("alloc")).Send(Sel("initWithDouble:"), v)
}

// NewNumberBool makes an NSNumber from a bool (+1: release it).
//
// Flutter: no counterpart
func NewNumberBool(v bool) objc.ID {
	foundation()
	var b int8
	if v {
		b = 1
	}
	return objc.ID(MustClass("NSNumber")).Send(Sel("alloc")).Send(Sel("initWithBool:"), b)
}

// Int is the integer value of an NSNumber.
//
// Flutter: no counterpart
func Int(id objc.ID) int64 { return objc.Send[int64](id, Sel("longLongValue")) }

// Float is the floating point value of an NSNumber.
//
// Flutter: no counterpart
func Float(id objc.ID) float64 { return objc.Send[float64](id, Sel("doubleValue")) }

// Bool is the boolean value of an NSNumber.
//
// Flutter: no counterpart
func Bool(id objc.ID) bool { return objc.Send[int8](id, Sel("boolValue")) != 0 }

// NewArray makes an NSArray of the objects (+1: release it; the array retains its elements).
//
// Flutter: no counterpart
func NewArray(items ...objc.ID) objc.ID {
	foundation()
	var data unsafe.Pointer
	if len(items) > 0 {
		data = unsafe.Pointer(&items[0])
	}
	arr := objc.ID(MustClass("NSArray")).Send(Sel("alloc"))
	return arr.Send(Sel("initWithObjects:count:"), data, uint64(len(items)))
}

// ArrayCount is the number of elements of an NSArray.
//
// Flutter: no counterpart
func ArrayCount(id objc.ID) int { return int(objc.Send[uint64](id, Sel("count"))) }

// ArrayAt is the element of an NSArray at index i (not owned by the caller). An index out of range
// raises an Objective-C exception, which ends the program: check ArrayCount.
//
// Flutter: no counterpart
func ArrayAt(id objc.ID, i int) objc.ID { return id.Send(Sel("objectAtIndex:"), uint64(i)) }

// NewFileURL makes a file NSURL for the path (+1: release it).
//
// Flutter: no counterpart
func NewFileURL(path string) objc.ID {
	s := NewString(path)
	defer Release(s)
	return objc.ID(MustClass("NSURL")).Send(Sel("alloc")).Send(Sel("initFileURLWithPath:"), s)
}

// NewURL makes an NSURL from a URL string (+1: release it), or returns 0 when the string is not a
// valid URL.
//
// Flutter: no counterpart
func NewURL(raw string) objc.ID {
	s := NewString(raw)
	defer Release(s)
	return objc.ID(MustClass("NSURL")).Send(Sel("alloc")).Send(Sel("initWithString:"), s)
}

// URLString is the absolute string of an NSURL.
//
// Flutter: no counterpart
func URLString(id objc.ID) string { return String(id.Send(Sel("absoluteString"))) }

// URLPath is the path of an NSURL (a file URL's path on disk).
//
// Flutter: no counterpart
func URLPath(id objc.ID) string { return String(id.Send(Sel("path"))) }

// Error turns the NSError that a call gave back (the NSError** argument that it filled) into a tagged
// error, or returns nil for 0. The error has the tag NSError, the NSError's localized description as its
// message, and the metadata "op" (what was being done), "domain" and "code".
//
// Flutter: no counterpart
func Error(nserror objc.ID, op string) error {
	if nserror == 0 {
		return nil
	}
	return errors.New(tagNSError, String(nserror.Send(Sel("localizedDescription"))),
		"op", op,
		"domain", String(nserror.Send(Sel("domain"))),
		"code", objc.Send[int64](nserror, Sel("code")))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNSError = "NSError"
)
