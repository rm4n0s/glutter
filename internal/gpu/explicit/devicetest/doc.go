// Package devicetest holds the scenes that every backend of the explicit device runs (SPEC §27.12):
// Vulkan on lavapipe, Direct3D 12 under Wine, WebGPU in Chromium and the fake device. A scene draws
// with the device and compares what it reads back with what it should be, exactly where the result is
// exact and within a tolerance where an anti-aliased edge or a rounding differs between GPUs. The
// shaders in shaders/ are compiled by `glutter-gen shaders -package devicetest`; a program that is not
// a test does not link this package.
//
// Flutter: no counterpart
package devicetest

//go:generate go run ../../../../cmd/glutter-gen shaders -dir . -package devicetest -gpu
