#pragma once

#include "a21_firmware_audio_playback.h"
#include "a21_firmware_state.h"

#include <stddef.h>
#include <stdint.h>

static constexpr uint8_t A21_SPEAKER_CHANNEL = 0;
static constexpr uint8_t A21_SPEAKER_SLOT_COUNT = 3;
static constexpr uint8_t A21_SPEAKER_MAX_DRIVER_QUEUE = 2;
static constexpr uint8_t A21_SPEAKER_PLAYBACK_BATCH_FRAMES = 8;
static constexpr uint8_t A21_SPEAKER_TASK_PRIORITY = 4;
static constexpr uint16_t A21_SPEAKER_PLAYBACK_BATCH_SAMPLES =
    A21_AUDIO_PCM_FRAME_SAMPLES * A21_SPEAKER_PLAYBACK_BATCH_FRAMES;

struct A21SpeakerDriver {
  void* ctx;
  size_t (*queued)(void* ctx, uint8_t channel);
  bool (*play_pcm16)(void* ctx, const int16_t* samples, size_t sample_count, uint32_t sample_rate_hz, uint8_t channel);
};

struct A21SpeakerPumpRuntime {
  int16_t slots[A21_SPEAKER_SLOT_COUNT][A21_SPEAKER_PLAYBACK_BATCH_SAMPLES];
  uint8_t next_slot;
  uint32_t frames_played;
  uint32_t busy_ticks;
  uint32_t driver_errors;
  bool preroll_complete;
  char active_stream_id[A21_STREAM_ID_CAP];
  char last_trace_id[A21_TRACE_ID_CAP];
  char last_stream_id[A21_STREAM_ID_CAP];
};

inline bool a21SpeakerDriverReady(const A21SpeakerDriver* driver) {
  return driver != nullptr && driver->queued != nullptr && driver->play_pcm16 != nullptr;
}

inline void a21InitSpeakerPumpRuntime(A21SpeakerPumpRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  for (uint8_t slot = 0; slot < A21_SPEAKER_SLOT_COUNT; ++slot) {
    for (uint16_t sample = 0; sample < A21_SPEAKER_PLAYBACK_BATCH_SAMPLES; ++sample) {
      runtime->slots[slot][sample] = 0;
    }
  }
  runtime->next_slot = 0;
  runtime->frames_played = 0;
  runtime->busy_ticks = 0;
  runtime->driver_errors = 0;
  runtime->preroll_complete = false;
  a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(runtime->last_trace_id, A21_TRACE_ID_CAP, "");
  a21CopyString(runtime->last_stream_id, A21_STREAM_ID_CAP, "");
}

inline void a21SpeakerCopyPCM16Frame(const A21AudioPCMFrame* frame, int16_t* output, size_t output_samples) {
  if (frame == nullptr || output == nullptr) {
    return;
  }
  const size_t samples = frame->sample_count < output_samples ? frame->sample_count : output_samples;
  for (size_t i = 0; i < samples; ++i) {
    const size_t byte_index = i * 2;
    const uint16_t value = static_cast<uint16_t>(frame->data[byte_index]) |
                           (static_cast<uint16_t>(frame->data[byte_index + 1]) << 8);
    output[i] = static_cast<int16_t>(value);
  }
}

inline uint8_t a21SpeakerCopyPlaybackBatch(
    const A21AudioPlaybackBuffer* buffer,
    const char* stream_id,
    int16_t* output,
    size_t output_samples) {
  if (buffer == nullptr || stream_id == nullptr || output == nullptr || output_samples == 0) {
    return 0;
  }
  const uint8_t max_frames_by_output =
      static_cast<uint8_t>(output_samples / A21_AUDIO_PCM_FRAME_SAMPLES);
  const uint8_t max_frames =
      max_frames_by_output < A21_SPEAKER_PLAYBACK_BATCH_FRAMES ? max_frames_by_output : A21_SPEAKER_PLAYBACK_BATCH_FRAMES;
  uint8_t copied_frames = 0;
  while (copied_frames < max_frames && copied_frames < buffer->queued_chunks) {
    const uint8_t frame_index =
        static_cast<uint8_t>((buffer->read_index + copied_frames) % A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP);
    const A21AudioPCMFrame* frame = &buffer->frames[frame_index];
    if (!a21StringEquals(frame->stream_id, stream_id)) {
      break;
    }
    a21SpeakerCopyPCM16Frame(
        frame,
        output + (copied_frames * A21_AUDIO_PCM_FRAME_SAMPLES),
        A21_AUDIO_PCM_FRAME_SAMPLES);
    copied_frames += 1;
  }
  return copied_frames;
}

inline bool a21SpeakerPumpTick(
    A21SpeakerPumpRuntime* runtime,
    const A21SpeakerDriver* driver,
    const A21FirmwareState* state,
    A21AudioPlaybackBuffer* buffer) {
  if (runtime == nullptr || !a21SpeakerDriverReady(driver) || state == nullptr || buffer == nullptr) {
    return false;
  }
  if (state->render_state != A21_RENDER_SPEAKING || state->stream_id[0] == '\0') {
    runtime->preroll_complete = false;
    a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, "");
    return true;
  }
  if (!a21StringEquals(runtime->active_stream_id, state->stream_id)) {
    runtime->preroll_complete = false;
    a21CopyString(runtime->active_stream_id, A21_STREAM_ID_CAP, state->stream_id);
  }

  const A21AudioPCMFrame* frame = a21AudioPlaybackBufferPeek(buffer);
  if (frame == nullptr) {
    return true;
  }
  if (!a21StringEquals(frame->stream_id, state->stream_id)) {
    return true;
  }
  if (!runtime->preroll_complete && buffer->queued_chunks < A21_SPEAKER_PLAYBACK_BATCH_FRAMES) {
    return true;
  }
  if (driver->queued(driver->ctx, A21_SPEAKER_CHANNEL) >= A21_SPEAKER_MAX_DRIVER_QUEUE) {
    runtime->busy_ticks += 1;
    return true;
  }

  const uint8_t slot = runtime->next_slot;
  const uint8_t playback_frames = a21SpeakerCopyPlaybackBatch(
      buffer,
      state->stream_id,
      runtime->slots[slot],
      A21_SPEAKER_PLAYBACK_BATCH_SAMPLES);
  if (playback_frames == 0) {
    return true;
  }
  const size_t playback_samples =
      static_cast<size_t>(playback_frames) * A21_AUDIO_PCM_FRAME_SAMPLES;
  if (!driver->play_pcm16(
          driver->ctx,
          runtime->slots[slot],
          playback_samples,
          A21_AUDIO_PCM_SAMPLE_RATE_HZ,
          A21_SPEAKER_CHANNEL)) {
    runtime->driver_errors += 1;
    return false;
  }

  runtime->next_slot = static_cast<uint8_t>((runtime->next_slot + 1) % A21_SPEAKER_SLOT_COUNT);
  runtime->frames_played += playback_frames;
  runtime->preroll_complete = true;
  a21CopyString(runtime->last_trace_id, A21_TRACE_ID_CAP, frame->trace_id);
  a21CopyString(runtime->last_stream_id, A21_STREAM_ID_CAP, frame->stream_id);
  for (uint8_t i = 0; i < playback_frames; ++i) {
    if (!a21AudioPlaybackBufferPop(buffer, nullptr)) {
      return false;
    }
  }
  return true;
}
