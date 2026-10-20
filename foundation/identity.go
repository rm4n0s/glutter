// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"fmt"
	"hash/maphash"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/glutter/internal/dartfmt"
)

// identitySeed seeds the hash of values that are not pointers. It is set once
// at start-up and never changes.
var identitySeed = maphash.MakeSeed()

// identityHash returns a hash that tells objects apart: the address of a
// pointer, map, slice, channel or function, and the hash of the value
// otherwise. A nil value hashes to 0.
func identityHash(v any) uint64 {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	var h uint64
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Slice:
		h = uint64(rv.Pointer())
	default:
		if !rv.Comparable() {
			return 0
		}
		h = maphash.Comparable(identitySeed, v)
	}
	// Addresses are aligned, so the low bits say little; mix them.
	h *= 0x9e3779b97f4a7c15
	return h >> 32
}

// ShortHash returns the first five hex digits of the object's identity hash,
// which tell objects apart in debug output.
//
// Flutter: foundation/diagnostics.dart shortHash
func ShortHash(object any) string {
	return fmt.Sprintf("%05x", identityHash(object)&0xfffff)
}

// DescribeIdentity returns the object's type name and ShortHash, such as
// "ValueNotifier<int>#1a2b3". A nil object is "Null#00000".
//
// Flutter: foundation/diagnostics.dart describeIdentity
func DescribeIdentity(object any) string {
	return RuntimeTypeName(object) + "#" + ShortHash(object)
}

var (
	pathPrefix    = regexp.MustCompile(`[\w.\-]+/`)
	pkgQualifier  = regexp.MustCompile(`\b[a-z_][a-z0-9_]*\.([A-Z])`)
	genericBraces = strings.NewReplacer("[", "<", "]", ">")
)

// RuntimeTypeName returns the name of the object's type the way Dart prints a
// runtimeType: no package, no pointer star, and angle brackets around type
// arguments ("ValueNotifier<int>"). A nil object is "Null".
//
// Flutter: foundation/object.dart objectRuntimeType
func RuntimeTypeName(object any) string {
	if object == nil {
		return "Null"
	}
	t := reflect.TypeOf(object)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	if name == "" {
		name = t.String()
	}
	name = pathPrefix.ReplaceAllString(name, "")
	name = pkgQualifier.ReplaceAllString(name, "$1")
	return genericBraces.Replace(name)
}

// describeValue returns what Dart's toString gives the value: "null" for nil,
// the String method when there is one, Dart's double format for floats and
// bracketed lists and braced maps.
func describeValue(v any) string {
	if v == nil {
		return "null"
	}
	if s, ok := v.(fmt.Stringer); ok {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer || !rv.IsNil() {
			return s.String()
		}
		return "null"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return dartfmt.Double(rv.Float())
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan:
		if rv.IsNil() {
			return "null"
		}
		if rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct {
			return "Instance of '" + RuntimeTypeName(v) + "'"
		}
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return "null"
		}
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = describeValue(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Map:
		if rv.IsNil() {
			return "null"
		}
		parts := make([]string, 0, rv.Len())
		it := rv.MapRange()
		for it.Next() {
			parts = append(parts, describeValue(it.Key().Interface())+": "+describeValue(it.Value().Interface()))
		}
		sort.Strings(parts)
		return "{" + strings.Join(parts, ", ") + "}"
	case reflect.Struct:
		return "Instance of '" + RuntimeTypeName(v) + "'"
	}
	return fmt.Sprint(v)
}

// isNil reports whether v is nil or holds a nil pointer, map, slice, function,
// channel or interface: Dart's null.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

// ValuesEqual is Dart's `==` for two values: Go equality for values that can be
// compared, and identity for slices, maps and functions. A type with its own
// notion of equality (a pointer to a struct, say) needs a function of its own.
//
// Flutter: no counterpart (the == operator)
func ValuesEqual(a, b any) bool { return equalValues(a, b) }

// equalValues is Dart's `==` for two values: Go equality for values that can
// be compared, and identity for slices, maps and functions (which Dart
// compares by identity too).
func equalValues(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	if ta.Comparable() {
		return a == b
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	switch ta.Kind() {
	case reflect.Slice:
		return ra.IsNil() == rb.IsNil() && ra.Len() == rb.Len() && (ra.Len() == 0 || ra.Pointer() == rb.Pointer())
	case reflect.Map, reflect.Func:
		return ra.Pointer() == rb.Pointer()
	}
	return reflect.DeepEqual(a, b)
}

// joinComma joins strings with ", ", as Dart's Iterable.join(', ') does.
func joinComma(parts []string) string { return strings.Join(parts, ", ") }

// itoa formats an int.
func itoa(i int) string { return strconv.Itoa(i) }
