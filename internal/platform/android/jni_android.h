#ifndef GLUTTER_JNI_ANDROID_H
#define GLUTTER_JNI_ANDROID_H

#include <android/log.h>
#include <stdint.h>
#include <jni.h>

// glutter_main_ended tells the Java side that main returned, from the thread that ran it:
// it attaches the thread to the JavaVM for the call and detaches it again. Returns 0 on success.
int glutter_main_ended(int code);

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
