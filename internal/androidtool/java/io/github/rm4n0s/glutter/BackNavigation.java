package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.window.BackEvent;
import android.window.OnBackAnimationCallback;
import android.window.OnBackInvokedCallback;
import android.window.OnBackInvokedDispatcher;

/**
 * The back navigation of the activity. While the framework wants the back gesture ({@code
 * SystemNavigator.setFrameworkHandlesBack}, which {@code PopScope} and the {@code Navigator}
 * call), the activity has an on-back-invoked callback registered with the system (API 33 and
 * later): an {@code OnBackAnimationCallback} on API 34 and later, whose start, progress, commit
 * and cancel become the framework's predictive back events, and a plain callback on API 33. With
 * no callback registered the system does its own default (move the task to the back, or finish).
 * Below API 33 and for the legacy back key, {@code GlutterActivity.onBackPressed} sends a pop.
 * Go sends the operations; everything runs on the main thread.
 *
 * <p>Ports: {@code FlutterActivityAndFragmentDelegate}
 * (shell/platform/android/io/flutter/embedding/android/FlutterActivityAndFragmentDelegate.java:
 * {@code createOnBackInvokedCallback}, {@code registerOnBackInvokedCallback}, {@code
 * unregisterOnBackInvokedCallback}, {@code setFrameworkHandlesBack}), {@code FlutterActivity.onBackPressed}
 * (shell/platform/android/io/flutter/embedding/android/FlutterActivity.java), and the messages of
 * {@code BackGestureChannel} and {@code NavigationChannel}
 * (shell/platform/android/io/flutter/embedding/engine/systemchannels/BackGestureChannel.java and
 * NavigationChannel.java: {@code startBackGesture}, {@code updateBackGestureProgress}, {@code
 * commitBackGesture}, {@code cancelBackGesture}, {@code popRoute}).
 */
final class BackNavigation {
  // The operations of GlutterActivity.command (lifecycle.go).
  static final int OP_FRAMEWORK_HANDLES_BACK = 14;
  static final int OP_FINISH = 15;

  // The kinds of Native.back.
  static final int KIND_POP = 1;
  static final int KIND_START = 2;
  static final int KIND_PROGRESS = 3;
  static final int KIND_COMMIT = 4;
  static final int KIND_CANCEL = 5;

  private static final String TAG = "GlutterJava";

  /** The pause between the steps of a gesture of the dev events script. */
  private static final long GESTURE_STEP_MS = 60;

  private final Activity activity;
  private final Handler handler = new Handler(Looper.getMainLooper());

  /** The callback that is registered with the system, or null: this is the registration state. */
  private OnBackInvokedCallback registered;

  BackNavigation(Activity activity) {
    this.activity = activity;
  }

  /** Does one operation (14 or 15) of the activity's back navigation. */
  void command(int op, long a) {
    if (op == OP_FRAMEWORK_HANDLES_BACK) {
      setFrameworkHandlesBack(a != 0);
    } else if (op == OP_FINISH) {
      activity.finish();
    }
  }

  /** Registers the callback ({@code true}) or removes it, once, as {@code setFrameworkHandlesBack} does. */
  private void setFrameworkHandlesBack(boolean handles) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
      return;
    }
    if (handles && registered == null) {
      OnBackInvokedCallback callback = createCallback();
      activity
          .getOnBackInvokedDispatcher()
          .registerOnBackInvokedCallback(OnBackInvokedDispatcher.PRIORITY_DEFAULT, callback);
      registered = callback;
    } else if (!handles && registered != null) {
      activity.getOnBackInvokedDispatcher().unregisterOnBackInvokedCallback(registered);
      registered = null;
    }
  }

  /** Removes the registration when the activity is destroyed; the next activity starts with none. */
  void release() {
    setFrameworkHandlesBack(false);
  }

  private OnBackInvokedCallback createCallback() {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
      return new GestureCallback();
    }
    return new OnBackInvokedCallback() {
      @Override
      public void onBackInvoked() {
        Native.back(KIND_POP, 0, 0, 0, 0);
      }
    };
  }

  /** The callback of API 34 and later: the predictive back gesture. */
  private static final class GestureCallback implements OnBackAnimationCallback {
    @Override
    public void onBackStarted(BackEvent event) {
      send(KIND_START, event);
    }

    @Override
    public void onBackProgressed(BackEvent event) {
      send(KIND_PROGRESS, event);
    }

    @Override
    public void onBackInvoked() {
      Native.back(KIND_COMMIT, 0, 0, 0, 0);
    }

    @Override
    public void onBackCancelled() {
      Native.back(KIND_CANCEL, 0, 0, 0, 0);
    }

    private static void send(int kind, BackEvent event) {
      // BackEvent.EDGE_LEFT is 0 and EDGE_RIGHT is 1; Go has 0 for "unspecified".
      Native.back(kind, event.getTouchX(), event.getTouchY(), event.getProgress(), event.getSwipeEdge() + 1);
    }
  }

  /**
   * The dev events script (main thread): a back through the registered callback, so that it takes
   * the path of the system's. {@code what} is {@code pop}, {@code gesture} (start, progress and
   * commit), {@code cancel} (start, progress and cancel), {@code hold} (start and progress to one
   * half, the finger stays down) or {@code release} (cancels a held gesture).
   */
  void inject(String what) {
    boolean pop = what.equals("pop");
    boolean gesture = what.equals("gesture");
    boolean cancel = what.equals("cancel");
    boolean hold = what.equals("hold");
    boolean release = what.equals("release");
    if (!pop && !gesture && !cancel && !hold && !release) {
      Log.w(TAG, "unknown back event: " + what);
      return;
    }
    boolean predictive = Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE;
    if (registered == null) {
      Log.w(TAG, "back injected but the framework does not handle back");
      if (!cancel) {
        activity.onBackPressed();
      }
      return;
    }
    if (pop || !predictive) {
      if (!pop) {
        Log.w(TAG, "back gesture injected below API 34");
      }
      if (gesture || pop) {
        registered.onBackInvoked();
      }
      return;
    }
    final GestureCallback callback = (GestureCallback) registered;
    if (release) {
      callback.onBackCancelled();
      return;
    }
    final boolean commit = what.equals("gesture");
    callback.onBackStarted(new BackEvent(20f, 1200f, 0f, BackEvent.EDGE_LEFT));
    float[] steps = commit ? new float[] {0.3f, 0.6f} : hold ? new float[] {0.25f, 0.5f} : new float[] {0.4f};
    long delay = 0;
    for (final float progress : steps) {
      delay += GESTURE_STEP_MS;
      handler.postDelayed(
          new Runnable() {
            @Override
            public void run() {
              callback.onBackProgressed(new BackEvent(20f, 1200f, progress, BackEvent.EDGE_LEFT));
            }
          },
          delay);
    }
    if (hold) {
      return;
    }
    handler.postDelayed(
        new Runnable() {
          @Override
          public void run() {
            if (commit) {
              callback.onBackInvoked();
            } else {
              callback.onBackCancelled();
            }
          }
        },
        delay + GESTURE_STEP_MS);
  }
}
