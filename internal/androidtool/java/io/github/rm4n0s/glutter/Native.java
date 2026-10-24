package io.github.rm4n0s.glutter;

/**
 * The native methods of the embedding and the calls from Go. Go implements the methods in
 * internal/platform/android (cgo, JNI); Go calls the static methods of GlutterActivity.
 *
 * <p>Ports: the {@code FlutterJNI} of Flutter's embedding
 * (shell/platform/android/io/flutter/embedding/engine/FlutterJNI.java), only the part that starts
 * the engine.
 */
final class Native {
  private Native() {}

  /**
   * Sets an environment variable of the Go program, as {@code os.Setenv} does. Go copies the
   * environment when it starts, so {@code setenv(3)} of libc would not be seen by {@code
   * os.Getenv}.
   */
  static native void setenv(String key, String value);

  /**
   * Runs the program's {@code main.main} on a goroutine locked to its own OS thread, once per
   * process. It returns at once.
   */
  static native void start(GlutterActivity activity);

  /**
   * The view has a new surface. Go acquires the native window and releases it when it has let go
   * of it.
   */
  static native void surfaceCreated(android.view.Surface surface);

  /** The size of the surface in pixels, the density and refresh rate of the display, and its size. */
  static native void surfaceChanged(
      int width, int height, float density, float refreshRate, int displayWidth, int displayHeight);

  /**
   * The surface is going away. It does not return until the renderer has let go of it (at most two
   * seconds), because Android destroys the surface when it returns.
   */
  static native void surfaceDestroyed();

  /** The lifecycle of the activity: 1 resumed, 2 paused, 3 stopped. */
  static native void lifecycle(int state);

  /** Whether the activity's window has the focus. */
  static native void focus(boolean focused);

  /**
   * A motion event copied out: {@code meta} is {actionMasked, actionIndex, source, buttonState,
   * eventTime, pointerCount, historySize, generic}; {@code ids} has the pointer id and the tool
   * type of each pointer; {@code vals} has 11 values for each pointer (x, y, pressure, size,
   * toolMajor, toolMinor, orientation, tilt, distance, hscroll, vscroll), first for the event,
   * then for each historical sample, oldest first; {@code histTimes} the time of each sample.
   */
  static native void motionEvent(
      long[] meta,
      int[] ids,
      float[] vals,
      long[] histTimes,
      float pressureMin,
      float pressureMax,
      float hScrollFactor,
      float vScrollFactor);

  /** A key event copied out; {@code id} comes back in {@code GlutterActivity.onKeyHandled}. */
  static native void keyEvent(
      int id,
      int action,
      int keyCode,
      int scanCode,
      int metaState,
      int repeatCount,
      long eventTime,
      int unicodeChar,
      int source,
      int deviceId);

  /**
   * The insets of the window in pixels: left, top, right and bottom of the system bars and the
   * display cutout, of the input method, and of the system gestures.
   */
  static native void insets(int[] values);

  /**
   * The system's settings as {@code key=value} pairs separated by {@code ;} (see settings.go of
   * the Go side): night, font, nonlinear, locales, h24, noanim, bold, contrast.
   */
  static native void settings(String text);

  /** The system showed ({@code true}) or hid its bars because of the user. */
  static native void systemUI(boolean visible);

  /** The density, the refresh rate and the size of the display changed. */
  static native void displayChanged(float density, float refreshRate, int displayWidth, int displayHeight);
}
