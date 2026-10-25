// Package gdiwindow is the last-resort renderer of Windows (SPEC §25.5): the software renderer
// (package soft) draws the frame, and a Presenter puts the image into the window with GDI,
// StretchDIBits at 1:1 from a top-down 32-bit DIB, into the window's device context, one call for
// each damaged rectangle. It is to Windows what canvas2d is to the web and nativewindow to
// Android.
//
// The presenter keeps the last frame, so a WM_PAINT after an exposure (the window was uncovered)
// paints it again with BeginPaint and no new frame is needed. The sequence of calls is plain Go
// over a windowAPI, which a test replaces with a fake; the calls to Windows are in the file that
// carries the windows tag.
//
// Flutter: no counterpart (Flutter's Windows embedder has a software compositor that
// presents with StretchDIBits, shell/platform/windows/compositor_software.cc)
package gdiwindow
