// Package fakedevice is an explicit.Device that records every call, for the unit
// tests of the explicit renderer without a GPU. It checks the design limits of
// SPEC §9.3 as the real backends do, so a renderer that breaks one fails its test
// with GPULimitExceeded.
//
// Flutter: no counterpart
package fakedevice
