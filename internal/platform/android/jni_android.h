#ifndef GLUTTER_JNI_ANDROID_H
#define GLUTTER_JNI_ANDROID_H

#include <android/log.h>
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
