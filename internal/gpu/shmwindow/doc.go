// Package shmwindow is the renderer of a Linux window without a GPU (SPEC §9.6): the software
// renderer (package soft) draws the frame into memory that a gpu.PixelTarget gives, and the
// window shows it, through wl_shm on Wayland. It is to Linux what gdiwindow is to Windows,
// nativewindow to Android and canvas2d to the web, with one difference that matters on a machine
// whose CPU is the rasterizer: it draws only what changed.
//
// A surface keeps the pixels of its window across frames, as offscreen.Surface does. Each frame
// clears and redraws only the rectangles that the damage tracker finds (SPEC §9.1, "Damage"), and
// then copies into the buffer the target gave the rectangles that buffer is missing: what this
// frame changed, and what the frames since the buffer was last shown changed (its age, as with
// EGL_EXT_buffer_age). The window is told only what this frame changed.
//
// The sequence of calls is plain Go over a gpu.PixelTarget, which a test replaces with a fake;
// the memory, the protocol and the release of buffers are the platform backend's.
//
// Flutter: no counterpart (Flutter's Linux embedder has a software compositor that copies the
// whole frame, shell/platform/linux/fl_compositor_software.cc)
package shmwindow
