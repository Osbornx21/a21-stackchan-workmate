#include <M5Unified.h>

#include "a21_firmware_config.h"

namespace {

void drawIdentityScreen() {
  M5.Display.fillScreen(TFT_BLACK);
  M5.Display.setTextColor(TFT_WHITE, TFT_BLACK);
  M5.Display.setTextDatum(middle_center);
  M5.Display.setFont(&fonts::Font2);
  M5.Display.drawString("A21", M5.Display.width() / 2, M5.Display.height() / 2 - 28);
  M5.Display.drawString(A21_FIRMWARE_VERSION, M5.Display.width() / 2, M5.Display.height() / 2);
  M5.Display.drawString("LOCAL", M5.Display.width() / 2, M5.Display.height() / 2 + 28);
}

}  // namespace

void setup() {
  auto config = M5.config();
  M5.begin(config);
  drawIdentityScreen();
}

void loop() {
  M5.update();
  delay(20);
}
