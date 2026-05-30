#pragma once

#include "a21_firmware_config.h"
#include "a21_firmware_state.h"

#include <stdint.h>

struct A21MotionDriver {
  void* ctx;
  bool (*write_y)(void* ctx, int y_deg);
};

struct A21MotionRuntime {
  bool has_y;
  int last_y_deg;
  uint32_t applied_count;
};

inline void a21InitMotionRuntime(A21MotionRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->has_y = false;
  runtime->last_y_deg = 45;
  runtime->applied_count = 0;
}

inline int a21MotionYForRenderState(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
    case A21_RENDER_INTERRUPTED:
      return a21ClampServoY(38);
    case A21_RENDER_THINKING:
      return a21ClampServoY(52);
    case A21_RENDER_SPEAKING:
      return a21ClampServoY(48);
    case A21_RENDER_IDLE:
    case A21_RENDER_PROFESSIONAL:
    case A21_RENDER_LOCAL:
    case A21_RENDER_ERROR:
    default:
      return a21ClampServoY(45);
  }
}

inline bool a21MotionRuntimeApplyState(A21MotionRuntime* runtime, A21MotionDriver* driver, const A21FirmwareState* state) {
  if (runtime == nullptr || driver == nullptr || driver->write_y == nullptr || state == nullptr) {
    return false;
  }
  const int target_y = a21MotionYForRenderState(state->render_state);
  if (runtime->has_y && runtime->last_y_deg == target_y) {
    return true;
  }
  if (!driver->write_y(driver->ctx, target_y)) {
    return false;
  }
  runtime->has_y = true;
  runtime->last_y_deg = target_y;
  runtime->applied_count += 1;
  return true;
}
