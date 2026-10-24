package io.github.rm4n0s.glutter;

import android.content.Context;
import android.os.Build;
import android.view.textservice.SentenceSuggestionsInfo;
import android.view.textservice.SpellCheckerSession;
import android.view.textservice.SuggestionsInfo;
import android.view.textservice.TextInfo;
import android.view.textservice.TextServicesManager;
import java.nio.charset.StandardCharsets;
import java.util.Locale;

/**
 * The spell checker of the system (spellcheck.go of the Go side): it asks {@code
 * TextServicesManager}'s spell checker service for the misspelled words of a text and sends back
 * their ranges (UTF-16 offsets) with the suggestions.
 *
 * <p>Ports: {@code SpellCheckPlugin}
 * (shell/platform/android/io/flutter/plugin/editing/SpellCheckPlugin.java).
 */
final class SpellCheck implements SpellCheckerSession.SpellCheckerSessionListener {
  /** A is the number of the request, S is the locale, a line feed and the text. */
  static final int OP_SPELL_CHECK = 24;

  /**
   * The maximum number of suggestions that the service gives for a word: what Android's TextViews
   * use.
   */
  private static final int MAX_SPELL_CHECK_SUGGESTIONS = 5;

  /** The answer's codes of {@code Native.spellCheckResult}. */
  private static final int CODE_OK = 0;

  private static final int CODE_FAILED = 1;

  private final Context context;
  private final TextServicesManager manager;
  private SpellCheckerSession session;

  /** The number of the request that waits for the service, or 0. */
  private int pending;

  SpellCheck(Context context) {
    this.context = context;
    this.manager = (TextServicesManager) context.getSystemService(Context.TEXT_SERVICES_MANAGER_SERVICE);
  }

  /**
   * Whether a spell checker can be asked: the user has not turned the spell checker off (API 31
   * and later says; before that, a request that cannot get a session fails).
   */
  boolean supported() {
    if (manager == null) {
      return false;
    }
    return Build.VERSION.SDK_INT < 31 || manager.isSpellCheckerEnabled();
  }

  /** Tells Go whether a spell checker can be asked. */
  void report() {
    Native.spellCheckSupport(supported());
  }

  /** The command of Go (main thread). */
  void command(int op, long id, String s) {
    if (op != OP_SPELL_CHECK) {
      return;
    }
    int cut = s.indexOf('\n');
    String locale = cut < 0 ? "" : s.substring(0, cut);
    String text = cut < 0 ? s : s.substring(cut + 1);
    initiateSpellCheck((int) id, locale, text);
  }

  /** Asks the spell checker, if no request waits for it. */
  private void initiateSpellCheck(int id, String locale, String text) {
    if (pending != 0) {
      fail(id, "Previous spell check request still pending.");
      return;
    }
    if (session == null) {
      Locale tag = locale.isEmpty() ? Locale.getDefault() : Locale.forLanguageTag(locale.replace('_', '-'));
      session =
          manager == null ? null : manager.newSpellCheckerSession(null, tag, this, /* referToSpellCheckerLanguageSettings= */ true);
      if (session == null) {
        fail(id, "The system has no spell checker for " + tag.toLanguageTag() + ".");
        return;
      }
    }
    pending = id;
    session.getSentenceSuggestions(new TextInfo[] {new TextInfo(text)}, MAX_SPELL_CHECK_SUGGESTIONS);
  }

  /**
   * The result of the service, as lines of {@code start}, {@code end} and the suggestions, separated
   * by tabs (UTF-16 offsets).
   */
  @Override
  public void onGetSentenceSuggestions(SentenceSuggestionsInfo[] results) {
    int id = pending;
    pending = 0;
    if (id == 0) {
      return;
    }
    StringBuilder out = new StringBuilder();
    if (results != null && results.length > 0 && results[0] != null) {
      SentenceSuggestionsInfo info = results[0];
      for (int i = 0; i < info.getSuggestionsCount(); i++) {
        SuggestionsInfo suggestions = info.getSuggestionsInfoAt(i);
        int count = suggestions.getSuggestionsCount();
        if (count <= 0) {
          continue;
        }
        StringBuilder line = new StringBuilder();
        boolean valid = false;
        for (int j = 0; j < count; j++) {
          String suggestion = suggestions.getSuggestionAt(j);
          // Some services (Samsung's) answer with empty suggestions: the span is left out.
          if (!suggestion.isEmpty()) {
            valid = true;
            line.append('\t').append(suggestion.replace('\t', ' ').replace('\n', ' '));
          }
        }
        if (!valid) {
          continue;
        }
        int start = info.getOffsetAt(i);
        out.append(start).append('\t').append(start + info.getLengthAt(i)).append(line).append('\n');
      }
    }
    Native.spellCheckResult(id, CODE_OK, out.toString().getBytes(StandardCharsets.UTF_8));
  }

  /** The deprecated callback; unused. */
  @Override
  public void onGetSuggestions(SuggestionsInfo[] results) {}

  private void fail(int id, String reason) {
    Native.spellCheckResult(id, CODE_FAILED, reason.getBytes(StandardCharsets.UTF_8));
  }

  /** The activity goes: the session is closed and a request that waits is given up. */
  void close() {
    if (session != null) {
      session.close();
      session = null;
    }
    pending = 0;
  }
}
