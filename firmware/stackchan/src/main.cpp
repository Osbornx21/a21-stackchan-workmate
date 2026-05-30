#include <M5Unified.h>
#include <M5StackChan.h>
#include <WebSocketsClient.h>
#include <WiFi.h>

#include "a21_firmware_config.h"
#include "a21_firmware_audio_playback.h"
#include "a21_firmware_audio_ws.h"
#include "a21_firmware_connection.h"
#include "a21_firmware_display.h"
#include "a21_firmware_gateway_ws.h"
#include "a21_firmware_mic.h"
#include "a21_firmware_motion.h"
#include "a21_firmware_network.h"
#include "a21_firmware_playback.h"
#include "a21_firmware_rgb.h"
#include "a21_firmware_speaker.h"
#include "a21_firmware_state.h"
#include "a21_firmware_touch.h"
#include "a21_firmware_wifi.h"
#include "a21_firmware_wifi_runtime.h"

#include <string.h>

namespace {

uint32_t state_color(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
      return TFT_GREEN;
    case A21_RENDER_THINKING:
      return TFT_YELLOW;
    case A21_RENDER_SPEAKING:
      return TFT_CYAN;
    case A21_RENDER_INTERRUPTED:
      return TFT_ORANGE;
    case A21_RENDER_PROFESSIONAL:
      return TFT_CYAN;
    case A21_RENDER_ERROR:
      return TFT_RED;
    case A21_RENDER_LOCAL:
      return TFT_DARKGREY;
    case A21_RENDER_IDLE:
    default:
      return TFT_WHITE;
  }
}

void drawStateScreen(const A21FirmwareState& state, const A21NetworkConfig& network, const A21ConnectionState& connection) {
  const uint32_t accent = state_color(state.render_state);
  char firmware_label[A21_FIRMWARE_LABEL_CAP];
  if (!a21BuildFirmwareLabel(firmware_label, sizeof(firmware_label))) {
    a21ConfigCopyString(firmware_label, sizeof(firmware_label), A21_FIRMWARE_VERSION);
  }
  M5.Display.fillScreen(TFT_BLACK);
  M5.Display.setTextColor(TFT_WHITE, TFT_BLACK);
  M5.Display.setTextDatum(middle_center);
  M5.Display.setFont(&fonts::Font2);
  M5.Display.drawString("A21", M5.Display.width() / 2, M5.Display.height() / 2 - 28);
  M5.Display.drawString(firmware_label, M5.Display.width() / 2, M5.Display.height() / 2);
  M5.Display.setTextColor(accent, TFT_BLACK);
  M5.Display.drawString(a21DisplayStateLabel(state.render_state), M5.Display.width() / 2, M5.Display.height() / 2 + 28);
  M5.Display.setTextDatum(top_center);
  M5.Display.setTextColor(TFT_LIGHTGREY, TFT_BLACK);
  char gateway_line[96];
  snprintf(gateway_line, sizeof(gateway_line), "%s:%u", network.gateway_host, network.gateway_port);
  M5.Display.drawString(gateway_line, M5.Display.width() / 2, 8);
  M5.Display.drawString(connection.status_text, M5.Display.width() / 2, 28);
  M5.Display.drawString(state.text, M5.Display.width() / 2, M5.Display.height() - 36);
}

A21FirmwareState g_state;
A21NetworkConfig g_network;
A21WiFiConfig g_wifi;
A21ConnectionState g_connection;
A21RenderState g_last_render_state = A21_RENDER_ERROR;
char g_last_text[A21_TEXT_CAP] = "";
char g_last_connection_text[A21_CONNECTION_TEXT_CAP] = "";
A21WiFiRuntime g_wifi_runtime;
A21GatewayWSRuntime g_gateway_ws_runtime;
A21AudioWSRuntime g_audio_ws_runtime;
A21MotionRuntime g_motion_runtime;
A21RGBRuntime g_rgb_runtime;
A21TouchRuntime g_touch_runtime;
A21PlaybackRuntime g_playback_runtime;
A21AudioPlaybackBuffer g_audio_playback_buffer;
A21SpeakerPumpRuntime g_speaker_pump_runtime;
A21MicCaptureRuntime g_mic_capture_runtime;
A21MicFrameQueue g_mic_frame_queue;

static constexpr size_t A21_ARDUINO_WS_TEXT_MESSAGE_CAP =
    A21_AUDIO_WS_TEXT_MESSAGE_CAP > A21_WS_TEXT_MESSAGE_CAP ? A21_AUDIO_WS_TEXT_MESSAGE_CAP : A21_WS_TEXT_MESSAGE_CAP;

struct A21ArduinoTextWS {
  WebSocketsClient client;
  bool connected;
  bool has_text;
  char text[A21_ARDUINO_WS_TEXT_MESSAGE_CAP];
};

A21ArduinoTextWS g_gateway_ws_client;
A21ArduinoTextWS g_audio_ws_client;

bool arduinoWiFiBegin(void* ctx, const char* ssid, const char* password) {
  (void)ctx;
  WiFi.mode(WIFI_STA);
  WiFi.begin(ssid, password);
  return true;
}

A21WiFiDriverStatus arduinoWiFiStatus(void* ctx) {
  (void)ctx;
  switch (WiFi.status()) {
    case WL_CONNECTED:
      return A21_WIFI_DRIVER_CONNECTED;
    case WL_IDLE_STATUS:
      return A21_WIFI_DRIVER_CONNECTING;
    default:
      return A21_WIFI_DRIVER_DISCONNECTED;
  }
}

bool arduinoWiFiLocalIP(void* ctx, char* output, size_t output_size) {
  (void)ctx;
  if (output == nullptr || output_size == 0) {
    return false;
  }
  IPAddress ip = WiFi.localIP();
  const int written = snprintf(output, output_size, "%u.%u.%u.%u", ip[0], ip[1], ip[2], ip[3]);
  return written > 0 && static_cast<size_t>(written) < output_size;
}

A21WiFiDriver g_wifi_driver = {
    nullptr,
    arduinoWiFiBegin,
    arduinoWiFiStatus,
    arduinoWiFiLocalIP,
};

void applyArduinoWSEvent(A21ArduinoTextWS* client, WStype_t type, uint8_t* payload, size_t length) {
  if (client == nullptr) {
    return;
  }
  switch (type) {
    case WStype_CONNECTED:
      client->connected = true;
      break;
    case WStype_DISCONNECTED:
      client->connected = false;
      client->has_text = false;
      break;
    case WStype_TEXT: {
      const size_t copy_len = length < (A21_ARDUINO_WS_TEXT_MESSAGE_CAP - 1) ? length : (A21_ARDUINO_WS_TEXT_MESSAGE_CAP - 1);
      memcpy(client->text, payload, copy_len);
      client->text[copy_len] = '\0';
      client->has_text = true;
      break;
    }
    default:
      break;
  }
}

void gatewayWSEvent(WStype_t type, uint8_t* payload, size_t length) {
  applyArduinoWSEvent(&g_gateway_ws_client, type, payload, length);
}

void audioWSEvent(WStype_t type, uint8_t* payload, size_t length) {
  applyArduinoWSEvent(&g_audio_ws_client, type, payload, length);
}

bool arduinoGatewayWSBegin(void* ctx, const char* host, uint16_t port, const char* path) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  if (client == nullptr) {
    return false;
  }
  client->connected = false;
  client->has_text = false;
  client->client.begin(host, port, path);
  client->client.onEvent(gatewayWSEvent);
  client->client.setReconnectInterval(0);
  return true;
}

bool arduinoAudioWSBegin(void* ctx, const char* host, uint16_t port, const char* path) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  if (client == nullptr) {
    return false;
  }
  client->connected = false;
  client->has_text = false;
  client->client.begin(host, port, path);
  client->client.onEvent(audioWSEvent);
  client->client.setReconnectInterval(0);
  return true;
}

void arduinoGatewayWSLoop(void* ctx) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  if (client != nullptr) {
    client->client.loop();
  }
}

bool arduinoGatewayWSConnected(void* ctx) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  return client != nullptr && client->connected;
}

bool arduinoGatewayWSReadText(void* ctx, char* output, size_t output_size) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  if (client == nullptr || output == nullptr || output_size == 0 || !client->has_text) {
    return false;
  }
  a21CopyString(output, output_size, client->text);
  client->has_text = false;
  client->text[0] = '\0';
  return true;
}

bool arduinoGatewayWSSendText(void* ctx, const char* text) {
  A21ArduinoTextWS* client = static_cast<A21ArduinoTextWS*>(ctx);
  if (client == nullptr || text == nullptr || text[0] == '\0') {
    return false;
  }
  return client->client.sendTXT(text);
}

A21GatewayWSDriver g_gateway_ws_driver = {
    &g_gateway_ws_client,
    arduinoGatewayWSBegin,
    arduinoGatewayWSLoop,
    arduinoGatewayWSConnected,
    arduinoGatewayWSReadText,
    arduinoGatewayWSSendText,
};

A21AudioWSDriver g_audio_ws_driver = {
    &g_audio_ws_client,
    arduinoAudioWSBegin,
    arduinoGatewayWSLoop,
    arduinoGatewayWSConnected,
    arduinoGatewayWSReadText,
    arduinoGatewayWSSendText,
};

bool arduinoMotionWriteY(void* ctx, int y_deg) {
  (void)ctx;
  const int safe_y_deg = a21ClampServoY(y_deg);
  M5StackChan.Motion.moveY(safe_y_deg * 10, 300);
  return true;
}

A21MotionDriver g_motion_driver = {
    nullptr,
    arduinoMotionWriteY,
};

bool arduinoRGBWrite(void* ctx, A21RGBColor color) {
  (void)ctx;
  for (int led_index = 0; led_index < 12; ++led_index) {
    M5StackChan.setRgbColor(led_index, color.r, color.g, color.b);
  }
  M5StackChan.refreshRgb();
  return true;
}

A21RGBDriver g_rgb_driver = {
    nullptr,
    arduinoRGBWrite,
};

struct A21ArduinoTouchState {
  bool has_sample;
  A21TouchSample sample;
};

A21ArduinoTouchState g_touch_state;
A21PhysicalTouchState g_physical_touch_state;

bool arduinoTouchRead(void* ctx, A21TouchSample* sample) {
  A21ArduinoTouchState* state = static_cast<A21ArduinoTouchState*>(ctx);
  if (state == nullptr || sample == nullptr || !state->has_sample) {
    return false;
  }
  *sample = state->sample;
  state->has_sample = false;
  return true;
}

A21TouchDriver g_touch_driver = {
    &g_touch_state,
    arduinoTouchRead,
};

void queueTouchSample(A21TouchSample sample) {
  g_touch_state.sample = sample;
  g_touch_state.has_sample = true;
}

void arduinoEndMicForSpeaker() {
  if (!M5.Mic.isRunning()) {
    return;
  }
  while (M5.Mic.isRecording()) {
    M5.delay(1);
  }
  M5.Mic.end();
}

bool arduinoPlaybackStart(void* ctx, const char* stream_id) {
  (void)ctx;
  (void)stream_id;
  arduinoEndMicForSpeaker();
  return M5.Speaker.begin();
}

bool arduinoPlaybackStop(void* ctx, const char* reason) {
  (void)ctx;
  (void)reason;
  M5.Speaker.stop(A21_SPEAKER_CHANNEL);
  return true;
}

bool arduinoPlaybackClear(void* ctx) {
  (void)ctx;
  M5.Speaker.stop(A21_SPEAKER_CHANNEL);
  return true;
}

A21PlaybackDriver g_playback_driver = {
    nullptr,
    arduinoPlaybackStart,
    arduinoPlaybackStop,
    arduinoPlaybackClear,
};

size_t arduinoSpeakerQueued(void* ctx, uint8_t channel) {
  (void)ctx;
  return M5.Speaker.isPlaying(channel);
}

bool arduinoSpeakerPlayPCM16(void* ctx, const int16_t* samples, size_t sample_count, uint32_t sample_rate_hz, uint8_t channel) {
  (void)ctx;
  if (samples == nullptr || sample_count == 0) {
    return false;
  }
  arduinoEndMicForSpeaker();
  if (!M5.Speaker.begin()) {
    return false;
  }
  return M5.Speaker.playRaw(samples, sample_count, sample_rate_hz, false, 1, channel, false);
}

A21SpeakerDriver g_speaker_driver = {
    nullptr,
    arduinoSpeakerQueued,
    arduinoSpeakerPlayPCM16,
};

bool arduinoMicEnabled(void* ctx) {
  (void)ctx;
  if (!a21CoreS3MicCaptureEnabled()) {
    return false;
  }
  return !M5.Speaker.isPlaying(A21_SPEAKER_CHANNEL);
}

bool arduinoWaitMicIdle(uint32_t timeout_ms) {
  const uint32_t started_at_ms = millis();
  while (M5.Mic.isRecording()) {
    delay(1);
    if (millis() - started_at_ms > timeout_ms) {
      return false;
    }
  }
  return true;
}

bool arduinoStartMicForCapture(uint32_t sample_rate_hz) {
  if (M5.Speaker.isPlaying(A21_SPEAKER_CHANNEL)) {
    return false;
  }
  if (M5.Speaker.isRunning()) {
    M5.Speaker.end();
  }
  M5.Mic.setSampleRate(sample_rate_hz);
  if (M5.Mic.isRunning()) {
    return true;
  }
  return M5.Mic.begin();
}

bool arduinoMicRecordPCM16(void* ctx, int16_t* samples, size_t sample_count, uint32_t sample_rate_hz) {
  (void)ctx;
  if (samples == nullptr || sample_count != A21_AUDIO_PCM_FRAME_SAMPLES ||
      sample_rate_hz != A21_AUDIO_PCM_SAMPLE_RATE_HZ ||
      M5.Speaker.isPlaying(A21_SPEAKER_CHANNEL)) {
    return false;
  }
  if (!arduinoStartMicForCapture(sample_rate_hz)) {
    return false;
  }
  if (!arduinoWaitMicIdle(80)) {
    return false;
  }
  if (!M5.Mic.record(samples, sample_count, sample_rate_hz, false)) {
    return false;
  }
  const uint32_t started_at_ms = millis();
  const uint32_t min_capture_wait_ms = A21_AUDIO_PCM_DURATION_MS + 5;
  const uint32_t max_capture_wait_ms = A21_AUDIO_PCM_DURATION_MS + 120;
  while (millis() - started_at_ms < min_capture_wait_ms) {
    delay(1);
  }
  while (M5.Mic.isRecording()) {
    delay(1);
    if (millis() - started_at_ms > max_capture_wait_ms) {
      return false;
    }
  }
  return true;
}

A21MicDriver g_mic_driver = {
    nullptr,
    arduinoMicEnabled,
    arduinoMicRecordPCM16,
};

void handleLocalControls(uint32_t now_ms) {
  int16_t touch_x = 0;
  int16_t touch_y = 0;
  const bool screen_touching = M5StackChan.Display().getTouch(&touch_x, &touch_y);
  A21TouchSample physical_sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  if (a21PhysicalTouchReadScreen(&g_physical_touch_state, screen_touching, &physical_sample)) {
    queueTouchSample(physical_sample);
  }

  auto& top_touch = M5StackChan.TouchSensor;
  if (a21PhysicalTouchReadTopSensor(
          &g_physical_touch_state,
          g_state.render_state == A21_RENDER_SPEAKING,
          top_touch.isPressed(),
          top_touch.wasClicked(),
          top_touch.wasSwipedForward(),
          top_touch.wasSwipedBackward(),
          &physical_sample)) {
    queueTouchSample(physical_sample);
  }

  if (M5.BtnA.wasClicked()) {
    queueTouchSample({A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_WAKE_OR_LISTEN});
  }
  if (M5.BtnB.wasClicked()) {
    queueTouchSample({A21_TOUCH_SOURCE_TOP_SENSOR, A21_TOUCH_INTENT_BARGE_IN});
  }
  if (M5.BtnC.wasClicked()) {
    a21AudioWSSendMockFrame(
        &g_audio_ws_runtime,
        &g_audio_ws_driver,
        &g_connection,
        &g_state,
        now_ms);
  }
  a21TouchRuntimeTick(
      &g_touch_runtime,
      &g_touch_driver,
      &g_gateway_ws_runtime,
      &g_gateway_ws_driver,
      &g_connection,
      &g_state,
      now_ms);
}

void drawIfChanged() {
  if (g_last_render_state == g_state.render_state &&
      strcmp(g_last_text, g_state.text) == 0 &&
      strcmp(g_last_connection_text, g_connection.status_text) == 0) {
    return;
  }
  g_last_render_state = g_state.render_state;
  a21CopyString(g_last_text, A21_TEXT_CAP, g_state.text);
  a21CopyString(g_last_connection_text, A21_CONNECTION_TEXT_CAP, g_connection.status_text);
  drawStateScreen(g_state, g_network, g_connection);
}

}  // namespace

void setup() {
  auto config = M5.config();
  config.internal_spk = true;
  M5.begin(config);
  M5StackChan.begin();
  M5.Speaker.setVolume(96);
  M5.Speaker.begin();
  a21InitFirmwareState(&g_state, A21_DEVICE_ID);
  a21InitNetworkConfig(&g_network);
  a21InitWiFiConfig(&g_wifi);
  a21InitConnectionStateWithWiFi(&g_connection, &g_network, &g_wifi, millis());
  a21InitWiFiRuntime(&g_wifi_runtime);
  a21InitGatewayWSRuntime(&g_gateway_ws_runtime);
  a21InitAudioWSRuntime(&g_audio_ws_runtime);
  a21InitMotionRuntime(&g_motion_runtime);
  a21InitRGBRuntime(&g_rgb_runtime);
  a21InitTouchRuntime(&g_touch_runtime);
  a21InitPlaybackRuntime(&g_playback_runtime);
  a21InitAudioPlaybackBuffer(&g_audio_playback_buffer);
  a21InitSpeakerPumpRuntime(&g_speaker_pump_runtime);
  a21InitMicCaptureRuntime(&g_mic_capture_runtime);
  a21InitMicFrameQueue(&g_mic_frame_queue);
  a21InitPhysicalTouchState(&g_physical_touch_state);
  g_touch_state.has_sample = false;
  g_touch_state.sample = {A21_TOUCH_SOURCE_SCREEN, A21_TOUCH_INTENT_NONE};
  if (!a21ValidateNetworkConfig(&g_network)) {
    a21CopyString(g_state.text, A21_TEXT_CAP, "A21 Gateway config error");
    a21CopyString(g_state.last_error, A21_ERROR_CAP, "network_config");
    g_state.render_state = A21_RENDER_ERROR;
  }
  drawIfChanged();
}

void loop() {
  M5StackChan.update();
  const uint32_t now_ms = millis();
  a21WiFiRuntimeTick(&g_wifi_runtime, &g_wifi_driver, &g_connection, &g_wifi, now_ms);
  a21GatewayWSRuntimeTick(&g_gateway_ws_runtime, &g_gateway_ws_driver, &g_connection, &g_network, &g_state, now_ms);
  a21AudioWSRuntimeTickWithPlayback(
      &g_audio_ws_runtime,
      &g_audio_ws_driver,
      &g_connection,
      &g_network,
      &g_state,
      &g_audio_playback_buffer,
      now_ms);
  a21PlaybackRuntimeApplyState(&g_playback_runtime, &g_playback_driver, &g_state);
  a21SpeakerPumpTick(&g_speaker_pump_runtime, &g_speaker_driver, &g_state, &g_audio_playback_buffer);
  a21AudioPlaybackBufferApplyState(&g_audio_playback_buffer, &g_state);
  const size_t speaker_queue_depth = g_speaker_driver.queued(g_speaker_driver.ctx, A21_SPEAKER_CHANNEL);
  const uint32_t mic_frames_before = g_mic_capture_runtime.frames_captured;
  if (a21MicCaptureTick(&g_mic_capture_runtime, &g_mic_driver, &g_state, speaker_queue_depth, now_ms) &&
      g_mic_capture_runtime.frames_captured > mic_frames_before) {
    a21MicFrameQueuePushCapture(&g_mic_frame_queue, &g_mic_capture_runtime);
  }
  a21AudioWSSendNextMicFrame(
      &g_audio_ws_runtime,
      &g_audio_ws_driver,
      &g_connection,
      &g_state,
      &g_mic_frame_queue,
      now_ms);
  a21MotionRuntimeApplyState(&g_motion_runtime, &g_motion_driver, &g_state);
  a21RGBRuntimeApplyState(&g_rgb_runtime, &g_rgb_driver, &g_state);
  handleLocalControls(now_ms);
  drawIfChanged();
  A21RuntimeEchoDiagnostics runtime_diagnostics = {};
  runtime_diagnostics.enabled = true;
  runtime_diagnostics.mic_frames_captured = g_mic_capture_runtime.frames_captured;
  runtime_diagnostics.mic_driver_errors = g_mic_capture_runtime.driver_errors;
  runtime_diagnostics.mic_skipped_render_state = g_mic_capture_runtime.skipped_render_state;
  runtime_diagnostics.mic_skipped_speaker_busy = g_mic_capture_runtime.skipped_speaker_busy;
  runtime_diagnostics.mic_skipped_unavailable = g_mic_capture_runtime.skipped_unavailable;
  runtime_diagnostics.mic_queue_depth = g_mic_frame_queue.queued_frames;
  runtime_diagnostics.mic_queue_total_frames = g_mic_frame_queue.total_frames;
  runtime_diagnostics.mic_queue_dropped_frames = g_mic_frame_queue.dropped_frames;
  runtime_diagnostics.audio_ws_sent_audio_frames = g_audio_ws_runtime.sent_audio_frames;
  runtime_diagnostics.mic_last_abs_peak = g_mic_capture_runtime.last_abs_peak;
  runtime_diagnostics.mic_last_nonzero_samples = g_mic_capture_runtime.last_nonzero_samples;
  runtime_diagnostics.playback_buffer_queued_chunks = g_audio_playback_buffer.queued_chunks;
  runtime_diagnostics.playback_buffer_total_chunks = g_audio_playback_buffer.total_chunks;
  runtime_diagnostics.playback_buffer_dropped_chunks = g_audio_playback_buffer.dropped_chunks;
  runtime_diagnostics.playback_buffer_clear_count = g_audio_playback_buffer.clear_count;
  runtime_diagnostics.speaker_frames_played = g_speaker_pump_runtime.frames_played;
  runtime_diagnostics.speaker_busy_ticks = g_speaker_pump_runtime.busy_ticks;
  runtime_diagnostics.speaker_driver_errors = g_speaker_pump_runtime.driver_errors;
  a21CopyString(
      runtime_diagnostics.speaker_last_stream_id,
      A21_STREAM_ID_CAP,
      g_speaker_pump_runtime.last_stream_id);
  a21GatewayWSSendRuntimeEchoIfChangedWithDiagnostics(
      &g_gateway_ws_runtime,
      &g_gateway_ws_driver,
      &g_connection,
      &g_state,
      &g_motion_runtime,
      &g_rgb_runtime,
      &runtime_diagnostics,
      now_ms);
  delay(20);
}
