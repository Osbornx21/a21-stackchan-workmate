#pragma once

#ifndef A21_FIRMWARE_ID
#define A21_FIRMWARE_ID "a21-stackchan"
#endif

#ifndef A21_FIRMWARE_VERSION
#define A21_FIRMWARE_VERSION "0.1.0"
#endif

#ifndef A21_DEVICE_ID
#define A21_DEVICE_ID "stackchan-001"
#endif

static constexpr int A21_SERVO_Y_MIN_DEG = 5;
static constexpr int A21_SERVO_Y_MAX_DEG = 85;

inline int a21ClampServoY(int degrees) {
  if (degrees < A21_SERVO_Y_MIN_DEG) {
    return A21_SERVO_Y_MIN_DEG;
  }
  if (degrees > A21_SERVO_Y_MAX_DEG) {
    return A21_SERVO_Y_MAX_DEG;
  }
  return degrees;
}
