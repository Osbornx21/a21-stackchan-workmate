#pragma once

#include "a21_firmware_connection.h"
#include "a21_firmware_network.h"
#include "a21_firmware_state.h"

#include <stddef.h>
#include <stdint.h>

static constexpr size_t A21_WS_TEXT_MESSAGE_CAP = 768;

struct A21GatewayWSDriver {
  void* ctx;
  bool (*begin)(void* ctx, const char* host, uint16_t port, const char* path);
  void (*loop)(void* ctx);
  bool (*connected)(void* ctx);
  bool (*read_text)(void* ctx, char* output, size_t output_size);
};

struct A21GatewayWSRuntime {
  bool begin_sent;
  uint32_t last_begin_at_ms;
  uint32_t received_control_events;
  uint32_t invalid_control_events;
};

inline void a21InitGatewayWSRuntime(A21GatewayWSRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->begin_sent = false;
  runtime->last_begin_at_ms = 0;
  runtime->received_control_events = 0;
  runtime->invalid_control_events = 0;
}

inline bool a21GatewayWSDriverReady(const A21GatewayWSDriver* driver) {
  return driver != nullptr &&
         driver->begin != nullptr &&
         driver->loop != nullptr &&
         driver->connected != nullptr &&
         driver->read_text != nullptr;
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
