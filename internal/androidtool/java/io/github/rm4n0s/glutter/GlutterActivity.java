package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.content.res.Configuration;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import java.io.File;
import java.nio.charset.StandardCharsets;

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

  /** The view of the activity that is shown. Only the main thread touches it. */
  private GlutterView view;

  /** The system interface of the activity (bars, orientation, feedback). */
  private SystemUI systemUI;

  /** Text input, the soft keyboard and the clipboard of the activity. */
  private TextInput textInput;

  /** Back navigation of the activity: the callback that is registered with the system. */
  private BackNavigation backNavigation;

  /** The accessibility bridge of the activity's view (TalkBack and the other services). */
  private GlutterAccessibility accessibility;

  /** The application's context, for what Go asks from a thread that is not the main one. */
  private static volatile Context appContext;

  /** Whether the settings are watched: once for the process. */
  private static boolean watching;

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    current = this;
    appContext = getApplicationContext();
    systemUI = new SystemUI(this);
    systemUI.enableEdgeToEdge();
    if (started) {
      // A new activity in a living process: the program runs already, and the view gets the
      // new surface (SPEC §24.2).
      createView();
      Native.settings(SystemSettings.describe(this));
      Native.activityCreated();
      accessibility.start();
      return;
    }
    started = true;
    System.loadLibrary("app");
    setEnvironment();
    createView();
    // The settings are known before main starts: the first frame is drawn with them.
    Native.settings(SystemSettings.describe(this));
    if (!watching) {
      watching = true;
      SystemSettings.watch(appContext);
    }
    Native.start(this);
    Native.activityCreated();
    // After activityCreated, which makes Go forget the tree of the activity before: the state of
    // the accessibility manager is reported again.
    accessibility.start();
  }

  /** Makes the view and the text input that goes with it, and shows the view. */
  private void createView() {
    view = new GlutterView(this, systemUI);
    textInput = new TextInput(view);
    backNavigation = new BackNavigation(this);
    view.setBackNavigation(backNavigation);
    view.setTextInput(textInput);
    accessibility = new GlutterAccessibility(view);
    view.setAccessibility(accessibility);
    setContentView(view);
  }

  @Override
  public void onConfigurationChanged(Configuration newConfig) {
    super.onConfigurationChanged(newConfig);
    Native.settings(SystemSettings.describe(this));
    if (view != null) {
      view.reportDisplay();
    }
  }

  /**
   * The legacy back (below API 33, or the back key): a pop through the framework, which exits the
   * app by itself when it cannot pop (FlutterActivity.onBackPressed calls {@code popRoute}).
   */
  @Override
  @SuppressWarnings("deprecation")
  public void onBackPressed() {
    Native.back(BackNavigation.KIND_POP, 0, 0, 0, 0);
  }

  /** The system is short of memory: the framework's memory pressure handlers run (SPEC §24.8). */
  @Override
  public void onTrimMemory(int level) {
    Native.trimMemory(level);
    super.onTrimMemory(level);
  }

  @Override
  protected void onResume() {
    super.onResume();
    Native.lifecycle(1);
  }

  @Override
  protected void onPause() {
    Native.lifecycle(2);
    super.onPause();
  }

  @Override
  protected void onStop() {
    Native.lifecycle(3);
    super.onStop();
  }

  @Override
  public void onWindowFocusChanged(boolean hasFocus) {
    super.onWindowFocusChanged(hasFocus);
    Native.focus(hasFocus);
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
    if (backNavigation != null) {
      backNavigation.release();
    }
    if (accessibility != null) {
      accessibility.release();
    }
    super.onDestroy();
    if (isFinishing()) {
      // Destroyed for good, not to be created again (a configuration change or the system).
      Native.lifecycle(4);
    }
    if (mainEnded) {
      System.exit(exitCode);
    }
  }

  /**
   * Called by Go, from any thread, when the framework has answered a key event: an event that it did
   * not handle is dispatched again to the rest of the view hierarchy and the system, so that Back,
   * the volume keys and the soft keyboard's keys still work.
   */
  static void onKeyHandled(final int id, final boolean handled) {
    MAIN.post(
        new Runnable() {
          @Override
          public void run() {
            if (current != null && current.view != null) {
              current.view.keyHandled(id, handled);
            }
          }
        });
  }

  /**
   * Called by Go, from any thread, for the dev events script of debug builds: a tap (kind 1), a swipe
   * (2), a key (3), text committed through the input connection (4), a composition (5), the keyboard's action key (6),
   * a back (7, {@code key} is {@code pop}, {@code gesture} or {@code cancel}) or an accessibility action (8, {@code
   * key} is the action, a comma and the label of the node), dispatched to the view as the user's input is. {@code key} is UTF-8.
   */
  static void inject(
      final int kind,
      final float x1,
      final float y1,
      final float x2,
      final float y2,
      final int ms,
      final byte[] key) {
    final String keyText = new String(key, StandardCharsets.UTF_8);
    MAIN.post(
        new Runnable() {
          @Override
          public void run() {
            if (current != null && current.view != null) {
              current.view.inject(kind, x1, y1, x2, y2, ms, keyText);
            }
          }
        });
  }

  /**
   * Called by Go, from any thread: one operation of the system's interface, haptic feedback, sounds
   * or the screen (systemui.go, operations 1 to 7; {@link SystemUI}), or of text input and the
   * clipboard (textinput.go, 8 to 13; {@link TextInput}), of back navigation (back.go, 14
   * and 15; {@link BackNavigation}) or of accessibility (accessibility.go, 16 to 18; {@link
   * GlutterAccessibility}), done on the main thread. {@code bytes} is the operation's text as UTF-8,
   * or, for the semantics update (16), a binary payload.
   */
  static void command(
      final int op, final long a, final long b, final long c, final long d, final byte[] bytes) {
    final boolean binary = op == GlutterAccessibility.OP_SEMANTICS;
    final String s = binary ? null : new String(bytes, StandardCharsets.UTF_8);
    MAIN.post(
        new Runnable() {
          @Override
          public void run() {
            if (current == null) {
              return;
            }
            if (op >= GlutterAccessibility.OP_SEMANTICS && op <= GlutterAccessibility.OP_RESET) {
              if (current.accessibility != null) {
                current.accessibility.command(op, a, bytes);
              }
            } else if (op == BackNavigation.OP_FRAMEWORK_HANDLES_BACK || op == BackNavigation.OP_FINISH) {
              if (current.backNavigation != null) {
                current.backNavigation.command(op, a);
              }
            } else if (op >= TextInput.OP_TEXT_START) {
              if (current.textInput != null) {
                current.textInput.command(op, a, b, c, d, s);
              }
            } else if (current.systemUI != null) {
              current.systemUI.command(op, a, b, c, d, s);
            }
          }
        });
  }

  /**
   * Called by Go, from the UI thread, and it returns at once: what the system makes of a font size
   * of {@code size} logical pixels (not linear on Android 14 and later).
   */
  static float scaledFontSize(float size) {
    Context context = appContext;
    return context == null ? size : SystemSettings.scaledFontSize(context, size);
  }

  /** Called by Go, from any thread: the mouse cursor, as a {@code PointerIcon} type. */
  static void setCursor(final int type) {
    MAIN.post(
        new Runnable() {
          @Override
          public void run() {
            if (current != null && current.view != null) {
              current.view.setCursorType(type);
            }
          }
        });
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
