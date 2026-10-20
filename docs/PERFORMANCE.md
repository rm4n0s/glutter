# Performance

This page records how the targets of SPEC §15 are measured, the numbers on the machine
they were last measured on, and the targets that are not met there.

## Machine

| | |
|---|---|
| CPU | AMD Ryzen AI Max+ 395 (32 threads) |
| Memory | 32 GB |
| OS | Linux 7.3, Fedora 44 |
| Graphics | Mesa **llvmpipe** (software OpenGL ES) under a headless Sway at 1920×1080, scale 1.25. There is no GPU |
| Date | 2026-10-21 |

SPEC §15 names a 2020 laptop with an Iris Xe. This machine is faster on the CPU side and
has no GPU, so: the numbers of the UI thread (build, layout, paint, allocations) are
usable, the raster time is **not** (the software renderer runs on the CPU, in the
compositor's process and in ours), and the cold start and the resident memory include the
software driver. The numbers must be taken again on the reference hardware before 1.0
(see "Pending").

The targets are for release builds (`-tags glutter_release`). Debug builds check the
thread of every call and the invariants of the framework, and are 8–10 times slower on
the frames of the framework (most of it `UIThread.AssertCurrent`).

## How to measure

```
# the framework alone (UI thread, no display)
go test -tags glutter_release ./widgets ./rendering ./examples/gallery ./internal/gpu/soft -run xxx -bench . -benchmem

# the numbers that need a running app: idle CPU, cold start, memory, the gallery's frames
scripts/perf.sh -n 7 -s 10
```

`GLUTTER_FRAME_STATS=1` makes a running app log its frame times (and the stages of the
start) once a second; `perf.sh` reads those lines. The performance overlay
(`ShowPerformanceOverlay`) shows the same numbers on screen.

| Benchmark | Measures |
|---|---|
| `widgets` `BenchmarkAnimationFrame` | a frame in which one animation changes a `RenderObject` (no build, no paint of the page) |
| `widgets` `BenchmarkAnimationFramePaint` | the same, with a `CustomPaint` that repaints (9 draw ops) |
| `widgets` `BenchmarkFullRebuildFrame` | a frame that rebuilds every element of a 1033-element screen |
| `widgets` `BenchmarkIdleFrame` | a pump with nothing scheduled |
| `rendering` `BenchmarkLayoutFrame`, `BenchmarkPaintFrame`, `BenchmarkIdleFrame` | layout and paint of a render tree, and the cost of nothing |
| `examples/gallery` `BenchmarkGalleryAnimationFrame` | the data page of the gallery with its indeterminate indicators (3465 elements) |
| `examples/gallery` `BenchmarkGalleryFullRebuildFrame` | every element of the gallery page (4889 elements) built again |
| `internal/gpu/soft` `BenchmarkRenderScene/<scene>` | the software renderer on the proof scenes (not a GPU number) |

The benchmarks drop the scenes (`DiscardScenes`), so they measure the framework and not
the renderer.

## Numbers

| Metric (SPEC §15) | Target | Measured | |
|---|---|---|---|
| Idle CPU | ~0% | 0.00%: no frame logged in 10 s | met |
| Frame rate, gallery data page with indeterminate indicators | the display's refresh rate, no dropped frames | 62 frames per second on a 60 Hz output, 0 frames over 16.7 ms | met |
| UI thread per frame, animation only, no rebuild | < 2 ms | 1.3 µs (one animated render object), 90.5 µs (gallery page, indicators repainting); the build time the app logs is 0.68 ms on average and 2.16 ms at worst | met |
| UI thread per frame, full rebuild of the gallery page | < 8 ms | 4.4 ms for 4889 elements (1.5 ms for 1033) | met |
| Allocations, animation-only frame | ≤ 1 KB | 932 B, 26 allocs (no repaint); 4.3 KB, 47 allocs (9-op repaint); 28.6 KB, 463 allocs (gallery page) | **met only without a repaint** |
| Cold start to first frame | < 150 ms | 300 ms from the start of the process (293 ms from the start of `Run`), median of 7 | **not met** |
| Resident memory, counter | < 60 MB | 287 MB; Go runtime 41 MB from the system, 24 MB heap in use | **not met** |

The raster time of the gallery on this machine is 9.2 ms on average and 12.1 ms at worst,
on llvmpipe; it is not a SPEC target.

## What was changed for the numbers

Before this step the full rebuild of the gallery was 14.6 ms. The profile was dominated by
`Theme.of`, which made a localized copy of the whole theme (and of its two text themes,
every style of them) on each call. Now:

- `ThemeOf` keeps the localized copy in the `ThemeData` it was made from, one per script
  category (SPEC §12.10: Flutter keeps five in a static). 14.6 ms → 4.4 ms; a rebuild
  of the gallery page now allocates 3.1 MB.
- The shaders are compiled when a frame first needs them, not all at the start
  (`Device.CompileAll` compiles all of them, for the tests).
- `Guard` of a build (the error handler of `ErrorWidget`) no longer makes the build
  context's description at every call (`GuardLazy`).
- The binding runs the persistent frame callbacks without copying the list.

## Exceptions that need the user's approval

These are listed because the "Done when" of step 17.4 asks for it. The user was not
available during this work, so they are **pending approval**.

1. **Cold start, 300 ms against 150 ms.** About 185 ms of it is the opening of the EGL
   display and the creation of the context in Mesa's software driver; the framework's own
   part (fonts, the first build and layout, the first paint) is about 100 ms. A GPU driver
   starts faster; this has to be measured on the reference laptop. If it is still above
   150 ms there, the next work is to load the fonts in parallel with the window creation.
2. **Resident memory, 287 MB against 60 MB.** The Go runtime holds 41 MB; the rest is
   llvmpipe (its JIT, its thread stacks and its textures), the Wayland buffers and the
   mapped executable and fonts. Same remark: it needs the reference hardware.
3. **Allocations of a frame in which a `CustomPaint` repaints.** A recorded display op
   keeps its `Paint` by value (152 bytes), so an op is about 200 bytes and a frame that
   repaints N ops allocates N × 200 bytes. The target holds for frames that do not repaint
   (a transform, an opacity or a scroll offset animation, 932 B). Lowering the size of
   `Paint` (pointers for the shader and filters) is a change of the `ui` API and of the
   layout of every op; it is not done here.

## Pending

- Measure again on the reference hardware (Iris Xe, 1920×1080 at 1.25): the cold start,
  the memory and the raster time.
