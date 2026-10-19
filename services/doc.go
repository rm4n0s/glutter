// Package services connects the framework to the platform: keyboard, text
// input, clipboard, mouse cursors and assets.
//
// Flutter: package:flutter/services.dart.
package services

//go:generate go run ../cmd/glutter-gen keys -reference ../reference/flutter -out "" -services-out keyboard_key_gen.go
