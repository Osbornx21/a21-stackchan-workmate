#pragma once

#include "a21_firmware_gateway_ws.h"

#include <stdint.h>

enum A21TouchSource {
  A21_TOUCH_SOURCE_SCREEN,
  A21_TOUCH_SOURCE_TOP_SENSOR,
};

enum A21TouchIntent {
  A21_TOUCH_INTENT_NONE,
  A21_TOUCH_INTENT_WAKE_OR_LISTEN,
  A21_TOUCH_INTENT_BARGE_IN,
};

struct A21TouchSample {
  A21TouchSource source;
  A21TouchIntent intent;
};

struct A21TouchDriver {
  void* ctx;
  bool (*read)(void* ctx, A21TouchSample* sample);
};

struct A21TouchRuntime {
  uint32_t handled_count;
  uint32_t ignored_count;
  A21TouchSource last_source;
  A21TouchIntent last_intent;
};

inline void a21InitTouchRuntime(A21TouchRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->handled_count = 0;
  runtime->ignored_count = 0;
  runtime->last_source = A21_TOUCH_SOURCE_SCREEN;
  runtime->last_intent = A21_TOUCH_INTENT_NONE;
}

inline bool a21TouchDriverReady(const A21TouchDriver* driver) {
  return driver != nullptr && driver->read != nullptr;
}

inline const char* a21TouchSourceName(A21TouchSource source) {
  switch (source) {
    case A21_TOUCH_SOURCE_TOP_SENSOR:
      return "top_sensor";
    case A21_TOUCH_SOURCE_SCREEN:
    default:
      return "screen";
  }
}

inline bool a21TouchIntentToDeviceEvent(
    A21TouchIntent intent,
    const char** event,
    const char** mode,
    const char** text) {
  if (event == nullptr || mode == nullptr || text == nullptr) {
    return false;
  }
  *mode = "workmate";
  switch (intent) {
    case A21_TOUCH_INTENT_WAKE_OR_LISTEN:
      *event = "touch.wake_or_listen";
      *text = "先说，我在";
      return true;
    case A21_TOUCH_INTENT_BARGE_IN:
      *event = "touch.barge_in";
      *text = "";
      return true;
    case A21_TOUCH_INTENT_NONE:
    default:
      *event = "";
      *text = "";
      return false;
  }
}

inline bool a21TouchRuntimeTick(
    A21TouchRuntime* runtime,
    const A21TouchDriver* touch_driver,
    A21GatewayWSRuntime* gateway_runtime,
    const A21GatewayWSDriver* gateway_driver,
    const A21ConnectionState* connection,
    const A21FirmwareState* state,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21TouchDriverReady(touch_driver) || gateway_runtime == nullptr || gateway_driver == nullptr || connection == nullptr || state == nullptr) {
    return false;
  }

  bool ok = true;
  A21TouchSample sample;
  uint8_t guard = 0;
  while (guard < 4 && touch_driver->read(touch_driver->ctx, &sample)) {
    ++guard;
    if (sample.intent == A21_TOUCH_INTENT_NONE) {
      continue;
    }
    const char* event = "";
    const char* mode = "";
    const char* text = "";
    if (!a21TouchIntentToDeviceEvent(sample.intent, &event, &mode, &text)) {
      runtime->ignored_count += 1;
      ok = false;
      continue;
    }
    const bool sent = a21GatewayWSSendDeviceEventWithTouchSource(
        gateway_runtime,
        gateway_driver,
        connection,
        state,
        event,
        mode,
        text,
        a21TouchSourceName(sample.source),
        now_ms);
    if (!sent) {
      runtime->ignored_count += 1;
      ok = false;
      continue;
    }
    runtime->handled_count += 1;
    runtime->last_source = sample.source;
    runtime->last_intent = sample.intent;
  }
  return ok;
}
