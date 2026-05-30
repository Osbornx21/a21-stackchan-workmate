#pragma once

#include "a21_firmware_state.h"

enum A21AvatarExpression {
  A21_AVATAR_EXPRESSION_NEUTRAL,
  A21_AVATAR_EXPRESSION_HAPPY,
  A21_AVATAR_EXPRESSION_ANGRY,
  A21_AVATAR_EXPRESSION_SAD,
  A21_AVATAR_EXPRESSION_DOUBT,
  A21_AVATAR_EXPRESSION_SLEEPY,
};

struct A21FaceFrame {
  const char* label;
  A21AvatarExpression expression;
  float eye_open_ratio;
  float gaze_vertical;
  float gaze_horizontal;
  float mouth_open_ratio;
  float breath_ratio;
  bool auto_blink;
  bool show_status_label;
};

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

inline A21FaceFrame a21FaceFrameForState(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_NEUTRAL, 0.95f, 0.25f, 0.0f, 0.0f, 0.18f, true, true};
    case A21_RENDER_THINKING:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_DOUBT, 0.72f, -0.45f, 0.12f, 0.0f, 0.14f, true, true};
    case A21_RENDER_SPEAKING:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_NEUTRAL, 1.0f, 0.0f, 0.0f, 0.82f, 0.24f, true, true};
    case A21_RENDER_INTERRUPTED:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_DOUBT, 0.84f, 0.20f, 0.0f, 0.0f, 0.12f, true, true};
    case A21_RENDER_PROFESSIONAL:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_NEUTRAL, 0.78f, 0.0f, 0.0f, 0.05f, 0.08f, true, true};
    case A21_RENDER_ERROR:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_SAD, 0.62f, 0.35f, 0.0f, 0.0f, 0.04f, false, true};
    case A21_RENDER_LOCAL:
      return {a21DisplayStateLabel(state), A21_AVATAR_EXPRESSION_SLEEPY, 0.55f, 0.25f, 0.0f, 0.0f, 0.04f, false, true};
    case A21_RENDER_IDLE:
    default:
      return {a21DisplayStateLabel(A21_RENDER_IDLE), A21_AVATAR_EXPRESSION_NEUTRAL, 0.82f, 0.0f, 0.0f, 0.0f, 0.16f, true, true};
  }
}
