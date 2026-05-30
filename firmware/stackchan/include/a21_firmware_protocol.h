#pragma once

#include <ArduinoJson.h>

#include <stddef.h>
#include <string.h>

static constexpr size_t A21_TRACE_ID_CAP = 64;
static constexpr size_t A21_SESSION_ID_CAP = 64;
static constexpr size_t A21_STATE_CAP = 24;
static constexpr size_t A21_MODE_CAP = 24;
static constexpr size_t A21_TEXT_CAP = 192;
static constexpr size_t A21_STREAM_ID_CAP = 64;
static constexpr size_t A21_ERROR_CAP = 48;

struct A21ControlEvent {
  char trace_id[A21_TRACE_ID_CAP];
  char session_id[A21_SESSION_ID_CAP];
  char state[A21_STATE_CAP];
  char mode[A21_MODE_CAP];
  char text[A21_TEXT_CAP];
  char stream_id[A21_STREAM_ID_CAP];
  char error[A21_ERROR_CAP];
  uint16_t diagnostic_tone_hz;
  uint16_t diagnostic_tone_duration_ms;
  uint8_t diagnostic_tone_volume;
  bool final;
};

inline void a21CopyString(char* target, size_t target_size, const char* source) {
  if (target_size == 0) {
    return;
  }
  if (source == nullptr) {
    source = "";
  }
  strncpy(target, source, target_size - 1);
  target[target_size - 1] = '\0';
}

inline void a21ResetControlEvent(A21ControlEvent* event) {
  if (event == nullptr) {
    return;
  }
  a21CopyString(event->trace_id, A21_TRACE_ID_CAP, "");
  a21CopyString(event->session_id, A21_SESSION_ID_CAP, "");
  a21CopyString(event->state, A21_STATE_CAP, "");
  a21CopyString(event->mode, A21_MODE_CAP, "");
  a21CopyString(event->text, A21_TEXT_CAP, "");
  a21CopyString(event->stream_id, A21_STREAM_ID_CAP, "");
  a21CopyString(event->error, A21_ERROR_CAP, "");
  event->diagnostic_tone_hz = 0;
  event->diagnostic_tone_duration_ms = 0;
  event->diagnostic_tone_volume = 0;
  event->final = false;
}

inline bool a21FailControlEvent(A21ControlEvent* event, const char* field) {
  if (event != nullptr) {
    a21CopyString(event->error, A21_ERROR_CAP, field);
  }
  return false;
}

inline bool a21StringEquals(const char* actual, const char* expected) {
  return actual != nullptr && expected != nullptr && strcmp(actual, expected) == 0;
}

inline bool a21ParseControlEvent(const char* json, const char* expected_device_id, A21ControlEvent* event) {
  if (event == nullptr) {
    return false;
  }
  a21ResetControlEvent(event);
  if (json == nullptr || expected_device_id == nullptr) {
    return a21FailControlEvent(event, "input");
  }

  JsonDocument doc;
  DeserializationError err = deserializeJson(doc, json);
  if (err) {
    return a21FailControlEvent(event, "json");
  }

  const char* protocol = doc["protocol"] | "";
  if (!a21StringEquals(protocol, "a21.device.v1")) {
    return a21FailControlEvent(event, "protocol");
  }

  const char* device_id = doc["device_id"] | "";
  if (!a21StringEquals(device_id, expected_device_id)) {
    return a21FailControlEvent(event, "device_id");
  }

  const char* kind = doc["kind"] | "";
  if (!a21StringEquals(kind, "control.event")) {
    return a21FailControlEvent(event, "kind");
  }

  JsonVariantConst payload = doc["payload"];
  if (payload.isNull()) {
    return a21FailControlEvent(event, "payload");
  }

  const char* state = payload["state"] | "";
  const char* mode = payload["mode"] | "";
  if (state[0] == '\0') {
    return a21FailControlEvent(event, "state");
  }
  if (mode[0] == '\0') {
    return a21FailControlEvent(event, "mode");
  }

  a21CopyString(event->trace_id, A21_TRACE_ID_CAP, doc["trace_id"] | "");
  a21CopyString(event->session_id, A21_SESSION_ID_CAP, doc["session_id"] | "");
  a21CopyString(event->state, A21_STATE_CAP, state);
  a21CopyString(event->mode, A21_MODE_CAP, mode);
  a21CopyString(event->text, A21_TEXT_CAP, payload["text"] | "");
  a21CopyString(event->stream_id, A21_STREAM_ID_CAP, payload["stream_id"] | "");
  event->diagnostic_tone_hz = payload["diagnostic_tone_hz"] | 0;
  event->diagnostic_tone_duration_ms = payload["diagnostic_tone_duration_ms"] | 0;
  event->diagnostic_tone_volume = payload["diagnostic_tone_volume"] | 0;
  event->final = payload["final"] | false;
  return true;
}
