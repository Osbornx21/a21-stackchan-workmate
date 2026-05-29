#include <unity.h>

#include "a21_firmware_protocol.h"
#include "a21_firmware_state.h"

void test_parse_control_event_listening() {
  const char* json =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"seq\":1,"
      "\"trace_id\":\"a21-trace-000001\","
      "\"session_id\":\"a21-session-000001\","
      "\"payload\":{\"state\":\"listening\",\"mode\":\"workmate\",\"text\":\"我在听\",\"final\":false}}";

  A21ControlEvent event;
  TEST_ASSERT_TRUE(a21ParseControlEvent(json, "stackchan-001", &event));
  TEST_ASSERT_EQUAL_STRING("a21-trace-000001", event.trace_id);
  TEST_ASSERT_EQUAL_STRING("a21-session-000001", event.session_id);
  TEST_ASSERT_EQUAL_STRING("listening", event.state);
  TEST_ASSERT_EQUAL_STRING("workmate", event.mode);
  TEST_ASSERT_EQUAL_STRING("我在听", event.text);
  TEST_ASSERT_FALSE(event.final);
}

void test_parse_control_event_rejects_wrong_protocol() {
  const char* json =
      "{\"protocol\":\"x21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"payload\":{\"state\":\"listening\",\"mode\":\"workmate\"}}";

  A21ControlEvent event;
  TEST_ASSERT_FALSE(a21ParseControlEvent(json, "stackchan-001", &event));
  TEST_ASSERT_EQUAL_STRING("protocol", event.error);
}

void test_parse_control_event_rejects_wrong_device() {
  const char* json =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"other-device\","
      "\"kind\":\"control.event\","
      "\"payload\":{\"state\":\"listening\",\"mode\":\"workmate\"}}";

  A21ControlEvent event;
  TEST_ASSERT_FALSE(a21ParseControlEvent(json, "stackchan-001", &event));
  TEST_ASSERT_EQUAL_STRING("device_id", event.error);
}

void test_apply_control_event_updates_runtime_state() {
  A21FirmwareState state;
  a21InitFirmwareState(&state, "stackchan-001");

  A21ControlEvent event;
  TEST_ASSERT_TRUE(a21ParseControlEvent(
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"trace_id\":\"a21-trace-000002\","
      "\"session_id\":\"a21-session-000002\","
      "\"payload\":{\"state\":\"speaking\",\"mode\":\"workmate\",\"text\":\"我在\",\"stream_id\":\"stream-1\",\"final\":true}}",
      "stackchan-001",
      &event));

  a21ApplyControlEvent(&state, &event, 1234);
  TEST_ASSERT_EQUAL(A21_RENDER_SPEAKING, state.render_state);
  TEST_ASSERT_EQUAL_STRING("workmate", state.mode);
  TEST_ASSERT_EQUAL_STRING("我在", state.text);
  TEST_ASSERT_EQUAL_STRING("stream-1", state.stream_id);
  TEST_ASSERT_EQUAL_STRING("a21-trace-000002", state.trace_id);
  TEST_ASSERT_TRUE(state.final);
  TEST_ASSERT_EQUAL_UINT32(1234, state.updated_at_ms);
}

void test_apply_invalid_control_event_enters_error_state() {
  A21FirmwareState state;
  a21InitFirmwareState(&state, "stackchan-001");

  A21ControlEvent event;
  a21ResetControlEvent(&event);
  a21CopyString(event.state, A21_STATE_CAP, "unknown-state");
  a21CopyString(event.mode, A21_MODE_CAP, "workmate");
  a21CopyString(event.text, A21_TEXT_CAP, "bad");

  a21ApplyControlEvent(&state, &event, 2000);
  TEST_ASSERT_EQUAL(A21_RENDER_ERROR, state.render_state);
  TEST_ASSERT_EQUAL_STRING("unsupported_state", state.last_error);
  TEST_ASSERT_EQUAL_STRING("bad", state.text);
}

int main(int argc, char** argv) {
  UNITY_BEGIN();
  RUN_TEST(test_parse_control_event_listening);
  RUN_TEST(test_parse_control_event_rejects_wrong_protocol);
  RUN_TEST(test_parse_control_event_rejects_wrong_device);
  RUN_TEST(test_apply_control_event_updates_runtime_state);
  RUN_TEST(test_apply_invalid_control_event_enters_error_state);
  return UNITY_END();
}
