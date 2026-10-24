# Manual checks on a phone

Checks that only a real phone shows. Each step's entry in `docs/IMPLEMENTED.md` says "pending" until the
owner has done them and their answers are recorded here. Tool: `adb devices` must list the phone (USB
debugging on). Run the commands from the module root.

## 25.4 GLES on Android

**Status:** pending

Run:

```
go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/gallery
```

Check, and record the answers:

1. Does the gallery appear and follow your finger when you scroll the page and the navigation list?
2. Switch the theme (the moon button at the top right, then Light and Dark): do the colors change at once, with no flicker?
3. Press Home and open the app again: is it drawn again at once, with nothing missing?
4. Record the renderer line of the log (`renderer chosen renderer="gles: ..."`): which GPU does it name?

Answers: _(not yet recorded)_

## 25.5 Vulkan on Android, and choosing the renderer

**Status:** pending

Run:

```
go run ./cmd/glutter-android run ./examples/gallery
```

Check, and record the answers:

1. Read the renderer and the device from the log (`renderer chosen renderer="vulkan: ..."` and the `Vulkan device` line before it, with its `api=`). Which GPU and which Vulkan version does it name? If it says `gles` instead, copy the `renderer not usable` line above it: that is the reason (Android below 10, a driver on the deny list, or a Vulkan error).
2. Turn the phone to landscape and back (turn the automatic rotation on first): is the gallery upright and sharp each time, and does the log say a `Vulkan swapchain` line with `pre_rotation=` 1 or 3 in landscape and 0 in portrait?
3. Scroll the gallery's navigation list and the page: is the scrolling smooth, with no flicker or tearing?
4. Try `go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/gallery`: does it still draw the same?

Answers: _(not yet recorded)_

## 25.6 Touch, mouse, stylus and keys on Android

**Status:** pending

Run:

```
go run ./cmd/glutter-android run ./examples/scrolling
go run ./cmd/glutter-android run ./examples/gallery
go run ./cmd/glutter-android run ./examples/dev/android-input
```

Check, and record the answers:

1. In `examples/scrolling`, fling the list with one finger: does it keep moving after you lift the finger and slow down smoothly, and does a touch on the moving list stop it?
2. In `examples/gallery`, use two fingers on a map or a zoomable widget if the page has one, and drag a slider and a switch: do they follow the finger with no lag?
3. With a stylus, if you have one: does it draw or scroll like a finger, and is its pressure in the log of `examples/dev/android-input`?
4. With a mouse or a trackpad, if you have one (USB or Bluetooth): does the pointer hover, do the buttons and the wheel work, and does the cursor change over a text field?
5. With a Bluetooth keyboard, if you have one: in `examples/dev/android-input`, press K and the arrow keys: does the log say `shortcut ran`? Does Back still leave the app (until step 25.9)?

Answers: _(not yet recorded)_
