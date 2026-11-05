package fakedevice

import (
	"image"
	"reflect"
	"strings"
)

var image2x2 = image.Rect(0, 0, 2, 2)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func reflectEqual(a, b any) bool { return reflect.DeepEqual(a, b) }
