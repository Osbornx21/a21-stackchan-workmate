#pragma once

#include "a21_firmware_protocol.h"

#include <stdint.h>

enum A21RenderState {
  A21_RENDER_IDLE,
  A21_RENDER_LISTENING,
  A21_RENDER_THINKING,
  A21_RENDER_SPEAKING,
  A21_RENDER_INTERRUPTED,
  A21_RENDER_PROFESSIONAL,
  A21_RENDER_LOCAL,
  A21_RENDER_ERROR,
};

struct A21FirmwareState {
  char device_id[32];
  char trace_id[A21_TRACE_ID_CAP];
  char session_id[A21_SESSION_ID_CAP];
  char mode[A21_MODE_CAP];
  char text[A21_TEXT_CAP];
  char stream_id[A21_STREAM_ID_CAP];
  char last_error[A21_ERROR_CAP];
  A21RenderState render_state;
  uint32_t updated_at_ms;
  uint16_t pending_diagnostic_tone_hz;
  uint16_t pending_diagnostic_tone_duration_ms;
  uint8_t pending_diagnostic_tone_volume;
  bool final;
};

inline void a21InitFirmwareState(A21FirmwareState* state, const char* device_id) {
  if (state == nullptr) {
    return;
  }
  a21CopyString(state->device_id, sizeof(state->device_id), device_id);
  a21CopyString(state->trace_id, A21_TRACE_ID_CAP, "");
  a21CopyString(state->session_id, A21_SESSION_ID_CAP, "");
  a21CopyString(state->mode, A21_MODE_CAP, "local_fallback");
  a21CopyString(state->text, A21_TEXT_CAP, "我现在连不上大脑，但我还在。");
  a21CopyString(state->stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(state->last_error, A21_ERROR_CAP, "");
  state->render_state = A21_RENDER_LOCAL;
  state->updated_at_ms = 0;
  state->pending_diagnostic_tone_hz = 0;
  state->pending_diagnostic_tone_duration_ms = 0;
  state->pending_diagnostic_tone_volume = 0;
  state->final = false;
}

inline A21RenderState a21RenderStateFromProtocol(const char* state) {
  if (a21StringEquals(state, "idle")) {
    return A21_RENDER_IDLE;
  }
  if (a21StringEquals(state, "listening")) {
    return A21_RENDER_LISTENING;
  }
  if (a21StringEquals(state, "thinking")) {
    return A21_RENDER_THINKING;
  }
  if (a21StringEquals(state, "speaking")) {
    return A21_RENDER_SPEAKING;
  }
  if (a21StringEquals(state, "interrupted")) {
    return A21_RENDER_INTERRUPTED;
  }
  if (a21StringEquals(state, "professional")) {
    return A21_RENDER_PROFESSIONAL;
  }
  if (a21StringEquals(state, "local_fallback")) {
    return A21_RENDER_LOCAL;
  }
  if (a21StringEquals(state, "error")) {
    return A21_RENDER_ERROR;
  }
  return A21_RENDER_ERROR;
}

inline const char* a21RenderStateProtocolName(A21RenderState state) {
  switch (state) {
    case A21_RENDER_IDLE:
      return "idle";
    case A21_RENDER_LISTENING:
      return "listening";
    case A21_RENDER_THINKING:
      return "thinking";
    case A21_RENDER_SPEAKING:
      return "speaking";
    case A21_RENDER_INTERRUPTED:
      return "interrupted";
    case A21_RENDER_PROFESSIONAL:
      return "professional";
    case A21_RENDER_LOCAL:
      return "local_fallback";
    case A21_RENDER_ERROR:
    default:
      return "error";
  }
}

inline bool a21ProtocolStateSupported(const char* state) {
  return a21StringEquals(state, "idle") ||
         a21StringEquals(state, "listening") ||
         a21StringEquals(state, "thinking") ||
         a21StringEquals(state, "speaking") ||
         a21StringEquals(state, "interrupted") ||
         a21StringEquals(state, "professional") ||
         a21StringEquals(state, "local_fallback") ||
         a21StringEquals(state, "error");
}

inline void a21ApplyControlEvent(A21FirmwareState* state, const A21ControlEvent* event, uint32_t now_ms) {
  if (state == nullptr || event == nullptr) {
    return;
  }
  a21CopyString(state->trace_id, A21_TRACE_ID_CAP, event->trace_id);
  a21CopyString(state->session_id, A21_SESSION_ID_CAP, event->session_id);
  a21CopyString(state->mode, A21_MODE_CAP, event->mode);
  a21CopyString(state->text, A21_TEXT_CAP, event->text);
  a21CopyString(state->stream_id, A21_STREAM_ID_CAP, event->stream_id);
  state->pending_diagnostic_tone_hz = event->diagnostic_tone_hz;
  state->pending_diagnostic_tone_duration_ms = event->diagnostic_tone_duration_ms;
  state->pending_diagnostic_tone_volume = event->diagnostic_tone_volume;
  state->final = event->final;
  state->updated_at_ms = now_ms;

  if (!a21ProtocolStateSupported(event->state)) {
    state->render_state = A21_RENDER_ERROR;
    a21CopyString(state->last_error, A21_ERROR_CAP, "unsupported_state");
    return;
  }

  state->render_state = a21RenderStateFromProtocol(event->state);
  a21CopyString(state->last_error, A21_ERROR_CAP, "");
}
