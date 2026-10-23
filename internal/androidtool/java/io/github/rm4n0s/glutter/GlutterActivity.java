package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import java.io.File;

/**
 * The activity of a Glutter app. It copies the {@code GLUTTER_} extras of the intent and the
 * directories of the app into the environment of the Go program, loads {@code libapp.so} and
 * starts the program's {@code main}. It finishes, and the process exits with the program's exit
 * code, when {@code main} returns.
 *
 * <p>Ports: {@code FlutterActivity}
 * (shell/platform/android/io/flutter/embedding/android/FlutterActivity.java, {@code onCreate})
 * and {@code FlutterActivityAndFragmentDelegate} (the order of the setup), only the parts that
 * step 25.2 needs.
 */
public class GlutterActivity extends Activity {
  /** The prefix of the intent extras that become environment variables. */
  private static final String EXTRA_PREFIX = "GLUTTER_";

  /** How long the process waits for the system to destroy the finished activity. */
  private static final long EXIT_DELAY_MS = 2000;

  private static final Handler MAIN = new Handler(Looper.getMainLooper());

  /** The activity that is shown now, or null. Only the main thread touches it. */
  private static GlutterActivity current;

  /** Whether {@code Native.start} was called in this process. */
  private static boolean started;

  /** Whether {@code main} returned, and the code it returned. */
  private static boolean mainEnded;

  private static int exitCode;

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    current = this;
    if (started) {
      // A new activity in a living process: the program runs already (SPEC §24.2).
      return;
    }
    started = true;
    System.loadLibrary("app");
    setEnvironment();
    Native.start(this);
  }

  /** Copies the app's directories and the {@code GLUTTER_} extras into the Go program's environment. */
  private void setEnvironment() {
    File files = getFilesDir();
    File cache = getCacheDir();
    Native.setenv("GLUTTER_APP_ID", getPackageName());
    Native.setenv("HOME", files.getAbsolutePath());
    Native.setenv("TMPDIR", cache.getAbsolutePath());
    Native.setenv("XDG_CONFIG_HOME", new File(files, ".config").getAbsolutePath());
    Native.setenv("XDG_DATA_HOME", new File(files, ".local/share").getAbsolutePath());
    Native.setenv("XDG_CACHE_HOME", cache.getAbsolutePath());
    Intent intent = getIntent();
    Bundle extras = intent == null ? null : intent.getExtras();
    if (extras == null) {
      return;
    }
    for (String key : extras.keySet()) {
      if (key.startsWith(EXTRA_PREFIX)) {
        Object value = extras.get(key);
        Native.setenv(key, value == null ? "" : String.valueOf(value));
      }
    }
  }

  @Override
  protected void onDestroy() {
    if (current == this) {
      current = null;
    }
    super.onDestroy();
    if (mainEnded) {
      System.exit(exitCode);
    }
  }

  /**
   * Called by Go, from any thread, when {@code main} returned or called {@code os.Exit}: the
   * activity finishes, and the process exits with {@code code} once the system has destroyed it.
   */
  static void onMainEnded(final int code) {
    MAIN.post(
        new Runnable() {
          @Override
          public void run() {
            mainEnded = true;
            exitCode = code;
            if (current == null) {
              System.exit(code);
              return;
            }
            current.finish();
            // Normally onDestroy exits. If the system is slow to destroy the activity, exit anyway.
            MAIN.postDelayed(
                new Runnable() {
                  @Override
                  public void run() {
                    System.exit(exitCode);
                  }
                },
                EXIT_DELAY_MS);
          }
        });
  }
}
