#include <unity.h>

#include "a21_firmware_protocol.h"

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

int main(int argc, char** argv) {
  UNITY_BEGIN();
  RUN_TEST(test_parse_control_event_listening);
  RUN_TEST(test_parse_control_event_rejects_wrong_protocol);
  RUN_TEST(test_parse_control_event_rejects_wrong_device);
  return UNITY_END();
}
