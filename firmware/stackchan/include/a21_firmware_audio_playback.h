#pragma once

#include "a21_firmware_protocol.h"
#include "a21_firmware_state.h"

#include <ArduinoJson.h>

#include <stdint.h>

static constexpr size_t A21_AUDIO_CODEC_CAP = 16;
static constexpr size_t A21_AUDIO_DATA_BASE64_CAP = 900;
static constexpr uint8_t A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP = 8;

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

struct A21AudioPlaybackBuffer {
  char active_stream_id[A21_STREAM_ID_CAP];
  char last_trace_id[A21_TRACE_ID_CAP];
  uint8_t queued_chunks;
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

inline bool a21FailAudioPlaybackChunk(A21AudioPlaybackChunk* chunk, const char* field) {
  if (chunk != nullptr) {
    a21CopyString(chunk->error, A21_ERROR_CAP, field);
  }
  return false;
}

inline void a21InitAudioPlaybackBuffer(A21AudioPlaybackBuffer* buffer) {
  if (buffer == nullptr) {
    return;
  }
  a21CopyString(buffer->active_stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(buffer->last_trace_id, A21_TRACE_ID_CAP, "");
  buffer->queued_chunks = 0;
  buffer->total_chunks = 0;
  buffer->dropped_chunks = 0;
  buffer->clear_count = 0;
  buffer->last_duration_ms = 0;
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
  if (sample_rate_hz != 16000) {
    return a21FailAudioPlaybackChunk(chunk, "sample_rate_hz");
  }
  if (channels != 1) {
    return a21FailAudioPlaybackChunk(chunk, "channels");
  }
  if (duration_ms != 20) {
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
  buffer->last_duration_ms = 0;
  a21CopyString(buffer->active_stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(buffer->last_trace_id, A21_TRACE_ID_CAP, "");
  buffer->clear_count += 1;
}

inline bool a21AudioPlaybackBufferPush(A21AudioPlaybackBuffer* buffer, const A21AudioPlaybackChunk* chunk) {
  if (buffer == nullptr || chunk == nullptr || chunk->stream_id[0] == '\0') {
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
  buffer->queued_chunks += 1;
  buffer->total_chunks += 1;
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
