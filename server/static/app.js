let LEDS = [];

const STRIP_SIZES = [97, 102, 55, 81, 70, 21, 22, 19, 11];
const TOTAL_LEDS = 478;
const BRIGHTNESS_BOOST = 255.0 / 20.0; // Boost back to full brightness for display

let pixelColors = new Uint8Array(TOTAL_LEDS * 3);
let lastFetchOk = false;
let activeCount = 0;
let lastSeq = 0;
let hoveredLED = -1;
let boardPaths = null;

const canvas = document.getElementById('c');
const ctx = canvas.getContext('2d');
const tooltip = document.getElementById('tooltip');
const statusDot = document.getElementById('dot');
const statusInfo = document.getElementById('info');

let boardMinX, boardMaxX, boardMinY, boardMaxY, boardW, boardH;

function computeBounds() {
  const xs = LEDS.map(l => l[0]);
  const ys = LEDS.map(l => l[1]);
  boardMinX = Math.min(...xs) - 15;
  boardMaxX = Math.max(...xs) + 15;
  boardMinY = Math.min(...ys) - 15;
  boardMaxY = Math.max(...ys) + 15;
  boardW = boardMaxX - boardMinX;
  boardH = boardMaxY - boardMinY;
}

let scale = 1, offsetX = 0, offsetY = 0, ledRadius = 3;

function resize() {
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.parentElement.getBoundingClientRect();
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  const pad = 20;
  const cw = rect.width - pad * 2;
  const ch = rect.height - pad * 2;
  scale = Math.min(cw / boardW, ch / boardH);
  offsetX = pad + (cw - boardW * scale) / 2;
  offsetY = pad + (ch - boardH * scale) / 2;
  ledRadius = Math.max(2.5, Math.min(5, scale * 1.8));
}

function toScreen(x, y) {
  // Mirror Y axis only
  const cy = (boardMinY + boardMaxY) / 2;
  const ry = cy - (y - cy);
  return [offsetX + (x - boardMinX) * scale, offsetY + (ry - boardMinY) * scale];
}

function screenToLedIndex(sx, sy) {
  let best = -1, bestDist = ledRadius * 3;
  for (let i = 0; i < LEDS.length; i++) {
    const [lx, ly] = toScreen(LEDS[i][0], LEDS[i][1]);
    const d = Math.hypot(sx - lx, sy - ly);
    if (d < bestDist) { bestDist = d; best = i; }
  }
  return best;
}

function getStripPixel(idx) {
  let off = 0;
  for (let s = 0; s < STRIP_SIZES.length; s++) {
    if (idx < off + STRIP_SIZES[s]) return [s, idx - off];
    off += STRIP_SIZES[s];
  }
  return [-1, -1];
}

// Load SVG board image (KiCad-exported, has all silkscreen lines)
let boardImg = null;
let boardImgLoaded = false;
function loadBoardImage() {
  boardImg = new Image();
  boardImg.onload = () => { boardImgLoaded = true; };
  boardImg.src = '/static/board.svg';
}

function draw() {
  const rect = canvas.parentElement.getBoundingClientRect();
  ctx.clearRect(0, 0, rect.width, rect.height);

  // Draw SVG board background
  // SVG viewBox: 0,0 to 297,420 mm (KiCad coords, Y increases downward)
  // LED coords use same X but negative Y (KiCad convention)
  // toScreen mirrors Y, so SVG Y=0 maps to KiCad Y=0, SVG Y=420 maps to KiCad Y=-420
  if (boardImgLoaded) {
    // SVG corners in KiCad coords: top-left (0, 0) -> kicad (0, 0), bottom-right (297, 420) -> kicad (297, -420)
    // But LED Y range is roughly -55 to -371, which is SVG Y range ~55 to 371
    // So SVG Y = -kicadY
    const [sx0, sy0] = toScreen(0, -0);     // SVG top-left
    const [sx1, sy1] = toScreen(297, -420);  // SVG bottom-right
    ctx.globalAlpha = 0.07;
    ctx.drawImage(boardImg, Math.min(sx0,sx1), Math.min(sy0,sy1), Math.abs(sx1-sx0), Math.abs(sy1-sy0));
    ctx.globalAlpha = 1.0;
  }

  // Draw LEDs
  for (let i = 0; i < LEDS.length; i++) {
    const [x, y] = toScreen(LEDS[i][0], LEDS[i][1]);
    let r = pixelColors[i * 3], g = pixelColors[i * 3 + 1], b = pixelColors[i * 3 + 2];
    const isActive = r > 0 || g > 0 || b > 0;

    // Boost brightness for display
    r = Math.min(255, Math.round(r * BRIGHTNESS_BOOST));
    g = Math.min(255, Math.round(g * BRIGHTNESS_BOOST));
    b = Math.min(255, Math.round(b * BRIGHTNESS_BOOST));

    if (isActive) {
      // Solid colored dot
      ctx.fillStyle = `rgb(${r},${g},${b})`;
      ctx.beginPath();
      ctx.arc(x, y, ledRadius * 1.3, 0, Math.PI * 2);
      ctx.fill();
    } else {
      ctx.fillStyle = 'rgba(50,50,40,0.4)';
      ctx.beginPath();
      ctx.arc(x, y, ledRadius * 0.6, 0, Math.PI * 2);
      ctx.fill();
    }

    if (i === hoveredLED) {
      ctx.strokeStyle = '#fff';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.arc(x, y, ledRadius + 4, 0, Math.PI * 2);
      ctx.stroke();
    }
  }

  requestAnimationFrame(draw);
}

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
    if (buf.byteLength === 0) { pixelColors.fill(0); lastFetchOk = true; activeCount = 0; updateStatus(); return; }
    const frame = decodePixelFrame(buf);
    lastSeq = frame.sequence;
    if (frame.pixels && frame.pixels.length >= TOTAL_LEDS * 3) pixelColors.set(frame.pixels.slice(0, TOTAL_LEDS * 3));
    activeCount = 0;
    for (let i = 0; i < TOTAL_LEDS; i++) if (pixelColors[i*3] || pixelColors[i*3+1] || pixelColors[i*3+2]) activeCount++;
    lastFetchOk = true;
  } catch (e) { lastFetchOk = false; }
  updateStatus();
}

function updateStatus() {
  statusDot.className = 'dot ' + (lastFetchOk ? 'ok' : 'err');
  statusInfo.textContent = lastFetchOk ? `${activeCount} trains | seq ${lastSeq}` : 'disconnected';
}

canvas.addEventListener('mousemove', (e) => {
  const rect = canvas.getBoundingClientRect();
  const idx = screenToLedIndex(e.clientX - rect.left, e.clientY - rect.top);
  hoveredLED = idx;
  if (idx >= 0) {
    const led = LEDS[idx];
    let r = pixelColors[idx*3], g = pixelColors[idx*3+1], b = pixelColors[idx*3+2];
    const [strip, pixel] = getStripPixel(idx);
    tooltip.innerHTML = `<span class="sid">${led[2]||'--'}</span> rgb(${r},${g},${b}) | strip ${strip} px ${pixel}`;
    tooltip.style.display = 'block';
    tooltip.style.left = (e.clientX + 14) + 'px';
    tooltip.style.top = (e.clientY - 10) + 'px';
  } else { tooltip.style.display = 'none'; }
});
canvas.addEventListener('mouseleave', () => { hoveredLED = -1; tooltip.style.display = 'none'; });

// Load board paths then start
async function init() {
  // Load LED positions
  const resp = await fetch('/static/leds.json');
  LEDS = await resp.json();
  computeBounds();
  resize();
  loadBoardImage();
  requestAnimationFrame(draw);
  await fetchPixels();
  setInterval(fetchPixels, 1000);
}

window.addEventListener('resize', resize);
init();