#pragma once
#include <climits>
#include <cstdint>

// Buffer ceilings
static constexpr uint32_t kMaxStrips = 16;
static constexpr uint32_t kMaxLeds = 512;
static constexpr uint32_t kMaxLedsPerStation = 32;

// Protobuf-derived sizes (must match subway.pb.h)
static constexpr uint32_t kMaxStations = 300;
static constexpr uint32_t kMaxConfig = 80;
static constexpr uint32_t kHashLen = 65;
static constexpr uint32_t kStopIdLen = 8;

// Server configuration
static constexpr const char* kDefaultServerUrl = "https://device.pcb.nyc";
static constexpr uint32_t kServerUrlMaxLen = 128;

// Firmware version
static constexpr const char* kFirmwareVersion = "0.7.0";

// Timing
static constexpr uint32_t kPollIntervalMs = 1000;
static constexpr uint32_t kRenderIntervalMs = 30;
static constexpr uint32_t kMutexTimeoutMs = 500;

// HTTP
static constexpr uint32_t kHttpMaxRetries = 3;
static constexpr uint32_t kHttpBufSize = 16384;

// LED
static constexpr uint8_t kDefaultBrightness = 10;

// OTA
static constexpr uint32_t kOtaCheckIntervalMin = 60;

// Lua
static constexpr uint32_t kLuaMaxMem = 30 * 1024;
static constexpr uint32_t kLuaMaxInstructions = 0;
static constexpr uint32_t kMaxConsecutiveFailures = 5;
