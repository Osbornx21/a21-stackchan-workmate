#pragma once

#include "a21_firmware_network.h"
#include "a21_firmware_protocol.h"
#include "a21_firmware_wifi.h"

#include <stddef.h>
#include <stdint.h>

static constexpr size_t A21_CONNECTION_TEXT_CAP = 32;
static constexpr size_t A21_CONNECTION_IP_CAP = 40;
static constexpr uint32_t A21_RECONNECT_BASE_MS = 1000;
static constexpr uint32_t A21_RECONNECT_MAX_MS = 30000;

enum A21ConnectionPhase {
  A21_CONN_LOCAL_FALLBACK,
  A21_CONN_WIFI_CONNECTING,
  A21_CONN_GATEWAY_CONNECTING,
  A21_CONN_GATEWAY_CONNECTED,
  A21_CONN_RECONNECT_WAIT,
  A21_CONN_ERROR,
};

struct A21ConnectionState {
  A21ConnectionPhase phase;
  char status_text[A21_CONNECTION_TEXT_CAP];
  char local_ip[A21_CONNECTION_IP_CAP];
  char last_error[A21_ERROR_CAP];
  uint8_t reconnect_attempt;
  uint32_t updated_at_ms;
  uint32_t next_retry_at_ms;
};

inline const char* a21ConnectionStatusForPhase(A21ConnectionPhase phase) {
  switch (phase) {
    case A21_CONN_WIFI_CONNECTING:
      return "Wi-Fi connecting";
    case A21_CONN_GATEWAY_CONNECTING:
      return "Gateway connecting";
    case A21_CONN_GATEWAY_CONNECTED:
      return "Gateway connected";
    case A21_CONN_RECONNECT_WAIT:
      return "Reconnect wait";
    case A21_CONN_ERROR:
      return "Connection error";
    case A21_CONN_LOCAL_FALLBACK:
    default:
      return "Local fallback";
  }
}

inline void a21SetConnectionPhase(A21ConnectionState* connection, A21ConnectionPhase phase, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  connection->phase = phase;
  connection->updated_at_ms = now_ms;
  a21CopyString(connection->status_text, A21_CONNECTION_TEXT_CAP, a21ConnectionStatusForPhase(phase));
}

inline uint32_t a21ReconnectBackoffMs(uint8_t attempt) {
  if (attempt == 0) {
    return 0;
  }
  uint32_t backoff_ms = A21_RECONNECT_BASE_MS;
  for (uint8_t i = 1; i < attempt; ++i) {
    if (backoff_ms >= A21_RECONNECT_MAX_MS / 2) {
      return A21_RECONNECT_MAX_MS;
    }
    backoff_ms *= 2;
  }
  return backoff_ms > A21_RECONNECT_MAX_MS ? A21_RECONNECT_MAX_MS : backoff_ms;
}

inline void a21InitConnectionState(A21ConnectionState* connection, const A21NetworkConfig* config, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  a21CopyString(connection->status_text, A21_CONNECTION_TEXT_CAP, "");
  a21CopyString(connection->local_ip, A21_CONNECTION_IP_CAP, "");
  a21CopyString(connection->last_error, A21_ERROR_CAP, "");
  connection->reconnect_attempt = 0;
  connection->updated_at_ms = now_ms;
  connection->next_retry_at_ms = 0;

  if (!a21ValidateNetworkConfig(config)) {
    a21SetConnectionPhase(connection, A21_CONN_LOCAL_FALLBACK, now_ms);
    a21CopyString(connection->last_error, A21_ERROR_CAP, "invalid_config");
    return;
  }

  a21SetConnectionPhase(connection, A21_CONN_WIFI_CONNECTING, now_ms);
}

inline void a21InitConnectionStateWithWiFi(
    A21ConnectionState* connection,
    const A21NetworkConfig* network,
    const A21WiFiConfig* wifi,
    uint32_t now_ms) {
  a21InitConnectionState(connection, network, now_ms);
  if (connection == nullptr || connection->phase == A21_CONN_LOCAL_FALLBACK) {
    return;
  }
  if (!a21ValidateWiFiConfig(wifi)) {
    a21SetConnectionPhase(connection, A21_CONN_LOCAL_FALLBACK, now_ms);
    a21CopyString(connection->last_error, A21_ERROR_CAP, (wifi == nullptr || wifi->ssid[0] == '\0') ? "missing_wifi" : "invalid_wifi");
  }
}

inline void a21ConnectionOnWiFiConnected(A21ConnectionState* connection, const char* local_ip, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  a21CopyString(connection->local_ip, A21_CONNECTION_IP_CAP, local_ip);
  a21CopyString(connection->last_error, A21_ERROR_CAP, "");
  a21SetConnectionPhase(connection, A21_CONN_GATEWAY_CONNECTING, now_ms);
}

inline void a21ConnectionOnGatewayConnected(A21ConnectionState* connection, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  connection->reconnect_attempt = 0;
  connection->next_retry_at_ms = 0;
  a21CopyString(connection->last_error, A21_ERROR_CAP, "");
  a21SetConnectionPhase(connection, A21_CONN_GATEWAY_CONNECTED, now_ms);
}

inline void a21ConnectionEnterReconnect(A21ConnectionState* connection, const char* reason, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  if (connection->reconnect_attempt < 250) {
    connection->reconnect_attempt += 1;
  }
  connection->next_retry_at_ms = now_ms + a21ReconnectBackoffMs(connection->reconnect_attempt);
  a21CopyString(connection->last_error, A21_ERROR_CAP, reason);
  a21SetConnectionPhase(connection, A21_CONN_RECONNECT_WAIT, now_ms);
}

inline void a21ConnectionOnGatewayDisconnected(A21ConnectionState* connection, const char* reason, uint32_t now_ms) {
  a21ConnectionEnterReconnect(connection, reason, now_ms);
}

inline void a21ConnectionOnWiFiDisconnected(A21ConnectionState* connection, const char* reason, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  a21CopyString(connection->local_ip, A21_CONNECTION_IP_CAP, "");
  a21ConnectionEnterReconnect(connection, reason, now_ms);
}

inline bool a21ConnectionRetryDue(const A21ConnectionState* connection, uint32_t now_ms) {
  if (connection == nullptr || connection->phase != A21_CONN_RECONNECT_WAIT) {
    return false;
  }
  return now_ms >= connection->next_retry_at_ms;
}

inline void a21ConnectionStartRetry(A21ConnectionState* connection, uint32_t now_ms) {
  if (connection == nullptr) {
    return;
  }
  connection->next_retry_at_ms = 0;
  a21SetConnectionPhase(connection, A21_CONN_WIFI_CONNECTING, now_ms);
}
