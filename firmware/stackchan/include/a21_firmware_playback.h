#pragma once

#include "a21_firmware_state.h"

#include <stdint.h>

struct A21PlaybackDriver {
  void* ctx;
  bool (*start)(void* ctx, const char* stream_id);
  bool (*stop)(void* ctx, const char* reason);
  bool (*clear)(void* ctx);
};

struct A21PlaybackRuntime {
  bool playing;
  char active_stream_id[A21_STREAM_ID_CAP];
  uint32_t start_count;
  uint32_t stop_count;
  uint32_t clear_count;
};

inline void a21InitPlaybackRuntime(A21PlaybackRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->playing = false;
  a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, "");
  runtime->start_count = 0;
  runtime->stop_count = 0;
  runtime->clear_count = 0;
}

inline bool a21PlaybackDriverReady(const A21PlaybackDriver* driver) {
  return driver != nullptr && driver->start != nullptr && driver->stop != nullptr && driver->clear != nullptr;
}

inline const char* a21PlaybackStopReasonForRenderState(A21RenderState state) {
  switch (state) {
    case A21_RENDER_INTERRUPTED:
      return "barge_in";
    case A21_RENDER_ERROR:
      return "error";
    case A21_RENDER_LOCAL:
      return "local_fallback";
    case A21_RENDER_IDLE:
      return "idle";
    case A21_RENDER_LISTENING:
      return "listening";
    case A21_RENDER_THINKING:
      return "thinking";
    case A21_RENDER_PROFESSIONAL:
      return "professional";
    case A21_RENDER_SPEAKING:
    default:
      return "state_change";
  }
}

inline bool a21PlaybackStopAndClear(A21PlaybackRuntime* runtime, const A21PlaybackDriver* driver, const char* reason) {
  if (runtime == nullptr || !a21PlaybackDriverReady(driver) || reason == nullptr) {
    return false;
  }
  if (!driver->stop(driver->ctx, reason)) {
    return false;
  }
  runtime->stop_count += 1;
  if (!driver->clear(driver->ctx)) {
    return false;
  }
  runtime->clear_count += 1;
  runtime->playing = false;
  a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, "");
  return true;
}

inline bool a21PlaybackRuntimeApplyState(A21PlaybackRuntime* runtime, const A21PlaybackDriver* driver, const A21FirmwareState* state) {
  if (runtime == nullptr || !a21PlaybackDriverReady(driver) || state == nullptr) {
    return false;
  }

  if (state->render_state != A21_RENDER_SPEAKING) {
    if (!runtime->playing) {
      return true;
    }
    return a21PlaybackStopAndClear(runtime, driver, a21PlaybackStopReasonForRenderState(state->render_state));
  }

  if (state->stream_id[0] == '\0') {
    if (!runtime->playing) {
      return true;
    }
    return a21PlaybackStopAndClear(runtime, driver, "missing_stream");
  }

  if (runtime->playing && a21StringEquals(runtime->active_stream_id, state->stream_id)) {
    return true;
  }

  if (runtime->playing && !a21PlaybackStopAndClear(runtime, driver, "replace_stream")) {
    return false;
  }

  if (!driver->start(driver->ctx, state->stream_id)) {
    return false;
  }
  runtime->playing = true;
  a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, state->stream_id);
  runtime->start_count += 1;
  return true;
}
