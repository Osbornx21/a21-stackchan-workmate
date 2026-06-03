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

struct A21WiFiProvisioningDriver {
  void* ctx;
  bool (*start)(void* ctx, A21WiFiProvisioningMethod method);
};

struct A21WiFiRuntime {
  bool begin_sent;
  bool provisioning_started;
  uint32_t last_begin_at_ms;
};

inline void a21InitWiFiRuntime(A21WiFiRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->begin_sent = false;
  runtime->provisioning_started = false;
  runtime->last_begin_at_ms = 0;
}

inline bool a21WiFiDriverReady(const A21WiFiDriver* driver) {
  return driver != nullptr && driver->begin != nullptr && driver->status != nullptr && driver->local_ip != nullptr;
}

inline bool a21WiFiProvisioningDriverReady(const A21WiFiProvisioningDriver* driver) {
  return driver != nullptr && driver->start != nullptr;
}

inline void a21WiFiRuntimeTick(
    A21WiFiRuntime* runtime,
    const A21WiFiDriver* driver,
    A21ConnectionState* connection,
    const A21WiFiConfig* wifi,
    uint32_t now_ms);

inline void a21WiFiRuntimeTickWithProvisioning(
    A21WiFiRuntime* runtime,
    const A21WiFiDriver* driver,
    const A21WiFiProvisioningDriver* provisioning,
    A21ConnectionState* connection,
    const A21WiFiConfig* wifi,
    uint32_t now_ms) {
  if (runtime == nullptr || connection == nullptr) {
    return;
  }

  if (connection->phase == A21_CONN_WIFI_PROVISIONING) {
    runtime->begin_sent = false;
    runtime->last_begin_at_ms = 0;
    if (!runtime->provisioning_started && a21WiFiProvisioningDriverReady(provisioning)) {
      if (provisioning->start(provisioning->ctx, a21DefaultWiFiProvisioningMethod())) {
        runtime->provisioning_started = true;
      } else {
        a21SetConnectionPhase(connection, A21_CONN_LOCAL_FALLBACK, now_ms);
        a21CopyString(connection->last_error, A21_ERROR_CAP, "wifi_provisioning_failed");
      }
    }
    return;
  }

  if (!a21WiFiDriverReady(driver) || !a21ValidateWiFiConfig(wifi)) {
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
    runtime->provisioning_started = false;
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
    runtime->provisioning_started = false;
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

inline void a21WiFiRuntimeTick(
    A21WiFiRuntime* runtime,
    const A21WiFiDriver* driver,
    A21ConnectionState* connection,
    const A21WiFiConfig* wifi,
    uint32_t now_ms) {
  a21WiFiRuntimeTickWithProvisioning(runtime, driver, nullptr, connection, wifi, now_ms);
}
