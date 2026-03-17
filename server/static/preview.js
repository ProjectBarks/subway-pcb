/**
 * preview.js — Client-side theme draft preview
 *
 * Fetches train state and LED map, then re-renders the board canvas
 * with custom route colors for instant theme preview feedback.
 */

class PreviewRenderer {
  constructor(board) {
    this.board = board;
    this.stateData = null;
    this.ledMap = null;
    this.ledMapFlat = null;
    this.routeColors = null;
    this._refreshInterval = null;
  }

  async init() {
    // Load LED map (station assignments)
    try {
      const resp = await fetch('/static/leds.json');
      this.ledMap = await resp.json();
    } catch (e) {
      console.warn('preview: failed to load leds.json', e);
    }

    // Load flat LED map for station-to-LED mapping
    try {
      const resp = await fetch('/static/led_map.json');
      const raw = await resp.json();
      // Build flat array matching server's strip layout
      const STRIP_SIZES = [97, 102, 55, 81, 70, 21, 22, 19, 11];
      const TOTAL = 478;
      this.ledMapFlat = new Array(TOTAL).fill('');
      let offset = 0;
      for (let strip = 0; strip < 9; strip++) {
        for (let pixel = 0; pixel < STRIP_SIZES[strip]; pixel++) {
          const key = strip + ',' + pixel;
          if (raw[key]) {
            this.ledMapFlat[offset] = raw[key];
          }
          offset++;
        }
      }
    } catch (e) {
      console.warn('preview: failed to load led_map.json', e);
    }

    // Fetch initial state
    await this.refreshState();

    // Auto-refresh state every 5 seconds
    this._refreshInterval = setInterval(() => this.refreshState(), 5000);
  }

  async refreshState() {
    try {
      const resp = await fetch('/api/v1/state?format=json');
      if (resp.ok) {
        this.stateData = await resp.json();
      }
    } catch (e) {
      console.warn('preview: failed to fetch state', e);
    }
  }

  /**
   * Set custom route colors and immediately re-render the board.
   * @param {Object} colors - Map of route key to hex color, e.g. {"ROUTE_1": "#ff0000"}
   */
  setThemeColors(colors) {
    this.routeColors = colors;
    this.render();
  }

  /**
   * Render the board with current route colors and train state.
   */
  render() {
    if (!this.stateData || !this.ledMapFlat || !this.routeColors) return;

    const TOTAL = 478;
    const pixels = new Uint8Array(TOTAL * 3);

    // Build station -> best route map from state
    const stationRoutes = {};
    if (this.stateData.stations) {
      for (const station of this.stateData.stations) {
        if (station.trains && station.trains.length > 0) {
          // Use the first train's route
          const routeEnum = station.trains[0].route;
          stationRoutes[station.stop_id] = routeEnum;
        }
      }
    }

    // Map station routes to LED pixels using custom colors
    for (let i = 0; i < TOTAL; i++) {
      const sid = this.ledMapFlat[i];
      if (!sid) continue;

      const routeEnum = stationRoutes[sid];
      if (!routeEnum) continue;

      // Convert proto enum name to route key (e.g., "ROUTE_1")
      const routeKey = routeEnum;
      const hexColor = this.routeColors[routeKey];
      if (!hexColor) continue;

      const rgb = hexToRgb(hexColor);
      if (rgb) {
        pixels[i * 3] = rgb.r;
        pixels[i * 3 + 1] = rgb.g;
        pixels[i * 3 + 2] = rgb.b;
      }
    }

    this.board.setPixels(pixels);
  }

  destroy() {
    if (this._refreshInterval) {
      clearInterval(this._refreshInterval);
      this._refreshInterval = null;
    }
  }
}

function hexToRgb(hex) {
  if (!hex || hex.length < 7) return null;
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return { r, g, b };
}

/**
 * Called from theme color picker inputs to update live preview.
 * Collects all color inputs and re-renders.
 */
function updatePreviewColor(input) {
  if (!window._previewRenderer) return;

  const colors = {};
  document.querySelectorAll('input[data-route-key]').forEach(el => {
    colors[el.dataset.routeKey] = el.value;
  });

  window._previewRenderer.setThemeColors(colors);
}
