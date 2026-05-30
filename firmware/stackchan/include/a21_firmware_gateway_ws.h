#pragma once

#include "a21_firmware_capabilities.h"
#include "a21_firmware_config.h"
#include "a21_firmware_connection.h"
#include "a21_firmware_mic.h"
#include "a21_firmware_motion.h"
#include "a21_firmware_network.h"
#include "a21_firmware_rgb.h"
#include "a21_firmware_state.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

static constexpr size_t A21_WS_TEXT_MESSAGE_CAP = 1800;

struct A21RuntimeEchoDiagnostics {
  bool enabled;
  uint32_t mic_frames_captured;
  uint32_t mic_driver_errors;
  uint32_t mic_skipped_render_state;
  uint32_t mic_skipped_speaker_busy;
  uint32_t mic_skipped_unavailable;
  uint8_t mic_queue_depth;
  uint32_t mic_queue_total_frames;
  uint32_t mic_queue_dropped_frames;
  uint32_t audio_ws_sent_audio_frames;
  int16_t mic_last_abs_peak;
  uint16_t mic_last_nonzero_samples;
  uint8_t playback_buffer_queued_chunks;
  uint32_t playback_buffer_total_chunks;
  uint32_t playback_buffer_dropped_chunks;
  uint32_t playback_buffer_clear_count;
  uint32_t speaker_frames_played;
  uint32_t speaker_busy_ticks;
  uint32_t speaker_driver_errors;
  char speaker_last_stream_id[A21_STREAM_ID_CAP];
};

struct A21GatewayWSDriver {
  void* ctx;
  bool (*begin)(void* ctx, const char* host, uint16_t port, const char* path);
  void (*loop)(void* ctx);
  bool (*connected)(void* ctx);
  bool (*read_text)(void* ctx, char* output, size_t output_size);
  bool (*send_text)(void* ctx, const char* text);
};

struct A21GatewayWSRuntime {
  bool begin_sent;
  uint32_t last_begin_at_ms;
  uint32_t received_control_events;
  uint32_t invalid_control_events;
  uint32_t sent_device_events;
  uint64_t next_seq;
  bool has_runtime_echo;
  A21RenderState last_runtime_echo_render_state;
  int last_runtime_echo_y_deg;
  A21RGBColor last_runtime_echo_rgb;
  bool has_runtime_echo_diagnostics;
  A21RuntimeEchoDiagnostics last_runtime_echo_diagnostics;
};

inline void a21InitGatewayWSRuntime(A21GatewayWSRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->begin_sent = false;
  runtime->last_begin_at_ms = 0;
  runtime->received_control_events = 0;
  runtime->invalid_control_events = 0;
  runtime->sent_device_events = 0;
  runtime->next_seq = 1;
  runtime->has_runtime_echo = false;
  runtime->last_runtime_echo_render_state = A21_RENDER_ERROR;
  runtime->last_runtime_echo_y_deg = 45;
  runtime->last_runtime_echo_rgb = a21RGBColorMake(0, 0, 0);
  runtime->has_runtime_echo_diagnostics = false;
  runtime->last_runtime_echo_diagnostics = {};
}

inline bool a21GatewayWSDriverReady(const A21GatewayWSDriver* driver) {
  return driver != nullptr &&
         driver->begin != nullptr &&
         driver->loop != nullptr &&
         driver->connected != nullptr &&
         driver->read_text != nullptr &&
         driver->send_text != nullptr;
}

inline bool a21GatewayWSBuildDeviceEventWithTouchSource(
    const A21GatewayWSRuntime* runtime,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    const char* touch_source,
    uint32_t now_ms,
    char* output,
    size_t output_size);

inline bool a21GatewayWSSendDeviceEventWithTouchSource(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    const char* touch_source,
    uint32_t now_ms);

inline bool a21GatewayWSSendRuntimeEchoIfChanged(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const A21MotionRuntime* motion_runtime,
    const A21RGBRuntime* rgb_runtime,
    uint32_t now_ms);

inline bool a21GatewayWSSendRuntimeEchoIfChangedWithDiagnostics(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const A21MotionRuntime* motion_runtime,
    const A21RGBRuntime* rgb_runtime,
    const A21RuntimeEchoDiagnostics* diagnostics,
    uint32_t now_ms);

inline bool a21GatewayWSBuildDeviceEvent(
    const A21GatewayWSRuntime* runtime,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    uint32_t now_ms,
    char* output,
    size_t output_size) {
  return a21GatewayWSBuildDeviceEventWithTouchSource(
      runtime,
      state,
      event,
      mode,
      text,
      "",
      now_ms,
      output,
      output_size);
}

inline bool a21GatewayWSBuildDeviceEventWithTouchSource(
    const A21GatewayWSRuntime* runtime,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    const char* touch_source,
    uint32_t now_ms,
    char* output,
    size_t output_size) {
  if (runtime == nullptr || state == nullptr || event == nullptr || event[0] == '\0' || output == nullptr || output_size == 0) {
    return false;
  }
  output[0] = '\0';

  char trace_id[A21_TRACE_ID_CAP];
  snprintf(trace_id, sizeof(trace_id), "a21-trace-device-%06llu", static_cast<unsigned long long>(runtime->next_seq));

  JsonDocument doc;
  doc["protocol"] = "a21.device.v1";
  doc["device_id"] = state->device_id;
  doc["kind"] = "device.event";
  doc["seq"] = runtime->next_seq;
  doc["trace_id"] = trace_id;
  doc["session_id"] = state->session_id[0] == '\0' ? "a21-session-device" : state->session_id;
  doc["sent_at_ms"] = now_ms;
  JsonObject payload = doc["payload"].to<JsonObject>();
  payload["event"] = event;
  payload["mode"] = (mode == nullptr || mode[0] == '\0') ? "workmate" : mode;
  payload["firmware_id"] = A21_FIRMWARE_ID;
  payload["firmware_version"] = A21_FIRMWARE_VERSION;
  payload["firmware_board"] = A21_FIRMWARE_BOARD;
  payload["firmware_commit"] = A21_FIRMWARE_COMMIT;
  JsonObject capabilities = payload["capabilities"].to<JsonObject>();
  a21WriteStackChanHardwareCapabilities(capabilities);
  if (text != nullptr && text[0] != '\0') {
    payload["text"] = text;
  }
  if (touch_source != nullptr && touch_source[0] != '\0') {
    payload["touch_source"] = touch_source;
  }

  const size_t written = serializeJson(doc, output, output_size);
  return written > 0 && written < output_size;
}

inline bool a21GatewayWSBuildRuntimeEchoEvent(
    const A21GatewayWSRuntime* runtime,
    const A21FirmwareState* state,
    const A21MotionRuntime* motion_runtime,
    const A21RGBRuntime* rgb_runtime,
    const A21RuntimeEchoDiagnostics* diagnostics,
    uint32_t now_ms,
    char* output,
    size_t output_size) {
  if (runtime == nullptr || state == nullptr || motion_runtime == nullptr || rgb_runtime == nullptr ||
      !motion_runtime->has_y || !rgb_runtime->has_color || output == nullptr || output_size == 0) {
    return false;
  }
  output[0] = '\0';

  char trace_id[A21_TRACE_ID_CAP];
  snprintf(trace_id, sizeof(trace_id), "a21-trace-device-%06llu", static_cast<unsigned long long>(runtime->next_seq));
  char servo_y[12];
  snprintf(servo_y, sizeof(servo_y), "%ddeg", motion_runtime->last_y_deg);
  char rgb_hex[8];
  snprintf(
      rgb_hex,
      sizeof(rgb_hex),
      "#%02X%02X%02X",
      rgb_runtime->last_color.r,
      rgb_runtime->last_color.g,
      rgb_runtime->last_color.b);

  JsonDocument doc;
  doc["protocol"] = "a21.device.v1";
  doc["device_id"] = state->device_id;
  doc["kind"] = "device.event";
  doc["seq"] = runtime->next_seq;
  doc["trace_id"] = trace_id;
  doc["session_id"] = state->session_id[0] == '\0' ? "a21-session-device" : state->session_id;
  doc["sent_at_ms"] = now_ms;
  JsonObject payload = doc["payload"].to<JsonObject>();
  payload["event"] = "runtime.echo";
  payload["mode"] = state->mode[0] == '\0' ? "workmate" : state->mode;
  payload["firmware_id"] = A21_FIRMWARE_ID;
  payload["firmware_version"] = A21_FIRMWARE_VERSION;
  payload["firmware_board"] = A21_FIRMWARE_BOARD;
  payload["firmware_commit"] = A21_FIRMWARE_COMMIT;
  JsonObject capabilities = payload["capabilities"].to<JsonObject>();
  a21WriteStackChanHardwareCapabilities(capabilities);
  JsonObject echo = payload["runtime_echo"].to<JsonObject>();
  echo["screen"] = a21RenderStateProtocolName(state->render_state);
  echo["servo_y"] = servo_y;
  echo["rgb"] = rgb_hex;
  if (diagnostics != nullptr && diagnostics->enabled) {
    char mic_frames_captured[12];
    char mic_driver_errors[12];
    char mic_skipped_render_state[12];
    char mic_skipped_speaker_busy[12];
    char mic_skipped_unavailable[12];
    char mic_queue_depth[8];
    char mic_queue_total_frames[12];
    char mic_queue_dropped_frames[12];
    char audio_ws_sent_audio_frames[12];
    char mic_last_abs_peak[12];
    char mic_last_nonzero_samples[12];
    char playback_buffer_queued_chunks[8];
    char playback_buffer_total_chunks[12];
    char playback_buffer_dropped_chunks[12];
    char playback_buffer_clear_count[12];
    char speaker_frames_played[12];
    char speaker_busy_ticks[12];
    char speaker_driver_errors[12];
    snprintf(mic_frames_captured, sizeof(mic_frames_captured), "%lu", static_cast<unsigned long>(diagnostics->mic_frames_captured));
    snprintf(mic_driver_errors, sizeof(mic_driver_errors), "%lu", static_cast<unsigned long>(diagnostics->mic_driver_errors));
    snprintf(mic_skipped_render_state, sizeof(mic_skipped_render_state), "%lu", static_cast<unsigned long>(diagnostics->mic_skipped_render_state));
    snprintf(mic_skipped_speaker_busy, sizeof(mic_skipped_speaker_busy), "%lu", static_cast<unsigned long>(diagnostics->mic_skipped_speaker_busy));
    snprintf(mic_skipped_unavailable, sizeof(mic_skipped_unavailable), "%lu", static_cast<unsigned long>(diagnostics->mic_skipped_unavailable));
    snprintf(mic_queue_depth, sizeof(mic_queue_depth), "%u", static_cast<unsigned>(diagnostics->mic_queue_depth));
    snprintf(mic_queue_total_frames, sizeof(mic_queue_total_frames), "%lu", static_cast<unsigned long>(diagnostics->mic_queue_total_frames));
    snprintf(mic_queue_dropped_frames, sizeof(mic_queue_dropped_frames), "%lu", static_cast<unsigned long>(diagnostics->mic_queue_dropped_frames));
    snprintf(audio_ws_sent_audio_frames, sizeof(audio_ws_sent_audio_frames), "%lu", static_cast<unsigned long>(diagnostics->audio_ws_sent_audio_frames));
    snprintf(mic_last_abs_peak, sizeof(mic_last_abs_peak), "%d", diagnostics->mic_last_abs_peak);
    snprintf(mic_last_nonzero_samples, sizeof(mic_last_nonzero_samples), "%u", static_cast<unsigned>(diagnostics->mic_last_nonzero_samples));
    snprintf(playback_buffer_queued_chunks, sizeof(playback_buffer_queued_chunks), "%u", static_cast<unsigned>(diagnostics->playback_buffer_queued_chunks));
    snprintf(playback_buffer_total_chunks, sizeof(playback_buffer_total_chunks), "%lu", static_cast<unsigned long>(diagnostics->playback_buffer_total_chunks));
    snprintf(playback_buffer_dropped_chunks, sizeof(playback_buffer_dropped_chunks), "%lu", static_cast<unsigned long>(diagnostics->playback_buffer_dropped_chunks));
    snprintf(playback_buffer_clear_count, sizeof(playback_buffer_clear_count), "%lu", static_cast<unsigned long>(diagnostics->playback_buffer_clear_count));
    snprintf(speaker_frames_played, sizeof(speaker_frames_played), "%lu", static_cast<unsigned long>(diagnostics->speaker_frames_played));
    snprintf(speaker_busy_ticks, sizeof(speaker_busy_ticks), "%lu", static_cast<unsigned long>(diagnostics->speaker_busy_ticks));
    snprintf(speaker_driver_errors, sizeof(speaker_driver_errors), "%lu", static_cast<unsigned long>(diagnostics->speaker_driver_errors));
    echo["mic_frames_captured"] = mic_frames_captured;
    echo["mic_driver_errors"] = mic_driver_errors;
    echo["mic_skipped_render_state"] = mic_skipped_render_state;
    echo["mic_skipped_speaker_busy"] = mic_skipped_speaker_busy;
    echo["mic_skipped_unavailable"] = mic_skipped_unavailable;
    echo["mic_queue_depth"] = mic_queue_depth;
    echo["mic_queue_total_frames"] = mic_queue_total_frames;
    echo["mic_queue_dropped_frames"] = mic_queue_dropped_frames;
    echo["audio_ws_sent_audio_frames"] = audio_ws_sent_audio_frames;
    echo["mic_last_abs_peak"] = mic_last_abs_peak;
    echo["mic_last_nonzero_samples"] = mic_last_nonzero_samples;
    echo["playback_buffer_queued_chunks"] = playback_buffer_queued_chunks;
    echo["playback_buffer_total_chunks"] = playback_buffer_total_chunks;
    echo["playback_buffer_dropped_chunks"] = playback_buffer_dropped_chunks;
    echo["playback_buffer_clear_count"] = playback_buffer_clear_count;
    echo["speaker_frames_played"] = speaker_frames_played;
    echo["speaker_busy_ticks"] = speaker_busy_ticks;
    echo["speaker_driver_errors"] = speaker_driver_errors;
    echo["speaker_last_stream_id"] = diagnostics->speaker_last_stream_id;
  }

  const size_t written = serializeJson(doc, output, output_size);
  return written > 0 && written < output_size;
}

inline bool a21GatewayWSSendDeviceEvent(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    uint32_t now_ms) {
  return a21GatewayWSSendDeviceEventWithTouchSource(
      runtime,
      driver,
      connection,
      state,
      event,
      mode,
      text,
      "",
      now_ms);
}

inline bool a21GatewayWSSendDeviceEventWithTouchSource(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const char* event,
    const char* mode,
    const char* text,
    const char* touch_source,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21GatewayWSDriverReady(driver) || connection == nullptr || state == nullptr) {
    return false;
  }
  if (connection->phase != A21_CONN_GATEWAY_CONNECTED || !driver->connected(driver->ctx)) {
    return false;
  }

  char message[A21_WS_TEXT_MESSAGE_CAP];
  if (!a21GatewayWSBuildDeviceEventWithTouchSource(runtime, state, event, mode, text, touch_source, now_ms, message, sizeof(message))) {
    return false;
  }
  if (!driver->send_text(driver->ctx, message)) {
    return false;
  }
  runtime->sent_device_events += 1;
  runtime->next_seq += 1;
  return true;
}

inline bool a21RuntimeEchoDiagnosticsEqual(
    const A21RuntimeEchoDiagnostics* left,
    const A21RuntimeEchoDiagnostics* right) {
  if (left == nullptr || right == nullptr) {
    return left == right;
  }
  return left->enabled == right->enabled &&
         left->mic_frames_captured == right->mic_frames_captured &&
         left->mic_driver_errors == right->mic_driver_errors &&
         left->mic_skipped_render_state == right->mic_skipped_render_state &&
         left->mic_skipped_speaker_busy == right->mic_skipped_speaker_busy &&
         left->mic_skipped_unavailable == right->mic_skipped_unavailable &&
         left->mic_queue_depth == right->mic_queue_depth &&
         left->mic_queue_total_frames == right->mic_queue_total_frames &&
         left->mic_queue_dropped_frames == right->mic_queue_dropped_frames &&
         left->audio_ws_sent_audio_frames == right->audio_ws_sent_audio_frames &&
         left->mic_last_abs_peak == right->mic_last_abs_peak &&
         left->mic_last_nonzero_samples == right->mic_last_nonzero_samples &&
         left->playback_buffer_queued_chunks == right->playback_buffer_queued_chunks &&
         left->playback_buffer_total_chunks == right->playback_buffer_total_chunks &&
         left->playback_buffer_dropped_chunks == right->playback_buffer_dropped_chunks &&
         left->playback_buffer_clear_count == right->playback_buffer_clear_count &&
         left->speaker_frames_played == right->speaker_frames_played &&
         left->speaker_busy_ticks == right->speaker_busy_ticks &&
         left->speaker_driver_errors == right->speaker_driver_errors &&
         a21StringEquals(left->speaker_last_stream_id, right->speaker_last_stream_id);
}

inline bool a21RuntimeEchoDiagnosticsChanged(
    const A21GatewayWSRuntime* runtime,
    const A21RuntimeEchoDiagnostics* diagnostics) {
  if (runtime == nullptr || diagnostics == nullptr || !diagnostics->enabled) {
    return false;
  }
  return !runtime->has_runtime_echo_diagnostics ||
         !a21RuntimeEchoDiagnosticsEqual(&runtime->last_runtime_echo_diagnostics, diagnostics);
}

inline bool a21GatewayWSSendRuntimeEchoIfChangedWithDiagnostics(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const A21MotionRuntime* motion_runtime,
    const A21RGBRuntime* rgb_runtime,
    const A21RuntimeEchoDiagnostics* diagnostics,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21GatewayWSDriverReady(driver) || connection == nullptr ||
      state == nullptr || motion_runtime == nullptr || rgb_runtime == nullptr) {
    return false;
  }
  if (!motion_runtime->has_y || !rgb_runtime->has_color) {
    return true;
  }
  if (runtime->has_runtime_echo &&
      runtime->last_runtime_echo_render_state == state->render_state &&
      runtime->last_runtime_echo_y_deg == motion_runtime->last_y_deg &&
      a21RGBColorEquals(runtime->last_runtime_echo_rgb, rgb_runtime->last_color) &&
      !a21RuntimeEchoDiagnosticsChanged(runtime, diagnostics)) {
    return true;
  }
  if (connection->phase != A21_CONN_GATEWAY_CONNECTED || !driver->connected(driver->ctx)) {
    return false;
  }

  char message[A21_WS_TEXT_MESSAGE_CAP];
  if (!a21GatewayWSBuildRuntimeEchoEvent(runtime, state, motion_runtime, rgb_runtime, diagnostics, now_ms, message, sizeof(message))) {
    return false;
  }
  if (!driver->send_text(driver->ctx, message)) {
    return false;
  }
  runtime->sent_device_events += 1;
  runtime->next_seq += 1;
  runtime->has_runtime_echo = true;
  runtime->last_runtime_echo_render_state = state->render_state;
  runtime->last_runtime_echo_y_deg = motion_runtime->last_y_deg;
  runtime->last_runtime_echo_rgb = rgb_runtime->last_color;
  if (diagnostics != nullptr && diagnostics->enabled) {
    runtime->has_runtime_echo_diagnostics = true;
    runtime->last_runtime_echo_diagnostics = *diagnostics;
  }
  return true;
}

inline bool a21GatewayWSSendRuntimeEchoIfChanged(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    const A21MotionRuntime* motion_runtime,
    const A21RGBRuntime* rgb_runtime,
    uint32_t now_ms) {
  return a21GatewayWSSendRuntimeEchoIfChangedWithDiagnostics(
      runtime,
      driver,
      connection,
      state,
      motion_runtime,
      rgb_runtime,
      nullptr,
      now_ms);
}

inline void a21GatewayWSApplyText(
    A21GatewayWSRuntime* runtime,
    A21FirmwareState* state,
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
  runtime->invalid_control_events += 1;
  a21CopyString(state->last_error, A21_ERROR_CAP, event.error);
  a21CopyString(state->text, A21_TEXT_CAP, "A21 control message error");
  state->render_state = A21_RENDER_ERROR;
  state->updated_at_ms = now_ms;
}

inline void a21GatewayWSDrainText(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    A21FirmwareState* state,
    uint32_t now_ms) {
  char text[A21_WS_TEXT_MESSAGE_CAP];
  uint8_t guard = 0;
  while (guard < 4 && driver->read_text(driver->ctx, text, sizeof(text))) {
    a21GatewayWSApplyText(runtime, state, text, now_ms);
    ++guard;
  }
}

inline void a21GatewayWSRuntimeTick(
    A21GatewayWSRuntime* runtime,
    const A21GatewayWSDriver* driver,
    A21ConnectionState* connection,
    const A21NetworkConfig* network,
    A21FirmwareState* state,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21GatewayWSDriverReady(driver) || connection == nullptr || network == nullptr || state == nullptr) {
    return;
  }

  if (connection->phase == A21_CONN_GATEWAY_CONNECTED) {
    driver->loop(driver->ctx);
    if (!driver->connected(driver->ctx)) {
      runtime->begin_sent = false;
      runtime->last_begin_at_ms = 0;
      a21ConnectionOnGatewayDisconnected(connection, "ws_disconnected", now_ms);
      return;
    }
    a21GatewayWSDrainText(runtime, driver, state, now_ms);
    return;
  }

  if (connection->phase != A21_CONN_GATEWAY_CONNECTING) {
    return;
  }

  driver->loop(driver->ctx);
  if (driver->connected(driver->ctx)) {
    a21ConnectionOnGatewayConnected(connection, now_ms);
    a21GatewayWSDrainText(runtime, driver, state, now_ms);
    return;
  }

  if (!runtime->begin_sent) {
    if (driver->begin(driver->ctx, network->gateway_host, network->gateway_port, network->control_path)) {
      runtime->begin_sent = true;
      runtime->last_begin_at_ms = now_ms;
    } else {
      a21ConnectionOnGatewayDisconnected(connection, "ws_begin_failed", now_ms);
    }
  }
}
