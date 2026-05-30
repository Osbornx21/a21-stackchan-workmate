#pragma once

#include <stddef.h>
#include <stdint.h>

#ifndef A21_ENABLE_SENSOR_DIAGNOSTIC_PROBE
#define A21_ENABLE_SENSOR_DIAGNOSTIC_PROBE 0
#endif

#ifndef A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE
#define A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE 0
#endif

#ifndef A21_ENABLE_STACKCHAN_BATTERY_DIAGNOSTIC_PROBE
#define A21_ENABLE_STACKCHAN_BATTERY_DIAGNOSTIC_PROBE 0
#endif

static constexpr uint32_t A21_SENSOR_DIAGNOSTIC_INTERVAL_MS = 250;

struct A21SensorSample {
  bool has_ambient_light;
  uint16_t ambient_light_raw;
  bool has_proximity;
  uint16_t proximity_raw;
  bool has_battery;
  int16_t battery_mv;
  int16_t battery_ma;
};

struct A21SensorDriver {
  void* ctx;
  bool (*read)(void* ctx, A21SensorSample* sample);
};

struct A21SensorDiagnosticRuntime {
  bool enabled;
  bool available;
  uint32_t samples;
  uint32_t read_errors;
  uint32_t last_sample_at_ms;
  bool has_ambient_light;
  uint16_t ambient_light_raw;
  bool has_proximity;
  uint16_t proximity_raw;
  bool has_battery;
  int16_t battery_mv;
  int16_t battery_ma;
};

inline bool a21SensorDiagnosticEnabled() {
  return A21_ENABLE_SENSOR_DIAGNOSTIC_PROBE == 1 ||
         A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE == 1 ||
         A21_ENABLE_STACKCHAN_BATTERY_DIAGNOSTIC_PROBE == 1;
}

inline const char* a21AmbientLightCapabilityStatus() {
#if defined(A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE) && A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE
  return "diagnostic_probe_ltr553_ambient_light";
#else
  return "planned_ambient_light_sensor";
#endif
}

inline const char* a21ProximityCapabilityStatus() {
#if defined(A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE) && A21_ENABLE_CORES3_LTR553_DIAGNOSTIC_PROBE
  return "diagnostic_probe_ltr553_proximity";
#else
  return "planned_proximity_sensor";
#endif
}

inline const char* a21BatteryCapabilityStatus() {
#if defined(A21_ENABLE_STACKCHAN_BATTERY_DIAGNOSTIC_PROBE) && A21_ENABLE_STACKCHAN_BATTERY_DIAGNOSTIC_PROBE
  return "diagnostic_probe_ina226_battery";
#else
  return "planned_550mah_battery";
#endif
}

inline void a21InitSensorDiagnosticRuntime(A21SensorDiagnosticRuntime* runtime) {
  if (runtime == nullptr) {
    return;
  }
  runtime->enabled = a21SensorDiagnosticEnabled();
  runtime->available = false;
  runtime->samples = 0;
  runtime->read_errors = 0;
  runtime->last_sample_at_ms = 0;
  runtime->has_ambient_light = false;
  runtime->ambient_light_raw = 0;
  runtime->has_proximity = false;
  runtime->proximity_raw = 0;
  runtime->has_battery = false;
  runtime->battery_mv = 0;
  runtime->battery_ma = 0;
}

inline int16_t a21SensorFloatToInt16(float value, float scale) {
  const float scaled = value * scale;
  if (scaled > 32767.0f) {
    return 32767;
  }
  if (scaled < -32768.0f) {
    return -32768;
  }
  return static_cast<int16_t>(scaled >= 0.0f ? scaled + 0.5f : scaled - 0.5f);
}

inline bool a21SensorDriverReady(const A21SensorDriver* driver) {
  return driver != nullptr && driver->read != nullptr;
}

inline bool a21SensorDiagnosticTick(A21SensorDiagnosticRuntime* runtime, const A21SensorDriver* driver, uint32_t now_ms) {
  if (runtime == nullptr) {
    return false;
  }
  if (!runtime->enabled) {
    return true;
  }
  if (runtime->last_sample_at_ms != 0 && now_ms - runtime->last_sample_at_ms < A21_SENSOR_DIAGNOSTIC_INTERVAL_MS) {
    return true;
  }
  runtime->last_sample_at_ms = now_ms;
  if (!a21SensorDriverReady(driver)) {
    runtime->available = false;
    runtime->read_errors += 1;
    return false;
  }

  A21SensorSample sample = {};
  if (!driver->read(driver->ctx, &sample)) {
    runtime->available = false;
    runtime->read_errors += 1;
    return false;
  }

  runtime->available = sample.has_ambient_light || sample.has_proximity || sample.has_battery;
  runtime->samples += 1;
  runtime->has_ambient_light = sample.has_ambient_light;
  runtime->ambient_light_raw = sample.ambient_light_raw;
  runtime->has_proximity = sample.has_proximity;
  runtime->proximity_raw = sample.proximity_raw;
  runtime->has_battery = sample.has_battery;
  runtime->battery_mv = sample.battery_mv;
  runtime->battery_ma = sample.battery_ma;
  return runtime->available;
}
