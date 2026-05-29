#include <unity.h>

#include "a21_firmware_connection.h"
#include "a21_firmware_audio_ws.h"
#include "a21_firmware_config.h"
#include "a21_firmware_network.h"
#include "a21_firmware_protocol.h"
#include "a21_firmware_state.h"
#include "a21_firmware_gateway_ws.h"
#include "a21_firmware_wifi.h"
#include "a21_firmware_wifi_runtime.h"

struct FakeWiFiDriver {
  int begin_count;
  const char* last_ssid;
  const char* last_password;
  A21WiFiDriverStatus status;
  const char* local_ip;
};

bool fakeWiFiBegin(void* ctx, const char* ssid, const char* password) {
  FakeWiFiDriver* driver = static_cast<FakeWiFiDriver*>(ctx);
  driver->begin_count += 1;
  driver->last_ssid = ssid;
  driver->last_password = password;
  return true;
}

A21WiFiDriverStatus fakeWiFiStatus(void* ctx) {
  FakeWiFiDriver* driver = static_cast<FakeWiFiDriver*>(ctx);
  return driver->status;
}

bool fakeWiFiLocalIP(void* ctx, char* output, size_t output_size) {
  FakeWiFiDriver* driver = static_cast<FakeWiFiDriver*>(ctx);
  a21CopyString(output, output_size, driver->local_ip);
  return true;
}

void initFakeWiFiDriver(FakeWiFiDriver* fake, A21WiFiDriver* driver) {
  fake->begin_count = 0;
  fake->last_ssid = "";
  fake->last_password = "";
  fake->status = A21_WIFI_DRIVER_DISCONNECTED;
  fake->local_ip = "192.168.31.21";
  driver->ctx = fake;
  driver->begin = fakeWiFiBegin;
  driver->status = fakeWiFiStatus;
  driver->local_ip = fakeWiFiLocalIP;
}

struct FakeGatewayWSDriver {
  int begin_count;
  int loop_count;
  int send_count;
  const char* last_host;
  uint16_t last_port;
  const char* last_path;
  char last_sent_text[A21_WS_TEXT_MESSAGE_CAP];
  bool connected;
  const char* pending_text;
};

bool fakeGatewayWSBegin(void* ctx, const char* host, uint16_t port, const char* path) {
  FakeGatewayWSDriver* driver = static_cast<FakeGatewayWSDriver*>(ctx);
  driver->begin_count += 1;
  driver->last_host = host;
  driver->last_port = port;
  driver->last_path = path;
  return true;
}

void fakeGatewayWSLoop(void* ctx) {
  FakeGatewayWSDriver* driver = static_cast<FakeGatewayWSDriver*>(ctx);
  driver->loop_count += 1;
}

bool fakeGatewayWSConnected(void* ctx) {
  FakeGatewayWSDriver* driver = static_cast<FakeGatewayWSDriver*>(ctx);
  return driver->connected;
}

bool fakeGatewayWSReadText(void* ctx, char* output, size_t output_size) {
  FakeGatewayWSDriver* driver = static_cast<FakeGatewayWSDriver*>(ctx);
  if (driver->pending_text == nullptr || driver->pending_text[0] == '\0') {
    return false;
  }
  a21CopyString(output, output_size, driver->pending_text);
  driver->pending_text = "";
  return true;
}

bool fakeGatewayWSSendText(void* ctx, const char* text) {
  FakeGatewayWSDriver* driver = static_cast<FakeGatewayWSDriver*>(ctx);
  driver->send_count += 1;
  a21CopyString(driver->last_sent_text, sizeof(driver->last_sent_text), text);
  return true;
}

void initFakeGatewayWSDriver(FakeGatewayWSDriver* fake, A21GatewayWSDriver* driver) {
  fake->begin_count = 0;
  fake->loop_count = 0;
  fake->send_count = 0;
  fake->last_host = "";
  fake->last_port = 0;
  fake->last_path = "";
  fake->last_sent_text[0] = '\0';
  fake->connected = false;
  fake->pending_text = "";
  driver->ctx = fake;
  driver->begin = fakeGatewayWSBegin;
  driver->loop = fakeGatewayWSLoop;
  driver->connected = fakeGatewayWSConnected;
  driver->read_text = fakeGatewayWSReadText;
  driver->send_text = fakeGatewayWSSendText;
}

void initFakeAudioWSDriver(FakeGatewayWSDriver* fake, A21AudioWSDriver* driver) {
  fake->begin_count = 0;
  fake->loop_count = 0;
  fake->send_count = 0;
  fake->last_host = "";
  fake->last_port = 0;
  fake->last_path = "";
  fake->last_sent_text[0] = '\0';
  fake->connected = false;
  fake->pending_text = "";
  driver->ctx = fake;
  driver->begin = fakeGatewayWSBegin;
  driver->loop = fakeGatewayWSLoop;
  driver->connected = fakeGatewayWSConnected;
  driver->read_text = fakeGatewayWSReadText;
  driver->send_text = fakeGatewayWSSendText;
}

void test_firmware_build_identity_contains_a21_release_fields() {
  A21FirmwareBuildIdentity identity;
  a21GetFirmwareBuildIdentity(&identity);

  TEST_ASSERT_EQUAL_STRING("a21-stackchan", identity.firmware_id);
  TEST_ASSERT_EQUAL_STRING("0.1.0", identity.version);
  TEST_ASSERT_EQUAL_STRING("m5stack-cores3", identity.board);
  TEST_ASSERT_NOT_EQUAL('\0', identity.commit[0]);
  TEST_ASSERT_TRUE(a21FirmwareBuildIdentityValid(&identity));

  char label[96];
  TEST_ASSERT_TRUE(a21BuildFirmwareLabel(label, sizeof(label)));
  TEST_ASSERT_NOT_NULL(strstr(label, "0.1.0"));
  TEST_ASSERT_NOT_NULL(strstr(label, identity.commit));
}

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

void test_wifi_config_defaults_to_missing_credentials() {
  A21WiFiConfig wifi;
  a21InitWiFiConfig(&wifi);

  TEST_ASSERT_EQUAL_STRING("", wifi.ssid);
  TEST_ASSERT_FALSE(a21ValidateWiFiConfig(&wifi));

  char status[96];
  TEST_ASSERT_TRUE(a21DescribeWiFiConfig(&wifi, status, sizeof(status)));
  TEST_ASSERT_NOT_NULL(strstr(status, "ssid=missing"));
  TEST_ASSERT_NOT_NULL(strstr(status, "password=missing"));
}

void test_wifi_config_redacts_password_in_status_text() {
  A21WiFiConfig wifi;
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "wang301");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");

  TEST_ASSERT_TRUE(a21ValidateWiFiConfig(&wifi));

  char status[96];
  TEST_ASSERT_TRUE(a21DescribeWiFiConfig(&wifi, status, sizeof(status)));
  TEST_ASSERT_NOT_NULL(strstr(status, "ssid=wang301"));
  TEST_ASSERT_NOT_NULL(strstr(status, "password=configured"));
  TEST_ASSERT_NULL(strstr(status, "secret-password"));
}

void test_wifi_config_rejects_legacy_project_ssid() {
  A21WiFiConfig wifi;
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "x21-lab");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");

  TEST_ASSERT_FALSE(a21ValidateWiFiConfig(&wifi));
}

void test_connection_state_machine_uses_local_fallback_without_wifi_credentials() {
  A21ConnectionState connection;
  A21NetworkConfig config;
  A21WiFiConfig wifi;
  a21InitNetworkConfig(&config);
  a21InitWiFiConfig(&wifi);

  a21InitConnectionStateWithWiFi(&connection, &config, &wifi, 1000);

  TEST_ASSERT_EQUAL(A21_CONN_LOCAL_FALLBACK, connection.phase);
  TEST_ASSERT_EQUAL_STRING("missing_wifi", connection.last_error);
  TEST_ASSERT_EQUAL_STRING("Local fallback", connection.status_text);
}

void test_connection_state_machine_starts_wifi_when_credentials_are_valid() {
  A21ConnectionState connection;
  A21NetworkConfig config;
  A21WiFiConfig wifi;
  a21InitNetworkConfig(&config);
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "wang301");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");

  a21InitConnectionStateWithWiFi(&connection, &config, &wifi, 1000);

  TEST_ASSERT_EQUAL(A21_CONN_WIFI_CONNECTING, connection.phase);
  TEST_ASSERT_EQUAL_STRING("", connection.last_error);
  TEST_ASSERT_EQUAL_STRING("Wi-Fi connecting", connection.status_text);
}

void test_wifi_runtime_does_not_begin_without_credentials() {
  A21WiFiRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21WiFiConfig wifi;
  FakeWiFiDriver fake;
  A21WiFiDriver driver;
  initFakeWiFiDriver(&fake, &driver);
  a21InitNetworkConfig(&network);
  a21InitWiFiConfig(&wifi);
  a21InitConnectionStateWithWiFi(&connection, &network, &wifi, 1000);
  a21InitWiFiRuntime(&runtime);

  a21WiFiRuntimeTick(&runtime, &driver, &connection, &wifi, 1200);

  TEST_ASSERT_EQUAL(0, fake.begin_count);
  TEST_ASSERT_EQUAL(A21_CONN_LOCAL_FALLBACK, connection.phase);
}

void test_wifi_runtime_begins_once_and_redacts_driver_state() {
  A21WiFiRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21WiFiConfig wifi;
  FakeWiFiDriver fake;
  A21WiFiDriver driver;
  initFakeWiFiDriver(&fake, &driver);
  a21InitNetworkConfig(&network);
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "wang301");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");
  a21InitConnectionStateWithWiFi(&connection, &network, &wifi, 1000);
  a21InitWiFiRuntime(&runtime);

  a21WiFiRuntimeTick(&runtime, &driver, &connection, &wifi, 1200);
  a21WiFiRuntimeTick(&runtime, &driver, &connection, &wifi, 1300);

  TEST_ASSERT_EQUAL(1, fake.begin_count);
  TEST_ASSERT_EQUAL_STRING("wang301", fake.last_ssid);
  TEST_ASSERT_EQUAL_STRING("secret-password", fake.last_password);
  TEST_ASSERT_TRUE(runtime.begin_sent);
  TEST_ASSERT_EQUAL_UINT32(1200, runtime.last_begin_at_ms);
  TEST_ASSERT_EQUAL(A21_CONN_WIFI_CONNECTING, connection.phase);
}

void test_wifi_runtime_moves_to_gateway_connecting_on_connected_status() {
  A21WiFiRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21WiFiConfig wifi;
  FakeWiFiDriver fake;
  A21WiFiDriver driver;
  initFakeWiFiDriver(&fake, &driver);
  fake.status = A21_WIFI_DRIVER_CONNECTED;
  a21InitNetworkConfig(&network);
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "wang301");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");
  a21InitConnectionStateWithWiFi(&connection, &network, &wifi, 1000);
  a21InitWiFiRuntime(&runtime);

  a21WiFiRuntimeTick(&runtime, &driver, &connection, &wifi, 1400);

  TEST_ASSERT_EQUAL(A21_CONN_GATEWAY_CONNECTING, connection.phase);
  TEST_ASSERT_EQUAL_STRING("192.168.31.21", connection.local_ip);
}

void test_wifi_runtime_retries_after_disconnect() {
  A21WiFiRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21WiFiConfig wifi;
  FakeWiFiDriver fake;
  A21WiFiDriver driver;
  initFakeWiFiDriver(&fake, &driver);
  a21InitNetworkConfig(&network);
  a21InitWiFiConfig(&wifi);
  a21CopyString(wifi.ssid, A21_WIFI_SSID_CAP, "wang301");
  a21CopyString(wifi.password, A21_WIFI_PASSWORD_CAP, "secret-password");
  a21InitConnectionStateWithWiFi(&connection, &network, &wifi, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1200);
  a21InitWiFiRuntime(&runtime);

  a21WiFiRuntimeTick(&runtime, &driver, &connection, &wifi, 1500);

  TEST_ASSERT_EQUAL(A21_CONN_RECONNECT_WAIT, connection.phase);
  TEST_ASSERT_EQUAL_STRING("wifi_disconnected", connection.last_error);
  TEST_ASSERT_FALSE(runtime.begin_sent);
}

void test_gateway_ws_runtime_does_not_connect_before_wifi_ready() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21InitGatewayWSRuntime(&runtime);

  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1200);

  TEST_ASSERT_EQUAL(0, fake.begin_count);
  TEST_ASSERT_EQUAL(A21_CONN_WIFI_CONNECTING, connection.phase);
}

void test_gateway_ws_runtime_begins_control_socket_once() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1100);
  a21InitGatewayWSRuntime(&runtime);

  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1200);
  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1300);

  TEST_ASSERT_EQUAL(1, fake.begin_count);
  TEST_ASSERT_EQUAL_STRING("10.21.0.1", fake.last_host);
  TEST_ASSERT_EQUAL_UINT16(21080, fake.last_port);
  TEST_ASSERT_EQUAL_STRING("/ws/control", fake.last_path);
  TEST_ASSERT_TRUE(runtime.begin_sent);
  TEST_ASSERT_EQUAL(A21_CONN_GATEWAY_CONNECTING, connection.phase);
}

void test_gateway_ws_runtime_marks_gateway_connected() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = true;
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1100);
  a21InitGatewayWSRuntime(&runtime);

  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1200);

  TEST_ASSERT_EQUAL(A21_CONN_GATEWAY_CONNECTED, connection.phase);
  TEST_ASSERT_EQUAL_STRING("", connection.last_error);
}

void test_gateway_ws_runtime_applies_control_event_text() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = true;
  fake.pending_text =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"trace_id\":\"a21-trace-ws-001\","
      "\"session_id\":\"a21-session-ws-001\","
      "\"payload\":{\"state\":\"speaking\",\"mode\":\"workmate\",\"text\":\"WebSocket 已接上\",\"stream_id\":\"stream-ws\",\"final\":true}}";
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1100);
  a21ConnectionOnGatewayConnected(&connection, 1200);
  a21InitGatewayWSRuntime(&runtime);

  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1300);

  TEST_ASSERT_EQUAL(A21_RENDER_SPEAKING, state.render_state);
  TEST_ASSERT_EQUAL_STRING("WebSocket 已接上", state.text);
  TEST_ASSERT_EQUAL_STRING("a21-trace-ws-001", state.trace_id);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.received_control_events);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.invalid_control_events);
}

void test_gateway_ws_runtime_enters_reconnect_after_disconnect() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = false;
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1100);
  a21ConnectionOnGatewayConnected(&connection, 1200);
  a21InitGatewayWSRuntime(&runtime);
  runtime.begin_sent = true;

  a21GatewayWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1500);

  TEST_ASSERT_EQUAL(A21_CONN_RECONNECT_WAIT, connection.phase);
  TEST_ASSERT_EQUAL_STRING("ws_disconnected", connection.last_error);
  TEST_ASSERT_FALSE(runtime.begin_sent);
}

void test_gateway_ws_send_mock_turn_builds_a21_device_event() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = true;
  a21InitGatewayWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  TEST_ASSERT_TRUE(a21GatewayWSSendDeviceEvent(
      &runtime,
      &driver,
      &connection,
      &state,
      "mock.turn",
      "workmate",
      "先说，我在",
      1600));

  TEST_ASSERT_EQUAL(1, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.sent_device_events);
  TEST_ASSERT_EQUAL_UINT64(2, runtime.next_seq);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("a21.device.v1", doc["protocol"] | "");
  TEST_ASSERT_EQUAL_STRING("stackchan-001", doc["device_id"] | "");
  TEST_ASSERT_EQUAL_STRING("device.event", doc["kind"] | "");
  TEST_ASSERT_EQUAL_UINT64(1, doc["seq"] | 0);
  TEST_ASSERT_EQUAL_STRING("a21-trace-device-000001", doc["trace_id"] | "");
  TEST_ASSERT_EQUAL_STRING("a21-session-device", doc["session_id"] | "");
  TEST_ASSERT_EQUAL_STRING("mock.turn", doc["payload"]["event"] | "");
  TEST_ASSERT_EQUAL_STRING("workmate", doc["payload"]["mode"] | "");
  TEST_ASSERT_EQUAL_STRING("先说，我在", doc["payload"]["text"] | "");
  TEST_ASSERT_EQUAL_STRING("a21-stackchan", doc["payload"]["firmware_id"] | "");
  TEST_ASSERT_EQUAL_STRING("0.1.0", doc["payload"]["firmware_version"] | "");
  TEST_ASSERT_EQUAL_STRING("m5stack-cores3", doc["payload"]["firmware_board"] | "");
  TEST_ASSERT_NOT_EQUAL('\0', (doc["payload"]["firmware_commit"] | "")[0]);
}

void test_gateway_ws_send_interrupt_increments_seq() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = true;
  a21InitGatewayWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  TEST_ASSERT_TRUE(a21GatewayWSSendDeviceEvent(&runtime, &driver, &connection, &state, "mock.turn", "workmate", "", 1600));
  TEST_ASSERT_TRUE(a21GatewayWSSendDeviceEvent(&runtime, &driver, &connection, &state, "interrupt", "workmate", "", 1700));

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake.last_sent_text));
  TEST_ASSERT_EQUAL_UINT64(2, doc["seq"] | 0);
  TEST_ASSERT_EQUAL_STRING("a21-trace-device-000002", doc["trace_id"] | "");
  TEST_ASSERT_EQUAL_STRING("interrupt", doc["payload"]["event"] | "");
  TEST_ASSERT_EQUAL_UINT32(2, runtime.sent_device_events);
  TEST_ASSERT_EQUAL_UINT64(3, runtime.next_seq);
}

void test_gateway_ws_send_device_event_rejects_when_not_connected() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = false;
  a21InitGatewayWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTING, 1600);

  TEST_ASSERT_FALSE(a21GatewayWSSendDeviceEvent(&runtime, &driver, &connection, &state, "mock.turn", "workmate", "", 1600));
  TEST_ASSERT_EQUAL(0, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.sent_device_events);
  TEST_ASSERT_EQUAL_UINT64(1, runtime.next_seq);
}

void test_audio_ws_runtime_waits_for_gateway_connection() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  a21InitAudioWSRuntime(&runtime);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitConnectionState(&connection, &network, 1000);
  a21ConnectionOnWiFiConnected(&connection, "192.168.31.21", 1100);

  a21AudioWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1200);

  TEST_ASSERT_EQUAL(0, fake.begin_count);
  TEST_ASSERT_FALSE(runtime.begin_sent);
}

void test_audio_ws_runtime_begins_audio_socket_once() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  a21InitAudioWSRuntime(&runtime);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  a21AudioWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1700);
  a21AudioWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 1800);

  TEST_ASSERT_EQUAL(1, fake.begin_count);
  TEST_ASSERT_EQUAL_STRING("10.21.0.1", fake.last_host);
  TEST_ASSERT_EQUAL_UINT16(21080, fake.last_port);
  TEST_ASSERT_EQUAL_STRING("/ws/audio", fake.last_path);
  TEST_ASSERT_TRUE(runtime.begin_sent);
  TEST_ASSERT_FALSE(runtime.was_connected);
}

void test_audio_ws_send_mock_frame_builds_a21_audio_frame() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  fake.connected = true;
  a21InitAudioWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  TEST_ASSERT_TRUE(a21AudioWSSendMockFrame(&runtime, &driver, &connection, &state, 2000));

  TEST_ASSERT_EQUAL(1, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.sent_audio_frames);
  TEST_ASSERT_EQUAL_UINT64(2, runtime.next_seq);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("a21.device.v1", doc["protocol"] | "");
  TEST_ASSERT_EQUAL_STRING("stackchan-001", doc["device_id"] | "");
  TEST_ASSERT_EQUAL_STRING("audio.frame", doc["kind"] | "");
  TEST_ASSERT_EQUAL_UINT64(1, doc["seq"] | 0);
  TEST_ASSERT_EQUAL_STRING("a21-trace-audio-000001", doc["trace_id"] | "");
  TEST_ASSERT_EQUAL_STRING("a21-session-device", doc["session_id"] | "");
  TEST_ASSERT_EQUAL_INT64(2000, doc["sent_at_ms"] | 0);
  TEST_ASSERT_EQUAL_STRING("pcm_s16le", doc["payload"]["codec"] | "");
  TEST_ASSERT_EQUAL_INT(16000, doc["payload"]["sample_rate_hz"] | 0);
  TEST_ASSERT_EQUAL_INT(1, doc["payload"]["channels"] | 0);
  TEST_ASSERT_EQUAL_INT(20, doc["payload"]["duration_ms"] | 0);
  TEST_ASSERT_EQUAL_INT64(1980, doc["payload"]["capture_started_at_ms"] | 0);
  TEST_ASSERT_EQUAL_INT64(2000, doc["payload"]["capture_ended_at_ms"] | 0);
  TEST_ASSERT_EQUAL_STRING("AAAA", doc["payload"]["data_base64"] | "");
}

void test_audio_ws_applies_gateway_ack_control_event() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  fake.connected = true;
  fake.pending_text =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"trace_id\":\"a21-trace-audio-001\","
      "\"session_id\":\"a21-session-audio-001\","
      "\"payload\":{\"state\":\"listening\",\"mode\":\"workmate\",\"text\":\"audio frame accepted\",\"final\":true}}";
  a21InitAudioWSRuntime(&runtime);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  a21AudioWSRuntimeTick(&runtime, &driver, &connection, &network, &state, 2100);

  TEST_ASSERT_EQUAL(A21_RENDER_LISTENING, state.render_state);
  TEST_ASSERT_EQUAL_STRING("audio frame accepted", state.text);
  TEST_ASSERT_EQUAL_STRING("a21-trace-audio-001", state.trace_id);
  TEST_ASSERT_EQUAL_STRING("a21-session-audio-001", state.session_id);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.received_control_events);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.invalid_control_events);
  TEST_ASSERT_TRUE(runtime.was_connected);
}

void test_audio_ws_send_mock_frame_rejects_when_audio_not_connected() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  fake.connected = false;
  a21InitAudioWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  TEST_ASSERT_FALSE(a21AudioWSSendMockFrame(&runtime, &driver, &connection, &state, 2200));
  TEST_ASSERT_EQUAL(0, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.sent_audio_frames);
  TEST_ASSERT_EQUAL_UINT64(1, runtime.next_seq);
}

void test_servo_y_angle_clamps_to_stackchan_safe_range() {
  TEST_ASSERT_EQUAL_INT(5, A21_SERVO_Y_MIN_DEG);
  TEST_ASSERT_EQUAL_INT(85, A21_SERVO_Y_MAX_DEG);
  TEST_ASSERT_EQUAL_INT(5, a21ClampServoY(-30));
  TEST_ASSERT_EQUAL_INT(5, a21ClampServoY(0));
  TEST_ASSERT_EQUAL_INT(45, a21ClampServoY(45));
  TEST_ASSERT_EQUAL_INT(85, a21ClampServoY(100));
}

int main(int argc, char** argv) {
  UNITY_BEGIN();
  RUN_TEST(test_firmware_build_identity_contains_a21_release_fields);
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
  RUN_TEST(test_wifi_config_defaults_to_missing_credentials);
  RUN_TEST(test_wifi_config_redacts_password_in_status_text);
  RUN_TEST(test_wifi_config_rejects_legacy_project_ssid);
  RUN_TEST(test_connection_state_machine_uses_local_fallback_without_wifi_credentials);
  RUN_TEST(test_connection_state_machine_starts_wifi_when_credentials_are_valid);
  RUN_TEST(test_wifi_runtime_does_not_begin_without_credentials);
  RUN_TEST(test_wifi_runtime_begins_once_and_redacts_driver_state);
  RUN_TEST(test_wifi_runtime_moves_to_gateway_connecting_on_connected_status);
  RUN_TEST(test_wifi_runtime_retries_after_disconnect);
  RUN_TEST(test_gateway_ws_runtime_does_not_connect_before_wifi_ready);
  RUN_TEST(test_gateway_ws_runtime_begins_control_socket_once);
  RUN_TEST(test_gateway_ws_runtime_marks_gateway_connected);
  RUN_TEST(test_gateway_ws_runtime_applies_control_event_text);
  RUN_TEST(test_gateway_ws_runtime_enters_reconnect_after_disconnect);
  RUN_TEST(test_gateway_ws_send_mock_turn_builds_a21_device_event);
  RUN_TEST(test_gateway_ws_send_interrupt_increments_seq);
  RUN_TEST(test_gateway_ws_send_device_event_rejects_when_not_connected);
  RUN_TEST(test_audio_ws_runtime_waits_for_gateway_connection);
  RUN_TEST(test_audio_ws_runtime_begins_audio_socket_once);
  RUN_TEST(test_audio_ws_send_mock_frame_builds_a21_audio_frame);
  RUN_TEST(test_audio_ws_applies_gateway_ack_control_event);
  RUN_TEST(test_audio_ws_send_mock_frame_rejects_when_audio_not_connected);
  RUN_TEST(test_servo_y_angle_clamps_to_stackchan_safe_range);
  return UNITY_END();
}
