package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ActivityNotFoundException;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.graphics.drawable.Icon;
import android.net.Uri;
import android.os.Build;
import android.util.Log;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

/**
 * The desktop services of package {@code desktop} on Android: opening a URI with an {@code
 * ACTION_VIEW} intent and notifications through the {@code NotificationManager}, on one channel per
 * urgency (desktop.go). Go sends a command for each ({@code GlutterActivity.command}, operations 19
 * to 23) and gets the answer in {@code Native.desktopResult}; the clicks on a notification and its
 * buttons come back in {@code Native.notificationEvent}. Everything here runs on the main thread.
 *
 * <p>Ports: nothing of Flutter's embedding; Flutter apps use the url_launcher_android and
 * flutter_local_notifications plugins, whose {@code UrlLauncher.openUrl} and {@code
 * FlutterLocalNotificationsPlugin.show} this follows in what it asks of the system.
 */
final class DesktopServices {
  private static final String TAG = "GlutterJava";

  // The operations of GlutterActivity.command (desktop.go).
  static final int OP_OPEN = 19;
  static final int OP_STATUS = 20;
  static final int OP_PERMISSION = 21;
  static final int OP_NOTIFY = 22;
  static final int OP_CANCEL = 23;

  /** The answers of {@code Native.desktopResult} (desktop.go has the meaning of each). */
  private static final int ANSWER_OK = 0;

  private static final int ANSWER_FAILED = 1;
  private static final int ANSWER_NO_APP = 2;
  private static final int PERMISSION_DENIED = 0;
  private static final int PERMISSION_GRANTED = 1;
  private static final int PERMISSION_UNDECIDED = 2;

  /** The bits of the answer to {@code OP_STATUS}. */
  private static final int STATUS_NEEDED = 1;

  private static final int STATUS_GRANTED = 2;
  private static final int STATUS_ENABLED = 4;

  /** The kinds of {@code Native.notificationEvent}. */
  private static final int EVENT_CLICK = 1;

  private static final int EVENT_ACTION = 2;
  private static final int EVENT_CLOSED = 3;

  /** The runtime permission of Android 13 and later (API 33). */
  private static final String POST_NOTIFICATIONS = "android.permission.POST_NOTIFICATIONS";

  private static final int API_POST_NOTIFICATIONS = 33;

  /** The request code of the permission prompt: it has to fit in 16 bits. */
  static final int REQUEST_CODE_PERMISSION = 0x4e50;

  /** The action of the intents of a notification, so that they are recognized. */
  static final String ACTION_NOTIFICATION = "io.github.rm4n0s.glutter.NOTIFICATION";

  static final String EXTRA_PROCESS = "io.github.rm4n0s.glutter.process";
  static final String EXTRA_SERIAL = "io.github.rm4n0s.glutter.serial";
  static final String EXTRA_KIND = "io.github.rm4n0s.glutter.kind";
  static final String EXTRA_ACTION = "io.github.rm4n0s.glutter.action";

  /**
   * Tells the processes apart: the intents of a notification that an earlier process posted come
   * back to this one when the user taps them, and their serials mean nothing here.
   */
  private static final long PROCESS = System.currentTimeMillis();

  /**
   * How long the system keeps a notification past its timeout. Go closes it at the timeout and
   * reports it; the system's timeout is the backstop if the process is frozen.
   */
  private static final long TIMEOUT_GRACE_MS = 2000;

  /** The channels, by urgency. */
  private static final String CHANNEL_LOW = "glutter_low";

  private static final String CHANNEL_NORMAL = "glutter_normal";
  private static final String CHANNEL_CRITICAL = "glutter_critical";

  /** The numbers of the requests that wait for the permission prompt. Only the main thread. */
  private static final List<Integer> permissionRequests = new ArrayList<>();

  private DesktopServices() {}

  /**
   * Does one operation of the desktop services. {@code activity} is the activity that is shown, or
   * null.
   */
  static void command(
      Activity activity, Context context, int op, long a, long b, long c, long d, byte[] bytes) {
    try {
      switch (op) {
        case OP_OPEN:
          open(activity, (int) a, new String(bytes, StandardCharsets.UTF_8));
          break;
        case OP_STATUS:
          status(context, (int) a);
          break;
        case OP_PERMISSION:
          requestPermission(activity, (int) a);
          break;
        case OP_NOTIFY:
          notify(context, (int) a, (int) b, (int) c, d, bytes);
          break;
        case OP_CANCEL:
          cancel(context, (int) b);
          break;
        default:
          Log.w(TAG, "unknown desktop command " + op);
      }
    } catch (RuntimeException e) {
      // The answer is owed to Go: a request must not stay unanswered.
      Log.w(TAG, "desktop command " + op + " failed", e);
      if (op != OP_CANCEL) {
        answer((int) a, op == OP_PERMISSION ? PERMISSION_UNDECIDED : ANSWER_FAILED, String.valueOf(e));
      }
    }
  }

  private static void answer(int request, int code, String text) {
    Native.desktopResult(request, code, text.getBytes(StandardCharsets.UTF_8));
  }

  /** {@code ACTION_VIEW} for the URI. No app that handles it, or no activity, is an answer. */
  private static void open(Activity activity, int request, String uri) {
    if (activity == null) {
      Log.i(TAG, "desktop open id=" + request + " result=noactivity uri=" + uri);
      answer(request, ANSWER_FAILED, "no activity is shown to start the intent from");
      return;
    }
    try {
      activity.startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(uri)));
      Log.i(TAG, "desktop open id=" + request + " result=started uri=" + uri);
      answer(request, ANSWER_OK, "");
    } catch (ActivityNotFoundException e) {
      Log.i(TAG, "desktop open id=" + request + " result=noapp uri=" + uri);
      answer(request, ANSWER_NO_APP, "no app can open " + uri);
    } catch (RuntimeException e) {
      // For example the exposure of a file: URI, which Android refuses to hand to another app.
      Log.i(TAG, "desktop open id=" + request + " result=failed uri=" + uri + " error=" + e);
      answer(request, ANSWER_FAILED, "Android refused the intent: " + e);
    }
  }

  private static boolean permissionNeeded() {
    return Build.VERSION.SDK_INT >= API_POST_NOTIFICATIONS;
  }

  private static boolean permissionGranted(Context context) {
    return !permissionNeeded()
        || context.checkSelfPermission(POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED;
  }

  private static NotificationManager manager(Context context) {
    return context.getSystemService(NotificationManager.class);
  }

  /** What the system says about the app's notifications now. */
  private static void status(Context context, int request) {
    boolean needed = permissionNeeded();
    boolean granted = permissionGranted(context);
    boolean enabled = manager(context).areNotificationsEnabled();
    Log.i(
        TAG,
        "desktop status id="
            + request
            + " needed="
            + needed
            + " granted="
            + granted
            + " enabled="
            + enabled);
    answer(
        request,
        (needed ? STATUS_NEEDED : 0) | (granted ? STATUS_GRANTED : 0) | (enabled ? STATUS_ENABLED : 0),
        "");
  }

  /**
   * Shows the system's prompt for {@code POST_NOTIFICATIONS}. The answer comes in {@link
   * #permissionResult}. Requests that come while a prompt is shown wait for the same answer.
   */
  private static void requestPermission(Activity activity, int request) {
    if (activity == null) {
      Log.i(TAG, "desktop permission id=" + request + " result=noactivity");
      answer(request, PERMISSION_UNDECIDED, "no activity is shown to ask the user from");
      return;
    }
    if (permissionGranted(activity)) {
      answer(request, PERMISSION_GRANTED, "");
      return;
    }
    permissionRequests.add(request);
    if (permissionRequests.size() == 1) {
      Log.i(TAG, "desktop permission id=" + request + " asked");
      try {
        activity.requestPermissions(new String[] {POST_NOTIFICATIONS}, REQUEST_CODE_PERMISSION);
      } catch (RuntimeException e) {
        permissionRequests.clear();
        Log.w(TAG, "the permission could not be asked for", e);
        answer(request, PERMISSION_UNDECIDED, String.valueOf(e));
      }
    }
  }

  /** The user answered the prompt (GlutterActivity.onRequestPermissionsResult). */
  static void permissionResult(int requestCode, int[] grantResults) {
    if (requestCode != REQUEST_CODE_PERMISSION) {
      return;
    }
    int code;
    String text = "";
    if (grantResults == null || grantResults.length == 0) {
      // The prompt was interrupted: the user did not decide.
      code = PERMISSION_UNDECIDED;
      text = "the permission prompt was interrupted";
    } else if (grantResults[0] == PackageManager.PERMISSION_GRANTED) {
      code = PERMISSION_GRANTED;
    } else {
      code = PERMISSION_DENIED;
    }
    List<Integer> waiting = new ArrayList<>(permissionRequests);
    permissionRequests.clear();
    for (int request : waiting) {
      Log.i(TAG, "desktop permission id=" + request + " result=" + code);
      answer(request, code, text);
    }
  }

  private static String channelOf(String urgency) {
    switch (urgency) {
      case "low":
        return CHANNEL_LOW;
      case "critical":
        return CHANNEL_CRITICAL;
      default:
        return CHANNEL_NORMAL;
    }
  }

  /** The channel, made the first time it is needed; the user can change its settings after that. */
  private static NotificationChannel channel(Context context, String urgency) {
    NotificationManager manager = manager(context);
    String id = channelOf(urgency);
    NotificationChannel channel = manager.getNotificationChannel(id);
    if (channel != null) {
      return channel;
    }
    CharSequence label = context.getApplicationInfo().loadLabel(context.getPackageManager());
    int importance;
    String name;
    switch (urgency) {
      case "low":
        importance = NotificationManager.IMPORTANCE_LOW;
        name = label + " (low priority)";
        break;
      case "critical":
        importance = NotificationManager.IMPORTANCE_HIGH;
        name = label + " (urgent)";
        break;
      default:
        importance = NotificationManager.IMPORTANCE_DEFAULT;
        name = String.valueOf(label);
        break;
    }
    channel = new NotificationChannel(id, name, importance);
    manager.createNotificationChannel(channel);
    return channel;
  }

  /** Reads the length-prefixed fields that Go wrote (encodeNotification in desktop.go). */
  private static List<String> fields(byte[] bytes) {
    List<String> out = new ArrayList<>();
    ByteBuffer buffer = ByteBuffer.wrap(bytes);
    while (buffer.remaining() >= 4) {
      int n = buffer.getInt();
      if (n < 0 || n > buffer.remaining()) {
        break;
      }
      byte[] field = new byte[n];
      buffer.get(field);
      out.add(new String(field, StandardCharsets.UTF_8));
    }
    return out;
  }

  /**
   * A PendingIntent to the activity for a click (index 0) or a button. The data makes each one its
   * own (an Intent's extras do not count when the system looks for an equal one).
   */
  private static PendingIntent activityIntent(
      Context context, int id, int serial, int index, String actionId) {
    Intent intent = new Intent(context, GlutterActivity.class);
    intent.setAction(ACTION_NOTIFICATION);
    intent.setData(Uri.parse("glutter-notification://" + id + "/" + serial + "/" + index));
    intent.addFlags(
        Intent.FLAG_ACTIVITY_NEW_TASK
            | Intent.FLAG_ACTIVITY_SINGLE_TOP
            | Intent.FLAG_ACTIVITY_CLEAR_TOP);
    intent.putExtra(EXTRA_PROCESS, PROCESS);
    intent.putExtra(EXTRA_SERIAL, serial);
    intent.putExtra(EXTRA_KIND, actionId == null ? EVENT_CLICK : EVENT_ACTION);
    if (actionId != null) {
      intent.putExtra(EXTRA_ACTION, actionId);
    }
    return PendingIntent.getActivity(
        context, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
  }

  /** The intent the system sends when the user swipes the notification away. */
  private static PendingIntent deleteIntent(Context context, int id, int serial) {
    Intent intent = new Intent(context, NotificationReceiver.class);
    intent.setAction(ACTION_NOTIFICATION);
    intent.setData(Uri.parse("glutter-notification://" + id + "/" + serial + "/delete"));
    intent.putExtra(EXTRA_PROCESS, PROCESS);
    intent.putExtra(EXTRA_SERIAL, serial);
    intent.putExtra(EXTRA_KIND, EVENT_CLOSED);
    return PendingIntent.getBroadcast(
        context, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
  }

  /**
   * Posts the notification: {@code id} is its ID (a notification with the ID of one that is shown
   * replaces it), {@code serial} tells it from the one it replaced, {@code timeoutMs} is the time it
   * stays (0 for none), and {@code bytes} the fields (urgency, title, body, then the ID and the
   * label of each button).
   */
  private static void notify(
      Context context, int request, int id, int serial, long timeoutMs, byte[] bytes) {
    List<String> f = fields(bytes);
    if (f.size() < 3) {
      answer(request, ANSWER_FAILED, "the notification has no title");
      return;
    }
    String urgency = f.get(0);
    NotificationManager manager = manager(context);
    if (!permissionGranted(context) || !manager.areNotificationsEnabled()) {
      Log.i(TAG, "desktop notify id=" + id + " serial=" + serial + " result=off");
      answer(request, ANSWER_NO_APP, "the notifications of the app are turned off");
      return;
    }
    NotificationChannel channel = channel(context, urgency);
    if (channel.getImportance() == NotificationManager.IMPORTANCE_NONE) {
      Log.i(TAG, "desktop notify id=" + id + " serial=" + serial + " result=channeloff");
      answer(request, ANSWER_NO_APP, "the notification channel " + channel.getId() + " is turned off");
      return;
    }
    int smallIcon = context.getApplicationInfo().icon;
    if (smallIcon == 0) {
      smallIcon = android.R.drawable.ic_dialog_info;
    }
    Notification.Builder builder =
        new Notification.Builder(context, channel.getId())
            .setSmallIcon(smallIcon)
            .setContentTitle(f.get(1))
            .setAutoCancel(true)
            .setOngoing(false)
            .setOnlyAlertOnce(true)
            .setContentIntent(activityIntent(context, id, serial, 0, null))
            .setDeleteIntent(deleteIntent(context, id, serial));
    String body = f.get(2);
    if (!body.isEmpty()) {
      builder.setContentText(body).setStyle(new Notification.BigTextStyle().bigText(body));
    }
    for (int i = 3; i + 1 < f.size(); i += 2) {
      builder.addAction(
          new Notification.Action.Builder(
                  Icon.createWithResource(context, smallIcon),
                  f.get(i + 1),
                  activityIntent(context, id, serial, 1 + (i - 3) / 2, f.get(i)))
              .build());
    }
    if (timeoutMs > 0) {
      builder.setTimeoutAfter(timeoutMs + TIMEOUT_GRACE_MS);
    }
    manager.notify(id, builder.build());
    Log.i(
        TAG,
        "desktop notify id="
            + id
            + " serial="
            + serial
            + " channel="
            + channel.getId()
            + " title="
            + f.get(1)
            + " actions="
            + ((f.size() - 3) / 2));
    answer(request, ANSWER_OK, "");
  }

  private static void cancel(Context context, int id) {
    manager(context).cancel(id);
    Log.i(TAG, "desktop cancel id=" + id);
  }

  /**
   * An intent that a notification sent: a click or a button (the activity's {@code onNewIntent} or
   * {@code onCreate}), or the swipe (the receiver). The intent of another process is ignored.
   */
  static void handleIntent(Intent intent) {
    if (intent == null || !ACTION_NOTIFICATION.equals(intent.getAction())) {
      return;
    }
    if (intent.getLongExtra(EXTRA_PROCESS, 0) != PROCESS) {
      return;
    }
    // Once: an activity that is made again must not hear the same click twice.
    intent.removeExtra(EXTRA_PROCESS);
    int serial = intent.getIntExtra(EXTRA_SERIAL, 0);
    int kind = intent.getIntExtra(EXTRA_KIND, 0);
    String action = intent.getStringExtra(EXTRA_ACTION);
    String text = kind == EVENT_CLOSED ? "dismissed" : (action == null ? "" : action);
    Log.i(
        TAG,
        "desktop notification event serial="
            + serial
            + " kind="
            + (kind == EVENT_CLICK ? "click" : kind == EVENT_ACTION ? "action" : "closed")
            + " text="
            + text);
    try {
      Native.notificationEvent(serial, kind, text.getBytes(StandardCharsets.UTF_8));
    } catch (UnsatisfiedLinkError e) {
      // The Go program is not loaded in this process (the receiver started it): nobody listens.
      Log.w(TAG, "a notification event arrived before the program started");
    }
  }
}
