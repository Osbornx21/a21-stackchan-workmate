#include <unity.h>

#include "a21_firmware_connection.h"
#include "a21_firmware_audio_playback.h"
#include "a21_firmware_audio_ws.h"
#include "a21_firmware_config.h"
#include "a21_firmware_display.h"
#include "a21_firmware_network.h"
#include "a21_firmware_motion.h"
#include "a21_firmware_mic.h"
#include "a21_firmware_playback.h"
#include "a21_firmware_protocol.h"
#include "a21_firmware_rgb.h"
#include "a21_firmware_speaker.h"
#include "a21_firmware_state.h"
#include "a21_firmware_touch.h"
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
  char last_sent_text[A21_AUDIO_WS_TEXT_MESSAGE_CAP];
  bool connected;
  const char* pending_text;
  const char* pending_texts[4];
  uint8_t pending_text_count;
  uint8_t pending_text_cursor;
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
  if (driver->pending_text_cursor < driver->pending_text_count) {
    a21CopyString(output, output_size, driver->pending_texts[driver->pending_text_cursor]);
    driver->pending_text_cursor += 1;
    return true;
  }
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
  for (int i = 0; i < 4; ++i) {
    fake->pending_texts[i] = "";
  }
  fake->pending_text_count = 0;
  fake->pending_text_cursor = 0;
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
  for (int i = 0; i < 4; ++i) {
    fake->pending_texts[i] = "";
  }
  fake->pending_text_count = 0;
  fake->pending_text_cursor = 0;
  driver->ctx = fake;
  driver->begin = fakeGatewayWSBegin;
  driver->loop = fakeGatewayWSLoop;
  driver->connected = fakeGatewayWSConnected;
  driver->read_text = fakeGatewayWSReadText;
  driver->send_text = fakeGatewayWSSendText;
}

struct FakeMotionDriver {
  int write_count;
  int last_y_deg;
};

bool fakeMotionWriteY(void* ctx, int y_deg) {
  FakeMotionDriver* driver = static_cast<FakeMotionDriver*>(ctx);
  driver->write_count += 1;
  driver->last_y_deg = y_deg;
  return true;
}

void initFakeMotionDriver(FakeMotionDriver* fake, A21MotionDriver* driver) {
  fake->write_count = 0;
  fake->last_y_deg = -1;
  driver->ctx = fake;
  driver->write_y = fakeMotionWriteY;
}

struct FakeRGBDriver {
  int write_count;
  A21RGBColor last_color;
};

bool fakeRGBWrite(void* ctx, A21RGBColor color) {
  FakeRGBDriver* driver = static_cast<FakeRGBDriver*>(ctx);
  driver->write_count += 1;
  driver->last_color = color;
  return true;
}

void initFakeRGBDriver(FakeRGBDriver* fake, A21RGBDriver* driver) {
  fake->write_count = 0;
  fake->last_color = {0, 0, 0};
  driver->ctx = fake;
  driver->write = fakeRGBWrite;
}

struct FakeTouchDriver {
  int read_count;
  uint8_t cursor;
  uint8_t sample_count;
  A21TouchSample samples[4];
};

bool fakeTouchRead(void* ctx, A21TouchSample* sample) {
  FakeTouchDriver* driver = static_cast<FakeTouchDriver*>(ctx);
  driver->read_count += 1;
  if (driver->cursor >= driver->sample_count) {
    return false;
  }
  *sample = driver->samples[driver->cursor];
  driver->cursor += 1;
  return true;
}

void initFakeTouchDriver(FakeTouchDriver* fake, A21TouchDriver* driver) {
  fake->read_count = 0;
  fake->cursor = 0;
  fake->sample_count = 0;
  for (int i = 0; i < 4; ++i) {
    fake->samples[i] = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  }
  driver->ctx = fake;
  driver->read = fakeTouchRead;
}

struct FakePlaybackDriver {
  int start_count;
  int stop_count;
  int clear_count;
  char last_stream_id[A21_STREAM_ID_CAP];
  char last_stop_reason[32];
};

bool fakePlaybackStart(void* ctx, const char* stream_id) {
  FakePlaybackDriver* driver = static_cast<FakePlaybackDriver*>(ctx);
  driver->start_count += 1;
  a21CopyString(driver->last_stream_id, sizeof(driver->last_stream_id), stream_id);
  return true;
}

bool fakePlaybackStop(void* ctx, const char* reason) {
  FakePlaybackDriver* driver = static_cast<FakePlaybackDriver*>(ctx);
  driver->stop_count += 1;
  a21CopyString(driver->last_stop_reason, sizeof(driver->last_stop_reason), reason);
  return true;
}

bool fakePlaybackClear(void* ctx) {
  FakePlaybackDriver* driver = static_cast<FakePlaybackDriver*>(ctx);
  driver->clear_count += 1;
  return true;
}

void initFakePlaybackDriver(FakePlaybackDriver* fake, A21PlaybackDriver* driver) {
  fake->start_count = 0;
  fake->stop_count = 0;
  fake->clear_count = 0;
  fake->last_stream_id[0] = '\0';
  fake->last_stop_reason[0] = '\0';
  driver->ctx = fake;
  driver->start = fakePlaybackStart;
  driver->stop = fakePlaybackStop;
  driver->clear = fakePlaybackClear;
}

struct FakeSpeakerDriver {
  int play_count;
  uint8_t queued_count;
  const int16_t* last_samples;
  size_t last_sample_count;
  uint32_t last_sample_rate_hz;
  uint8_t last_channel;
  bool fail_play;
};

size_t fakeSpeakerQueued(void* ctx, uint8_t channel) {
  (void)channel;
  FakeSpeakerDriver* driver = static_cast<FakeSpeakerDriver*>(ctx);
  return driver->queued_count;
}

bool fakeSpeakerPlayPCM16(void* ctx, const int16_t* samples, size_t sample_count, uint32_t sample_rate_hz, uint8_t channel) {
  FakeSpeakerDriver* driver = static_cast<FakeSpeakerDriver*>(ctx);
  if (driver->fail_play) {
    return false;
  }
  driver->play_count += 1;
  driver->last_samples = samples;
  driver->last_sample_count = sample_count;
  driver->last_sample_rate_hz = sample_rate_hz;
  driver->last_channel = channel;
  return true;
}

void initFakeSpeakerDriver(FakeSpeakerDriver* fake, A21SpeakerDriver* driver) {
  fake->play_count = 0;
  fake->queued_count = 0;
  fake->last_samples = nullptr;
  fake->last_sample_count = 0;
  fake->last_sample_rate_hz = 0;
  fake->last_channel = 255;
  fake->fail_play = false;
  driver->ctx = fake;
  driver->queued = fakeSpeakerQueued;
  driver->play_pcm16 = fakeSpeakerPlayPCM16;
}

struct FakeMicDriver {
  int record_count;
  bool enabled;
  bool fail_record;
  int16_t fill_sample;
  size_t last_sample_count;
  uint32_t last_sample_rate_hz;
};

bool fakeMicEnabled(void* ctx) {
  FakeMicDriver* driver = static_cast<FakeMicDriver*>(ctx);
  return driver->enabled;
}

bool fakeMicRecordPCM16(void* ctx, int16_t* samples, size_t sample_count, uint32_t sample_rate_hz) {
  FakeMicDriver* driver = static_cast<FakeMicDriver*>(ctx);
  if (driver->fail_record || samples == nullptr) {
    return false;
  }
  driver->record_count += 1;
  driver->last_sample_count = sample_count;
  driver->last_sample_rate_hz = sample_rate_hz;
  for (size_t i = 0; i < sample_count; ++i) {
    samples[i] = driver->fill_sample;
  }
  return true;
}

void initFakeMicDriver(FakeMicDriver* fake, A21MicDriver* driver) {
  fake->record_count = 0;
  fake->enabled = true;
  fake->fail_record = false;
  fake->fill_sample = 0;
  fake->last_sample_count = 0;
  fake->last_sample_rate_hz = 0;
  driver->ctx = fake;
  driver->enabled = fakeMicEnabled;
  driver->record_pcm16 = fakeMicRecordPCM16;
}

void fillPCM16SilenceBase64(char* output, size_t output_size) {
  if (!a21FillPCM16SilenceBase64(output, output_size) && output != nullptr && output_size > 0) {
    output[0] = '\0';
  }
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

void test_display_state_label_makes_professional_mode_explicit() {
  TEST_ASSERT_EQUAL_STRING("PRO MODE", a21DisplayStateLabel(A21_RENDER_PROFESSIONAL));
  TEST_ASSERT_EQUAL_STRING("LISTENING", a21DisplayStateLabel(A21_RENDER_LISTENING));
  TEST_ASSERT_EQUAL_STRING("LOCAL", a21DisplayStateLabel(A21_RENDER_LOCAL));
}

void test_network_config_defaults_to_a21_gateway() {
  A21NetworkConfig config;
  a21InitNetworkConfig(&config);

  TEST_ASSERT_EQUAL_STRING("10.21.0.1", config.gateway_host);
  TEST_ASSERT_EQUAL_UINT16(21080, config.gateway_port);
  TEST_ASSERT_EQUAL_STRING("/ws/control", config.control_path);
  TEST_ASSERT_EQUAL_STRING("/ws/audio?device_id=stackchan-001", config.audio_path);
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
  TEST_ASSERT_EQUAL_STRING("ws://192.168.31.50:21080/ws/audio?device_id=stackchan-001", audio_url);
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
  TEST_ASSERT_EQUAL_STRING("disabled_m5unified_i2s_stop_crash_guard", doc["payload"]["capabilities"]["microphone"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["speaker"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["screen"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["screen_touch"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["top_touch"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["servo_y"] | "");
  TEST_ASSERT_EQUAL_STRING("available", doc["payload"]["capabilities"]["rgb"] | "");
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

void test_gateway_ws_send_runtime_echo_reports_applied_screen_motion_rgb() {
  A21GatewayWSRuntime runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  A21MotionRuntime motion_runtime;
  A21RGBRuntime rgb_runtime;
  FakeGatewayWSDriver fake;
  A21GatewayWSDriver driver;
  initFakeGatewayWSDriver(&fake, &driver);
  fake.connected = true;
  a21InitGatewayWSRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21CopyString(state.mode, A21_MODE_CAP, "workmate");
  state.render_state = A21_RENDER_SPEAKING;
  a21InitMotionRuntime(&motion_runtime);
  motion_runtime.has_y = true;
  motion_runtime.last_y_deg = 48;
  a21InitRGBRuntime(&rgb_runtime);
  rgb_runtime.has_color = true;
  rgb_runtime.last_color = a21RGBColorMake(0, 36, 48);
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 2000);

  TEST_ASSERT_TRUE(a21GatewayWSSendRuntimeEchoIfChanged(
      &runtime,
      &driver,
      &connection,
      &state,
      &motion_runtime,
      &rgb_runtime,
      2000));

  TEST_ASSERT_EQUAL(1, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.sent_device_events);
  TEST_ASSERT_EQUAL_UINT64(2, runtime.next_seq);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("a21.device.v1", doc["protocol"] | "");
  TEST_ASSERT_EQUAL_STRING("stackchan-001", doc["device_id"] | "");
  TEST_ASSERT_EQUAL_STRING("device.event", doc["kind"] | "");
  TEST_ASSERT_EQUAL_STRING("runtime.echo", doc["payload"]["event"] | "");
  TEST_ASSERT_EQUAL_STRING("workmate", doc["payload"]["mode"] | "");
  TEST_ASSERT_EQUAL_STRING("speaking", doc["payload"]["runtime_echo"]["screen"] | "");
  TEST_ASSERT_EQUAL_STRING("48deg", doc["payload"]["runtime_echo"]["servo_y"] | "");
  TEST_ASSERT_EQUAL_STRING("#002430", doc["payload"]["runtime_echo"]["rgb"] | "");
  TEST_ASSERT_EQUAL_STRING("a21-stackchan", doc["payload"]["firmware_id"] | "");
  TEST_ASSERT_EQUAL_STRING("m5stack-cores3", doc["payload"]["firmware_board"] | "");

  TEST_ASSERT_TRUE(a21GatewayWSSendRuntimeEchoIfChanged(
      &runtime,
      &driver,
      &connection,
      &state,
      &motion_runtime,
      &rgb_runtime,
      2020));
  TEST_ASSERT_EQUAL(1, fake.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.sent_device_events);
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
  TEST_ASSERT_EQUAL_STRING("/ws/audio?device_id=stackchan-001", fake.last_path);
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
  const char* data_base64 = doc["payload"]["data_base64"] | "";
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BASE64_CHARS), static_cast<uint32_t>(strlen(data_base64)));
  A21AudioPCMFrame frame;
  TEST_ASSERT_TRUE(a21DecodeBase64PCM16(data_base64, &frame));
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BYTES), static_cast<uint32_t>(frame.byte_count));
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

void test_parse_audio_playback_chunk_accepts_a21_downlink() {
  const char* json =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"audio.playback.chunk\","
      "\"seq\":8,"
      "\"trace_id\":\"a21-trace-playback-001\","
      "\"session_id\":\"a21-session-playback-001\","
      "\"payload\":{\"stream_id\":\"a21-audio-stream-000001\",\"codec\":\"pcm_s16le\",\"sample_rate_hz\":16000,\"channels\":1,\"duration_ms\":20,\"data_base64\":\"AAAA\"}}";

  A21AudioPlaybackChunk chunk;
  TEST_ASSERT_TRUE(a21ParseAudioPlaybackChunk(json, "stackchan-001", &chunk));
  TEST_ASSERT_EQUAL_STRING("a21-trace-playback-001", chunk.trace_id);
  TEST_ASSERT_EQUAL_STRING("a21-session-playback-001", chunk.session_id);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", chunk.stream_id);
  TEST_ASSERT_EQUAL_STRING("pcm_s16le", chunk.codec);
  TEST_ASSERT_EQUAL_UINT32(16000, chunk.sample_rate_hz);
  TEST_ASSERT_EQUAL_UINT8(1, chunk.channels);
  TEST_ASSERT_EQUAL_UINT16(20, chunk.duration_ms);
  TEST_ASSERT_EQUAL_STRING("AAAA", chunk.data_base64);
}

void test_parse_audio_playback_chunk_keeps_full_20ms_pcm_base64_payload() {
  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  char json[1400];
  snprintf(
      json,
      sizeof(json),
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"audio.playback.chunk\","
      "\"trace_id\":\"a21-trace-playback-long\","
      "\"session_id\":\"a21-session-playback-long\","
      "\"payload\":{\"stream_id\":\"a21-audio-stream-000001\",\"codec\":\"pcm_s16le\",\"sample_rate_hz\":16000,\"channels\":1,\"duration_ms\":20,\"data_base64\":\"%s\"}}",
      data);

  A21AudioPlaybackChunk chunk;
  TEST_ASSERT_TRUE(a21ParseAudioPlaybackChunk(json, "stackchan-001", &chunk));
  TEST_ASSERT_EQUAL_STRING(data, chunk.data_base64);
}

void test_audio_playback_chunk_decodes_full_pcm16_frame() {
  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.trace_id, A21_TRACE_ID_CAP, "a21-trace-playback-pcm");
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;

  A21AudioPCMFrame frame;
  TEST_ASSERT_TRUE(a21DecodeAudioPlaybackChunkPCM(&chunk, &frame));
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BYTES), static_cast<uint32_t>(frame.byte_count));
  TEST_ASSERT_EQUAL_UINT16(A21_AUDIO_PCM_FRAME_SAMPLES, frame.sample_count);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", frame.stream_id);
  TEST_ASSERT_EQUAL_STRING("a21-trace-playback-pcm", frame.trace_id);
  for (size_t i = 0; i < frame.byte_count; ++i) {
    TEST_ASSERT_EQUAL_UINT8(0, frame.data[i]);
  }
}

void test_audio_playback_chunk_rejects_invalid_base64_payload() {
  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, "not-a21-pcm");
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;

  A21AudioPCMFrame frame;
  TEST_ASSERT_FALSE(a21DecodeAudioPlaybackChunkPCM(&chunk, &frame));
  TEST_ASSERT_EQUAL_STRING("base64", frame.error);
}

void test_audio_playback_buffer_tracks_bounded_stream_chunks() {
  A21AudioPlaybackBuffer buffer;
  a21InitAudioPlaybackBuffer(&buffer);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;

  for (int i = 0; i < A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP + 2; ++i) {
    TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));
  }

  TEST_ASSERT_EQUAL_UINT8(A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(A21_AUDIO_PLAYBACK_BUFFER_CHUNK_CAP, buffer.total_chunks);
  TEST_ASSERT_EQUAL_UINT32(2, buffer.dropped_chunks);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", buffer.active_stream_id);
  TEST_ASSERT_EQUAL_UINT16(20, buffer.last_duration_ms);
}

void test_audio_playback_buffer_peeks_decoded_pcm_frame() {
  A21AudioPlaybackBuffer buffer;
  a21InitAudioPlaybackBuffer(&buffer);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.trace_id, A21_TRACE_ID_CAP, "a21-trace-playback-buffer");
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;

  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  const A21AudioPCMFrame* frame = a21AudioPlaybackBufferPeek(&buffer);
  TEST_ASSERT_NOT_NULL(frame);
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BYTES), static_cast<uint32_t>(frame->byte_count));
  TEST_ASSERT_EQUAL_UINT16(A21_AUDIO_PCM_FRAME_SAMPLES, frame->sample_count);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", frame->stream_id);
  TEST_ASSERT_EQUAL_STRING("a21-trace-playback-buffer", frame->trace_id);
}

void test_audio_playback_buffer_pops_decoded_pcm_frame() {
  A21AudioPlaybackBuffer buffer;
  a21InitAudioPlaybackBuffer(&buffer);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.trace_id, A21_TRACE_ID_CAP, "a21-trace-playback-pop");
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;

  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  A21AudioPCMFrame frame;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPop(&buffer, &frame));
  TEST_ASSERT_EQUAL_UINT8(0, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BYTES), static_cast<uint32_t>(frame.byte_count));
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", frame.stream_id);
  TEST_ASSERT_EQUAL_STRING("a21-trace-playback-pop", frame.trace_id);
  TEST_ASSERT_NULL(a21AudioPlaybackBufferPeek(&buffer));
  TEST_ASSERT_FALSE(a21AudioPlaybackBufferPop(&buffer, &frame));
}

void test_speaker_pump_plays_one_decoded_pcm_frame_when_queue_has_room() {
  A21AudioPlaybackBuffer buffer;
  A21SpeakerPumpRuntime runtime;
  A21FirmwareState state;
  FakeSpeakerDriver fake;
  A21SpeakerDriver driver;
  a21InitAudioPlaybackBuffer(&buffer);
  a21InitSpeakerPumpRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeSpeakerDriver(&fake, &driver);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));
  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.trace_id, A21_TRACE_ID_CAP, "a21-trace-speaker-pump");
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  TEST_ASSERT_TRUE(a21SpeakerPumpTick(&runtime, &driver, &state, &buffer));

  TEST_ASSERT_EQUAL_INT(1, fake.play_count);
  TEST_ASSERT_EQUAL_UINT8(0, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.frames_played);
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_SAMPLES), static_cast<uint32_t>(fake.last_sample_count));
  TEST_ASSERT_EQUAL_UINT32(A21_AUDIO_PCM_SAMPLE_RATE_HZ, fake.last_sample_rate_hz);
  TEST_ASSERT_EQUAL_UINT8(A21_SPEAKER_CHANNEL, fake.last_channel);
  TEST_ASSERT_NOT_NULL(fake.last_samples);
  TEST_ASSERT_EQUAL_INT16(0, fake.last_samples[0]);
  TEST_ASSERT_EQUAL_STRING("a21-trace-speaker-pump", runtime.last_trace_id);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", runtime.last_stream_id);
}

void test_speaker_pump_waits_when_driver_queue_is_full() {
  A21AudioPlaybackBuffer buffer;
  A21SpeakerPumpRuntime runtime;
  A21FirmwareState state;
  FakeSpeakerDriver fake;
  A21SpeakerDriver driver;
  a21InitAudioPlaybackBuffer(&buffer);
  a21InitSpeakerPumpRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeSpeakerDriver(&fake, &driver);
  fake.queued_count = A21_SPEAKER_MAX_DRIVER_QUEUE;

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));
  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  TEST_ASSERT_TRUE(a21SpeakerPumpTick(&runtime, &driver, &state, &buffer));

  TEST_ASSERT_EQUAL_INT(0, fake.play_count);
  TEST_ASSERT_EQUAL_UINT8(1, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_played);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.busy_ticks);
}

void test_speaker_pump_does_not_play_when_not_speaking() {
  A21AudioPlaybackBuffer buffer;
  A21SpeakerPumpRuntime runtime;
  A21FirmwareState state;
  FakeSpeakerDriver fake;
  A21SpeakerDriver driver;
  a21InitAudioPlaybackBuffer(&buffer);
  a21InitSpeakerPumpRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeSpeakerDriver(&fake, &driver);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));
  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  state.render_state = A21_RENDER_INTERRUPTED;
  TEST_ASSERT_TRUE(a21SpeakerPumpTick(&runtime, &driver, &state, &buffer));

  TEST_ASSERT_EQUAL_INT(0, fake.play_count);
  TEST_ASSERT_EQUAL_UINT8(1, buffer.queued_chunks);
}

void test_speaker_pump_keeps_frame_when_stream_id_mismatches_state() {
  A21AudioPlaybackBuffer buffer;
  A21SpeakerPumpRuntime runtime;
  A21FirmwareState state;
  FakeSpeakerDriver fake;
  A21SpeakerDriver driver;
  a21InitAudioPlaybackBuffer(&buffer);
  a21InitSpeakerPumpRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeSpeakerDriver(&fake, &driver);

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));
  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-other");
  TEST_ASSERT_TRUE(a21SpeakerPumpTick(&runtime, &driver, &state, &buffer));

  TEST_ASSERT_EQUAL_INT(0, fake.play_count);
  TEST_ASSERT_EQUAL_UINT8(1, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_played);
}

void test_mic_capture_records_one_frame_when_listening_and_speaker_idle() {
  A21MicCaptureRuntime runtime;
  A21FirmwareState state;
  FakeMicDriver fake;
  A21MicDriver driver;
  a21InitMicCaptureRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMicDriver(&fake, &driver);
  fake.fill_sample = 1234;
  state.render_state = A21_RENDER_LISTENING;

  TEST_ASSERT_TRUE(a21MicCaptureTick(&runtime, &driver, &state, 0, 4020));

  TEST_ASSERT_EQUAL_INT(1, fake.record_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.frames_captured);
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_SAMPLES), static_cast<uint32_t>(fake.last_sample_count));
  TEST_ASSERT_EQUAL_UINT32(A21_AUDIO_PCM_SAMPLE_RATE_HZ, fake.last_sample_rate_hz);
  TEST_ASSERT_EQUAL_INT16(1234, runtime.samples[0]);
  TEST_ASSERT_EQUAL_UINT32(4000, runtime.capture_started_at_ms);
  TEST_ASSERT_EQUAL_UINT32(4020, runtime.capture_ended_at_ms);
}

void test_core_s3_mic_capture_defaults_to_crash_guard_disabled() {
  TEST_ASSERT_FALSE(a21CoreS3MicCaptureEnabled());
  TEST_ASSERT_EQUAL_STRING("disabled_m5unified_i2s_stop_crash_guard", a21MicrophoneCapabilityStatus());
}

void test_mic_capture_skips_when_speaker_queue_is_active() {
  A21MicCaptureRuntime runtime;
  A21FirmwareState state;
  FakeMicDriver fake;
  A21MicDriver driver;
  a21InitMicCaptureRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMicDriver(&fake, &driver);
  state.render_state = A21_RENDER_LISTENING;

  TEST_ASSERT_TRUE(a21MicCaptureTick(&runtime, &driver, &state, 1, 4020));

  TEST_ASSERT_EQUAL_INT(0, fake.record_count);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_captured);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.skipped_speaker_busy);
}

void test_mic_capture_skips_when_render_state_is_speaking() {
  A21MicCaptureRuntime runtime;
  A21FirmwareState state;
  FakeMicDriver fake;
  A21MicDriver driver;
  a21InitMicCaptureRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMicDriver(&fake, &driver);
  state.render_state = A21_RENDER_SPEAKING;

  TEST_ASSERT_TRUE(a21MicCaptureTick(&runtime, &driver, &state, 0, 4020));

  TEST_ASSERT_EQUAL_INT(0, fake.record_count);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_captured);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.skipped_render_state);
}

void test_mic_capture_skips_when_render_state_is_thinking() {
  A21MicCaptureRuntime runtime;
  A21FirmwareState state;
  FakeMicDriver fake;
  A21MicDriver driver;
  a21InitMicCaptureRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMicDriver(&fake, &driver);
  state.render_state = A21_RENDER_THINKING;

  TEST_ASSERT_TRUE(a21MicCaptureTick(&runtime, &driver, &state, 0, 4020));

  TEST_ASSERT_EQUAL_INT(0, fake.record_count);
  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_captured);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.skipped_render_state);
}

void test_mic_capture_reports_driver_failure() {
  A21MicCaptureRuntime runtime;
  A21FirmwareState state;
  FakeMicDriver fake;
  A21MicDriver driver;
  a21InitMicCaptureRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMicDriver(&fake, &driver);
  fake.fail_record = true;
  state.render_state = A21_RENDER_LISTENING;

  TEST_ASSERT_FALSE(a21MicCaptureTick(&runtime, &driver, &state, 0, 4020));

  TEST_ASSERT_EQUAL_UINT32(0, runtime.frames_captured);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.driver_errors);
}

void test_pcm16_base64_encoder_preserves_nonzero_samples() {
  int16_t samples[A21_AUDIO_PCM_FRAME_SAMPLES];
  for (size_t i = 0; i < A21_AUDIO_PCM_FRAME_SAMPLES; ++i) {
    samples[i] = static_cast<int16_t>(i % 2 == 0 ? 1234 : -1234);
  }

  char encoded[A21_AUDIO_DATA_BASE64_CAP];
  TEST_ASSERT_TRUE(a21EncodePCM16Base64(samples, A21_AUDIO_PCM_FRAME_SAMPLES, encoded, sizeof(encoded)));
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BASE64_CHARS), static_cast<uint32_t>(strlen(encoded)));

  A21AudioPCMFrame decoded;
  TEST_ASSERT_TRUE(a21DecodeBase64PCM16(encoded, &decoded));
  TEST_ASSERT_EQUAL_UINT32(static_cast<uint32_t>(A21_AUDIO_PCM_FRAME_BYTES), static_cast<uint32_t>(decoded.byte_count));

  const uint16_t first = static_cast<uint16_t>(decoded.data[0]) | (static_cast<uint16_t>(decoded.data[1]) << 8);
  const uint16_t second = static_cast<uint16_t>(decoded.data[2]) | (static_cast<uint16_t>(decoded.data[3]) << 8);
  TEST_ASSERT_EQUAL_INT16(1234, static_cast<int16_t>(first));
  TEST_ASSERT_EQUAL_INT16(-1234, static_cast<int16_t>(second));
}

void test_audio_ws_sends_queued_mic_capture_frame_with_real_pcm_payload() {
  A21AudioWSRuntime audio_runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  A21MicCaptureRuntime mic_runtime;
  A21MicFrameQueue mic_queue;
  FakeMicDriver fake_mic;
  A21MicDriver mic_driver;
  FakeGatewayWSDriver fake_ws;
  A21AudioWSDriver audio_driver;
  initFakeMicDriver(&fake_mic, &mic_driver);
  initFakeAudioWSDriver(&fake_ws, &audio_driver);
  fake_ws.connected = true;
  fake_mic.fill_sample = 3210;
  a21InitAudioWSRuntime(&audio_runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitMicCaptureRuntime(&mic_runtime);
  a21InitMicFrameQueue(&mic_queue);
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);
  state.render_state = A21_RENDER_LISTENING;

  TEST_ASSERT_TRUE(a21MicCaptureTick(&mic_runtime, &mic_driver, &state, 0, 5020));
  TEST_ASSERT_TRUE(a21MicFrameQueuePushCapture(&mic_queue, &mic_runtime));
  TEST_ASSERT_EQUAL_UINT8(1, mic_queue.queued_frames);

  TEST_ASSERT_TRUE(a21AudioWSSendNextMicFrame(&audio_runtime, &audio_driver, &connection, &state, &mic_queue, 5030));

  TEST_ASSERT_EQUAL(1, fake_ws.send_count);
  TEST_ASSERT_EQUAL_UINT8(0, mic_queue.queued_frames);
  TEST_ASSERT_EQUAL_UINT32(1, audio_runtime.sent_audio_frames);
  TEST_ASSERT_EQUAL_UINT64(2, audio_runtime.next_seq);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake_ws.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("a21.device.v1", doc["protocol"] | "");
  TEST_ASSERT_EQUAL_STRING("stackchan-001", doc["device_id"] | "");
  TEST_ASSERT_EQUAL_STRING("audio.frame", doc["kind"] | "");
  TEST_ASSERT_EQUAL_STRING("a21-trace-audio-000001", doc["trace_id"] | "");
  TEST_ASSERT_EQUAL_INT64(5030, doc["sent_at_ms"] | 0);
  TEST_ASSERT_EQUAL_INT64(5000, doc["payload"]["capture_started_at_ms"] | 0);
  TEST_ASSERT_EQUAL_INT64(5020, doc["payload"]["capture_ended_at_ms"] | 0);

  const char* data_base64 = doc["payload"]["data_base64"] | "";
  A21AudioPCMFrame frame;
  TEST_ASSERT_TRUE(a21DecodeBase64PCM16(data_base64, &frame));
  const uint16_t first = static_cast<uint16_t>(frame.data[0]) | (static_cast<uint16_t>(frame.data[1]) << 8);
  TEST_ASSERT_EQUAL_INT16(3210, static_cast<int16_t>(first));
}

void test_audio_ws_buffers_playback_chunk_without_error_state() {
  A21AudioWSRuntime runtime;
  A21ConnectionState connection;
  A21NetworkConfig network;
  A21FirmwareState state;
  A21AudioPlaybackBuffer playback_buffer;
  FakeGatewayWSDriver fake;
  A21AudioWSDriver driver;
  initFakeAudioWSDriver(&fake, &driver);
  fake.connected = true;
  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));
  char playback_json[1400];
  snprintf(
      playback_json,
      sizeof(playback_json),
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"audio.playback.chunk\","
      "\"trace_id\":\"a21-trace-playback-002\","
      "\"session_id\":\"a21-session-playback-002\","
      "\"payload\":{\"stream_id\":\"a21-audio-stream-000001\",\"codec\":\"pcm_s16le\",\"sample_rate_hz\":16000,\"channels\":1,\"duration_ms\":20,\"data_base64\":\"%s\"}}",
      data);
  fake.pending_texts[0] =
      "{\"protocol\":\"a21.device.v1\","
      "\"device_id\":\"stackchan-001\","
      "\"kind\":\"control.event\","
      "\"trace_id\":\"a21-trace-playback-002\","
      "\"session_id\":\"a21-session-playback-002\","
      "\"payload\":{\"state\":\"speaking\",\"mode\":\"workmate\",\"text\":\"mock playback chunk\",\"stream_id\":\"a21-audio-stream-000001\",\"final\":true}}";
  fake.pending_texts[1] = playback_json;
  fake.pending_text_count = 2;
  a21InitAudioWSRuntime(&runtime);
  a21InitNetworkConfig(&network);
  a21InitFirmwareState(&state, "stackchan-001");
  a21InitAudioPlaybackBuffer(&playback_buffer);
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 1600);

  a21AudioWSRuntimeTickWithPlayback(&runtime, &driver, &connection, &network, &state, &playback_buffer, 2100);

  TEST_ASSERT_EQUAL_UINT32(0, runtime.invalid_control_events);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.received_control_events);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.received_audio_chunks);
  TEST_ASSERT_EQUAL_UINT8(1, playback_buffer.queued_chunks);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", playback_buffer.active_stream_id);
  TEST_ASSERT_EQUAL(A21_RENDER_SPEAKING, state.render_state);
  TEST_ASSERT_EQUAL_STRING("a21-audio-stream-000001", state.stream_id);
}

void test_audio_playback_buffer_clears_on_barge_in_state() {
  A21AudioPlaybackBuffer buffer;
  A21FirmwareState state;
  a21InitAudioPlaybackBuffer(&buffer);
  a21InitFirmwareState(&state, "stackchan-001");

  char data[900];
  fillPCM16SilenceBase64(data, sizeof(data));

  A21AudioPlaybackChunk chunk;
  a21ResetAudioPlaybackChunk(&chunk);
  a21CopyString(chunk.stream_id, A21_STREAM_ID_CAP, "a21-audio-stream-000001");
  a21CopyString(chunk.codec, A21_AUDIO_CODEC_CAP, "pcm_s16le");
  a21CopyString(chunk.data_base64, A21_AUDIO_DATA_BASE64_CAP, data);
  chunk.sample_rate_hz = 16000;
  chunk.channels = 1;
  chunk.duration_ms = 20;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferPush(&buffer, &chunk));

  state.render_state = A21_RENDER_INTERRUPTED;
  TEST_ASSERT_TRUE(a21AudioPlaybackBufferApplyState(&buffer, &state));

  TEST_ASSERT_EQUAL_UINT8(0, buffer.queued_chunks);
  TEST_ASSERT_EQUAL_UINT32(1, buffer.clear_count);
  TEST_ASSERT_EQUAL_STRING("", buffer.active_stream_id);
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

void test_touch_runtime_sends_wake_or_listen_with_semantic_source() {
  A21TouchRuntime touch;
  A21GatewayWSRuntime gateway_runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake_gateway;
  A21GatewayWSDriver gateway_driver;
  FakeTouchDriver fake_touch;
  A21TouchDriver touch_driver;
  initFakeGatewayWSDriver(&fake_gateway, &gateway_driver);
  initFakeTouchDriver(&fake_touch, &touch_driver);
  fake_gateway.connected = true;
  fake_touch.samples[0] = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_WAKE_OR_LISTEN};
  fake_touch.sample_count = 1;
  a21InitTouchRuntime(&touch);
  a21InitGatewayWSRuntime(&gateway_runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 2300);

  TEST_ASSERT_TRUE(a21TouchRuntimeTick(
      &touch,
      &touch_driver,
      &gateway_runtime,
      &gateway_driver,
      &connection,
      &state,
      2300));

  TEST_ASSERT_EQUAL_INT(1, fake_gateway.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, touch.handled_count);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_WAKE_OR_LISTEN, touch.last_intent);
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_SCREEN, touch.last_source);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake_gateway.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("touch.wake_or_listen", doc["payload"]["event"] | "");
  TEST_ASSERT_EQUAL_STRING("screen", doc["payload"]["touch_source"] | "");
  TEST_ASSERT_EQUAL_STRING("workmate", doc["payload"]["mode"] | "");
  TEST_ASSERT_EQUAL_STRING("先说，我在", doc["payload"]["text"] | "");
}

void test_touch_runtime_sends_barge_in_from_top_sensor() {
  A21TouchRuntime touch;
  A21GatewayWSRuntime gateway_runtime;
  A21ConnectionState connection;
  A21FirmwareState state;
  FakeGatewayWSDriver fake_gateway;
  A21GatewayWSDriver gateway_driver;
  FakeTouchDriver fake_touch;
  A21TouchDriver touch_driver;
  initFakeGatewayWSDriver(&fake_gateway, &gateway_driver);
  initFakeTouchDriver(&fake_touch, &touch_driver);
  fake_gateway.connected = true;
  fake_touch.samples[0] = {A21_TOUCH_SOURCE_TOP_SENSOR, A21_TOUCH_INTENT_BARGE_IN};
  fake_touch.sample_count = 1;
  a21InitTouchRuntime(&touch);
  a21InitGatewayWSRuntime(&gateway_runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  a21SetConnectionPhase(&connection, A21_CONN_GATEWAY_CONNECTED, 2400);

  TEST_ASSERT_TRUE(a21TouchRuntimeTick(
      &touch,
      &touch_driver,
      &gateway_runtime,
      &gateway_driver,
      &connection,
      &state,
      2400));

  TEST_ASSERT_EQUAL_INT(1, fake_gateway.send_count);
  TEST_ASSERT_EQUAL_UINT32(1, touch.handled_count);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_BARGE_IN, touch.last_intent);
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, touch.last_source);

  JsonDocument doc;
  TEST_ASSERT_FALSE(deserializeJson(doc, fake_gateway.last_sent_text));
  TEST_ASSERT_EQUAL_STRING("touch.barge_in", doc["payload"]["event"] | "");
  TEST_ASSERT_EQUAL_STRING("top_sensor", doc["payload"]["touch_source"] | "");
  TEST_ASSERT_FALSE(doc["payload"]["text"].is<const char*>());
}

void test_physical_touch_screen_reports_only_rising_edge() {
  A21PhysicalTouchState physical_touch;
  A21TouchSample sample = {A21_TOUCH_SOURCE_TOP_SENSOR, A21_TOUCH_INTENT_NONE};
  a21InitPhysicalTouchState(&physical_touch);

  TEST_ASSERT_TRUE(a21PhysicalTouchReadScreen(&physical_touch, true, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_SCREEN, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_WAKE_OR_LISTEN, sample.intent);
  TEST_ASSERT_FALSE(a21PhysicalTouchReadScreen(&physical_touch, true, &sample));
  TEST_ASSERT_FALSE(a21PhysicalTouchReadScreen(&physical_touch, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadScreen(&physical_touch, true, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_SCREEN, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_WAKE_OR_LISTEN, sample.intent);
}

void test_physical_touch_top_click_reports_barge_in() {
  A21PhysicalTouchState physical_touch;
  A21TouchSample sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  a21InitPhysicalTouchState(&physical_touch);

  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, false, false, false, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, true, true, false, false, false, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_BARGE_IN, sample.intent);
}

void test_physical_touch_top_tap_reports_tap_when_not_speaking() {
  A21PhysicalTouchState physical_touch;
  A21TouchSample sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  a21InitPhysicalTouchState(&physical_touch);

  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, false, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, false, false, true, false, false, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_TOP_TAP, sample.intent);
}

void test_physical_touch_top_swipes_report_direction_when_not_speaking() {
  A21PhysicalTouchState physical_touch;
  A21TouchSample sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  a21InitPhysicalTouchState(&physical_touch);

  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, false, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, true, false, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_TOP_SWIPE_FORWARD, sample.intent);
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, true, false, &sample));
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, false, false, false, false, &sample));

  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, false, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, false, true, false, false, true, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_TOP_SWIPE_BACKWARD, sample.intent);
}

void test_physical_touch_top_hold_reports_once_until_released() {
  A21PhysicalTouchState physical_touch;
  A21TouchSample sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  a21InitPhysicalTouchState(&physical_touch);

  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, true, true, false, false, false, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_BARGE_IN, sample.intent);
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, true, true, true, false, false, &sample));
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, true, false, true, false, false, &sample));
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, true, false, true, false, false, &sample));
  TEST_ASSERT_FALSE(a21PhysicalTouchReadTopSensor(&physical_touch, true, false, false, true, false, &sample));
  TEST_ASSERT_TRUE(a21PhysicalTouchReadTopSensor(&physical_touch, true, true, false, false, false, &sample));
  TEST_ASSERT_EQUAL(A21_TOUCH_SOURCE_TOP_SENSOR, sample.source);
  TEST_ASSERT_EQUAL(A21_TOUCH_INTENT_BARGE_IN, sample.intent);
}

void test_playback_runtime_starts_once_for_speaking_stream() {
  A21PlaybackRuntime runtime;
  A21FirmwareState state;
  FakePlaybackDriver fake;
  A21PlaybackDriver driver;
  a21InitPlaybackRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakePlaybackDriver(&fake, &driver);

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "stream-1");
  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_TRUE(runtime.playing);
  TEST_ASSERT_EQUAL_STRING("stream-1", runtime.active_stream_id);
  TEST_ASSERT_EQUAL_INT(1, fake.start_count);
  TEST_ASSERT_EQUAL_INT(0, fake.stop_count);
  TEST_ASSERT_EQUAL_INT(0, fake.clear_count);

  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(1, fake.start_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.start_count);
}

void test_playback_runtime_stops_and_clears_on_barge_in() {
  A21PlaybackRuntime runtime;
  A21FirmwareState state;
  FakePlaybackDriver fake;
  A21PlaybackDriver driver;
  a21InitPlaybackRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakePlaybackDriver(&fake, &driver);

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "stream-1");
  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));

  state.render_state = A21_RENDER_INTERRUPTED;
  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_FALSE(runtime.playing);
  TEST_ASSERT_EQUAL_STRING("", runtime.active_stream_id);
  TEST_ASSERT_EQUAL_INT(1, fake.stop_count);
  TEST_ASSERT_EQUAL_INT(1, fake.clear_count);
  TEST_ASSERT_EQUAL_STRING("barge_in", fake.last_stop_reason);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.stop_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.clear_count);
}

void test_playback_runtime_replaces_stream_with_stop_and_clear() {
  A21PlaybackRuntime runtime;
  A21FirmwareState state;
  FakePlaybackDriver fake;
  A21PlaybackDriver driver;
  a21InitPlaybackRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakePlaybackDriver(&fake, &driver);

  state.render_state = A21_RENDER_SPEAKING;
  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "stream-1");
  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));

  a21CopyString(state.stream_id, A21_STREAM_ID_CAP, "stream-2");
  TEST_ASSERT_TRUE(a21PlaybackRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_TRUE(runtime.playing);
  TEST_ASSERT_EQUAL_STRING("stream-2", runtime.active_stream_id);
  TEST_ASSERT_EQUAL_INT(2, fake.start_count);
  TEST_ASSERT_EQUAL_INT(1, fake.stop_count);
  TEST_ASSERT_EQUAL_INT(1, fake.clear_count);
  TEST_ASSERT_EQUAL_STRING("replace_stream", fake.last_stop_reason);
}

void test_servo_y_angle_clamps_to_stackchan_safe_range() {
  TEST_ASSERT_EQUAL_INT(5, A21_SERVO_Y_MIN_DEG);
  TEST_ASSERT_EQUAL_INT(85, A21_SERVO_Y_MAX_DEG);
  TEST_ASSERT_EQUAL_INT(5, a21ClampServoY(-30));
  TEST_ASSERT_EQUAL_INT(5, a21ClampServoY(0));
  TEST_ASSERT_EQUAL_INT(45, a21ClampServoY(45));
  TEST_ASSERT_EQUAL_INT(85, a21ClampServoY(100));
}

void test_motion_target_maps_render_states_to_safe_y_angles() {
  TEST_ASSERT_EQUAL_INT(45, a21MotionYForRenderState(A21_RENDER_IDLE));
  TEST_ASSERT_EQUAL_INT(38, a21MotionYForRenderState(A21_RENDER_LISTENING));
  TEST_ASSERT_EQUAL_INT(52, a21MotionYForRenderState(A21_RENDER_THINKING));
  TEST_ASSERT_EQUAL_INT(48, a21MotionYForRenderState(A21_RENDER_SPEAKING));
  TEST_ASSERT_EQUAL_INT(38, a21MotionYForRenderState(A21_RENDER_INTERRUPTED));
  TEST_ASSERT_EQUAL_INT(45, a21MotionYForRenderState(A21_RENDER_PROFESSIONAL));
  TEST_ASSERT_EQUAL_INT(45, a21MotionYForRenderState(A21_RENDER_LOCAL));
  TEST_ASSERT_EQUAL_INT(45, a21MotionYForRenderState(A21_RENDER_ERROR));
}

void test_motion_runtime_writes_once_per_y_angle_change() {
  A21MotionRuntime runtime;
  A21FirmwareState state;
  FakeMotionDriver fake;
  A21MotionDriver driver;
  a21InitMotionRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeMotionDriver(&fake, &driver);

  state.render_state = A21_RENDER_LISTENING;
  TEST_ASSERT_TRUE(a21MotionRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(1, fake.write_count);
  TEST_ASSERT_EQUAL_INT(38, fake.last_y_deg);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.applied_count);

  TEST_ASSERT_TRUE(a21MotionRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(1, fake.write_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.applied_count);

  state.render_state = A21_RENDER_SPEAKING;
  TEST_ASSERT_TRUE(a21MotionRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(2, fake.write_count);
  TEST_ASSERT_EQUAL_INT(48, fake.last_y_deg);
  TEST_ASSERT_EQUAL_UINT32(2, runtime.applied_count);
}

void test_rgb_target_maps_render_states_to_state_colors() {
  A21RGBColor idle = a21RGBForRenderState(A21_RENDER_IDLE);
  TEST_ASSERT_EQUAL_UINT8(16, idle.r);
  TEST_ASSERT_EQUAL_UINT8(16, idle.g);
  TEST_ASSERT_EQUAL_UINT8(16, idle.b);

  A21RGBColor listening = a21RGBForRenderState(A21_RENDER_LISTENING);
  TEST_ASSERT_EQUAL_UINT8(0, listening.r);
  TEST_ASSERT_EQUAL_UINT8(48, listening.g);
  TEST_ASSERT_EQUAL_UINT8(16, listening.b);

  A21RGBColor thinking = a21RGBForRenderState(A21_RENDER_THINKING);
  TEST_ASSERT_EQUAL_UINT8(48, thinking.r);
  TEST_ASSERT_EQUAL_UINT8(32, thinking.g);
  TEST_ASSERT_EQUAL_UINT8(0, thinking.b);

  A21RGBColor speaking = a21RGBForRenderState(A21_RENDER_SPEAKING);
  TEST_ASSERT_EQUAL_UINT8(0, speaking.r);
  TEST_ASSERT_EQUAL_UINT8(36, speaking.g);
  TEST_ASSERT_EQUAL_UINT8(48, speaking.b);

  A21RGBColor interrupted = a21RGBForRenderState(A21_RENDER_INTERRUPTED);
  TEST_ASSERT_EQUAL_UINT8(64, interrupted.r);
  TEST_ASSERT_EQUAL_UINT8(24, interrupted.g);
  TEST_ASSERT_EQUAL_UINT8(0, interrupted.b);

  A21RGBColor professional = a21RGBForRenderState(A21_RENDER_PROFESSIONAL);
  TEST_ASSERT_EQUAL_UINT8(0, professional.r);
  TEST_ASSERT_EQUAL_UINT8(16, professional.g);
  TEST_ASSERT_EQUAL_UINT8(64, professional.b);

  A21RGBColor local = a21RGBForRenderState(A21_RENDER_LOCAL);
  TEST_ASSERT_EQUAL_UINT8(8, local.r);
  TEST_ASSERT_EQUAL_UINT8(8, local.g);
  TEST_ASSERT_EQUAL_UINT8(8, local.b);

  A21RGBColor error = a21RGBForRenderState(A21_RENDER_ERROR);
  TEST_ASSERT_EQUAL_UINT8(64, error.r);
  TEST_ASSERT_EQUAL_UINT8(0, error.g);
  TEST_ASSERT_EQUAL_UINT8(0, error.b);
}

void test_rgb_runtime_writes_once_per_color_change() {
  A21RGBRuntime runtime;
  A21FirmwareState state;
  FakeRGBDriver fake;
  A21RGBDriver driver;
  a21InitRGBRuntime(&runtime);
  a21InitFirmwareState(&state, "stackchan-001");
  initFakeRGBDriver(&fake, &driver);

  state.render_state = A21_RENDER_LISTENING;
  TEST_ASSERT_TRUE(a21RGBRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(1, fake.write_count);
  TEST_ASSERT_EQUAL_UINT8(0, fake.last_color.r);
  TEST_ASSERT_EQUAL_UINT8(48, fake.last_color.g);
  TEST_ASSERT_EQUAL_UINT8(16, fake.last_color.b);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.applied_count);

  TEST_ASSERT_TRUE(a21RGBRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(1, fake.write_count);
  TEST_ASSERT_EQUAL_UINT32(1, runtime.applied_count);

  state.render_state = A21_RENDER_ERROR;
  TEST_ASSERT_TRUE(a21RGBRuntimeApplyState(&runtime, &driver, &state));
  TEST_ASSERT_EQUAL_INT(2, fake.write_count);
  TEST_ASSERT_EQUAL_UINT8(64, fake.last_color.r);
  TEST_ASSERT_EQUAL_UINT8(0, fake.last_color.g);
  TEST_ASSERT_EQUAL_UINT8(0, fake.last_color.b);
  TEST_ASSERT_EQUAL_UINT32(2, runtime.applied_count);
}

int main(int argc, char** argv) {
  UNITY_BEGIN();
  RUN_TEST(test_firmware_build_identity_contains_a21_release_fields);
  RUN_TEST(test_parse_control_event_listening);
  RUN_TEST(test_parse_control_event_rejects_wrong_protocol);
  RUN_TEST(test_parse_control_event_rejects_wrong_device);
  RUN_TEST(test_apply_control_event_updates_runtime_state);
  RUN_TEST(test_apply_invalid_control_event_enters_error_state);
  RUN_TEST(test_display_state_label_makes_professional_mode_explicit);
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
  RUN_TEST(test_gateway_ws_send_runtime_echo_reports_applied_screen_motion_rgb);
  RUN_TEST(test_audio_ws_runtime_waits_for_gateway_connection);
  RUN_TEST(test_audio_ws_runtime_begins_audio_socket_once);
  RUN_TEST(test_audio_ws_send_mock_frame_builds_a21_audio_frame);
  RUN_TEST(test_audio_ws_applies_gateway_ack_control_event);
  RUN_TEST(test_parse_audio_playback_chunk_accepts_a21_downlink);
  RUN_TEST(test_parse_audio_playback_chunk_keeps_full_20ms_pcm_base64_payload);
  RUN_TEST(test_audio_playback_chunk_decodes_full_pcm16_frame);
  RUN_TEST(test_audio_playback_chunk_rejects_invalid_base64_payload);
  RUN_TEST(test_audio_playback_buffer_tracks_bounded_stream_chunks);
  RUN_TEST(test_audio_playback_buffer_peeks_decoded_pcm_frame);
  RUN_TEST(test_audio_playback_buffer_pops_decoded_pcm_frame);
  RUN_TEST(test_speaker_pump_plays_one_decoded_pcm_frame_when_queue_has_room);
  RUN_TEST(test_speaker_pump_waits_when_driver_queue_is_full);
  RUN_TEST(test_speaker_pump_does_not_play_when_not_speaking);
  RUN_TEST(test_speaker_pump_keeps_frame_when_stream_id_mismatches_state);
  RUN_TEST(test_mic_capture_records_one_frame_when_listening_and_speaker_idle);
  RUN_TEST(test_core_s3_mic_capture_defaults_to_crash_guard_disabled);
  RUN_TEST(test_mic_capture_skips_when_speaker_queue_is_active);
  RUN_TEST(test_mic_capture_skips_when_render_state_is_speaking);
  RUN_TEST(test_mic_capture_skips_when_render_state_is_thinking);
  RUN_TEST(test_mic_capture_reports_driver_failure);
  RUN_TEST(test_pcm16_base64_encoder_preserves_nonzero_samples);
  RUN_TEST(test_audio_ws_sends_queued_mic_capture_frame_with_real_pcm_payload);
  RUN_TEST(test_audio_ws_buffers_playback_chunk_without_error_state);
  RUN_TEST(test_audio_playback_buffer_clears_on_barge_in_state);
  RUN_TEST(test_audio_ws_send_mock_frame_rejects_when_audio_not_connected);
  RUN_TEST(test_touch_runtime_sends_wake_or_listen_with_semantic_source);
  RUN_TEST(test_touch_runtime_sends_barge_in_from_top_sensor);
  RUN_TEST(test_physical_touch_screen_reports_only_rising_edge);
  RUN_TEST(test_physical_touch_top_click_reports_barge_in);
  RUN_TEST(test_physical_touch_top_tap_reports_tap_when_not_speaking);
  RUN_TEST(test_physical_touch_top_swipes_report_direction_when_not_speaking);
  RUN_TEST(test_physical_touch_top_hold_reports_once_until_released);
  RUN_TEST(test_playback_runtime_starts_once_for_speaking_stream);
  RUN_TEST(test_playback_runtime_stops_and_clears_on_barge_in);
  RUN_TEST(test_playback_runtime_replaces_stream_with_stop_and_clear);
  RUN_TEST(test_servo_y_angle_clamps_to_stackchan_safe_range);
  RUN_TEST(test_motion_target_maps_render_states_to_safe_y_angles);
  RUN_TEST(test_motion_runtime_writes_once_per_y_angle_change);
  RUN_TEST(test_rgb_target_maps_render_states_to_state_colors);
  RUN_TEST(test_rgb_runtime_writes_once_per_color_change);
  return UNITY_END();
}
