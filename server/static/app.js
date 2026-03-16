/**
 * app.js — Live subway train visualizer (server mode)
 * Uses board.js for PCB rendering, adds protobuf pixel fetching.
 */

const board = new Board('c', 'tooltip');
board.brightnessBoost = 255.0 / 20.0;
board.boardSvgOpacity = 0.07;

let lastSeq = 0;
let activeCount = 0;
let lastFetchOk = false;

const statusDot = document.getElementById('dot');
const statusInfo = document.getElementById('info');

function decodePixelFrame(buf) {
  const bytes = new Uint8Array(buf);
  let pos = 0;
  let result = { timestamp: 0, sequence: 0, ledCount: 0, pixels: null };
  function readVarint() {
    let val = 0, shift = 0;
    while (pos < bytes.length) {
      const b = bytes[pos++];
      val |= (b & 0x7f) << shift; shift += 7;
      if ((b & 0x80) === 0) break;
    }
    return val >>> 0;
  }
  while (pos < bytes.length) {
    const tag = readVarint();
    const fieldNum = tag >>> 3, wireType = tag & 0x7;
    if (wireType === 0) {
      const val = readVarint();
      if (fieldNum === 1) result.timestamp = val;
      else if (fieldNum === 2) result.sequence = val;
      else if (fieldNum === 3) result.ledCount = val;
    } else if (wireType === 2) {
      const len = readVarint();
      if (fieldNum === 4) result.pixels = bytes.slice(pos, pos + len);
      pos += len;
    } else if (wireType === 5) { pos += 4; }
    else if (wireType === 1) { pos += 8; }
  }
  return result;
}

async function fetchPixels() {
  try {
    const resp = await fetch('/api/v1/pixels');
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const buf = await resp.arrayBuffer();
    if (buf.byteLength === 0) { board.clearPixels(); lastFetchOk = true; activeCount = 0; updateStatus(); return; }
    const frame = decodePixelFrame(buf);
    lastSeq = frame.sequence;
    if (frame.pixels && frame.pixels.length >= TOTAL_LEDS * 3) {
      board.setPixels(frame.pixels);
    }
    activeCount = 0;
    for (let i = 0; i < TOTAL_LEDS; i++) {
      if (board.pixelColors[i*3] || board.pixelColors[i*3+1] || board.pixelColors[i*3+2]) activeCount++;
    }
    lastFetchOk = true;
  } catch (e) { lastFetchOk = false; }
  updateStatus();
}

function updateStatus() {
  statusDot.className = 'dot ' + (lastFetchOk ? 'ok' : 'err');
  statusInfo.textContent = lastFetchOk ? `${activeCount} trains | seq ${lastSeq}` : 'disconnected';
}

async function init() {
  await board.init('/static/leds.json', '/static/board.svg');
  board.startDrawLoop();
  await fetchPixels();
  setInterval(fetchPixels, 1000);
}

init();
