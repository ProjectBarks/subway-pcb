/**
 * board.js — Shared PCB board renderer
 *
 * Provides: LED position loading, coordinate transforms, board SVG background,
 * LED dot rendering, tooltip hover, hit testing.
 *
 * Usage:
 *   const board = new Board('canvasId', 'tooltipId');
 *   await board.init('/static/leds.json', '/static/board.svg');
 *   board.onHover = (idx, led) => { ... };
 *   board.onClick = (idx, led) => { ... };
 *   board.setPixels(uint8Array);  // RGB per LED
 *   board.startDrawLoop();
 */

const STRIP_SIZES = [97, 102, 55, 81, 70, 21, 22, 19, 11];
const TOTAL_LEDS = 478;

class Board {
  constructor(canvasId, tooltipId) {
    this.canvas = document.getElementById(canvasId);
    this.ctx = this.canvas.getContext('2d');
    this.tooltip = document.getElementById(tooltipId);
    this.leds = [];
    this.pixelColors = new Uint8Array(TOTAL_LEDS * 3);
    this.brightnessBoost = 1.0;
    this.boardSvgOpacity = 0.07;
    this.hoveredLED = -1;
    this.boardImg = null;
    this.boardImgLoaded = false;
    this.boardMinX = 0; this.boardMaxX = 0;
    this.boardMinY = 0; this.boardMaxY = 0;
    this.boardW = 0; this.boardH = 0;
    this.scale = 1; this.offsetX = 0; this.offsetY = 0; this.ledRadius = 3;

    // Callbacks
    this.onHover = null;   // (idx, led) => void
    this.onClick = null;   // (idx, led) => void
    this.drawOverlay = null; // (ctx, toScreen, ledRadius) => void — custom drawing after LEDs

    // Custom LED renderer (optional override)
    this.renderLed = null; // (ctx, x, y, idx, r, g, b, isHovered) => void

    this._setupEvents();
  }

  async init(ledsUrl, svgUrl) {
    const resp = await fetch(ledsUrl);
    this.leds = await resp.json();
    this._computeBounds();
    this._resize();
    if (svgUrl) {
      this.boardImg = new Image();
      this.boardImg.onload = () => { this.boardImgLoaded = true; };
      this.boardImg.src = svgUrl;
    }
  }

  setPixels(data) {
    if (data.length >= TOTAL_LEDS * 3) {
      this.pixelColors.set(data.subarray ? data.subarray(0, TOTAL_LEDS * 3) : data.slice(0, TOTAL_LEDS * 3));
    }
  }

  setPixel(idx, r, g, b) {
    this.pixelColors[idx * 3] = r;
    this.pixelColors[idx * 3 + 1] = g;
    this.pixelColors[idx * 3 + 2] = b;
  }

  clearPixels() {
    this.pixelColors.fill(0);
  }

  startDrawLoop() {
    const loop = () => { this._draw(); requestAnimationFrame(loop); };
    requestAnimationFrame(loop);
  }

  toScreen(x, y) {
    const cy = (this.boardMinY + this.boardMaxY) / 2;
    const ry = cy - (y - cy);
    return [this.offsetX + (x - this.boardMinX) * this.scale,
            this.offsetY + (ry - this.boardMinY) * this.scale];
  }

  hitTest(sx, sy) {
    let best = -1, bestDist = this.ledRadius * 3;
    for (let i = 0; i < this.leds.length; i++) {
      const [lx, ly] = this.toScreen(this.leds[i][0], this.leds[i][1]);
      const d = Math.hypot(sx - lx, sy - ly);
      if (d < bestDist) { bestDist = d; best = i; }
    }
    return best;
  }

  static getStripPixel(idx) {
    let off = 0;
    for (let s = 0; s < STRIP_SIZES.length; s++) {
      if (idx < off + STRIP_SIZES[s]) return [s, idx - off];
      off += STRIP_SIZES[s];
    }
    return [-1, -1];
  }

  // --- Private ---

  _computeBounds() {
    const xs = this.leds.map(l => l[0]), ys = this.leds.map(l => l[1]);
    this.boardMinX = Math.min(...xs) - 15; this.boardMaxX = Math.max(...xs) + 15;
    this.boardMinY = Math.min(...ys) - 15; this.boardMaxY = Math.max(...ys) + 15;
    this.boardW = this.boardMaxX - this.boardMinX;
    this.boardH = this.boardMaxY - this.boardMinY;
  }

  _resize() {
    const dpr = window.devicePixelRatio || 1;
    const rect = this.canvas.parentElement.getBoundingClientRect();
    this.canvas.width = rect.width * dpr;
    this.canvas.height = rect.height * dpr;
    this.ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    const pad = 20, cw = rect.width - pad * 2, ch = rect.height - pad * 2;
    this.scale = Math.min(cw / this.boardW, ch / this.boardH);
    this.offsetX = pad + (cw - this.boardW * this.scale) / 2;
    this.offsetY = pad + (ch - this.boardH * this.scale) / 2;
    this.ledRadius = Math.max(2.5, Math.min(5, this.scale * 1.8));
  }

  _draw() {
    const rect = this.canvas.parentElement.getBoundingClientRect();
    this.ctx.clearRect(0, 0, rect.width, rect.height);

    // Board SVG background
    if (this.boardImgLoaded) {
      const [sx0, sy0] = this.toScreen(0, 0);
      const [sx1, sy1] = this.toScreen(297, -420);
      this.ctx.globalAlpha = this.boardSvgOpacity;
      this.ctx.drawImage(this.boardImg, Math.min(sx0, sx1), Math.min(sy0, sy1),
                         Math.abs(sx1 - sx0), Math.abs(sy1 - sy0));
      this.ctx.globalAlpha = 1.0;
    }

    // LEDs
    for (let i = 0; i < this.leds.length; i++) {
      const [x, y] = this.toScreen(this.leds[i][0], this.leds[i][1]);
      let r = this.pixelColors[i * 3], g = this.pixelColors[i * 3 + 1], b = this.pixelColors[i * 3 + 2];
      const isHovered = i === this.hoveredLED;

      if (this.renderLed) {
        this.renderLed(this.ctx, x, y, i, r, g, b, isHovered);
        continue;
      }

      // Default rendering
      r = Math.min(255, Math.round(r * this.brightnessBoost));
      g = Math.min(255, Math.round(g * this.brightnessBoost));
      b = Math.min(255, Math.round(b * this.brightnessBoost));

      if (r > 0 || g > 0 || b > 0) {
        this.ctx.fillStyle = `rgb(${r},${g},${b})`;
        this.ctx.beginPath();
        this.ctx.arc(x, y, this.ledRadius * 1.3, 0, Math.PI * 2);
        this.ctx.fill();
      } else {
        this.ctx.fillStyle = isHovered ? 'rgba(200,200,200,0.6)' : 'rgba(50,50,40,0.4)';
        this.ctx.beginPath();
        this.ctx.arc(x, y, this.ledRadius * 0.6, 0, Math.PI * 2);
        this.ctx.fill();
      }

      if (isHovered) {
        this.ctx.strokeStyle = '#fff';
        this.ctx.lineWidth = 1.5;
        this.ctx.beginPath();
        this.ctx.arc(x, y, this.ledRadius + 4, 0, Math.PI * 2);
        this.ctx.stroke();
      }
    }

    if (this.drawOverlay) {
      this.drawOverlay(this.ctx, this.toScreen.bind(this), this.ledRadius);
    }
  }

  _setupEvents() {
    window.addEventListener('resize', () => this._resize());

    this.canvas.addEventListener('mousemove', (e) => {
      const rect = this.canvas.getBoundingClientRect();
      const idx = this.hitTest(e.clientX - rect.left, e.clientY - rect.top);
      this.hoveredLED = idx;

      if (idx >= 0) {
        const led = this.leds[idx];
        const [strip, pixel] = Board.getStripPixel(idx);
        const r = this.pixelColors[idx * 3], g = this.pixelColors[idx * 3 + 1], b = this.pixelColors[idx * 3 + 2];
        if (this.onHover) {
          this.onHover(idx, led, strip, pixel, r, g, b);
        }
        this.tooltip.innerHTML = `<span class="sid">${led[2] || '--'}</span> rgb(${r},${g},${b}) | strip ${strip} px ${pixel}`;
        this.tooltip.style.display = 'block';
        this.tooltip.style.left = (e.clientX + 14) + 'px';
        this.tooltip.style.top = (e.clientY - 10) + 'px';
      } else {
        this.tooltip.style.display = 'none';
      }
    });

    this.canvas.addEventListener('mouseleave', () => {
      this.hoveredLED = -1;
      this.tooltip.style.display = 'none';
    });

    this.canvas.addEventListener('click', (e) => {
      const rect = this.canvas.getBoundingClientRect();
      const idx = this.hitTest(e.clientX - rect.left, e.clientY - rect.top);
      if (idx >= 0 && this.onClick) {
        const led = this.leds[idx];
        const [strip, pixel] = Board.getStripPixel(idx);
        this.onClick(idx, led, strip, pixel);
      }
    });
  }
}
