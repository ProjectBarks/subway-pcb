#include <Arduino.h>
#include <Adafruit_NeoPixel.h>

#define NUM_STRIPS 9
const uint8_t pins[NUM_STRIPS]  = {16, 17, 18, 19, 21, 22, 23, 25, 26};
const uint16_t leds[NUM_STRIPS] = {97, 102, 55, 81, 70, 21, 22, 19, 11};

Adafruit_NeoPixel* strips[NUM_STRIPS];

void setup() {
  Serial.begin(115200);
  for (int i = 0; i < NUM_STRIPS; i++) {
    strips[i] = new Adafruit_NeoPixel(leds[i], pins[i], NEO_GRB + NEO_KHZ800);
    strips[i]->begin();
    strips[i]->setBrightness(30);
    strips[i]->clear();
    strips[i]->show();
  }
  Serial.println("ready");
}

void loop() {
  if (!Serial.available()) return;
  String cmd = Serial.readStringUntil('\n');
  cmd.trim();

  if (cmd == "ping") {
    Serial.println("pong");
  } else if (cmd == "clear") {
    for (int i = 0; i < NUM_STRIPS; i++) { strips[i]->clear(); strips[i]->show(); }
    Serial.println("ok");
  } else if (cmd.startsWith("on ")) {
    int s, p, r, g, b;
    if (sscanf(cmd.c_str() + 3, "%d %d %d %d %d", &s, &p, &r, &g, &b) == 5 &&
        s >= 0 && s < NUM_STRIPS && p >= 0 && p < leds[s]) {
      strips[s]->setPixelColor(p, strips[s]->Color(r, g, b));
      strips[s]->show();
      Serial.println("ok");
    } else {
      Serial.println("err");
    }
  } else {
    Serial.println("err");
  }
}
