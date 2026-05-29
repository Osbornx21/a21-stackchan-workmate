#pragma once

#include "a21_firmware_audio_playback.h"
#include "a21_firmware_connection.h"
#include "a21_firmware_network.h"
#include "a21_firmware_state.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

static constexpr size_t A21_AUDIO_WS_TEXT_MESSAGE_CAP = 1400;

struct A21AudioWSDriver {
  void* ctx;
  bool (*begin)(void* ctx, const char* host, uint16_t port, const char* path);
  void (*loop)(void* ctx);
  bool (*connected)(void* ctx);
  bool (*read_text)(void* ctx, char* output, size_t output_size);
  bool (*send_text)(void* ctx, const char* text);
};

struct A21AudioWSRuntime {
  bool begin_sent;
  bool was_connected;
  uint32_t last_begin_at_ms;
  uint32_t received_control_events;
  uint32_t invalid_control_events;
  uint32_t received_audio_chunks;
  uint32_t sent_audio_frames;
  uint64_t next_seq;
};

inline void a21InitAudioWSRuntime(A21AudioWSRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->begin_sent = false;
  runtime->was_connected = false;
  runtime->last_begin_at_ms = 0;
  runtime->received_control_events = 0;
  runtime->invalid_control_events = 0;
  runtime->received_audio_chunks = 0;
  runtime->sent_audio_frames = 0;
  runtime->next_seq = 1;
}

inline bool a21AudioWSDriverReady(const A21AudioWSDriver* driver) {
  return driver != nullptr &&
         driver->begin != nullptr &&
         driver->loop != nullptr &&
         driver->connected != nullptr &&
         driver->read_text != nullptr &&
         driver->send_text != nullptr;
}

inline bool a21AudioWSBuildMockFrame(
    const A21AudioWSRuntime* runtime,
    const A21FirmwareState* state,
    uint32_t now_ms,
    char* output,
    size_t output_size) {
  if (runtime == nullptr || state == nullptr || output == nullptr || output_size == 0) {
    return false;
  }
  output[0] = '\0';

  char trace_id[A21_TRACE_ID_CAP];
  snprintf(trace_id, sizeof(trace_id), "a21-trace-audio-%06llu", static_cast<unsigned long long>(runtime->next_seq));

  JsonDocument doc;
  doc["protocol"] = "a21.device.v1";
  doc["device_id"] = state->device_id;
  doc["kind"] = "audio.frame";
  doc["seq"] = runtime->next_seq;
  doc["trace_id"] = trace_id;
  doc["session_id"] = state->session_id[0] == '\0' ? "a21-session-device" : state->session_id;
  doc["sent_at_ms"] = now_ms;

  JsonObject payload = doc["payload"].to<JsonObject>();
  payload["codec"] = "pcm_s16le";
  payload["sample_rate_hz"] = 16000;
  payload["channels"] = 1;
  payload["duration_ms"] = 20;
  payload["capture_started_at_ms"] = now_ms >= 20 ? now_ms - 20 : 0;
  payload["capture_ended_at_ms"] = now_ms;
  payload["data_base64"] = "AAAA";

  const size_t written = serializeJson(doc, output, output_size);
  return written > 0 && written < output_size;
}

inline bool a21AudioWSSendMockFrame(
    A21AudioWSRuntime* runtime,
    const A21AudioWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21AudioWSDriverReady(driver) || connection == nullptr || state == nullptr) {
    return false;
  }
  if (connection->phase != A21_CONN_GATEWAY_CONNECTED || !driver->connected(driver->ctx)) {
    return false;
  }

  char message[A21_AUDIO_WS_TEXT_MESSAGE_CAP];
  if (!a21AudioWSBuildMockFrame(runtime, state, now_ms, message, sizeof(message))) {
    return false;
  }
  if (!driver->send_text(driver->ctx, message)) {
    return false;
  }
  runtime->sent_audio_frames += 1;
  runtime->next_seq += 1;
  return true;
}

inline void a21AudioWSApplyTextWithPlayback(
    A21AudioWSRuntime* runtime,
    A21FirmwareState* state,
    A21AudioPlaybackBuffer* playback_buffer,
    const char* text,
    uint32_t now_ms) {
  if (runtime == nullptr || state == nullptr || text == nullptr || text[0] == '\0') {
    return;
  }
  A21ControlEvent event;
  if (a21ParseControlEvent(text, state->device_id, &event)) {
    a21ApplyControlEvent(state, &event, now_ms);
    runtime->received_control_events += 1;
    return;
  }
  A21AudioPlaybackChunk chunk;
  if (playback_buffer != nullptr && a21ParseAudioPlaybackChunk(text, state->device_id, &chunk)) {
    if (a21AudioPlaybackBufferPush(playback_buffer, &chunk)) {
      runtime->received_audio_chunks += 1;
      return;
    }
  }
  runtime->invalid_control_events += 1;
  a21CopyString(state->last_error, A21_ERROR_CAP, event.error);
  a21CopyString(state->text, A21_TEXT_CAP, "A21 audio message error");
  state->render_state = A21_RENDER_ERROR;
  state->updated_at_ms = now_ms;
}

inline void a21AudioWSApplyText(
    A21AudioWSRuntime* runtime,
    A21FirmwareState* state,
    const char* text,
    uint32_t now_ms) {
  a21AudioWSApplyTextWithPlayback(runtime, state, nullptr, text, now_ms);
}

inline void a21AudioWSDrainTextWithPlayback(
    A21AudioWSRuntime* runtime,
    const A21AudioWSDriver* driver,
    A21FirmwareState* state,
    A21AudioPlaybackBuffer* playback_buffer,
    uint32_t now_ms) {
  char text[A21_AUDIO_WS_TEXT_MESSAGE_CAP];
  uint8_t guard = 0;
  while (guard < 4 && driver->read_text(driver->ctx, text, sizeof(text))) {
    a21AudioWSApplyTextWithPlayback(runtime, state, playback_buffer, text, now_ms);
    ++guard;
  }
}

inline void a21AudioWSDrainText(
    A21AudioWSRuntime* runtime,
    const A21AudioWSDriver* driver,
    A21FirmwareState* state,
    uint32_t now_ms) {
  a21AudioWSDrainTextWithPlayback(runtime, driver, state, nullptr, now_ms);
}

inline void a21AudioWSRuntimeTickWithPlayback(
    A21AudioWSRuntime* runtime,
    const A21AudioWSDriver* driver,
    const A21ConnectionState* connection,
    const A21NetworkConfig* network,
    A21FirmwareState* state,
    A21AudioPlaybackBuffer* playback_buffer,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21AudioWSDriverReady(driver) || connection == nullptr || network == nullptr || state == nullptr) {
    return;
  }

  if (connection->phase != A21_CONN_GATEWAY_CONNECTED) {
    runtime->begin_sent = false;
    runtime->was_connected = false;
    return;
  }

  driver->loop(driver->ctx);
  if (driver->connected(driver->ctx)) {
    runtime->was_connected = true;
    a21AudioWSDrainTextWithPlayback(runtime, driver, state, playback_buffer, now_ms);
    return;
  }

  if (runtime->was_connected) {
    runtime->begin_sent = false;
    runtime->was_connected = false;
  }

  if (!runtime->begin_sent) {
    if (driver->begin(driver->ctx, network->gateway_host, network->gateway_port, network->audio_path)) {
      runtime->begin_sent = true;
      runtime->last_begin_at_ms = now_ms;
    } else {
      a21CopyString(state->last_error, A21_ERROR_CAP, "audio_ws_begin_failed");
    }
  }
}

inline void a21AudioWSRuntimeTick(
    A21AudioWSRuntime* runtime,
    const A21AudioWSDriver* driver,
    const A21ConnectionState* connection,
    const A21NetworkConfig* network,
    A21FirmwareState* state,
    uint32_t now_ms) {
  a21AudioWSRuntimeTickWithPlayback(runtime, driver, connection, network, state, nullptr, now_ms);
}
