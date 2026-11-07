package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.content.Context;
import android.content.pm.PackageManager;
import android.util.Log;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

/**
 * The permission to record sound ({@code RECORD_AUDIO}) for the audio package's capture (audio.go in
 * internal/platform/android). Go sends a command for each ({@code GlutterActivity.command},
 * operations 29 and 30) and gets the answer in {@code Native.desktopResult}, as the desktop services
 * do. The permission has to be in the app's manifest ({@code glutter-android build -permission
 * android.permission.RECORD_AUDIO}): without it Android refuses the prompt. Everything here runs on
 * the main thread.
 *
 * <p>Ports: nothing of Flutter's embedding; Flutter apps ask for the microphone through plugins such
 * as permission_handler, whose Java side makes the same two calls ({@code checkSelfPermission} and
 * {@code requestPermissions}).
 */
final class AudioPermission {
  private static final String TAG = "GlutterJava";

  // The operations of GlutterActivity.command (audio.go).
  static final int OP_STATUS = 29;
  static final int OP_REQUEST = 30;

  /** The answers of {@code Native.desktopResult} (the same codes as the desktop services). */
  private static final int ANSWER_DENIED = 0;

  private static final int ANSWER_GRANTED = 1;
  private static final int ANSWER_UNDECIDED = 2;

  private static final String RECORD_AUDIO = "android.permission.RECORD_AUDIO";

  /** The request code of the prompt: it has to fit in 16 bits. */
  static final int REQUEST_CODE = 0x4e51;

  /** The numbers of the requests that wait for the prompt. Only the main thread. */
  private static final List<Integer> requests = new ArrayList<>();

  private AudioPermission() {}

  /** Does one operation. {@code activity} is the activity that is shown, or null. */
  static void command(Activity activity, Context context, int op, long a) {
    int request = (int) a;
    switch (op) {
      case OP_STATUS:
        if (context == null) {
          answer(request, ANSWER_UNDECIDED, "no context");
          return;
        }
        boolean granted = granted(context);
        Log.i(TAG, "audio permission status id=" + request + " granted=" + granted);
        answer(request, granted ? ANSWER_GRANTED : ANSWER_DENIED, "");
        break;
      case OP_REQUEST:
        request(activity, request);
        break;
      default:
        break;
    }
  }

  private static boolean granted(Context context) {
    return context.checkSelfPermission(RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED;
  }

  /**
   * Shows the system's prompt. The answer comes in {@link #permissionResult}. Requests that come
   * while a prompt is shown wait for the same answer.
   */
  private static void request(Activity activity, int request) {
    if (activity == null) {
      Log.i(TAG, "audio permission id=" + request + " result=noactivity");
      answer(request, ANSWER_UNDECIDED, "no activity is shown to ask the user from");
      return;
    }
    if (granted(activity)) {
      answer(request, ANSWER_GRANTED, "");
      return;
    }
    requests.add(request);
    if (requests.size() == 1) {
      Log.i(TAG, "audio permission id=" + request + " asked");
      try {
        activity.requestPermissions(new String[] {RECORD_AUDIO}, REQUEST_CODE);
      } catch (RuntimeException e) {
        requests.clear();
        Log.w(TAG, "the permission could not be asked for", e);
        answer(request, ANSWER_UNDECIDED, String.valueOf(e));
      }
    }
  }

  /** The user answered the prompt (GlutterActivity.onRequestPermissionsResult). */
  static void permissionResult(int requestCode, int[] grantResults) {
    if (requestCode != REQUEST_CODE) {
      return;
    }
    int code;
    String text = "";
    if (grantResults == null || grantResults.length == 0) {
      // The prompt was interrupted: the user did not decide.
      code = ANSWER_UNDECIDED;
      text = "the permission prompt was interrupted";
    } else if (grantResults[0] == PackageManager.PERMISSION_GRANTED) {
      code = ANSWER_GRANTED;
    } else {
      code = ANSWER_DENIED;
    }
    List<Integer> waiting = new ArrayList<>(requests);
    requests.clear();
    for (int request : waiting) {
      Log.i(TAG, "audio permission id=" + request + " result=" + code);
      answer(request, code, text);
    }
  }

  private static void answer(int request, int code, String text) {
    Native.desktopResult(request, code, text.getBytes(StandardCharsets.UTF_8));
  }
}
