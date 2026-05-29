#include <M5Unified.h>

#include "a21_firmware_config.h"
#include "a21_firmware_connection.h"
#include "a21_firmware_network.h"
#include "a21_firmware_state.h"

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
A21ConnectionState g_connection;
A21RenderState g_last_render_state = A21_RENDER_ERROR;
char g_last_text[A21_TEXT_CAP] = "";
char g_last_connection_text[A21_CONNECTION_TEXT_CAP] = "";

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
  a21InitConnectionState(&g_connection, &g_network, millis());
  if (!a21ValidateNetworkConfig(&g_network)) {
    a21CopyString(g_state.text, A21_TEXT_CAP, "A21 Gateway config error");
    a21CopyString(g_state.last_error, A21_ERROR_CAP, "network_config");
    g_state.render_state = A21_RENDER_ERROR;
  }
  drawIfChanged();
}

void loop() {
  M5.update();
  drawIfChanged();
  delay(20);
}
