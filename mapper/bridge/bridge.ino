#include <Adafruit_NeoPixel.h>

// Define pins and LED counts for each strip.
#define STRIP_1_PIN 16
#define STRIP_1_LED 98

#define STRIP_2_PIN 17
#define STRIP_2_LED 102

#define STRIP_3_PIN 18
#define STRIP_3_LED 55

#define STRIP_4_PIN 19
#define STRIP_4_LED 82

#define STRIP_5_PIN 21
#define STRIP_5_LED 70

#define STRIP_6_PIN 22
#define STRIP_6_LED 21

#define STRIP_7_PIN 23
#define STRIP_7_LED 22

#define STRIP_8_PIN 25
#define STRIP_8_LED 19

#define STRIP_9_PIN 26
#define STRIP_9_LED 11

// Instantiate each strip.
Adafruit_NeoPixel strip1(STRIP_1_LED, STRIP_1_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip2(STRIP_2_LED, STRIP_2_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip3(STRIP_3_LED, STRIP_3_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip4(STRIP_4_LED, STRIP_4_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip5(STRIP_5_LED, STRIP_5_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip6(STRIP_6_LED, STRIP_6_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip7(STRIP_7_LED, STRIP_7_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip8(STRIP_8_LED, STRIP_8_PIN, NEO_GRB + NEO_KHZ800);
Adafruit_NeoPixel strip9(STRIP_9_LED, STRIP_9_PIN, NEO_GRB + NEO_KHZ800);

// Create an array of strips for easy iteration.
Adafruit_NeoPixel* strips[] = { &strip1, &strip2, &strip3, &strip4, &strip5, &strip6, &strip7, &strip8, &strip9 };
const int numStrips = sizeof(strips) / sizeof(strips[0]);

// For each strip we know its LED count.
const int ledCounts[9] = { STRIP_1_LED, STRIP_2_LED, STRIP_3_LED, STRIP_4_LED,
                             STRIP_5_LED, STRIP_6_LED, STRIP_7_LED, STRIP_8_LED, STRIP_9_LED };

// To simplify storage we use a fixed maximum LED count (here 102, the maximum among our strips).
const int maxLeds = 102;

// Instead of storing labels, we now simply track if an LED was visited.
bool visited[9][102] = { { false } };

// Color definitions (set in setup, using strip1's Color() method).
uint32_t VISITED_COLOR;   // Green for visited LEDs.
uint32_t CURRENT_COLOR;   // White for the current LED.

// Navigation state.
int currentStrip = 0;
int currentPixel = 0;

//
// Helper function: mark the current LED as visited.
//
void markCurrentVisited() {
  visited[currentStrip][currentPixel] = true;
}

//
// Update the display for all strips:
// - The current LED is shown in white.
// - Any LED that has been visited is shown in green.
// - All others are turned off.
//
void updateDisplay() {
  for (int i = 0; i < numStrips; i++) {
    int numPixels = strips[i]->numPixels();
    for (int j = 0; j < numPixels; j++) {
      if (i == currentStrip && j == currentPixel) {
        strips[i]->setPixelColor(j, CURRENT_COLOR);
      } 
      else if (visited[i][j]) {
        strips[i]->setPixelColor(j, VISITED_COLOR);
      } 
      else {
        strips[i]->setPixelColor(j, 0);
      }
    }
    strips[i]->show();
  }
}

//
// Print the current navigation position.
//
void printCurrentPosition() {
  Serial.print("Current Position -> Strip: ");
  Serial.print(currentStrip);
  Serial.print(", LED: ");
  Serial.println(currentPixel);
}

//
// Print all visited LEDs.
//
void printAllVisited() {
  Serial.println("Visited LEDs:");
  for (int i = 0; i < numStrips; i++) {
    int numPixels = strips[i]->numPixels();
    for (int j = 0; j < numPixels; j++) {
      if (visited[i][j]) {
        Serial.print("Strip ");
        Serial.print(i);
        Serial.print(" LED ");
        Serial.println(j);
      }
    }
  }
}

//
// Navigate forward to the next LED (wrapping among strips as needed).
//
void navigateForward() {
  int numPixels = strips[currentStrip]->numPixels();
  currentPixel++;
  if (currentPixel >= numPixels) {
    if (currentStrip < numStrips - 1) {
      currentStrip++;
      currentPixel = 0;
    } else {
      currentStrip = 0;
      currentPixel = 0;
    }
  }
  markCurrentVisited();
  updateDisplay();
  printCurrentPosition();
}

void turnOffAllLeds() {
  for (int i = 0; i < numStrips; i++) {
    strips[i]->clear();
    strips[i]->show();
  }
  Serial.println("All LEDs turned off.");
}

//
// Navigate backward to the previous LED (wrapping among strips as needed).
//
void navigateBackward() {
  currentPixel--;
  if (currentPixel < 0) {
    if (currentStrip > 0) {
      currentStrip--;
      currentPixel = strips[currentStrip]->numPixels() - 1;
    } else {
      currentStrip = numStrips - 1;
      currentPixel = strips[currentStrip]->numPixels() - 1;
    }
  }
  markCurrentVisited();
  updateDisplay();
  printCurrentPosition();
}

//
// Handle a jump command. Expected format: "jump <strip> <LED>".
//
void handleJump(String input) {
  int firstSpace = input.indexOf(' ');
  if (firstSpace == -1) {
    Serial.println("Invalid jump command. Format: jump <strip> <LED>");
    return;
  }
  String params = input.substring(firstSpace + 1);
  params.trim();
  int secondSpace = params.indexOf(' ');
  if (secondSpace == -1) {
    Serial.println("Invalid jump command. Format: jump <strip> <LED>");
    return;
  }
  String stripStr = params.substring(0, secondSpace);
  String ledStr = params.substring(secondSpace + 1);
  int targetStrip = stripStr.toInt();
  int targetLED = ledStr.toInt();
  if (targetStrip < 0 || targetStrip >= numStrips) {
    Serial.println("Invalid strip number.");
    return;
  }
  if (targetLED < 0 || targetLED >= strips[targetStrip]->numPixels()) {
    Serial.println("Invalid LED index.");
    return;
  }
  currentStrip = targetStrip;
  currentPixel = targetLED;
  markCurrentVisited();
  updateDisplay();
  printCurrentPosition();
}

//
// Setup: initialize serial, strips, colors, and starting LED.
//
void setup() {
  Serial.begin(9600);
  
  // Initialize all strips.
  for (int i = 0; i < numStrips; i++) {
    strips[i]->begin();
    strips[i]->setBrightness(10);
    strips[i]->clear();
    strips[i]->show();
  }
  
  // Define our colors (using strip1 to create the color value).
  VISITED_COLOR = strip1.Color(0, 255, 0);     // Green for visited.
  CURRENT_COLOR = strip1.Color(255, 255, 255);   // White for the current LED.
  
  // Mark the starting LED as visited.
  markCurrentVisited();
  
  Serial.println("Navigation mode started.");
  Serial.println("Commands:");
  Serial.println("  [           -> Move backward");
  Serial.println("  ]           -> Move forward");
  Serial.println("  jump s l    -> Jump to strip 's' and LED 'l'");
  Serial.println("  !           -> Print all visited LEDs");
  
  updateDisplay();
  printCurrentPosition();
}

//
// Main loop: read serial commands and call the appropriate function.
//
void loop() {
  if (Serial.available() > 0) {
    String input = Serial.readStringUntil('\n');
    input.trim();
    if (input.length() == 0)
      return;
      
    if (input == "[") {
      navigateBackward();
    } 
    else if (input == "]") {
      navigateForward();
    } 
    else if (input.startsWith("jump ")) {
      handleJump(input);
    } 
    else if (input == "!") {
      printAllVisited();
    } 
    else if (input == "off") {
      turnOffAllLeds();
    }
    else {
      Serial.println("Invalid input. Use [ , ] , jump <strip> <LED>, or off.");
    }
  }
}
