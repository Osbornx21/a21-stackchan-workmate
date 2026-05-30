#pragma once

#include "a21_firmware_protocol.h"
#include "a21_firmware_state.h"

#include <ArduinoJson.h>

#include <stddef.h>
#include <stdint.h>
#include <string.h>

static constexpr size_t A21_AUDIO_CODEC_CAP = 16;
static constexpr size_t A21_AUDIO_DATA_BASE64_CAP = 900;
static constexpr uint8_t A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP = 8;
static constexpr uint32_t A21_AUDIO_PCM_SAMPLE_RATE_HZ = 16000;
static constexpr uint8_t A21_AUDIO_PCM_CHANNELS = 1;
static constexpr uint16_t A21_AUDIO_PCM_DURATION_MS = 20;
static constexpr uint16_t A21_AUDIO_PCM_FRAME_SAMPLES =
    static_cast<uint16_t>((A21_AUDIO_PCM_SAMPLE_RATE_HZ * A21_AUDIO_PCM_DURATION_MS) / 1000);
static constexpr size_t A21_AUDIO_PCM_FRAME_BYTES = static_cast<size_t>(A21_AUDIO_PCM_FRAME_SAMPLES) * 2;

struct A21AudioPlaybackChunk {
  char trace_id[A21_TRACE_ID_CAP];
  char session_id[A21_SESSION_ID_CAP];
  char stream_id[A21_STREAM_ID_CAP];
  char codec[A21_AUDIO_CODEC_CAP];
  uint32_t sample_rate_hz;
  uint8_t channels;
  uint16_t duration_ms;
  char data_base64[A21_AUDIO_DATA_BASE64_CAP];
  char error[A21_ERROR_CAP];
};

struct A21AudioPCMFrame {
  char trace_id[A21_TRACE_ID_CAP];
  char stream_id[A21_STREAM_ID_CAP];
  uint8_t data[A21_AUDIO_PCM_FRAME_BYTES];
  size_t byte_count;
  uint16_t sample_count;
  char error[A21_ERROR_CAP];
};

struct A21AudioPlaybackBuffer {
  char active_stream_id[A21_STREAM_ID_CAP];
  char last_trace_id[A21_TRACE_ID_CAP];
  A21AudioPCMFrame frames[A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP];
  uint8_t queued_chunks;
  uint8_t read_index;
  uint8_t write_index;
  uint32_t total_chunks;
  uint32_t dropped_chunks;
  uint32_t clear_count;
  uint16_t last_duration_ms;
};

inline void a21ResetAudioPlaybackChunk(A21AudioPlaybackChunk* chunk) {
  if (chunk == nullptr) {
    return;
  }
  a21CopyString(chunk->trace_id, A21_TRACE_ID_CAP, "");
  a21CopyString(chunk->session_id, A21_SESSION_ID_CAP, "");
  a21CopyString(chunk->stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(chunk->codec, A21_AUDIO_CODEC_CAP, "");
  chunk->sample_rate_hz = 0;
  chunk->channels = 0;
  chunk->duration_ms = 0;
  a21CopyString(chunk->data_base64, A21_AUDIO_DATA_BASE64_CAP, "");
  a21CopyString(chunk->error, A21_ERROR_CAP, "");
}

inline void a21ResetAudioPCMFrame(A21AudioPCMFrame* frame) {
  if (frame == nullptr) {
    return;
  }
  a21CopyString(frame->trace_id, A21_TRACE_ID_CAP, "");
  a21CopyString(frame->stream_id, A21_STREAM_ID_CAP, "");
  memset(frame->data, 0, sizeof(frame->data));
  frame->byte_count = 0;
  frame->sample_count = 0;
  a21CopyString(frame->error, A21_ERROR_CAP, "");
}

inline bool a21FailAudioPlaybackChunk(A21AudioPlaybackChunk* chunk, const char* field) {
  if (chunk != nullptr) {
    a21CopyString(chunk->error, A21_ERROR_CAP, field);
  }
  return false;
}

inline bool a21FailAudioPCMFrame(A21AudioPCMFrame* frame, const char* field) {
  if (frame != nullptr) {
    a21CopyString(frame->error, A21_ERROR_CAP, field);
  }
  return false;
}

inline int8_t a21Base64Value(char value) {
  if (value >= 'A' && value <= 'Z') {
    return static_cast<int8_t>(value - 'A');
  }
  if (value >= 'a' && value <= 'z') {
    return static_cast<int8_t>(26 + value - 'a');
  }
  if (value >= '0' && value <= '9') {
    return static_cast<int8_t>(52 + value - '0');
  }
  if (value == '+') {
    return 62;
  }
  if (value == '/') {
    return 63;
  }
  if (value == '=') {
    return -2;
  }
  return -1;
}

inline void a21InitAudioPlaybackBuffer(A21AudioPlaybackBuffer* buffer) {
  if (buffer == nullptr) {
    return;
  }
  a21CopyString(buffer->active_stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(buffer->last_trace_id, A21_TRACE_ID_CAP, "");
  for (uint8_t i = 0; i < A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP; ++i) {
    a21ResetAudioPCMFrame(&buffer->frames[i]);
  }
  buffer->queued_chunks = 0;
  buffer->read_index = 0;
  buffer->write_index = 0;
  buffer->total_chunks = 0;
  buffer->dropped_chunks = 0;
  buffer->clear_count = 0;
  buffer->last_duration_ms = 0;
}

inline bool a21DecodeBase64PCM16(const char* input, A21AudioPCMFrame* frame) {
  if (frame == nullptr) {
    return false;
  }
  a21ResetAudioPCMFrame(frame);
  if (input == nullptr || input[0] == '\0') {
    return a21FailAudioPCMFrame(frame, "base64");
  }

  const size_t input_len = strlen(input);
  if (input_len % 4 != 0) {
    return a21FailAudioPCMFrame(frame, "base64");
  }

  size_t output_len = 0;
  bool reached_padding = false;
  for (size_t i = 0; i < input_len; i += 4) {
    const int8_t v0 = a21Base64Value(input[i]);
    const int8_t v1 = a21Base64Value(input[i + 1]);
    const int8_t v2 = a21Base64Value(input[i + 2]);
    const int8_t v3 = a21Base64Value(input[i + 3]);
    const bool final_group = i + 4 == input_len;

    if (reached_padding || v0 < 0 || v1 < 0 || v0 == -2 || v1 == -2 || v2 == -1 || v3 == -1) {
      return a21FailAudioPCMFrame(frame, "base64");
    }

    if (output_len >= A21_AUDIO_PCM_FRAME_BYTES) {
      return a21FailAudioPCMFrame(frame, "pcm_size");
    }
    frame->data[output_len++] = static_cast<uint8_t>((v0 << 2) | (v1 >> 4));

    if (v2 == -2) {
      if (v3 != -2 || !final_group) {
        return a21FailAudioPCMFrame(frame, "base64");
      }
      reached_padding = true;
      continue;
    }

    if (output_len >= A21_AUDIO_PCM_FRAME_BYTES) {
      return a21FailAudioPCMFrame(frame, "pcm_size");
    }
    frame->data[output_len++] = static_cast<uint8_t>(((v1 & 0x0F) << 4) | (v2 >> 2));

    if (v3 == -2) {
      if (!final_group) {
        return a21FailAudioPCMFrame(frame, "base64");
      }
      reached_padding = true;
      continue;
    }

    if (output_len >= A21_AUDIO_PCM_FRAME_BYTES) {
      return a21FailAudioPCMFrame(frame, "pcm_size");
    }
    frame->data[output_len++] = static_cast<uint8_t>(((v2 & 0x03) << 6) | v3);
  }

  if (output_len != A21_AUDIO_PCM_FRAME_BYTES) {
    return a21FailAudioPCMFrame(frame, "pcm_size");
  }
  frame->byte_count = output_len;
  frame->sample_count = A21_AUDIO_PCM_FRAME_SAMPLES;
  return true;
}

inline bool a21DecodeAudioPlaybackChunkPCM(const A21AudioPlaybackChunk* chunk, A21AudioPCMFrame* frame) {
  if (frame == nullptr) {
    return false;
  }
  a21ResetAudioPCMFrame(frame);
  if (chunk == nullptr || chunk->stream_id[0] == '\0') {
    return a21FailAudioPCMFrame(frame, "chunk");
  }
  if (!a21StringEquals(chunk->codec, "pcm_s16le") ||
      chunk->sample_rate_hz != A21_AUDIO_PCM_SAMPLE_RATE_HZ ||
      chunk->channels != A21_AUDIO_PCM_CHANNELS ||
      chunk->duration_ms != A21_AUDIO_PCM_DURATION_MS) {
    return a21FailAudioPCMFrame(frame, "format");
  }
  if (!a21DecodeBase64PCM16(chunk->data_base64, frame)) {
    return false;
  }
  a21CopyString(frame->trace_id, A21_TRACE_ID_CAP, chunk->trace_id);
  a21CopyString(frame->stream_id, A21_STREAM_ID_CAP, chunk->stream_id);
  return true;
}

inline bool a21ParseAudioPlaybackChunk(const char* json, const char* expected_device_id, A21AudioPlaybackChunk* chunk) {
  if (chunk == nullptr) {
    return false;
  }
  a21ResetAudioPlaybackChunk(chunk);
  if (json == nullptr || expected_device_id == nullptr) {
    return a21FailAudioPlaybackChunk(chunk, "input");
  }

  JsonDocument doc;
  DeserializationError err = deserializeJson(doc, json);
  if (err) {
    return a21FailAudioPlaybackChunk(chunk, "json");
  }

  const char* protocol = doc["protocol"] | "";
  if (!a21StringEquals(protocol, "a21.device.v1")) {
    return a21FailAudioPlaybackChunk(chunk, "protocol");
  }

  const char* device_id = doc["device_id"] | "";
  if (!a21StringEquals(device_id, expected_device_id)) {
    return a21FailAudioPlaybackChunk(chunk, "device_id");
  }

  const char* kind = doc["kind"] | "";
  if (!a21StringEquals(kind, "audio.playback.chunk")) {
    return a21FailAudioPlaybackChunk(chunk, "kind");
  }

  JsonVariantConst payload = doc["payload"];
  if (payload.isNull()) {
    return a21FailAudioPlaybackChunk(chunk, "payload");
  }

  const char* stream_id = payload["stream_id"] | "";
  const char* codec = payload["codec"] | "";
  const char* data_base64 = payload["data_base64"] | "";
  const uint32_t sample_rate_hz = payload["sample_rate_hz"] | 0;
  const uint8_t channels = payload["channels"] | 0;
  const uint16_t duration_ms = payload["duration_ms"] | 0;

  if (stream_id[0] == '\0') {
    return a21FailAudioPlaybackChunk(chunk, "stream_id");
  }
  if (!a21StringEquals(codec, "pcm_s16le")) {
    return a21FailAudioPlaybackChunk(chunk, "codec");
  }
  if (sample_rate_hz != A21_AUDIO_PCM_SAMPLE_RATE_HZ) {
    return a21FailAudioPlaybackChunk(chunk, "sample_rate_hz");
  }
  if (channels != A21_AUDIO_PCM_CHANNELS) {
    return a21FailAudioPlaybackChunk(chunk, "channels");
  }
  if (duration_ms != A21_AUDIO_PCM_DURATION_MS) {
    return a21FailAudioPlaybackChunk(chunk, "duration_ms");
  }
  if (data_base64[0] == '\0') {
    return a21FailAudioPlaybackChunk(chunk, "data_base64");
  }

  a21CopyString(chunk->trace_id, A21_TRACE_ID_CAP, doc["trace_id"] | "");
  a21CopyString(chunk->session_id, A21_SESSION_ID_CAP, doc["session_id"] | "");
  a21CopyString(chunk->stream_id, A21_STREAM_ID_CAP, stream_id);
  a21CopyString(chunk->codec, A21_AUDIO_CODEC_CAP, codec);
  chunk->sample_rate_hz = sample_rate_hz;
  chunk->channels = channels;
  chunk->duration_ms = duration_ms;
  a21CopyString(chunk->data_base64, A21_AUDIO_DATA_BASE64_CAP, data_base64);
  return true;
}

inline void a21AudioPlaybackBufferClear(A21AudioPlaybackBuffer* buffer) {
  if (buffer == nullptr) {
    return;
  }
  buffer->queued_chunks = 0;
  buffer->read_index = 0;
  buffer->write_index = 0;
  buffer->last_duration_ms = 0;
  for (uint8_t i = 0; i < A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP; ++i) {
    a21ResetAudioPCMFrame(&buffer->frames[i]);
  }
  a21CopyString(buffer->active_stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(buffer->last_trace_id, A21_TRACE_ID_CAP, "");
  buffer->clear_count += 1;
}

inline bool a21AudioPlaybackBufferPush(A21AudioPlaybackBuffer* buffer, const A21AudioPlaybackChunk* chunk) {
  if (buffer == nullptr || chunk == nullptr || chunk->stream_id[0] == '\0') {
    return false;
  }
  A21AudioPCMFrame frame;
  if (!a21DecodeAudioPlaybackChunkPCM(chunk, &frame)) {
    return false;
  }
  if (buffer->active_stream_id[0] != '\0' && !a21StringEquals(buffer->active_stream_id, chunk->stream_id)) {
    a21AudioPlaybackBufferClear(buffer);
  }
  a21CopyString(buffer->active_stream_id, A21_STREAM_ID_CAP, chunk->stream_id);
  a21CopyString(buffer->last_trace_id, A21_TRACE_ID_CAP, chunk->trace_id);
  buffer->last_duration_ms = chunk->duration_ms;
  if (buffer->queued_chunks >= A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP) {
    buffer->dropped_chunks += 1;
    return true;
  }
  buffer->frames[buffer->write_index] = frame;
  buffer->write_index = static_cast<uint8_t>((buffer->write_index + 1) % A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP);
  buffer->queued_chunks += 1;
  buffer->total_chunks += 1;
  return true;
}

inline const A21AudioPCMFrame* a21AudioPlaybackBufferPeek(const A21AudioPlaybackBuffer* buffer) {
  if (buffer == nullptr || buffer->queued_chunks == 0) {
    return nullptr;
  }
  return &buffer->frames[buffer->read_index];
}

inline bool a21AudioPlaybackBufferPop(A21AudioPlaybackBuffer* buffer, A21AudioPCMFrame* output) {
  if (buffer == nullptr || buffer->queued_chunks == 0) {
    return false;
  }
  if (output != nullptr) {
    *output = buffer->frames[buffer->read_index];
  }
  a21ResetAudioPCMFrame(&buffer->frames[buffer->read_index]);
  buffer->read_index = static_cast<uint8_t>((buffer->read_index + 1) % A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP);
  buffer->queued_chunks -= 1;
  return true;
}

inline bool a21AudioPlaybackBufferApplyState(A21AudioPlaybackBuffer* buffer, const A21FirmwareState* state) {
  if (buffer == nullptr || state == nullptr) {
    return false;
  }
  if (state->render_state == A21_RENDER_SPEAKING) {
    return true;
  }
  if (buffer->queued_chunks > 0 || buffer->active_stream_id[0] != '\0') {
    a21AudioPlaybackBufferClear(buffer);
  }
  return true;
}
