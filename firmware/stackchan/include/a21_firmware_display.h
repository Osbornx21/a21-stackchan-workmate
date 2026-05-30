#pragma once

#include "a21_firmware_state.h"

inline const char* a21DisplayStateLabel(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
      return "LISTENING";
    case A21_RENDER_THINKING:
      return "THINKING";
    case A21_RENDER_SPEAKING:
      return "SPEAKING";
    case A21_RENDER_INTERRUPTED:
      return "INTERRUPTED";
    case A21_RENDER_PROFESSIONAL:
      return "PRO MODE";
    case A21_RENDER_ERROR:
      return "ERROR";
    case A21_RENDER_LOCAL:
      return "LOCAL";
    case A21_RENDER_IDLE:
    default:
      return "IDLE";
  }
}
