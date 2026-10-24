package io.github.rm4n0s.glutter;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

/**
 * Receives the delete intent of a notification: the system sends it when the user swipes the
 * notification away or clears them all. The process is alive when it matters (Go keeps the
 * callbacks of the notification), so the event goes to Go as it is; an intent that an earlier
 * process posted is ignored ({@link DesktopServices#handleIntent}).
 *
 * <p>Ports: nothing of Flutter's embedding; {@code flutter_local_notifications} has the
 * {@code DismissedReceiver} for the same purpose.
 */
public final class NotificationReceiver extends BroadcastReceiver {
  @Override
  public void onReceive(Context context, Intent intent) {
    DesktopServices.handleIntent(intent);
  }
}
