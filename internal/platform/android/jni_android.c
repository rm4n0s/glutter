#include "jni_android.h"

#include <android/native_window_jni.h>
#include <stdint.h>
#include <stdlib.h>
#include <unistd.h>

// Implemented in Go (main_android.go).
extern void goSetenv(char *key, char *value);
extern void goStart(void);
extern void goSurfaceCreated(uintptr_t window);
extern void goSurfaceChanged(int width, int height, float density, float refresh, int displayWidth, int displayHeight);
extern void goSurfaceDestroyed(void);
extern void goLifecycle(int state);
extern void goFocus(int focused);
extern void goMotionEvent(long long *meta, int nmeta, int *ids, int nids, float *vals, int nvals, long long *times, int ntimes,
                          float pressureMin, float pressureMax, float hScroll, float vScroll);
extern void goKeyEvent(int id, int action, int keyCode, int scanCode, int metaState, int repeatCount, long long eventTime,
                       int unicodeChar, int source, int deviceId);

static JavaVM *g_vm;
static jclass g_activity_class;       // global reference to GlutterActivity
static jmethodID g_on_main_ended;     // GlutterActivity.onMainEnded(int)
static jmethodID g_on_key_handled;    // GlutterActivity.onKeyHandled(int, boolean)
static jmethodID g_set_cursor;        // GlutterActivity.setCursor(int)
static jmethodID g_inject;            // GlutterActivity.inject(int, float x4, int, String)

JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM *vm, void *reserved) {
  JNIEnv *env;
  g_vm = vm;
  if ((*vm)->GetEnv(vm, (void **)&env, JNI_VERSION_1_6) != JNI_OK) {
    return JNI_ERR;
  }
  jclass cls = (*env)->FindClass(env, "io/github/rm4n0s/glutter/GlutterActivity");
  if (cls == NULL) {
    return JNI_ERR;
  }
  g_activity_class = (*env)->NewGlobalRef(env, cls);
  g_on_main_ended = (*env)->GetStaticMethodID(env, cls, "onMainEnded", "(I)V");
  g_on_key_handled = (*env)->GetStaticMethodID(env, cls, "onKeyHandled", "(IZ)V");
  g_set_cursor = (*env)->GetStaticMethodID(env, cls, "setCursor", "(I)V");
  g_inject = (*env)->GetStaticMethodID(env, cls, "inject", "(IFFFFILjava/lang/String;)V");
  if (g_on_main_ended == NULL || g_on_key_handled == NULL || g_set_cursor == NULL || g_inject == NULL) {
    return JNI_ERR;
  }
  return JNI_VERSION_1_6;
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_setenv(JNIEnv *env, jclass cls, jstring key, jstring value) {
  const char *k = (*env)->GetStringUTFChars(env, key, NULL);
  const char *v = (*env)->GetStringUTFChars(env, value, NULL);
  if (k != NULL && v != NULL) {
    goSetenv((char *)k, (char *)v);
  }
  if (k != NULL) (*env)->ReleaseStringUTFChars(env, key, k);
  if (v != NULL) (*env)->ReleaseStringUTFChars(env, value, v);
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_start(JNIEnv *env, jclass cls, jobject activity) {
  goStart();
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_surfaceCreated(JNIEnv *env, jclass cls, jobject surface) {
  // ANativeWindow_fromSurface acquires a reference, which the UI thread releases (SPEC §24.4).
  ANativeWindow *window = ANativeWindow_fromSurface(env, surface);
  if (window != NULL) goSurfaceCreated((uintptr_t)window);
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_surfaceChanged(JNIEnv *env, jclass cls, jint width, jint height,
                                                                         jfloat density, jfloat refresh, jint displayWidth, jint displayHeight) {
  goSurfaceChanged(width, height, density, refresh, displayWidth, displayHeight);
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_surfaceDestroyed(JNIEnv *env, jclass cls) { goSurfaceDestroyed(); }

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_lifecycle(JNIEnv *env, jclass cls, jint state) { goLifecycle(state); }

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_focus(JNIEnv *env, jclass cls, jboolean focused) { goFocus(focused ? 1 : 0); }

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_motionEvent(JNIEnv *env, jclass cls, jlongArray meta, jintArray ids,
                                                                      jfloatArray vals, jlongArray times, jfloat pressureMin,
                                                                      jfloat pressureMax, jfloat hScroll, jfloat vScroll) {
  jsize nmeta = (*env)->GetArrayLength(env, meta), nids = (*env)->GetArrayLength(env, ids);
  jsize nvals = (*env)->GetArrayLength(env, vals), ntimes = (*env)->GetArrayLength(env, times);
  jlong *m = (*env)->GetLongArrayElements(env, meta, NULL);
  jint *i = (*env)->GetIntArrayElements(env, ids, NULL);
  jfloat *v = (*env)->GetFloatArrayElements(env, vals, NULL);
  jlong *t = (*env)->GetLongArrayElements(env, times, NULL);
  if (m != NULL && i != NULL && v != NULL && t != NULL) {
    goMotionEvent((long long *)m, nmeta, (int *)i, nids, (float *)v, nvals, (long long *)t, ntimes, pressureMin, pressureMax, hScroll,
                  vScroll);
  }
  // The arrays were only read: JNI_ABORT frees the copy without writing it back.
  if (m != NULL) (*env)->ReleaseLongArrayElements(env, meta, m, JNI_ABORT);
  if (i != NULL) (*env)->ReleaseIntArrayElements(env, ids, i, JNI_ABORT);
  if (v != NULL) (*env)->ReleaseFloatArrayElements(env, vals, v, JNI_ABORT);
  if (t != NULL) (*env)->ReleaseLongArrayElements(env, times, t, JNI_ABORT);
}

JNIEXPORT void JNICALL Java_io_github_rm4n0s_glutter_Native_keyEvent(JNIEnv *env, jclass cls, jint id, jint action, jint keyCode,
                                                                   jint scanCode, jint metaState, jint repeatCount, jlong eventTime,
                                                                   jint unicodeChar, jint source, jint deviceId) {
  goKeyEvent(id, action, keyCode, scanCode, metaState, repeatCount, eventTime, unicodeChar, source, deviceId);
}

// call_activity calls a static method of GlutterActivity that takes an int and, for the key answer,
// a boolean, from the calling thread: it attaches the thread to the JavaVM for the call and
// detaches it again. Returns 0 on success.
static int call_activity(jmethodID method, jint a, jboolean b, int withBool) {
  JNIEnv *env;
  int attached = 0;
  if (g_vm == NULL) return -1;
  if ((*g_vm)->GetEnv(g_vm, (void **)&env, JNI_VERSION_1_6) == JNI_EDETACHED) {
    if ((*g_vm)->AttachCurrentThread(g_vm, &env, NULL) != JNI_OK) return -1;
    attached = 1;
  }
  if (withBool) {
    (*env)->CallStaticVoidMethod(env, g_activity_class, method, a, b);
  } else {
    (*env)->CallStaticVoidMethod(env, g_activity_class, method, a);
  }
  int failed = (*env)->ExceptionCheck(env) ? -1 : 0;
  if (failed) (*env)->ExceptionClear(env);
  if (attached) (*g_vm)->DetachCurrentThread(g_vm);
  return failed;
}

int glutter_key_handled(int id, int handled) { return call_activity(g_on_key_handled, id, handled ? JNI_TRUE : JNI_FALSE, 1); }

int glutter_set_cursor(int type) { return call_activity(g_set_cursor, type, JNI_FALSE, 0); }

int glutter_inject(int kind, float x1, float y1, float x2, float y2, int ms, const char *key) {
  JNIEnv *env;
  int attached = 0;
  if (g_vm == NULL) return -1;
  if ((*g_vm)->GetEnv(g_vm, (void **)&env, JNI_VERSION_1_6) == JNI_EDETACHED) {
    if ((*g_vm)->AttachCurrentThread(g_vm, &env, NULL) != JNI_OK) return -1;
    attached = 1;
  }
  jstring name = (*env)->NewStringUTF(env, key);
  (*env)->CallStaticVoidMethod(env, g_activity_class, g_inject, kind, x1, y1, x2, y2, ms, name);
  (*env)->DeleteLocalRef(env, name);
  int failed = (*env)->ExceptionCheck(env) ? -1 : 0;
  if (failed) (*env)->ExceptionClear(env);
  if (attached) (*g_vm)->DetachCurrentThread(g_vm);
  return failed;
}

int glutter_main_ended(int code) {
  JNIEnv *env;
  int attached = 0;
  if (g_vm == NULL) return -1;
  if ((*g_vm)->GetEnv(g_vm, (void **)&env, JNI_VERSION_1_6) == JNI_EDETACHED) {
    if ((*g_vm)->AttachCurrentThread(g_vm, &env, NULL) != JNI_OK) return -1;
    attached = 1;
  }
  (*env)->CallStaticVoidMethod(env, g_activity_class, g_on_main_ended, (jint)code);
  int failed = (*env)->ExceptionCheck(env) ? -1 : 0;
  if (failed) (*env)->ExceptionClear(env);
  if (attached) (*g_vm)->DetachCurrentThread(g_vm);
  return failed;
}

void glutter_log(int priority, const char *tag, const char *line) {
  __android_log_write(priority, tag, line);
}

int glutter_redirect_stdio(void) {
  int fds[2];
  if (pipe(fds) != 0) return -1;
  if (dup2(fds[1], 1) < 0 || dup2(fds[1], 2) < 0) return -1;
  close(fds[1]);
  return fds[0];
}
