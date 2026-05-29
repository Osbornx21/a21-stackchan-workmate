#pragma once

#include "a21_firmware_connection.h"
#include "a21_firmware_wifi.h"

#include <stddef.h>
#include <stdint.h>

enum A21WiFiDriverStatus {
  A21_WIFI_DRIVER_DISCONNECTED,
  A21_WIFI_DRIVER_CONNECTING,
  A21_WIFI_DRIVER_CONNECTED,
};

struct A21WiFiDriver {
  void* ctx;
  bool (*begin)(void* ctx, const char* ssid, const char* password);
  A21WiFiDriverStatus (*status)(void* ctx);
  bool (*local_ip)(void* ctx, char* output, size_t output_size);
};

struct A21WiFiRuntime {
  bool begin_sent;
  uint32_t last_begin_at_ms;
};

inline void a21InitWiFiRuntime(A21WiFiRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->begin_sent = false;
  runtime->last_begin_at_ms = 0;
}

inline bool a21WiFiDriverReady(const A21WiFiDriver* driver) {
  return driver != nullptr && driver->begin != nullptr && driver->status != nullptr && driver->local_ip != nullptr;
}

inline void a21WiFiRuntimeTick(
    A21WiFiRuntime* runtime,
    const A21WiFiDriver* driver,
    A21ConnectionState* connection,
    const A21WiFiConfig* wifi,
    uint32_t now_ms) {
  if (runtime == nullptr || !a21WiFiDriverReady(driver) || connection == nullptr || !a21ValidateWiFiConfig(wifi)) {
    return;
  }

  const A21WiFiDriverStatus status = driver->status(driver->ctx);

  if ((connection->phase == A21_CONN_GATEWAY_CONNECTING || connection->phase == A21_CONN_GATEWAY_CONNECTED) &&
      status == A21_WIFI_DRIVER_DISCONNECTED) {
    runtime->begin_sent = false;
    runtime->last_begin_at_ms = 0;
    a21ConnectionOnWiFiDisconnected(connection, "wifi_disconnected", now_ms);
    return;
  }

  if (connection->phase == A21_CONN_RECONNECT_WAIT && a21ConnectionRetryDue(connection, now_ms)) {
    runtime->begin_sent = false;
    runtime->last_begin_at_ms = 0;
    a21ConnectionStartRetry(connection, now_ms);
  }

  if (connection->phase != A21_CONN_WIFI_CONNECTING) {
    return;
  }

  if (status == A21_WIFI_DRIVER_CONNECTED) {
    char local_ip[A21_CONNECTION_IP_CAP];
    if (!driver->local_ip(driver->ctx, local_ip, sizeof(local_ip)) || local_ip[0] == '\0') {
      a21CopyString(local_ip, sizeof(local_ip), "0.0.0.0");
    }
    a21ConnectionOnWiFiConnected(connection, local_ip, now_ms);
    runtime->begin_sent = false;
    runtime->last_begin_at_ms = 0;
    return;
  }

  if (!runtime->begin_sent) {
    if (driver->begin(driver->ctx, wifi->ssid, wifi->password)) {
      runtime->begin_sent = true;
      runtime->last_begin_at_ms = now_ms;
    } else {
      a21ConnectionOnWiFiDisconnected(connection, "wifi_begin_failed", now_ms);
    }
  }
}
