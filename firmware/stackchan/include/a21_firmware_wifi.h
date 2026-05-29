#pragma once

#include "a21_firmware_network.h"
#include "a21_firmware_protocol.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#if __has_include("a21_firmware_secrets.local.h")
#include "a21_firmware_secrets.local.h"
#endif

#ifndef A21_WIFI_SSID
#define A21_WIFI_SSID ""
#endif

#ifndef A21_WIFI_PASSWORD
#define A21_WIFI_PASSWORD ""
#endif

static constexpr size_t A21_WIFI_SSID_CAP = 33;
static constexpr size_t A21_WIFI_PASSWORD_CAP = 65;

struct A21WiFiConfig {
  char ssid[A21_WIFI_SSID_CAP];
  char password[A21_WIFI_PASSWORD_CAP];
};

inline void a21InitWiFiConfig(A21WiFiConfig* config) {
  if (config == nullptr) {
    return;
  }
  a21CopyString(config->ssid, A21_WIFI_SSID_CAP, A21_WIFI_SSID);
  a21CopyString(config->password, A21_WIFI_PASSWORD_CAP, A21_WIFI_PASSWORD);
}

inline bool a21ValidateWiFiConfig(const A21WiFiConfig* config) {
  if (config == nullptr || config->ssid[0] == '\0') {
    return false;
  }
  return !a21ContainsForbiddenFirmwareRoute(config->ssid);
}

inline bool a21DescribeWiFiConfig(const A21WiFiConfig* config, char* output, size_t output_size) {
  if (output == nullptr || output_size == 0) {
    return false;
  }
  output[0] = '\0';
  if (config == nullptr || config->ssid[0] == '\0') {
    const int written = snprintf(output, output_size, "ssid=missing password=missing");
    return written > 0 && static_cast<size_t>(written) < output_size;
  }
  const char* password_status = config->password[0] == '\0' ? "missing" : "configured";
  const int written = snprintf(output, output_size, "ssid=%s password=%s", config->ssid, password_status);
  return written > 0 && static_cast<size_t>(written) < output_size;
}
