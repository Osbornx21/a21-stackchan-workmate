#pragma once

#include "a21_firmware_audio_playback.h"
#include "a21_firmware_state.h"

#include <stddef.h>
#include <stdint.h>

struct A21MicDriver {
  void* ctx;
  bool (*enabled)(void* ctx);
  bool (*record_pcm16)(void* ctx, int16_t* samples, size_t sample_count, uint32_t sample_rate_hz);
};

struct A21MicCaptureRuntime {
  int16_t samples[A21_AUDIO_PCM_FRAME_SAMPLES];
  uint32_t frames_captured;
  uint32_t skipped_speaker_busy;
  uint32_t skipped_render_state;
  uint32_t skipped_unavailable;
  uint32_t driver_errors;
  uint32_t capture_started_at_ms;
  uint32_t capture_ended_at_ms;
};

inline bool a21MicDriverReady(const A21MicDriver* driver) {
  return driver != nullptr && driver->enabled != nullptr && driver->record_pcm16 != nullptr;
}

inline void a21InitMicCaptureRuntime(A21MicCaptureRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  for (uint16_t i = 0; i < A21_AUDIO_PCM_FRAME_SAMPLES; ++i) {
    runtime->samples[i] = 0;
  }
  runtime->frames_captured = 0;
  runtime->skipped_speaker_busy = 0;
  runtime->skipped_render_state = 0;
  runtime->skipped_unavailable = 0;
  runtime->driver_errors = 0;
  runtime->capture_started_at_ms = 0;
  runtime->capture_ended_at_ms = 0;
}

inline bool a21MicCaptureRenderStateAllowed(A21RenderState state) {
  switch (state) {
    case A21_RENDER_IDLE:
    case A21_RENDER_LISTENING:
    case A21_RENDER_THINKING:
    case A21_RENDER_PROFESSIONAL:
      return true;
    case A21_RENDER_SPEAKING:
    case A21_RENDER_INTERRUPTED:
    case A21_RENDER_ERROR:
    case A21_RENDER_LOCAL:
    default:
      return false;
  }
}

inline bool a21MicCaptureTick(
    A21MicCaptureRuntime* runtime,
    const A21MicDriver* driver,
    const A21FirmwareState* state,
    size_t speaker_queue_depth,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21MicDriverReady(driver) || state == nullptr) {
    return false;
  }
  if (!a21MicCaptureRenderStateAllowed(state->render_state)) {
    runtime->skipped_render_state += 1;
    return true;
  }
  if (speaker_queue_depth > 0) {
    runtime->skipped_speaker_busy += 1;
    return true;
  }
  if (!driver->enabled(driver->ctx)) {
    runtime->skipped_unavailable += 1;
    return true;
  }

  if (!driver->record_pcm16(driver->ctx, runtime->samples, A21_AUDIO_PCM_FRAME_SAMPLES, A21_AUDIO_PCM_SAMPLE_RATE_HZ)) {
    runtime->driver_errors += 1;
    return false;
  }

  runtime->frames_captured += 1;
  runtime->capture_ended_at_ms = now_ms;
  runtime->capture_started_at_ms = now_ms >= A21_AUDIO_PCM_DURATION_MS ? now_ms - A21_AUDIO_PCM_DURATION_MS : 0;
  return true;
}
