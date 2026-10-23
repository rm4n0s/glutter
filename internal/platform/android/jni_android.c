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

static JavaVM *g_vm;
static jclass g_activity_class;       // global reference to GlutterActivity
static jmethodID g_on_main_ended;     // GlutterActivity.onMainEnded(int)

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
  if (g_on_main_ended == NULL) {
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
