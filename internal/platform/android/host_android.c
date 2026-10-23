#include "jni_android.h"

#include <android/choreographer.h>
#include <android/looper.h>
#include <android/native_window_jni.h>
#include <dlfcn.h>
#include <errno.h>
#include <stdint.h>
#include <sys/eventfd.h>
#include <unistd.h>

// The state of the UI thread's looper. Only the UI thread touches it, except the eventfd, which
// any thread writes.
static ALooper *g_looper;
static AChoreographer *g_choreographer;
static int g_wake_fd = -1;
static int g_vsync_fired;
static long long g_vsync_ns;

// The 64-bit frame callback is API 29; the older one is API 24. Which one exists is found at run
// time, since the minimum API level is 26.
typedef void (*frame_callback64)(int64_t frameTimeNanos, void *data);
typedef void (*post_callback64_fn)(AChoreographer *, frame_callback64, void *);
static post_callback64_fn g_post64;

static int wake_callback(int fd, int events, void *data) {
  uint64_t v;
  while (read(fd, &v, sizeof v) < 0 && errno == EINTR) {
  }
  return 1;  // keep watching
}

static void on_frame64(int64_t frameTimeNanos, void *data) {
  g_vsync_fired = 1;
  g_vsync_ns = frameTimeNanos;
}

static void on_frame32(long frameTimeNanos, void *data) {
  g_vsync_fired = 1;
  g_vsync_ns = frameTimeNanos;
}

int glutter_host_init(void) {
  g_looper = ALooper_prepare(ALOOPER_PREPARE_ALLOW_NON_CALLBACKS);
  if (g_looper == NULL) return -1;
  g_wake_fd = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
  if (g_wake_fd < 0) return -1;
  if (ALooper_addFd(g_looper, g_wake_fd, ALOOPER_POLL_CALLBACK, ALOOPER_EVENT_INPUT, wake_callback, NULL) != 1) return -1;
  g_choreographer = AChoreographer_getInstance();
  if (g_choreographer == NULL) return -1;
  g_post64 = (post_callback64_fn)dlsym(RTLD_DEFAULT, "AChoreographer_postFrameCallback64");
  return 0;
}

int glutter_host_poll(long long *t) {
  ALooper_pollOnce(-1, NULL, NULL, NULL);
  if (g_vsync_fired) {
    g_vsync_fired = 0;
    *t = g_vsync_ns;
    return 1;
  }
  return 0;
}

void glutter_host_wake(void) {
  uint64_t one = 1;
  if (g_wake_fd >= 0) {
    // A full counter means a wakeup is pending already.
    ssize_t n = write(g_wake_fd, &one, sizeof one);
    (void)n;
  }
}

void glutter_host_request_vsync(void) {
  if (g_post64 != NULL) {
    g_post64(g_choreographer, on_frame64, NULL);
  } else {
    AChoreographer_postFrameCallback(g_choreographer, on_frame32, NULL);
  }
}

void glutter_release_window(uintptr_t window) { ANativeWindow_release((ANativeWindow *)window); }
