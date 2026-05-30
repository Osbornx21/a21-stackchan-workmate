#pragma once

#include "a21_firmware_mic.h"

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
  capabilities["imu"] = "planned_9_axis_imu";
  capabilities["ambient_light"] = "planned_ambient_light_sensor";
  capabilities["proximity"] = "planned_proximity_sensor";
  capabilities["battery"] = "planned_550mah_battery";
  capabilities["nfc"] = "planned_nfc";
  capabilities["infrared"] = "planned_infrared_tx_rx";
}
