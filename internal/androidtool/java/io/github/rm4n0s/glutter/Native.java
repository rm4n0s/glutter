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
}
