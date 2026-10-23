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
