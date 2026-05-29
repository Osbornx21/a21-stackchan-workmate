#include <unity.h>

#include "a21_firmware_connection.h"
#include "a21_firmware_network.h"
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

void test_network_config_defaults_to_a21_gateway() {
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);

  TEST_ASSERT_EQUAL_STRING("10.21.0.1", config.gateway_host);
  TEST_ASSERT_EQUAL_UINT16(21080, config.gateway_port);
  TEST_ASSERT_EQUAL_STRING("/ws/control", config.control_path);
  TEST_ASSERT_EQUAL_STRING("/ws/audio", config.audio_path);
  TEST_ASSERT_TRUE(a21ValidateNetworkConfig(&config));
}

void test_network_config_builds_control_and_audio_urls() {
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);
  a21CopyString(config.gateway_host, A21_GATEWAY_HOST_CAP, "192.168.31.50");

  char control_url[A21_WS_URL_CAP];
  char audio_url[A21_WS_URL_CAP];
  TEST_ASSERT_TRUE(a21BuildControlWSURL(&config, control_url, sizeof(control_url)));
  TEST_ASSERT_TRUE(a21BuildAudioWSURL(&config, audio_url, sizeof(audio_url)));
  TEST_ASSERT_EQUAL_STRING("ws://192.168.31.50:21080/ws/control", control_url);
  TEST_ASSERT_EQUAL_STRING("ws://192.168.31.50:21080/ws/audio", audio_url);
}

void test_network_config_rejects_legacy_ports_and_names() {
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);
  config.gateway_port = 8080;
  TEST_ASSERT_FALSE(a21ValidateNetworkConfig(&config));

  a21InitNetworkConfig(&config);
  a21CopyString(config.gateway_host, A21_GATEWAY_HOST_CAP, "x21.local");
  TEST_ASSERT_FALSE(a21ValidateNetworkConfig(&config));
}

void test_connection_state_machine_reaches_gateway_connected() {
  A21ConnectionState connection;
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);
  a21InitConnectionState(&connection, &config, 1000);

  TEST_ASSERT_EQUAL(A21_CONN_WIFI_CONNECTING, connection.phase);
  TEST_ASSERT_EQUAL_STRING("Wi-Fi connecting", connection.status_text);

  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1200);
  TEST_ASSERT_EQUAL(A21_CONN_GATEWAY_CONNECTING, connection.phase);
  TEST_ASSERT_EQUAL_STRING("192.168.31.21", connection.local_ip);
  TEST_ASSERT_EQUAL_STRING("Gateway connecting", connection.status_text);

  a21ConnectionOnGatewayConnected(&connection, 1300);
  TEST_ASSERT_EQUAL(A21_CONN_GATEWAY_CONNECTED, connection.phase);
  TEST_ASSERT_EQUAL_STRING("Gateway connected", connection.status_text);
  TEST_ASSERT_EQUAL_UINT8(0, connection.reconnect_attempt);
}

void test_connection_state_machine_enters_reconnect_after_gateway_loss() {
  A21ConnectionState connection;
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);
  a21InitConnectionState(&connection, &config, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1200);
  a21ConnectionOnGatewayConnected(&connection, 1300);

  a21ConnectionOnGatewayDisconnected(&connection, "ws_closed", 1500);
  TEST_ASSERT_EQUAL(A21_CONN_RECONNECT_WAIT, connection.phase);
  TEST_ASSERT_EQUAL_STRING("ws_closed", connection.last_error);
  TEST_ASSERT_EQUAL_UINT8(1, connection.reconnect_attempt);
  TEST_ASSERT_TRUE(connection.next_retry_at_ms > 1500);

  TEST_ASSERT_FALSE(a21ConnectionRetryDue(&connection, connection.next_retry_at_ms - 1));
  TEST_ASSERT_TRUE(a21ConnectionRetryDue(&connection, connection.next_retry_at_ms));
}

void test_connection_state_machine_rejects_invalid_config() {
  A21ConnectionState connection;
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);
  config.gateway_port = 8080;
  a21InitConnectionState(&connection, &config, 1000);

  TEST_ASSERT_EQUAL(A21_CONN_LOCAL_FALLBACK, connection.phase);
  TEST_ASSERT_EQUAL_STRING("invalid_config", connection.last_error);
  TEST_ASSERT_EQUAL_STRING("Local fallback", connection.status_text);
}

int main(int argc, char** argv) {
  UNITY_BEGIN();
  RUN_TEST(test_parse_control_event_listening);
  RUN_TEST(test_parse_control_event_rejects_wrong_protocol);
  RUN_TEST(test_parse_control_event_rejects_wrong_device);
  RUN_TEST(test_apply_control_event_updates_runtime_state);
  RUN_TEST(test_apply_invalid_control_event_enters_error_state);
  RUN_TEST(test_network_config_defaults_to_a21_gateway);
  RUN_TEST(test_network_config_builds_control_and_audio_urls);
  RUN_TEST(test_network_config_rejects_legacy_ports_and_names);
  RUN_TEST(test_connection_state_machine_reaches_gateway_connected);
  RUN_TEST(test_connection_state_machine_enters_reconnect_after_gateway_loss);
  RUN_TEST(test_connection_state_machine_rejects_invalid_config);
  return UNITY_END();
}
