#pragma once

#include "a21_firmware_state.h"

#include <stdint.h>

struct A21RGBColor {
  uint8_t r;
  uint8_t g;
  uint8_t b;
};

struct A21RGBDriver {
  void* ctx;
  bool (*write)(void* ctx, A21RGBColor color);
};

struct A21RGBRuntime {
  bool has_color;
  A21RGBColor last_color;
  uint32_t applied_count;
};

inline A21RGBColor a21RGBColorMake(uint8_t r, uint8_t g, uint8_t b) {
  return A21RGBColor{r, g, b};
}

inline bool a21RGBColorEquals(A21RGBColor left, A21RGBColor right) {
  return left.r == right.r && left.g == right.g && left.b == right.b;
}

inline void a21InitRGBRuntime(A21RGBRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->has_color = false;
  runtime->last_color = a21RGBColorMake(0, 0, 0);
  runtime->applied_count = 0;
}

inline A21RGBColor a21RGBForRenderState(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
      return a21RGBColorMake(0, 48, 16);
    case A21_RENDER_THINKING:
      return a21RGBColorMake(48, 32, 0);
    case A21_RENDER_SPEAKING:
      return a21RGBColorMake(0, 36, 48);
    case A21_RENDER_INTERRUPTED:
      return a21RGBColorMake(64, 24, 0);
    case A21_RENDER_PROFESSIONAL:
      return a21RGBColorMake(0, 16, 64);
    case A21_RENDER_LOCAL:
      return a21RGBColorMake(8, 8, 8);
    case A21_RENDER_ERROR:
      return a21RGBColorMake(64, 0, 0);
    case A21_RENDER_IDLE:
    default:
      return a21RGBColorMake(16, 16, 16);
  }
}

inline bool a21RGBRuntimeApplyState(A21RGBRuntime* runtime, A21RGBDriver* driver, const A21FirmwareState* state) {
  if (runtime == nullptr || driver == nullptr || driver->write == nullptr || state == nullptr) {
    return false;
  }
  const A21RGBColor target = a21RGBForRenderState(state->render_state);
  if (runtime->has_color && a21RGBColorEquals(runtime->last_color, target)) {
    return true;
  }
  if (!driver->write(driver->ctx, target)) {
    return false;
  }
  runtime->has_color = true;
  runtime->last_color = target;
  runtime->applied_count += 1;
  return true;
}
