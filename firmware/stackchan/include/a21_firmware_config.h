#pragma once

#include <stddef.h>
#include <stdio.h>
#include <string.h>

#ifndef A21_FIRMWARE_ID
#define A21_FIRMWARE_ID "a21-stackchan"
#endif

#ifndef A21_FIRMWARE_VERSION
#define A21_FIRMWARE_VERSION "0.1.0"
#endif

#ifndef A21_FIRMWARE_BOARD
#define A21_FIRMWARE_BOARD "m5stack-cores3"
#endif

#if __has_include("a21_firmware_build.generated.h")
#include "a21_firmware_build.generated.h"
#endif

#ifndef A21_FIRMWARE_COMMIT
#define A21_FIRMWARE_COMMIT "dev"
#endif

#ifndef A21_DEVICE_ID
#define A21_DEVICE_ID "stackchan-001"
#endif

static constexpr size_t A21_FIRMWARE_ID_CAP = 32;
static constexpr size_t A21_FIRMWARE_VERSION_CAP = 24;
static constexpr size_t A21_FIRMWARE_BOARD_CAP = 32;
static constexpr size_t A21_FIRMWARE_COMMIT_CAP = 48;
static constexpr size_t A21_FIRMWARE_LABEL_CAP = 96;

struct A21FirmwareBuildIdentity {
  char firmware_id[A21_FIRMWARE_ID_CAP];
  char version[A21_FIRMWARE_VERSION_CAP];
  char board[A21_FIRMWARE_BOARD_CAP];
  char commit[A21_FIRMWARE_COMMIT_CAP];
};

static constexpr int A21_SERVO_Y_MIN_DEG = 5;
static constexpr int A21_SERVO_Y_MAX_DEG = 85;

inline void a21ConfigCopyString(char* target, size_t target_size, const char* source) {
  if (target == nullptr || target_size == 0) {
    return;
  }
  if (source == nullptr) {
    source = "";
  }
  strncpy(target, source, target_size - 1);
  target[target_size - 1] = '\0';
}

inline bool a21ConfigStringContainsForbiddenIdentity(const char* value) {
  return value != nullptr &&
         (strstr(value, "x21") != nullptr ||
          strstr(value, "X21") != nullptr ||
          strstr(value, "v21") != nullptr ||
          strstr(value, "V21") != nullptr);
}

inline void a21GetFirmwareBuildIdentity(A21FirmwareBuildIdentity* identity) {
  if (identity == nullptr) {
    return;
  }
  a21ConfigCopyString(identity->firmware_id, A21_FIRMWARE_ID_CAP, A21_FIRMWARE_ID);
  a21ConfigCopyString(identity->version, A21_FIRMWARE_VERSION_CAP, A21_FIRMWARE_VERSION);
  a21ConfigCopyString(identity->board, A21_FIRMWARE_BOARD_CAP, A21_FIRMWARE_BOARD);
  a21ConfigCopyString(identity->commit, A21_FIRMWARE_COMMIT_CAP, A21_FIRMWARE_COMMIT);
}

inline bool a21FirmwareBuildIdentityValid(const A21FirmwareBuildIdentity* identity) {
  if (identity == nullptr) {
    return false;
  }
  if (identity->firmware_id[0] == '\0' || identity->version[0] == '\0' || identity->board[0] == '\0' || identity->commit[0] == '\0') {
    return false;
  }
  return !a21ConfigStringContainsForbiddenIdentity(identity->firmware_id) &&
         !a21ConfigStringContainsForbiddenIdentity(identity->version) &&
         !a21ConfigStringContainsForbiddenIdentity(identity->board) &&
         !a21ConfigStringContainsForbiddenIdentity(identity->commit);
}

inline bool a21BuildFirmwareLabel(char* output, size_t output_size) {
  if (output == nullptr || output_size == 0) {
    return false;
  }
  const int written = snprintf(output, output_size, "%s %s", A21_FIRMWARE_VERSION, A21_FIRMWARE_COMMIT);
  return written > 0 && static_cast<size_t>(written) < output_size;
}

inline int a21ClampServoY(int degrees) {
  if (degrees < A21_SERVO_Y_MIN_DEG) {
    return A21_SERVO_Y_MIN_DEG;
  }
  if (degrees > A21_SERVO_Y_MAX_DEG) {
    return A21_SERVO_Y_MAX_DEG;
  }
  return degrees;
}
