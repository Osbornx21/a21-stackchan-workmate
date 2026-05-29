#include <M5Unified.h>
#include <WebSocketsClient.h>
#include <WiFi.h>

#include "a21_firmware_config.h"
#include "a21_firmware_connection.h"
#include "a21_firmware_gateway_ws.h"
#include "a21_firmware_network.h"
#include "a21_firmware_state.h"
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
      return TFT_BLUE;
    case A21_RENDER_ERROR:
      return TFT_RED;
    case A21_RENDER_LOCAL:
      return TFT_DARKGREY;
    case A21_RENDER_IDLE:
    default:
      return TFT_WHITE;
  }
}

const char* state_label(A21RenderState state) {
  switch (state) {
    case A21_RENDER_LISTENING:
      return "LISTENING";
    case A21_RENDER_THINKING:
      return "THINKING";
    case A21_RENDER_SPEAKING:
      return "SPEAKING";
    case A21_RENDER_INTERRUPTED:
      return "INTERRUPTED";
    case A21_RENDER_PROFESSIONAL:
      return "PRO";
    case A21_RENDER_ERROR:
      return "ERROR";
    case A21_RENDER_LOCAL:
      return "LOCAL";
    case A21_RENDER_IDLE:
    default:
      return "IDLE";
  }
}

void drawStateScreen(const A21FirmwareState& state, const A21NetworkConfig& network, const A21ConnectionState& connection) {
  const uint32_t accent = state_color(state.render_state);
  M5.Display.fillScreen(TFT_BLACK);
  M5.Display.setTextColor(TFT_WHITE, TFT_BLACK);
  M5.Display.setTextDatum(middle_center);
  M5.Display.setFont(&fonts::Font2);
  M5.Display.drawString("A21", M5.Display.width() / 2, M5.Display.height() / 2 - 28);
  M5.Display.drawString(A21_FIRMWARE_VERSION, M5.Display.width() / 2, M5.Display.height() / 2);
  M5.Display.setTextColor(accent, TFT_BLACK);
  M5.Display.drawString(state_label(state.render_state), M5.Display.width() / 2, M5.Display.height() / 2 + 28);
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

struct A21ArduinoGatewayWS {
  WebSocketsClient client;
  bool connected;
  bool has_text;
  char text[A21_WS_TEXT_MESSAGE_CAP];
};

A21ArduinoGatewayWS g_gateway_ws_client;

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

void gatewayWSEvent(WStype_t type, uint8_t* payload, size_t length) {
  switch (type) {
    case WStype_CONNECTED:
      g_gateway_ws_client.connected = true;
      break;
    case WStype_DISCONNECTED:
      g_gateway_ws_client.connected = false;
      g_gateway_ws_client.has_text = false;
      break;
    case WStype_TEXT: {
      const size_t copy_len = length < (A21_WS_TEXT_MESSAGE_CAP - 1) ? length : (A21_WS_TEXT_MESSAGE_CAP - 1);
      memcpy(g_gateway_ws_client.text, payload, copy_len);
      g_gateway_ws_client.text[copy_len] = '\0';
      g_gateway_ws_client.has_text = true;
      break;
    }
    default:
      break;
  }
}

bool arduinoGatewayWSBegin(void* ctx, const char* host, uint16_t port, const char* path) {
  A21ArduinoGatewayWS* client = static_cast<A21ArduinoGatewayWS*>(ctx);
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

void arduinoGatewayWSLoop(void* ctx) {
  A21ArduinoGatewayWS* client = static_cast<A21ArduinoGatewayWS*>(ctx);
  if (client != nullptr) {
    client->client.loop();
  }
}

bool arduinoGatewayWSConnected(void* ctx) {
  A21ArduinoGatewayWS* client = static_cast<A21ArduinoGatewayWS*>(ctx);
  return client != nullptr && client->connected;
}

bool arduinoGatewayWSReadText(void* ctx, char* output, size_t output_size) {
  A21ArduinoGatewayWS* client = static_cast<A21ArduinoGatewayWS*>(ctx);
  if (client == nullptr || output == nullptr || output_size == 0 || !client->has_text) {
    return false;
  }
  a21CopyString(output, output_size, client->text);
  client->has_text = false;
  client->text[0] = '\0';
  return true;
}

A21GatewayWSDriver g_gateway_ws_driver = {
    &g_gateway_ws_client,
    arduinoGatewayWSBegin,
    arduinoGatewayWSLoop,
    arduinoGatewayWSConnected,
    arduinoGatewayWSReadText,
};

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
  M5.begin(config);
  a21InitFirmwareState(&g_state, A21_DEVICE_ID);
  a21InitNetworkConfig(&g_network);
  a21InitWiFiConfig(&g_wifi);
  a21InitConnectionStateWithWiFi(&g_connection, &g_network, &g_wifi, millis());
  a21InitWiFiRuntime(&g_wifi_runtime);
  a21InitGatewayWSRuntime(&g_gateway_ws_runtime);
  if (!a21ValidateNetworkConfig(&g_network)) {
    a21CopyString(g_state.text, A21_TEXT_CAP, "A21 Gateway config error");
    a21CopyString(g_state.last_error, A21_ERROR_CAP, "network_config");
    g_state.render_state = A21_RENDER_ERROR;
  }
  drawIfChanged();
}

void loop() {
  M5.update();
  a21WiFiRuntimeTick(&g_wifi_runtime, &g_wifi_driver, &g_connection, &g_wifi, millis());
  a21GatewayWSRuntimeTick(&g_gateway_ws_runtime, &g_gateway_ws_driver, &g_connection, &g_network, &g_state, millis());
  drawIfChanged();
  delay(20);
}
