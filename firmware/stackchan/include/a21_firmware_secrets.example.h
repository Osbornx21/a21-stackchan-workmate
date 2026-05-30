#pragma once

// Copy this file to a21_firmware_secrets.local.h for local hardware bring-up.
// The local file is ignored by git and must never be committed.

#undef A21_GATEWAY_HOST
#define A21_GATEWAY_HOST "10.21.0.1"

#undef A21_WIFI_SSID
#define A21_WIFI_SSID "A21-LAB"

#undef A21_WIFI_PASSWORD
#define A21_WIFI_PASSWORD "replace-with-local-password"
