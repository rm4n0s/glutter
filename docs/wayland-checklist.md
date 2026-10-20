# Wayland checklist

The manual part of the Wayland compatibility matrix (SPEC §17). The automated part is
`scripts/wayland-matrix.sh`, which runs the display tests and the examples under the headless
compositors that are installed (sway, weston, mutter, kwin) and prints a pass/fail table. This
list is what a headless compositor cannot show: real desktops, real GPUs, real monitors and
real input devices.

One checkbox per item of SPEC §8.11. Fill in the tables for the machines you have; leave the
rest empty. Write `[x]` for works, `[!]` for a problem (and add a line under "Problems" below
with what you saw), and leave `[ ]` for not tried. A cell of `-` means the item does not apply
there.

How to run: `go run ./examples/<name>` inside the session you are testing; set
`GLUTTER_BACKEND=wayland` to be sure it is not XWayland. The hint in each row says which
example shows it.

## Your machine

| | |
|---|---|
| Date | |
| Compositor and version | |
| GPU and driver (`glxinfo -B` or `eglinfo`) | |
| Scales tried | 100% / 125% / 150% / 200% |
| Monitors | |

## Items

### Frame pacing

| # | Item | How to check |
|---|---|---|
| F1 | Frames are paced by `wl_surface.frame`; `eglSwapBuffers` never blocks, also with the window hidden or covered | run `examples/animations`, cover the window with another one; CPU stays near zero |
| F2 | A `suspended` window (minimized, other workspace) renders nothing; animations jump to the right point when it comes back | minimize `examples/animations`, wait 5 s, restore: the animation is not replaying |
| F3 | One frame in flight per window; input between frames is batched | drag a slider in `examples/gallery` quickly: no lag that grows |

### Resizing and scaling

| # | Item | How to check |
|---|---|---|
| R1 | Interactive resize tracks the pointer; no frame of the old size after the new one | drag a corner of `examples/gallery` for 10 s: no flicker, no black strips |
| R2 | The decoration and the content resize in step (libdecor subsurfaces) | GNOME only: resize with the Adwaita title bar |
| R3 | Fractional scale: text and shapes are sharp at the buffer resolution (125%, 150%, 175%) | zoom into a screenshot of the gallery text |
| R4 | Moving a window between monitors of different scales re-renders at once, no blurry frame | two monitors: drag `examples/multiwindow` across |
| R5 | Outputs plugged and unplugged while the app runs are noticed (`Displays()`, the window's metrics) | unplug a monitor under `examples/multiwindow` |

### Window decorations

| # | Item | How to check |
|---|---|---|
| D1 | Server-side decorations where the compositor has them (KDE, Sway, Hyprland, COSMIC) | title bar is the compositor's |
| D2 | libdecor where it has none (GNOME): native Adwaita title bar, buttons work, window menu opens | GNOME: `examples/counter` |
| D3 | `WindowDecoration` when neither exists (and with `DecorationsApp`): move, resize at the edges, buttons, double click, right click menu | `GLUTTER_DECORATIONS=app go run ./examples/counter` |

### Input

| # | Item | How to check |
|---|---|---|
| I1 | Keyboard layouts and repeat follow the compositor; keys are released when the focus leaves | type in `examples/editor`, switch layout, hold a key, alt-tab |
| I2 | IME: `zwp_text_input_v3` (GNOME, KDE, wlroots) or IBus/Fcitx5 over D-Bus: preedit, commit, candidates near the caret | type Japanese or Chinese in `examples/editor` |
| I3 | Wheel (high resolution), touchpad scrolling with kinetic stop, pinch and rotate | scroll `examples/scrolling`, pinch in `examples/gallery` (InteractiveViewer page) |
| I4 | Touch screen: taps, drags, multi-touch | if you have one |
| I5 | Stylus: hover, pressure, tilt, the barrel buttons, the eraser (`zwp_tablet_v2`) | draw with a tablet in a `Listener` example |
| I6 | Cursors: the compositor draws the shape (`wp_cursor_shape_v1`) or the XCursor theme at the right size and scale | hover buttons, text fields, resize edges; change the cursor theme and size in the settings |

### Data exchange

| # | Item | How to check |
|---|---|---|
| X1 | Clipboard: text, an image and a list of files, both directions with another app | copy from the editor to a text editor and back; copy an image and files from the file manager |
| X2 | Primary selection: select text, middle click pastes | select in `examples/editor`, paste in a terminal |
| X3 | Drop of files and text from the file manager and an editor onto `examples/multiwindow` | Files/Dolphin, then a text editor |
| X4 | Drag from `examples/multiwindow` ("Drag me") into another application, with the drag icon | drop into a text editor |

### Lifecycle and correctness

| # | Item | How to check |
|---|---|---|
| L1 | Launching from a launcher or a notification ends the startup feedback (`XDG_ACTIVATION_TOKEN`) | start an app from the application menu: the spinner stops |
| L2 | `Activate()` on a window asks the compositor for the focus (it may only flash the task bar) | `examples/multiwindow`: activate the first window from the second |
| L3 | The app id matches the `.desktop` file: the icon and name in the task bar are the app's | install a `.desktop` file for an example |
| L4 | The window icon from `WindowConfig.Icon` shows where `xdg_toplevel_icon_v1` exists (and the `.desktop` icon elsewhere) | task bar, Alt-Tab |
| L5 | `desktop.KeepScreenOn(bc, true)` keeps the screen from blanking or locking; `false` lets it | set a 1 minute idle timeout and watch |
| L6 | `WindowConfig.ContentType` reaches the compositor (visible only where it uses it, e.g. adaptive sync for games) | compositor log or `WAYLAND_DEBUG=1` |

### Settings

| # | Item | How to check |
|---|---|---|
| S1 | Dark mode, accent color, text scale, cursor theme and size, high contrast and reduced motion follow the system and change live | toggle each in the system settings while `examples/gallery` runs |

## Per compositor

Every item above, once for each desktop. `-` is filled in where the item cannot apply (for example R2 and D2 are GNOME only, D1 is not GNOME).

| # | GNOME (Mutter) | KDE (KWin) | Sway | Hyprland | Weston | COSMIC |
|---|---|---|---|---|---|---|
| F1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| F2 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| F3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| R1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| R2 | [ ] | - | - | - | - | - |
| R3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| R4 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| R5 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| D1 | - | [ ] | [ ] | [ ] | [ ] | [ ] |
| D2 | [ ] | - | - | - | - | - |
| D3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I2 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I4 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I5 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| I6 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| X1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| X2 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| X3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| X4 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L2 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L3 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L4 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L5 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| L6 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |
| S1 | [ ] | [ ] | [ ] | [ ] | [ ] | [ ] |

## Per GPU and driver

The items that depend on the driver, once for each GPU (llvmpipe is `LIBGL_ALWAYS_SOFTWARE=1`). Also run `go test -tags glutter_display ./internal/gpu/gles/` on each GPU: the GPU must match the software renderer within the tolerances.

| # | Intel (Mesa) | AMD (Mesa) | NVIDIA (proprietary) | llvmpipe |
|---|---|---|---|---|
| F1 | [ ] | [ ] | [ ] | [ ] |
| F2 | [ ] | [ ] | [ ] | [ ] |
| R1 | [ ] | [ ] | [ ] | [ ] |
| R3 | [ ] | [ ] | [ ] | [ ] |
| R4 | [ ] | [ ] | [ ] | [ ] |
| D3 | [ ] | [ ] | [ ] | [ ] |
| L5 | [ ] | [ ] | [ ] | [ ] |
| display tests (`glutter_display`) | [ ] | [ ] | [ ] | [ ] |

## Per scale

R1, R3 and I6 at each scale of one compositor, and the pointer landing where the cursor is drawn (click the edge of a button).

| # | 100% | 125% | 150% | 175% | 200% |
|---|---|---|---|---|---|
| R1 | [ ] | [ ] | [ ] | [ ] | [ ] |
| R3 | [ ] | [ ] | [ ] | [ ] | [ ] |
| I6 | [ ] | [ ] | [ ] | [ ] | [ ] |
| pointer position | [ ] | [ ] | [ ] | [ ] | [ ] |

## Two monitors with different scales

| Check | Done |
|---|---|
| R4: drag a window across, sharp at once on both | [ ] |
| R5: unplug and plug the second monitor | [ ] |
| The cursor has the right size on each monitor | [ ] |
| A window that opens on the second monitor has its scale from the start | [ ] |
| `examples/multiwindow`: a dialog on the other monitor than its parent | [ ] |

## Problems

Write what you saw: the item, the compositor and GPU, what happened, and what you expected.

- 
