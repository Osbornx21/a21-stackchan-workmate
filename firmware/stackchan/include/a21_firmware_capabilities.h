#pragma once

#include "a21_firmware_imu.h"
#include "a21_firmware_mic.h"
#include "a21_firmware_sensors.h"

#include <ArduinoJson.h>

inline void a21WriteStackChanHardwareCapabilities(JsonObject capabilities) {
  capabilities["microphone"] = a21MicrophoneCapabilityStatus();
  capabilities["speaker"] = "available";
  capabilities["screen"] = "available";
  capabilities["screen_touch"] = "available";
  capabilities["top_touch"] = "available";
  capabilities["servo_y"] = "available";
  capabilities["servo_x"] = "planned_continuous_rotation_axis";
  capabilities["rgb"] = "available";
  capabilities["camera"] = "planned_core_s3_camera";
  capabilities["imu"] = a21IMUCapabilityStatus();
  capabilities["ambient_light"] = a21AmbientLightCapabilityStatus();
  capabilities["proximity"] = a21ProximityCapabilityStatus();
  capabilities["battery"] = a21BatteryCapabilityStatus();
  capabilities["nfc"] = "planned_nfc";
  capabilities["infrared"] = "planned_infrared_tx_rx";
}
