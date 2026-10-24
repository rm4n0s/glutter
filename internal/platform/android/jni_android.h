#ifndef GLUTTER_JNI_ANDROID_H
#define GLUTTER_JNI_ANDROID_H

#include <android/log.h>
#include <stdint.h>
#include <jni.h>

// glutter_main_ended tells the Java side that main returned, from the thread that ran it:
// it attaches the thread to the JavaVM for the call and detaches it again. Returns 0 on success.
int glutter_main_ended(int code);

// glutter_key_handled tells the activity what the framework made of a key event: an event that it
// did not handle is dispatched again to the view hierarchy. Returns 0 on success.
int glutter_key_handled(int id, int handled);

// glutter_set_cursor sets the mouse cursor, a PointerIcon type. Returns 0 on success.
int glutter_set_cursor(int type);

// glutter_inject asks the activity to inject a synthetic event of the dev events script (kind 1 tap
// at x1,y1; 2 swipe from x1,y1 to x2,y2 in ms milliseconds; 3 the key of that name; 4 text to
// commit; 5 text to compose; 6 the keyboard's action key; 7 a back, the key being pop, gesture or cancel). The key is n bytes of UTF-8, with no terminating NUL.
int glutter_inject(int kind, float x1, float y1, float x2, float y2, int ms, const char *key, int n);

// glutter_command asks the activity to do one operation of the system's interface, feedback or screen
// (systemui.go) or of text input and the clipboard (textinput.go): op with up to four integers and a
// string of n bytes of UTF-8, with no terminating NUL. Returns 0 on success.
int glutter_command(int op, long long a, long long b, long long c, long long d, const char *s, int n);

// glutter_scaled_font_size returns what the system makes of a font size in logical pixels
// (the sp unit of Android), or the size itself when the call fails.
float glutter_scaled_font_size(float size);

// glutter_log writes one line to logcat. The priority is an ANDROID_LOG_* value.
void glutter_log(int priority, const char *tag, const char *line);

// glutter_redirect_stdio makes standard output and standard error write into a pipe, and
// returns the read end (or -1). The caller reads lines from it.
int glutter_redirect_stdio(void);

#endif

// The looper of the UI thread (SPEC §24.3).

// glutter_host_init prepares the calling thread's ALooper, an eventfd the looper watches, and
// the thread's AChoreographer. It returns 0 on success.
int glutter_host_init(void);

// glutter_host_poll blocks the calling thread until the eventfd was written or a frame callback
// came. It returns 1 when a frame callback came, with its vsync time in nanoseconds in *t.
int glutter_host_poll(long long *t);

// glutter_host_wake writes the eventfd; it is for any thread.
void glutter_host_wake(void);

// glutter_host_request_vsync asks the choreographer for one frame callback.
void glutter_host_request_vsync(void);

// glutter_release_window lets go of an ANativeWindow that was acquired.
void glutter_release_window(uintptr_t window);
