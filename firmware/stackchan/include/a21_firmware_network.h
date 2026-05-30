#pragma once

#include "a21_firmware_config.h"
#include "a21_firmware_protocol.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#ifndef A21_GATEWAY_HOST
#define A21_GATEWAY_HOST "10.21.0.1"
#endif

#ifndef A21_GATEWAY_PORT
#define A21_GATEWAY_PORT 21080
#endif

#ifndef A21_CONTROL_WS_PATH
#define A21_CONTROL_WS_PATH "/ws/control"
#endif

#ifndef A21_AUDIO_WS_PATH
#define A21_AUDIO_WS_PATH "/ws/audio?device_id=" A21_DEVICE_ID
#endif

static constexpr size_t A21_GATEWAY_HOST_CAP = 64;
static constexpr size_t A21_WS_PATH_CAP = 64;
static constexpr size_t A21_WS_URL_CAP = 128;

struct A21NetworkConfig {
  char gateway_host[A21_GATEWAY_HOST_CAP];
  uint16_t gateway_port;
  char control_path[A21_WS_PATH_CAP];
  char audio_path[A21_WS_PATH_CAP];
};

inline void a21InitNetworkConfig(A21NetworkConfig* config) {
  if (config == nullptr) {
    return;
  }
  a21CopyString(config->gateway_host, A21_GATEWAY_HOST_CAP, A21_GATEWAY_HOST);
  config->gateway_port = A21_GATEWAY_PORT;
  a21CopyString(config->control_path, A21_WS_PATH_CAP, A21_CONTROL_WS_PATH);
  a21CopyString(config->audio_path, A21_WS_PATH_CAP, A21_AUDIO_WS_PATH);
}

inline char a21LowerASCII(char c) {
  if (c >= 'A' && c <= 'Z') {
    return static_cast<char>(c - 'A' + 'a');
  }
  return c;
}

inline bool a21ContainsTokenCaseInsensitive(const char* text, const char* token) {
  if (text == nullptr || token == nullptr || token[0] == '\0') {
    return false;
  }
  const size_t token_len = strlen(token);
  for (size_t i = 0; text[i] != '\0'; ++i) {
    size_t j = 0;
    while (j < token_len && text[i + j] != '\0' && a21LowerASCII(text[i + j]) == a21LowerASCII(token[j])) {
      ++j;
    }
    if (j == token_len) {
      return true;
    }
  }
  return false;
}

inline bool a21ContainsForbiddenFirmwareRoute(const char* text) {
  return a21ContainsTokenCaseInsensitive(text, "x21") ||
         a21ContainsTokenCaseInsensitive(text, "v21");
}

inline bool a21IsLegacyGatewayPort(uint16_t port) {
  return port == 8000 || port == 8080 || port == 10095 || port == 18080 || port == 4173;
}

inline bool a21ValidWSPath(const char* path) {
  if (path == nullptr || path[0] != '/') {
    return false;
  }
  return !a21ContainsForbiddenFirmwareRoute(path);
}

inline bool a21ValidateNetworkConfig(const A21NetworkConfig* config) {
  if (config == nullptr) {
    return false;
  }
  if (config->gateway_host[0] == '\0' || a21ContainsForbiddenFirmwareRoute(config->gateway_host)) {
    return false;
  }
  if (config->gateway_port == 0 || a21IsLegacyGatewayPort(config->gateway_port)) {
    return false;
  }
  if (!a21ValidWSPath(config->control_path) || !a21ValidWSPath(config->audio_path)) {
    return false;
  }
  return true;
}

inline bool a21BuildWSURL(const A21NetworkConfig* config, const char* path, char* output, size_t output_size) {
  if (output == nullptr || output_size == 0) {
    return false;
  }
  output[0] = '\0';
  if (config == nullptr || path == nullptr || !a21ValidateNetworkConfig(config)) {
    return false;
  }
  const int written = snprintf(output, output_size, "ws://%s:%u%s", config->gateway_host, config->gateway_port, path);
  return written > 0 && static_cast<size_t>(written) < output_size;
}

inline bool a21BuildControlWSURL(const A21NetworkConfig* config, char* output, size_t output_size) {
  if (config == nullptr) {
    return false;
  }
  return a21BuildWSURL(config, config->control_path, output, output_size);
}

inline bool a21BuildAudioWSURL(const A21NetworkConfig* config, char* output, size_t output_size) {
  if (config == nullptr) {
    return false;
  }
  return a21BuildWSURL(config, config->audio_path, output, output_size);
}
