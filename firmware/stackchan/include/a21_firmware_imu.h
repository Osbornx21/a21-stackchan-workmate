#pragma once

#include "a21_firmware_config.h"

#include <stddef.h>
#include <stdint.h>

static constexpr uint32_t A21_IMU_DIAGNOSTIC_INTERVAL_MS = 100;
static constexpr size_t A21_IMU_POSTURE_CAP = 16;

struct A21IMUSample {
  bool has_accel;
  float accel_x_g;
  float accel_y_g;
  float accel_z_g;
  bool has_gyro;
  float gyro_x_dps;
  float gyro_y_dps;
  float gyro_z_dps;
};

struct A21IMUDriver {
  void* ctx;
  bool (*read)(void* ctx, A21IMUSample* sample);
};

struct A21IMUDiagnosticRuntime {
  bool enabled;
  bool available;
  uint32_t samples;
  uint32_t read_errors;
  uint32_t last_sample_at_ms;
  int16_t accel_mg_x;
  int16_t accel_mg_y;
  int16_t accel_mg_z;
  int16_t gyro_mdps_x;
  int16_t gyro_mdps_y;
  int16_t gyro_mdps_z;
  char posture[A21_IMU_POSTURE_CAP];
};

inline const char* a21IMUCapabilityStatus() {
#if defined(A21_ENABLE_IMU_DIAGNOSTIC_PROBE) && A21_ENABLE_IMU_DIAGNOSTIC_PROBE
  return "diagnostic_probe_m5unified_imu";
#else
  return "planned_9_axis_imu";
#endif
}

inline void a21InitIMUDiagnosticRuntime(A21IMUDiagnosticRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
#if defined(A21_ENABLE_IMU_DIAGNOSTIC_PROBE) && A21_ENABLE_IMU_DIAGNOSTIC_PROBE
  runtime->enabled = true;
#else
  runtime->enabled = false;
#endif
  runtime->available = false;
  runtime->samples = 0;
  runtime->read_errors = 0;
  runtime->last_sample_at_ms = 0;
  runtime->accel_mg_x = 0;
  runtime->accel_mg_y = 0;
  runtime->accel_mg_z = 0;
  runtime->gyro_mdps_x = 0;
  runtime->gyro_mdps_y = 0;
  runtime->gyro_mdps_z = 0;
  a21CopyString(runtime->posture, A21_IMU_POSTURE_CAP, "unknown");
}

inline int16_t a21ScaledFloatToInt16(float value, float scale) {
  const float scaled = value * scale;
  if (scaled > 32767.0f) {
    return 32767;
  }
  if (scaled < -32768.0f) {
    return -32768;
  }
  return static_cast<int16_t>(scaled >= 0.0f ? scaled + 0.5f : scaled - 0.5f);
}

inline const char* a21IMUPostureFromAccel(const A21IMUSample* sample) {
  if (sample == nullptr || !sample->has_accel) {
    return "unknown";
  }
  if (sample->accel_z_g >= 0.75f) {
    return "face_up";
  }
  if (sample->accel_z_g <= -0.75f) {
    return "face_down";
  }
  if (sample->accel_y_g >= 0.65f) {
    return "upright";
  }
  if (sample->accel_y_g <= -0.65f) {
    return "lean_back";
  }
  return "tilted";
}

inline bool a21IMUDriverReady(const A21IMUDriver* driver) {
  return driver != nullptr && driver->read != nullptr;
}

inline bool a21IMUDiagnosticTick(A21IMUDiagnosticRuntime* runtime, const A21IMUDriver* driver, uint32_t now_ms) {
  if (runtime == nullptr) {
    return false;
  }
  if (!runtime->enabled) {
    return true;
  }
  if (runtime->last_sample_at_ms != 0 && now_ms - runtime->last_sample_at_ms < A21_IMU_DIAGNOSTIC_INTERVAL_MS) {
    return true;
  }
  runtime->last_sample_at_ms = now_ms;
  if (!a21IMUDriverReady(driver)) {
    runtime->available = false;
    runtime->read_errors += 1;
    return false;
  }
  A21IMUSample sample = {};
  if (!driver->read(driver->ctx, &sample)) {
    runtime->available = false;
    runtime->read_errors += 1;
    return false;
  }
  runtime->available = true;
  runtime->samples += 1;
  if (sample.has_accel) {
    runtime->accel_mg_x = a21ScaledFloatToInt16(sample.accel_x_g, 1000.0f);
    runtime->accel_mg_y = a21ScaledFloatToInt16(sample.accel_y_g, 1000.0f);
    runtime->accel_mg_z = a21ScaledFloatToInt16(sample.accel_z_g, 1000.0f);
  }
  if (sample.has_gyro) {
    runtime->gyro_mdps_x = a21ScaledFloatToInt16(sample.gyro_x_dps, 1000.0f);
    runtime->gyro_mdps_y = a21ScaledFloatToInt16(sample.gyro_y_dps, 1000.0f);
    runtime->gyro_mdps_z = a21ScaledFloatToInt16(sample.gyro_z_dps, 1000.0f);
  }
  a21CopyString(runtime->posture, A21_IMU_POSTURE_CAP, a21IMUPostureFromAccel(&sample));
  return true;
}
