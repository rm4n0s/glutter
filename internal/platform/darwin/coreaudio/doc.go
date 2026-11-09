// Package coreaudio is the audio backend of macOS and iOS (SPEC §29.11, §30.10): Core Audio's AudioQueue for playback and capture,
// with the list of devices and its changes from the property listeners of the system object. The design follows SDL 3's
// src/audio/coreaudio/SDL_coreaudio.m, which is where the choice of AudioQueue and the handling of the system's default device
// came from; the code here is a new Go version of them and is changed.
//
// What decides something is plain Go with no OS tag, so tables test it on any host: the stream description of a format (format.go),
// the channel count of a stream configuration and the list of devices of what the system said (list.go). The calls are in the
// files with the darwin tag, which go through a library table that the tests replace.
//
// Flutter: no counterpart
package coreaudio
