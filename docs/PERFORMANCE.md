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

# the web numbers (step 21.7): sizes, start, idle CPU, frames, a full rebuild, clicks; Chromium and Firefox
scripts/perf-web.sh
# the framework in WebAssembly under Node (the same V8 as Chromium's): add -cpuprofile with node --cpu-prof
GOOS=js GOARCH=wasm go test -tags glutter_release ./examples/gallery -run xxx -bench 'Gallery' -benchmem
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
| `examples/gallery` `BenchmarkGalleryPageSwitch` | a click on an entry of the navigation rail and the frame that builds the new page (every page built from nothing) |
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

## Vulkan and GLES (step 20.7)

`scripts/perf.sh -n 5 -s 10` with `GLUTTER_RENDERER=gles` and with `GLUTTER_RENDERER=vulkan` (Mesa's
**lavapipe**, with `VK_DRIVER_FILES` pointing at it), on the same machine and the same headless
Sway, 2026-10-21. Both are software rasterizers on the CPU, so the raster time compares two
drivers' JITs, not a GPU through its two APIs; the rest of the numbers are the engine's.

| Metric | GLES (llvmpipe) | Vulkan (lavapipe) |
|---|---|---|
| Idle CPU, counter | 0.00% | 0.00% |
| Cold start to first frame, counter (median of 5) | 280 ms | 330 ms |
| Resident memory, counter | 281 MB | 253 MB |
| Gallery frames per second / over 16.7 ms | 60.3 / 1 | 61.0 / 100 |
| Gallery build time per frame, average / worst | 0.61 / 1.09 ms | 0.43 / 0.80 ms |
| Gallery raster time per frame, average / worst | 10.2 / 299.6 ms | 15.2 / 21.1 ms |

**Not decided by these numbers.** Step 20.7 asks that Vulkan is not more than 10% slower than
GLES on any metric. Here Vulkan's raster time is 49% higher on average and its cold start 18%
higher, but its worst frame is far better and its memory lower, and with a software device all
of it is the CPU's work inside lavapipe against llvmpipe. A GPU can present only to a window it
can import buffers from, which the headless compositor does not give (the real GPU of this
machine fails at the swapchain there), so the comparison that counts has to be run on the
user's desktop (the commands are in the manual checks of step 20.7).

## Web (step 21.7)

`scripts/perf-web.sh` (release builds, `glutter-web perf`, SPEC §15 and §23.8), 2026-10-22.

| | |
|---|---|
| Machine | the one above: AMD Ryzen AI Max+ 395, Radeon 8060S, Fedora 44, Go 1.27.0 |
| Chromium | 155.0.8059.84, headless, WebGPU on the machine's **AMD GPU through Mesa RADV** (`--enable-features=Vulkan --use-angle=vulkan`, adapter `amd rdna-3`) |
| Firefox | 158.0, headless, WebGL 2 on **Mesa's software rasterizer behind a private Xvfb**: it is not a GPU (headless Firefox on Linux has no GL without a display, and the browser reports a masked "Radeon R9 200 Series" name). Its raster times and CPU numbers include the rasterizer, and say nothing of a GPU |
| View | 1536x864 CSS pixels at a device pixel ratio of 1.25 (a 1920x1080 display), a temporary profile for each run, nothing of the user's reached |
| What is run | the counter (idle), and the gallery: the data page with its indicators animating, the buttons page rebuilt in every frame (`GLUTTER_GALLERY_REBUILD=1`), and a click on every entry of the navigation rail, twice through |

Headless browsers have no display, so their frame cadence is the browser's own (about 120 a second
here) and not a monitor's vertical sync. The "UI thread per frame" figures are the engine's own
(`GLUTTER_FRAME_STATS`): the time on the UI goroutine of the build, layout and paint, and of the
renderer's calls into the browser; the GPU's work after them is not in it.

| Metric (SPEC §15, web) | Target | Chromium, WebGPU (AMD) | Firefox, WebGL 2 (software) | |
|---|---|---|---|---|
| Idle CPU, counter | ~0%: nothing asked for while nothing is scheduled | 0 `requestAnimationFrame` and 0 timers armed in 10 s; 0.4% of a core, against 0.8% for a blank page | 0 and 0; 3.7% of a core against 6.5% for a blank page (Xvfb and Firefox's own work) | met |
| Frame rate, gallery data page, one animation | the display's refresh rate with no dropped frames | 117.8 frames a second; 13 of 1,172 intervals between animation callbacks over 16.7 ms, longest 47.7 ms | 119.2; 12 of 1,194 over 16.7 ms, longest 35 ms | **not met strictly**: about 1% of the frames are late. No frame took over 16.7 ms on the Go side |
| UI thread per frame, animation only | < 4 ms | 3.2 ms (build 0.6, raster 2.6); worst second 3.4 ms | 3.2 ms (build 0.65, raster 2.5); worst second 3.3 ms | met on average (other runs of the same build: 3.5 ms, worst second 4.2 ms) |
| UI thread per frame, full rebuild of a gallery page | < 16 ms | 11.1 ms (build 9.5, raster 1.7); worst second 12.3 ms. Before this step: 20.6 ms | 13.0 ms (build 11.2, raster 1.8); worst second 13.4 ms. Before: 22.8 ms | met |
| Size of `app.wasm.gz`, counter | < 4 MB | **8.8 MB** (9,255,700 bytes) | the same file | **not met**, 2.2 times |
| Size of `app.wasm.gz`, gallery | < 5 MB | **11.9 MB** (12,499,424 bytes) | the same file | **not met**, 2.4 times |
| Start to first frame, `app.wasm` in the cache | < 1 s | counter 440 ms, gallery 601 ms (median of 3 loads) | counter 284 ms, gallery 419 ms | met |

Sizes of the release builds (`-tags glutter_release`), counter / gallery: 40.0 MB / 61.4 MB with
symbols, 38.3 MB / 58.3 MB with `-ldflags=-s -w`, and 8.8 MB / 11.9 MB gzipped as `glutter-web`
serves it (best compression). With an empty cache the first frame comes after 1.05 s (counter) and
1.15 s (gallery) in Chromium, and 0.90 s and 1.11 s in Firefox; the target is for the cache. The Go
runtime's memory while the gallery animates or switches pages is 150 to 280 MB (see "The collector").

### Targets not met

1. **Size of `app.wasm.gz`: 8.8 MB against 4 MB (counter), 11.9 MB against 5 MB (gallery).** The
   figure of SPEC §15 for these programs without the web platform and the renderers (2.3 MB / 2.4 MB)
   was not a measurement of them; a "hello world" in Go for js/wasm is 2.4 MB (0.7 MB gzipped) alone.
   Where the size is: the WebAssembly module has 18.5 MB of code and 21.6 MB of data; 4 MB of the data
   are the two bundled fonts, which gzip to 1.96 MB (Roboto 1.4 MB, Material Icons 0.56 MB), and the rest of
   the data (Go's function tables and type information, the shader sources, the tables of go-text) was not
   broken down. The Linux binary of the counter is also 35 MB, so this is the framework and not the web
   layer: by the symbol sizes of that binary, `material` is 1.4 MB, `widgets` 1.1 MB and `rendering` 0.6 MB. Levers, none applied (each needs a decision): brotli instead of gzip
   (measured: 6.5 MB counter, 8.6 MB gallery, 29% less, but a `.br` file and a server that sends it);
   the fonts fetched as separate files on demand (the same bytes, but not in the module, and cached
   apart); a subset of Roboto and of the icon font; fewer generated methods (`DebugFillProperties`,
   `Lerp` and `Equal` of every theme class). None of them reaches 4 MB / 5 MB alone; brotli with
   subset fonts would be about 5 / 7 MB. **Recommendation:** the user decides between raising the
   targets to what a Go framework can do (about 10 MB / 12 MB gzipped, 7 / 9 with brotli) and a size
   work item of its own (a step after 1.1).
2. **Frame rate, "no dropped frames": about 1% of the frames are late** (13 of 1,172 animation
   callbacks in Chromium were over 16.7 ms apart, the longest 47.7 ms, and 12 of 1,194 in Firefox). The
   frames themselves are short (3.2 ms), so the cause is something else that takes the thread between
   frames; the likely one is the garbage collector (one thread: its marking runs when the frame is done
   and the goroutine waits), which was not proved. A headless browser has no vertical sync, so the
   cadence is also the browser's own. **Recommendation:** measure it again with a real display on the
   reference machine before deciding; it may be met there.

The other web targets are met on this machine. They were met by the changes below; the machine is
faster than the laptop of SPEC §15, so the margin there is smaller (the full rebuild is 11 to 13 ms of a
16 ms target here).

### What was changed for the web numbers

A profile of the benchmarks under Node (`node --cpu-prof` on the test binaries; the browser gives the
same numbers as Node, see `scripts/perf-web.sh`) found, in the full rebuild of the gallery page
(20.6 ms in Chromium, 22.8 ms in Firefox before):

- `RuntimeTypeName` (two regular-expression replacements) was called for every `CustomPaint` and every
  clipper in every build: 10%. Now `foundation.SameRuntimeType`.
- Every button made a new `ThemeData` (`copyWith(iconTheme:)`) and its `AnimatedTheme` compared it field
  by field with the last: 11%. Now the same copy comes back for an equal icon theme (SPEC §12.10).
- The gallery made its two themes again in every build (a color scheme is solved from the seed) and
  compared them: 17%. Now it makes them when the seed changes.
- The inherited-widget maps were written again for every dependency in every build: 2%.
- 21.2 ms to 12.2 ms under Node, and in the browsers 20.6 ms to 11.1 ms (Chromium), 22.8 ms to 13.0 ms (Firefox).

### Clicks on the gallery's navigation

The owner found a click on the page list of the gallery slow, in both browsers. The measurement is the
time from the release of the mouse button to the end of the next frame (`glutter-web perf -clicks`, the
frame's callback, which includes the build, layout, paint and the renderer's calls; the GPU after it is not
counted), clicking on every entry twice through, in a release build. There is no target in SPEC §15.

| | Before (first visit / second) | After (first visit / second) |
|---|---|---|
| Chromium, median of 13 clicks | 122 ms / 124 ms | **119 ms / 66 ms** |
| Chromium, mean; worst | 119, 201 ms / 115, 204 ms | 109, 196 ms / 72, 179 ms |
| Firefox, median of 13 clicks | 132 ms / 128 ms | **110 ms / 79 ms** |
| Firefox, mean; worst | 120, 212 ms / 116, 192 ms | 102, 172 ms / 80, 130 ms |

**It is still slow** on a first visit (about 110 to 120 ms, up to 200 ms), which is what a user sees when
they click around a page they have not opened; the second visit is better (66 to 79 ms). What the profile
shows, the same in both browsers (so it is not the renderer):

- **Go on one thread in WebAssembly.** The frame that follows a click builds a new screen from nothing:
  a page of the gallery is 3 to 16 ms in a native build (`BenchmarkGalleryPageSwitch`: 8.5 ms) and 17 to
  90 ms in WebAssembly (33 ms on average under Node, warm), four to seven times. It is split between the
  build (element, state and render object creation, text layout 13%, theme and semantics objects) and
  layout; there is no one hot spot left, the profile is flat. About 5% of it is `runtime.Stack` in the
  check that `SetState` is on the UI goroutine (`goid.ID`), which only a change of how that check is made
  (SPEC §13) can remove.
- **The raster of a new scene** is 25 to 90 ms: the explicit renderer draws the clips of the page's cards on the CPU
  (about half: coverage masks, then their upload) and flattens the scene; a scene whose layers it has
  not seen has no raster cache hits. The calls into the browser are not the cost (SPEC §23.8, §22).
- **The collector.** With the default settings about a quarter of the click's time was garbage
  collection (a page allocates 5.7 MB in 57,000 objects), and a collection can take the thread for tens of
  milliseconds. The web now collects at five times the live heap (`engine.tuneGC`): mean click 112 ms
  to 88 ms in the experiment that chose the setting, and 84 ms with no collection at all.
- **A first visit** is slower than the second, after the changes, by about 50 ms in Chromium and 30 ms in
  Firefox. Likely causes, not separated: the browser compiles the WebAssembly code the first time it runs
  (`--no-liftoff`, which compiles each function when it is first called, made a click take 0.2 to 1 s), and
  the caches of the text, the masks and the glyphs are empty. Before the changes the second visit was
  not faster (the collector's work grew with the heap through the run), so the two effects were mixed.
- Changes that came out of it: `min` and `max` of Go in the rasterizer's loops (the `math` functions are
  not inlined in WebAssembly: 80 ms of 315 in the cold raster of 13 pages), a faster clip-coverage
  loop, and the collector setting. A first visit is not better because most of its time is code
  that has not run before.

Not done, in the order of what it would give: make the UI-thread check cheap on the web (about 5% of a
page switch); allocate the semantics configuration of a render object when it is needed, not when it is
attached (about 2% and many small objects); cache the text layouts of the page across page switches;
draw rounded-rectangle clips analytically on the GPU instead of through CPU masks (the largest part of
the raster of a new page); and a gallery that builds fewer widgets for a page that is mostly below the
fold. None of these is a one-line change; the largest ones change SPEC §9 or §13.

### The collector

A WebAssembly program has one thread, so the collector takes its time from the frames. `engine.tuneGC` sets
`GOGC` to 400 and a soft memory limit of 768 MiB when the program has not been given `GOGC` or
`GOMEMLIMIT`. The cost is memory: the Go runtime held 150 to 280 MB in the runs above against 66 to 85 MB
at the default.

### Instrumentation

- `GLUTTER_FRAME_STATS=1` also logs each frame over a 60th of a second on its own (`slow frame
  build=... raster=...`), so that the frame behind a click can be told from its second's average.
- `glutter-web perf` is described in SPEC §23.8; it needs `chromium`, `firefox` and `Xvfb`.

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

- The web table: measure the targets again with a real display, and Firefox on a GPU (the Firefox numbers here are Mesa's software rasterizer). The two targets that are not met (sizes, late frames) need the user's decision (see "Targets not met" under Web).

- Run `GLUTTER_RENDERER=gles scripts/perf.sh` and `GLUTTER_RENDERER=vulkan scripts/perf.sh` (a headless compositor cannot present GPU buffers: use the benchmarks of the gallery with `GLUTTER_FRAME_STATS=1` on the desktop) on a machine with a GPU, and record the comparison of step 20.7.
- Measure again on the reference hardware (Iris Xe, 1920×1080 at 1.25): the cold start,
  the memory and the raster time.
